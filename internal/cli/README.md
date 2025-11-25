# CLI フラグヘルパー

`cli.BindFlags` は構造体のタグ情報を元に、cobraコマンドにフラグを自動でバインドするヘルパー関数です。

## 特徴

- 構造体のタグからフラグを自動生成
- `required` タグで必須フラグを指定
- `choices` タグで選択肢を制限（自動バリデーション）
- 型安全で読みやすい
- ボイラープレートが最小限

## サポートされる型

- `string`
- `bool`
- `int`
- `[]string`

## サポートされるタグ

| タグ | 説明 | 例 |
|------|------|-----|
| `flag` | フラグ名（省略時はフィールド名をkebab-caseに変換） | `flag:"filter"` |
| `short` | ショートハンド（1文字） | `short:"f"` |
| `default` | デフォルト値 | `default:"table"` |
| `usage` | 説明文 | `usage:"フィルタパターン"` |
| `required` | 必須フラグ（`"true"` で指定） | `required:"true"` |
| `choices` | カンマ区切りで選択肢を指定（バリデーション付き） | `choices:"table,json,yaml"` |

## 使用例

### 基本的な使用方法

```go
package main

import (
    "github.com/spf13/cobra"
    "github.com/plinx2/wsmg/internal/cli"
)

func newExampleCmd() *cobra.Command {
    // オプション定義
    opts := &struct {
        Filter string `flag:"filter" short:"f" usage:"フィルタパターン"`
        Format string `flag:"format" default:"table" usage:"出力フォーマット"`
        Verbose bool  `flag:"verbose" short:"v" default:"false" usage:"詳細出力"`
    }{}

    // コマンド定義
    cmd := &cobra.Command{
        Use:   "example",
        Short: "サンプルコマンド",
        RunE: func(c *cobra.Command, args []string) error {
            // opts のフィールドを直接使用
            return runExample(opts.Filter, opts.Format, opts.Verbose)
        },
    }

    // フラグを自動バインド
    if err := cli.BindFlags(cmd, opts); err != nil {
        panic(err)
    }

    return cmd
}
```

### 選択肢の制限（choices）

```go
opts := &struct {
    Format string `flag:"format" default:"table" usage:"出力フォーマット" choices:"table,json,yaml"`
    Sort   string `flag:"sort" default:"path" usage:"ソートフィールド" choices:"path,name,date"`
}{}
```

実行例：
```bash
$ ./wsmg example --format=csv
Error: invalid value for --format: csv (must be one of: table, json, yaml)
```

### 必須フラグ（required）

```go
opts := &struct {
    Config string `flag:"config" short:"c" usage:"設定ファイルのパス" required:"true"`
    Force  bool   `flag:"force" usage:"強制実行" required:"true"`
}{}
```

実行例：
```bash
$ ./wsmg example
Error: required flag(s) "config", "force" not set
```

### スライス型

```go
opts := &struct {
    Repos []string `flag:"repos" short:"r" usage:"リポジトリのリスト（カンマ区切り）"`
    Tags  []string `flag:"tags" default:"latest,stable" usage:"タグのリスト"`
}{}
```

実行例：
```bash
$ ./wsmg example --repos=repo1,repo2,repo3
# opts.Repos = []string{"repo1", "repo2", "repo3"}
```

### フィールド名からフラグ名を自動生成

`flag` タグを省略すると、フィールド名を kebab-case に変換してフラグ名として使用します。

```go
opts := &struct {
    FilterPattern string `usage:"フィルタパターン"`  // --filter-pattern
    MaxCount      int    `usage:"最大件数"`          // --max-count
    EnableDebug   bool   `usage:"デバッグモード"`    // --enable-debug
}{}
```

## 実装例

プロジェクト内の実装例：

- `cmd/wsmg/repos_list.go` - 基本的な使用例
- `cmd/wsmg/workspace_create.go` - スライス型とショートハンドの使用例
- `cmd/wsmg/workspace_list.go` - bool型とchoicesの使用例

## 注意点

### エクスポートされていないフィールド

エクスポートされていない（小文字で始まる）フィールドはスキップされます。

```go
opts := &struct {
    Public  string `flag:"public" usage:"公開フラグ"`    // ✓ バインドされる
    private string `flag:"private" usage:"非公開フラグ"` // ✗ スキップされる
}{}
```

### choices と PreRunE の競合

`choices` タグを使用すると、`PreRunE` が自動的に設定されます。
既存の `PreRunE` がある場合は、それも呼び出されるため、競合は発生しません。

```go
cmd := &cobra.Command{
    Use: "example",
    PreRunE: func(c *cobra.Command, args []string) error {
        // カスタムバリデーション
        return nil
    },
    RunE: func(c *cobra.Command, args []string) error {
        // メイン処理
        return nil
    },
}

// PreRunE が上書きされることなく、両方が実行される
cli.BindFlags(cmd, opts)
```

## エラーハンドリング

`BindFlags` はエラーを返すので、適切に処理してください。

```go
if err := cli.BindFlags(cmd, opts); err != nil {
    // 開発時のエラーなので、panicで問題ない
    panic(fmt.Sprintf("failed to bind flags: %v", err))
}
```

## テスト

各コマンドをテストする際は、オプション構造体を直接使用できます。

```go
func TestRunExample(t *testing.T) {
    err := runExample("test-filter", "json", true)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```
