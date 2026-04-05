package discovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/db"
)

type Engine struct {
	root string
	db   *db.Engine
}

func New(root string, db *db.Engine) *Engine {
	return &Engine{root: root, db: db}
}

type StackFingerprint struct {
	Stack      string
	Frameworks string
	HasDB      bool
	HasPubSub  bool
	HasAuth    bool
	HasDocker  bool
	HasCI      bool
	GoModule   string
	NodePkg    string
}

func (e *Engine) Run(full bool) (*db.ProjectMap, error) {
	start := time.Now()

	if !full {
		var pm db.ProjectMap
		if err := e.db.Get(db.NSDiscovery, "project_map", &pm); err == nil && pm.Checksum != "" {
			current := e.computeGlobalChecksum()
			if current == pm.Checksum {
				return &pm, nil
			}
		}
	}

	pm := &db.ProjectMap{
		Version:   "v2",
		ScannedAt: time.Now().UTC().Format(time.RFC3339),
	}

	fingerprint := e.detectStack()
	pm.Stack = fingerprint.Stack
	pm.Frameworks = fingerprint.Frameworks

	files, err := e.scanFiles()
	if err != nil {
		return nil, err
	}
	pm.FilesCount = len(files)
	pm.Checksum = e.computeGlobalChecksum()

	if fingerprint.HasDB {
		pm.DBTables = e.extractDBTables()
	}
	pm.APIRoutes = e.extractAPIRoutes()
	pm.Patterns = e.detectPatterns()
	pm.Dependencies = e.extractDependencies(fingerprint)

	if err := e.db.Put(db.NSDiscovery, "project_map", pm); err != nil {
		return nil, err
	}

	for _, f := range files {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(f.Path)))[:12]
		e.db.Put(db.NSDiscovery, "file:"+hash, f)
	}

	for _, route := range pm.APIRoutes {
		e.db.Put(db.NSDiscovery, "api_route:"+route.Endpoint, route)
	}

	for _, table := range pm.DBTables {
		e.db.Put(db.NSDiscovery, "db_table:"+table.Name, table)
	}

	elapsed := time.Since(start)
	fmt.Printf("✅ discovery: %d files, checksum: %s…, stack: %s (%v)\n",
		pm.FilesCount, pm.Checksum[:8], pm.Stack, elapsed.Round(time.Millisecond))

	return pm, nil
}

func (e *Engine) detectStack() StackFingerprint {
	fp := StackFingerprint{}

	if exists(e.root, "go.mod") {
		fp.Stack = "go"
		fp.GoModule = readModule(e.root)
		if contains(e.root, "go.mod", "encore.dev") {
			fp.Stack = "go-encore"
			fp.Frameworks = "encore"
		} else if contains(e.root, "go.mod", "gofiber/fiber") {
			fp.Frameworks = "fiber"
		} else if contains(e.root, "go.mod", "gin-gonic/gin") {
			fp.Frameworks = "gin"
		} else if contains(e.root, "go.mod", "go-chi/chi") {
			fp.Frameworks = "chi"
		}
	}

	if exists(e.root, "package.json") {
		if fp.Stack == "" {
			fp.Stack = "node"
		}
		pkg := readFile(e.root, "package.json")
		if strings.Contains(pkg, `"next"`) {
			fp.Frameworks += ",nextjs"
		}
		if strings.Contains(pkg, `"react"`) {
			fp.Frameworks += ",react"
		}
		fp.NodePkg = extractName(pkg)
	}

	if exists(e.root, "pyproject.toml") || exists(e.root, "requirements.txt") {
		if fp.Stack == "" {
			fp.Stack = "python"
		}
		if contains(e.root, "pyproject.toml", "fastapi") {
			fp.Frameworks += ",fastapi"
		} else if contains(e.root, "pyproject.toml", "django") {
			fp.Frameworks += ",django"
		}
	}

	if exists(e.root, "Cargo.toml") {
		if fp.Stack == "" {
			fp.Stack = "rust"
		}
		if contains(e.root, "Cargo.toml", "axum") {
			fp.Frameworks += ",axum"
		}
	}

	fp.HasDB = exists(e.root, "migrations") || contains(e.root, "go.mod", "gorm.io") || contains(e.root, "go.mod", "pgx")
	fp.HasPubSub = contains(e.root, "go.mod", "pubsub") || containsGlob(e.root, "encore/**/*.go", "pubsub.NewTopic")
	fp.HasAuth = containsGlob(e.root, "encore/**/*.go", "//encore:api auth")
	fp.HasDocker = exists(e.root, "docker-compose.yml") || exists(e.root, "Dockerfile")
	fp.HasCI = exists(e.root, ".github/workflows") || exists(e.root, ".gitlab-ci.yml")

	return fp
}

