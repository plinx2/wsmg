package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigSetOptions はコンフィグ設定コマンドのオプションです
type ConfigSetOptions struct {
	Type string `flag:"type" short:"t" default:"" usage:"値の型を指定 (string, int, bool, array)"`
}

func newConfigSetCmd() *cobra.Command {
	// オプション定義
	opts := &ConfigSetOptions{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  `指定したキーに値を設定します。設定ファイルに永続化されます。`,
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigSet(c.Context(), args[0], args[1], opts)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigSet(ctx context.Context, key, value string, opts *ConfigSetOptions) error {
	// 値を適切な型に変換
	var convertedValue any
	var err error

	if opts.Type == "" {
		// 型を自動判定
		convertedValue, err = autoConvertValue(value)
		if err != nil {
			return fmt.Errorf("値の変換に失敗しました: %w", err)
		}
	} else {
		// 指定された型に変換
		convertedValue, err = convertValueByType(value, opts.Type)
		if err != nil {
			return fmt.Errorf("値の変換に失敗しました: %w", err)
		}
	}

	// viperに値を設定
	viper.Set(key, convertedValue)

	// 設定ファイルに書き込み
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return fmt.Errorf("設定ファイルが見つかりません。'wsmg config init' を実行してください")
	}

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗しました: %w", err)
	}

	// 確認メッセージを出力
	displayValue := value
	if isSensitiveKey(key) {
		displayValue = "****** (masked)"
	}
	fmt.Printf("Updated: %s = %s\n", key, displayValue)

	return nil
}

func autoConvertValue(value string) (any, error) {
	// bool値のチェック
	if value == "true" {
		return true, nil
	}
	if value == "false" {
		return false, nil
	}

	// int値のチェック
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal, nil
	}

	// float値のチェック
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}

	// それ以外はstring
	return value, nil
}

func convertValueByType(value, valueType string) (any, error) {
	switch valueType {
	case "string":
		return value, nil
	case "int":
		return strconv.Atoi(value)
	case "bool":
		return strconv.ParseBool(value)
	case "array":
		// カンマ区切りで配列に変換
		parts := strings.Split(value, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", valueType)
	}
}

func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	return strings.Contains(lowerKey, "token") ||
		strings.Contains(lowerKey, "password") ||
		strings.Contains(lowerKey, "secret") ||
		strings.Contains(lowerKey, "key")
}
