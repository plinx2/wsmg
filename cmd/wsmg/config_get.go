package main

import (
	"context"
	"fmt"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigGetOptions はコンフィグ取得コマンドのオプションです
type ConfigGetOptions struct {
	Default string `flag:"default" short:"d" default:"" usage:"キーが存在しない場合のデフォルト値"`
}

func newConfigGetCmd() *cobra.Command {
	// オプション定義
	opts := &ConfigGetOptions{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `指定したキーの設定値を取得します。ドット記法で階層的なキーにアクセスできます。`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runConfigGet(c.Context(), args[0], opts)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigGet(ctx context.Context, key string, opts *ConfigGetOptions) error {
	// キーが存在するかチェック
	if !viper.IsSet(key) {
		if opts.Default != "" {
			fmt.Println(opts.Default)
			return nil
		}
		return fmt.Errorf("キー '%s' は存在しません", key)
	}

	// 値を取得
	value := viper.Get(key)

	// 値を出力
	switch v := value.(type) {
	case string:
		fmt.Println(v)
	case int, int64, float64:
		fmt.Println(v)
	case bool:
		fmt.Println(v)
	case []any:
		for _, item := range v {
			fmt.Println(item)
		}
	case map[string]any:
		for k, val := range v {
			fmt.Printf("%s: %v\n", k, val)
		}
	default:
		fmt.Printf("%v\n", v)
	}

	return nil
}
