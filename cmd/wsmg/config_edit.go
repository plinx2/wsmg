package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigEditOptions はコンフィグ編集コマンドのオプションです
type ConfigEditOptions struct {
	Editor string `flag:"editor" short:"e" default:"" usage:"使用するエディタを指定"`
}

func newConfigEditCmd() *cobra.Command {
	// オプション定義
	opts := &ConfigEditOptions{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open configuration file in editor",
		Long: `設定ファイルをデフォルトエディタで開きます。
$EDITOR 環境変数が設定されていない場合は vi を使用します。`,
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigEdit(c.Context(), opts)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigEdit(ctx context.Context, opts *ConfigEditOptions) error {
	configFile := viper.ConfigFileUsed()

	if configFile == "" {
		return fmt.Errorf("設定ファイルが見つかりません。'wsmg config init' を実行してください")
	}

	// 使用するエディタを決定
	editor := opts.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	fmt.Printf("Opening %s with %s...\n", configFile, editor)

	// エディタを起動
	cmd := exec.Command(editor, configFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("エディタの起動に失敗しました: %w", err)
	}

	// 編集後、JSONの構文チェックを実行
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("\nWarning: JSON syntax error: %v\n", err)
		if cli.ConfirmAction("Re-edit?") {
			return runConfigEdit(ctx, opts)
		}
		return fmt.Errorf("config file has syntax error")
	}

	fmt.Println("Configuration saved successfully.")

	return nil
}
