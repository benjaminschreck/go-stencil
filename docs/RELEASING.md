# Releasing go-stencil

go-stencil now uses semantic version tags for module and CLI releases.

## Versioning Policy

- Use `v0.x.y` while the public API is still evolving.
- Increment `x` for breaking changes or notable feature batches during the `v0` phase.
- Increment `y` for backwards-compatible bug fixes and small improvements.
- Reserve `v1.0.0` for the point where the public API and template behavior are intentionally stabilized.

The first tagged release should be `v0.1.0`.

## Release Process

1. Ensure the target commit on `main` is merged and all tests pass.
2. Pick the next semantic version tag, for example `v0.1.0`.
3. Create an annotated tag locally:

   ```bash
   ./scripts/release.sh v0.1.0
   ```

4. Push the tag when you are ready to publish:

   ```bash
   ./scripts/release.sh v0.1.0 --push
   ```

5. GitHub Actions will run `go test ./...`, build release binaries, and publish a GitHub Release for the tag.

## What the Release Publishes

- Go module version tags that make `go get github.com/benjaminschreck/go-stencil@v0.x.y` reproducible
- CLI binaries for Linux, macOS, and Windows
- A `checksums.txt` file for the uploaded assets
- Auto-generated GitHub release notes
