package main

import (
	"github.com/techagentng/telair-erp/config"
	"github.com/techagentng/telair-erp/db"
	"github.com/techagentng/telair-erp/mailingservices"
	"github.com/techagentng/telair-erp/server"
	"github.com/techagentng/telair-erp/services"
	"log"
	_ "net/url"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// gormDB := db.GetDB(conf)

	s := &server.Server{
		Config:                   conf,
		DB:                       db.GormDB{},
	}

	s.Start()
}