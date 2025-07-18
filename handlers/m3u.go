package handlers

import (
	"net/http"

	"github.com/savid/iptv-proxy/internal/data"
	"github.com/sirupsen/logrus"
)

// M3UHandler handles HTTP requests for M3U playlists.
type M3UHandler struct {
	store  *data.Store
	logger *logrus.Logger
}

// NewM3UHandler creates a new M3U handler instance.
func NewM3UHandler(store *data.Store, logger *logrus.Logger) *M3UHandler {
	return &M3UHandler{
		store:  store,
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

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write(data)
}
