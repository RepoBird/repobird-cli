# RepoBird CLI Test Tasks

This directory contains organized test configurations for RepoBird CLI bulk runs implementation.

## Directory Structure

```
run-tasks/
├── single-runs/           # Individual run configurations
├── bulk-runs/             # Bulk/batch run configurations  
├── continuous-tasks/      # Tasks that always produce changes
└── README.md             # This file
```

## Single Run Configurations

Located in `single-runs/`, these test individual run functionality:

- `test-improvement.json` - Ruby test coverage improvements
- `refactor-code.yaml` - Code refactoring and optimization
- `security-audit.md` - Security audit and improvements  
- `documentation.json` - Documentation improvements

**Usage:**
```bash
repobird run single-runs/test-improvement.json
repobird run single-runs/refactor-code.yaml
```

## Bulk Run Configurations

Located in `bulk-runs/`, these test bulk/batch functionality:

- `comprehensive-improvements.json` - Multiple quality improvements
- `quality-improvements.yaml` - Code quality focused tasks
- `feature-additions.jsonl` - New feature additions
- `maintenance-tasks.md` - Maintenance and updates

**Usage:**
```bash
repobird bulk bulk-runs/comprehensive-improvements.json
repobird bulk bulk-runs/quality-improvements.yaml
```

## Continuous Tasks

Located in `continuous-tasks/`, these **always produce changes** and never reach a "complete" state:

### Single Continuous Tasks
- `alphabet-cycle.json` - Cycles A→B→C→...→Z→A (always has next letter)
- `counter-increment.yaml` - Increments counter infinitely (1→2→3→...)
- `timestamp-log.md` - Adds timestamp entries (always unique)
- `fibonacci-sequence.json` - Generates Fibonacci numbers (infinite sequence)
- `prime-numbers.yaml` - Generates prime numbers (infinite sequence)
- `word-chain.md` - Creates word chains (RUBY→YAML→LOGIC→...)

### Bulk Continuous Tasks
- `multi-continuous.json` - Combines alphabet, counter, and Fibonacci
- `sequence-tasks.jsonl` - Multiple mathematical sequences

**Usage:**
```bash
# Single continuous tasks
repobird run continuous-tasks/alphabet-cycle.json
repobird run continuous-tasks/counter-increment.yaml

# Bulk continuous tasks  
repobird bulk continuous-tasks/multi-continuous.json
repobird bulk continuous-tasks/sequence-tasks.jsonl
```

## Repository Target

All tasks target the same repository for consistency:
- **Repository:** `support-rb/test-ruby`
- **Source Branch:** `main` 
- **Target Branches:** Various feature branches

## Testing Strategy

### 1. Continuous Testing
Run continuous tasks multiple times to verify they always produce changes:

```bash
# These can be run indefinitely and will always make changes
repobird run continuous-tasks/alphabet-cycle.json --follow
repobird run continuous-tasks/counter-increment.yaml --follow
repobird bulk continuous-tasks/multi-continuous.json --follow
```

### 2. Format Testing
Test different file formats:

```bash
# JSON format
repobird run single-runs/test-improvement.json

# YAML format  
repobird run single-runs/refactor-code.yaml

# Markdown format
repobird run single-runs/security-audit.md

# JSONL format
repobird bulk bulk-runs/feature-additions.jsonl
```

### 3. Bulk Testing
Test bulk operations with multiple runs:

```bash
# Single bulk file
repobird bulk bulk-runs/comprehensive-improvements.json

# Multiple single files combined
repobird bulk single-runs/test-improvement.json single-runs/refactor-code.yaml

# Mixed formats
repobird bulk single-runs/*.json single-runs/*.yaml
```

## Key Features Tested

- ✅ **JSON, YAML, Markdown, JSONL formats**
- ✅ **Single and bulk run configurations** 
- ✅ **Continuous tasks that never complete**
- ✅ **Mathematical sequences (Fibonacci, primes, powers)**
- ✅ **Text sequences (alphabet, word chains)**
- ✅ **Counters and timestamps**
- ✅ **Consistent repository targeting**
- ✅ **Various branch targeting strategies**

## Continuous Task Benefits

The continuous tasks solve the testing challenge of "what happens when AI has already implemented everything?" by creating tasks that:

1. **Always have work to do** - Mathematical sequences, counters, and cycles never end
2. **Produce unique changes** - Each run creates distinct modifications
3. **Never reach completion** - There's always a "next" item to generate
4. **Enable infinite testing** - Can run the same prompt thousands of times

This ensures robust testing of the bulk runs implementation under realistic conditions where the AI continuously has meaningful work to perform.