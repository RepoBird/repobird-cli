# Language And Framework Recipes

These recipes provide starting prompts and validation commands for common RepoBird tasks. Replace repository names, package managers, and command names with the project's actual conventions.

## Universal Maintenance Prompt

```markdown
---
repository: owner/repo
baseBranch: main
outputMode: pull_request
prTargetBranch: main
title: Fix repository maintenance checks
---

Fix the requested maintenance issue.

Scope:
- Prefer the repository's existing commands and documented workflows.
- Keep behavior unchanged unless a failing test proves behavior is wrong.
- Run focused validation first, then broader checks when practical.
- Include changed files, commands run, and residual risk in the PR description.
```

## Go

Use for Go modules, CLIs, services, and libraries.

```markdown
Fix Go formatting, vet, lint, and unit-test failures.

Expected commands:
- gofmt -w on changed Go files
- go test ./... or focused package tests
- go vet ./... when the repository uses it
- golangci-lint run when configured

Preserve public CLI/API behavior unless the failing test demonstrates a bug.
```

CLI:

```bash
repobird run -r owner/repo -p @go-maintenance.md --wait --json
```

## Node, TypeScript, React, Next.js, Vue, Svelte

```markdown
Fix TypeScript, lint, formatting, and unit-test failures.

Expected commands:
- npm/pnpm/yarn install command already used by the repository
- typecheck script if present
- lint script if present
- format or prettier check if present
- test script or focused test command

Respect the existing framework conventions. Do not introduce a new state manager,
CSS framework, test runner, or formatter unless the project already uses it.
```

CI-friendly command:

```bash
repobird run -r owner/webapp \
  -p @frontend-maintenance.md \
  --idempotency-key "frontend-${GITHUB_RUN_ID:-local}" \
  --wait \
  --json
```

## Python, Django, FastAPI

```markdown
Fix Python lint, formatting, type-check, and test failures.

Expected commands:
- pytest or the repository's focused test command
- ruff check and ruff format when configured
- black, isort, mypy, or pyright only when already used
- framework checks such as python manage.py check for Django

Preserve migrations unless the task explicitly requires schema changes.
```

## Ruby, Rails

```markdown
Fix Ruby lint, formatting, and test failures.

Expected commands:
- bundle exec rspec or focused test files
- bundle exec rubocop when configured
- bin/rails test or bin/rails test:system when used by the repo

Do not change database migrations unless the failure is migration-related.
```

## Java, Kotlin, Spring, Android

```markdown
Fix Java/Kotlin build, formatting, lint, and test failures.

Expected commands:
- ./gradlew test or ./mvnw test
- ./gradlew check when practical
- Android lint/unit tests when this is an Android project

Preserve package structure and dependency injection conventions.
```

## .NET, C#

```markdown
Fix .NET build, formatting, analyzer, and test failures.

Expected commands:
- dotnet restore
- dotnet build
- dotnet test
- dotnet format when configured

Keep public contracts stable unless tests clearly require a behavior fix.
```

## Rust

```markdown
Fix Rust formatting, clippy, and test failures.

Expected commands:
- cargo fmt
- cargo clippy --all-targets --all-features
- cargo test

Prefer small, idiomatic changes. Avoid broad lifetime or trait refactors unless required.
```

## PHP, Laravel, Symfony

```markdown
Fix PHP lint, formatting, static-analysis, and test failures.

Expected commands:
- composer install when needed
- vendor/bin/phpunit or pest
- phpstan/psalm when configured
- pint/php-cs-fixer when configured
- framework-specific checks for Laravel or Symfony

Do not change migrations, seeders, or config defaults unless the task requires it.
```

## Mobile

React Native:

```markdown
Fix React Native lint, type-check, and unit-test failures. Preserve native project
configuration unless the failing check is native-build related. Run JS tests first,
then platform-specific checks when configured.
```

iOS:

```markdown
Fix Swift/SwiftUI build, formatting, and test failures. Preserve project settings
unless the Xcode build failure points to them. Run focused XCTest targets when practical.
```

Android:

```markdown
Fix Android build, lint, and unit-test failures. Preserve Gradle structure and
module boundaries. Run focused module tests before full checks.
```

## Documentation Update

```markdown
Update repository documentation for the current behavior.

Check:
- README quick start and examples
- docs/ guides and command references
- generated API docs or OpenAPI docs when present
- changelog or release notes if the repository uses them

Do not invent unsupported commands. Verify examples against local help, tests,
or source files where practical.
```

## Test Improvement

```markdown
Improve unit-test coverage for the specified module.

Constraints:
- Add focused tests for real edge cases.
- Avoid snapshot churn unless snapshots are the repository norm.
- Do not weaken assertions to make tests pass.
- Run the focused test package before broader tests.
```

## Dependency Update Fallout

```markdown
Fix failures caused by a dependency update.

Steps:
- Identify the breaking API or behavior change.
- Update code using the repository's current patterns.
- Avoid broad dependency churn.
- Run focused tests for affected modules and the dependency update checks.
```
