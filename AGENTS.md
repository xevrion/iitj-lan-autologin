# AGENTS.md

This file defines how human contributors and LLM agents should work in this repository.

If you are an LLM, read this file before making changes.

## Project summary

`iitj-login` keeps IIT Jodhpur hostel Ethernet authenticated against the FortiGate captive portal.

The tool exists because the LAN session expires after roughly 10,000 seconds, which breaks SSH sessions, downloads, builds, and unattended work. The current implementation is a cross-platform Go binary with background-service support for Linux, macOS, and Windows.

## Product goals

- Keep the Ethernet session alive reliably
- Handle the actual IITJ network edge cases, not an idealized portal flow
- Stay easy to install for normal users
- Ship as a single binary when possible
- Keep credentials local and encrypted
- Keep the project maintainable for a solo maintainer

## Core technical facts

These are not optional details. Do not regress them.

### FortiGate flow

1. Trigger the captive portal with a plain HTTP request such as `http://neverssl.com`.
2. Detect FortiGate interception and extract the `fgtauth` token.
3. Fetch `https://gateway.iitj.ac.in:1003/fgtauth?TOKEN`.
4. Extract the hidden `magic` value from the returned HTML.
5. POST IITJ LDAP credentials to `https://gateway.iitj.ac.in:1003/`.
6. Set both `Referer` and `4Tredir` to `https://gateway.iitj.ac.in:1003/login?TOKEN`.
7. Treat `keepalive?` in the response body as the success signal.

### DNS and routing realities

- FortiGate DNS behavior is different on Ethernet than on WiFi.
- The portal hostname must resolve to the internal portal IP seen on the Ethernet path.
- Browser and libc DNS behavior can differ from raw DNS queries.
- Docker can conflict with the `172.17.x.x` network used by the portal.
- WiFi can steal traffic that must go out through Ethernet.

If a change touches portal access, DNS, routing, or interface binding, verify that it still respects these facts.

## Codebase map

Current structure:

```text
main.go
internal/
  creds/
  detect/
  fix/
  installer/
  login/
  manual/
  service/
bootstrap.sh
bootstrap.ps1
install.sh
README.md
CHANGELOG.md
RELEASING.md
AGENTS.md
```

Module responsibilities:

- `main.go`: CLI entry point and command dispatch
- `internal/login`: captive portal flow and login loop
- `internal/detect`: OS and interface detection
- `internal/fix`: system-specific repair steps such as hosts, routing, Docker, and MAC behavior
- `internal/installer`: install wizard orchestration
- `internal/service`: service integration per platform
- `internal/creds`: encrypted credential and config storage
- `internal/manual`: man page installation and removal
- `bootstrap.sh` and `bootstrap.ps1`: bootstrap installers
- `install.sh`: legacy bash implementation kept for reference

## Expectations for all changes

Every change should meet these bars:

- Prefer small, understandable changes over clever ones
- Preserve cross-platform behavior unless the change is intentionally platform-specific
- Avoid adding external runtime dependencies unless there is a strong reason
- Keep the binary-first distribution model intact
- Match the existing repository tone: direct, practical, professional
- Do not write README copy that sounds generated or padded
- Do not use em dashes in project docs

## Before changing code

Start by reading the files directly related to the change. Do not guess architecture from filenames alone.

For behavior changes, check these first:

- `README.md`
- `CHANGELOG.md`
- `RELEASING.md`
- the relevant package under `internal/`

For release or install changes, also check:

- `.github/workflows/release.yml`
- `bootstrap.sh`
- `bootstrap.ps1`
- `main.go`

## Documentation rules

Documentation is part of the product.

### Update `README.md` when

- user-facing behavior changes
- installation changes
- supported platforms or architectures change
- commands or workflows change
- a new operational caveat matters to users

The README should stay useful for a real user landing on the repository for the first time.

### Update `CHANGELOG.md` when

- preparing a new release
- documenting a shipped hotfix

Do not add changelog entries for unreleased local experiments unless you are intentionally preparing that release.

### Update `RELEASING.md` when

- the release process changes
- asset names change
- verification steps change
- versioning policy changes

### Update `AGENTS.md` when

- project conventions change
- repo structure changes materially
- release or docs discipline changes
- there is a repeated mistake that future agents should avoid

## Release discipline

This repository uses `vMAJOR.MINOR.PATCH`.

- `MAJOR`: breaking change or major release line change
- `MINOR`: new feature or meaningful platform/install improvement
- `PATCH`: bug fix, release fix, small operational improvement

Release checklist:

1. Make the code changes.
2. Update `main.go` version.
3. Update `CHANGELOG.md`.
4. Update `README.md` if user-facing behavior changed.
5. Run tests.
6. Commit the release prep.
7. Create an annotated tag.
8. Push `main` and tags.
9. Verify the GitHub release and assets.

See `RELEASING.md` for the detailed command flow.

Important release rules:

- Never reuse or move a published tag
- Never create a release tag before the release commit exists
- Keep release assets, bootstrap scripts, and documented install steps aligned

## Git rules

- Use conventional commit prefixes such as `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `release:`
- Keep commits logically grouped
- Do not include unrelated files in a commit
- Do not commit scratch files, local prompts, or transient notes
- Do not rewrite published release history

## Testing expectations

Minimum expectation for most code changes:

```bash
go test ./...
```

If the environment blocks the default Go build cache:

```bash
GOCACHE=/tmp/go-build-cache go test ./...
```

For docs-only changes, tests are optional unless the docs describe changed behavior that should be verified.

For release-related changes, verify at least:

- Go tests pass
- release workflow still matches expected asset names
- bootstrap scripts still point to the right release assets

## Man page rules

- The man page source lives under `internal/manual/`
- If commands, install behavior, or service behavior changes, update the man page too
- If the man page is installed by setup, `uninstall` must remove it
- Keep the man page concise and accurate

## Bootstrap rules

The bootstrap scripts are part of the release contract.

- Prefer downloading a released binary first
- Use source build only as fallback
- Keep binary names aligned with the release workflow
- Do not silently change install locations without updating docs

## Security rules

- Never log credentials
- Never store plaintext credentials
- Keep credential handling local to the machine
- Do not add telemetry
- Be careful with TLS changes because the captive portal uses a self-signed certificate

## Platform-specific guardrails

### Linux

- Systemd user service is the current supported service path
- Keep `man iitj-login` working when install succeeds
- Be careful with `/etc/hosts`, route changes, and NetworkManager interaction

### macOS

- Launchd agent is the current service path
- Keep user-level installation behavior

### Windows

- Task Scheduler is the current service path
- Keep the bootstrap and binary naming aligned with the release workflow

## What not to do

- Do not replace accurate technical wording with marketing fluff
- Do not add vague AI-style README filler
- Do not change release flow casually
- Do not break legacy references unless intentionally removing them
- Do not leave docs stale after changing user-facing behavior

## If you are an LLM making a change

Follow this order:

1. Read the relevant code.
2. Make the smallest correct change.
3. Update docs that are now stale.
4. If this is a release, update version metadata and changelog.
5. Run verification.
6. Keep the commit clean.

When in doubt, choose clarity over novelty.
