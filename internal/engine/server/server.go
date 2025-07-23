package server

import (
	"net/http"
	"topi/internal/shared/database"
)

type EngineServer struct {
	db database.Service
}

func NewEngineServer(db database.Service) *EngineServer {

	return &EngineServer{
		db: db,
	}
}

func (e *EngineServer) RegisterRouters(mux *http.ServeMux) {
	mux.HandleFunc("/version", e.versionHandler)
	mux.HandleFunc("/hello", e.helloHandler)
}
