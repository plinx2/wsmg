package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigPathCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Create bool `flag:"create" default:"false" usage:"パスが存在しない場合、ディレクトリを作成"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long:  `設定ファイルのパスを表示します。スクリプトから設定ファイルにアクセスする際に便利です。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigPath(opts.Create)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigPath(create bool) error {
	configFile := viper.ConfigFileUsed()

	if configFile == "" {
		// 設定ファイルが見つからない場合、デフォルトパスを返す
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("ホームディレクトリの取得に失敗しました: %w", err)
		}
		configFile = filepath.Join(home, ".config", "wsmg", "wsmg.json")
	}

	// ディレクトリを作成するオプションが指定されている場合
	if create {
		configDir := filepath.Dir(configFile)
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("ディレクトリの作成に失敗しました: %w", err)
			}
			fmt.Printf("Created directory: %s\n", configDir)
		}
	}

	fmt.Println(configFile)

	return nil
}
