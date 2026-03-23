# Releasing

## Current release flow

1. Update `CHANGELOG.md` if needed.
2. Make sure `main.go` reports the intended release version.
3. Create and push a tag:

```bash
git tag -a v4.0.0 -m "v4.0.0"
git push origin main --follow-tags
```

4. GitHub Actions builds the binaries and publishes a GitHub Release with:
   - `iitj-login-linux-amd64`
   - `iitj-login-linux-arm64`
   - `iitj-login-darwin-amd64`
   - `iitj-login-darwin-arm64`
   - `iitj-login-windows-amd64.exe`
   - `iitj-login-windows-arm64.exe`
   - `SHA256SUMS`

## Notes

- `bootstrap.sh` and `bootstrap.ps1` install the latest GitHub Release binary first.
- Source builds remain as a fallback when no matching release asset exists.
- The old script-based release history is preserved in `CHANGELOG.md`.
