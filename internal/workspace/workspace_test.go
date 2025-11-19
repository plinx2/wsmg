package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		name      string
		relPath   string
		patterns  []string
		wantMatch bool
	}{
		{
			name:      ".env で始まるファイル（ルート）",
			relPath:   ".env",
			patterns:  []string{".env*"},
			wantMatch: true,
		},
		{
			name:      ".env.local で始まるファイル（ルート）",
			relPath:   ".env.local",
			patterns:  []string{".env*"},
			wantMatch: true,
		},
		{
			name:      ".envrc で始まるファイル（ルート）",
			relPath:   ".envrc",
			patterns:  []string{".env*"},
			wantMatch: true,
		},
		{
			name:      ".tool-versions の完全一致",
			relPath:   ".tool-versions",
			patterns:  []string{".tool-versions"},
			wantMatch: true,
		},
		{
			name:      "マッチしないファイル",
			relPath:   "README.md",
			patterns:  []string{".env*", ".tool-versions"},
			wantMatch: false,
		},
		{
			name:      "複数パターンでマッチ",
			relPath:   ".nvmrc",
			patterns:  []string{".env*", ".nvmrc", ".ruby-version"},
			wantMatch: true,
		},
		{
			name:      "環境ファイルではないもの",
			relPath:   "environment.txt",
			patterns:  []string{".env*"},
			wantMatch: false,
		},
		{
			name:      ".*-version パターンマッチ (.node-version)",
			relPath:   ".node-version",
			patterns:  []string{".*-version"},
			wantMatch: true,
		},
		{
			name:      ".*-version パターンマッチ (.ruby-version)",
			relPath:   ".ruby-version",
			patterns:  []string{".*-version"},
			wantMatch: true,
		},
		{
			name:      ".*-version パターンマッチ (.python-version)",
			relPath:   ".python-version",
			patterns:  []string{".*-version"},
			wantMatch: true,
		},
		{
			name:      "go.work の完全一致",
			relPath:   "go.work",
			patterns:  []string{"go.work"},
			wantMatch: true,
		},
		{
			name:      ".npmrc の完全一致",
			relPath:   ".npmrc",
			patterns:  []string{".npmrc"},
			wantMatch: true,
		},
		{
			name:      ".vscode ディレクトリ自体",
			relPath:   ".vscode",
			patterns:  []string{".vscode/*"},
			wantMatch: true,
		},
		{
			name:      ".vscode/settings.json ファイル",
			relPath:   ".vscode/settings.json",
			patterns:  []string{".vscode/*"},
			wantMatch: true,
		},
		{
			name:      ".vscode/launch.json ファイル",
			relPath:   ".vscode/launch.json",
			patterns:  []string{".vscode/*"},
			wantMatch: true,
		},
		{
			name:      ".idea ディレクトリはマッチしない",
			relPath:   ".idea",
			patterns:  []string{".vscode/*"},
			wantMatch: false,
		},
		{
			name:      "サブディレクトリ内の .env",
			relPath:   "config/.env",
			patterns:  []string{".env*"},
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch := matchesAnyPattern(tt.relPath, tt.patterns)
			if gotMatch != tt.wantMatch {
				t.Errorf("matchesAnyPattern(%q, %v) = %v, want %v", tt.relPath, tt.patterns, gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestCopyEnvironmentFiles(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")

	// ソースディレクトリを作成
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// ターゲットディレクトリを作成
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// テストファイルを作成
	testFiles := map[string]string{
		".env":            "DB_HOST=localhost",
		".env.local":      "DB_HOST=127.0.0.1",
		".envrc":          "export PATH=$PATH:/usr/local/bin",
		".tool-versions":  "nodejs 18.0.0",
		".nvmrc":          "18.0.0",
		".node-version":   "18.0.0",
		".ruby-version":   "3.2.0",
		".python-version": "3.11.0",
		".npmrc":          "registry=https://registry.npmjs.org/",
		"go.work":         "go 1.21",
		"README.md":       "# Test",
		"package.json":    "{}",
	}

	for name, content := range testFiles {
		path := filepath.Join(sourceDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// .vscode ディレクトリとファイルを作成
	vscodeDir := filepath.Join(sourceDir, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		t.Fatal(err)
	}
	vscodeFiles := map[string]string{
		"settings.json": `{"editor.fontSize": 14}`,
		"launch.json":   `{"version": "0.2.0"}`,
	}
	for name, content := range vscodeFiles {
		path := filepath.Join(vscodeDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// パターン（更新されたデフォルトパターン）
	patterns := []string{
		".env*",
		".vscode/*",
		".tool-versions",
		".*-version",
		".nvmrc",
		".npmrc",
		"go.work",
	}

	// CopyEnvironmentFiles を実行
	copiedFiles, err := CopyEnvironmentFiles(sourceDir, targetDir, patterns)
	if err != nil {
		t.Fatalf("CopyEnvironmentFiles() error = %v", err)
	}

	// コピーされたファイルを確認（最低限の数）
	if len(copiedFiles) < 10 {
		t.Errorf("Copied files count = %d, want at least 10 (files: %v)", len(copiedFiles), copiedFiles)
	}

	// コピーされたファイルを確認
	expectedFiles := []string{
		".env",
		".env.local",
		".envrc",
		".tool-versions",
		".nvmrc",
		".node-version",
		".ruby-version",
		".python-version",
		".npmrc",
		"go.work",
	}
	for _, fileName := range expectedFiles {
		// ファイルが存在するか確認
		targetPath := filepath.Join(targetDir, fileName)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found in target directory", fileName)
			continue
		}

		// ファイル内容を確認
		content, err := os.ReadFile(targetPath)
		if err != nil {
			t.Errorf("Failed to read copied file %s: %v", fileName, err)
			continue
		}

		expectedContent := testFiles[fileName]
		if string(content) != expectedContent {
			t.Errorf("File %s content = %s, want %s", fileName, string(content), expectedContent)
		}
	}

	// .vscode ディレクトリが存在するか確認
	targetVscodeDir := filepath.Join(targetDir, ".vscode")
	if _, err := os.Stat(targetVscodeDir); os.IsNotExist(err) {
		t.Errorf("Expected directory .vscode not found in target directory")
	} else {
		// .vscode 内のファイルを確認
		for fileName, expectedContent := range vscodeFiles {
			targetPath := filepath.Join(targetVscodeDir, fileName)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				t.Errorf("Expected file .vscode/%s not found", fileName)
				continue
			}

			content, err := os.ReadFile(targetPath)
			if err != nil {
				t.Errorf("Failed to read copied file .vscode/%s: %v", fileName, err)
				continue
			}

			if string(content) != expectedContent {
				t.Errorf("File .vscode/%s content = %s, want %s", fileName, string(content), expectedContent)
			}
		}
	}

	// コピーされてはいけないファイルを確認
	notCopiedFiles := []string{"README.md", "package.json"}
	for _, fileName := range notCopiedFiles {
		targetPath := filepath.Join(targetDir, fileName)
		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Errorf("File %s should not have been copied", fileName)
		}
	}
}

func TestCopyFile(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()

	// ソースファイルを作成
	sourceFile := filepath.Join(tmpDir, "source.txt")
	content := "test content"
	if err := os.WriteFile(sourceFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// ファイルをコピー
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := copyFile(sourceFile, targetFile); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// コピーされたファイルを確認
	copiedContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != content {
		t.Errorf("Copied content = %s, want %s", string(copiedContent), content)
	}

	// パーミッションを確認
	sourceInfo, _ := os.Stat(sourceFile)
	targetInfo, _ := os.Stat(targetFile)

	if sourceInfo.Mode() != targetInfo.Mode() {
		t.Errorf("Copied file permissions = %v, want %v", targetInfo.Mode(), sourceInfo.Mode())
	}
}
