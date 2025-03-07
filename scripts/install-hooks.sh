#!/bin/bash

mkdir -p .git/hooks

ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit

echo "Git hooks installed successfully!" 