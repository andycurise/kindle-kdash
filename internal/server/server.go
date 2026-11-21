// Package server provides the cross-platform rendering service and browser settings API.
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/dashboard"
	"github.com/andycurise/kindle-kdash/internal/weather"
)

var Version = "dev"

// RefreshFunc asks a Kindle client to refresh immediately. The HTTP polling
// client does not depend on this optional path.
type RefreshFunc func(context.Context, *config.Config) (string, error)

type Server struct {
	loc        *time.Location
	dashboard  *dashboard.Dashboard
	landscapeW int
	landscapeH int
	portraitW  int
	portraitH  int
	startTime  time.Time
	configPath string
	refresh    RefreshFunc
	offline    bool

	configMu   sync.RWMutex
	cfg        *config.Config
	renderMu   sync.Mutex
	weatherMu  sync.RWMutex
	weather    *weather.Weather
	batteryMu  sync.RWMutex
	battery    int
	charging   bool
	hasPower   bool
	shutdownCh chan struct{}
	shutdown   sync.Once
}

func New(cfg *config.Config, loc *time.Location, dash *dashboard.Dashboard, landscapeW, landscapeH, portraitW, portraitH int, configPath string) *Server {
	return &Server{
		cfg:        cfg,
		loc:        loc,
		dashboard:  dash,
		landscapeW: landscapeW,
		landscapeH: landscapeH,
		portraitW:  portraitW,
		portraitH:  portraitH,
		startTime:  time.Now(),
		configPath: configPath,
		refresh:    refreshOverSSH,
		battery:    100,
		hasPower:   true,
		shutdownCh: make(chan struct{}),
	}
}

// SetRefreshFunc replaces the optional device refresher, primarily for tests.
func (s *Server) SetRefreshFunc(fn RefreshFunc) {
	if fn != nil {
		s.refresh = fn
	}
}

// UseStaticWeather installs a fixed snapshot and disables all weather network
// requests. It is intended for local development and browser acceptance tests.
func (s *Server) UseStaticWeather(snapshot *weather.Weather) {
	s.weatherMu.Lock()
	s.weather = snapshot
	s.offline = true
	s.weatherMu.Unlock()
}

func (s *Server) requestShutdown() {
	s.shutdown.Do(func() {
		close(s.shutdownCh)
	})
}

func (s *Server) configSnapshot() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.cfg.Clone()
}

func (s *Server) weatherSnapshot() *weather.Weather {
	s.weatherMu.RLock()
	defer s.weatherMu.RUnlock()
	return s.weather
}

func (s *Server) StartWeatherLoop(ctx context.Context) {
	now := time.Now().In(s.loc)
	initial := weather.LoadCached(s.configSnapshot(), now)
	s.weatherMu.Lock()
	s.weather = initial
	s.weatherMu.Unlock()

	go func() {
		nextFetch := time.Now().Add(500 * time.Millisecond)
		for {
			timer := time.NewTimer(time.Until(nextFetch))
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}

			cfg := s.configSnapshot()
			updated := weather.Get(cfg, time.Now().In(s.loc))
			s.weatherMu.Lock()
			s.weather, nextFetch = weather.ScheduleNext(s.weather, updated, cfg.WeatherRefreshSeconds)
			s.weatherMu.Unlock()
		}
	}()
}

func (s *Server) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !s.offline {
		s.StartWeatherLoop(runCtx)
	}
	cfg := s.configSnapshot()
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("kdash 服务端监听 http://%s", cfg.ListenAddr)
		log.Printf("设置页: http://%s/  Kindle 图片: http://%s/screen.png", cfg.ListenAddr, cfg.ListenAddr)
		errCh <- httpServer.ListenAndServe()
	}()

	shutdownServer := func() error {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return shutdownServer()
	case <-s.shutdownCh:
		return shutdownServer()
	}
}

func publicConfig(cfg *config.Config) *config.Config {
	out := cfg.Clone()
	out.SettingsToken = ""
	out.RuntimeSettingsToken = ""
	out.KindleSSH.PrivateKeyPath = ""
	out.KindleSSH.KnownHostsPath = ""
	return out
}

func (s *Server) dashboardData(now time.Time, w *weather.Weather) dashboard.DashboardData {
	s.batteryMu.RLock()
	defer s.batteryMu.RUnlock()
	return dashboard.DashboardData{
		Now:        now,
		Day:        dashboard.DayInfo(now),
		Weather:    w,
		Battery:    s.battery,
		IsCharging: s.charging,
		HasBattery: s.hasPower,
	}
}
