package scaffold

import (
	"os"
	"path/filepath"

	"github.com/Lenstack/opencode-scaffold/internal/output"
)

type Writer struct {
	Root     string
	Force    bool
	DryRun   bool
	Renderer output.Renderer
	Result   *Result
}

func (w *Writer) Dir(rel string) {
	if w.DryRun {
		return
	}
	full := filepath.Join(w.Root, rel)
	if err := os.MkdirAll(full, 0755); err != nil {
		w.Result.AddError(err)
	}
}

func (w *Writer) Dirs(paths []string) {
	for _, p := range paths {
		w.Dir(p)
	}
}

func (w *Writer) File(rel, content string, mode ...os.FileMode) {
	m := os.FileMode(0644)
	if len(mode) > 0 {
		m = mode[0]
	}

	if w.DryRun {
		w.Result.AddCreated(rel)
		if w.Renderer != nil {
			w.Renderer.FileCreated(rel)
		}
		return
	}

	full := filepath.Join(w.Root, rel)
	if !w.Force {
		if _, err := os.Stat(full); err == nil {
			w.Result.AddSkipped(rel)
			if w.Renderer != nil {
				w.Renderer.FileSkipped(rel, "already exists")
			}
			return
		}
	}

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		w.Result.AddError(err)
		if w.Renderer != nil {
			w.Renderer.Error(err)
		}
		return
	}
	if err := os.WriteFile(full, []byte(content), m); err != nil {
		w.Result.AddError(err)
		if w.Renderer != nil {
			w.Renderer.Error(err)
		}
		return
	}
	w.Result.AddCreated(rel)
	if w.Renderer != nil {
		w.Renderer.FileCreated(rel)
	}
}

func (w *Writer) Chmod(rel string, mode os.FileMode) {
	if w.DryRun {
		return
	}
	full := filepath.Join(w.Root, rel)
	_ = os.Chmod(full, mode)
}

func Exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}
