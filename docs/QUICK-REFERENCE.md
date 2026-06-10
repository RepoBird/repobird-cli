# RepoBird CLI Quick Reference

## Essential Commands

```bash
repobird tui                        # Launch interactive dashboard
repobird run task.json              # Submit task
repobird basic "Fix a bug"          # Basic run, repo auto-detected from git
repobird pro "Implement OAuth"      # Pro run, repo auto-detected from git
repobird status                     # View all runs
repobird status RUN_ID --follow     # Follow specific run
repobird logs RUN_ID                # Inspect run logs
repobird logs RUN_ID --follow       # Follow run logs as NDJSON
repobird repo show repo_123         # Inspect repository defaults
repobird config set api-key KEY     # Set API key
```

## TUI Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `Tab` | Next column |
| `↑/↓` or `j/k` | Move up/down |
| `←/→` or `l` | Move right |
| `h` | Go back (vim/ranger style) |
| `Enter` | Select & advance |

### Fuzzy Search (FZF)
| Key | Action |
|-----|--------|
| `f` | **Activate FZF mode** |
| `Type...` | Filter items |
| `↑/↓` | Navigate results |
| `Enter` | Select result |
| `ESC` | Cancel FZF |

### Actions
| Key | Action |
|-----|--------|
| `n` | New run |
| `s` | Status info |
| `r` | Refresh |
| `y` | Copy selection/field |
| `Y` | Copy all content |
| `?` | Help |
| `q` | Dashboard (from child views) / Quit (from dashboard) |
| `Q` | Force quit (from anywhere) |

## Create Run View

### Insert Mode (editing)
| Key | Action |
|-----|--------|
| `Ctrl+F` | **FZF for repository** |
| `Ctrl+R` | Repository browser |
| `Tab` | Next field |
| `Ctrl+S` | Submit |
| `ESC` | Normal mode |

### Normal Mode (vim-style)
| Key | Action |
|-----|--------|
| `f` | **FZF for repository** |
| `i` | Insert mode |
| `j/k` | Navigate fields |
| `Ctrl+S` | Submit |
| `q` or `ESC ESC` | Exit to dashboard |

## Details View Navigation

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate fields |
| `g` | Jump to first field |
| `G` | Jump to last field |
| `y` | Copy selected field |
| `Y` | Copy all content |
| `l` | Toggle logs view |
| `q` | Back to dashboard |
| `Q` | Force quit |

## FZF Mode Features

### Dashboard View
- Press `f` on any column
- Real-time fuzzy filtering
- Enter selects and advances to next column
- Works on repositories, runs, and details

### Create Run View
- `Ctrl+F` (insert) or `f` (normal) on repository field
- Shows repository history with icons:
  - 📁 Current git repository
  - 🔄 Recently used
  - ✏️ Manually entered
- Auto-saves selections to history

## Task File Format

```json
{
  "prompt": "Task description",
  "repository": "org/repo",
  "source": "main",
  "target": "feature/branch",
  "runType": "run",
  "title": "Brief title",
  "context": "Additional context",
  "files": ["file1.js", "file2.js"]
}
```

## Environment Variables

```bash
export REPOBIRD_API_KEY=<your-api-key>
export REPOBIRD_API_URL=https://repobird.ai  # Optional
export REPOBIRD_DEBUG=true                       # Debug mode
export REPOBIRD_COLOR=never                      # Disable color output
```

## Configuration File

```yaml
# ~/.config/repobird/config.yaml
api_key: <your-api-key>
api_url: https://repobird.ai
color: auto
tui:
  refresh_interval: 30s
```

## Tips

1. **Fastest way to find a run**: `f` → type → `Enter`
2. **Quick repository selection**: `n` → `Tab` → `Ctrl+F` → type → `Enter`
3. **Copy run details**: Navigate to run → `yy`
4. **Refresh dashboard**: `r` at any time
5. **Exit from anywhere**: `q` or `ESC ESC`

## Common Workflows

### Submit New Task
```bash
# CLI
repobird run task.json --follow

# TUI
1. Press 'n' for new run
2. Fill in fields (Tab to navigate)
3. Ctrl+F for repository search
4. Ctrl+S to submit
```

### Check Run Status
```bash
# CLI
repobird status RUN_ID
repobird logs RUN_ID

# TUI
1. Press 'f' to search repositories
2. Enter to see runs
3. Press 'f' to search runs
4. Enter to see details
```

### Inspect Failed Run Logs
```bash
repobird logs RUN_ID
repobird logs RUN_ID --json
repobird logs RUN_ID --follow
```

### Quick Copy Run Info
```
# In TUI
1. Navigate to run
2. Press 'y' to copy selection
3. Or 'yy' for full details
4. Or 'yp' for prompt only
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| FZF not working | Check terminal Unicode support |
| Display issues | Set `TERM=xterm-256color` |
| API errors | Verify API key with `repobird config list` |
| Slow refresh | Adjust refresh_interval in config |

## Getting Help

- In TUI: Press `?` for context help
- CLI: `repobird --help` or `repobird COMMAND --help`
- Docs: See [full documentation](./README.md)
