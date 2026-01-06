package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/handlers"
	"github.com/Grivvus/ym/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		slog.Error("Can't load .env file", "err", err)
		os.Exit(1)
	}

	appHost, ok := os.LookupEnv("APPLICATION_HOST")
	if !ok {
		panic("can't lookup variable APPLICATION_HOST")
	}
	appPort, ok := os.LookupEnv("APPLICATION_PORT")
	if !ok {
		panic("can't lookup variable APPLICATION_PORT")
	}

	pool, err := pgxpool.New(context.TODO(), formDBConnString())
	if err != nil {
		slog.Error("Can't create connection pool to database", "err", err)
		os.Exit(1)
	}
	dbInst := db.New(pool)
	authService := service.NewAuthService(dbInst)
	userService := service.NewUserService(dbInst)
	var server api.ServerInterface = handlers.NewRootHandler(authService, userService)

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
		Addr:    fmt.Sprintf("%v:%v", appHost, appPort),
		Handler: h,
	}

	slog.Info("starting server on", "port", appPort)

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

func formDBConnString() string {
	pgHost, ok := os.LookupEnv("POSTGRES_HOST")
	if !ok {
		panic("can't lookup variable POSTGRES_HOST")
	}
	pgPort, ok := os.LookupEnv("POSTGRES_PORT")
	if !ok {
		panic("can't lookup variable POSTGRES_PORT")
	}
	pgUser, ok := os.LookupEnv("POSTGRES_USER")
	if !ok {
		panic("can't lookup variable POSTGRES_USER")
	}
	pgPassword, ok := os.LookupEnv("POSTGRES_PASSWORD")
	if !ok {
		panic("can't lookup variable POSTGRES_PASSWORD")
	}
	pgDBName, ok := os.LookupEnv("POSTGRES_DB")
	if !ok {
		panic("can't lookup variable POSTGRES_DB")
	}
	return fmt.Sprintf("postgres://%v:%v@%v:%v/%v", pgUser, pgPassword, pgHost, pgPort, pgDBName)
}
