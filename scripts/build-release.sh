#!/usr/bin/env bash
set -euo pipefail
version="${1:?usage: build-release.sh vMAJOR.MINOR.PATCH[-suffix]}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
  echo 'Invalid release version' >&2
  exit 2
fi
mkdir -p dist
for target_os in darwin linux; do
  for target_arch in amd64 arm64; do
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
      go build -trimpath -ldflags "-s -w -X main.version=$version" \
      -o "dist/gh-mutual-follow_${version}_${target_os}-${target_arch}" .
  done
done
(
  cd dist
  shasum -a 256 "gh-mutual-follow_${version}_"* > checksums.txt
)
