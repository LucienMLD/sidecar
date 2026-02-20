---
status: pending
priority: p2
issue_id: "004"
tags: [code-review, error-handling, debuggability]
dependencies: []
---

# Insufficient Error Handling in Plugin Detection

## Problem Statement

The `IsPluginInstalled()` function silently returns `false` for all error conditions (file not found, permission denied, malformed JSON). This masks legitimate security issues and makes debugging difficult when installation problems occur.

## Findings

**Source:** Security Sentinel Agent

**Location:** `internal/plugins/workspace/prompts.go:237-252`

**Current Code:**
```go
func IsPluginInstalled(pluginID string) bool {
    home, err := os.UserHomeDir()
    if err != nil {
        return false  // ⚠️ Silent failure
    }
    data, err := os.ReadFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"))
    if err != nil {
        return false  // ⚠️ Permission denied? File tampering? Unknown.
    }
    var registry installedPluginsRegistry
    if err := json.Unmarshal(data, &registry); err != nil {
        return false  // ⚠️ Corrupted file? Attack attempt?
    }
    _, ok := registry.Plugins[pluginID]
    return ok
}
```

**Problems:**
- Cannot distinguish between "plugin not installed" (expected) and "can't read HOME dir" (system error)
- Permission denied errors are hidden
- Corrupted plugin registry is treated same as missing plugin
- No logging for debugging installation issues

**Impact:** MEDIUM - Difficult to diagnose when plugin installation fails for non-obvious reasons.

## Proposed Solutions

### Solution 1: Return Error + Boolean (Recommended)

**Implementation:**
```go
func IsPluginInstalled(pluginID string) (bool, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return false, fmt.Errorf("get home dir: %w", err)
    }

    pluginFile := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")

    // Check file ownership/permissions before reading
    fileInfo, err := os.Stat(pluginFile)
    if os.IsNotExist(err) {
        return false, nil  // File doesn't exist is expected (plugin not installed)
    }
    if err != nil {
        return false, fmt.Errorf("stat plugin file: %w", err)
    }

    // Warn if file has suspicious permissions
    if fileInfo.Mode().Perm() & 0002 != 0 {
        slog.Warn("plugin registry is world-writable", "path", pluginFile)
    }

    data, err := os.ReadFile(pluginFile)
    if err != nil {
        return false, fmt.Errorf("read plugin file: %w", err)
    }

    var registry installedPluginsRegistry
    if err := json.Unmarshal(data, &registry); err != nil {
        return false, fmt.Errorf("parse plugin registry: %w", err)
    }

    _, ok := registry.Plugins[pluginID]
    return ok, nil
}
```

**Pros:**
- Clear distinction between "not installed" (false, nil) and "error" (false, err)
- Debuggable (errors contain context)
- Security check for world-writable registry

**Cons:**
- Requires updating caller in `agent.go` to handle error

**Effort:** Medium (3 hours including caller updates + tests)
**Risk:** Low

### Solution 2: Log Errors, Keep Boolean Return

**Implementation:**
```go
func IsPluginInstalled(pluginID string) bool {
    home, err := os.UserHomeDir()
    if err != nil {
        slog.Error("failed to get home dir", "error", err)
        return false
    }

    pluginFile := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
    data, err := os.ReadFile(pluginFile)
    if err != nil {
        if !os.IsNotExist(err) {
            slog.Warn("failed to read plugin registry", "path", pluginFile, "error", err)
        }
        return false
    }

    var registry installedPluginsRegistry
    if err := json.Unmarshal(data, &registry); err != nil {
        slog.Error("corrupted plugin registry", "path", pluginFile, "error", err)
        return false
    }

    _, ok := registry.Plugins[pluginID]
    return ok
}
```

**Pros:**
- No API change (still returns boolean)
- Errors are logged for debugging
- Backward compatible

**Cons:**
- Caller can't distinguish error types programmatically
- Logs may be missed if not monitored

**Effort:** Small (1 hour)
**Risk:** None

## Recommended Action

**Solution 2** (logging) for now, **Solution 1** (return error) in future refactor.

Rationale:
- Logging provides immediate value without breaking changes
- Can upgrade to error return in next iteration
- Balances debuggability with implementation effort

## Technical Details

**Affected Files:**
- `internal/plugins/workspace/prompts.go` (add logging to `IsPluginInstalled`)
- `internal/plugins/workspace/prompts_test.go` (verify logs appear in error cases)

**Database Changes:** None

**API Changes:** None (if using Solution 2)

## Acceptance Criteria

- [ ] Errors logged with appropriate severity:
  - [ ] HOME dir unavailable: ERROR level
  - [ ] File not found: No log (expected case)
  - [ ] Permission denied: WARN level
  - [ ] Corrupted JSON: ERROR level
- [ ] Logs include contextual information (file path, error message)
- [ ] No change to function signature (Solution 2)
- [ ] Test cases verify logging behavior:
  - [ ] Missing file: no log, returns false
  - [ ] Corrupted JSON: ERROR log, returns false
  - [ ] Permission denied: WARN log, returns false
- [ ] Existing behavior preserved (all tests pass)

## Work Log

**2026-02-20:** Created todo from code review findings (Security Sentinel Agent)

## Resources

- PR: https://github.com/LucienMLD/sidecar/pull/1
- Security Review: Agent report (Security Sentinel)
- Related: Silent failures anti-pattern
- Go logging: `log/slog` package
