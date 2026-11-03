#!/usr/bin/env bash
set -euo pipefail
repo_dir=$(cd "$(dirname "$0")/.." && pwd)
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT
mkdir -p "$test_dir/bin" "$test_dir/dist"
touch "$test_dir/dist/asset.tar.gz"
cat > "$test_dir/bin/gh" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$RELEASE_TEST_LOG"
if [[ "$1 $2" == 'release view' ]]; then exit "${RELEASE_EXISTS_STATUS:-1}"; fi
if [[ "$1 $2" == 'release create' ]]; then exit "${RELEASE_CREATE_STATUS:-0}"; fi
exit 99
MOCK
chmod +x "$test_dir/bin/gh"
export PATH="$test_dir/bin:$PATH" RELEASE_TEST_LOG="$test_dir/calls"
cd "$test_dir"
RELEASE_EXISTS_STATUS=0 bash "$repo_dir/scripts/publish-release.sh" v1.0.0
if grep -q 'release create' "$RELEASE_TEST_LOG"; then echo 'Existing release was recreated' >&2; exit 1; fi
: > "$RELEASE_TEST_LOG"
bash "$repo_dir/scripts/publish-release.sh" v1.0.1
grep -q 'release create v1.0.1 .*--verify-tag --generate-notes' "$RELEASE_TEST_LOG"
: > "$RELEASE_TEST_LOG"
bash "$repo_dir/scripts/publish-release.sh" v1.0.2-rc.1
grep -q -- '--prerelease' "$RELEASE_TEST_LOG"
if RELEASE_CREATE_STATUS=7 bash "$repo_dir/scripts/publish-release.sh" v1.0.3; then echo 'Publication error was ignored' >&2; exit 1; fi
