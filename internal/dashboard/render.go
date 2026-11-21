package dashboard

// 渲染：横版设计稿（W=长边, H=短边），由调用方旋转后提供给 Kindle。
// 使用 opentype（Non-Zero Winding 规则），彻底杜绝笔画交叉区域变白镂空的缺陷。

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/fs"
	"math"
	"time"

	"github.com/andycurise/kindle-kdash/assets"
	"github.com/andycurise/kindle-kdash/internal/weather"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

type fontSet struct {
	reg, med, bold    *opentype.Font
	clockBebas        *opentype.Font
	clockJetBrains    *opentype.Font
	clockMontserrat   *opentype.Font
	clockPlayfair     *opentype.Font
	calligraphyKai    *opentype.Font // 颜楷毛笔书法（颜柳风骨）
	calligraphyWei    *opentype.Font // 魏碑宋韵体
	calligraphyXing   *opentype.Font // 龙苍行书
	calligraphyLively *opentype.Font // 站酷快乐体
}

func loadFonts() (*fontSet, error) {
	loadOT := func(name string) (*opentype.Font, error) {
		b, err := fs.ReadFile(assets.Fonts, name)
		if err != nil {
			return nil, err
		}
		return opentype.Parse(b)
	}
	reg, err := loadOT("fonts/NotoSansSC-400.ttf")
	if err != nil {
		return nil, fmt.Errorf("加载 NotoSansSC-400: %w", err)
	}
	med, err := loadOT("fonts/NotoSansSC-500.ttf")
	if err != nil {
		return nil, fmt.Errorf("加载 NotoSansSC-500: %w", err)
	}
	bold, err := loadOT("fonts/NotoSansSC-700.ttf")
	if err != nil {
		return nil, fmt.Errorf("加载 NotoSansSC-700: %w", err)
	}
	bebas, _ := loadOT("fonts/BebasNeue.ttf")
	jetbrains, _ := loadOT("fonts/JetBrainsMono.ttf")
	montserrat, _ := loadOT("fonts/Montserrat.ttf")
	playfair, _ := loadOT("fonts/PlayfairDisplay.ttf")

	kai, _ := loadOT("fonts/MaShanZheng.ttf")
	wei, _ := loadOT("fonts/ZCOOLXiaoWei.ttf")
	xing, _ := loadOT("fonts/LongCang.ttf")
	lively, _ := loadOT("fonts/ZCOOLKuaiLe.ttf")

	return &fontSet{
		reg:               reg,
		med:               med,
		bold:              bold,
		clockBebas:        bebas,
		clockJetBrains:    jetbrains,
		clockMontserrat:   montserrat,
		clockPlayfair:     playfair,
		calligraphyKai:    kai,
		calligraphyWei:    wei,
		calligraphyXing:   xing,
		calligraphyLively: lively,
	}, nil
}

var (
	ink        = color.RGBA{18, 18, 20, 255}    // 极深黑，高对比度主文本
	inkSoft    = color.RGBA{75, 78, 85, 255}    // 深炭灰，二级文本与说明
	inkMuted   = color.RGBA{120, 125, 135, 255} // 中灰，辅助标签与单位
	inkBorder  = color.RGBA{190, 195, 202, 255} // 细边框灰
	inkLine    = color.RGBA{222, 225, 230, 255} // 分割线灰
	cardBg     = color.RGBA{244, 245, 247, 255} // 卡片柔和浅灰底色
	cardBgWarm = color.RGBA{236, 238, 242, 255} // 今日/重点卡片强调底色
	paper      = color.RGBA{255, 255, 255, 255} // 纯白背景
)

type renderer struct {
	img   *image.RGBA
	w, h  int
	f     *fontSet
	faces map[string]font.Face
}

// clip returns a transparent full-canvas renderer. The caller composites only
// bounds back to the frame, which clips both text and vector paths while
// retaining absolute coordinates and the shared font cache.
func (r *renderer) clip(bounds image.Rectangle) *renderer {
	return &renderer{img: image.NewRGBA(image.Rect(0, 0, r.w, r.h)), w: r.w, h: r.h, f: r.f, faces: r.faces}
}

func newRenderer(w, h int, f *fontSet, faces map[string]font.Face) (*renderer, error) {
	if f == nil {
		var err error
		f, err = loadFonts()
		if err != nil {
			return nil, err
		}
	}
	if faces == nil {
		faces = map[string]font.Face{}
	}
	r := &renderer{
		img:   image.NewRGBA(image.Rect(0, 0, w, h)),
		w:     w,
		h:     h,
		f:     f,
		faces: faces,
	}
	draw.Draw(r.img, r.img.Bounds(), &image.Uniform{paper}, image.Point{}, draw.Src)
	return r, nil
}

