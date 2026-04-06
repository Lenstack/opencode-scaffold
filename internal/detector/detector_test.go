package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStack(t *testing.T) {
	stack := &Stack{
		ID:          "go",
		Name:        "Go",
		Backend:     "go",
		Frontend:    "",
		Framework:   "stdlib",
		HasDB:       true,
		HasPubSub:   false,
		HasAuth:     true,
		HasDocker:   true,
		HasCI:       true,
		GoModule:    "example.com/app",
		NodePkgName: "",
		Confidence:  0.8,
	}

	if stack.ID != "go" {
		t.Errorf("expected ID 'go', got %q", stack.ID)
	}
	if !stack.HasDB {
		t.Error("expected HasDB true")
	}
	if stack.Confidence != 0.8 {
		t.Errorf("expected Confidence 0.8, got %f", stack.Confidence)
	}
}

func TestStack_Fields(t *testing.T) {
	stack := &Stack{}

	// Test zero values
	if stack.ID != "" {
		t.Errorf("expected empty ID, got %q", stack.ID)
	}
	if stack.Confidence != 0 {
		t.Errorf("expected Confidence 0, got %f", stack.Confidence)
	}
}

func TestDetect(t *testing.T) {
	dir := t.TempDir()

	stack := Detect(dir)

	if stack == nil {
		t.Fatal("expected stack to be returned")
	}
	// Empty directory should return generic
	if stack.ID != "generic" {
		t.Errorf("expected ID 'generic', got %q", stack.ID)
	}
}

func TestDetect_Go(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\ngo 1.21\n"), 0644)

	stack := Detect(dir)

	if stack.Backend != "go" {
		t.Errorf("expected Backend 'go', got %q", stack.Backend)
	}
	if stack.ID != "go-stdlib" {
		t.Errorf("expected ID 'go-stdlib', got %q", stack.ID)
	}
	if stack.GoModule != "example.com/test" {
		t.Errorf("expected GoModule 'example.com/test', got %q", stack.GoModule)
	}
}

func TestDetect_GoEncore(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\nrequire encore.dev v1.0.0\n"), 0644)

	stack := Detect(dir)

	if stack.Backend != "go" {
		t.Errorf("expected Backend 'go', got %q", stack.Backend)
	}
	if stack.Framework != "encore" {
		t.Errorf("expected Framework 'encore', got %q", stack.Framework)
	}
	if stack.ID != "go-encore" {
		t.Errorf("expected ID 'go-encore', got %q", stack.ID)
	}
}

func TestDetect_Node(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test-app", "dependencies": {}}`), 0644)

	stack := Detect(dir)

	if stack.Backend != "node" {
		t.Errorf("expected Backend 'node', got %q", stack.Backend)
	}
}

func TestDetect_NodeNextjs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test-app", "dependencies": {"next": "^14.0.0"}}`), 0644)

	stack := Detect(dir)

	if stack.Backend != "node" {
		t.Errorf("expected Backend 'node', got %q", stack.Backend)
	}
	if stack.Frontend != "nextjs" {
		t.Errorf("expected Frontend 'nextjs', got %q", stack.Frontend)
	}
}

func TestDetect_Python(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "test"
`), 0644)

	stack := Detect(dir)

	if stack.Backend != "python" {
		t.Errorf("expected Backend 'python', got %q", stack.Backend)
	}
}

func TestDetect_PythonFastAPI(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "test"
dependencies = ["fastapi"]
`), 0644)

	stack := Detect(dir)

	if stack.Backend != "python" {
		t.Errorf("expected Backend 'python', got %q", stack.Backend)
	}
	if stack.Framework != "fastapi" {
		t.Errorf("expected Framework 'fastapi', got %q", stack.Framework)
	}
}

func TestDetect_Rust(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "test"
version = "0.1.0"
`), 0644)

	stack := Detect(dir)

	if stack.Backend != "rust" {
		t.Errorf("expected Backend 'rust', got %q", stack.Backend)
	}
}

func TestDetect_RustAxum(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "test"
version = "0.1.0"

[dependencies]
axum = "0.7"
`), 0644)

	stack := Detect(dir)

	if stack.Backend != "rust" {
		t.Errorf("expected Backend 'rust', got %q", stack.Backend)
	}
	if stack.Framework != "axum" {
		t.Errorf("expected Framework 'axum', got %q", stack.Framework)
	}
}

func TestDetect_Generic(t *testing.T) {
	dir := t.TempDir()

	stack := Detect(dir)

	if stack.ID != "generic" {
		t.Errorf("expected ID 'generic', got %q", stack.ID)
	}
	if stack.Backend != "generic" {
		t.Errorf("expected Backend 'generic', got %q", stack.Backend)
	}
	if stack.Confidence != 0.1 {
		t.Errorf("expected Confidence 0.1, got %f", stack.Confidence)
	}
}

func TestDetect_HasCI(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)

	stack := Detect(dir)

	if !stack.HasCI {
		t.Error("expected HasCI true")
	}
}

func TestDetect_HasDocker(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM golang:1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)

	stack := Detect(dir)

	if !stack.HasDocker {
		t.Error("expected HasDocker true")
	}
}

