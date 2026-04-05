package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "update golden test files")

var volatileFiles = []string{
	".opencode/memory/semantic/index.json",
}

func isVolatile(path string) bool {
	for _, v := range volatileFiles {
		if strings.HasSuffix(path, v) {
			return true
		}
	}
	return false
}

func AssertGolden(t *testing.T, dir string, name string) {
	t.Helper()

	goldenDir := filepath.Join("testdata", name)

	if *updateGolden {
		_ = os.RemoveAll(goldenDir)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(dir, path)
			if isVolatile(rel) {
				return nil
			}
			dest := filepath.Join(goldenDir, rel)
			_ = os.MkdirAll(filepath.Dir(dest), 0755)
			data, _ := os.ReadFile(path)
			return os.WriteFile(dest, data, 0644)
		})
		if err != nil {
			t.Fatalf("failed to update golden files: %v", err)
		}
		t.Logf("golden files updated: %s", name)
		return
	}

	err := filepath.Walk(goldenDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(goldenDir, path)
		if isVolatile(rel) {
			return nil
		}
		actualPath := filepath.Join(dir, rel)
		expected, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("golden file missing: %s", rel)
			return nil
		}
		actual, err := os.ReadFile(actualPath)
		if err != nil {
			t.Errorf("actual file missing: %s", rel)
			return nil
		}
		if string(expected) != string(actual) {
			t.Errorf("golden mismatch: %s\n--- expected\n+++ actual\n", rel)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("golden walk failed: %v", err)
	}
}
