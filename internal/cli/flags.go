package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// BindFlags は構造体のタグ情報を元に、cobraコマンドにフラグを自動でバインドします。
//
// サポートされるタグ:
//   - flag:     フラグ名（省略時はフィールド名をkebab-caseに変換）
//   - short:    ショートハンド（1文字）
//   - default:  デフォルト値
//   - usage:    説明文
//   - required: "true" の場合、必須フラグとしてマーク
//   - choices:  カンマ区切りで選択肢を指定（バリデーション付き）
//
// 使用例:
//
//	opts := &struct {
//	    Filter string `flag:"filter" short:"f" usage:"フィルタパターン"`
//	    Format string `flag:"format" default:"table" usage:"出力フォーマット" choices:"table,json,yaml"`
//	    Force  bool   `flag:"force" required:"true" usage:"強制実行"`
//	}{}
//	cli.BindFlags(cmd, opts)
func BindFlags(cmd *cobra.Command, opts any) error {
	val := reflect.ValueOf(opts)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("opts must be a pointer to struct")
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// エクスポートされていないフィールドはスキップ
		if !field.CanAddr() {
			continue
		}

		// タグから情報を取得
		flagName := fieldType.Tag.Get("flag")
		if flagName == "" {
			// flag タグがない場合はフィールド名を kebab-case に変換
			flagName = toKebabCase(fieldType.Name)
		}

		shorthand := fieldType.Tag.Get("short")
		defaultValue := fieldType.Tag.Get("default")
		usage := fieldType.Tag.Get("usage")
		required := fieldType.Tag.Get("required") == "true"
		choicesStr := fieldType.Tag.Get("choices")

		// 型に応じてフラグを登録
		if err := bindFlag(cmd, field, flagName, shorthand, defaultValue, usage); err != nil {
			return fmt.Errorf("field %s: %w", fieldType.Name, err)
		}

		// required フラグの設定
		if required {
			if err := cmd.MarkFlagRequired(flagName); err != nil {
				return fmt.Errorf("field %s: failed to mark as required: %w", fieldType.Name, err)
			}
		}

		// choices のバリデーション設定
		if choicesStr != "" {
			choices := strings.Split(choicesStr, ",")
			for i, c := range choices {
				choices[i] = strings.TrimSpace(c)
			}

			// PreRunE でバリデーション
			originalPreRunE := cmd.PreRunE
			cmd.PreRunE = func(c *cobra.Command, args []string) error {
				if originalPreRunE != nil {
					if err := originalPreRunE(c, args); err != nil {
						return err
					}
				}
				return validateChoices(c, flagName, choices, field)
			}
		}
	}

	return nil
}

// bindFlag は型に応じて適切なフラグをバインドします
func bindFlag(cmd *cobra.Command, field reflect.Value, flagName, shorthand, defaultValue, usage string) error {
	switch field.Kind() {
	case reflect.String:
		ptr := field.Addr().Interface().(*string)
		if shorthand != "" {
			cmd.Flags().StringVarP(ptr, flagName, shorthand, defaultValue, usage)
		} else {
			cmd.Flags().StringVar(ptr, flagName, defaultValue, usage)
		}

	case reflect.Bool:
		ptr := field.Addr().Interface().(*bool)
		defaultBool := defaultValue == "true"
		if shorthand != "" {
			cmd.Flags().BoolVarP(ptr, flagName, shorthand, defaultBool, usage)
		} else {
			cmd.Flags().BoolVar(ptr, flagName, defaultBool, usage)
		}

	case reflect.Int:
		ptr := field.Addr().Interface().(*int)
		defaultInt := 0
		if defaultValue != "" {
			var err error
			defaultInt, err = strconv.Atoi(defaultValue)
			if err != nil {
				return fmt.Errorf("invalid default value for int: %s", defaultValue)
			}
		}
		if shorthand != "" {
			cmd.Flags().IntVarP(ptr, flagName, shorthand, defaultInt, usage)
		} else {
			cmd.Flags().IntVar(ptr, flagName, defaultInt, usage)
		}

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			ptr := field.Addr().Interface().(*[]string)
			var defaultSlice []string
			if defaultValue != "" {
				defaultSlice = strings.Split(defaultValue, ",")
				for i, s := range defaultSlice {
					defaultSlice[i] = strings.TrimSpace(s)
				}
			}
			if shorthand != "" {
				cmd.Flags().StringSliceVarP(ptr, flagName, shorthand, defaultSlice, usage)
			} else {
				cmd.Flags().StringSliceVar(ptr, flagName, defaultSlice, usage)
			}
		} else {
			return fmt.Errorf("unsupported slice type: %v", field.Type())
		}

	default:
		return fmt.Errorf("unsupported type: %v", field.Kind())
	}

	return nil
}

// validateChoices は指定された選択肢に含まれているかをバリデーションします
func validateChoices(cmd *cobra.Command, flagName string, choices []string, field reflect.Value) error {
	// フラグが変更されていない場合はスキップ
	if !cmd.Flags().Changed(flagName) {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		value := field.String()
		if value == "" {
			return nil // 空文字列は許可
		}
		for _, choice := range choices {
			if value == choice {
				return nil
			}
		}
		return fmt.Errorf("invalid value for --%s: %s (must be one of: %s)", flagName, value, strings.Join(choices, ", "))

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			values := field.Interface().([]string)
			for _, value := range values {
				found := false
				for _, choice := range choices {
					if value == choice {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("invalid value for --%s: %s (must be one of: %s)", flagName, value, strings.Join(choices, ", "))
				}
			}
		}
	}

	return nil
}

// toKebabCase は CamelCase を kebab-case に変換します
func toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32) // 小文字に変換
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
