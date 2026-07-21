package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/config"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/handler"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/routes"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/service"
	pkgmiddleware "github.com/laconic-smart-solution/communication-service/pkg/middleware"
)

func main() {
	cfg := config.Load()
	log := newLogger(cfg.LogLevel)

	if cfg.GatewaySharedSecret == "" {
		log.Fatal("GATEWAY_SECRET is required")
	}
	if cfg.ShopeePartnerID == "" || cfg.ShopeePartnerKey == "" {
		log.Fatal("SHOPEE_PARTNER_ID and SHOPEE_PARTNER_KEY are required")
	}
	if cfg.ShopeeRedirectURI == "" {
		log.Fatal("SHOPEE_REDIRECT_URI is required")
	}

	shopeeClient := client.NewShopeeClient(cfg, 15)
	storeSvc := service.NewStoreService(log, shopeeClient)
	orderSvc := service.NewOrderService(log)
	productSvc := service.NewProductService(log)

	handlers := routes.Handlers{
		Store:   handler.NewStoreHandler(storeSvc),
		Order:   handler.NewOrderHandler(orderSvc),
		Product: handler.NewProductHandler(productSvc),
	}

	app := fiber.New(fiber.Config{
		AppName:      "shopee-service",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Actor-ID, X-User-ID, X-Gateway-Secret",
		AllowMethods: strings.Join([]string{
			fiber.MethodGet, fiber.MethodPost, fiber.MethodPut,
			fiber.MethodPatch, fiber.MethodDelete, fiber.MethodOptions,
		}, ","),
	}))
	app.Use(pkgmiddleware.RequireGatewaySecret(cfg.GatewaySharedSecret))
	routes.Register(app, handlers)

	addr := ":" + cfg.Port
	go func() {
		log.Infof("shopee-service listening on %s", addr)
		if err := app.Listen(addr); err != nil && !errors.Is(err, fiber.ErrServiceUnavailable) {
			log.WithError(err).Fatal("server stopped")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.WithError(err).Error("graceful shutdown failed")
	}
	log.Info("bye")
}

func newLogger(level string) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	l.SetLevel(lvl)
	return l
}
