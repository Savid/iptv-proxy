// Package main implements the IPTV proxy server.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/handlers"
	"github.com/savid/iptv-proxy/internal/data"
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

	// Create store and fetcher
	store := data.NewStore()
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
	setupRoutes(mux, store, logger)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
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

	logger.WithField("port", cfg.Port).Info("Starting IPTV proxy server")
	logger.WithField("endpoint", fmt.Sprintf("%s/iptv.m3u", cfg.BaseURL)).Info("M3U endpoint")
	logger.WithField("endpoint", fmt.Sprintf("%s/epg.xml", cfg.BaseURL)).Info("EPG endpoint")

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.WithError(err).Fatal("Failed to start server")
	}

	<-ctx.Done()
	logger.Info("Server stopped")
	cancel()
}

func setupRoutes(mux *http.ServeMux, store *data.Store, logger *logrus.Logger) {
	m3uHandler := handlers.NewM3UHandler(store, logger)
	epgHandler := handlers.NewEPGHandler(store, logger)
	streamHandler := handlers.NewStreamHandler(logger)

	mux.Handle("/iptv.m3u", m3uHandler)
	mux.Handle("/epg.xml", epgHandler)
	mux.Handle("/stream/", streamHandler)

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
