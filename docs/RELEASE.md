# Release & Homebrew

This project publishes release tarballs and updates Homebrew formula automatically.

## Prerequisites

1. GitHub repo: `FerdiKT/appstore-cli`
2. Homebrew tap repo: `FerdiKT/homebrew-tap`
3. GitHub Actions secret in `appstore-cli` repo:
   - `HOMEBREW_TAP_TOKEN` (PAT with repo access to `FerdiKT/homebrew-tap`)

## Release flow

1. Commit to `main`
2. Create version tag (example `v0.1.0`)
3. Push tag
4. GitHub Action `.github/workflows/release.yml` will:
   - build `dist/*.tar.gz` via `make brew-dist`
   - create GitHub Release
   - update Homebrew formula in `FerdiKT/homebrew-tap`

## Commands

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Install via Homebrew

```bash
brew tap FerdiKT/tap
brew install appstore
```
