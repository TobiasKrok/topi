package server

import (
	"net/http"
	"topi/internal/shared/database"
	"topi/internal/shared/rabbitmq"
)

type EngineServer struct {
	db       database.Service
	rabbitmq *rabbitmq.RabbitMQ
}

func NewEngineServer() *EngineServer {

	return &EngineServer{
		db:       database.New(),
		rabbitmq: rabbitmq.New(),
	}
}

func (e *EngineServer) RegisterRouters(mux *http.ServeMux) {
	mux.HandleFunc("/version", e.versionHandler)
	mux.HandleFunc("/hello", e.helloHandler)
	mux.HandleFunc("/event", e.eventHandler) // git webhooks

}
