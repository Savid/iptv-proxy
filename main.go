// Package main implements the IPTV proxy server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/handlers"
	"github.com/savid/iptv-proxy/internal/data"
	"github.com/savid/iptv-proxy/internal/hardware"
	"github.com/sirupsen/logrus"
)

func main() {
	// Configure logrus
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	cfg, err := config.New()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level based on config
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to parse log level")
	}
	logrus.SetLevel(level)

	logger := logrus.StandardLogger()

	// List available hardware devices
	if cfg.TranscodeMode == "transcode" {
		stdLogger := log.New(logger.Writer(), "", 0)
		detector := hardware.NewDetector(stdLogger)
		devices, err := detector.DetectAllDevices()
		if err == nil && len(devices) > 0 {
			logger.Info("Available hardware devices:")
			for _, dev := range devices {
				logger.WithFields(logrus.Fields{
					"type":         dev.Type,
					"id":           dev.DeviceID,
					"name":         dev.DeviceName,
					"capabilities": dev.Capabilities,
				}).Info("  Device")
			}
		}
	}

	// Create store and fetcher
	store := data.NewStore()
	store.SetTestChannelsEnabled(cfg.EnableTestChannels)
	fetcher := data.NewFetcher(cfg, logger)

	// Perform initial data fetch (blocking)
	logger.Info("Fetching initial data...")
	result, err := fetcher.FetchAll()
	if err != nil {
		logger.WithError(err).Fatal("Failed to fetch initial data")
	}
	store.SetM3U(result.M3U.Raw, result.M3U.Channels)
	store.SetEPG(result.EPG.Raw, result.EPG.Filtered)
	logger.Info("Initial data loaded successfully")

	// Start background refresh manager
	refresher := data.NewRefresher(store, fetcher, cfg.RefreshInterval, logger)
	ctx, cancel := context.WithCancel(context.Background())
	go refresher.Start(ctx)

	mux := http.NewServeMux()
	setupRoutes(mux, cfg, store, logger)

	// Apply logging middleware to the mux
	handler := handlers.LoggingMiddleware(logger)(mux)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port),
		Handler:      handler,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down server...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Failed to gracefully shutdown")
		}
	}()

	logger.WithFields(logrus.Fields{
		"bind": cfg.BindAddr,
		"port": cfg.Port,
	}).Info("Starting IPTV proxy server")
	logger.WithField("endpoint", fmt.Sprintf("%s/iptv.m3u", cfg.BaseURL)).Info("M3U endpoint")
	logger.WithField("endpoint", fmt.Sprintf("%s/epg.xml", cfg.BaseURL)).Info("EPG endpoint")

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.WithError(err).Fatal("Failed to start server")
	}

	<-ctx.Done()
	logger.Info("Server stopped")
	cancel()
}

func setupRoutes(mux *http.ServeMux, cfg *config.Config, store *data.Store, logger *logrus.Logger) {
	// Tuner advertising routes
	mux.HandleFunc("/", handlers.RootXMLHandler(cfg))
	mux.HandleFunc("/discovery.json", handlers.DiscoveryHandler(cfg))
	mux.HandleFunc("/discover.json", handlers.DiscoveryHandler(cfg)) // Plex compatibility
	mux.HandleFunc("/lineup.json", handlers.LineupHandler(cfg, store))
	mux.HandleFunc("/lineup_status.json", handlers.LineupStatusHandler())

	m3uHandler := handlers.NewM3UHandler(store, cfg, logger)
	epgHandler := handlers.NewEPGHandler(store, cfg, logger)

	// Use transcoding handler when transcode mode is not "copy"
	if cfg.TranscodeMode != "copy" {
		// Create a standard logger wrapper for logrus
		stdLogger := log.New(logger.Writer(), "", 0)
		streamHandler, err := handlers.NewStreamV2Handler(cfg, stdLogger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to create transcoding stream handler")
		}
		logger.WithFields(logrus.Fields{
			"video_codec": cfg.VideoCodec,
			"audio_codec": cfg.AudioCodec,
		}).Info("Using transcoding stream handler")
		mux.Handle("/stream/", streamHandler)
	} else {
		streamHandler := handlers.NewStreamHandler(logger)
		logger.Info("Using direct stream handler (no transcoding)")
		mux.Handle("/stream/", streamHandler)
	}

	mux.Handle("/iptv.m3u", m3uHandler)
	mux.Handle("/epg.xml", epgHandler)

	// Add test channel handlers if enabled
	if cfg.EnableTestChannels {
		mux.HandleFunc("/test/", handlers.TestChannelHandler)
		mux.HandleFunc("/test-icon/", handlers.TestIconHandler)
	}

	// Debug endpoints for troubleshooting
	mux.HandleFunc("/debug", handlers.DebugHandler)
	mux.HandleFunc("/plex-debug", handlers.PlexDebugHandler)

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
