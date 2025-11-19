package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigValidateCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Strict bool `flag:"strict" short:"s" default:"false" usage:"厳格なバリデーション（未使用キーなども警告）"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  `設定ファイルの妥当性を検証します。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigValidate(opts.Strict)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigValidate(strict bool) error {
	configFile := viper.ConfigFileUsed()

	if configFile == "" {
		return fmt.Errorf("設定ファイルが見つかりません")
	}

	// JSONの構文チェック
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("✗ 設定ファイルの読み込みに失敗しました: %w", err)
	}

	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("✗ JSON の構文エラー: %w", err)
	}

	// 設定を読み込み
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("✗ 設定の読み込みに失敗しました: %w", err)
	}

	errors := []string{}
	warnings := []string{}

	// 必須キーのチェック
	if cfg.Repos == "" {
		errors = append(errors, "repos: required field is missing")
	}
	if cfg.Workspaces == "" {
		errors = append(errors, "workspaces: required field is missing")
	}

	// repos ディレクトリの存在チェック（警告のみ）
	if _, err := os.Stat(cfg.Repos); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("repos: directory does not exist: %s", cfg.Repos))
	}

	// workspaces ディレクトリの存在チェック（警告のみ）
	if _, err := os.Stat(cfg.Workspaces); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("workspaces: directory does not exist: %s", cfg.Workspaces))
	}

	// リモート設定のチェック
	for i, remote := range cfg.Remotes {
		if remote.Type == "" {
			errors = append(errors, fmt.Sprintf("remotes.%d.type: required field is missing", i))
		}
		if remote.URL == "" {
			errors = append(errors, fmt.Sprintf("remotes.%d.url: required field is missing", i))
		} else {
			// URL形式のチェック
			if _, err := url.Parse(remote.URL); err != nil {
				errors = append(errors, fmt.Sprintf("remotes.%d.url: invalid URL format", i))
			}
		}
	}

	// 厳格モードの場合、既知のキー以外を警告
	if strict {
		knownKeys := map[string]bool{
			"repos":      true,
			"workspaces": true,
			"env":        true,
			"remotes":    true,
		}
		for key := range rawConfig {
			if !knownKeys[key] {
				warnings = append(warnings, fmt.Sprintf("unused or unknown key: %s", key))
			}
		}
	}

	// 結果を出力
	if len(errors) > 0 {
		fmt.Println("✗ Configuration file is invalid:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("validation failed")
	}

	fmt.Println("✓ Configuration file is valid")

	if len(warnings) > 0 {
		for _, warn := range warnings {
			fmt.Printf("⚠ Warning: %s\n", warn)
		}
	}

	return nil
}
