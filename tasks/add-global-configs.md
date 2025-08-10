# Task: Add Global Configuration Settings to RepoBird CLI

## Overview
Extend the RepoBird CLI configuration system to support additional global settings documented in the configuration guide but not yet implemented. This will improve user experience by allowing persistent customization of behavior, appearance, and performance settings.

## Current State
Currently only 3 settings are implemented in `internal/config/config.go`:
- `api_key` - API authentication key
- `api_url` - API endpoint URL  
- `debug` - Debug mode flag

## Priority Settings to Implement

### Phase 1: Essential Settings (High Priority)
These settings directly impact user workflow and should be implemented first:

1. **TUI Dashboard Layout**
   - Field: `tui.default_layout`
   - Type: string enum ("TripleColumn", "AllRuns", "RepositoriesOnly")
   - Default: "TripleColumn"
   - Usage: Persist user's preferred dashboard layout across sessions

2. **Request Timeout**
   - Field: `timeout`
   - Type: duration string (e.g., "45m", "30s")
   - Default: "45m"
   - Usage: Override default timeout for long-running operations

3. **Output Format**
   - Field: `output_format`
   - Type: string enum ("table", "json", "yaml", "plain")
   - Default: "table"
   - Usage: Default format for CLI output

4. **Polling Interval**
   - Field: `polling.interval`
   - Type: duration string
   - Default: "5s"
   - Usage: Control status check frequency

### Phase 2: TUI Enhancements (Medium Priority)
Improve TUI user experience:

5. **TUI Theme**
   - Field: `tui.theme`
   - Type: string enum ("dark", "light", "auto")
   - Default: "dark"

6. **TUI Vim Mode**
   - Field: `tui.vim_mode`
   - Type: boolean
   - Default: false

7. **TUI Refresh Interval**
   - Field: `tui.refresh_interval`
   - Type: duration string
   - Default: "5s"

8. **Auto-follow New Runs**
   - Field: `tui.auto_follow_new_runs`
   - Type: boolean
   - Default: true

### Phase 3: Performance & Caching (Lower Priority)
Optimize performance:

9. **Cache Settings**
   - Fields: `cache.enabled`, `cache.ttl`, `cache.max_size`
   - Types: boolean, duration, string
   - Defaults: true, "30s", "100MB"

10. **Retry Configuration**
    - Fields: `retry.max_attempts`, `retry.initial_delay`
    - Types: integer, duration
    - Defaults: 5, "1s"

## Implementation Steps

### Step 1: Extend Config Structure
Update `internal/config/config.go`:

```go
type Config struct {
    // Existing fields
    APIKey string `mapstructure:"api_key"`
    APIURL string `mapstructure:"api_url"`
    Debug  bool   `mapstructure:"debug"`
    
    // New core settings
    Timeout       string `mapstructure:"timeout"`
    OutputFormat  string `mapstructure:"output_format"`
    NoColor       bool   `mapstructure:"no_color"`
    
    // TUI settings
    TUI TUIConfig `mapstructure:"tui"`
    
    // Polling settings
    Polling PollingConfig `mapstructure:"polling"`
    
    // Cache settings
    Cache CacheConfig `mapstructure:"cache"`
    
    // Retry settings
    Retry RetryConfig `mapstructure:"retry"`
}

type TUIConfig struct {
    DefaultLayout      string        `mapstructure:"default_layout"`
    Theme             string        `mapstructure:"theme"`
    VimMode           bool          `mapstructure:"vim_mode"`
    RefreshInterval   string        `mapstructure:"refresh_interval"`
    AutoFollowNewRuns bool          `mapstructure:"auto_follow_new_runs"`
    ShowHelp          bool          `mapstructure:"show_help"`
    Compact           bool          `mapstructure:"compact"`
}

type PollingConfig struct {
    Interval    string `mapstructure:"interval"`
    MaxDuration string `mapstructure:"max_duration"`
}

type CacheConfig struct {
    Enabled    bool   `mapstructure:"enabled"`
    TTL        string `mapstructure:"ttl"`
    MaxSize    string `mapstructure:"max_size"`
    Location   string `mapstructure:"location"`
    Persistent bool   `mapstructure:"persistent"`
}

type RetryConfig struct {
    MaxAttempts   int     `mapstructure:"max_attempts"`
    InitialDelay  string  `mapstructure:"initial_delay"`
    MaxDelay      string  `mapstructure:"max_delay"`
    Multiplier    float64 `mapstructure:"multiplier"`
    Jitter        float64 `mapstructure:"jitter"`
}
```

