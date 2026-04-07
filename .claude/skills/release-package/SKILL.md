---
name: release-package
description: Release a new version of an ins* package following the multi-module release process
---

## Instructions

Release package: $ARGUMENTS

First, gather information:
1. Identify the package to release from the arguments
2. Run `git log $(git describe --tags --match "<package>/v*" --abbrev=0 2>/dev/null)..HEAD -- <package>/` to see changes since last release
3. Run `git tag -l "<package>/v*" | sort -V | tail -5` to check recent version tags
4. Check if this package has dependents that need updating:
   - `inssqs` depends on `insdash`, `inslogger`
   - `insssm` depends on `inscacheable`

Then follow the release process:

1. **Determine version bump**
   - **Patch** (v1.0.X): Bug fixes, internal changes
   - **Minor** (v1.X.0): New features, backward-compatible additions
   - **Major** (vX.0.0): Breaking API changes

2. **Verify tests pass**
   ```bash
   cd <package>
   go mod tidy
   go test ./...
   ```

3. **Run dependency check**
   ```bash
   ./scripts/check-deps.sh
   ```

4. **Create and push tag**
   ```bash
   git tag <package>/v<version>
   git push origin <package>/v<version>
   ```

5. **Create GitHub release**
   ```bash
   gh release create <package>/v<version> --title "<package> v<version>" --notes "Description of changes"
   ```

6. **Update dependents** (if applicable)
   - If releasing `insdash` or `inslogger` → check if `inssqs` needs updating
   - If releasing `inscacheable` → check if `insssm` needs updating
   - Update `go.mod` in dependent packages and release them too

7. **Notify** — Commit message format: `<package>: release v<version>`
