#!/bin/bash

# Exit on error
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if a version was provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: No version specified${NC}"
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.5.4"
    exit 1
fi

VERSION=$1

# Validate version format
if ! [[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Invalid version format${NC}"
    echo "Version must be in the format v0.0.0"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: go.mod not found${NC}"
    echo "Please run this script from the root of the dj-set-downloader repository"
    exit 1
fi

# Check if the working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}Error: Working directory is not clean${NC}"
    echo "Please commit or stash your changes before creating a release"
    exit 1
fi

# Check if the version tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}Error: Version $VERSION already exists${NC}"
    exit 1
fi

echo -e "${GREEN}Creating release $VERSION...${NC}"

# Run tests
echo -e "${GREEN}Running tests...${NC}"
go test ./... || {
    echo -e "${RED}Tests failed${NC}"
    exit 1
}

# Create and push tag
echo -e "${GREEN}Creating and pushing tag $VERSION...${NC}"
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

# Create GitHub release using gh CLI if available
if command -v gh &> /dev/null; then
    echo -e "${GREEN}Creating GitHub release...${NC}"
    gh release create "$VERSION" \
        --title "Release $VERSION" \
        --notes "Release $VERSION of dj-set-downloader" \
        --draft
    
    echo -e "${GREEN}Release $VERSION created successfully!${NC}"
    echo "Please review and publish the draft release on GitHub:"
    echo "https://github.com/jaki95/dj-set-downloader/releases"
else
    echo -e "${GREEN}Release $VERSION created successfully!${NC}"
    echo "Please create the release on GitHub manually:"
    echo "https://github.com/jaki95/dj-set-downloader/releases/new?tag=$VERSION"
fi 