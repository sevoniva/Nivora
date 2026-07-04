package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App         AppConfig         `mapstructure:"app" yaml:"app"`
	Env         string            `mapstructure:"environment" yaml:"environment"`
	HTTP        HTTPConfig        `mapstructure:"http" yaml:"http"`
	MCP         MCPConfig         `mapstructure:"mcp" yaml:"mcp"`
	Database    DatabaseConfig    `mapstructure:"database" yaml:"database"`
	EventBus    EventBusConfig    `mapstructure:"event_bus" yaml:"event_bus"`
	ObjectStore ObjectStoreConfig `mapstructure:"object_store" yaml:"object_store"`
	Log         LogConfig         `mapstructure:"log" yaml:"log"`
	Telemetry   TelemetryConfig   `mapstructure:"telemetry" yaml:"telemetry"`
	Auth        AuthConfig        `mapstructure:"auth" yaml:"auth"`
	Runner      RunnerConfig      `mapstructure:"runner" yaml:"runner"`
	Runtime     RuntimeConfig     `mapstructure:"runtime" yaml:"runtime"`
}

type AppConfig struct {
	Name string `mapstructure:"name" yaml:"name"`
}

type HTTPConfig struct {
	BindAddress string `mapstructure:"bind_address" yaml:"bind_address"`
}

