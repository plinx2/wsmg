package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigUnsetCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Force bool `flag:"force" short:"f" default:"false" usage:"確認なしで削除"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Delete a configuration value",
		Long:  `指定したキーを設定ファイルから削除します。`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			key := args[0]
			return runConfigUnset(key, opts.Force)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigUnset(key string, force bool) error {
	// 必須キーのチェック
	requiredKeys := []string{"repos", "workspaces"}
	for _, reqKey := range requiredKeys {
		if key == reqKey {
			return fmt.Errorf("cannot unset required key: %s", key)
		}
	}

	// キーが存在するかチェック
	if !viper.IsSet(key) {
		return fmt.Errorf("キー '%s' は存在しません", key)
	}

	// 確認プロンプト
	if !force {
		fmt.Printf("キー '%s' を削除しますか? (y/N): ", key)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("キャンセルしました。")
			return nil
		}
	}

	// 設定ファイルを読み込み
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return fmt.Errorf("設定ファイルが見つかりません")
	}

	// JSONファイルを読み込み
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// JSONをパース
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("JSON のパースに失敗しました: %w", err)
	}

	// キーを削除（ドット記法に対応）
	if err := deleteNestedKey(config, key); err != nil {
		return fmt.Errorf("キーの削除に失敗しました: %w", err)
	}

	// JSONファイルに書き戻し
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON の生成に失敗しました: %w", err)
	}

	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗しました: %w", err)
	}

	fmt.Printf("Removed: %s\n", key)

	return nil
}

func deleteNestedKey(config map[string]interface{}, key string) error {
	parts := strings.Split(key, ".")

	if len(parts) == 1 {
		// トップレベルのキー
		delete(config, key)
		return nil
	}

	// ネストされたキー
	current := config
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return fmt.Errorf("キーが見つかりません: %s", key)
		}
	}

	delete(current, parts[len(parts)-1])
	return nil
}
