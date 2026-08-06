#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if we're in the project root
if [[ ! -f "go.mod" ]] || [[ ! -f ".flox/env/manifest.toml" ]]; then
    error "This script must be run from the project root directory"
    exit 1
fi

# Check for uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
    error "You have uncommitted changes. Please commit or stash them first."
    git status --short
    exit 1
fi

# Get current branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    warning "You are not on the main branch (current: $CURRENT_BRANCH)"
    read -p "Do you want to continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        info "Release cancelled"
        exit 0
    fi
fi

# Prompt for version
echo ""
info "Current version in manifest.toml:"
grep -A 1 '^\[build\.kyt\]' .flox/env/manifest.toml | grep "^version" || echo "version not found"
echo ""

# Get latest git tag if any
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
info "Latest git tag: $LATEST_TAG"
echo ""

read -p "Enter the new version (e.g., 0.1.0, 1.0.0-rc.1): " VERSION

# Validate version format (semantic versioning)
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    error "Invalid version format. Please use semantic versioning (e.g., 1.0.0 or 1.0.0-rc.1)"
    exit 1
fi

# Add 'v' prefix for git tag
TAG="v${VERSION}"

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
    error "Tag $TAG already exists!"
    exit 1
fi

info "Version to release: $VERSION"
info "Git tag to create: $TAG"
echo ""

# Confirm
read -p "Update manifest.toml and create tag $TAG? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    info "Release cancelled"
    exit 0
fi

# Update manifest.toml
info "Updating .flox/env/manifest.toml..."
MANIFEST_FILE=".flox/env/manifest.toml"

# Use sed to update the version in [build.kyt] section
# This finds the [build.kyt] section and updates the version line that follows
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS sed
    sed -i '' "/^\[build\.kyt\]/,/^\[/ s/^version = .*/version = \"$VERSION\"/" "$MANIFEST_FILE"
else
    # Linux sed
    sed -i "/^\[build\.kyt\]/,/^\[/ s/^version = .*/version = \"$VERSION\"/" "$MANIFEST_FILE"
fi

# Verify the change
NEW_VERSION=$(grep -A 3 '^\[build\.kyt\]' "$MANIFEST_FILE" | grep "^version" | cut -d'"' -f2)
if [[ "$NEW_VERSION" != "$VERSION" ]]; then
    error "Failed to update version in manifest.toml"
    exit 1
fi

success "Updated version to $VERSION in manifest.toml"

# Update manifest.lock
info "Updating .flox/env/manifest.lock..."
LOCK_FILE=".flox/env/manifest.lock"

# Use jq to update the version in manifest.lock for build.kyt
if command -v jq >/dev/null 2>&1; then
    TMP_FILE=$(mktemp)
    jq ".manifest.build.kyt.version = \"$VERSION\"" "$LOCK_FILE" > "$TMP_FILE"
    mv "$TMP_FILE" "$LOCK_FILE"
    
    # Verify the change
    NEW_LOCK_VERSION=$(jq -r '.manifest.build.kyt.version' "$LOCK_FILE")
    if [[ "$NEW_LOCK_VERSION" != "$VERSION" ]]; then
        error "Failed to update version in manifest.lock"
        exit 1
    fi
    success "Updated version to $VERSION in manifest.lock"
else
    warning "jq not found, skipping manifest.lock update"
    warning "Install jq with: flox install jq"
fi

# Show the diff
info "Changes to manifest files:"
git diff "$MANIFEST_FILE" "$LOCK_FILE"
echo ""

# Commit the change
read -p "Commit this change? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    warning "Reverting changes..."
    git checkout "$MANIFEST_FILE" "$LOCK_FILE"
    info "Release cancelled"
    exit 0
fi

info "Committing changes..."
git add "$MANIFEST_FILE" "$LOCK_FILE"
git commit -m "chore: bump version to $VERSION"
success "Committed version bump"

# Create and push tag
echo ""
read -p "Create and push git tag $TAG? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    warning "Tag not created. You can create it manually with:"
    echo "  git tag $TAG"
    echo "  git push origin $TAG"
    exit 0
fi

info "Creating tag $TAG..."
git tag -a "$TAG" -m "Release $TAG"
success "Created tag $TAG"

# Ask about pushing
echo ""
info "Ready to push commit and tag to remote"
read -p "Push to origin? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    warning "Changes not pushed. You can push manually with:"
    echo "  git push origin $CURRENT_BRANCH"
    echo "  git push origin $TAG"
    exit 0
fi

info "Pushing commit and tag to origin..."
git push origin "$CURRENT_BRANCH"
git push origin "$TAG"

success "Successfully released version $VERSION!"
echo ""
info "Next steps:"
echo "  1. Create a GitHub release for tag $TAG"
echo "  2. The release workflow will automatically:"
echo "     - Build binaries with GoReleaser"
echo "     - Publish to Flox catalog (nhuray/kyt)"
echo ""
echo "To create the GitHub release, run:"
echo "  gh release create $TAG --generate-notes"
