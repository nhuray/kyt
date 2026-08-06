# Release Process

This document describes how to create a new release of kyt using the automated release script.

## Overview

The release process is automated via `make release`, which runs the `scripts/release.sh` script. This script:

1. Prompts for the new version
2. Validates version format (semantic versioning)
3. Checks if the tag already exists
4. Updates the version in `.flox/env/manifest.toml`
5. Commits the version change
6. Creates and pushes a git tag
7. Reminds you to create a GitHub release

## Quick Start

```bash
# Run the release script
make release

# Follow the prompts:
# - Enter version (e.g., 1.0.0 or 1.0.0-rc.1)
# - Confirm changes
# - Create and push tag

# Create GitHub release (triggers workflows)
gh release create v1.0.0 --generate-notes
```

## Detailed Steps

### 1. Run the Release Script

```bash
make release
```

The script will:
- Check for uncommitted changes (must be clean)
- Show current version from manifest.toml
- Show latest git tag
- Prompt for new version

### 2. Enter Version

When prompted, enter the new version number:

```
Enter the new version (e.g., 0.1.0, 1.0.0-rc.1): 1.0.0
```

**Version format**: Must follow semantic versioning:
- `1.0.0` - Production release
- `1.0.0-rc.1` - Release candidate
- `1.0.0-beta.1` - Beta release
- `1.0.0-alpha.1` - Alpha release

### 3. Confirm Changes

The script will:
- Show the version to be set
- Show the git tag to be created (e.g., `v1.0.0`)
- Ask for confirmation

```
Version to release: 1.0.0
Git tag to create: v1.0.0

Update manifest.toml and create tag v1.0.0? [y/N]
```

### 4. Review and Commit

The script updates `.flox/env/manifest.toml` and shows the diff:

```diff
-version = "0.0.0"
+version = "1.0.0"
```

Confirm to commit:
```
Commit this change? [y/N] y
```

### 5. Create and Push Tag

The script creates the git tag and asks to push:

```
Create and push git tag v1.0.0? [y/N] y
```

Then:
```
Push to origin? [y/N] y
```

### 6. Create GitHub Release

After the script completes, create the GitHub release:

```bash
gh release create v1.0.0 --generate-notes
```

Or manually at: https://github.com/nhuray/kyt/releases/new

## What Happens After Release

Once the GitHub release is published, two workflows automatically run:

### 1. GoReleaser Workflow (`release.yml`)
- Builds binaries for all platforms
- Creates GitHub release with binaries
- Generates changelog

### 2. Flox Publish Workflow (`publish-flox.yml`)
- Builds kyt for 3 platforms (Linux x64, macOS Intel, macOS ARM64)
- Publishes to `nhuray` catalog on FloxHub
- Users can install with: `flox install nhuray/kyt`

## Example Release Session

```bash
$ make release

ℹ Current version in manifest.toml:
version = "0.0.0"

ℹ Latest git tag: none

Enter the new version (e.g., 0.1.0, 1.0.0-rc.1): 1.0.0

ℹ Version to release: 1.0.0
ℹ Git tag to create: v1.0.0

Update manifest.toml and create tag v1.0.0? [y/N] y

ℹ Updating .flox/env/manifest.toml...
✓ Updated version to 1.0.0 in manifest.toml

ℹ Changes to manifest.toml:
diff --git a/.flox/env/manifest.toml b/.flox/env/manifest.toml
-version = "0.0.0"
+version = "1.0.0"

Commit this change? [y/N] y
ℹ Committing changes...
✓ Committed version bump

Create and push git tag v1.0.0? [y/N] y
ℹ Creating tag v1.0.0...
✓ Created tag v1.0.0

ℹ Ready to push commit and tag to remote
Push to origin? [y/N] y
ℹ Pushing commit and tag to origin...
✓ Successfully released version 1.0.0!

ℹ Next steps:
  1. Create a GitHub release for tag v1.0.0
  2. The release workflow will automatically:
     - Build binaries with GoReleaser
     - Publish to Flox catalog (nhuray/kyt)

To create the GitHub release, run:
  gh release create v1.0.0 --generate-notes

$ gh release create v1.0.0 --generate-notes
✓ Created release v1.0.0
```

## Version Verification

After release, you can verify the version in builds:

```bash
# Build with Flox
flox build kyt

# Check version
./result-kyt/bin/kyt version
# Output: kyt 1.0.0
#   commit: abc1234
#   date:   2026-08-06T15:00:00Z
```

## Rollback

If you need to undo a release before pushing:

```bash
# After committing but before pushing
git reset --hard HEAD~1  # Undo commit
git tag -d v1.0.0        # Delete local tag
```

If you've already pushed, you'll need to:
1. Delete the GitHub release
2. Delete the tag from GitHub
3. Revert the commit

## Troubleshooting

### "You have uncommitted changes"
Commit or stash your changes before running `make release`.

### "Tag already exists"
The version/tag already exists. Choose a different version or delete the existing tag.

### "Invalid version format"
Use semantic versioning format: `MAJOR.MINOR.PATCH` or `MAJOR.MINOR.PATCH-prerelease`

### Want to test before releasing?
Create a pre-release:
```bash
make release  # Enter version: 1.0.0-rc.1
gh release create v1.0.0-rc.1 --prerelease
```

## Tips

- **Pre-releases**: Use `-rc.1`, `-beta.1`, `-alpha.1` suffixes for testing
- **Branch**: The script works from any branch, but warns if not on `main`
- **Testing**: Create a pre-release first to test workflows
- **Automation**: The entire process (except entering the version) can be automated in CI if needed

## See Also

- [Semantic Versioning](https://semver.org/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases)
- [Flox Documentation](https://flox.dev/docs)
