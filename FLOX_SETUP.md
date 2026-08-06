# Flox Setup - Quick Reference

## What Was Done

1. **Initialized Flox environment**
   - Created `.flox/` directory with environment configuration
   - Created build manifest in `.flox/env/manifest.toml`

2. **Created build configuration**
   - Build target: `kyt` (main binary)
   - Test target: `kyt-tests` (test suite)
   - Pure sandbox mode for reproducible builds
   - Runtime dependencies: tzdata, mailcap, iana-etc

3. **Created GitHub workflow**
   - File: `.github/workflows/publish-flox.yml`
   - Triggers on release publication
   - Builds on 3 platforms: Linux x64, macOS Intel, macOS ARM64
   - Publishes to `nhuray` catalog

4. **Updated documentation**
   - Added Flox installation section to README.md
   - Created comprehensive guide in docs/flox.md

## Next Steps

### 1. Set Up FloxHub Authentication

Before you can publish, you need to:

1. Go to https://hub.flox.dev and sign in
2. Navigate to Settings → API Tokens
3. Create a new token with `publish` permissions
4. Copy the token

### 2. Add GitHub Secret

1. Go to your GitHub repository: https://github.com/nhuray/kyt
2. Navigate to Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `FLOXHUB_TOKEN`
5. Value: (paste the token from FloxHub)
6. Click "Add secret"

### 3. Test with a Pre-release (Recommended)

Before doing a production release, test the workflow:

```bash
# Create a pre-release tag
git tag v0.1.0-rc.1
git push origin v0.1.0-rc.1

# Create a pre-release on GitHub
gh release create v0.1.0-rc.1 \
  --prerelease \
  --title "Test Release for Flox" \
  --notes "Testing Flox publish workflow"
```

This will trigger both workflows:
- `release.yml` - Creates GitHub release with GoReleaser
- `publish-flox.yml` - Publishes to Flox catalog

### 4. Verify the Package

After the workflow completes, verify the package was published:

```bash
# Search for your package
flox search nhuray/kyt

# Show package info (should list platforms)
flox show nhuray/kyt

# Install and test
mkdir test-install && cd test-install
flox init
flox install nhuray/kyt
flox activate
kyt version
```

### 5. Production Release

Once testing is successful, proceed with normal releases:

```bash
# Tag and push
git tag v1.0.0
git push origin v1.0.0

# Create release
gh release create v1.0.0 --generate-notes
```

The Flox publish workflow will run automatically.

## Files Created/Modified

```
.flox/                                    # Flox environment (NEW)
├── .gitattributes
├── .gitignore
├── env.json
├── env/
│   ├── manifest.toml                    # Build configuration
│   └── manifest.lock
└── run/

.github/workflows/
└── publish-flox.yml                     # Flox publish workflow (NEW)

docs/
└── flox.md                              # Comprehensive Flox documentation (NEW)

.gitignore                               # Updated to ignore result-*
README.md                                # Added Flox installation section
```

## Local Testing Commands

```bash
# Test build
flox build kyt

# Test binary
./result-kyt/bin/kyt version
./result-kyt/bin/kyt diff --help

# Run tests
flox build kyt-tests
cat ./result-kyt-tests/test-results/test.log

# Clean build outputs
rm -rf result-*
```

## Troubleshooting

### "Git working tree is not clean"
Commit and push all changes before publishing:
```bash
git add .
git commit -m "Your changes"
git push
```

### "Package not found in '[install]' section"
Add the package to both `[install]` and `runtime-packages` in manifest.toml

### "Authentication failed"
Verify your FLOXHUB_TOKEN is correct and has publish permissions

## Platform Support

Currently building for:
- ✅ x86_64-linux (Ubuntu)
- ✅ x86_64-darwin (macOS Intel)
- ✅ aarch64-darwin (macOS Apple Silicon)
- ⏸️  aarch64-linux (Skipped - requires QEMU or self-hosted runner)

To add ARM64 Linux later, uncomment the matrix entry in the workflow and add QEMU setup.
