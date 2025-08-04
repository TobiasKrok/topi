package server

import (
	"fmt"
	"net/http"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port int
}
type createHandlers func(*http.ServeMux)

func NewServer(port int, cb createHandlers) *http.Server {

	mux := http.NewServeMux()
	cb(mux) // adds custom handlers
	// TODO ADD CORS
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      LoggingMiddleware(mux),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
