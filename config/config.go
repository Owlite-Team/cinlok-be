package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Server   ServerConfig
}

type DatabaseConfig struct {
	Host       string
	Port       string
	DbUser     string
	DbPassword string
	DbName     string
	SSLMode    string
}

type JWTConfig struct {
	Secret string
}

type ServerConfig struct {
	Host string
	Port string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variable")
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:       getEnv("DB_HOST", "localhost"),
			Port:       getEnv("DB_PORT", "5432"),
			DbUser:     getEnv("DB_USER", "postgres"),
			DbPassword: getEnv("DB_PASSWORD", "postgres"),
			DbName:     getEnv("DB_NAME", "cinlokdb"),
			SSLMode:    getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
	}

	return config, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.DbUser, c.DbPassword, c.DbName, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}
