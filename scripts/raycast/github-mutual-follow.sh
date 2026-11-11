#!/bin/bash

# @raycast.schemaVersion 1
# @raycast.title GitHub Mutual Follow
# @raycast.mode fullOutput
# @raycast.packageName GitHub
# @raycast.description Unfollow non-followers, then follow back all followers without confirmation.
# @raycast.icon 🔄

set -euo pipefail

# Raycast does not load interactive shell configuration. Support Homebrew gh.
export PATH="${PATH:-/usr/bin:/bin}:/opt/homebrew/bin:/usr/local/bin"

echo 'Unfollowing accounts that do not follow you back...'
gh mutual-follow unfollow --all --yes

echo 'Following back your followers...'
gh mutual-follow follow-back --all --yes