func TestApplySignal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)

	stack := &Stack{}
	sig := signal{file: "go.mod", score: 0.3, field: "backend", value: "go"}

	applySignal(stack, sig, dir)

	if stack.Backend != "go" {
		t.Errorf("expected Backend 'go', got %q", stack.Backend)
	}
}

func TestReadModuleName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\ngo 1.21\n"), 0644)

	module := readModuleName(dir)
	if module != "example.com/test" {
		t.Errorf("expected 'example.com/test', got %q", module)
	}
}

func TestReadModuleNameEmpty(t *testing.T) {
	dir := t.TempDir()

	module := readModuleName(dir)
	if module != "" {
		t.Errorf("expected empty module, got %q", module)
	}
}

func TestReadPackageName(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test-app", "version": "1.0.0"}`), 0644)

	name := readPackageName(filepath.Join(dir, "package.json"))
	if name != "test-app" {
		t.Errorf("expected 'test-app', got %q", name)
	}
}

func TestReadPackageNameEmpty(t *testing.T) {
	dir := t.TempDir()

	name := readPackageName(filepath.Join(dir, "package.json"))
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test"), 0644)

	if !exists(dir, "test.txt") {
		t.Error("expected file to exist")
	}
	if exists(dir, "nonexistent.txt") {
		t.Error("expected file to not exist")
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	content := readFile(filepath.Join(dir, "test.txt"))
	if content != "hello world" {
		t.Errorf("expected 'hello world', got %q", content)
	}
}

func TestReadFileNotFound(t *testing.T) {
	dir := t.TempDir()

	content := readFile(filepath.Join(dir, "nonexistent.txt"))
	if content != "" {
		t.Errorf("expected empty content, got %q", content)
	}
}

func TestSignals(t *testing.T) {
	// Test that signals are defined
	if len(signals) == 0 {
		t.Error("expected signals to be defined")
	}

	// Check for expected signals
	foundGo := false
	foundNode := false
	for _, sig := range signals {
		if sig.file == "go.mod" && sig.field == "backend" {
			foundGo = true
		}
		if sig.file == "package.json" && sig.field == "backend" {
			foundNode = true
		}
	}

	if !foundGo {
		t.Error("expected go.mod signal")
	}
	if !foundNode {
		t.Error("expected package.json signal")
	}
}

func TestDetect_GoFiber(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\nrequire github.com/gofiber/fiber/v2 v2.0.0\n"), 0644)

	stack := Detect(dir)

	if stack.Framework != "fiber" {
		t.Errorf("expected Framework 'fiber', got %q", stack.Framework)
	}
	if stack.ID != "go-fiber" {
		t.Errorf("expected ID 'go-fiber', got %q", stack.ID)
	}
}

func TestDetect_GoGin(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\nrequire github.com/gin-gonic/gin v1.0.0\n"), 0644)

	stack := Detect(dir)

	if stack.Framework != "gin" {
		t.Errorf("expected Framework 'gin', got %q", stack.Framework)
	}
	if stack.ID != "go-gin" {
		t.Errorf("expected ID 'go-gin', got %q", stack.ID)
	}
}

func TestDetect_GoChi(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\nrequire github.com/go-chi/chi/v5 v5.0.0\n"), 0644)

	stack := Detect(dir)

	if stack.Framework != "chi" {
		t.Errorf("expected Framework 'chi', got %q", stack.Framework)
	}
	if stack.ID != "go-chi" {
		t.Errorf("expected ID 'go-chi', got %q", stack.ID)
	}
}

func TestDetect_PythonDjango(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "test"
dependencies = ["django"]
`), 0644)

	stack := Detect(dir)

	if stack.Framework != "django" {
		t.Errorf("expected Framework 'django', got %q", stack.Framework)
	}
}

func TestDetect_PythonFlask(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "test"
dependencies = ["flask"]
`), 0644)

	stack := Detect(dir)

	if stack.Framework != "flask" {
		t.Errorf("expected Framework 'flask', got %q", stack.Framework)
	}
}

func TestDetect_DockerCompose(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)

	stack := Detect(dir)

	if !stack.HasDocker {
		t.Error("expected HasDocker true")
	}
}

func TestDetect_GitLabCI(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitlab-ci.yml"), []byte("stages:\n  - test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)

	stack := Detect(dir)

	if !stack.HasCI {
		t.Error("expected HasCI true")
	}
}

func TestDetect_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.0.0\n"), 0644)

	stack := Detect(dir)

	if stack.Backend != "python" {
		t.Errorf("expected Backend 'python', got %q", stack.Backend)
	}
}

func TestDetect_NodeReact(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test-app", "dependencies": {"react": "^18.0.0"}}`), 0644)

	stack := Detect(dir)

	if stack.Frontend != "react" {
		t.Errorf("expected Frontend 'react', got %q", stack.Frontend)
	}
}

func TestDetect_NodeVue(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test-app", "dependencies": {"vue": "^3.0.0"}}`), 0644)

	stack := Detect(dir)

	if stack.Frontend != "vue" {
		t.Errorf("expected Frontend 'vue', got %q", stack.Frontend)
	}
}
