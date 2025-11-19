package main

import (
	"fmt"

	"github.com/plinx2/wsmg/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigGetCmd() *cobra.Command {
	// オプション定義
	opts := &struct {
		Default string `flag:"default" short:"d" default:"" usage:"キーが存在しない場合のデフォルト値"`
	}{}

	// コマンド定義
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `指定したキーの設定値を取得します。ドット記法で階層的なキーにアクセスできます。`,
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			key := args[0]
			return runConfigGet(key, opts.Default)
		},
	}

	// フラグ定義
	if err := cli.BindFlags(cmd, opts); err != nil {
		panic(fmt.Sprintf("failed to bind flags: %v", err))
	}

	return cmd
}

func runConfigGet(key, defaultValue string) error {
	// キーが存在するかチェック
	if !viper.IsSet(key) {
		if defaultValue != "" {
			fmt.Println(defaultValue)
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
	case []interface{}:
		for _, item := range v {
			fmt.Println(item)
		}
	case map[string]interface{}:
		for k, val := range v {
			fmt.Printf("%s: %v\n", k, val)
		}
	default:
		fmt.Printf("%v\n", v)
	}

	return nil
}