func (r *renderer) face(weight string, size int) font.Face {
	if size < 1 {
		size = 1
	}
	key := fmt.Sprintf("%s@%d", weight, size)
	if f, ok := r.faces[key]; ok {
		return f
	}
	var of *opentype.Font
	switch weight {
	case "med":
		of = r.f.med
	case "bold":
		of = r.f.bold
	case "kai", "calligraphy-kai":
		if r.f.calligraphyKai != nil {
			of = r.f.calligraphyKai
		} else {
			of = r.f.bold
		}
	case "wei", "weibei", "calligraphy-wei":
		if r.f.calligraphyWei != nil {
			of = r.f.calligraphyWei
		} else {
			of = r.f.bold
		}
	case "xing", "xingshu", "calligraphy-xing":
		if r.f.calligraphyXing != nil {
			of = r.f.calligraphyXing
		} else {
			of = r.f.bold
		}
	case "lively", "kuaile", "calligraphy-lively":
		if r.f.calligraphyLively != nil {
			of = r.f.calligraphyLively
		} else {
			of = r.f.bold
		}
	case "clock-bebas":
		if r.f.clockBebas != nil {
			of = r.f.clockBebas
		} else {
			of = r.f.bold
		}
	case "clock-jetbrains":
		if r.f.clockJetBrains != nil {
			of = r.f.clockJetBrains
		} else {
			of = r.f.bold
		}
	case "clock-montserrat":
		if r.f.clockMontserrat != nil {
			of = r.f.clockMontserrat
		} else {
			of = r.f.bold
		}
	case "clock-playfair":
		if r.f.clockPlayfair != nil {
			of = r.f.clockPlayfair
		} else {
			of = r.f.bold
		}
	default:
		of = r.f.reg
	}
	fc, err := opentype.NewFace(of, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		fc, _ = opentype.NewFace(r.f.reg, &opentype.FaceOptions{
			Size:    float64(size),
			DPI:     72,
			Hinting: font.HintingNone,
		})
	}
	r.faces[key] = fc
	return fc
}

func (r *renderer) drawer(weight string, size int, col color.Color) *font.Drawer {
	if size < 1 {
		size = 1
	}
	return &font.Drawer{Dst: r.img, Src: image.NewUniform(col), Face: r.face(weight, size)}
}

func (r *renderer) textW(weight string, size int, s string) int {
	d := r.drawer(weight, size, ink)
	return d.MeasureString(s).Ceil()
}

// text 以 (x, y) 为基线左端绘制，返回文本宽度
func (r *renderer) text(x, y int, weight string, size int, s string, col color.Color) int {
	d := r.drawer(weight, size, col)
	d.Dot = fixed.P(x, y)
	d.DrawString(s)
	return d.MeasureString(s).Ceil()
}

func (r *renderer) textRight(xRight, y int, weight string, size int, s string, col color.Color) int {
	w := r.textW(weight, size, s)
	return r.text(xRight-w, y, weight, size, s, col)
}

// textMixedDate 混合排版：数字使用西文字体（如 Playfair/Bebas），汉字使用中文字体
func (r *renderer) textMixedDate(x, y int, dateFontKey, numFontKey string, size int, text string, col color.Color) int {
	curX := x
	runes := []rune(text)
	var numBuf []rune

	flushNum := func() {
		if len(numBuf) > 0 {
			s := string(numBuf)
			curX += r.text(curX, y, numFontKey, size, s, col)
			numBuf = nil
		}
	}

	for _, ru := range runes {
		if ru >= '0' && ru <= '9' {
			numBuf = append(numBuf, ru)
		} else {
			flushNum()
			curX += r.text(curX, y, dateFontKey, size, string(ru), col)
		}
	}
	flushNum()
	return curX - x
}

func (r *renderer) textMixedDateW(dateFontKey, numFontKey string, size int, text string) int {
	w := 0
	runes := []rune(text)
	var numBuf []rune

	flushNum := func() {
		if len(numBuf) > 0 {
			w += r.textW(numFontKey, size, string(numBuf))
			numBuf = nil
		}
	}

	for _, ru := range runes {
		if ru >= '0' && ru <= '9' {
			numBuf = append(numBuf, ru)
		} else {
			flushNum()
			w += r.textW(dateFontKey, size, string(ru))
		}
	}
	flushNum()
	return w
}

// fit 若文本超出 maxW，按比例缩小字号；非法尺寸和宽度使用 1px 安全值。
func (r *renderer) fit(weight string, size int, s string, maxW int) int {
	if size < 1 {
		size = 1
	}
	if maxW <= 0 || s == "" {
		return 1
	}
	w := r.textW(weight, size, s)
	if w <= maxW || w <= 0 {
		return size
	}
	fitted := int(float64(size) * float64(maxW) / float64(w))
	if fitted < 1 {
		return 1
	}
	return fitted
}

func (r *renderer) rect(x0, y0, x1, y1 int, col color.Color) {
	draw.Draw(r.img, image.Rect(x0, y0, x1, y1), image.NewUniform(col), image.Point{}, draw.Src)
}

func (r *renderer) fillCircle(cx, cy, rad float64, col color.Color) {
	r.fillPolys(col, circle(cx, cy, rad))
}

func (r *renderer) fillRoundRect(x0, y0, x1, y1 int, rad float64, col color.Color) {
	if rad <= 0 {
		r.rect(x0, y0, x1, y1, col)
		return
	}
	fx0, fy0, fx1, fy1 := float32(x0), float32(y0), float32(x1), float32(y1)
	fr := float32(rad)
	if maxR := (fx1 - fx0) / 2; fr > maxR {
		fr = maxR
	}
	if maxH := (fy1 - fy0) / 2; fr > maxH {
		fr = maxH
	}
	k := fr * 0.55228475

	var ras vector.Rasterizer
	ras.Reset(r.w, r.h)
	ras.MoveTo(fx0+fr, fy0)
	ras.LineTo(fx1-fr, fy0)
	ras.CubeTo(fx1-fr+k, fy0, fx1, fy0+fr-k, fx1, fy0+fr)
	ras.LineTo(fx1, fy1-fr)
	ras.CubeTo(fx1, fy1-fr+k, fx1-fr+k, fy1, fx1-fr, fy1)
	ras.LineTo(fx0+fr, fy1)
	ras.CubeTo(fx0+fr-k, fy1, fx0, fy1-fr+k, fx0, fy1-fr)
	ras.LineTo(fx0, fy0+fr)
	ras.CubeTo(fx0, fy0+fr-k, fx0+fr-k, fy0, fx0+fr, fy0)
	ras.ClosePath()
	ras.Draw(r.img, r.img.Bounds(), image.NewUniform(col), image.Point{})
}

