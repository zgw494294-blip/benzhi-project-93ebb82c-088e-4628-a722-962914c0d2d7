package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func writeAtomicJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("编码快照: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时快照: %w", err)
	}
	tmpName := tmp.Name()
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时快照: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("同步临时快照: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时快照: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("原子替换快照: %w", err)
	}
	remove = false
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("同步快照目录: %w", err)
	}
	return nil
}
