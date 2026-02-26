## Tasks

### TASK-001: Create homebrew-hoofy tap repository on GitHub
**Description**: Create the `HendryAvila/homebrew-hoofy` repo with a README
**Acceptance Criteria**:
- [ ] Repo exists at `github.com/HendryAvila/homebrew-hoofy`
- [ ] Has a README explaining usage: `brew install HendryAvila/hoofy/hoofy`

### TASK-002: Add `brews` section to .goreleaser.yaml
**Description**: Configure GoReleaser to auto-generate and push the Homebrew Formula on each release
**Acceptance Criteria**:
- [ ] `brews` section added with correct tap repo reference
- [ ] Formula targets darwin (amd64, arm64) and linux (amd64, arm64)
- [ ] Description, homepage, and license set correctly
- [ ] GoReleaser token needs repo write access to the tap repo (existing GITHUB_TOKEN should work)
