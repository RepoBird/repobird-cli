# RepoBird CLI Examples

This directory contains example configurations for RepoBird CLI, demonstrating various use cases and features.

## Directory Structure

```
examples/
├── single-runs/           # Individual run configurations
├── continuous-tasks/      # Tasks that always produce changes
└── README.md             # This file
```

## Single Run Configurations

Located in `single-runs/`, these demonstrate individual run configurations:

- `test-improvement.json` - Ruby test coverage improvements
- `refactor-code.yaml` - Code refactoring and optimization
- `security-audit.md` - Security audit and improvements  
- `documentation.json` - Documentation improvements

**Usage:**
```bash
repobird run single-runs/test-improvement.json
repobird run single-runs/refactor-code.yaml
```

## Continuous Tasks

Located in `continuous-tasks/`, these **always produce changes** and never reach a "complete" state:

- `alphabet-cycle.json` - Cycles A→B→C→...→Z→A (always has next letter)
- `counter-increment.yaml` - Increments counter infinitely (1→2→3→...)
- `timestamp-log.md` - Adds timestamp entries (always unique)
- `fibonacci-sequence.json` - Generates Fibonacci numbers (infinite sequence)
- `prime-numbers.yaml` - Generates prime numbers (infinite sequence)
- `word-chain.md` - Creates word chains (RUBY→YAML→LOGIC→...)

**Usage:**
```bash
repobird run continuous-tasks/alphabet-cycle.json
repobird run continuous-tasks/counter-increment.yaml
```

## Repository Configuration

These examples use placeholder repository names. Replace them with your actual repository:
- **Repository:** `your-org/your-repo` (update in each example file)
- **Source Branch:** Your default branch (usually `main` or `master`)
- **Target Branches:** Feature branches for the changes

## Usage Examples

### 1. Continuous Tasks
Run tasks that continuously generate changes (useful for testing and demonstrations):

```bash
# These can be run indefinitely and will always make changes
repobird run continuous-tasks/alphabet-cycle.json --follow
repobird run continuous-tasks/counter-increment.yaml --follow
```

### 2. Different File Formats
RepoBird supports multiple configuration formats:

```bash
# JSON format
repobird run single-runs/test-improvement.json

# YAML format
repobird run single-runs/refactor-code.yaml

# Markdown format
repobird run single-runs/security-audit.md
```

## Supported Features

- ✅ **JSON, YAML, and Markdown formats**
- ✅ **Single run configurations**
- ✅ **Continuous tasks that never complete**
- ✅ **Mathematical sequences (Fibonacci, primes, powers)**
- ✅ **Text sequences (alphabet, word chains)**
- ✅ **Counters and timestamps**
- ✅ **Consistent repository targeting**
- ✅ **Various branch targeting strategies**

## About Continuous Tasks

The continuous tasks demonstrate RepoBird's ability to handle iterative improvements and ongoing maintenance:

1. **Iterative Development** - Tasks that build upon previous results
2. **Unique Changes** - Each run produces distinct modifications
3. **Long-running Operations** - Demonstrates handling of extended workflows
4. **Automation Testing** - Useful for testing CI/CD integrations

These examples show how RepoBird can handle both one-time fixes and ongoing development tasks.
