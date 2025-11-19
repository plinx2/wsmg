package config

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestGetCopyPatterns(t *testing.T) {
	tests := []struct {
		name           string
		configPatterns []string
		want           []string
	}{
		{
			name:           "設定なし（デフォルトパターン）",
			configPatterns: nil,
			want: []string{
				".env*",
				".vscode/*",
				".tool-versions",
				".*-version",
				".nvmrc",
				".npmrc",
				"go.work",
			},
		},
		{
			name:           "カスタムパターン",
			configPatterns: []string{".env*", ".custom"},
			want:           []string{".env*", ".custom"},
		},
		{
			name:           "空の設定（デフォルトパターン）",
			configPatterns: []string{},
			want: []string{
				".env*",
				".vscode/*",
				".tool-versions",
				".*-version",
				".nvmrc",
				".npmrc",
				"go.work",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// viper をリセット
			viper.Reset()

			// 設定をセット
			if tt.configPatterns != nil {
				viper.Set("copy_patterns", tt.configPatterns)
			}

			got := GetCopyPatterns()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCopyPatterns() = %v, want %v", got, tt.want)
			}
		})
	}
}
