package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/programs/billing/internal/config"
	httpapi "github.com/QuantumNous/new-api/programs/billing/internal/http"
	"github.com/QuantumNous/new-api/programs/billing/internal/store"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("open postgres connection: %v", err)
	}

	billingStore := store.Store{DB: db}
	if err := billingStore.ValidateSchema(); err != nil {
		log.Fatal(err)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	handler := httpapi.Handler{
		Store:           billingStore,
		DefaultPageSize: cfg.DefaultPageSize,
		MaxPageSize:     cfg.MaxPageSize,
	}
	handler.RegisterRoutes(router)

	server := &http.Server{
		Addr:              cfg.Host + ":" + itoa(cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
	}

	log.Printf("billing service listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
