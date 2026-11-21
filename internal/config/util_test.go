package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "weather.json")
	if err := WriteFileAtomic(path, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "second" {
		t.Fatalf("原子替换结果=%q, err=%v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".kdash-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("存在未清理临时文件: %v, err=%v", matches, err)
	}
}
