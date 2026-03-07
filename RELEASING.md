# Releasing

## How to release

```bash
git tag v0.X.Y
git push origin v0.X.Y
```

That's it. The release workflow handles everything automatically:

1. **Proto tagging** - Creates a matching `proto/gen/go/v0.X.Y` tag so the Go proxy can resolve the proto module.
2. **go.mod updates** - GoReleaser before-hook runs `go get proto/gen/go@v0.X.Y` + `go mod tidy` in all consumer modules.
3. **Build** - GoReleaser cross-compiles binaries for linux/amd64 and linux/arm64.
4. **Docker** - Builds and pushes api and web images to GHCR.
5. **Packages** - Publishes DEB/RPM packages to APT/YUM repos on GitHub Pages.
6. **SDK** - Publishes Python SDK to PyPI.

## How it works

### Local development

`go.work` at the repo root resolves `proto/gen/go` from the local directory. All modules see local proto changes immediately without tagging.

### Remote installs / releases

`go install` and the Go proxy ignore `go.work`. They need a real tagged version of each module dependency. The release workflow creates matching proto tags automatically.

### CI workflows

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| `ci.yaml` | Push to main, PRs | Tests all Go modules, web, SDK |
| `release.yaml` | `v*` tags | Tags proto, builds via GoReleaser, Docker images to GHCR |
| `publish-packages.yaml` | After release | APT/YUM repos on GitHub Pages |
| `publish-sdk.yaml` | GitHub release | Python SDK to PyPI |

## go install

After a release:

```bash
go install github.com/aspectrr/fluid.sh/fluid-cli/cmd/fluid@latest
```

## Manual proto tagging (if needed)

If you need to tag proto outside of a release:

```bash
git tag proto/gen/go/v0.X.Y
git push origin proto/gen/go/v0.X.Y
```
