package server

import (
	"net/http"
	"os"
	"topi/internal/shared/database"
	"topi/internal/shared/git"
	"topi/internal/shared/rabbitmq"
)

type EngineServer struct {
	db       database.Service
	rabbitmq *rabbitmq.RabbitMQ
	git      git.GitService
}

func NewEngineServer() *EngineServer {
	gitInstance := os.Getenv("TOPI_GIT_INSTANCE")
	var gitService git.GitService
	switch gitInstance {
	case "gitea":
		gitService = git.NewGiteaService()

	}
	return &EngineServer{
		db:       database.New(),
		rabbitmq: rabbitmq.New(),
		git:      gitService,
	}
}

func (e *EngineServer) RegisterRouters(mux *http.ServeMux) {
	mux.HandleFunc("/version", e.versionHandler)
	mux.HandleFunc("/hello", e.helloHandler)
	mux.HandleFunc("/event", e.eventHandler) // git webhooks

}
