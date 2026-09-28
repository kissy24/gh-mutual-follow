#!/usr/bin/env bash
# Use an isolated gh configuration; never touch the user's installed extensions.
set -euo pipefail
task_config_dir="$(mktemp -d)"
trap 'rm -rf "$task_config_dir"' EXIT
export GH_CONFIG_DIR="$task_config_dir"
export GH_TOKEN=local-offline-test
unset GITHUB_TOKEN GH_DEBUG GH_HOST GH_ENTERPRISE_TOKEN GITHUB_ENTERPRISE_TOKEN
export GH_PROMPT_DISABLED=1 GH_NO_UPDATE_NOTIFIER=1 GH_NO_EXTENSION_UPDATE_NOTIFIER=1
gh extension install .
gh mutual-follow --help
gh mutual-follow --version
gh extension remove mutual-follow
