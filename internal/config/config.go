package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// Config описывает основные параметры агента.
type Config struct {
	Agent struct {
		Mode     string `yaml:"mode"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"agent"`
	Security struct {
		ExecAllowlist []string            `yaml:"exec_allowlist"`
		AuthAllowlist map[string][]string `yaml:"auth_allowlist"`
	} `yaml:"security"`
	SQLite struct {
		Path          string `yaml:"path"`
		RetentionDays int    `yaml:"retention_days"`
	} `yaml:"sqlite"`
	Scheduler struct {
		IntervalSeconds int `yaml:"interval_seconds"`
	} `yaml:"scheduler"`
	Web struct {
		Enabled          bool   `yaml:"enabled"`
		ListenAddr       string `yaml:"listen_addr"`
		ReadTimeoutMS    int    `yaml:"read_timeout_ms"`
		WriteTimeoutMS   int    `yaml:"write_timeout_ms"`
		RequestTimeoutMS int    `yaml:"request_timeout_ms"`
		ShutdownTimeoutS int    `yaml:"shutdown_timeout_s"`
		MaxBodyBytes     int64  `yaml:"max_body_bytes"`
	} `yaml:"web"`
	LLM struct {
		Enabled       bool     `yaml:"enabled"`
		ProviderOrder []string `yaml:"provider_order"`
		TimeoutMS     int      `yaml:"timeout_ms"`
	} `yaml:"llm"`
}

// Default возвращает конфигурацию по умолчанию.
func Default() Config {
	var cfg Config
	cfg.Agent.Mode = "cli"
	cfg.Agent.LogLevel = "info"
	cfg.SQLite.Path = "/var/lib/goadmin/state.db"
	cfg.SQLite.RetentionDays = 30
	cfg.Scheduler.IntervalSeconds = 60
	cfg.Web.Enabled = false
	cfg.Web.ListenAddr = "127.0.0.1:8080"
	cfg.Web.ReadTimeoutMS = 2000
	cfg.Web.WriteTimeoutMS = 5000
	cfg.Web.RequestTimeoutMS = 3000
	cfg.Web.ShutdownTimeoutS = 5
	cfg.Web.MaxBodyBytes = 1 << 20
	cfg.LLM.ProviderOrder = []string{"local", "cloud"}
	cfg.LLM.TimeoutMS = 2000
	cfg.Security.AuthAllowlist = map[string][]string{"telegram": {}, "maxbot": {}, "web": {}}
	return cfg
}

// Load читает конфиг из файла YAML, поверх значений по умолчанию.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- путь к конфигу задается доверенным оператором/CI.
	if err != nil {
		return cfg, err
	}
	if len(data) == 0 {
		return cfg, errors.New("config file is empty")
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
