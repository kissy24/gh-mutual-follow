#!/usr/bin/env bash
set -euo pipefail

tag=${1:?Usage: publish-release.sh TAG}
# A retry must preserve a release that was already published successfully.
if gh release view "$tag" --json tagName >/dev/null 2>&1; then
  echo "Release $tag already exists; preserving the published release."
  exit 0
fi

flags=(--verify-tag --generate-notes)
if [[ "$tag" == *-* ]]; then
  flags+=(--prerelease)
fi
gh release create "$tag" dist/* "${flags[@]}"
