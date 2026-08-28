package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"production43/internal/api"
	"production43/internal/config"
	"production43/internal/service"
	"production43/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st, nil)
	handler := api.CORS(api.Recover(api.RequestLogger(api.NewHandler(svc, cfg.MaxBodyBytes).Routes(), log.Default())))
	server := &http.Server{Addr: cfg.Address, Handler: handler}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