func (r *renderer) strokeRoundRect(x0, y0, x1, y1 int, rad float64, strokeW float32, col color.Color) {
	if rad <= 0 {
		sw := int(strokeW + 0.5)
		if sw < 1 {
			sw = 1
		}
		r.rect(x0, y0, x1, y0+sw, col)
		r.rect(x0, y1-sw, x1, y1, col)
		r.rect(x0, y0, x0+sw, y1, col)
		r.rect(x1-sw, y0, x1, y1, col)
		return
	}
	fx0, fy0, fx1, fy1 := float32(x0), float32(y0), float32(x1), float32(y1)
	fr := float32(rad)
	if maxR := (fx1 - fx0) / 2; fr > maxR {
		fr = maxR
	}
	if maxH := (fy1 - fy0) / 2; fr > maxH {
		fr = maxH
	}
	k := fr * 0.55228475

	in0, inY0, in1, inY1 := fx0+strokeW, fy0+strokeW, fx1-strokeW, fy1-strokeW
	inR := fr - strokeW
	if inR < 1 {
		inR = 1
	}
	inK := inR * 0.55228475

	var ras vector.Rasterizer
	ras.Reset(r.w, r.h)
	// 顺时针外轮廓
	ras.MoveTo(fx0+fr, fy0)
	ras.LineTo(fx1-fr, fy0)
	ras.CubeTo(fx1-fr+k, fy0, fx1, fy0+fr-k, fx1, fy0+fr)
	ras.LineTo(fx1, fy1-fr)
	ras.CubeTo(fx1, fy1-fr+k, fx1-fr+k, fy1, fx1-fr, fy1)
	ras.LineTo(fx0+fr, fy1)
	ras.CubeTo(fx0+fr-k, fy1, fx0, fy1-fr+k, fx0, fy1-fr)
	ras.LineTo(fx0, fy0+fr)
	ras.CubeTo(fx0, fy0+fr-k, fx0+fr-k, fy0, fx0+fr, fy0)
	ras.ClosePath()

	// 逆时针内轮廓
	ras.MoveTo(in0+inR, inY0)
	ras.CubeTo(in0+inR-inK, inY0, in0, inY0+inR-inK, in0, inY0+inR)
	ras.LineTo(in0, inY1-inR)
	ras.CubeTo(in0, inY1-inR+inK, in0+inR-inK, inY1, in0+inR, inY1)
	ras.LineTo(in1-inR, inY1)
	ras.CubeTo(in1-inR+inK, inY1, in1, inY1-inR+inK, in1, inY1-inR)
	ras.LineTo(in1, inY0+inR)
	ras.CubeTo(in1, inY0+inR-inK, in1-inR+inK, inY0, in1-inR, inY0)
	ras.ClosePath()

	ras.Draw(r.img, r.img.Bounds(), image.NewUniform(col), image.Point{})
}

// ---- 矢量形状 ----

func (r *renderer) fillPolys(col color.Color, polys ...[][2]float64) {
	var ras vector.Rasterizer
	ras.Reset(r.w, r.h)
	for _, pts := range polys {
		for i, p := range pts {
			if i == 0 {
				ras.MoveTo(float32(p[0]), float32(p[1]))
			} else {
				ras.LineTo(float32(p[0]), float32(p[1]))
			}
		}
		ras.ClosePath()
	}
	ras.Draw(r.img, r.img.Bounds(), image.NewUniform(col), image.Point{})
}

func circle(cx, cy, rad float64) [][2]float64 {
	pts := make([][2]float64, 0, 48)
	for i := 0; i < 48; i++ {
		a := 2 * math.Pi * float64(i) / 48
		pts = append(pts, [2]float64{cx + rad*math.Cos(a), cy + rad*math.Sin(a)})
	}
	return pts
}

func thickLine(x0, y0, x1, y1, w float64) [][2]float64 {
	dx, dy := x1-x0, y1-y0
	l := math.Sqrt(dx*dx + dy*dy)
	if l == 0 {
		l = 1
	}
	hw := w / 2
	ux, uy := -dy/l*hw, dx/l*hw
	return [][2]float64{{x0 + ux, y0 + uy}, {x1 + ux, y1 + uy}, {x1 - ux, y1 - uy}, {x0 - ux, y0 - uy}}
}

