---
name: release
description: "Prepare and execute a RepoBird CLI release: pre-flight checks, chlog stamping, git tagging, GitHub Actions/Goreleaser publishing, and post-release verification."
---

# RepoBird CLI Release Workflow

Use this private skill when asked to release RepoBird CLI, prepare a version, tag a release, publish GitHub artifacts, or recover from failed release CI.

User instructions for the current release take precedence over this workflow. Preserve the safety rules: do not use `git stash`, do not use `git add -A` or `git add .`, and do not rewrite shared branches unless the user explicitly asks.

## Repo Shape

- GitLab remote: `glab` (`git@gitlab.com:ariel-frischer/repobird-cli.git`)
- GitHub remote: `gh` (`https://github.com/RepoBird/repobird-cli.git`), but confirm with `git remote -v`
- Main development branch: `dev`
- Release branch: `main`
- GitHub release target in GoReleaser: `RepoBird/repobird-cli`
- Binary name: `repobird`
- Version source: `VERSION`
- Changelog source: `CHANGELOG.yaml`, rendered to `CHANGELOG.md` with `chlog`
- Release publishing: GitHub Actions release workflow and/or local GoReleaser scripts

Remote policy:

- Push both `dev` and `main` to `glab` (GitLab).
- Push only `main` and release tags to `gh` (GitHub). Never push `dev` to GitHub.
- If no GitHub remote is configured, do not invent one during release work; report that GitHub publishing is pending remote setup.

Branch content policy:

- `.agents/` and similar local agent workflow folders are allowed to be tracked on `dev`.
- `.agents/` must not be present, staged, or tracked on `main`.
- Do not add `.agents/` to a global repo ignore as the fix; this folder may be intentionally versioned on `dev`.
- Before committing or pushing `main`, verify:
  ```bash
  git status --short -- .agents
  git ls-tree -r --name-only HEAD -- .agents
  ```
  Both commands must produce no `.agents` paths on `main`.

## Preflight

Run these before release work. Stop and fix release blockers before tagging.

```bash
git status --short --branch
git remote -v
git fetch glab --prune --tags
gh auth status
glab auth status
make fmt-check
make vet
make lint
make test
chlog validate
chlog check
make build
```

If the working tree has unrelated local changes, do not overwrite them. Work around them or ask before proceeding.

`gh auth status` may fail when the GitHub remote has not been configured yet. Treat that as a GitHub publishing blocker, not a GitLab release blocker, unless the user explicitly requires GitHub publishing for the release.

## Determine Version

Check the current version and pending changelog:

```bash
cat VERSION
git tag --sort=-v:refname | head -10
chlog show unreleased
```

Version guide:

- Patch (`0.0.x`): fixes, docs, small behavior improvements.
- Minor (`0.x.0`): user-facing features or meaningful non-breaking behavior changes.
- Major (`x.0.0`): breaking CLI/API behavior.

Tags should use the `vX.Y.Z` form. `chlog release` expects `X.Y.Z`.

## Prepare Release Commit

1. Curate `CHANGELOG.yaml`. Move user-facing unreleased entries into the target version.
2. Stamp the changelog:
   ```bash
   chlog release X.Y.Z
   chlog sync
   chlog check
   ```
3. Update `VERSION` to `X.Y.Z` if needed.
4. Re-run the focused release gate:
   ```bash
   make fmt-check
   make vet
   make lint
   make test
   make build
   ```
5. Commit only the release files:
   ```bash
   git status --short
   git add CHANGELOG.yaml CHANGELOG.md VERSION
   git commit -m "release: vX.Y.Z"
   ```

## Merge To Main

Release from `main` unless the user explicitly asks for another branch.

```bash
git switch dev
git pull --ff-only glab dev
git switch main
git pull --ff-only glab main
git merge --no-ff --no-commit dev
if git diff --cached --name-only -- .agents | grep -q .; then
  git rm -r --cached .agents
  rm -rf .agents
fi
git diff --cached --name-only -- .agents
git commit -m "merge dev into main for vX.Y.Z"
make fmt-check
make vet
make lint
make test
make build
chlog check
git ls-tree -r --name-only HEAD -- .agents
git push glab main
```

Do not proceed if `.agents/` appears in the staged merge diff or in `HEAD` on `main` after the guard commands. Fix the merge before pushing.

If CI fails after pushing `main`, inspect logs and fix on `dev`, then merge forward again. Avoid direct `main` hotfixes unless explicitly requested.

## Tag And Publish

Create an annotated tag from `main`:

```bash
git switch main
git pull --ff-only glab main
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push glab vX.Y.Z
```

Push only the same `main` commit and tag to GitHub so GitHub Actions can publish:

```bash
git push gh main
git push gh vX.Y.Z
```

Use the actual GitHub remote name from `git remote -v`; the expected name is `gh`. Do not push `dev` to GitHub.

Watch GitHub Actions when available:

```bash
gh run list --branch main --limit 10
gh run watch
gh release view vX.Y.Z --json tagName,url,body,assets
```

If GitHub Actions is unavailable, use the local release path only after confirming the required credentials:

```bash
chlog extract X.Y.Z > .release/notes.md
GITLAB_TOKEN="" goreleaser release --clean --release-notes=.release/notes.md
```

For a dry run:

```bash
goreleaser release --clean --snapshot --skip=publish
```

## Post-release

```bash
git tag --sort=-v:refname | head -3
make build
./build/repobird version
gh release view vX.Y.Z
git switch dev
git merge main
git push glab dev
```

Final handoff should include the release version, main SHA, CI/release URL if available, release URL if available, and any follow-up Beads.
