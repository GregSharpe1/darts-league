package httpapi

import "net/http"

type VersionHandler struct {
	version string
}

func NewVersionHandler(version string) VersionHandler {
	return VersionHandler{version: version}
}

func (h VersionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/version", h.handleVersion)
}

func (h VersionHandler) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": h.version})
}