### Step 2: Add Default Values
Update `LoadConfig()` function to set defaults:

```go
func LoadConfig() (*Config, error) {
    // ... existing code ...
    
    // Set new defaults
    viper.SetDefault("timeout", "45m")
    viper.SetDefault("output_format", "table")
    viper.SetDefault("no_color", false)
    
    // TUI defaults
    viper.SetDefault("tui.default_layout", "TripleColumn")
    viper.SetDefault("tui.theme", "dark")
    viper.SetDefault("tui.vim_mode", false)
    viper.SetDefault("tui.refresh_interval", "5s")
    viper.SetDefault("tui.auto_follow_new_runs", true)
    viper.SetDefault("tui.show_help", true)
    viper.SetDefault("tui.compact", false)
    
    // Polling defaults
    viper.SetDefault("polling.interval", "5s")
    viper.SetDefault("polling.max_duration", "45m")
    
    // Cache defaults
    viper.SetDefault("cache.enabled", true)
    viper.SetDefault("cache.ttl", "30s")
    viper.SetDefault("cache.max_size", "100MB")
    viper.SetDefault("cache.persistent", true)
    
    // Retry defaults
    viper.SetDefault("retry.max_attempts", 5)
    viper.SetDefault("retry.initial_delay", "1s")
    viper.SetDefault("retry.max_delay", "30s")
    viper.SetDefault("retry.multiplier", 2.0)
    viper.SetDefault("retry.jitter", 0.1)
    
    // ... rest of function ...
}
```

### Step 3: Update Config Commands
Extend `internal/commands/config.go` to handle new settings:

```go
// Add new config keys
const (
    configKeyAPIKey         = "api-key"
    configKeyAPIURL         = "api-url"
    configKeyDebug          = "debug"
    configKeyTimeout        = "timeout"
    configKeyOutputFormat   = "output-format"
    configKeyTUILayout      = "tui.default-layout"
    configKeyTUITheme       = "tui.theme"
    configKeyTUIVimMode     = "tui.vim-mode"
    configKeyPollingInterval = "polling.interval"
    // ... more keys ...
)

// Update availableKeysHelp
const availableKeysHelp = `
Available keys:
  Core Settings:
    api-key                  API authentication key
    api-url                  API endpoint URL
    debug                    Enable debug output (true/false)
    timeout                  Request timeout (e.g., 45m, 30s)
    output-format            Output format (table/json/yaml/plain)
    
  TUI Settings:
    tui.default-layout       Dashboard layout (TripleColumn/AllRuns/RepositoriesOnly)
    tui.theme               Theme (dark/light/auto)
    tui.vim-mode            Enable vim keybindings (true/false)
    tui.refresh-interval    Auto-refresh interval (e.g., 5s)
    tui.auto-follow-new-runs Auto-follow new runs (true/false)
    
  Polling Settings:
    polling.interval         Status check interval (e.g., 5s)
    polling.max-duration    Maximum polling duration (e.g., 45m)
    
  Cache Settings:
    cache.enabled           Enable caching (true/false)
    cache.ttl              Cache time-to-live (e.g., 30s)
    cache.max-size         Maximum cache size (e.g., 100MB)`
```

### Step 4: Add Validation
Create validation functions for new settings:

```go
func validateTimeout(value string) error {
    _, err := time.ParseDuration(value)
    if err != nil {
        return fmt.Errorf("invalid timeout format: %w", err)
    }
    return nil
}

func validateOutputFormat(value string) error {
    validFormats := []string{"table", "json", "yaml", "plain"}
    for _, format := range validFormats {
        if value == format {
            return nil
        }
    }
    return fmt.Errorf("invalid output format: must be one of %v", validFormats)
}

func validateLayout(value string) error {
    validLayouts := []string{"TripleColumn", "AllRuns", "RepositoriesOnly"}
    for _, layout := range validLayouts {
        if value == layout {
            return nil
        }
    }
    return fmt.Errorf("invalid layout: must be one of %v", validLayouts)
}

func validateTheme(value string) error {
    validThemes := []string{"dark", "light", "auto"}
    for _, theme := range validThemes {
        if value == theme {
            return nil
        }
    }
    return fmt.Errorf("invalid theme: must be one of %v", validThemes)
}
```

### Step 5: Use Settings in Application
Update components to use the new configuration:

