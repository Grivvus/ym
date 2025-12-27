package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {

	var server api.ServerInterface = handlers.RootHandler{}

	r := chi.NewMux()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	/* swagger-related routes */
	r.Get("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/api/openapi.yml")
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/openapi.yml"),
	))
	/* swagger-related routes end */

	h := api.HandlerFromMux(server, r)

	s := &http.Server{
		Addr:    ":8000",
		Handler: h,
	}

	slog.Info("starting server on", "port", "8000")

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Println("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
