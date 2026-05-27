#!/bin/bash
set -e

# AnyAgent Release Script
# Usage: ./scripts/release.sh [major|minor|patch]

BUMP_TYPE="${1:-patch}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Get current version from git tags
get_current_version() {
    local tag
    tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo "${tag#v}"
}

# Bump version
bump_version() {
    local current="$1"
    local major minor patch

    IFS='.' read -r major minor patch <<< "$current"

    case "$BUMP_TYPE" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo -e "${RED}Invalid bump type: ${BUMP_TYPE}${NC}"
            echo "Usage: $0 [major|minor|patch]"
            exit 1
            ;;
    esac

    echo "${major}.${minor}.${patch}"
}

# Update version in files
update_version_files() {
    local version="$1"

    echo "Updating version to ${version}..."

    # CLI Cargo.toml
    sed -i.bak "s/^version = \".*\"/version = \"${version}\"/" "${REPO_ROOT}/cli/Cargo.toml"
    rm -f "${REPO_ROOT}/cli/Cargo.toml.bak"

    # MCP Server package.json
    sed -i.bak "s/\"version\": \".*\"/\"version\": \"${version}\"/" "${REPO_ROOT}/mcp-server/package.json"
    rm -f "${REPO_ROOT}/mcp-server/package.json.bak"

    # Homebrew formula
    sed -i.bak "s/version \".*\"/version \"${version}\"/" "${REPO_ROOT}/dist/homebrew/agentx.rb"
    rm -f "${REPO_ROOT}/dist/homebrew/agentx.rb.bak"

    echo -e "${GREEN}✓ Version files updated${NC}"
}

# Generate changelog
generate_changelog() {
    local version="$1"
    local prev_tag
    prev_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

    echo "## v${version}" > /tmp/changelog_entry.md
    echo "" >> /tmp/changelog_entry.md
    echo "### Changes" >> /tmp/changelog_entry.md
    echo "" >> /tmp/changelog_entry.md

    if [ -n "$prev_tag" ]; then
        git log "${prev_tag}..HEAD" --pretty=format:"- %s" --no-merges >> /tmp/changelog_entry.md
    else
        git log --pretty=format:"- %s" --no-merges >> /tmp/changelog_entry.md
    fi

    echo "" >> /tmp/changelog_entry.md
    echo "" >> /tmp/changelog_entry.md

    # Prepend to CHANGELOG.md
    if [ -f "${REPO_ROOT}/CHANGELOG.md" ]; then
        cat /tmp/changelog_entry.md "${REPO_ROOT}/CHANGELOG.md" > /tmp/changelog_full.md
        mv /tmp/changelog_full.md "${REPO_ROOT}/CHANGELOG.md"
    else
        cp /tmp/changelog_entry.md "${REPO_ROOT}/CHANGELOG.md"
    fi

    rm -f /tmp/changelog_entry.md

    echo -e "${GREEN}✓ Changelog generated${NC}"
}

main() {
    cd "$REPO_ROOT"

    # Check for clean working directory
    if [ -n "$(git status --porcelain)" ]; then
        echo -e "${RED}Working directory is not clean. Commit or stash changes first.${NC}"
        exit 1
    fi

    # Get versions
    local current_version new_version
    current_version=$(get_current_version)
    new_version=$(bump_version "$current_version")

    echo -e "${YELLOW}Current version: ${current_version}${NC}"
    echo -e "${GREEN}New version: ${new_version}${NC}"
    echo ""

    # Confirm
    read -p "Create release v${new_version}? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi

    # Update version files
    update_version_files "$new_version"

    # Generate changelog
    generate_changelog "$new_version"

    # Commit changes
    git add -A
    git commit -m "release: v${new_version}"

    # Create tag
    git tag -a "v${new_version}" -m "Release v${new_version}"

    echo ""
    echo -e "${GREEN}✓ Release v${new_version} prepared${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Review changes: git log --oneline -5"
    echo "  2. Push to remote: git push origin main --tags"
    echo "  3. GitHub Actions will build and publish automatically"
    echo ""
    echo "Or to undo:"
    echo "  git reset --hard HEAD~1"
    echo "  git tag -d v${new_version}"
}

main "$@"
