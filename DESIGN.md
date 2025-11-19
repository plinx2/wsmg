# wsmg - Design Document

## Project Overview

`wsmg` (Workspace Manager) is a command-line tool designed to streamline multi-repository development by leveraging Git worktree for ticket-based workflows.

## Core Concept

### Problem

- Working on tasks across multiple repositories requires switching branches in each repository
- Managing branch switches and stashes becomes complex when working on multiple tickets in parallel
- Lack of mechanism to manage related repositories together

### Solution

- Use Git worktree to create isolated working directories for each ticket
- Manage default branches in the `repos` directory and working branches in the `workspaces` directory
- Integrate with VSCode workspace feature to handle multiple repositories in a single workspace

## Directory Structure

### repos Directory

Clone destination for remote repositories. Primarily maintains the default branch (main/master) state.

```
~/repos/
├── github.com/
│   ├── org1/
│   │   ├── repo1/  # Default branch
│   │   └── repo2/  # Default branch
│   └── org2/
│       └── repo3/  # Default branch
└── gitlab.com/
    └── org3/
        └── repo4/  # Default branch
```

### workspaces Directory

Creates workspaces per ticket. Working branches for each repository are placed using Git worktree.

```
~/workspaces/
├── TICKET-001/
│   ├── github.com/
│   │   └── org1/
│   │       ├── repo1/  # Git worktree (TICKET-001 branch)
│   │       └── repo2/  # Git worktree (TICKET-001 branch)
│   └── TICKET-001.code-workspace
├── TICKET-002/
│   ├── github.com/
│   │   └── org1/
│   │       └── repo1/  # Git worktree (TICKET-002 branch)
│   └── TICKET-002.code-workspace
└── feature-xyz/
    ├── github.com/
    │   └── org2/
    │       └── repo3/  # Git worktree (feature-xyz branch)
    └── feature-xyz.code-workspace
```

## Configuration File

### Configuration File Path

Default: `~/.config/wsmg/wsmg.json`

Alternative path can be specified with `--config` command-line option.

Configuration precedence by viper (highest to lowest):

1. Command-line flags (`--repos`, `--workspaces`, etc.)
2. Environment variables (`WSMG_REPOS`, `WSMG_WORKSPACES`, etc.)
3. Configuration file (`wsmg.json`)
4. Default values

### Configuration Structure

```json
{
  "repos": "~/repos",
  "workspaces": "~/workspaces",
  "env": [
    {
      "key": "GITHUB_TOKEN",
      "value": "xxxxxx"
    },
    {
      "key": "GITLAB_TOKEN",
      "value": "yyyyyy"
    }
  ],
  "remotes": [
    {
      "type": "GitHub",
      "url": "https://github.com"
    },
    {
      "type": "GitLab",
      "url": "https://gitlab.com"
    }
  ]
}
```

#### Configuration Fields

##### `repos`

- **Type**: string
- **Default**: `~/repos`
- **Description**: Directory for cloning repositories

##### `workspaces`

- **Type**: string
- **Default**: `~/workspaces`
- **Description**: Directory for creating workspaces

##### `env`

- **Type**: array of objects (key-value pairs)
- **Default**: `[]`
- **Description**: Environment variables loaded automatically when running commands. Configure API tokens, etc. Each entry contains `key` and `value` fields.

##### `remotes`

- **Type**: array of objects
- **Default**: `[]`
- **Description**: Configuration for Git servers to access

###### remote Object Structure

- `type`: Type of Git server ("GitHub", "GitLab", etc.)
- `url`: Git server URL

## Command Specification

### Global Options

- `--config <path>`: Specify configuration file path
- `--repos <path>`: Specify repos directory path (overrides configuration file)
- `--workspaces <path>`: Specify workspaces directory path (overrides configuration file)

### repos Subcommands

#### `wsmg repos list`

Display a list of repositories in the repos directory.

**Options:**

- `--filter <pattern>`: Filter by repository name or path
- `--format <format>`: Output format (table, json, yaml)
- `--sort <field>`: Sort field (path, lastcommit, name)

**Example output:**

```
PATH                           BRANCH  LAST COMMIT                  LAST MODIFIED
github.com/org1/repo1          main    abc1234 Fix bug             2025-11-15 10:30
github.com/org1/repo2          main    def5678 Add feature         2025-11-14 15:20
github.com/org2/repo3          master  ghi9012 Update docs         2025-11-10 09:15
```

#### `wsmg repos sync`

repos 内のリポジトリのデフォルトブランチを最新に更新します。

**オプション:**

- `--filter <pattern>`: 同期対象のリポジトリをフィルタリング
- `--prune`: リモートで削除されたブランチをローカルからも削除

### config サブコマンド

#### `wsmg config init`

設定ファイルを初期化します。既に設定ファイルが存在する場合は、上書き確認を行います。

**オプション:**

- `--force`: 既存の設定ファイルを確認なしで上書き
- `--interactive`: 対話形式で設定値を入力

**動作:**

1. `~/.config/wsmg/` ディレクトリを作成（存在しない場合）
2. デフォルト値を含む `wsmg.json` を作成
3. 対話モードの場合は repos, workspaces のパスを入力

**出力例:**

```
設定ファイルを作成しました: ~/.config/wsmg/wsmg.json

デフォルト設定:
  repos:      ~/repos
  workspaces: ~/workspaces
```

