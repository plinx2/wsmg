package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
)

func newConfigInitCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Force       bool `flag:"force" short:"f" default:"false" usage:"既存の設定ファイルを確認なしで上書き"`
		Interactive bool `flag:"interactive" short:"i" default:"false" usage:"対話形式で設定値を入力"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long:  `設定ファイルを初期化します。既に設定ファイルが存在する場合は、上書き確認を行います。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigInit(opts.Force, opts.Interactive)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigInit(force, interactive bool) error {
	// ホームディレクトリを取得
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("ホームディレクトリの取得に失敗しました: %w", err)
	}

	// 設定ディレクトリのパス
	configDir := filepath.Join(home, ".config", "wsmg")
	configFile := filepath.Join(configDir, "wsmg.json")

	// 既に設定ファイルが存在する場合
	if _, err := os.Stat(configFile); err == nil {
		if !force {
			fmt.Printf("設定ファイルが既に存在します: %s\n", configFile)
			fmt.Print("上書きしますか? (y/N): ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("キャンセルしました。")
				return nil
			}
		}
	}

	// 設定ディレクトリを作成
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("設定ディレクトリの作成に失敗しました: %w", err)
	}

	// デフォルト設定を作成
	config := map[string]interface{}{
		"repos":      filepath.Join(home, "repos"),
		"workspaces": filepath.Join(home, "workspaces"),
		"env":        []interface{}{},
		"remotes":    []interface{}{},
	}

	// 対話形式の場合
	if interactive {
		fmt.Println("\n設定を入力してください（Enterでデフォルト値）:")

		// repos ディレクトリ
		fmt.Printf("repos ディレクトリ [%s]: ", config["repos"])
		var reposInput string
		fmt.Scanln(&reposInput)
		if reposInput != "" {
			config["repos"] = reposInput
		}

		// workspaces ディレクトリ
		fmt.Printf("workspaces ディレクトリ [%s]: ", config["workspaces"])
		var workspacesInput string
		fmt.Scanln(&workspacesInput)
		if workspacesInput != "" {
			config["workspaces"] = workspacesInput
		}
	}

	// JSONファイルに書き込み
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON の生成に失敗しました: %w", err)
	}

	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗しました: %w", err)
	}

	fmt.Printf("\n設定ファイルを作成しました: %s\n\n", configFile)
	fmt.Println("デフォルト設定:")
	fmt.Printf("  repos:      %s\n", config["repos"])
	fmt.Printf("  workspaces: %s\n", config["workspaces"])

	return nil
}
