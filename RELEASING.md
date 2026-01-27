# Releasing Raterunner

This guide describes how to release a new version of Raterunner CLI.

## Version Numbering

We use [Semantic Versioning](https://semver.org/):

- **MAJOR** (`v1.0.0` → `v2.0.0`): Breaking changes
- **MINOR** (`v1.0.0` → `v1.1.0`): New features, backwards compatible
- **PATCH** (`v1.0.0` → `v1.0.1`): Bug fixes, backwards compatible

## Prerequisites

1. All changes committed and pushed to `main`
2. All CI checks passing (tests, lint, release-check)
3. `HOMEBREW_TAP_TOKEN` secret configured in GitHub repository

## Release Process

### Quick Release

```bash
make release VERSION=v0.2.0
```

This command will:
1. Create an annotated git tag
2. Push the tag to GitHub
3. Trigger the release workflow

### Manual Release

```bash
# 1. Ensure you're on main and up to date
git checkout main
git pull origin main

# 2. Verify everything passes
make test
make release-check

# 3. Create and push tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### What Happens Automatically

When a tag starting with `v` is pushed, GitHub Actions will:

1. **Build binaries** for all platforms:
   - macOS (amd64, arm64)
   - Linux (amd64, arm64)
   - Windows (amd64, arm64)

2. **Create packages**:
   - `.tar.gz` archives (macOS, Linux)
   - `.zip` archives (Windows)
   - `.deb` packages (Debian/Ubuntu)
   - `.rpm` packages (RHEL/Fedora)
   - `.apk` packages (Alpine)

3. **Publish release** to GitHub Releases with:
   - All binaries and packages
   - Checksums file
   - Auto-generated changelog

4. **Update Homebrew tap** at `raterunner/homebrew-tap`

## Verifying the Release

After the workflow completes (~2-3 minutes):

1. Check [GitHub Releases](https://github.com/raterunner/cli/releases)
2. Verify all artifacts are present
3. Test installation:

```bash
# Homebrew (may take a few minutes to propagate)
brew update
brew install raterunner/tap/raterunner
raterunner --version

# Direct download
curl -Lo raterunner.tar.gz https://github.com/raterunner/cli/releases/download/v0.2.0/raterunner_0.2.0_Darwin_arm64.tar.gz
tar -xzf raterunner.tar.gz
./raterunner --version
```

## Troubleshooting

### Release workflow didn't trigger

- Ensure tag starts with `v` (e.g., `v0.2.0`, not `0.2.0`)
- Check if tag was pushed: `git ls-remote --tags origin`

### Homebrew formula not updated

- Check `HOMEBREW_TAP_TOKEN` secret is set and valid
- Verify `raterunner/homebrew-tap` repository exists
- Check workflow logs for errors

### Build failed

- Check CI logs in GitHub Actions
- Run locally to debug: `make release-snapshot`

## Hotfix Release

For urgent fixes to a released version:

```bash
# 1. Create hotfix branch from tag
git checkout -b hotfix/v0.2.1 v0.2.0

# 2. Make fixes, commit
git add .
git commit -m "Fix critical bug"

# 3. Merge to main
git checkout main
git merge hotfix/v0.2.1
git push origin main

# 4. Tag and release
make release VERSION=v0.2.1

# 5. Clean up
git branch -d hotfix/v0.2.1
```

## Release Checklist

- [ ] All tests passing
- [ ] CHANGELOG updated (if maintained)
- [ ] Version number follows semver
- [ ] Tag starts with `v`
- [ ] Release notes describe changes
- [ ] Homebrew formula updated
- [ ] Installation tested on at least one platform
