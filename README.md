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
ddr version
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
- `version`: prints the current build version.

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

Maintainers publish these ZIPs from `main` only after the release pull request has been reviewed and merged:

```bash
git checkout main
git pull origin main
git tag v0.1.0
git push origin v0.1.0
```

Do not push release changes directly to `main`. Put the change in a pull request first, merge it, then create and push the release tag.

You can also run the release workflow manually from GitHub Actions with a version input like `v0.1.0`, after the release changes are already on `main`.

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

## Contributing

Contributions are welcome. Keep the project simple, conservative, and friendly for non-technical macOS users.

### Local Setup

Requirements:

- macOS.
- Go installed.
- Docker Desktop and VS Code are optional, but useful for testing those reports.

Clone and build:

```bash
git clone https://github.com/dmonteirosouza/ddr.git
cd ddr
go test ./...
go build -o ./ddr ./cmd/ddr
./ddr scan
```

### Development Guidelines

- Keep `cmd/ddr/main.go` as a thin entrypoint.
- Put CLI workflows and system checks under `internal/app`.
- Put terminal rendering helpers under `internal/ui`.
- Prefer parsed summaries and plain-language tips over raw command dumps.
- Avoid deleting user data automatically.
- Never delete Docker volumes automatically.
- Any destructive cleanup must require `--yes`.
- Keep release artifacts out of Git; use `scripts/package-release.sh`.

### Before Opening A Pull Request

Create a branch for your change:

```bash
git checkout main
git pull origin main
git checkout -b docs/update-readme
```

Run:

```bash
gofmt -w cmd/ddr/main.go internal/app/*.go internal/ui/*.go
go test ./...
go build -o ./ddr ./cmd/ddr
./ddr help
./ddr clean
```

If your change affects a report, also run the related command, for example:

```bash
./ddr scan
./ddr memory
./ddr docker
./ddr vscode
```

Commit and open a pull request:

```bash
git add .
git commit -m "Describe your change"
git push -u origin docs/update-readme
gh pr create --base main --head docs/update-readme --fill
```

If you need to adjust the last local commit before pushing, amend it:

```bash
git add .
git commit --amend
```

Do not push directly to `main`; use a pull request so changes can be reviewed.

### Release Checklist

For maintainers, after the release PR has been reviewed and merged:

```bash
git checkout main
git pull origin main
go test ./...
./scripts/package-release.sh v0.1.0
git tag v0.1.0
git push origin v0.1.0
```

The GitHub release should contain:

- `ddr_vX.Y.Z_darwin_arm64.zip` for Apple Silicon Macs.
- `ddr_vX.Y.Z_darwin_amd64.zip` for Intel Macs.

## Safety Notes

`ddr` never removes Docker volumes automatically. Volumes may contain databases, local object storage, model files, and other project data.

VS Code settings changes always create a timestamped backup next to `settings.json`.
