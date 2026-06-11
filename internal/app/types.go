package app

import "regexp"

type sizeEntry struct {
	label string
	path  string
}

type folderSize struct {
	label string
	path  string
	size  string
	bytes uint64
	ok    bool
}

type commandResult struct {
	ok     bool
	output string
	err    error
}

type diskInfo struct {
	mount      string
	totalBytes uint64
	usedBytes  uint64
	freeBytes  uint64
	usedPct    float64
}

type swapUsage struct {
	totalMB float64
	usedMB  float64
	freeMB  float64
}

type memoryInfo struct {
	ramBytes       uint64
	swap           swapUsage
	swapOK         bool
	freePct        int
	freePctOK      bool
	compressorSize string
}

type processInfo struct {
	pid     string
	command string
	memRaw  string
	bytes   uint64
}

type dockerRow struct {
	kind          string
	total         string
	active        string
	size          string
	reclaimable   string
	reclaimBytes  uint64
	reclaimPctRaw string
}

var (
	homeDir = mustHomeDir()

	swapRegexp       = regexp.MustCompile(`total = ([0-9.]+)M\s+used = ([0-9.]+)M\s+free = ([0-9.]+)M`)
	memoryFreeRegexp = regexp.MustCompile(`System-wide memory free percentage:\s+([0-9]+)%`)
	compressorRegexp = regexp.MustCompile(`Pages used by compressor:\s+([0-9]+)`)

	vscodeSettingsPath         = expand("~/Library/Application Support/Code/User/settings.json")
	vscodeWorkspaceStoragePath = expand("~/Library/Application Support/Code/User/workspaceStorage")

	candidatePaths = []sizeEntry{
		{"Docker Desktop", "~/Library/Containers/com.docker.docker"},
		{"VS Code workspaceStorage", "~/Library/Application Support/Code/User/workspaceStorage"},
		{"VS Code extensions", "~/.vscode/extensions"},
		{"npm cache", "~/.npm"},
		{"Gradle", "~/.gradle"},
		{"Pub cache", "~/.pub-cache"},
		{"User caches", "~/Library/Caches"},
		{"Downloads", "~/Downloads"},
		{"CoreSimulator", "~/Library/Developer/CoreSimulator"},
		{"Google app data", "~/Library/Application Support/Google"},
		{"Claude app data", "~/Library/Application Support/Claude"},
		{"VS Code app data", "~/Library/Application Support/Code"},
	}

	watcherExcludes = map[string]any{
		"**/.git/objects/**":       true,
		"**/.git/subtree-cache/**": true,
		"**/node_modules/**":       true,
		"**/.next/**":              true,
		"**/.nuxt/**":              true,
		"**/dist/**":               true,
		"**/build/**":              true,
		"**/coverage/**":           true,
		"**/.turbo/**":             true,
		"**/.cache/**":             true,
		"**/.dart_tool/**":         true,
		"**/.gradle/**":            true,
		"**/Pods/**":               true,
		"**/vendor/**":             true,
	}
)
