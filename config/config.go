package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Database struct {
	PsqlUrl string `env:"PSQL_URL" envDefault:"postgres://line_accounting_db_15da_user:IKVjUV5gNFX7CQrepHP6cekZwhSFzDJd@dpg-d0vqdkemcj7s73fsjn80-a.oregon-postgres.render.com/line_accounting_db_15da"`
}

type Line struct {
	ChannelSecret      string `env:"LINE_CHANNEL_SECRET" envDefault:"0ab1620bd47876769a74f3a39b973362"`
	ChannelAccessToken string `env:"LINE_CHANNEL_ACCESS_TOKEN" envDefault:"TnjFBPwgMFShYIpFNWrDCGEruQoTtT7t/Hm516P/ordoWuBiiQ8lPGPbRDCp/5L0s/hUMM19M49KWyp+CwWS3O AtGCJyXSBGdR7/Krr88yWILueL9JS7khKYXjBCYR+zQcEv59PxKvYKoTrgO4HaSgdB04t89/1O/w1cDnyilFU="`
}

type Config struct {
	Db   Database
	Line Line
	Port string `env:"PORT" envDefault:"8080"`
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
