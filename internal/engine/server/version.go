package server

import (
	"encoding/json"
	"net/http"
)

type engineVersion struct {
	version string
}

func (e *EngineServer) versionHandler(w http.ResponseWriter, r *http.Request) {
	v := engineVersion{
		version: "0.0.1",
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
