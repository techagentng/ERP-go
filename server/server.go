package server

import (
	"context"
	"fmt"
	"github.com/techagentng/telair-erp/config"
	"github.com/techagentng/telair-erp/db"
	"github.com/techagentng/citizenx/mailingservices"
	"github.com/techagentng/telair-erp/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	Config                   *config.Config
	DB                       db.GormDB
}

// Server serves requests to DB with rout
func (s *Server) Start() {
	r := s.setupRouter()
	// TODO: user config.PORT here
	PORT := fmt.Sprintf(":%s", os.Getenv("PORT"))
	if PORT == ":" {
		PORT = ":8080"
	}
	srv := &http.Server{
		Addr:    PORT,
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("Server started on %s\n", PORT)
	gracefulShutdown(srv)
}