package config

import "os"

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	FrontendOrigin string
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() Config {
	return Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      getenv("JWT_SECRET", "dev-secret"),
		FrontendOrigin: getenv("FRONTEND_ORIGIN", "http://localhost:5173"),
	}
}
