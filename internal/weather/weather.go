package weather

// Open-Meteo 免费天气 API（无需 key）。抓取失败时回退到本地缓存，
// 缓存过期、时间异常或缺少当天预报的数据在界面上标记为离线。

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
)

var weatherTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   20 * time.Second,
	ResponseHeaderTimeout: 20 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var weatherClient = &http.Client{
	Transport: weatherTransport,
	Timeout:   35 * time.Second,
}

// 包内测试可替换以下地址；常规测试不需要也不允许访问真实外网。
var (
	// 国内镜像使用明文 HTTP，默认禁用；测试可注入 httptest 地址。
	chinaAirQualityBaseURL = ""
	openMeteoAirQualityURL = "https://air-quality-api.open-meteo.com/v1/air-quality"
)

type IconKind int

const (
	IconSun IconKind = iota
	IconPartly
	IconCloud
	IconFog
	IconDrizzle
	IconRain
	IconSleet
	IconSnow
	IconThunder
)

type Weather struct {
	Fetched time.Time `json:"fetched"`
	Cached  bool      `json:"-"`
	Offline bool      `json:"-"`

	Current struct {
		Temp            float64 `json:"temperature_2m"`
		Apparent        float64 `json:"apparent_temperature"`
		Humidity        int     `json:"relative_humidity_2m"`
		Code            int     `json:"weather_code"`
		UVIndex         float64 `json:"uv_index"`
		AQI             int     `json:"aqi,omitempty"`
		USAQI           int     `json:"us_aqi,omitempty"`
		PM25            float64 `json:"pm25,omitempty"`
		PM10            float64 `json:"pm10,omitempty"`
		Quality         string  `json:"quality,omitempty"`
		Notice          string  `json:"notice,omitempty"`
		WindDir         string  `json:"wind_dir,omitempty"`
		WindScale       string  `json:"wind_scale,omitempty"`
		AirSource       string  `json:"air_source,omitempty"`
		AirQualityValid bool    `json:"air_quality_valid"`
		AirError        string  `json:"air_error,omitempty"`
	} `json:"current"`

	Daily struct {
		Time        []string  `json:"time"`
		WeatherCode []int     `json:"weather_code"`
		TMax        []float64 `json:"temperature_2m_max"`
		TMin        []float64 `json:"temperature_2m_min"`
		PrecipProb  []int     `json:"precipitation_probability_max"`
		Sunrise     []string  `json:"sunrise"`
		Sunset      []string  `json:"sunset"`
		UVIndexMax  []float64 `json:"uv_index_max,omitempty"`
	} `json:"daily"`
}

type DayWeather struct {
	Label     string
	Code      int
	Kind      IconKind
	TMax      float64
	TMin      float64
	Precip    int
	Sunrise   string
	Sunset    string
	UVMax     float64
	AQI       int
	PM25      float64
	PM10      float64
	Quality   string
	Notice    string
	WindDir   string
	WindScale string
	AirSource string
}

func (w *Weather) validate() error {
	if w == nil {
		return fmt.Errorf("天气数据为空")
	}
	n := len(w.Daily.Time)
	if n == 0 {
		return fmt.Errorf("天气预报日期为空")
	}
	series := []struct {
		name string
		n    int
	}{
		{"weather_code", len(w.Daily.WeatherCode)},
		{"temperature_2m_max", len(w.Daily.TMax)},
		{"temperature_2m_min", len(w.Daily.TMin)},
		{"precipitation_probability_max", len(w.Daily.PrecipProb)},
		{"sunrise", len(w.Daily.Sunrise)},
		{"sunset", len(w.Daily.Sunset)},
	}
	for _, s := range series {
		if s.n < n {
			return fmt.Errorf("天气字段 %s 只有 %d 项，需要 %d 项", s.name, s.n, n)
		}
	}
	for _, day := range w.Daily.Time {
		if _, err := time.Parse("2006-01-02", day); err != nil {
			return fmt.Errorf("天气日期 %q 无效: %w", day, err)
		}
	}
	return nil
}

