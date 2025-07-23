package server

import (
	"fmt"
	"net/http"
	"time"
	"topi/internal/shared/database"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port int

	db database.Service
}
type createHandlers func(*http.ServeMux)

func NewServer(port int, cb createHandlers) *http.Server {

	mux := http.NewServeMux()
	cb(mux) // adds custom handlers
	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      CorsMiddleware(LoggingMiddleware(mux)),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
