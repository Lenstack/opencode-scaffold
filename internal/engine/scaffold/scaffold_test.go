package scaffold_test

import (
	"os"
	"testing"

	"github.com/Lenstack/opencode-scaffold/internal/detector"
	"github.com/Lenstack/opencode-scaffold/internal/engine/scaffold"
	"github.com/Lenstack/opencode-scaffold/internal/testutil"
)

func TestScaffoldGeneric(t *testing.T) {
	dir := t.TempDir()

	opts := scaffold.Options{
		Root:  dir,
		Stack: &detector.Stack{ID: "generic", Name: "Generic", Backend: "generic"},
		Force: true,
	}

	result := scaffold.Run(opts)
	if len(result.Errors) > 0 {
		t.Fatalf("scaffold errors: %v", result.Errors)
	}

	testutil.AssertGolden(t, dir, "generic")
}

func goldenFilter(path string) bool {
	return path != ".opencode/memory/semantic/index.json"
}

func TestScaffoldGoEncore(t *testing.T) {
	dir := t.TempDir()

	opts := scaffold.Options{
		Root:  dir,
		Stack: &detector.Stack{ID: "go-encore", Name: "Go + Encore", Backend: "go", Framework: "encore", GoModule: "example.com/myapp"},
		Force: true,
	}

	result := scaffold.Run(opts)
	if len(result.Errors) > 0 {
		t.Fatalf("scaffold errors: %v", result.Errors)
	}

	testutil.AssertGolden(t, dir, "go-encore")
}

func TestScaffoldPythonFastAPI(t *testing.T) {
	dir := t.TempDir()

	opts := scaffold.Options{
		Root:  dir,
		Stack: &detector.Stack{ID: "python-fastapi", Name: "Python + FastAPI", Backend: "python", Framework: "fastapi"},
		Force: true,
	}

	result := scaffold.Run(opts)
	if len(result.Errors) > 0 {
		t.Fatalf("scaffold errors: %v", result.Errors)
	}

	testutil.AssertGolden(t, dir, "python-fastapi")
}

func TestScaffoldIdempotent(t *testing.T) {
	dir := t.TempDir()

	opts := scaffold.Options{
		Root:  dir,
		Stack: &detector.Stack{ID: "generic", Name: "Generic", Backend: "generic"},
		Force: true,
	}

	r1 := scaffold.Run(opts)
	if len(r1.Errors) > 0 {
		t.Fatalf("first run errors: %v", r1.Errors)
	}

	opts2 := scaffold.Options{
		Root:  dir,
		Stack: &detector.Stack{ID: "generic", Name: "Generic", Backend: "generic"},
		Force: false,
	}
	result2 := scaffold.Run(opts2)
	if len(result2.Errors) > 0 {
		t.Fatalf("second run errors: %v", result2.Errors)
	}

	if len(result2.Skipped) == 0 {
		t.Fatal("second run should have skipped files")
	}

	if len(result2.Created) > 0 {
		t.Fatalf("second run should not create new files (got %d)", len(result2.Created))
	}
}

func TestScaffoldDryRun(t *testing.T) {
	dir := t.TempDir()

	opts := scaffold.Options{
		Root:   dir,
		Stack:  &detector.Stack{ID: "generic", Name: "Generic", Backend: "generic"},
		DryRun: true,
	}

	result := scaffold.Run(opts)
	if len(result.Errors) > 0 {
		t.Fatalf("dry run errors: %v", result.Errors)
	}

	if len(result.Created) == 0 {
		t.Fatal("dry run should report files that would be created")
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) > 0 {
		t.Fatal("dry run should not create any files")
	}
}