func rectPoly(x0, y0, x1, y1 float64) [][2]float64 {
	return [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

func sunPolys(cx, cy, s float64) [][][2]float64 {
	pts := [][][2]float64{circle(cx, cy, 0.26*s)}
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		x0 := cx + math.Cos(a)*0.40*s
		y0 := cy + math.Sin(a)*0.40*s
		x1 := cx + math.Cos(a)*0.55*s
		y1 := cy + math.Sin(a)*0.55*s
		pts = append(pts, thickLine(x0, y0, x1, y1, 0.09*s))
	}
	return pts
}

func cloudPolys(cx, cy, s float64) [][][2]float64 {
	return [][][2]float64{
		circle(cx-0.20*s, cy+0.06*s, 0.20*s),
		circle(cx+0.05*s, cy-0.06*s, 0.26*s),
		circle(cx+0.26*s, cy+0.08*s, 0.17*s),
		rectPoly(cx-0.36*s, cy+0.06*s, cx+0.36*s, cy+0.26*s),
	}
}

func drawIcon(r *renderer, kind weather.IconKind, cx, cy, s float64, col color.Color) {
	switch kind {
	case weather.IconSun:
		r.fillPolys(col, sunPolys(cx, cy, s)...)
	case weather.IconPartly:
		r.fillPolys(col, append(sunPolys(cx-0.16*s, cy-0.20*s, 0.62*s), cloudPolys(cx+0.06*s, cy+0.12*s, 0.80*s)...)...)
	case weather.IconCloud:
		r.fillPolys(col, cloudPolys(cx, cy, s)...)
	case weather.IconRain:
		var pts [][][2]float64
		pts = append(pts, cloudPolys(cx, cy-0.12*s, 0.85*s)...)
		for _, ox := range []float64{-0.22, 0.02, 0.24} {
			pts = append(pts, thickLine(cx+ox*s, cy+0.14*s, cx+(ox-0.05)*s, cy+0.32*s, 0.055*s))
		}
		r.fillPolys(col, pts...)
	case weather.IconDrizzle:
		var pts [][][2]float64
		pts = append(pts, cloudPolys(cx, cy-0.12*s, 0.85*s)...)
		for _, ox := range []float64{-0.24, -0.02, 0.20} {
			pts = append(pts, thickLine(cx+ox*s, cy+0.14*s, cx+(ox-0.03)*s, cy+0.24*s, 0.05*s))
		}
		r.fillPolys(col, pts...)
	case weather.IconSleet:
		var pts [][][2]float64
		pts = append(pts, cloudPolys(cx, cy-0.12*s, 0.85*s)...)
		pts = append(pts, thickLine(cx-0.20*s, cy+0.14*s, cx-0.25*s, cy+0.30*s, 0.05*s))
		pts = append(pts, circle(cx+0.0*s, cy+0.22*s, 0.045*s))
		pts = append(pts, thickLine(cx+0.22*s, cy+0.14*s, cx+0.17*s, cy+0.30*s, 0.05*s))
		r.fillPolys(col, pts...)
	case weather.IconSnow:
		var pts [][][2]float64
		pts = append(pts, cloudPolys(cx, cy-0.12*s, 0.85*s)...)
		for _, o := range [][2]float64{{-0.24, 0.20}, {0.0, 0.26}, {0.24, 0.20}} {
			pts = append(pts, circle(cx+o[0]*s, cy+o[1]*s, 0.05*s))
		}
		r.fillPolys(col, pts...)
	case weather.IconThunder:
		var pts [][][2]float64
		pts = append(pts, cloudPolys(cx, cy-0.12*s, 0.85*s)...)
		pts = append(pts, [][2]float64{
			{cx + 0.04*s, cy + 0.10*s},
			{cx - 0.12*s, cy + 0.34*s},
			{cx - 0.02*s, cy + 0.34*s},
			{cx - 0.10*s, cy + 0.56*s},
			{cx + 0.12*s, cy + 0.28*s},
			{cx + 0.02*s, cy + 0.28*s},
			{cx + 0.14*s, cy + 0.10*s},
		})
		r.fillPolys(col, pts...)
	case weather.IconFog:
		var pts [][][2]float64
		pts = append(pts, thickLine(cx-0.30*s, cy-0.16*s, cx+0.30*s, cy-0.16*s, 0.07*s))
		pts = append(pts, thickLine(cx-0.36*s, cy+0.02*s, cx+0.18*s, cy+0.02*s, 0.07*s))
		pts = append(pts, thickLine(cx-0.14*s, cy+0.20*s, cx+0.34*s, cy+0.20*s, 0.07*s))
		r.fillPolys(col, pts...)
	}
}

func (r *renderer) strokeLine(x0, y0, x1, y1, strokeW float64, col color.Color) {
	r.fillPolys(col, thickLine(x0, y0, x1, y1, strokeW))
}

func (r *renderer) strokeCircle(cx, cy, rad, strokeW float64, col color.Color) {
	r1 := float32(rad + strokeW/2)
	r0 := float32(rad - strokeW/2)
	if r0 < 0 {
		r0 = 0
	}
	k1 := r1 * 0.55228475
	k0 := r0 * 0.55228475
	fcx, fcy := float32(cx), float32(cy)

	var ras vector.Rasterizer
	ras.Reset(r.w, r.h)
	// 顺时针外圆
	ras.MoveTo(fcx, fcy-r1)
	ras.CubeTo(fcx+k1, fcy-r1, fcx+r1, fcy-k1, fcx+r1, fcy)
	ras.CubeTo(fcx+r1, fcy+k1, fcx+k1, fcy+r1, fcx, fcy+r1)
	ras.CubeTo(fcx-k1, fcy+r1, fcx-r1, fcy+k1, fcx-r1, fcy)
	ras.CubeTo(fcx-r1, fcy-k1, fcx-r1, fcy-r1, fcx, fcy-r1)
	ras.ClosePath()

	// 逆时针内圆
	if r0 > 0 {
		ras.MoveTo(fcx, fcy-r0)
		ras.CubeTo(fcx-k0, fcy-r0, fcx-r0, fcy-k0, fcx-r0, fcy)
		ras.CubeTo(fcx-r0, fcy+k0, fcx-k0, fcy+r0, fcx, fcy+r0)
		ras.CubeTo(fcx+k0, fcy+r0, fcx+r0, fcy+k0, fcx+r0, fcy)
		ras.CubeTo(fcx+r0, fcy-k0, fcx+k0, fcy-r0, fcx, fcy-r0)
		ras.ClosePath()
	}

	ras.Draw(r.img, r.img.Bounds(), image.NewUniform(col), image.Point{})
}

func (r *renderer) strokePolyline(pts [][2]float64, strokeW float64, col color.Color) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		r.strokeLine(pts[i][0], pts[i][1], pts[i+1][0], pts[i+1][1], strokeW, col)
		if strokeW > 1.5 {
			r.fillCircle(pts[i+1][0], pts[i+1][1], strokeW/2, col)
		}
	}
}

