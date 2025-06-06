package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Database struct {
	PsqlUrl string `env:"PSQL_URL" envDefault:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"`
}

type Line struct {
	ChannelSecret      string `env:"LINE_CHANNEL_SECRET" envDefault:"SECRET_KEY"`
	ChannelAccessToken string `env:"LINE_CHANNEL_ACCESS_TOKEN" envDefault:"ACCESS_TOKEN"`
}

type Trace struct {
	Endpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://localhost:4317"`
}

type Config struct {
	Db          Database
	Line        Line
	Trace       Trace
	Environment string `env:"ENVIRONMENT" envDefault:"DEVELOPMENT"`
	Port        string `env:"PORT" envDefault:"8080"`
}

var cfg Config

func Get() Config {
	return cfg
}

// Init initializes the configuration by parsing environment variables
func Init() (*Config, error) {
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