func (e *Engine) scanFiles() ([]db.FileEntry, error) {
	var files []db.FileEntry
	var mu sync.Mutex
	var wg sync.WaitGroup

	extensions := []string{"*.go", "*.ts", "*.tsx", "*.py", "*.rs", "*.sql"}
	for _, ext := range extensions {
		matches, _ := filepath.Glob(filepath.Join(e.root, "**/"+ext))
		for _, m := range matches {
			if isIgnored(m) {
				continue
			}
			wg.Add(1)
			go func(match string) {
				defer wg.Done()
				info, err := os.Stat(match)
				if err != nil {
					return
				}
				rel, _ := filepath.Rel(e.root, match)
				entry := db.FileEntry{
					Path:     rel,
					Type:     filepath.Ext(rel),
					Size:     info.Size(),
					Modified: info.ModTime().Unix(),
					Imports:  extractImports(match),
				}
				mu.Lock()
				files = append(files, entry)
				mu.Unlock()
			}(m)
		}
	}

	wg.Wait()
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (e *Engine) extractDBTables() []db.DBTable {
	var tables []db.DBTable
	matches, _ := filepath.Glob(filepath.Join(e.root, "**/*.sql"))
	for _, m := range matches {
		content := readFile(e.root, m)
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "CREATE TABLE") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					tables = append(tables, db.DBTable{Name: strings.TrimSuffix(parts[2], "(")})
				}
			}
		}
	}
	return tables
}

func (e *Engine) extractAPIRoutes() []db.APIRoute {
	var routes []db.APIRoute
	matches, _ := filepath.Glob(filepath.Join(e.root, "encore/**/*.go"))
	for _, m := range matches {
		content := readFile(e.root, m)
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(line, "//encore:api") {
				route := db.APIRoute{Handler: m}
				if strings.Contains(line, "public") {
					route.AuthLevel = "public"
				} else if strings.Contains(line, "auth") {
					route.AuthLevel = "auth"
				} else {
					route.AuthLevel = "private"
				}
				if idx := strings.Index(line, "path="); idx >= 0 {
					path := strings.SplitN(line[idx+5:], " ", 2)[0]
					route.Endpoint = strings.Trim(path, `"`)
				}
				routes = append(routes, route)
			}
		}
	}
	return routes
}

func (e *Engine) detectPatterns() map[string]string {
	patterns := map[string]string{
		"pagination": "unknown",
		"auth":       "unknown",
		"forms":      "unknown",
	}

	if containsGlob(e.root, "encore/**/*.go", "keyset") || containsGlob(e.root, "encore/**/*.go", "cursor") {
		patterns["pagination"] = "keyset-cursor"
	}
	if containsGlob(e.root, "encore/**/*.go", "OFFSET") {
		patterns["pagination"] = "offset-legacy"
	}
	if containsGlob(e.root, "encore/**/*.go", "auth.UserID") {
		patterns["auth"] = "encore-auth-middleware"
	}
	if containsGlob(e.root, "**/*.tsx", "react-hook-form") {
		patterns["forms"] = "react-hook-form+zod"
	}

	return patterns
}

func (e *Engine) extractDependencies(fp StackFingerprint) map[string][]string {
	deps := map[string][]string{}

	if fp.Stack == "go" || strings.HasPrefix(fp.Stack, "go-") {
		content := readFile(e.root, "go.mod")
		for _, line := range strings.Split(content, "\n") {
			if strings.Contains(line, "encore.dev") || strings.Contains(line, "testify") || strings.Contains(line, "pgx") {
				deps["critical_go"] = append(deps["critical_go"], strings.Fields(line)[0])
			}
		}
	}

	if fp.Stack == "node" {
		content := readFile(e.root, "package.json")
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal([]byte(content), &pkg); err == nil {
			critical := []string{"react-query", "tanstack", "zod", "react-hook-form", "shadcn", "lucide", "next", "typescript"}
			allDeps := map[string]string{}
			for k, v := range pkg.Dependencies {
				allDeps[k] = v
			}
			for k, v := range pkg.DevDependencies {
				allDeps[k] = v
			}
			for dep := range allDeps {
				for _, c := range critical {
					if strings.Contains(dep, c) {
						deps["critical_js"] = append(deps["critical_js"], dep)
						break
					}
				}
			}
		}
	}

	return deps
}

func (e *Engine) computeGlobalChecksum() string {
	var files []string
	filepath.Walk(e.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || isIgnored(path) {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".py" || ext == ".rs" || ext == ".sql" {
			files = append(files, path)
		}
		return nil
	})

	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		content, _ := os.ReadFile(f)
		h.Write(content)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func exists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func readFile(root, name string) string {
	b, _ := os.ReadFile(filepath.Join(root, name))
	return string(b)
}

func contains(root, name, sub string) bool {
	return strings.Contains(readFile(root, name), sub)
}

func containsGlob(root, pattern, sub string) bool {
	matches, _ := filepath.Glob(filepath.Join(root, pattern))
	for _, m := range matches {
		if strings.Contains(readFile(root, m), sub) {
			return true
		}
	}
	return false
}

func readModule(root string) string {
	content := readFile(root, "go.mod")
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func extractName(content string) string {
	_, after, ok := strings.Cut(content, `"name"`)
	if !ok {
		return ""
	}
	rest := strings.TrimSpace(after)
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) > 0 && rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end >= 0 {
			return rest[1 : end+1]
		}
	}
	return ""
}

func extractImports(path string) []string {
	content, _ := os.ReadFile(path)
	var imports []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, `"`) {
			line = strings.Trim(line, `" ()`)
			line = strings.TrimPrefix(line, "import ")
			if line != "" && line != "(" {
				imports = append(imports, line)
			}
		}
	}
	return imports
}

func isIgnored(path string) bool {
	ignored := []string{"node_modules", ".next", "dist", ".opencode", ".git", "vendor", "target"}
	for _, ign := range ignored {
		if strings.Contains(path, ign) {
			return true
		}
	}
	return false
}
