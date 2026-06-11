# ddr

`ddr` is a small Go CLI for macOS developer machines that get heavy with Docker, VS Code/Codex, Chrome, Flutter, Node, Gradle, and local caches.

The default posture is conservative: reports are read-only, and cleanup commands require `--yes`.

Instead of dumping raw command output, `ddr` extracts the important numbers and renders a colorful terminal dashboard with simple statuses and tips.

## Commands

```bash
ddr scan
ddr memory
ddr docker
ddr vscode
ddr vscode --apply
ddr clean
ddr clean --safe --yes
ddr clean --all-safe --yes
ddr clean --vscode-storage --yes
ddr chrome
```

## What It Does

- `scan`: shows parsed disk, memory, Docker, process, and heavy-folder summaries.
- `memory`: shows RAM/swap pressure plus process families and top memory processes.
- `docker`: shows parsed Docker disk usage and reminds you that volumes are not cleaned by default.
- `vscode`: reports VS Code storage, tuning status, and installed extensions.
- `vscode --apply`: backs up VS Code settings and applies lighter defaults for Codex, GitLens, TypeScript, watchers, and editor limits.
- `clean --safe --yes`: prunes Docker build cache, npm cache, and Gradle caches.
- `clean --all-safe --yes`: also prunes stopped containers, unused networks, and unused Docker images, preserving volumes.
- `clean --vscode-storage --yes`: removes VS Code workspace cache/state. Close VS Code first.
- `chrome`: prints the Chrome settings checklist for tab-heavy usage.

## Install Locally

From this repository:

```bash
mkdir -p ~/.local/bin
go build -o ~/.local/bin/ddr ./cmd/ddr
```

Make sure `~/.local/bin` is in your `PATH`.

## Releases For Non-Technical Users

Installers are generated only as release artifacts, not committed to the repository.

To build release ZIPs locally:

```bash
./scripts/package-release.sh v0.1.0
```

This creates:

```text
dist/ddr_v0.1.0_darwin_arm64.zip
dist/ddr_v0.1.0_darwin_amd64.zip
```

Each ZIP contains:

- `ddr`: the macOS binary.
- `install.command`: double-click installer for non-technical users.
- `LEIA-ME.txt`: short install/removal instructions.

On GitHub, create a tag to publish these ZIPs automatically:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Project Layout

```text
.
├── .github/workflows/release.yml # tagged release packaging
├── cmd/ddr/main.go             # CLI binary entrypoint
├── scripts/package-release.sh  # builds release ZIPs
├── internal/app/app.go         # command routing
├── internal/app/commands.go    # command workflows
├── internal/app/chrome.go      # Chrome checklist
├── internal/app/clean.go       # conservative cleanup workflow
├── internal/app/disk.go        # disk report
├── internal/app/docker.go      # Docker report/parsing
├── internal/app/memory.go      # memory/swap report
├── internal/app/processes.go   # process collection/grouping
├── internal/app/sizes.go       # heavy-folder sizing
├── internal/app/system.go      # shell/system helpers
├── internal/app/vscode.go      # VS Code report/settings
├── internal/app/vscode_settings.go # VS Code settings writer
├── internal/ui/ui.go           # colored terminal UI helpers
├── go.mod
└── README.md
```

This follows the common Go layout where `cmd/ddr` stays thin and code that should not be imported by external projects lives under `internal`.

## Safety Notes

`ddr` never removes Docker volumes automatically. Volumes may contain databases, local object storage, model files, and other project data.

VS Code settings changes always create a timestamped backup next to `settings.json`.