type MCPConfig struct {
	Enabled              bool   `mapstructure:"enabled" yaml:"enabled"`
	Mode                 string `mapstructure:"mode" yaml:"mode"`
	ReadOnly             bool   `mapstructure:"readonly" yaml:"readonly"`
	AllowPlanTools       bool   `mapstructure:"allow_plan_tools" yaml:"allow_plan_tools"`
	AllowActionTools     bool   `mapstructure:"allow_action_tools" yaml:"allow_action_tools"`
	SubjectID            string `mapstructure:"subject_id" yaml:"subject_id"`
	SubjectRole          string `mapstructure:"subject_role" yaml:"subject_role"`
	TokenEnv             string `mapstructure:"token_env" yaml:"token_env"`
	RequestTimeout       string `mapstructure:"request_timeout" yaml:"request_timeout"`
	MaxRequestBytes      int    `mapstructure:"max_request_bytes" yaml:"max_request_bytes"`
	MaxResponseBytes     int    `mapstructure:"max_response_bytes" yaml:"max_response_bytes"`
	MaxRequestsPerMinute int    `mapstructure:"max_requests_per_minute" yaml:"max_requests_per_minute"`
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

// RunnerIsolationProfile defines the execution isolation level for runners.
// Shell executor is not an OS-level sandbox regardless of profile.
const (
	RunnerProfileLocalDev      = "local-dev"
	RunnerProfileShellHardened = "shell-hardened"
	RunnerProfileContainer     = "container-isolated"
	RunnerProfileKubernetesJob = "kubernetes-job"
	RunnerProfileExternal      = "external-runner"
)

type RuntimeConfig struct {
	AllowLocalShellExecutor bool   `mapstructure:"allow_local_shell_executor" yaml:"allow_local_shell_executor"`
	AllowPrivilegedExecutor bool   `mapstructure:"allow_privileged_executor" yaml:"allow_privileged_executor"`
	AllowRemoteHostDeploy   bool   `mapstructure:"allow_remote_host_deploy" yaml:"allow_remote_host_deploy"`
	AllowKubernetesApply    bool   `mapstructure:"allow_kubernetes_apply" yaml:"allow_kubernetes_apply"`
	AllowArgoSync           bool   `mapstructure:"allow_argo_sync" yaml:"allow_argo_sync"`
	AllowInsecureRegistry   bool   `mapstructure:"allow_insecure_registry" yaml:"allow_insecure_registry"`
	RunnerIsolationProfile  string `mapstructure:"runner_isolation_profile" yaml:"runner_isolation_profile"`
	AllowDockerSocketMount  bool   `mapstructure:"allow_docker_socket_mount" yaml:"allow_docker_socket_mount"`
	AllowHostPathMount      bool   `mapstructure:"allow_host_path_mount" yaml:"allow_host_path_mount"`
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
		MCP: MCPConfig{
			Enabled:              false,
			Mode:                 "stdio",
			ReadOnly:             true,
			AllowPlanTools:       true,
			SubjectRole:          "viewer",
			TokenEnv:             "NIVORA_MCP_TOKEN",
			RequestTimeout:       "15s",
			MaxRequestBytes:      1024 * 1024,
			MaxResponseBytes:     256 * 1024,
			MaxRequestsPerMinute: 120,
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
		Runtime: RuntimeConfig{
			AllowLocalShellExecutor: true,
			RunnerIsolationProfile:  RunnerProfileLocalDev,
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
	if c.MCP.Mode == "" {
		c.MCP.Mode = "stdio"
	}
	if c.MCP.Mode != "stdio" && c.MCP.Mode != "http" {
		return errors.New("config mcp.mode must be stdio or http")
	}
	if c.MCP.RequestTimeout != "" {
		if timeout, err := time.ParseDuration(c.MCP.RequestTimeout); err != nil || timeout <= 0 {
			return errors.New("config mcp.request_timeout must be a positive duration")
		}
	}
	if c.MCP.MaxResponseBytes < 0 {
		return errors.New("config mcp.max_response_bytes must be zero or greater")
	}
	if c.MCP.MaxRequestBytes < 0 {
		return errors.New("config mcp.max_request_bytes must be zero or greater")
	}
	if c.MCP.MaxRequestsPerMinute < 0 {
		return errors.New("config mcp.max_requests_per_minute must be zero or greater")
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
	if (c.Env == "production" || c.Env == "prod") && c.Database.RuntimeStore == "memory" {
		return errors.New("config database.runtime_store=memory is dev-only; use postgres for production")
	}
	if c.Env == "production" || c.Env == "prod" {
		if !c.Auth.Enabled {
			return errors.New("config auth.enabled=false is not allowed in production")
		}
		if c.Auth.Mode == "" || c.Auth.Mode == "dev" || c.Auth.Mode == "disabled" {
			return errors.New("config auth.mode must not be dev or disabled in production")
		}
		if c.Auth.Mode == "token" && c.Auth.StaticTokenEnv == "" {
			return errors.New("config auth.static_token_env is required when auth.mode=token in production")
		}
		if c.MCP.Enabled {
			if !c.MCP.ReadOnly {
				return errors.New("config mcp.readonly=false is not allowed in this foundation phase")
			}
			if c.MCP.AllowActionTools {
				return errors.New("config mcp.allow_action_tools=true is not allowed in this foundation phase")
			}
			if c.MCP.Mode == "stdio" && c.MCP.TokenEnv == "" {
				return errors.New("config mcp.token_env is required when mcp.enabled=true in production")
			}
			if c.MCP.RequestTimeout == "" {
				return errors.New("config mcp.request_timeout is required when mcp.enabled=true in production")
			}
			if c.MCP.MaxResponseBytes <= 0 {
				return errors.New("config mcp.max_response_bytes must be positive when mcp.enabled=true in production")
			}
			if c.MCP.MaxRequestBytes <= 0 {
				return errors.New("config mcp.max_request_bytes must be positive when mcp.enabled=true in production")
			}
			if c.MCP.MaxRequestsPerMinute <= 0 {
				return errors.New("config mcp.max_requests_per_minute must be positive when mcp.enabled=true in production")
			}
		}
		if hasInlineDatabasePassword(c.Database.URL) {
			return errors.New("config database.url must not include an inline password in production; inject credentials through a secret or environment-specific config")
		}
		if c.Runtime.AllowLocalShellExecutor {
			return errors.New("config runtime.allow_local_shell_executor=true is not allowed in production")
		}
		if c.Runtime.AllowPrivilegedExecutor {
			return errors.New("config runtime.allow_privileged_executor=true is not allowed in production")
		}
		if c.Runtime.AllowRemoteHostDeploy {
			return errors.New("config runtime.allow_remote_host_deploy=true is not allowed in production")
		}
		if c.Runtime.AllowKubernetesApply {
			return errors.New("config runtime.allow_kubernetes_apply=true is not allowed in production")
		}
		if c.Runtime.AllowArgoSync {
			return errors.New("config runtime.allow_argo_sync=true is not allowed in production")
		}
		if c.Runtime.AllowInsecureRegistry {
			return errors.New("config runtime.allow_insecure_registry=true is not allowed in production")
		}
		profile := c.Runtime.RunnerIsolationProfile
		if profile == "" {
			profile = RunnerProfileLocalDev
		}
		switch profile {
		case RunnerProfileLocalDev:
			return errors.New("config runtime.runner_isolation_profile=local-dev is not allowed in production; use container-isolated, kubernetes-job, or external-runner")
		case RunnerProfileShellHardened:
			if !c.Runtime.AllowLocalShellExecutor {
				return errors.New("config runtime.runner_isolation_profile=shell-hardened requires runtime.allow_local_shell_executor=true (must be explicitly enabled)")
			}
		case RunnerProfileContainer, RunnerProfileKubernetesJob, RunnerProfileExternal:
		default:
			return errors.New("config runtime.runner_isolation_profile must be one of: shell-hardened, container-isolated, kubernetes-job, external-runner")
		}
		if c.Runtime.AllowDockerSocketMount {
			return errors.New("config runtime.allow_docker_socket_mount=true is not allowed in production")
		}
		if c.Runtime.AllowHostPathMount {
			return errors.New("config runtime.allow_host_path_mount=true is not allowed in production")
		}
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

func hasInlineDatabasePassword(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return false
	}
	_, ok := parsed.User.Password()
	return ok
}