#### `wsmg config show`

現在の設定を表示します。コマンドラインフラグ、環境変数、設定ファイルの優先順位を考慮した、実際に使用される設定値を表示します。

**オプション:**

- `--format <format>`: 出力フォーマット（table, json, yaml）
- `--sources`: 各設定値の取得元を表示

**出力例:**

```
Current Configuration:
  repos:      /home/user/repos (config file)
  workspaces: /home/user/workspaces (config file)

Environment Variables:
  GITHUB_TOKEN: ****** (masked)

Remotes:
  - type: GitHub
    url:  https://github.com
```

`--sources` オプション使用時:

```
Configuration Sources:
  repos:      /home/user/repos      [config file: ~/.config/wsmg/wsmg.json]
  workspaces: /home/user/workspaces [config file: ~/.config/wsmg/wsmg.json]

Configuration file: ~/.config/wsmg/wsmg.json
```

#### `wsmg config get <key>`

指定したキーの設定値を取得します。ドット記法で階層的なキーにアクセスできます。

**引数:**

- `<key>`: 取得する設定キー（例: `repos`, `env.GITHUB_TOKEN`, `remotes.0.type`）

**オプション:**

- `--default <value>`: キーが存在しない場合のデフォルト値

**出力例:**

```bash
$ wsmg config get repos
/home/user/repos

$ wsmg config get env.GITHUB_TOKEN
ghp_xxxxxxxxxxxx

$ wsmg config get remotes.0.type
GitHub

$ wsmg config get notexist --default "fallback"
fallback
```

#### `wsmg config set <key> <value>`

指定したキーに値を設定します。設定ファイルに永続化されます。

**引数:**

- `<key>`: 設定するキー（例: `repos`, `env.GITHUB_TOKEN`）
- `<value>`: 設定する値

**オプション:**

- `--type <type>`: 値の型を指定（string, int, bool, array）。省略時は自動判定

**出力例:**

```bash
$ wsmg config set repos ~/my-repos
Updated: repos = ~/my-repos

$ wsmg config set env.GITHUB_TOKEN ghp_xxxxxxxxxxxx
Updated: env.GITHUB_TOKEN = ****** (masked)
```

**注意:**

- 配列やオブジェクトの追加には `wsmg config edit` を使用することを推奨
- トークンなどの機密情報は自動的にマスクして表示

#### `wsmg config unset <key>`

指定したキーを設定ファイルから削除します。

**引数:**

- `<key>`: 削除するキー

**オプション:**

- `--force`: 確認なしで削除

**出力例:**

```bash
$ wsmg config unset env.GITHUB_TOKEN
Removed: env.GITHUB_TOKEN

$ wsmg config unset repos
Error: cannot unset required key: repos
```

#### `wsmg config edit`

設定ファイルをデフォルトエディタで開きます。`$EDITOR` 環境変数が設定されていない場合は `vi` を使用します。

**オプション:**

- `--editor <command>`: 使用するエディタを指定

**動作:**

1. 設定ファイルをエディタで開く
2. 編集後、JSON の構文チェックを実行
3. エラーがある場合は再編集を促す

**出力例:**

```bash
$ wsmg config edit
Opening ~/.config/wsmg/wsmg.json with vim...
Configuration saved successfully.
```

#### `wsmg config path`

設定ファイルのパスを表示します。スクリプトから設定ファイルにアクセスする際に便利です。

**オプション:**

- `--create`: パスが存在しない場合、ディレクトリを作成

**出力例:**

```bash
$ wsmg config path
~/.config/wsmg/wsmg.json

$ wsmg config path --create
Created directory: ~/.config/wsmg
~/.config/wsmg/wsmg.json
```

#### `wsmg config validate`

設定ファイルの妥当性を検証します。

**オプション:**

- `--strict`: 厳格なバリデーション（未使用キーなども警告）

**出力例:**

```bash
$ wsmg config validate
✓ Configuration file is valid

$ wsmg config validate --strict
✓ Configuration file is valid
⚠ Warning: unused key "deprecated_option"
```

エラーがある場合:

```bash
$ wsmg config validate
✗ Configuration file is invalid:
  - remotes.0.url: invalid URL format
  - repos: path does not exist
```

### remote サブコマンド

#### `wsmg remote list`

設定されたリモートサーバーからリポジトリ一覧を取得して表示します。

**オプション:**

- `--filter <pattern>`: リポジトリ名や組織名でフィルタリング
- `--org <organization>`: 特定の組織のみ表示
- `--format <format>`: 出力フォーマット（table, json, yaml）
- `--no-cache`: キャッシュを使用せず、常に API から取得
- `--refresh`: キャッシュを更新
- `--details`: 詳細情報を表示（説明、スター数、最終更新日時など）
- `--archived`: アーカイブされたリポジトリも表示

**動作:**

1. 設定ファイルの `remotes` から接続先を読み込み
2. 各プロバイダー（GitHub, GitLab 等）の API を呼び出し
3. レート制限を考慮してキャッシュを活用
4. 結果を集約して指定フォーマットで出力

**出力例:**

