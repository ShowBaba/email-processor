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
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	queue := NewQueue(config.QueueSize)
	sender := DefaultEmailSender{}

	service := NewEmailService(config, sender, queue)

	if err := service.Start(); err != nil {
		panic("could not start email service: " + err.Error())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/send-email", service.SendEmailHandler)
	mux.HandleFunc("/dlq", service.DLQHandler)
	mux.HandleFunc("/health", service.HealthHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  config.HttpTimeout,
		WriteTimeout: config.HttpTimeout,
	}

	go func() {
		log.Printf("starting server on port %d", config.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("received shutdown signal")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	service.Stop()

	log.Println("server shutdown")
}
