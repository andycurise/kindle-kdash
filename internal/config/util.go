package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic 在目标目录写入临时文件，关闭后原子替换目标文件。
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录 %s: %w", dir, err)
	}
	file, err := os.CreateTemp(dir, ".kdash-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	name := file.Name()
	defer os.Remove(name)

	effectivePerm := perm
	if info, err := os.Stat(path); err == nil {
		// 保留 owner 权限，group/other 只能保持或收紧，不能放宽。
		effectivePerm = (info.Mode().Perm() & 0700) | (info.Mode().Perm() & perm & 0077)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取目标文件权限 %s: %w", path, err)
	}
	if err := file.Chmod(effectivePerm); err != nil {
		_ = file.Close()
		return fmt.Errorf("设置临时文件权限: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("写入临时文件: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("同步临时文件: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭临时文件: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("替换 %s: %w", path, err)
	}
	if runtime.GOOS != "windows" {
		directory, err := os.Open(dir)
		if err != nil {
			return fmt.Errorf("打开目录 %s 以同步: %w", dir, err)
		}
		defer directory.Close()
		if err := directory.Sync(); err != nil {
			return fmt.Errorf("同步目录 %s: %w", dir, err)
		}
	}
	return nil
}
