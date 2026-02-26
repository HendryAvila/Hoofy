## What

Add Homebrew tap support so macOS and Linux users can install Hoofy with `brew install HendryAvila/hoofy/hoofy` (or `brew tap HendryAvila/hoofy && brew install hoofy`).

## Why

- `brew install` is the de facto standard for CLI tools on macOS
- Homebrew also works on Linux — covers both platforms
- Automatic updates via `brew upgrade hoofy`
- Professional/polished distribution channel
- GoReleaser has built-in support for generating Homebrew Formulas on each release

## Scope

1. Create `homebrew-hoofy` repo on GitHub with an initial README
2. Add `brews` section to `.goreleaser.yaml` so each release auto-publishes the Formula
3. GoReleaser will generate and push the Formula to the tap repo on every tagged release

## Out of Scope

- Homebrew Core submission (requires popularity threshold)
- Cask (Hoofy is a CLI, not a GUI app)
- Linux-specific package managers (apt, dnf, pacman)
