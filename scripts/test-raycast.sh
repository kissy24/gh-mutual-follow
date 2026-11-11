#!/bin/bash
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
task_dir="$(mktemp -d)"
trap 'rm -rf "$task_dir"' EXIT

# Only this mock is invoked; no GitHub account or credentials are accessed.
cat > "$task_dir/gh" <<'MOCK'
#!/bin/bash
printf '%s\n' "$*" >> "$TEST_CALLS"
echo "mock result: $2"
case "$2" in
  unfollow) exit "$TEST_UNFOLLOW_STATUS" ;;
  follow-back) exit "$TEST_FOLLOW_STATUS" ;;
  *) exit 99 ;;
esac
MOCK
chmod +x "$task_dir/gh"
export PATH="$task_dir:/usr/bin:/bin"
export TEST_CALLS="$task_dir/calls"

for scenario in success unfollow_failure follow_failure; do
  export TEST_UNFOLLOW_STATUS=0 TEST_FOLLOW_STATUS=0
  expected_status=0
  printf '%s\n' 'mutual-follow unfollow --all --yes' > "$task_dir/expected"
  case "$scenario" in
    unfollow_failure) TEST_UNFOLLOW_STATUS=7; expected_status=7 ;;
    follow_failure) TEST_FOLLOW_STATUS=9; expected_status=9 ;;
  esac
  if [[ "$scenario" != unfollow_failure ]]; then
    printf '%s\n' 'mutual-follow follow-back --all --yes' >> "$task_dir/expected"
  fi
  : > "$TEST_CALLS"
  status=0
  /bin/bash "$script_dir/raycast/github-mutual-follow.sh" > "$task_dir/output" 2>&1 || status=$?
  if [[ "$status" != "$expected_status" ]]; then
    echo "FAIL: $scenario: expected exit $expected_status, got $status" >&2
    cat "$task_dir/output" >&2
    exit 1
  fi
  diff -u "$task_dir/expected" "$TEST_CALLS"
  grep -q 'mock result: unfollow' "$task_dir/output"
  if [[ "$scenario" != unfollow_failure ]]; then
    grep -q 'mock result: follow-back' "$task_dir/output"
  fi
  echo "PASS: $scenario"
done
