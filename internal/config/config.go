package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Repos        string      `mapstructure:"repos"`
	Workspaces   string      `mapstructure:"workspaces"`
	Env          []EnvEntry  `mapstructure:"env"`
	Remotes      []Remote    `mapstructure:"remotes"`
	Cache        CacheConfig `mapstructure:"cache"`
	CopyPatterns []string    `mapstructure:"copy_patterns"`
}

// EnvEntry represents a single environment variable entry
type EnvEntry struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

// Remote represents a remote Git server configuration
type Remote struct {
	Type string `mapstructure:"type"`
	Host string `mapstructure:"host"` // Hostname (e.g., "github.com", "my.gitserver.com")
}

// CacheConfig represents cache settings
type CacheConfig struct {
	Enabled bool `mapstructure:"enabled"`
	TTL     int  `mapstructure:"ttl"` // TTL in seconds
}

// GetConfig は現在の設定を返す
func GetConfig() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// チルダを展開
	cfg.Repos = expandPath(cfg.Repos)
	cfg.Workspaces = expandPath(cfg.Workspaces)

	return &cfg, nil
}

// GetReposDir はreposディレクトリのパスを返す
func GetReposDir() string {
	path := viper.GetString("repos")
	return expandPath(path)
}

// GetWorkspacesDir はworkspacesディレクトリのパスを返す
func GetWorkspacesDir() string {
	path := viper.GetString("workspaces")
	return expandPath(path)
}

// GetRemotes はリモート設定のスライスを返す
func GetRemotes() ([]Remote, error) {
	var remotes []Remote
	if err := viper.UnmarshalKey("remotes", &remotes); err != nil {
		return nil, err
	}
	return remotes, nil
}

// expandPath はチルダを含むパスを展開する
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// LoadEnv は設定された環境変数をプロセスに読み込む
func LoadEnv() error {
	cfg, err := GetConfig()
	if err != nil {
		return err
	}
	for _, entry := range cfg.Env {
		if err := os.Setenv(entry.Key, entry.Value); err != nil {
			return err
		}
	}
	return nil
}

// GetCopyPatterns は設定されたコピーパターンのリストを返す
// 設定されていない場合はデフォルトパターンを返す
func GetCopyPatterns() []string {
	patterns := viper.GetStringSlice("copy_patterns")
	if len(patterns) == 0 {
		// デフォルトパターン: 環境設定ファイルやバージョン管理ファイル
		return []string{
			".env*",

			// Editor settings
			".vscode/*",

			// Version management files
			".tool-versions",
			".*-version",

			// Node.js
			".nvmrc",
			".npmrc",

			// Golang
			"go.work",
		}
	}
	return patterns
}
