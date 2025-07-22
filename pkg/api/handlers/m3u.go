package handlers

import (
	"net/http"

	"github.com/savid/iptv-proxy/config"
	"github.com/savid/iptv-proxy/pkg/data"
	"github.com/savid/iptv-proxy/pkg/m3u"
	"github.com/sirupsen/logrus"
)

// M3UHandler handles HTTP requests for M3U playlists.
type M3UHandler struct {
	store  *data.Store
	config *config.Config
	logger *logrus.Logger
}

// NewM3UHandler creates a new M3U handler instance.
func NewM3UHandler(store *data.Store, cfg *config.Config, logger *logrus.Logger) *M3UHandler {
	return &M3UHandler{
		store:  store,
		config: cfg,
		logger: logger,
	}
}

func (h *M3UHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	data, _, ok := h.store.GetM3U()
	if !ok {
		h.logger.Error("M3U data not available")
		http.Error(w, "M3U data not available", http.StatusServiceUnavailable)
		return
	}

	// Convert to string for processing
	m3uContent := string(data)

	// Add test channels if enabled
	if h.config.EnableTestChannels {
		m3uContent = m3u.AppendTestChannels(m3uContent, h.config.BaseURL)
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write([]byte(m3uContent))
}