// Days 从 now 当天开始返回最多三天预报；缓存中的昨天不会被标成今天。
func (w *Weather) Days(now time.Time) []DayWeather {
	if w == nil || w.validate() != nil {
		return nil
	}
	today := now.Format("2006-01-02")
	start := -1
	for i, day := range w.Daily.Time {
		if day == today {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}

	out := make([]DayWeather, 0, 3)
	for i := start; i < len(w.Daily.Time) && len(out) < 3; i++ {
		d := DayWeather{
			Code:      w.Daily.WeatherCode[i],
			Kind:      Icon(w.Daily.WeatherCode[i]),
			TMax:      w.Daily.TMax[i],
			TMin:      w.Daily.TMin[i],
			Precip:    w.Daily.PrecipProb[i],
			Sunrise:   HHMM(w.Daily.Sunrise[i]),
			Sunset:    HHMM(w.Daily.Sunset[i]),
			AQI:       w.Current.AQI,
			PM25:      w.Current.PM25,
			PM10:      w.Current.PM10,
			Quality:   w.Current.Quality,
			Notice:    w.Current.Notice,
			WindDir:   w.Current.WindDir,
			WindScale: w.Current.WindScale,
			AirSource: w.Current.AirSource,
		}
		if len(w.Daily.UVIndexMax) > i {
			d.UVMax = w.Daily.UVIndexMax[i]
		} else if i == start {
			d.UVMax = w.Current.UVIndex
		}
		switch len(out) {
		case 0:
			d.Label = "今天"
		case 1:
			d.Label = "明天"
		default:
			day, err := time.ParseInLocation("2006-01-02", w.Daily.Time[i], now.Location())
			if err == nil {
				d.Label = WeekCN(day.Weekday())
			} else {
				d.Label = "后天"
			}
		}
		out = append(out, d)
	}
	return out
}

// HHMM 从 Open-Meteo 的日期时间字符串中提取 HH:MM。
func HHMM(s string) string {
	if i := strings.IndexByte(s, 'T'); i >= 0 && len(s) >= i+6 {
		return s[i+1 : i+6]
	}
	return s
}

// WeekCN 返回中文星期标签。
func WeekCN(d time.Weekday) string {
	names := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	if d < time.Sunday || d > time.Saturday {
		return ""
	}
	return names[int(d)]
}

// Text 返回 WMO 天气代码对应的中文描述。
func Text(code int) string {
	switch code {
	case 0:
		return "晴"
	case 1:
		return "大部晴"
	case 2:
		return "多云"
	case 3:
		return "阴"
	case 45, 48:
		return "雾"
	case 51, 53, 55:
		return "毛毛雨"
	case 56, 57, 66, 67:
		return "冻雨"
	case 61:
		return "小雨"
	case 63:
		return "中雨"
	case 65:
		return "大雨"
	case 71:
		return "小雪"
	case 73:
		return "中雪"
	case 75:
		return "大雪"
	case 77:
		return "霰"
	case 80, 81:
		return "阵雨"
	case 82:
		return "强阵雨"
	case 85, 86:
		return "阵雪"
	case 95:
		return "雷阵雨"
	case 96, 99:
		return "雷雨冰雹"
	default:
		return "天气"
	}
}

// Icon 返回 WMO 天气代码对应的图标类别。
func Icon(code int) IconKind {
	switch {
	case code == 0:
		return IconSun
	case code == 1 || code == 2:
		return IconPartly
	case code == 3:
		return IconCloud
	case code == 45 || code == 48:
		return IconFog
	case code >= 51 && code <= 57:
		return IconDrizzle
	case (code >= 61 && code <= 65) || (code >= 80 && code <= 82):
		return IconRain
	case code == 66 || code == 67:
		return IconSleet
	case (code >= 71 && code <= 77) || code == 85 || code == 86:
		return IconSnow
	case code >= 95:
		return IconThunder
	default:
		return IconCloud
	}
}

func cachePath(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	path := cfg.WeatherCachePath
	if strings.TrimSpace(path) == "" {
		path = "data/cache_weather.json"
	}
	return cfg.ResolvePath(path)
}

// LoadCached 读取持久缓存，并对过期、未来时间或缺少当天数据标记 Offline。
func LoadCached(cfg *config.Config, now time.Time) *Weather {
	if cfg == nil {
		return nil
	}
	data, err := os.ReadFile(cachePath(cfg))
	if err != nil {
		return nil
	}
	var cached Weather
	if err := json.Unmarshal(data, &cached); err != nil {
		log.Printf("天气缓存解析失败: %v", err)
		return nil
	}
	if err := cached.validate(); err != nil {
		log.Printf("天气缓存无效: %v", err)
		return nil
	}
	cached.Cached = true
	age := now.Sub(cached.Fetched)
	cached.Offline = age > time.Duration(cfg.OfflineGraceMin)*time.Minute || age < -5*time.Minute || !cached.hasDate(now)
	log.Printf("使用天气缓存（抓取于 %s）", cached.Fetched.Format("2006-01-02 15:04"))
	return &cached
}

func (w *Weather) hasDate(now time.Time) bool {
	if w == nil {
		return false
	}
	today := now.Format("2006-01-02")
	for _, day := range w.Daily.Time {
		if day == today {
			return true
		}
	}
	return false
}

// Get 在线抓取天气并写缓存，失败时回退到持久缓存。
func Get(cfg *config.Config, now time.Time) *Weather {
	if cfg == nil {
		return nil
	}
	w := fetchOpenMeteo(cfg)
	if w != nil {
		w.Fetched = now
		w.Cached = false
		w.Offline = false
		if data, err := json.Marshal(w); err == nil {
			path := cachePath(cfg)
			if err := config.WriteFileAtomic(path, data, 0644); err != nil {
				log.Printf("写天气缓存失败: %v", err)
			} else {
				log.Printf("已写入天气持久缓存: %s", path)
			}
		}
		log.Printf("天气抓取成功: 当前 %.1f°C %s", w.Current.Temp, Text(w.Current.Code))
		return w
	}
	return LoadCached(cfg, now)
}

func weatherRequestURL(cfg *config.Config) (string, error) {
	u, err := url.Parse(cfg.WeatherURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("latitude", strconv.FormatFloat(cfg.Location.Lat, 'f', 6, 64))
	q.Set("longitude", strconv.FormatFloat(cfg.Location.Lon, 'f', 6, 64))
	q.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,weather_code,uv_index")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,sunrise,sunset,uv_index_max")
	q.Set("timezone", cfg.Timezone)
	q.Set("forecast_days", "3")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

var chinaCityCodes = map[string]string{
	"西安":   "101110101",
	"北京":   "101010100",
	"上海":   "101020100",
	"广州":   "101280101",
	"深圳":   "101280601",
	"成都":   "101270101",
	"杭州":   "101210101",
	"武汉":   "101200101",
	"南京":   "101190101",
	"重庆":   "101040100",
	"天津":   "101030100",
	"苏州":   "101190401",
	"长沙":   "101250101",
	"郑州":   "101180101",
	"沈阳":   "101070101",
	"青岛":   "101120201",
	"济南":   "101120101",
	"大连":   "101070201",
	"厦门":   "101230201",
	"福州":   "101230101",
	"昆明":   "101290101",
	"合肥":   "101220101",
	"石家庄":  "101090101",
	"哈尔滨":  "101050101",
	"长春":   "101060101",
	"南昌":   "101240101",
	"贵阳":   "101260101",
	"南宁":   "101300101",
	"兰州":   "101160101",
	"太原":   "101100101",
	"海口":   "101310101",
	"三亚":   "101310201",
	"银川":   "101170101",
	"西宁":   "101150101",
	"呼和浩特": "101080101",
	"乌鲁木齐": "101130101",
	"拉萨":   "101140101",
}

func resolveChinaCityCode(cfg *config.Config) string {
	if cfg.Location.CityCode != "" {
		return cfg.Location.CityCode
	}
	name := strings.TrimSpace(cfg.Location.Name)
	if name == "" {
		return ""
	}
	for k, v := range chinaCityCodes {
		if strings.Contains(name, k) || strings.Contains(k, name) {
			return v
		}
	}
	return ""
}

type itboyResponse struct {
	Status int `json:"status"`
	Data   struct {
		PM25     *float64 `json:"pm25"`
		PM10     *float64 `json:"pm10"`
		Quality  string   `json:"quality"`
		Ganmao   string   `json:"ganmao"`
		Forecast []struct {
			AQI    *int   `json:"aqi"`
			Notice string `json:"notice"`
			Fx     string `json:"fx"`
			Fl     string `json:"fl"`
		} `json:"forecast"`
	} `json:"data"`
}

func fetchChinaAirQuality(cityCode string) (*itboyResponse, error) {
	if cityCode == "" {
		return nil, fmt.Errorf("缺少国内天气城市代码")
	}
	u, err := url.Parse(chinaAirQualityBaseURL)
	if err != nil {
		return nil, fmt.Errorf("国内空气质量地址无效: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + url.PathEscape(cityCode)
	resp, err := weatherClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var result itboyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	if result.Status != 200 {
		return nil, fmt.Errorf("接口状态 %d", result.Status)
	}
	if result.Data.PM25 == nil || result.Data.PM10 == nil {
		return nil, fmt.Errorf("缺少 PM2.5 或 PM10")
	}
	if !validConcentration(*result.Data.PM25) || !validConcentration(*result.Data.PM10) {
		return nil, fmt.Errorf("PM 浓度无效")
	}
	return &result, nil
}

type openMeteoAirQuality struct {
	USAQI int
	PM25  float64
	PM10  float64
}

func fetchOpenMeteoAirQuality(cfg *config.Config) (*openMeteoAirQuality, error) {
	u, err := url.Parse(openMeteoAirQualityURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("latitude", strconv.FormatFloat(cfg.Location.Lat, 'f', 6, 64))
	q.Set("longitude", strconv.FormatFloat(cfg.Location.Lon, 'f', 6, 64))
	q.Set("current", "us_aqi,pm10,pm2_5")
	q.Set("timezone", cfg.Timezone)
	u.RawQuery = q.Encode()

	resp, err := weatherClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var data struct {
		Current struct {
			USAQI *int     `json:"us_aqi"`
			PM10  *float64 `json:"pm10"`
			PM25  *float64 `json:"pm2_5"`
		} `json:"current"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	if data.Current.USAQI == nil || data.Current.PM10 == nil || data.Current.PM25 == nil {
		return nil, fmt.Errorf("缺少当前空气质量字段")
	}
	if *data.Current.USAQI < 0 || !validConcentration(*data.Current.PM25) || !validConcentration(*data.Current.PM10) {
		return nil, fmt.Errorf("空气质量数值无效")
	}
	return &openMeteoAirQuality{
		USAQI: *data.Current.USAQI,
		PM25:  *data.Current.PM25,
		PM10:  *data.Current.PM10,
	}, nil
}

func validConcentration(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func calcIAQI(value float64, concentrations []float64, indexes []int) int {
	if value <= concentrations[0] {
		return indexes[0]
	}
	n := len(concentrations)
	if value >= concentrations[n-1] {
		return indexes[n-1]
	}
	for i := 1; i < n; i++ {
		if value <= concentrations[i] {
			cLo, cHi := concentrations[i-1], concentrations[i]
			iLo, iHi := float64(indexes[i-1]), float64(indexes[i])
			return int(math.Round((iHi-iLo)/(cHi-cLo)*(value-cLo) + iLo))
		}
	}
	return int(value)
}

func calcChinaAQI(pm25, pm10 float64) (int, string) {
	iaqi25 := calcIAQI(pm25, []float64{0, 35, 75, 115, 150, 250, 350, 500}, []int{0, 50, 100, 150, 200, 300, 400, 500})
	iaqi10 := calcIAQI(pm10, []float64{0, 50, 150, 250, 350, 420, 500, 600}, []int{0, 50, 100, 150, 200, 300, 400, 500})
	aqi := iaqi25
	if iaqi10 > aqi {
		aqi = iaqi10
	}
	return aqi, aqiLevel(aqi)
}

func populateAirQuality(w *Weather, cfg *config.Config) {
	cnErr := fmt.Errorf("国内空气质量镜像未启用")
	if chinaAirQualityBaseURL != "" {
		cn, err := fetchChinaAirQuality(resolveChinaCityCode(cfg))
		cnErr = err
		if err == nil {
			w.Current.PM25 = *cn.Data.PM25
			w.Current.PM10 = *cn.Data.PM10
			w.Current.Quality = cn.Data.Quality
			w.Current.AirSource = "中国环境监测总站"
			aqi := 0
			hasAQI := false
			if len(cn.Data.Forecast) > 0 {
				forecast := cn.Data.Forecast[0]
				if forecast.AQI != nil && *forecast.AQI >= 0 {
					aqi = *forecast.AQI
					hasAQI = true
				}
				w.Current.Notice = forecast.Notice
				w.Current.WindDir = forecast.Fx
				w.Current.WindScale = forecast.Fl
			}
			if !hasAQI {
				aqi, _ = calcChinaAQI(w.Current.PM25, w.Current.PM10)
			}
			w.Current.AQI = aqi
			if w.Current.Quality == "" {
				w.Current.Quality = aqiLevel(aqi)
			}
			if w.Current.Notice == "" && cn.Data.Ganmao != "" {
				w.Current.Notice = cn.Data.Ganmao
			}
			w.Current.AirQualityValid = true
			w.Current.AirError = ""
			log.Printf("国内空气质量获取成功: %s AQI %d (%s), PM2.5: %.0f, PM10: %.0f",
				cfg.Location.Name, w.Current.AQI, w.Current.Quality, w.Current.PM25, w.Current.PM10)
			return
		}
	}

	air, openErr := fetchOpenMeteoAirQuality(cfg)
	if openErr == nil {
		w.Current.USAQI = air.USAQI
		w.Current.PM25 = air.PM25
		w.Current.PM10 = air.PM10
		w.Current.AirSource = "气象遥感监测"
		w.Current.AQI, w.Current.Quality = calcChinaAQI(air.PM25, air.PM10)
		w.Current.AirQualityValid = true
		w.Current.AirError = ""
		log.Printf("气象空气质量获取成功: AQI %d (%s), PM2.5: %.1f", w.Current.AQI, w.Current.Quality, air.PM25)
		return
	}

	w.Current.AQI = 0
	w.Current.USAQI = 0
	w.Current.PM25 = 0
	w.Current.PM10 = 0
	w.Current.Quality = "未知"
	w.Current.AirSource = ""
	w.Current.AirQualityValid = false
	w.Current.AirError = fmt.Sprintf("国内空气质量: %v；Open-Meteo 空气质量: %v", cnErr, openErr)
	log.Printf("空气质量获取失败: %s", w.Current.AirError)
}

func aqiLevel(aqi int) string {
	switch {
	case aqi < 0:
		return "未知"
	case aqi <= 50:
		return "优"
	case aqi <= 100:
		return "良"
	case aqi <= 150:
		return "轻度"
	case aqi <= 200:
		return "中度"
	case aqi <= 300:
		return "重度"
	default:
		return "严重"
	}
}

func uvLevel(uv float64) string {
	switch {
	case uv < 0.5:
		return "最弱"
	case uv < 3.0:
		return "较弱"
	case uv < 6.0:
		return "中等"
	case uv < 8.0:
		return "较强"
	case uv < 11.0:
		return "极强"
	default:
		return "危险"
	}
}

func fetchOpenMeteo(cfg *config.Config) *Weather {
	requestURL, err := weatherRequestURL(cfg)
	if err != nil {
		log.Printf("天气地址无效: %v", err)
		return nil
	}
	resp, err := weatherClient.Get(requestURL)
	if err != nil {
		log.Printf("天气请求失败: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("天气请求 HTTP %d", resp.StatusCode)
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Printf("天气响应读取失败: %v", err)
		return nil
	}
	var w Weather
	if err := json.Unmarshal(data, &w); err != nil {
		log.Printf("天气 JSON 解析失败: %v", err)
		return nil
	}
	if err := w.validate(); err != nil {
		log.Printf("天气数据无效: %v", err)
		return nil
	}
	populateAirQuality(&w, cfg)
	return &w
}

// ScheduleNext 根据在线成功或缓存重试结果选择天气快照及下一次抓取时间。
func ScheduleNext(current, updated *Weather, refreshSec int) (*Weather, time.Time) {
	if refreshSec <= 0 {
		refreshSec = 60
	}
	if updated != nil && !updated.Cached {
		log.Printf("天气在线更新成功，下次将在 %ds 后刷新", refreshSec)
		return updated, time.Now().Add(time.Duration(refreshSec) * time.Second)
	}
	retrySec := 60
	if refreshSec < retrySec {
		retrySec = refreshSec
	}
	if updated != nil {
		log.Printf("网络天气更新未成功，使用本地缓存，将在 %ds 后重试", retrySec)
		return updated, time.Now().Add(time.Duration(retrySec) * time.Second)
	}
	if current != nil {
		log.Printf("网络天气更新未成功，保留现有缓存，将在 %ds 后重试", retrySec)
		return current, time.Now().Add(time.Duration(retrySec) * time.Second)
	}
	log.Printf("网络天气获取失败且暂无缓存，将在 %ds 后重试", retrySec)
	return nil, time.Now().Add(time.Duration(retrySec) * time.Second)
}

// Example 返回不依赖网络的三天示例天气。
func Example(now time.Time) *Weather {
	w := &Weather{Fetched: now}
	w.Current.Temp = 14
	w.Current.Apparent = 13
	w.Current.Humidity = 59
	w.Current.Code = 2
	codes := []int{3, 61, 2}
	highs := []float64{16, 11, 13}
	lows := []float64{5, 7, 4}
	precip := []int{0, 80, 20}
	for i := 0; i < 3; i++ {
		day := now.AddDate(0, 0, i)
		w.Daily.Time = append(w.Daily.Time, day.Format("2006-01-02"))
		w.Daily.WeatherCode = append(w.Daily.WeatherCode, codes[i])
		w.Daily.TMax = append(w.Daily.TMax, highs[i])
		w.Daily.TMin = append(w.Daily.TMin, lows[i])
		w.Daily.PrecipProb = append(w.Daily.PrecipProb, precip[i])
		w.Daily.Sunrise = append(w.Daily.Sunrise, day.Format("2006-01-02")+"T07:15")
		w.Daily.Sunset = append(w.Daily.Sunset, day.Format("2006-01-02")+"T17:40")
	}
	return w
}