#### Dashboard Layout (internal/tui/views/dashboard.go)
```go
func NewDashboardView() *DashboardView {
    cfg, _ := config.LoadConfig()
    
    // Set initial layout from config
    initialLayout := models.LayoutTripleColumn
    if cfg.TUI.DefaultLayout == "AllRuns" {
        initialLayout = models.LayoutAllRuns
    } else if cfg.TUI.DefaultLayout == "RepositoriesOnly" {
        initialLayout = models.LayoutRepositoriesOnly
    }
    
    return &DashboardView{
        currentLayout: initialLayout,
        // ... other fields ...
    }
}

// Save layout preference when user changes it
func (d *DashboardView) cycleLayout() {
    // ... existing layout cycling code ...
    
    // Save the new layout preference
    cfg, _ := config.LoadConfig()
    cfg.TUI.DefaultLayout = d.currentLayout.String()
    config.SaveConfig(cfg)
}
```

#### API Client Timeout (internal/api/client.go)
```go
func NewClient(apiKey, apiURL string) *Client {
    cfg, _ := config.LoadConfig()
    
    timeout := 45 * time.Minute // default
    if cfg.Timeout != "" {
        if duration, err := time.ParseDuration(cfg.Timeout); err == nil {
            timeout = duration
        }
    }
    
    return &Client{
        httpClient: &http.Client{
            Timeout: timeout,
        },
        // ... other fields ...
    }
}
```

#### Output Formatting (internal/commands/status.go)
```go
func formatOutput(data interface{}) error {
    cfg, _ := config.LoadConfig()
    
    format := cfg.OutputFormat
    if flagOutputFormat != "" {
        format = flagOutputFormat // Command flag overrides config
    }
    
    switch format {
    case "json":
        return outputJSON(data)
    case "yaml":
        return outputYAML(data)
    case "plain":
        return outputPlain(data)
    default:
        return outputTable(data)
    }
}
```

### Step 6: Add Config Migration
Create migration for existing configs:

```go
func migrateConfig(cfg *Config) (*Config, bool) {
    migrated := false
    
    // Add new fields with defaults if missing
    if cfg.Timeout == "" {
        cfg.Timeout = "45m"
        migrated = true
    }
    
    if cfg.OutputFormat == "" {
        cfg.OutputFormat = "table"
        migrated = true
    }
    
    // Initialize nested structs if nil
    if cfg.TUI.DefaultLayout == "" {
        cfg.TUI.DefaultLayout = "TripleColumn"
        migrated = true
    }
    
    return cfg, migrated
}
```

### Step 7: Update Tests
Add tests for new configuration options:

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        value   string
        wantErr bool
    }{
        {"valid timeout", "timeout", "30m", false},
        {"invalid timeout", "timeout", "invalid", true},
        {"valid output format", "output-format", "json", false},
        {"invalid output format", "output-format", "xml", true},
        {"valid layout", "tui.default-layout", "AllRuns", false},
        {"invalid layout", "tui.default-layout", "InvalidLayout", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateConfigValue(tt.key, tt.value)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateConfigValue() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Testing Plan

1. **Unit Tests**
   - Test config loading with new fields
   - Test default values
   - Test validation functions
   - Test migration from old configs

2. **Integration Tests**
   - Test config command with new keys
   - Test that settings affect application behavior
   - Test config file parsing

3. **Manual Testing**
   - Set each new config value via CLI
   - Verify settings persist across sessions
   - Test that settings affect runtime behavior
   - Test invalid values are rejected

## Rollout Plan

1. **Phase 1** (1-2 days)
   - Implement core structure changes
   - Add essential settings (layout, timeout, output format)
   - Update config commands
   - Add validation

2. **Phase 2** (1 day)
   - Implement TUI settings
   - Update dashboard to use settings
   - Add polling configuration

3. **Phase 3** (1 day)
   - Add cache and retry settings
   - Implement migration logic
   - Complete test coverage

4. **Documentation** (concurrent)
   - Update CLI help text
   - Update configuration guide
   - Add examples to README

## Success Criteria

- [ ] All new config fields can be set via `repobird config set`
- [ ] Settings persist in `~/.repobird/config.yaml`
- [ ] Application components use configured values
- [ ] Invalid values are rejected with helpful errors
- [ ] Existing configs are migrated without data loss
- [ ] Tests pass with >80% coverage
- [ ] Documentation is complete and accurate

## Notes

- Maintain backward compatibility with existing configs
- Use sensible defaults that match current behavior
- Validate all user input to prevent runtime errors
- Consider adding a `config doctor` command to diagnose issues
- Plan for future settings without breaking changes