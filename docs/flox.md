# Flox Build and Publish

This document describes how kyt is built and published using [Flox](https://flox.dev), a tool for creating reproducible development environments and publishing packages to private catalogs.

## Overview

kyt uses Flox to:
- Build reproducible binaries in a sandboxed environment
- Publish packages to a private Flox catalog (`nhuray/kyt`)
- Ensure consistent builds across platforms
- Provide an alternative installation method via `flox install`

## Installation via Flox

### Prerequisites

Install Flox: https://flox.dev/docs/install-flox/

### Install kyt

```bash
# Install directly
flox install nhuray/kyt

# Or add to your Flox environment manifest
flox init
flox edit
# Add to [install] section:
# "nhuray/kyt".pkg-path = "nhuray/kyt"
flox activate
kyt version
```

### Available Platforms

The kyt package is built and published for:
- `x86_64-linux` (Linux x86_64)
- `x86_64-darwin` (macOS Intel)
- `aarch64-darwin` (macOS Apple Silicon)

## Development with Flox

### Local Development

1. **Activate the Flox environment:**
   ```bash
   cd /path/to/kyt
   flox activate
   ```

2. **Build kyt locally:**
   ```bash
   flox build kyt
   ```

3. **Test the binary:**
   ```bash
   ./result-kyt/bin/kyt version
   ./result-kyt/bin/kyt diff --help
   ```

4. **Run tests:**
   ```bash
   flox build kyt-tests
   cat ./result-kyt-tests/test-results/test.log
   ```

### Build Configuration

The Flox build configuration is defined in `.flox/env/manifest.toml`:

**Build Dependencies:**
- `go` (1.26+)
- `git` (for version extraction)
- `gnumake`
- `coreutils`

**Runtime Dependencies:**
- `tzdata` (timezone data)
- `mailcap` (MIME type mappings)
- `iana-etc` (protocol/service definitions)

**Build Process:**
1. Extracts version from git tags using `git describe`
2. Builds static binary with CGO disabled and vendored Go modules (`-mod=vendor`)
3. Embeds version info via `-ldflags`
4. Copies documentation to `$out/share/doc/kyt/`
5. Copies example config to `$out/share/examples/kyt/`

**Sandbox Mode:**
The build runs in pure sandbox mode (`sandbox = "pure"`) for maximum reproducibility. Only explicitly declared dependencies are available during the build.

## Publishing (Maintainers)

### Automatic Publishing

When a new GitHub release is published (e.g., `v1.2.3`), the workflow `.github/workflows/publish-flox.yml` automatically:

1. Builds kyt on 3 platforms (Linux x64, macOS Intel, macOS ARM64)
2. Tests the binary (`kyt version`)
3. Publishes to the `nhuray` catalog

This runs **after** the GitHub release is created, independently from the GoReleaser workflow.

### Manual Publishing

To publish manually (for testing or patches):

```bash
# Ensure you're on a clean git state
git status

# Authenticate to FloxHub
flox auth login --token YOUR_FLOXHUB_TOKEN

# Build locally
flox build kyt

# Publish to catalog
flox publish -o nhuray kyt
```

### Requirements for Publishing

Flox requires:
- Clean git working tree (no uncommitted changes)
- Current commit pushed to a remote
- All build files tracked by git

### Version Management

Version is extracted from git tags:
- Use semantic versioning: `v1.2.3`
- Pre-releases work: `v1.2.3-rc.1`, `v1.2.3-beta.2`
- The build script uses `git describe --tags --always --dirty`

## Troubleshooting

### Build Fails with "Unexpected dependencies found"

If you see errors about unexpected dependencies in the build output, you need to add them to both:
1. `[install]` section (so they're available in the environment)
2. `runtime-packages` array in `[build.kyt]` (so they're included in the package closure)

Example:
```toml
[install]
new-dep.pkg-path = "new-dep"

[build.kyt]
runtime-packages = ["tzdata", "mailcap", "iana-etc", "new-dep"]
```

### Git Working Tree Not Clean

Flox requires a clean git state. Ensure all changes are committed and pushed:

```bash
git status
git add .
git commit -m "Your changes"
git push
```

### Authentication Failed

If `flox auth login` fails:
1. Check your FloxHub token is valid
2. Verify token has `publish` permissions
3. Generate a new token at https://hub.flox.dev

### Package Already Published

You cannot re-publish the same version. Either:
1. Delete the version from FloxHub (via web interface)
2. Create a new version with a bumped tag

## Architecture

### Build Output Structure

After `flox build kyt`, the output structure follows the Filesystem Hierarchy Standard (FHS):

```
./result-kyt/
├── bin/
│   └── kyt                          # Main binary
└── share/
    ├── doc/kyt/                     # Documentation
    │   ├── README.md
    │   ├── LICENSE
    │   ├── cluster-comparison.md
    │   ├── diff.md
    │   └── fmt.md
    └── examples/kyt/                # Example config
        └── example-config.yaml
```

### Runtime Closure

When you `flox install nhuray/kyt`, Flox:
1. Downloads the package from your private catalog
2. Materializes the runtime closure (kyt + dependencies)
3. Creates a symlink forest under the environment
4. Makes `kyt` available in `$PATH`

All dependencies (tzdata, mailcap, iana-etc) are included automatically.

## References

- [Flox Documentation](https://flox.dev/docs)
- [Flox Build Guide](https://flox.dev/docs/concepts/manifest-builds/)
- [Flox Publish Guide](https://flox.dev/docs/concepts/publishing/)
- [Introducing Flox Build and Publish](https://flox.dev/blog/introducing-flox-build-and-publish/)
