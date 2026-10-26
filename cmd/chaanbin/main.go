package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhiraj-ku/ChaanBin/internal/api"
	"github.com/abhiraj-ku/ChaanBin/internal/db"
	"github.com/abhiraj-ku/ChaanBin/internal/store"
)

func main() {

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		panic("missing db url abort....")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	fmt.Println("init db conn")
	db, err := db.NewPostgresPool(ctx, dbUrl)
	if err != nil {
		log.Fatalf("failed to connecgt to db: %v", err)
	}
	defer db.Close()

	repo := store.NewLogRepository(db)
	handler := &api.Handler{Repo: repo}

	// routiung

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/search", handler.SearchLogs)

	// html serce file endpoint
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fileServer)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// start the serve in go routine '
	go func() {
		log.Printf("ChaanBin api server listening on http://localhost: %s\n", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server crashed: %v", err)
		}
	}()

	// gracefil shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("server forced closed: %v", err)
	}

	log.Println("server exited cleanly")
}
