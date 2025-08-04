package main

import (
	"context"
	"errors"
	"fmt"
	engine "github.com/tobiaskrok/topi/engine/internal/server"
	"github.com/tobiaskrok/topi/shared/server"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

//TODO graceful shutdown of rabbitmq, gitea and more!!!

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func main() {

	port, _ := strconv.Atoi(os.Getenv("TOPI_ENGINE_PORT"))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Println("Starting server on port: ", port)
	e := engine.NewEngineServer(ctx)
	s := server.NewServer(port, func(mux *http.ServeMux) {
		e.RegisterRouters(mux)
	})
	// Create a done channel to signal when the shutdown is complete
	go func() {
		<-ctx.Done() // block until signal
		log.Println("shutting down…")

		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.Shutdown(shCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}
	}()
	log.Printf("listening on :%d", port)
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}

	log.Println("graceful shutdown complete")
}
