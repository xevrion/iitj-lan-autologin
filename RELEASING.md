# Releasing

This file is the maintainer checklist for shipping a new `iitj-login` version.

## What counts as a new release

Create a new release when users would benefit from a versioned binary update, for example:

- new features
- bug fixes that affect login reliability or installation
- bootstrap or service-management changes
- security-related fixes
- platform support changes

Do not create a release for every tiny commit. Batch related fixes together unless a hotfix is needed.

## Versioning rule

This project uses `vMAJOR.MINOR.PATCH`.

- `MAJOR`: breaking behavior changes, major rewrites, or a release line change
- `MINOR`: new features or meaningful installer/platform improvements
- `PATCH`: bug fixes, docs tied to shipped behavior, or small reliability fixes

Examples:

- `v4.0.0` -> large new release line
- `v4.1.0` -> new feature or platform capability
- `v4.1.1` -> small fix after `v4.1.0`

## Files to update before release

Before tagging a release, check these files:

- `main.go`
  Make sure the `version` constant matches the release tag.
- `CHANGELOG.md`
  Add a short entry for the new version.
- `README.md`
  Update only if install steps, platform support, commands, or behavior changed.
- `bootstrap.sh` and `bootstrap.ps1`
  Update only if install or release asset naming changed.

## Pre-release checklist

Run this before creating a tag:

```bash
go test ./...
```

If the sandboxed environment blocks Go cache writes, use:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

Then verify:

- the working tree is clean except intentional untracked files
- `main.go` has the correct version
- `CHANGELOG.md` includes the new version
- the release workflow still matches the expected binary names

## Release steps

Assume the next version is `v4.0.1`. Replace it with the real version.

1. Update version metadata.

```bash
$EDITOR main.go
$EDITOR CHANGELOG.md
```

2. Commit the release prep.

```bash
git add main.go CHANGELOG.md README.md bootstrap.sh bootstrap.ps1 .github/workflows/release.yml
git commit -m "release: prepare v4.0.1"
```

3. Create the annotated tag.

```bash
git tag -a v4.0.1 -m "v4.0.1"
```

4. Push the branch and tags.

```bash
git push origin main --follow-tags
```

5. Wait for GitHub Actions to build and publish the release assets.

```bash
gh run list --limit 5
gh release view v4.0.1
```

## What the release workflow publishes

The tag-triggered workflow builds and attaches:

- `iitj-login-linux-amd64`
- `iitj-login-linux-arm64`
- `iitj-login-darwin-amd64`
- `iitj-login-darwin-arm64`
- `iitj-login-windows-amd64.exe`
- `iitj-login-windows-arm64.exe`
- `iitj-login.1`
- `SHA256SUMS`

## Post-release checks

After GitHub finishes the workflow, verify:

- the GitHub Release exists for the tag
- all expected binaries are attached
- `SHA256SUMS` is attached
- the release notes look sane
- `bootstrap.sh` downloads the latest release asset
- `bootstrap.ps1` downloads the latest release asset

Useful commands:

```bash
gh release view v4.0.1
gh release download v4.0.1 --dir /tmp/iitj-release-check
```

## Hotfix flow

If you discover a release-breaking bug immediately after publishing:

1. Fix the bug on `main`
2. Bump `PATCH`
3. Update `main.go` and `CHANGELOG.md`
4. Tag a new release, for example `v4.0.2`

Do not move or reuse an existing public tag.

## Important rules

- Never retag an already-published version.
- Always use annotated tags: `git tag -a ...`
- Tag only after the release automation commit is already in `main`.
- Keep binary names stable unless you also update both bootstrap scripts and the workflow.
- Keep `CHANGELOG.md` human-readable; short and accurate is enough.

## Current setup notes

- `bootstrap.sh` and `bootstrap.ps1` install the latest GitHub Release binary first.
- Source builds remain as a fallback when no matching release asset exists.
- Old script-based releases are preserved in `CHANGELOG.md`.
