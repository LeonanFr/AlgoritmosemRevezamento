package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"Algorithms/internal/executor"
	"Algorithms/internal/executor-server/config"
	"Algorithms/internal/executor-server/handler"
	"Algorithms/internal/executor-server/prewarm"
)

func main() {
	cfg := config.Load()

	if err := executor.InitJVMPool(cfg.JVMPoolSize, cfg.JVMClasspath, cfg.JVMXmx); err != nil {
		log.Fatalf("Falha crítica ao iniciar pool JVM: %v", err)
	}

	if cfg.EnablePreWarm {
		prewarm.LanguagesWarm()
	}

	h := handler.NewHandler(cfg)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Executor server listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
