package main

import (
	"cinlok-be/config"
	httpHandler "cinlok-be/internal/delivery/http"
	"cinlok-be/internal/delivery/http/middleware"
	"cinlok-be/internal/infrastructure/database"
	infraRepo "cinlok-be/internal/infrastructure/repository"
	"cinlok-be/internal/usecase"
	"cinlok-be/pkg/logger"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	// Init logger
	var zapLogger *zap.Logger
	env := os.Getenv("GIN_MODE")
	if env == "release" || env == "production" {
		zapLogger, err = logger.InitProductionLogger()
	} else {
		zapLogger, err = logger.InitDevelopmentLogger()
	}
	if err != nil {
		log.Fatal("Failed to initialize logger", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting platform",
		zap.String("environment", env),
		zap.String("server port", cfg.Server.Port),
	)

	// Init database
	db, err := database.NewPostgresDb(cfg.Database.ConnectionString())
	if err != nil {
		zapLogger.Fatal("Failed to connect to the database", zap.Error(err))
	}
	defer db.Close()

	zapLogger.Info("Database connected successfully",
		zap.String("host", cfg.Database.Host),
		zap.String("database", cfg.Database.DbName),
	)

	migrator := database.NewMigrator(db.DB, zapLogger)
	if err := migrator.RunMigrations("migrations"); err != nil {
		zapLogger.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Init repositories
	refreshTokenRepo := infraRepo.NewRefreshTokenRepository(db.DB)
	userRepo := infraRepo.NewUserRepository(db.DB)

	// Init usecase
	authUC := usecase.NewAuthUseCase(userRepo, refreshTokenRepo, []byte(cfg.JWT.Secret), zapLogger)

	// Init handlers
	authHandler := httpHandler.NewAuthHandler(authUC)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(middleware.ZapLogger(zapLogger))
	router.Use(middleware.ZapRecovery(zapLogger, true))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "OK",
			"service": "cinlok-be",
		})
	})

	v1 := router.Group("/api")
	{
		// Public
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh-token", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}
	}

	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(authUC))
	{
		protected.POST("auth/logout-all", authHandler.LogoutAllDevices)
	}

	// Start server
	serverAddr := ":" + cfg.Server.Port
	zapLogger.Info("Server starting",
		zap.String("address", serverAddr),
		zap.String("environment", env),
	)
	if err := router.Run(serverAddr); err != nil {
		zapLogger.Fatal("Failed to start server", zap.Error(err))
	}
}
