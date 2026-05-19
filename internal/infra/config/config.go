package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App         AppConfig         `mapstructure:"app" yaml:"app"`
	Env         string            `mapstructure:"environment" yaml:"environment"`
	HTTP        HTTPConfig        `mapstructure:"http" yaml:"http"`
	Database    DatabaseConfig    `mapstructure:"database" yaml:"database"`
	EventBus    EventBusConfig    `mapstructure:"event_bus" yaml:"event_bus"`
	ObjectStore ObjectStoreConfig `mapstructure:"object_store" yaml:"object_store"`
	Log         LogConfig         `mapstructure:"log" yaml:"log"`
	Telemetry   TelemetryConfig   `mapstructure:"telemetry" yaml:"telemetry"`
	Auth        AuthConfig        `mapstructure:"auth" yaml:"auth"`
	Runner      RunnerConfig      `mapstructure:"runner" yaml:"runner"`
}

type AppConfig struct {
	Name string `mapstructure:"name" yaml:"name"`
}

type HTTPConfig struct {
	BindAddress string `mapstructure:"bind_address" yaml:"bind_address"`
}

type DatabaseConfig struct {
	URL          string `mapstructure:"url" yaml:"url"`
	RuntimeStore string `mapstructure:"runtime_store" yaml:"runtime_store"`
}

type EventBusConfig struct {
	Type string `mapstructure:"type" yaml:"type"`
}

type ObjectStoreConfig struct {
	Type string `mapstructure:"type" yaml:"type"`
	Path string `mapstructure:"path" yaml:"path"`
}

type LogConfig struct {
	Level string `mapstructure:"level" yaml:"level"`
}

type TelemetryConfig struct {
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled"`
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint"`
}

type AuthConfig struct {
	Enabled        bool       `mapstructure:"enabled" yaml:"enabled"`
	Mode           string     `mapstructure:"mode" yaml:"mode"`
	DevUser        string     `mapstructure:"dev_user" yaml:"dev_user"`
	StaticTokenEnv string     `mapstructure:"static_token_env" yaml:"static_token_env"`
	Issuer         string     `mapstructure:"issuer" yaml:"issuer"`
	OIDC           OIDCConfig `mapstructure:"oidc" yaml:"oidc"`
}

type OIDCConfig struct {
	Issuer        string   `mapstructure:"issuer" yaml:"issuer"`
	ClientID      string   `mapstructure:"client_id" yaml:"client_id"`
	JWKSURL       string   `mapstructure:"jwks_url" yaml:"jwks_url"`
	Scopes        []string `mapstructure:"scopes" yaml:"scopes"`
	GroupsClaim   string   `mapstructure:"groups_claim" yaml:"groups_claim"`
	UsernameClaim string   `mapstructure:"username_claim" yaml:"username_claim"`
}

type RunnerConfig struct {
	Name              string `mapstructure:"name" yaml:"name"`
	Group             string `mapstructure:"group" yaml:"group"`
	HeartbeatInterval string `mapstructure:"heartbeat_interval" yaml:"heartbeat_interval"`
}

func Load(path string) (Config, error) {
	cfg := Default()

	if path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config %q: %w", path, err)
		}
		if err := yaml.Unmarshal(body, &cfg); err != nil {
			return Config{}, fmt.Errorf("decode config %q: %w", path, err)
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		App: AppConfig{
			Name: "nivora",
		},
		Env: "local",
		HTTP: HTTPConfig{
			BindAddress: ":8080",
		},
		Database: DatabaseConfig{
			URL:          "postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable",
			RuntimeStore: "memory",
		},
		EventBus: EventBusConfig{
			Type: "memory",
		},
		ObjectStore: ObjectStoreConfig{
			Type: "local",
			Path: ".nivora/objectstore",
		},
		Log: LogConfig{
			Level: "info",
		},
		Telemetry: TelemetryConfig{
			Enabled: false,
		},
		Auth: AuthConfig{
			Enabled:        false,
			Mode:           "dev",
			DevUser:        "local-admin",
			StaticTokenEnv: "NIVORA_AUTH_TOKEN",
		},
		Runner: RunnerConfig{
			Name:              "local-runner",
			Group:             "default",
			HeartbeatInterval: "30s",
		},
	}
}

func (c Config) Validate() error {
	if c.App.Name == "" {
		return errors.New("config app.name is required")
	}
	if c.Env == "" {
		return errors.New("config environment is required")
	}
	if c.HTTP.BindAddress == "" {
		return errors.New("config http.bind_address is required")
	}
	if c.Database.URL == "" {
		return errors.New("config database.url is required")
	}
	if c.Database.RuntimeStore == "" {
		return errors.New("config database.runtime_store is required")
	}
	if c.Database.RuntimeStore != "memory" && c.Database.RuntimeStore != "postgres" {
		return errors.New("config database.runtime_store must be memory or postgres")
	}
	if c.EventBus.Type == "" {
		return errors.New("config event_bus.type is required")
	}
	if c.ObjectStore.Type == "" {
		return errors.New("config object_store.type is required")
	}
	if c.Log.Level == "" {
		return errors.New("config log.level is required")
	}
	if c.Auth.Enabled && c.Auth.Mode == "oidc" {
		if c.Auth.OIDC.Issuer == "" && c.Auth.Issuer == "" {
			return errors.New("config auth.oidc.issuer is required when auth.mode is oidc")
		}
		if c.Auth.OIDC.ClientID == "" {
			return errors.New("config auth.oidc.client_id is required when auth.mode is oidc")
		}
	}
	return nil
}