func (r *renderer) strokeArcLight(cx, cy, rad, startAngle, endAngle, strokeW float64, col color.Color) {
	steps := 16
	var pts [][2]float64
	for i := 0; i <= steps; i++ {
		ang := startAngle + float64(i)*(endAngle-startAngle)/float64(steps)
		pts = append(pts, [2]float64{cx + rad*math.Cos(ang), cy + rad*math.Sin(ang)})
	}
	r.strokePolyline(pts, strokeW, col)
}

func (r *renderer) drawCartoonSun(cx, cy, radius float64, now time.Time) {
	// 1. 放射状阳光光芒（8道带微圆角光柱）
	numRays := 8
	rayInner := radius + 3.0
	rayOuter := radius + 8.5
	for i := 0; i < numRays; i++ {
		ang := float64(i) * (2 * math.Pi / float64(numRays))
		x1 := cx + math.Cos(ang)*rayInner
		y1 := cy + math.Sin(ang)*rayInner
		x2 := cx + math.Cos(ang)*rayOuter
		y2 := cy + math.Sin(ang)*rayOuter
		r.strokeLine(x1, y1, x2, y2, 2.2, ink)
		r.fillCircle(x2, y2, 1.1, ink)
	}

	// 2. 太阳脸蛋主体：纯白底色与高对比度轮廓
	r.fillCircle(cx, cy, radius, paper)
	r.strokeCircle(cx, cy, radius, 2.0, ink)

	hour := now.Hour()

	// 3. 腮红微晕（柔和浅灰）
	blushOffsetX := radius * 0.58
	blushOffsetY := radius * 0.18
	r.fillCircle(cx-blushOffsetX, cy+blushOffsetY, radius*0.14, inkLine)
	r.fillCircle(cx+blushOffsetX, cy+blushOffsetY, radius*0.14, inkLine)

	eyeOffsetX := radius * 0.36
	eyeOffsetY := -radius * 0.12

	if hour < 10 {
		// 【清晨刚起床：温和微笑】
		eyeR := radius * 0.15
		r.fillCircle(cx-eyeOffsetX, cy+eyeOffsetY, eyeR, ink)
		r.fillCircle(cx-eyeOffsetX-eyeR*0.3, cy+eyeOffsetY-eyeR*0.3, eyeR*0.4, paper)
		r.fillCircle(cx+eyeOffsetX, cy+eyeOffsetY, eyeR, ink)
		r.fillCircle(cx+eyeOffsetX-eyeR*0.3, cy+eyeOffsetY-eyeR*0.3, eyeR*0.4, paper)

		// 眉毛：清晨微挑轻柔眉
		r.strokeLine(cx-eyeOffsetX-radius*0.18, cy+eyeOffsetY-radius*0.30, cx-eyeOffsetX+radius*0.14, cy+eyeOffsetY-radius*0.34, 1.3, inkSoft)
		r.strokeLine(cx+eyeOffsetX-radius*0.14, cy+eyeOffsetY-radius*0.34, cx+eyeOffsetX+radius*0.18, cy+eyeOffsetY-radius*0.30, 1.3, inkSoft)

		// 嘴巴：恬淡温柔微笑
		smileY := cy + radius*0.18
		smileW := radius * 0.30
		smileH := radius * 0.18
		var smilePts [][2]float64
		for i := 0; i <= 10; i++ {
			t := float64(i) / 10.0
			px := cx - smileW + 2*smileW*t
			py := smileY + smileH*math.Sin(math.Pi*t)
			smilePts = append(smilePts, [2]float64{px, py})
		}
		r.strokePolyline(smilePts, 1.8, ink)

	} else if hour >= 10 && hour < 15 {
		// 【中午：开心大笑】
		eyeW := radius * 0.22
		eyeH := radius * 0.16
		var eyeLeft, eyeRight [][2]float64
		for i := 0; i <= 8; i++ {
			t := float64(i) / 8.0
			pxL := cx - eyeOffsetX - eyeW + 2*eyeW*t
			pyL := cy + eyeOffsetY - eyeH*math.Sin(math.Pi*t)
			eyeLeft = append(eyeLeft, [2]float64{pxL, pyL})

			pxR := cx + eyeOffsetX - eyeW + 2*eyeW*t
			pyR := cy + eyeOffsetY - eyeH*math.Sin(math.Pi*t)
			eyeRight = append(eyeRight, [2]float64{pxR, pyR})
		}
		r.strokePolyline(eyeLeft, 2.0, ink)
		r.strokePolyline(eyeRight, 2.0, ink)

		// 嘴巴：张嘴开心大笑（半圆笑口）
		mouthW := radius * 0.34
		mouthH := radius * 0.26
		mouthY := cy + radius*0.12
		var mouthPts [][2]float64
		mouthPts = append(mouthPts, [2]float64{cx - mouthW, mouthY})
		for i := 0; i <= 12; i++ {
			t := float64(i) / 12.0
			px := cx - mouthW + 2*mouthW*t
			py := mouthY + mouthH*math.Sin(math.Pi*t)
			mouthPts = append(mouthPts, [2]float64{px, py})
		}
		r.fillPolys(ink, mouthPts)
		r.fillCircle(cx, mouthY+mouthH*0.7, radius*0.14, paper)

	} else {
		// 【傍晚/日落前：安详惬意的笑】
		eyeW := radius * 0.22
		eyeH := radius * 0.12
		var eyeLeft, eyeRight [][2]float64
		for i := 0; i <= 8; i++ {
			t := float64(i) / 8.0
			pxL := cx - eyeOffsetX - eyeW + 2*eyeW*t
			pyL := cy + eyeOffsetY + eyeH*math.Sin(math.Pi*t)
			eyeLeft = append(eyeLeft, [2]float64{pxL, pyL})

			pxR := cx + eyeOffsetX - eyeW + 2*eyeW*t
			pyR := cy + eyeOffsetY + eyeH*math.Sin(math.Pi*t)
			eyeRight = append(eyeRight, [2]float64{pxR, pyR})
		}
		r.strokePolyline(eyeLeft, 1.8, ink)
		r.strokePolyline(eyeRight, 1.8, ink)

		// 嘴巴：安详舒心的微扬嘴角
		smileY := cy + radius*0.18
		smileW := radius * 0.26
		smileH := radius * 0.15
		var smilePts [][2]float64
		for i := 0; i <= 10; i++ {
			t := float64(i) / 10.0
			px := cx - smileW + 2*smileW*t
			py := smileY + smileH*math.Sin(math.Pi*t)
			smilePts = append(smilePts, [2]float64{px, py})
		}
		r.strokePolyline(smilePts, 1.8, ink)
	}
}

