## Verification

### TASK-001: homebrew-hoofy tap repo ✅
- [x] Repo created at `github.com/HendryAvila/homebrew-hoofy` (public)
- [x] Description set: "Homebrew tap for Hoofy — Spec-Driven Development MCP Server"

### TASK-002: GoReleaser brews config ✅
- [x] `brews` section added to `.goreleaser.yaml` with correct `repository` (owner/name/token)
- [x] `directory: Formula` — standard Homebrew convention
- [x] `install` block: `bin.install "hoofy"`
- [x] `test` block: `system "#{bin}/hoofy", "--version"`
- [x] Homepage, description, license (MIT) set
- [x] Uses `TAP_GITHUB_TOKEN` env var (separate from `GITHUB_TOKEN`)
- [x] Release workflow updated to pass `TAP_GITHUB_TOKEN` secret

### Manual step required
- [ ] **User must create a GitHub PAT** with `repo` scope and add it as `TAP_GITHUB_TOKEN` secret in the Hoofy repo settings (Settings → Secrets → Actions → New repository secret)
- Without this, the GoReleaser brew step will fail silently (or skip) on the next release
