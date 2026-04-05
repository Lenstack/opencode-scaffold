package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSection struct {
	Path      string
	StartLine int
	EndLine   int
	Content   string
	Reason    string
}

type Extractor struct {
	Root string
}

func NewExtractor(root string) *Extractor {
	return &Extractor{Root: root}
}

func (e *Extractor) ExtractRelevantSections(task string, files []string) ([]FileSection, error) {
	var sections []FileSection
	keywords := extractKeywords(task)

	for _, file := range files {
		fullPath := filepath.Join(e.Root, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")

		for i, line := range lines {
			if containsAny(line, keywords) {
				start := max(0, i-10)
				end := min(len(lines), i+11)
				sections = append(sections, FileSection{
					Path:      file,
					StartLine: start + 1,
					EndLine:   end,
					Content:   formatWithLineNumbers(lines[start:end], start+1),
					Reason:    fmt.Sprintf("matched keyword in line %d", i+1),
				})
			}
		}
	}

	return mergeOverlapping(sections), nil
}

func formatWithLineNumbers(lines []string, startLine int) string {
	var sb strings.Builder
	for i, line := range lines {
		sb.WriteString(fmt.Sprintf("%4d | %s\n", startLine+i, line))
	}
	return sb.String()
}

func extractKeywords(task string) []string {
	var keywords []string
	current := ""
	for _, r := range task {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			current += string(r)
		} else if len(current) >= 3 {
			keywords = append(keywords, strings.ToLower(current))
			current = ""
		} else {
			current = ""
		}
	}
	if len(current) >= 3 {
		keywords = append(keywords, strings.ToLower(current))
	}
	return keywords
}

func containsAny(s string, substrings []string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

func mergeOverlapping(sections []FileSection) []FileSection {
	if len(sections) == 0 {
		return sections
	}

	merged := []FileSection{sections[0]}
	for i := 1; i < len(sections); i++ {
		last := &merged[len(merged)-1]
		curr := sections[i]

		if curr.Path == last.Path && curr.StartLine <= last.EndLine+1 {
			if curr.EndLine > last.EndLine {
				last.EndLine = curr.EndLine
				last.Content = formatWithLineNumbers(
					readLines(filepath.Join("/tmp", curr.Path), last.StartLine-1, last.EndLine),
					last.StartLine,
				)
			}
		} else {
			merged = append(merged, curr)
		}
	}
	return merged
}

func readLines(path string, start, end int) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")
	if start >= len(lines) {
		return nil
	}
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