func (r *renderer) drawCartoonMoon(cx, cy, radius float64, now time.Time, lunarDay int) {
	if lunarDay <= 0 {
		lunarDay = 15
	}
	hour := now.Hour()

	// 1. 依据月相绘制月亮本体几何轮廓
	var pts [][2]float64
	n := 32

	switch {
	case lunarDay == 15 || lunarDay == 16:
		// 满月：饱满圆盘
		r.fillCircle(cx, cy, radius, paper)
		r.strokeCircle(cx, cy, radius, 2.0, ink)

	case lunarDay >= 7 && lunarDay <= 8:
		// 上弦月：半月（右侧半圆发光，D形）
		pts = append(pts, [2]float64{cx, cy - radius})
		pts = append(pts, [2]float64{cx, cy + radius})
		for i := 0; i <= n/2; i++ {
			ang := math.Pi*0.5 - float64(i)*math.Pi/float64(n/2)
			pts = append(pts, [2]float64{cx + radius*math.Cos(ang), cy + radius*math.Sin(ang)})
		}
		r.fillPolys(paper, pts)
		r.strokePolyline(pts, 1.8, ink)
		r.strokeLine(pts[len(pts)-1][0], pts[len(pts)-1][1], pts[0][0], pts[0][1], 1.8, ink)

	case lunarDay >= 22 && lunarDay <= 23:
		// 下弦月：半月（左侧半圆发光，反D形）
		pts = append(pts, [2]float64{cx, cy - radius})
		pts = append(pts, [2]float64{cx, cy + radius})
		for i := 0; i <= n/2; i++ {
			ang := math.Pi*0.5 + float64(i)*math.Pi/float64(n/2)
			pts = append(pts, [2]float64{cx + radius*math.Cos(ang), cy + radius*math.Sin(ang)})
		}
		r.fillPolys(paper, pts)
		r.strokePolyline(pts, 1.8, ink)
		r.strokeLine(pts[len(pts)-1][0], pts[len(pts)-1][1], pts[0][0], pts[0][1], 1.8, ink)

	case lunarDay >= 2 && lunarDay <= 6:
		// 峨眉月：右侧发光弯月
		for i := 0; i <= n; i++ {
			ang := -math.Pi*0.52 + float64(i)*(math.Pi*1.04)/float64(n)
			pts = append(pts, [2]float64{cx + radius*math.Cos(ang), cy + radius*math.Sin(ang)})
		}
		innerR := radius * 0.85
		innerCX := cx + radius*0.38
		innerCY := cy
		for i := n; i >= 0; i-- {
			ang := -math.Pi*0.48 + float64(i)*(math.Pi*0.96)/float64(n)
			pts = append(pts, [2]float64{innerCX + innerR*math.Cos(ang), innerCY + innerR*math.Sin(ang)})
		}
		r.fillPolys(paper, pts)
		r.strokePolyline(pts, 1.8, ink)
		r.strokeLine(pts[len(pts)-1][0], pts[len(pts)-1][1], pts[0][0], pts[0][1], 1.8, ink)

	case lunarDay >= 24 && lunarDay <= 29:
		// 残月：左侧发光弯月
		for i := 0; i <= n; i++ {
			ang := -math.Pi*0.52 + float64(i)*(math.Pi*1.04)/float64(n)
			pts = append(pts, [2]float64{cx - radius*math.Cos(ang), cy + radius*math.Sin(ang)})
		}
		innerR := radius * 0.85
		innerCX := cx - radius*0.38
		innerCY := cy
		for i := n; i >= 0; i-- {
			ang := -math.Pi*0.48 + float64(i)*(math.Pi*0.96)/float64(n)
			pts = append(pts, [2]float64{innerCX - innerR*math.Cos(ang), innerCY + innerR*math.Sin(ang)})
		}
		r.fillPolys(paper, pts)
		r.strokePolyline(pts, 1.8, ink)
		r.strokeLine(pts[len(pts)-1][0], pts[len(pts)-1][1], pts[0][0], pts[0][1], 1.8, ink)

	case lunarDay >= 9 && lunarDay <= 14:
		// 盈凸月：大部分满（右凸）
		r.fillCircle(cx, cy, radius, paper)
		r.strokeCircle(cx, cy, radius, 2.0, ink)
		r.strokeArcLight(cx, cy, radius, -math.Pi*0.5, math.Pi*0.5, 1.5, inkBorder)

	case lunarDay >= 17 && lunarDay <= 21:
		// 亏凸月：大部分满（左凸）
		r.fillCircle(cx, cy, radius, paper)
		r.strokeCircle(cx, cy, radius, 2.0, ink)
		r.strokeArcLight(cx, cy, radius, math.Pi*0.5, math.Pi*1.5, 1.5, inkBorder)

	default: // 新月
		r.strokeCircle(cx, cy, radius, 1.5, inkBorder)
	}

	// 2. 依据时间绘制月亮萌系表情
	faceCX := cx
	faceCY := cy
	if lunarDay >= 2 && lunarDay <= 6 {
		faceCX = cx + radius*0.42
	} else if (lunarDay >= 7 && lunarDay <= 8) || (lunarDay >= 9 && lunarDay <= 14) {
		faceCX = cx + radius*0.28
	} else if (lunarDay >= 22 && lunarDay <= 23) || (lunarDay >= 17 && lunarDay <= 21) {
		faceCX = cx - radius*0.28
	} else if lunarDay >= 24 && lunarDay <= 29 {
		faceCX = cx - radius*0.42
	}

	if hour < 22 && hour >= 17 {
		// 【傍晚至22点前：精神奕奕·清醒微笑】
		eyeR := radius * 0.14
		r.fillCircle(faceCX-radius*0.18, faceCY-radius*0.10, eyeR, ink)
		r.fillCircle(faceCX-radius*0.18-eyeR*0.3, faceCY-radius*0.10-eyeR*0.3, eyeR*0.4, paper)
		r.fillCircle(faceCX+radius*0.18, faceCY-radius*0.10, eyeR, ink)
		r.fillCircle(faceCX+radius*0.18-eyeR*0.3, faceCY-radius*0.10-eyeR*0.3, eyeR*0.4, paper)

		smileY := faceCY + radius*0.16
		smileW := radius * 0.18
		smileH := radius * 0.12
		var smilePts [][2]float64
		for i := 0; i <= 6; i++ {
			t := float64(i) / 6.0
			px := faceCX - smileW + 2*smileW*t
			py := smileY + smileH*math.Sin(math.Pi*t)
			smilePts = append(smilePts, [2]float64{px, py})
		}
		r.strokePolyline(smilePts, 1.4, ink)

		r.fillCircle(faceCX-radius*0.32, faceCY+radius*0.08, radius*0.08, inkLine)
		r.fillCircle(faceCX+radius*0.32, faceCY+radius*0.08, radius*0.08, inkLine)

	} else if hour == 22 {
		// 【22点后慢慢入睡：困倦打哈欠】
		eyeW := radius * 0.16
		eyeH := radius * 0.08
		var eyeLeft, eyeRight [][2]float64
		for i := 0; i <= 6; i++ {
			t := float64(i) / 6.0
			pxL := faceCX - radius*0.18 - eyeW + 2*eyeW*t
			pyL := faceCY - radius*0.08 + eyeH*math.Sin(math.Pi*t)
			eyeLeft = append(eyeLeft, [2]float64{pxL, pyL})

			pxR := faceCX + radius*0.18 - eyeW + 2*eyeW*t
			pyR := faceCY - radius*0.08 + eyeH*math.Sin(math.Pi*t)
			eyeRight = append(eyeRight, [2]float64{pxR, pyR})
		}
		r.strokePolyline(eyeLeft, 1.5, ink)
		r.strokePolyline(eyeRight, 1.5, ink)

		// 可爱打哈欠小圆嘴
		r.strokeCircle(faceCX, faceCY+radius*0.18, radius*0.11, 1.4, ink)
		r.fillCircle(faceCX, faceCY+radius*0.18, radius*0.09, inkSoft)

		// 头顶冒出的小 "z"
		r.text(int(faceCX+radius*0.48), int(faceCY-radius*0.35), "bold", int(radius*0.60), "z", inkSoft)

	} else {
		// 【23点至次日黎明：安详甜睡熟睡】
		eyeW := radius * 0.18
		eyeH := radius * 0.12
		var eyePts [][2]float64
		for i := 0; i <= 8; i++ {
			t := float64(i) / 8.0
			px := faceCX - eyeW + 2*eyeW*t
			py := faceCY - radius*0.08 + eyeH*math.Sin(math.Pi*t)
			eyePts = append(eyePts, [2]float64{px, py})
		}
		r.strokePolyline(eyePts, 1.6, ink)

		smileY := faceCY + radius*0.18
		smileW := radius * 0.14
		smileH := radius * 0.10
		var smilePts [][2]float64
		for i := 0; i <= 6; i++ {
			t := float64(i) / 6.0
			px := faceCX - smileW + 2*smileW*t
			py := smileY + smileH*math.Sin(math.Pi*t)
			smilePts = append(smilePts, [2]float64{px, py})
		}
		r.strokePolyline(smilePts, 1.4, ink)

		r.text(int(cx+radius*0.45), int(cy-radius*0.35), "bold", int(radius*0.65), "z", inkSoft)
		r.text(int(cx+radius*0.85), int(cy-radius*0.75), "bold", int(radius*0.85), "Z", ink)

		r.drawSparkle(cx-radius*1.4, cy-radius*0.5, radius*0.35, inkSoft)
		r.drawSparkle(cx+radius*1.5, cy+radius*0.4, radius*0.28, inkMuted)
	}
}