```
PROVIDER    ORGANIZATION  REPOSITORY      DESCRIPTION              STARS  LANGUAGE    UPDATED
--------    ------------  ----------      -----------              -----  --------    -------
github.com  org1          backend-api     Backend REST API         123    Go          2025-11-15
github.com  org1          frontend-app    Frontend React App       456    TypeScript  2025-11-14
github.com  org2          docs            Documentation            45     Markdown    2025-11-10
gitlab.com  group1        infra           Infrastructure as Code   89     Terraform   2025-11-12

Total: 4 repositories
```

**詳細モード（--details）:**

```
Repository: github.com/org1/backend-api
  Description: Backend REST API
  URL: https://github.com/org1/backend-api
  Clone URL: git@github.com:org1/backend-api.git
  Default Branch: main
  Stars: 123 | Forks: 45 | Language: Go
  Private: false | Archived: false
  Last Updated: 2025-11-15 14:30:00
```

**キャッシュの動作:**

- デフォルトでキャッシュを使用（有効期限: 1 時間）
- `~/.cache/wsmg/remote-cache.json` に保存
- `--no-cache` で無効化、`--refresh` で強制更新

#### `wsmg remote clone <repository>`

リモートリポジトリを repos ディレクトリにクローンします。

**引数:**

- `<repository>`: クローンするリポジトリ
  - 完全形式: `github.com/org1/repo1`
  - 短縮形式: `org1/repo1` (デフォルトプロバイダーを使用)

**オプション:**

- `--branch <branch>`: デフォルト以外のブランチを指定
- `--ssh`: SSH URL を使用してクローン（デフォルト: HTTPS）
- `--depth <n>`: shallow clone の深さを指定
- `--no-checkout`: チェックアウトせずにクローン

**動作:**

1. リポジトリ指定を解析（プロバイダー、組織、リポジトリ名）
2. リモート API からリポジトリ情報を取得
3. `repos/プロバイダー/組織/リポジトリ` の構造でクローン
4. デフォルトブランチを設定

**出力例:**

```bash
$ wsmg remote clone github.com/org1/backend-api
Fetching repository info from github.com...
Cloning into '/home/user/repos/github.com/org1/backend-api'...
remote: Enumerating objects: 1234, done.
remote: Counting objects: 100% (1234/1234), done.
remote: Compressing objects: 100% (567/567), done.
remote: Total 1234 (delta 678), reused 1123 (delta 589)
Receiving objects: 100% (1234/1234), 2.45 MiB | 5.23 MiB/s, done.
Resolving deltas: 100% (678/678), done.

✓ Cloned: github.com/org1/backend-api
  Path: /home/user/repos/github.com/org1/backend-api
  Branch: main
```

#### `wsmg remote clone --interactive`

対話形式で複数のリポジトリを選択してクローンします。

**オプション:**

- `--org <organization>`: 特定の組織のリポジトリから選択
- `--filter <pattern>`: 表示するリポジトリをフィルタリング
- `--ssh`: SSH URL を使用

**動作:**

```
リモートリポジトリを選択してください（番号をカンマ区切りで入力、または 'all' で全て選択）:

  [1] github.com/org1/backend-api (Go, ⭐123)
  [2] github.com/org1/frontend-app (TypeScript, ⭐456)
  [3] github.com/org2/docs (Markdown, ⭐45)

選択 (例: 1,3 または all): 1,2

選択されたリポジトリ: 2 個
  - github.com/org1/backend-api
  - github.com/org1/frontend-app

クローンを開始しますか? (y/N): y

Cloning backend-api... ✓
Cloning frontend-app... ✓

Summary:
  Success: 2 repositories
```

#### `wsmg remote sync`

すでにクローンされているリポジトリのリモート情報を同期します。

**オプション:**

- `--add-missing`: リモートに存在するがローカルにないリポジトリを表示
- `--remove-deleted`: リモートで削除されたリポジトリをローカルからも削除

**動作:**

```bash
$ wsmg remote sync --add-missing
Checking remote repositories...

Missing repositories (exists on remote but not local):
  - github.com/org1/new-repo
  - github.com/org1/another-repo

Clone missing repositories? (y/N): y
```

#### `wsmg remote orgs`

アクセス可能な組織/グループの一覧を表示します。

**オプション:**

- `--format <format>`: 出力フォーマット（table, json, yaml）

**出力例:**

```
PROVIDER    ORGANIZATION  REPOSITORIES  MEMBERS
--------    ------------  ------------  -------
github.com  org1          23            15
github.com  org2          45            8
gitlab.com  group1        12            6

Total: 3 organizations
```

### workspace サブコマンド

#### `wsmg workspace list`

作成済みのワークスペース一覧を表示します。

**オプション:**

- `--format <format>`: 出力フォーマット（table, json, yaml）
- `--details`: 詳細情報を表示（含まれるリポジトリ、ブランチ状態など）

**出力例:**

```
WORKSPACE      REPOSITORIES  LAST MODIFIED        STATUS
TICKET-001     2             2025-11-15 14:30     Active
TICKET-002     1             2025-11-14 09:15     Active
feature-xyz    1             2025-11-10 16:45     Stale
```

#### `wsmg workspace create <ticket-name>`

新しいワークスペースを作成します。

**引数:**

- `<ticket-name>`: ワークスペース名（ブランチ名としても使用）

**オプション:**

- `--repos <repo1,repo2,...>`: 含めるリポジトリを指定（対話モードをスキップ）
- `--base-branch <branch>`: 作業ブランチの元となるブランチ（デフォルト: main/master）
- `--no-vscode`: VSCode ワークスペースファイルを作成しない

