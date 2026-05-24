#!/usr/bin/env bash
set -euo pipefail

BRANCH="${1:?Usage: worktree-setup.sh <branch-name> [base-branch]}"
BASE="${2:-HEAD}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKTREE_DIR="$REPO_ROOT/.worktrees/$BRANCH"

sync_agent_context() {
  local src_root="$1"
  local dst_root="$2"
  local path

  for path in skills .agents .opencode .claude; do
    [ -e "$src_root/$path" ] || continue
    if [ -e "$dst_root/$path" ]; then
      echo "agent context exists, skipping: $path" >&2
      continue
    fi

    echo "syncing agent context: $path" >&2
    if command -v rsync >/dev/null 2>&1; then
      rsync -a \
        --exclude '.git' \
        --exclude '.beads' \
        --exclude '.env' \
        --exclude '.env.*' \
        --exclude 'node_modules' \
        --exclude '.venv' \
        --exclude 'venv' \
        --exclude 'dist' \
        --exclude 'build' \
        --exclude 'target' \
        --exclude '.cache' \
        --exclude '.pytest_cache' \
        --exclude '.mypy_cache' \
        --exclude '.ruff_cache' \
        "$src_root/$path" "$dst_root/"
    else
      cp -R "$src_root/$path" "$dst_root/"
    fi
  done
}

link_beads_db() {
  local src_root="$1"
  local dst_root="$2"

  [ -e "$src_root/.beads" ] || return 0

  if [ -e "$dst_root/.beads" ] || [ -L "$dst_root/.beads" ]; then
    echo "beads database exists, preserving: $dst_root/.beads" >&2
    return 0
  fi

  echo "linking canonical beads database: .beads" >&2
  ln -s "$src_root/.beads" "$dst_root/.beads"
}

cd "$REPO_ROOT"

if [ -d "$WORKTREE_DIR" ]; then
  link_beads_db "$REPO_ROOT" "$WORKTREE_DIR"
  sync_agent_context "$REPO_ROOT" "$WORKTREE_DIR"
  echo "$WORKTREE_DIR"
  exit 0
fi

git worktree add "$WORKTREE_DIR" -b "$BRANCH" "$BASE" 2>/dev/null || \
  git worktree add "$WORKTREE_DIR" "$BRANCH"

link_beads_db "$REPO_ROOT" "$WORKTREE_DIR"
sync_agent_context "$REPO_ROOT" "$WORKTREE_DIR"

cd "$WORKTREE_DIR"
go mod download

echo "$WORKTREE_DIR"
