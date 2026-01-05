package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config contiene toda la configuración de la aplicación
type Config struct {
	// Server configuration
	ServerPort  string
	ServerHost  string
	Environment string

	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// Executor configuration
	ExecutorTimeout       time.Duration
	ExecutorMemoryLimit   string // e.g., "256m"
	ExecutorCPULimit      string // e.g., "0.5" (50% of one CPU)
	ExecutorMaxConcurrent int

	// Docker configuration
	DockerHost string
	DockerAPI  string
}

var AppConfig *Config

// Load carga la configuración desde variables de entorno
func Load() *Config {
	// Cargar .env si existe (útil para desarrollo)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config := &Config{
		// Server
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		ServerHost:  getEnv("SERVER_HOST", "0.0.0.0"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "rojudger"),
		DBPassword: getEnv("DB_PASSWORD", "rojudger"),
		DBName:     getEnv("DB_NAME", "rojudger"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// Executor
		ExecutorTimeout:       getEnvAsDuration("EXECUTOR_TIMEOUT", 10*time.Second),
		ExecutorMemoryLimit:   getEnv("EXECUTOR_MEMORY_LIMIT", "256m"),
		ExecutorCPULimit:      getEnv("EXECUTOR_CPU_LIMIT", "0.5"),
		ExecutorMaxConcurrent: getEnvAsInt("EXECUTOR_MAX_CONCURRENT", 5),

		// Docker
		DockerHost: getEnv("DOCKER_HOST", "unix:///var/run/docker.sock"),
		DockerAPI:  getEnv("DOCKER_API_VERSION", "1.42"),
	}

	AppConfig = config
	return config
}

// GetDatabaseDSN retorna el connection string para PostgreSQL
func (c *Config) GetDatabaseDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

// GetRedisAddr retorna la dirección de Redis
func (c *Config) GetRedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// Helper functions para leer variables de entorno

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid duration value for %s, using default: %v", key, defaultValue)
		return defaultValue
	}
	return value
}

// IsDevelopment verifica si estamos en modo desarrollo
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction verifica si estamos en modo producción
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
