package template

import (
	"github.com/Lenstack/opencode-scaffold/internal/detector"
)

func DetectTemplate(stack *detector.Stack, fileCount int, hasCI bool) string {
	score := map[string]int{
		"standard":        1,
		"minimal":         0,
		"solo-dev":        0,
		"team-production": 0,
		"api-backend":     0,
		"frontend-app":    0,
		"fullstack":       0,
	}

	// Stack-based scoring
	switch {
	case stack.HasPubSub && stack.HasAuth:
		score["fullstack"] += 5
	case stack.Backend == "go" && stack.Framework == "encore":
		score["fullstack"] += 3
	case stack.HasDB && (stack.Framework == "fastapi" || stack.Framework == "gin" || stack.Framework == "axum" || stack.Framework == "fiber"):
		score["api-backend"] += 5
	case stack.Frontend == "nextjs" || stack.Frontend == "react" || stack.Frontend == "vue":
		score["frontend-app"] += 4
	case stack.Backend == "go" && stack.Framework != "":
		score["api-backend"] += 2
	case stack.Backend == "python" && stack.Framework != "":
		score["api-backend"] += 2
	case stack.Backend == "rust" && stack.Framework != "":
		score["api-backend"] += 2
	}

	// Multiple frameworks = fullstack
	if stack.Frontend != "" && stack.Backend != "" {
		score["fullstack"] += 3
	}

	// Project size scoring
	if fileCount < 30 {
		score["solo-dev"] += 4
		score["minimal"] += 2
	} else if fileCount < 100 {
		score["minimal"] += 2
		score["standard"] += 1
	} else if fileCount > 200 {
		score["team-production"] += 2
	}

	// Team signals
	if hasCI {
		score["team-production"] += 3
		score["standard"] += 1
	}

	// Find highest scoring template
	bestID := "standard"
	bestScore := 0
	for id, s := range score {
		if s > bestScore {
			bestScore = s
			bestID = id
		}
	}

	return bestID
}
