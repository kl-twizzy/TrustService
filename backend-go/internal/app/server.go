package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	transport "seller-trust-map/backend-go/internal/http"
	"seller-trust-map/backend-go/internal/repository"
	"seller-trust-map/backend-go/internal/service"
)

type Server struct {
	engine *gin.Engine
	port   string
}

func NewServer() (*Server, error) {
	port := envOrDefault("PORT", "8080")

	redisClient := initRedisClient()
	dataRepo, err := initRepository()
	if err != nil {
		return nil, err
	}

	mlClient := service.NewMLClient()
	pageFetcher := service.NewPageFetcher()
	trustService := service.NewTrustService(dataRepo, mlClient, pageFetcher, redisClient)
	handler := transport.NewHandler(trustService)

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"chrome-extension://*", "moz-extension://*", "http://localhost:*", "http://127.0.0.1:*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
	}))
	handler.RegisterRoutes(router)
	if err := registerDashboard(router); err != nil {
		return nil, err
	}

	return &Server{
		engine: router,
		port:   port,
	}, nil
}

func (s *Server) Run() error {
	return s.engine.Run(fmt.Sprintf(":%s", s.port))
}

func initRepository() (repository.Repository, error) {
	fallback := repository.NewDemoRepository()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return repository.NewCombinedRepository(nil, fallback), nil
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return repository.NewCombinedRepository(nil, fallback), nil
	}

	return repository.NewCombinedRepository(repository.NewPostgresRepository(db), fallback), nil
}

func initRedisClient() *redis.Client {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		return nil
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil
	}

	return redisClient
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
