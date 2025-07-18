package handlers

import (
	"net/http"

	"github.com/savid/iptv-proxy/internal/data"
	"github.com/sirupsen/logrus"
)

// EPGHandler handles HTTP requests for EPG (Electronic Program Guide) data.
type EPGHandler struct {
	store  *data.Store
	logger *logrus.Logger
}

// NewEPGHandler creates a new EPG handler instance.
func NewEPGHandler(store *data.Store, logger *logrus.Logger) *EPGHandler {
	return &EPGHandler{
		store:  store,
		logger: logger,
	}
}

func (h *EPGHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	data, ok := h.store.GetEPG()
	if !ok {
		h.logger.Error("EPG data not available")
		http.Error(w, "EPG data not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(data)
}
