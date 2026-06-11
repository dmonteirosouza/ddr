# ddr

`ddr` is a small Go CLI for macOS developer machines that get heavy with Docker, VS Code/Codex, Chrome, Flutter, Node, Gradle, and local caches.

The default posture is conservative: reports are read-only, and cleanup commands require `--yes`.

Instead of dumping raw command output, `ddr` extracts the important numbers and renders compact terminal summaries for disk, memory, Docker, VS Code, and heavy folders.

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

## Safety Notes

`ddr` never removes Docker volumes automatically. Volumes may contain databases, local object storage, model files, and other project data.

VS Code settings changes always create a timestamped backup next to `settings.json`.
