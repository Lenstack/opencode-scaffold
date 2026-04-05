package detector

import (
	"os"
	"path/filepath"
	"strings"
)

func Detect(root string) *Stack {
	s := &Stack{}
	bestScore := 0.0

	for _, sig := range signals {
		p := filepath.Join(root, sig.file)
		if _, err := os.Stat(p); err != nil {
			continue
		}

		if sig.contains != "" {
			content := readFile(p)
			if !strings.Contains(content, sig.contains) {
				continue
			}
		}

		s.Confidence += sig.score
		applySignal(s, sig, root)
	}

	if s.Confidence > bestScore {
		bestScore = s.Confidence
	}

	if s.ID == "" {
		s.ID = "generic"
		s.Name = "generic"
		s.Backend = "generic"
		s.Confidence = 0.1
	}

	if s.ID == "" {
		s.ID = s.Backend + "-" + s.Framework
	}

	if exists(root, ".github/workflows") || exists(root, ".gitlab-ci.yml") {
		s.HasCI = true
	}

	return s
}

func applySignal(s *Stack, sig signal, root string) {
	switch sig.field {
	case "backend":
		s.Backend = sig.value
	case "frontend":
		s.Frontend = sig.value
	case "framework":
		s.Framework = sig.value
	case "docker":
		s.HasDocker = true
	}

	switch {
	case s.Backend == "go" && s.Framework == "encore":
		s.ID = "go-encore"
		s.Name = "Go + Encore"
		s.GoModule = readModuleName(root)
	case s.Backend == "go" && s.Framework == "fiber":
		s.ID = "go-fiber"
		s.Name = "Go + Fiber"
		s.GoModule = readModuleName(root)
	case s.Backend == "go" && s.Framework == "gin":
		s.ID = "go-gin"
		s.Name = "Go + Gin"
		s.GoModule = readModuleName(root)
	case s.Backend == "go" && s.Framework == "chi":
		s.ID = "go-chi"
		s.Name = "Go + Chi"
		s.GoModule = readModuleName(root)
	case s.Backend == "go":
		s.ID = "go-stdlib"
		s.Name = "Go (stdlib)"
		s.GoModule = readModuleName(root)
	case s.Backend == "python" && s.Framework == "fastapi":
		s.ID = "python-fastapi"
		s.Name = "Python + FastAPI"
	case s.Backend == "python" && s.Framework == "django":
		s.ID = "python-django"
		s.Name = "Python + Django"
	case s.Backend == "python" && s.Framework == "flask":
		s.ID = "python-flask"
		s.Name = "Python + Flask"
	case s.Backend == "python":
		s.ID = "python"
		s.Name = "Python"
	case s.Backend == "node" && s.Frontend == "nextjs":
		s.ID = "node-nextjs"
		s.Name = "Node + Next.js"
		s.NodePkgName = readPackageName(filepath.Join(root, "package.json"))
	case s.Backend == "node":
		s.ID = "node"
		s.Name = "Node.js"
		s.NodePkgName = readPackageName(filepath.Join(root, "package.json"))
	case s.Backend == "rust" && s.Framework == "axum":
		s.ID = "rust-axum"
		s.Name = "Rust + Axum"
	case s.Backend == "rust":
		s.ID = "rust"
		s.Name = "Rust"
	}
}

func readModuleName(root string) string {
	p := filepath.Join(root, "go.mod")
	content := readFile(p)
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func readPackageName(path string) string {
	content := readFile(path)
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

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
