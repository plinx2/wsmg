package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/plinx2/wsmg/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigShowCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Format  string `flag:"format" default:"table" usage:"出力フォーマット" choices:"table,json,yaml"`
		Sources bool   `flag:"sources" default:"false" usage:"各設定値の取得元を表示"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  `現在の設定を表示します。コマンドラインフラグ、環境変数、設定ファイルの優先順位を考慮した、実際に使用される設定値を表示します。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigShow(opts.Format, opts.Sources)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigShow(format string, showSources bool) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("設定の取得に失敗しました: %w", err)
	}

	switch format {
	case "json":
		return showConfigJSON(cfg)
	case "yaml":
		return showConfigYAML(cfg)
	default:
		return showConfigTable(cfg, showSources)
	}
}

func showConfigJSON(cfg *config.Config) error {
	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON の生成に失敗しました: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func showConfigYAML(cfg *config.Config) error {
	// 簡易的なYAML出力（yamlライブラリを使わない場合）
	fmt.Println("repos:", cfg.Repos)
	fmt.Println("workspaces:", cfg.Workspaces)

	if len(cfg.Env) > 0 {
		fmt.Println("env:")
		for _, entry := range cfg.Env {
			masked := maskSensitiveValue(entry.Key, entry.Value)
			fmt.Printf("  - key: %s\n", entry.Key)
			fmt.Printf("    value: %s\n", masked)
		}
	}

	if len(cfg.Remotes) > 0 {
		fmt.Println("remotes:")
		for _, remote := range cfg.Remotes {
			fmt.Println("  - type:", remote.Type)
			fmt.Println("    url:", remote.URL)
		}
	}

	return nil
}

func showConfigTable(cfg *config.Config, showSources bool) error {
	fmt.Println("Current Configuration:")
	fmt.Printf("  repos:      %s", cfg.Repos)
	if showSources {
		fmt.Printf(" [%s]", getConfigSource("repos"))
	}
	fmt.Println()

	fmt.Printf("  workspaces: %s", cfg.Workspaces)
	if showSources {
		fmt.Printf(" [%s]", getConfigSource("workspaces"))
	}
	fmt.Println()

	if len(cfg.Env) > 0 {
		fmt.Println("\nEnvironment Variables:")
		for i, entry := range cfg.Env {
			masked := maskSensitiveValue(entry.Key, entry.Value)
			fmt.Printf("  [%d] %s: %s", i, entry.Key, masked)
			if showSources {
				fmt.Printf(" [%s]", getConfigSource(fmt.Sprintf("env.%d.key", i)))
			}
			fmt.Println()
		}
	}

	if len(cfg.Remotes) > 0 {
		fmt.Println("\nRemotes:")
		for i, remote := range cfg.Remotes {
			fmt.Printf("  [%d] type: %s\n", i, remote.Type)
			fmt.Printf("      url:  %s\n", remote.URL)
		}
	}

	if showSources {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			configFile = "not found (using defaults)"
		}
		fmt.Printf("\nConfiguration file: %s\n", configFile)
	}

	return nil
}

func getConfigSource(key string) string {
	// フラグで設定されているかチェック
	if viper.IsSet(key) {
		// 簡易的な判定（実際はviperの内部状態を見る必要がある）
		return "config file"
	}
	return "default"
}

func maskSensitiveValue(key, value string) string {
	// トークンやパスワードなどの機密情報をマスク
	lowerKey := strings.ToLower(key)
	if strings.Contains(lowerKey, "token") ||
		strings.Contains(lowerKey, "password") ||
		strings.Contains(lowerKey, "secret") ||
		strings.Contains(lowerKey, "key") {
		if len(value) > 6 {
			return value[:3] + "***" + value[len(value)-3:]
		}
		return "******"
	}
	return value
}
