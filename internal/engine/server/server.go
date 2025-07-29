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
	// git setup
	gitInstance := os.Getenv("TOPI_GIT_INSTANCE")
	var gitService git.GitService
	switch gitInstance {
	case "gitea":
		gitService = git.NewGiteaService()

	}

	// rabbitmq setup
	rmq := rabbitmq.New()
	err := rmq.Channel.ExchangeDeclare("topi", "topic", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	q, err := rmq.Channel.QueueDeclare("engine.trigger", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	err = rmq.Channel.QueueBind(q.Name, "engine.trigger", "topi", false, nil)
	if err != nil {
		panic(err)
	}

	return &EngineServer{
		db:       database.New(),
		rabbitmq: rmq,
		git:      gitService,
	}
}

func (e *EngineServer) RegisterRouters(mux *http.ServeMux) {
	mux.HandleFunc("/version", e.versionHandler)
	mux.HandleFunc("/hello", e.helloHandler)
	mux.HandleFunc("/event", e.eventHandler) // git webhooks

}
