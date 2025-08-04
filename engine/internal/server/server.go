package server

import (
	"context"
	"github.com/tobiaskrok/topi/shared/database"
	"github.com/tobiaskrok/topi/shared/git"
	"github.com/tobiaskrok/topi/shared/rabbitmq"
	"net/http"
	"os"
)

type EngineServer struct {
	db       database.Database
	rabbitmq *rabbitmq.RabbitMQ
	git      git.GitService
}

func NewEngineServer(ctx context.Context) *EngineServer {
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

	q, err := rmq.Channel.QueueDeclare("engine", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = rmq.Channel.QueueBind(q.Name, "engine.*", "topi", false, nil)
	if err != nil {
		panic(err)
	}
	db, err := database.Open(ctx, database.Config{
		Database: os.Getenv("DB_DATABASE"),
		Password: os.Getenv("DB_PASSWORD"),
		Username: os.Getenv("DB_USERNAME"),
		Port:     os.Getenv("DB_PORT"),
		Host:     os.Getenv("DB_HOST"),
	})
	if err != nil {
		panic(err)
	}
	return &EngineServer{
		db:       database.WithResilience(db, database.ResilientDatabaseSettings{BreakerThreshold: 10, InitialBackoff: 1000, BreakerTimeout: 10000}),
		rabbitmq: rmq,
		git:      gitService,
	}
}

func (e *EngineServer) RegisterRouters(mux *http.ServeMux) {
	mux.HandleFunc("/version", e.versionHandler)
	mux.HandleFunc("/hello", e.helloHandler)
	mux.HandleFunc("/event", e.eventHandler) // git webhooks

}
