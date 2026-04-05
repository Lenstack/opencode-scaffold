package hub

const (
	NSDiscovery        = "discovery"
	NSSpecs            = "specs"
	NSMemoryEpisodic   = "memory:episodic"
	NSMemorySemantic   = "memory:semantic"
	NSMemoryHeuristic  = "memory:heuristic"
	NSMemoryQuarantine = "memory:quarantine"
	NSSessions         = "sessions"
	NSSkills           = "skills"
	NSQuality          = "quality"
	NSContext          = "context"
	NSOptimization     = "optimization"
)

type ProjectMap struct {
	Version      string              `json:"version"`
	ScannedAt    string              `json:"scanned_at"`
	FilesCount   int                 `json:"files_count"`
	Checksum     string              `json:"checksum"`
	Stack        string              `json:"stack"`
	Frameworks   string              `json:"frameworks"`
	ChangedFiles []string            `json:"changed_files"`
	DirtyDomains []string            `json:"dirty_domains"`
	HotDomains   []HotDomain         `json:"hot_domains"`
	DBTables     []DBTable           `json:"db_tables"`
	APIRoutes    []APIRoute          `json:"api_routes"`
	Patterns     map[string]string   `json:"patterns"`
	Dependencies map[string][]string `json:"dependencies"`
}

type HotDomain struct {
	Domain  string `json:"domain"`
	Commits int    `json:"commits"`
}

type FileEntry struct {
	Path     string   `json:"path"`
	Type     string   `json:"type"`
	Size     int64    `json:"size"`
	Modified int64    `json:"modified"`
	Imports  []string `json:"imports"`
}

type APIRoute struct {
	Method    string `json:"method"`
	Endpoint  string `json:"endpoint"`
	Handler   string `json:"handler"`
	AuthLevel string `json:"auth_level"`
}

type DBTable struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	Indexes    []string `json:"indexes"`
	Migrations []string `json:"migrations"`
}

type SpecEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type SpecRequirements struct {
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EdgeCases          []string `json:"edge_cases"`
}

type SpecImplementation struct {
	Files      []string `json:"files"`
	Tests      []string `json:"tests"`
	VerifiedAt string   `json:"verified_at"`
}

type SpecVerification struct {
	Status         string   `json:"status"`
	Results        []string `json:"results"`
	FailedCriteria []string `json:"failed_criteria"`
}

type EpisodicMemory struct {
	SessionID             string   `json:"session_id"`
	TS                    string   `json:"ts"`
	ExpiresAt             string   `json:"expires_at"`
	Feature               string   `json:"feature"`
	AgentsRan             []string `json:"agents_ran"`
	SelfHealEvents        []string `json:"self_heal_events"`
	TDDPhase2FirstAttempt string   `json:"tdd_phase2_first_attempt"`
	HeuristicOverrides    []string `json:"heuristic_overrides"`
	Outcome               string   `json:"outcome"`
	KeyLesson             string   `json:"key_lesson"`
}

type SemanticMemory struct {
	TS           string  `json:"ts"`
	ExpiresAt    string  `json:"expires_at"`
	Category     string  `json:"category"`
	FactKey      string  `json:"fact_key"`
	Fact         string  `json:"fact"`
	Confidence   float64 `json:"confidence"`
	SessionCount int     `json:"session_count"`
	Source       string  `json:"source"`
}

type HeuristicRule struct {
	ID              string   `json:"id"`
	PromotedAt      string   `json:"promoted_at"`
	SourceSessions  []string `json:"source_sessions"`
	Rule            string   `json:"rule"`
	Rationale       string   `json:"rationale"`
	OverrideCount   int      `json:"override_count"`
	InvocationCount int      `json:"invocation_count"`
	OverrideRate    float64  `json:"override_rate"`
	Confidence      float64  `json:"confidence"`
	Active          bool     `json:"active"`
}

type SessionEntry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Agent     string `json:"agent"`
	Model     string `json:"model"`
	StartedAt string `json:"started_at"`
	Status    string `json:"status"`
}

type SessionContext struct {
	FilesRead    []string `json:"files_read"`
	FilesWritten []string `json:"files_written"`
	Decisions    []string `json:"decisions"`
}

type SessionSummary struct {
	Summary    string `json:"summary"`
	Duration   int    `json:"duration"`
	TokensUsed int    `json:"tokens_used"`
	Outcome    string `json:"outcome"`
}

type SkillEntry struct {
	Name          string  `json:"name"`
	CreatedAt     string  `json:"created_at"`
	UsageCount    int     `json:"usage_count"`
	LastUsed      string  `json:"last_used"`
	Effectiveness float64 `json:"effectiveness"`
}

type SkillUsage struct {
	SessionID     string `json:"session_id"`
	LoadedAt      string `json:"loaded_at"`
	Outcome       string `json:"outcome"`
	AgentFeedback string `json:"agent_feedback"`
}

type SkillKnowledge struct {
	Patterns       []string `json:"patterns"`
	AntiPatterns   []string `json:"anti_patterns"`
	CodeExamples   []string `json:"code_examples"`
	ProjectContext string   `json:"project_context"`
}

type OptimizationLog struct {
	Skill              string  `json:"skill"`
	Change             string  `json:"change"`
	Reason             string  `json:"reason"`
	Timestamp          string  `json:"timestamp"`
	EffectivenessDelta float64 `json:"effectiveness_delta"`
}

type TestRun struct {
	Suite    string  `json:"suite"`
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	Coverage float64 `json:"coverage"`
	Duration int     `json:"duration"`
}

type SecurityScan struct {
	Findings       []Finding      `json:"findings"`
	SeverityCounts map[string]int `json:"severity_counts"`
	ScannedAt      string         `json:"scanned_at"`
}

type Finding struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Issue    string `json:"issue"`
	Fix      string `json:"fix"`
}

type DependencyGraph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Path string `json:"path"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type FileIndex struct {
	Path       string   `json:"path"`
	Type       string   `json:"type"`
	Imports    []string `json:"imports"`
	ImportedBy []string `json:"imported_by"`
}

type SymbolEntry struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Location   string   `json:"location"`
	References []string `json:"references"`
}
