package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func readConfig() (config, error) {
	v := viper.New()
	v.SetConfigType("json")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	envID := strings.TrimSpace(os.Getenv("ENV_ID"))
	if envID == "" {
		envID = "default"
	}

	configFileName := "config." + envID
	if os.Getenv("NATIVE") == "1" {
		configFileName = "config.native"
	}
	v.SetConfigName(configFileName)

	setConfigDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	cfg := config{
		Port:                             v.GetInt("http_port"),
		SchemaRegistryHost:               v.GetString("schema_registry_url"),
		SelfURL:                          v.GetString("self_url"),
		RedisHost:                        v.GetString("redis_host"),
		RedisPort:                        v.GetInt("redis_port"),
		RedisPassword:                    v.GetString("redis_password"),
		RedisDB:                          v.GetInt("redis_db"),
		HistoricalWeatherCacheTTLSeconds: v.GetInt("historical_weather_cache_ttl_seconds"),
		EnvironmentID:                    envID,
		LogLevel:                         v.GetString("log_level"),
	}

	overrideConfigFromEnv(&cfg)
	return cfg, nil
}

func setConfigDefaults(v *viper.Viper) {
	v.SetDefault("http_port", 8070)
	v.SetDefault("schema_registry_url", "http://gql-schema-registry:3000")
	v.SetDefault("self_url", "weather:8070")
	v.SetDefault("redis_host", "")
	v.SetDefault("redis_port", 6379)
	v.SetDefault("redis_password", "")
	v.SetDefault("redis_db", 0)
	v.SetDefault("historical_weather_cache_ttl_seconds", 1800)
	v.SetDefault("log_level", "")
}

func overrideConfigFromEnv(cfg *config) {
	if value := os.Getenv("PORT"); value != "" {
		cfg.Port = getenvInt("PORT", cfg.Port)
	}
	if value := os.Getenv("SCHEMA_REGISTRY_HOST"); value != "" {
		cfg.SchemaRegistryHost = value
	}
	if value := os.Getenv("SELF_URL"); value != "" {
		cfg.SelfURL = value
	}
	if value := os.Getenv("REDIS_HOST"); value != "" {
		cfg.RedisHost = value
	}
	if value := os.Getenv("REDIS_PORT"); value != "" {
		cfg.RedisPort = getenvInt("REDIS_PORT", cfg.RedisPort)
	}
	if value := os.Getenv("REDIS_PASSWORD"); value != "" {
		cfg.RedisPassword = value
	}
	if value := os.Getenv("REDIS_DB"); value != "" {
		cfg.RedisDB = getenvInt("REDIS_DB", cfg.RedisDB)
	}
	if value := os.Getenv("HISTORICAL_WEATHER_CACHE_TTL_SECONDS"); value != "" {
		cfg.HistoricalWeatherCacheTTLSeconds = getenvInt("HISTORICAL_WEATHER_CACHE_TTL_SECONDS", cfg.HistoricalWeatherCacheTTLSeconds)
	}
	if value := os.Getenv("LOG_LEVEL"); value != "" {
		cfg.LogLevel = value
	}
}