func (r *renderer) drawSparkle(cx, cy, s float64, col color.Color) {
	r.strokeLine(cx-s, cy, cx+s, cy, 1.2, col)
	r.strokeLine(cx, cy-s, cx, cy+s, 1.2, col)
	r.fillCircle(cx, cy, s*0.25, col)
}

func (r *renderer) drawMiniSun(cx, cy, rad float64, col color.Color) {
	for i := 0; i < 4; i++ {
		ang := float64(i)*(math.Pi/2) + math.Pi/4
		x1 := cx + math.Cos(ang)*(rad+1.5)
		y1 := cy + math.Sin(ang)*(rad+1.5)
		x2 := cx + math.Cos(ang)*(rad+3.5)
		y2 := cy + math.Sin(ang)*(rad+3.5)
		r.strokeLine(x1, y1, x2, y2, 1.2, col)
	}
	r.fillCircle(cx, cy, rad, col)
}

func (r *renderer) drawMiniMoon(cx, cy, rad float64, col color.Color) {
	r.fillCircle(cx, cy, rad, col)
	r.fillCircle(cx+rad*0.4, cy-rad*0.2, rad*0.8, cardBg)
}

func (r *renderer) drawMiniLeaf(cx, cy, s float64, col color.Color) {
	steps := 10
	var topPts [][2]float64
	var botPts [][2]float64
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := (cx - s) + 2*s*t
		yTop := cy - s*0.65*math.Sin(math.Pi*t)
		yBot := cy + s*0.65*math.Sin(math.Pi*t)
		topPts = append(topPts, [2]float64{x, yTop})
		botPts = append(botPts, [2]float64{x, yBot})
	}
	r.strokePolyline(topPts, 1.4, col)
	r.strokePolyline(botPts, 1.4, col)
	r.strokeLine(cx-s*0.75, cy, cx+s*0.8, cy, 1.1, col)
}

