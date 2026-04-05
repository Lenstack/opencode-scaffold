package detector

type signal struct {
	file     string
	contains string
	score    float64
	field    string
	value    string
}

var signals = []signal{
	{file: "go.mod", score: 0.3, field: "backend", value: "go"},
	{file: "go.mod", contains: "encore.dev", score: 0.5, field: "framework", value: "encore"},
	{file: "go.mod", contains: "gofiber/fiber", score: 0.5, field: "framework", value: "fiber"},
	{file: "go.mod", contains: "gin-gonic/gin", score: 0.5, field: "framework", value: "gin"},
	{file: "go.mod", contains: "go-chi/chi", score: 0.5, field: "framework", value: "chi"},
	{file: "package.json", score: 0.3, field: "backend", value: "node"},
	{file: "package.json", contains: `"next"`, score: 0.5, field: "frontend", value: "nextjs"},
	{file: "package.json", contains: `"react"`, score: 0.3, field: "frontend", value: "react"},
	{file: "package.json", contains: `"vue"`, score: 0.3, field: "frontend", value: "vue"},
	{file: "pyproject.toml", score: 0.3, field: "backend", value: "python"},
	{file: "pyproject.toml", contains: "fastapi", score: 0.5, field: "framework", value: "fastapi"},
	{file: "pyproject.toml", contains: "django", score: 0.5, field: "framework", value: "django"},
	{file: "pyproject.toml", contains: "flask", score: 0.5, field: "framework", value: "flask"},
	{file: "requirements.txt", score: 0.2, field: "backend", value: "python"},
	{file: "Cargo.toml", score: 0.3, field: "backend", value: "rust"},
	{file: "Cargo.toml", contains: "axum", score: 0.5, field: "framework", value: "axum"},
	{file: "docker-compose.yml", score: 0.1, field: "docker", value: "true"},
	{file: "Dockerfile", score: 0.1, field: "docker", value: "true"},
}