**動作:**

1. 対話モードで repos 内のリポジトリから選択（`--repos`で省略可）
2. 各リポジトリで`<ticket-name>`ブランチを作成
3. Git worktree で`workspaces/<ticket-name>/`配下に作業ディレクトリを作成
4. VSCode ワークスペース設定ファイルを生成

#### `wsmg workspace delete <ticket-name>`

ワークスペースを削除します。

**引数:**

- `<ticket-name>`: 削除するワークスペース名

**オプション:**

- `--force`: 未コミットの変更があっても強制削除
- `--keep-branches`: ブランチは削除せず worktree のみ削除
- `--delete-remote-branches`: リモートのブランチも削除

**動作:**

1. 未コミットの変更がある場合は警告（`--force`で省略可）
2. Git worktree を削除
3. ブランチを削除（`--keep-branches`で省略可）
4. ワークスペースディレクトリを削除

#### `wsmg workspace open <ticket-name>`

ワークスペースを VSCode で開きます。

**引数:**

- `<ticket-name>`: 開くワークスペース名

#### `wsmg workspace status [ticket-name]`

ワークスペースの状態を表示します。

**引数:**

- `[ticket-name]`: ワークスペース名（省略時は全て）

**出力:**

- 各リポジトリのブランチ状態
- 未コミットの変更
- リモートとの差分（ahead/behind）

## 追加機能の提案

### 1. ブランチ戦略の統合

**`wsmg workspace merge <ticket-name>`**

ワークスペース内の全リポジトリのブランチをデフォルトブランチにマージします。

- PR の作成（GitHub/GitLab API 利用）
- マージ後の自動クリーンアップ

### 2. テンプレート機能

**`wsmg template create <template-name>`**

よく使うリポジトリの組み合わせをテンプレートとして保存します。

```json
{
  "templates": {
    "fullstack": {
      "repos": [
        "github.com/org1/backend",
        "github.com/org1/frontend",
        "github.com/org1/shared"
      ],
      "env": {
        "API_URL": "http://localhost:3000"
      }
    }
  }
}
```

**`wsmg workspace create <ticket-name> --template fullstack`**

テンプレートを使用してワークスペースを作成します。

### 3. スナップショット機能

**`wsmg workspace snapshot <ticket-name>`**

ワークスペースの現在の状態（各リポジトリのコミットハッシュ、未コミットの変更など）を保存します。

**`wsmg workspace restore <ticket-name> <snapshot-id>`**

保存したスナップショットからワークスペースを復元します。

### 4. 依存関係管理

**`wsmg deps check`**

ワークスペース内のリポジトリ間の依存関係（例: package.json の依存）をチェックし、バージョンの不整合を検出します。

**`wsmg deps update`**

依存関係を一括で更新します。

### 5. クリーンアップ機能

**`wsmg cleanup stale`**

長期間更新されていないワークスペースやブランチを検出します。

**オプション:**

- `--days <n>`: n 日以上更新されていないものを対象
- `--dry-run`: 削除対象を表示のみ

**`wsmg cleanup merged`**

既にマージ済みのブランチを持つワークスペースを削除します。

### 6. 統計情報

**`wsmg stats`**

- ワークスペース数
- リポジトリ数
- ディスク使用量
- 最もアクティブなワークスペース

### 7. フック機能

ワークスペースの作成/削除時に任意のスクリプトを実行します。

```json
{
  "hooks": {
    "post-create": "~/.config/wsmg/hooks/post-create.sh",
    "pre-delete": "~/.config/wsmg/hooks/pre-delete.sh"
  }
}
```

### 8. インタラクティブ TUI

**`wsmg tui`**