func (r *renderer) drawWiFiIcon(cx, cy, size float64, col color.Color) {
	cx = math.Round(cx)
	baseY := math.Round(cy + size*0.30)
	// 底部信号基点：像素对齐的规整实心小圆，消除边缘毛糙与歪斜
	r.fillCircle(cx, baseY, 2.0, col)
	// 中层同心扩散弧：弧度对称平滑，线宽适中
	r.strokeArcLight(cx, baseY, size*0.48, -math.Pi*0.75, -math.Pi*0.25, 1.8, col)
	// 外层同心扩散弧
	r.strokeArcLight(cx, baseY, size*0.84, -math.Pi*0.75, -math.Pi*0.25, 1.8, col)
}

func (r *renderer) drawBatteryIcon(x, y, w, h float64, percent int, charging bool, col color.Color) {
	x = math.Round(x)
	y = math.Round(y)
	w = math.Round(w)
	h = math.Round(h)

	// 1. 电池主体外壳轮廓
	r.strokeRoundRect(int(x), int(y), int(x+w), int(y+h), 2.5, 1.3, col)
	// 2. 右侧正极端子凸起
	capW := 2.5
	capH := math.Round(h * 0.44)
	capY := math.Round(y + (h-capH)/2)
	r.fillRoundRect(int(x+w), int(capY), int(x+w+capW), int(capY+capH), 1.0, col)

	if charging {
		// 3. 充电状态：在电池中心绘制纯矢量实心闪电符号，彻底杜绝 Unicode 缺字叉号
		cx := x + w/2
		cy := y + h/2
		s := h * 0.88
		bolt := [][2]float64{
			{cx + 0.10*s, cy - 0.48*s},
			{cx - 0.20*s, cy + 0.04*s},
			{cx - 0.02*s, cy + 0.04*s},
			{cx - 0.14*s, cy + 0.48*s},
			{cx + 0.20*s, cy - 0.04*s},
			{cx + 0.02*s, cy - 0.04*s},
		}
		r.fillPolys(col, bolt)
	} else {
		// 4. 正常电量状态：内部填充电量条
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		pad := 2.0
		innerMaxW := w - 2*pad
		innerW := innerMaxW * float64(percent) / 100.0
		if innerW > 1.0 {
			r.fillRoundRect(int(x+pad), int(y+pad), int(x+pad+innerW), int(y+h-pad), 1.2, col)
		}
	}
}
