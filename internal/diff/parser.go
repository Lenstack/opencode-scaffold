package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []Line
}

type Line struct {
	Type    string
	Content string
}

type FileDiff struct {
	OldPath string
	NewPath string
	Hunks   []Hunk
}

var hunkRe = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func ParseUnifiedDiff(diffText string) ([]FileDiff, error) {
	var diffs []FileDiff
	var currentDiff *FileDiff
	var currentHunk *Hunk

	for _, line := range strings.Split(diffText, "\n") {
		if strings.HasPrefix(line, "--- ") {
			currentDiff = &FileDiff{OldPath: strings.TrimPrefix(line, "--- ")}
			diffs = append(diffs, *currentDiff)
			currentHunk = nil
		} else if strings.HasPrefix(line, "+++ ") {
			if currentDiff != nil {
				currentDiff.NewPath = strings.TrimPrefix(line, "+++ ")
			}
		} else if strings.HasPrefix(line, "@@ ") {
			matches := hunkRe.FindStringSubmatch(line)
			if len(matches) >= 5 {
				hunk := Hunk{}
				hunk.OldStart, _ = strconv.Atoi(matches[1])
				if matches[2] != "" {
					hunk.OldCount, _ = strconv.Atoi(matches[2])
				} else {
					hunk.OldCount = 1
				}
				hunk.NewStart, _ = strconv.Atoi(matches[3])
				if matches[4] != "" {
					hunk.NewCount, _ = strconv.Atoi(matches[4])
				} else {
					hunk.NewCount = 1
				}
				currentHunk = &hunk
				if currentDiff != nil {
					currentDiff.Hunks = append(currentDiff.Hunks, *currentHunk)
					currentHunk = &currentDiff.Hunks[len(currentDiff.Hunks)-1]
				}
			}
		} else if currentHunk != nil {
			lineType := "context"
			content := line
			if strings.HasPrefix(line, "+") {
				lineType = "add"
				content = line[1:]
			} else if strings.HasPrefix(line, "-") {
				lineType = "remove"
				content = line[1:]
			}
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    lineType,
				Content: content,
			})
		}
	}

	return diffs, nil
}

func ApplyDiff(root string, diff FileDiff) error {
	if diff.OldPath == "/dev/null" {
		var content strings.Builder
		for _, hunk := range diff.Hunks {
			for _, line := range hunk.Lines {
				if line.Type == "add" || line.Type == "context" {
					content.WriteString(line.Content)
					content.WriteString("\n")
				}
			}
		}
		return os.WriteFile(filepath.Join(root, diff.NewPath), []byte(content.String()), 0644)
	}

	content, err := os.ReadFile(filepath.Join(root, diff.OldPath))
	if err != nil {
		return fmt.Errorf("read file %s: %w", diff.OldPath, err)
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	currentLine := 0

	for _, hunk := range diff.Hunks {
		for currentLine < hunk.OldStart-1 && currentLine < len(lines) {
			result = append(result, lines[currentLine])
			currentLine++
		}

		for _, line := range hunk.Lines {
			switch line.Type {
			case "context":
				if currentLine < len(lines) {
					result = append(result, lines[currentLine])
					currentLine++
				}
			case "add":
				result = append(result, line.Content)
			case "remove":
				if currentLine < len(lines) {
					currentLine++
				}
			}
		}
	}

	for currentLine < len(lines) {
		result = append(result, lines[currentLine])
		currentLine++
	}

	outPath := diff.NewPath
	if outPath == "" {
		outPath = diff.OldPath
	}

	return os.WriteFile(filepath.Join(root, outPath), []byte(strings.Join(result, "\n")), 0644)
}

func ValidateDiff(root string, diff FileDiff) error {
	if diff.OldPath != "/dev/null" {
		if _, err := os.Stat(filepath.Join(root, diff.OldPath)); err != nil {
			return fmt.Errorf("file %s does not exist", diff.OldPath)
		}
	}
	return nil
}

func DiffStats(diffs []FileDiff) map[string]int {
	stats := map[string]int{"added": 0, "removed": 0, "files": len(diffs)}
	for _, diff := range diffs {
		for _, hunk := range diff.Hunks {
			for _, line := range hunk.Lines {
				if line.Type == "add" {
					stats["added"]++
				} else if line.Type == "remove" {
					stats["removed"]++
				}
			}
		}
	}
	return stats
}