ターミナルベースのインタラクティブ UI を提供します（[bubbletea](https://github.com/charmbracelet/bubbletea)などを使用）。

- ワークスペースの一覧表示・選択
- リポジトリの選択・管理
- ステータスの可視化

### 9. リポジトリのアーカイブ

**`wsmg repos archive <repository>`**

使用していないリポジトリをアーカイブします（削除せずに別の場所に移動）。

### 10. 同期機能の拡張

**`wsmg workspace sync <ticket-name>`**

ワークスペース内の全リポジトリでデフォルトブランチの変更を作業ブランチにマージまたはリベースします。

**オプション:**

- `--strategy <merge|rebase>`: 統合方法を指定

### 11. ワークスペースのエクスポート/インポート

**`wsmg workspace export <ticket-name>`**

ワークスペースの設定（リポジトリ一覧、ブランチ名など）をエクスポートします。

**`wsmg workspace import <export-file>`**

エクスポートした設定からワークスペースを再作成します。チーム間での環境共有に便利です。

## アーキテクチャ

### ディレクトリ構成

```
wsmg/
├── cmd/
│   └── wsmg/
│       ├── main.go
│       ├── root.go
│       ├── config.go           # config コマンド本体
│       ├── config_*.go         # config サブコマンド群
│       ├── repos.go            # repos コマンド本体
│       ├── repos_*.go          # repos サブコマンド群
│       ├── remote.go           # remote コマンド本体
│       ├── remote_*.go         # remote サブコマンド群
│       ├── workspace.go        # workspace コマンド本体
│       └── workspace_*.go      # workspace サブコマンド群
├── internal/
│   ├── config/
│   │   ├── config.go        # 設定ファイルの読み込み・管理
│   │   └── defaults.go      # デフォルト値の定義
│   ├── cli/
│   │   ├── flags.go         # フラグ自動バインディング
│   │   └── README.md        # 使用方法ドキュメント
│   ├── git/
│   │   ├── repository.go    # ローカルリポジトリ操作
│   │   ├── worktree.go      # Git worktree操作
│   │   ├── clone.go         # クローン操作
│   │   └── errors.go        # エラー型定義
│   ├── remote/
│   │   ├── provider.go      # リモートプロバイダーのインターフェース
│   │   ├── repository.go    # RemoteRepository構造体
│   │   ├── github.go        # GitHub API連携
│   │   ├── gitlab.go        # GitLab API連携
│   │   ├── cache.go         # キャッシュ機能
│   │   └── errors.go        # エラー型定義
│   ├── workspace/
│   │   ├── workspace.go     # ワークスペース管理
│   │   └── vscode.go        # VSCode設定ファイル生成
│   └── ui/
│       ├── table.go         # テーブル表示
│       └── prompt.go        # 対話的入力
├── go.mod
├── go.sum
├── README.md
├── DESIGN.md
└── LICENSE
```

### 主要パッケージ

#### internal/config

設定ファイルの読み込みと管理を行います。viper を使用して設定ファイル、環境変数、コマンドラインフラグを統合的に管理します。

- `InitConfig()`: viper の初期化と設定ファイルの読み込み
- `GetConfig()`: 現在の設定を返す
- `GetString(key string)`: 設定値を文字列として取得
- `GetStringMap(key string)`: 設定値をマップとして取得
- `GetStringSlice(key string)`: 設定値をスライスとして取得
- `SetDefaults()`: デフォルト値を設定
- `SaveConfig()`: 設定ファイルを保存

viper の利点：

- 設定ファイル（JSON, YAML, TOML 等）、環境変数、コマンドラインフラグを統合管理
- cobra との緊密な連携
- 設定値の監視と動的リロード（必要に応じて）
- 階層的な設定のサポート

#### internal/cli

CLI フラグの管理を行います。構造体のタグから自動的に cobra のフラグを生成するヘルパー関数を提供します。

- `BindFlags(cmd *cobra.Command, opts interface{})`: 構造体のタグから自動的にフラグを生成してバインド

サポートされるタグ：

- `flag`: フラグ名（省略時はフィールド名を kebab-case に変換）
- `short`: ショートハンド（1 文字）
- `default`: デフォルト値
- `usage`: 説明文
- `required`: "true" の場合、必須フラグとしてマーク
- `choices`: カンマ区切りで選択肢を指定（自動バリデーション）

使用例：

```go
opts := &struct {
    Filter string `flag:"filter" short:"f" usage:"フィルタパターン"`
    Format string `flag:"format" default:"table" usage:"出力フォーマット" choices:"table,json,yaml"`
}{}
cli.BindFlags(cmd, opts)
```

この方法により：

- 各コマンドのオプションが完全に独立（名前空間の競合なし）
- タグによる宣言的な定義
- 型安全で読みやすい
- ボイラープレートが最小限

#### internal/git

Git 操作のラッパーです。

- `Repository`: リポジトリを表す構造体
- `CreateWorktree()`: worktree を作成
- `RemoveWorktree()`: worktree を削除
- `CreateBranch()`: ブランチを作成
- `DeleteBranch()`: ブランチを削除

#### internal/remote

リモート Git サーバーとの API 連携です。

##### provider.go - プロバイダーインターフェース

```go
type Provider interface {
    // Name はプロバイダー名を返す (例: "github", "gitlab")
    Name() string

    // ListRepositories はリポジトリ一覧を取得
    ListRepositories(ctx context.Context) ([]*RemoteRepository, error)

    // GetRepository は指定されたリポジトリの詳細情報を取得
    GetRepository(ctx context.Context, owner, repo string) (*RemoteRepository, error)
}
```

##### repository.go - リモートリポジトリ構造体

```go
type RemoteRepository struct {
    Provider      string    // プロバイダー名 ("github", "gitlab")
    Organization  string    // 組織名
    Name          string    // リポジトリ名
    FullName      string    // 完全名 (org/repo)
    Description   string    // 説明
    URL           string    // Web URL
    CloneURL      string    // Clone URL (HTTPS)
    SSHURL        string    // SSH URL
    DefaultBranch string    // デフォルトブランチ
    Stars         int       // スター数
    Forks         int       // フォーク数
    Language      string    // 主要言語
    Private       bool      // プライベートか
    Archived      bool      // アーカイブされているか
    UpdatedAt     time.Time // 最終更新日時
}
```

##### github.go - GitHub 実装

- トークン認証（Personal Access Token）
- リポジトリ一覧取得（ページネーション対応）
- 組織一覧取得
- レート制限の監視と対応
- キャッシュとの統合

##### gitlab.go - GitLab 実装

- トークン認証
- グループのリポジトリ取得
- サブグループ対応

##### cache.go - キャッシュ機能

```go
type Cache struct {
    Path string           // キャッシュファイルのパス
    TTL  time.Duration    // 有効期限
}

// Get はキャッシュからデータを取得
func (c *Cache) Get(key string) (interface{}, error)

// Set はキャッシュにデータを保存
func (c *Cache) Set(key string, value interface{}) error

// IsExpired はキャッシュが期限切れかチェック
func (c *Cache) IsExpired(key string) bool

// Clear はキャッシュをクリア
func (c *Cache) Clear() error
```

キャッシュ構造：

```json
{
  "github.com": {
    "org1": {
      "repositories": [...],
      "cached_at": "2025-11-16T12:00:00Z",
      "expires_at": "2025-11-16T13:00:00Z"
    }
  }
}
```

##### errors.go - エラー型

- `AuthenticationError` - 認証エラー
- `RateLimitError` - レート制限エラー
- `RepositoryNotFoundError` - リポジトリが見つからない
- `NetworkError` - ネットワークエラー

#### internal/workspace

ワークスペースの管理です。

- `Workspace`: ワークスペースを表す構造体
- `Create()`: ワークスペースを作成
- `Delete()`: ワークスペースを削除
- `List()`: ワークスペース一覧を取得
- `GenerateVSCodeWorkspace()`: .code-workspace ファイルを生成

## 実装の優先順位

### Phase 1: 基本機能

1. viper + cobra による設定ファイルの読み込み ✅
   - `internal/config` パッケージの実装
   - デフォルト値、環境変数、コマンドラインフラグの統合
   - `internal/cli` フラグ自動バインディングの実装
2. `config` コマンド（設定管理）
   - `config init` - 設定ファイルの初期化
   - `config show` - 現在の設定を表示
   - `config get` - 設定値の取得
   - `config set` - 設定値の設定
   - `config path` - 設定ファイルのパス表示
   - `config edit` - 設定ファイルの編集
   - `config validate` - 設定ファイルの検証
3. `repos list` コマンド ✅（骨組みのみ）
   - repos ディレクトリのスキャン
   - リポジトリ情報の表示
4. `workspace create` コマンド（基本版）✅（骨組みのみ）
   - 対話的なリポジトリ選択
   - Git worktree の作成
   - VSCode ワークスペースファイルの生成
5. `workspace list` コマンド ✅（骨組みのみ）
   - workspaces ディレクトリのスキャン
   - ワークスペース情報の表示
6. `workspace delete` コマンド
   - Git worktree の削除
   - ブランチの削除

### Phase 2: リモート連携

1. プロバイダーインターフェース設計
   - `internal/remote/provider.go` - 共通インターフェース
   - `internal/remote/errors.go` - エラー型定義
   - `internal/remote/cache.go` - キャッシュ機能
2. GitHub API 連携
   - `internal/remote/github.go` - GitHub API 実装
   - トークン認証
   - リポジトリ一覧取得
   - 組織一覧取得
   - レート制限対応
3. `remote list` コマンド
   - リモートリポジトリ一覧表示
   - フィルタリング、ソート
   - キャッシュ管理
4. `remote clone` コマンド
   - `internal/git/clone.go` - クローン操作
   - 単一リポジトリのクローン
   - 対話的な複数リポジトリクローン
5. GitLab API 連携（オプション）
   - `internal/remote/gitlab.go` - GitLab API 実装
6. `remote sync` コマンド（拡張）
   - リモートとローカルの差分検出
7. `remote orgs` コマンド（拡張）
   - 組織一覧表示

### Phase 3: 利便性向上

1. `workspace open` コマンド
2. `workspace status` コマンド
3. `repos sync` コマンド
4. テンプレート機能

### Phase 4: 高度な機能

1. `workspace merge` コマンド
2. スナップショット機能
3. クリーンアップ機能
4. インタラクティブ TUI

## 技術スタック

- **言語**: Go 1.21+
- **CLI フレームワーク**: [cobra](https://github.com/spf13/cobra) v1.10+
- **設定ファイル管理**: [viper](https://github.com/spf13/viper) v1.21+
- **Git 操作**: `os/exec` で git コマンドを実行（シンプルで信頼性が高い）
- **GitHub API**: [go-github](https://github.com/google/go-github) v57+
- **GitLab API**: [go-gitlab](https://github.com/xanzy/go-gitlab) v0.95+
- **OAuth2**: [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) - API 認証
- **対話的 UI**: 標準入力/出力（`bufio`パッケージ）
- **テーブル表示**: 標準の`text/tabwriter`パッケージ
- **JSON/YAML**: 標準の`encoding/json`パッケージ
- **TUI**: [bubbletea](https://github.com/charmbracelet/bubbletea)（Phase 4、オプション）

### 依存関係の方針

**Phase 1（完了）:**

- 外部依存は最小限（cobra, viper のみ）
- 標準ライブラリを優先使用

**Phase 2（リモート連携）:**

- GitHub/GitLab API ライブラリを追加
- OAuth2 ライブラリを追加

**Phase 3 以降:**

- 必要に応じて追加検討

## 実装例

### viper + cobra の統合

#### cmd/wsmg/root.go

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "wsmg",
	Short: "Workspace Manager - Git worktreeを活用したマルチリポジトリ開発支援ツール",
	Long: `wsmg (Workspace Manager) は、Git worktreeを活用してチケット単位での
マルチリポジトリ開発を効率化するコマンドラインツールです。`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// グローバルフラグの設定
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "設定ファイルのパス (デフォルト: ~/.config/wsmg/wsmg.json)")
	rootCmd.PersistentFlags().String("repos", "", "reposディレクトリのパス")
	rootCmd.PersistentFlags().String("workspaces", "", "workspacesディレクトリのパス")

	// フラグをviperにバインド
	viper.BindPFlag("repos", rootCmd.PersistentFlags().Lookup("repos"))
	viper.BindPFlag("workspaces", rootCmd.PersistentFlags().Lookup("workspaces"))
}

func initConfig() {
	if cfgFile != "" {
		// コマンドラインで指定された設定ファイルを使用
		viper.SetConfigFile(cfgFile)
	} else {
		// ホームディレクトリを取得
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// 設定ファイルのパスを設定
		configDir := filepath.Join(home, ".config", "wsmg")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("wsmg")
		viper.SetConfigType("json")
	}

	// 環境変数を読み込む（プレフィックス: WSMG_）
	viper.SetEnvPrefix("WSMG")
	viper.AutomaticEnv()

	// デフォルト値を設定
	home, _ := os.UserHomeDir()
	viper.SetDefault("repos", filepath.Join(home, "repos"))
	viper.SetDefault("workspaces", filepath.Join(home, "workspaces"))
	viper.SetDefault("env", map[string]string{})
	viper.SetDefault("remotes", []interface{}{})

	// 設定ファイルを読み込む
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 設定ファイルが見つからない場合はデフォルト値を使用
			fmt.Fprintln(os.Stderr, "設定ファイルが見つかりません。デフォルト値を使用します。")
		} else {
			// その他のエラー
			fmt.Fprintf(os.Stderr, "設定ファイルの読み込みエラー: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "設定ファイルを読み込みました: %s\n", viper.ConfigFileUsed())
	}
}
```

#### internal/config/config.go

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Repos      string     `mapstructure:"repos"`
	Workspaces string     `mapstructure:"workspaces"`
	Env        []EnvEntry `mapstructure:"env"`
	Remotes    []Remote   `mapstructure:"remotes"`
}

type EnvEntry struct {
	Key   string `mapstructure:"key"`
	Value string `mapstructure:"value"`
}

type Remote struct {
	Type          string   `mapstructure:"type"`
	URL           string   `mapstructure:"url"`
}

// GetConfig は現在の設定を返す
func GetConfig() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// チルダを展開
	cfg.Repos = expandPath(cfg.Repos)
	cfg.Workspaces = expandPath(cfg.Workspaces)

	return &cfg, nil
}

// GetReposDir はreposディレクトリのパスを返す
func GetReposDir() string {
	path := viper.GetString("repos")
	return expandPath(path)
}

// GetWorkspacesDir はworkspacesディレクトリのパスを返す
func GetWorkspacesDir() string {
	path := viper.GetString("workspaces")
	return expandPath(path)
}

// GetEnv は環境変数のマップを返す
func GetEnv() map[string]string {
	var envEntries []EnvEntry
	if err := viper.UnmarshalKey("env", &envEntries); err != nil {
		return map[string]string{}
	}

	// 配列をマップに変換
	envMap := make(map[string]string)
	for _, entry := range envEntries {
		envMap[entry.Key] = entry.Value
	}

	return envMap
}

// GetRemotes はリモート設定のスライスを返す
func GetRemotes() ([]Remote, error) {
	var remotes []Remote
	if err := viper.UnmarshalKey("remotes", &remotes); err != nil {
		return nil, err
	}
	return remotes, nil
}

// expandPath はチルダを含むパスを展開する
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// LoadEnv は設定された環境変数をプロセスに読み込む
func LoadEnv() error {
	envMap := GetEnv()
	for key, value := range envMap {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}
```

## 今後の検討事項

### Phase 2 関連

1. **API レート制限対応**:
   - キャッシュの最適化
   - レート制限に達した場合の待機処理
   - バッチ処理の実装
2. **認証トークンの管理**:
   - キーチェーン統合（macOS Keychain、Windows Credential Manager、Linux Secret Service）
   - トークンの有効性チェック
   - 複数アカウント対応
3. **プロバイダーの拡張**:
   - Bitbucket 対応
   - 自己ホスト GitLab サポート
   - Azure DevOps 対応

### 一般的な事項

4. **マルチプラットフォーム対応**: Windows、macOS、Linux での動作確認
5. **パフォーマンス**: 大量のリポジトリがある場合の最適化
   - 並列処理の導入
   - プログレスバーの表示
6. **テスト**: ユニットテスト、統合テストの整備
   - モックを使用した API テスト
   - E2E テスト
7. **CI/CD**: 自動ビルド、リリースの仕組み
   - GitHub Actions
   - クロスコンパイル
   - バイナリの配布（Homebrew、Scoop 等）
8. **ドキュメント**: 詳細なユーザーガイド、チュートリアル
   - スクリーンキャスト
   - ユースケース集

## Phase 2 実装ガイド

### 実装順序

#### Step 1: 基盤の構築（2-3 時間）

1. **依存関係の追加**

   ```bash
   go get github.com/google/go-github/v57@latest
   go get golang.org/x/oauth2@latest
   ```

2. **ファイル作成**

   - `internal/remote/provider.go` - インターフェース定義
   - `internal/remote/repository.go` - RemoteRepository 構造体
   - `internal/remote/errors.go` - エラー型
   - `internal/remote/cache.go` - キャッシュ機能

3. **設定の拡張**
   - `internal/config/config.go` にキャッシュ設定を追加

#### Step 2: GitHub 実装（3-4 時間）

1. **`internal/remote/github.go`**

   - GitHubProvider 構造体
   - トークン認証
   - ListRepositories 実装
   - ListOrganizations 実装
   - レート制限対応

2. **テスト**
   - 実際の GitHub アカウントでテスト
   - エラーハンドリングの確認

#### Step 3: remote list コマンド（2-3 時間）

1. **ファイル作成**

   - `cmd/wsmg/remote.go` - remote コマンド本体
   - `cmd/wsmg/remote_list.go` - remote list 実装

2. **機能実装**
   - プロバイダーの初期化
   - リポジトリ一覧の取得と表示
   - フィルタリング
   - キャッシュ統合

#### Step 4: remote clone コマンド（2-3 時間）

1. **ファイル作成**

   - `internal/git/clone.go` - クローン操作
   - `cmd/wsmg/remote_clone.go` - remote clone 実装

2. **機能実装**
   - リポジトリ指定の解析
   - クローン実行
   - 進捗表示

#### Step 5: 追加機能（オプション）

1. **GitLab 対応**（3-4 時間）

   ```bash
   go get github.com/xanzy/go-gitlab@latest
   ```

   - `internal/remote/gitlab.go` 実装

2. **remote clone --interactive**（1-2 時間）

   - 対話的選択機能

3. **remote orgs コマンド**（1 時間）
   - `cmd/wsmg/remote_orgs.go` 実装

### 認証の実装例

#### GitHub Personal Access Token

```go
// internal/remote/github.go
import (
    "context"
    "github.com/google/go-github/v57/github"
    "golang.org/x/oauth2"
)

type GitHubProvider struct {
    client *github.Client
    ctx    context.Context
}

func NewGitHubProvider(token string) (*GitHubProvider, error) {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )
    tc := oauth2.NewClient(ctx, ts)
    client := github.NewClient(tc)

    return &GitHubProvider{
        client: client,
        ctx:    ctx,
    }, nil
}
```

### レート制限の対応例

```go
func (p *GitHubProvider) checkRateLimit() error {
    rate, _, err := p.client.RateLimits(p.ctx)
    if err != nil {
        return err
    }

    if rate.Core.Remaining < 10 {
        return &RateLimitError{
            Provider:  "github",
            Remaining: rate.Core.Remaining,
            ResetTime: rate.Core.Reset.Time,
        }
    }

    return nil
}
```

### キャッシュの実装例

```go
// internal/remote/cache.go
type CacheEntry struct {
    Repositories []*RemoteRepository `json:"repositories"`
    CachedAt     time.Time           `json:"cached_at"`
    ExpiresAt    time.Time           `json:"expires_at"`
}

func (c *Cache) GetRepositories(provider, org string) ([]*RemoteRepository, error) {
    key := fmt.Sprintf("%s:%s", provider, org)
    entry, err := c.get(key)
    if err != nil {
        return nil, err
    }

    if time.Now().After(entry.ExpiresAt) {
        return nil, ErrCacheExpired
    }

    return entry.Repositories, nil
}
```

### エラーメッセージの例

```go
// 認証エラー
if err := provider.Authenticate(); err != nil {
    fmt.Fprintf(os.Stderr, "認証エラー: %v\n", err)
    fmt.Fprintf(os.Stderr, "\nGitHub Personal Access Tokenを設定してください:\n")
    fmt.Fprintf(os.Stderr, "  1. https://github.com/settings/tokens で新しいトークンを作成\n")
    fmt.Fprintf(os.Stderr, "  2. 'repo' スコープを選択\n")
    fmt.Fprintf(os.Stderr, "  3. wsmg config set env.GITHUB_TOKEN <token> を実行\n")
    return err
}

// レート制限エラー
if rateLimitErr, ok := err.(*RateLimitError); ok {
    fmt.Fprintf(os.Stderr, "APIレート制限に達しました。\n")
    fmt.Fprintf(os.Stderr, "リセット時刻: %s\n", rateLimitErr.ResetTime.Format("15:04:05"))
    duration := time.Until(rateLimitErr.ResetTime)
    fmt.Fprintf(os.Stderr, "待ち時間: %s\n", duration.Round(time.Second))
    fmt.Fprintf(os.Stderr, "\nキャッシュを使用するには --no-cache を外してください。\n")
    return err
}
```

### テストの方針

#### ユニットテスト

```go
// internal/remote/provider_test.go
func TestGitHubProvider_ListRepositories(t *testing.T) {
    // モックサーバーを使用
    // または実際のGitHub APIを使用（環境変数でトークンを渡す）
}
```

#### 統合テスト

```bash
# 実際のGitHub APIを使用したテスト
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
go test -v ./internal/remote/... -integration
```

### デバッグのヒント

1. **API リクエストのログ**

   ```go
   // 環境変数 WSMG_DEBUG=1 でデバッグモード
   if os.Getenv("WSMG_DEBUG") == "1" {
       fmt.Fprintf(os.Stderr, "API Request: GET %s\n", url)
   }
   ```

2. **レート制限の確認**

   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" \
        https://api.github.com/rate_limit
   ```

3. **キャッシュの確認**
   ```bash
   cat ~/.cache/wsmg/remote-cache.json | jq .
   ```
