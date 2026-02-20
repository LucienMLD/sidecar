---
status: pending
priority: p1
issue_id: "003"
tags: [code-review, performance, error-handling]
dependencies: []
---

# Plugin Check Happens Too Late in StartAgentWithOptions

## Problem Statement

The plugin installation check occurs AFTER session name computation and session existence validation in `StartAgentWithOptions()`. If the plugin is missing, the user sees an error toast, but the code has already performed unnecessary work. This violates the fail-fast principle.

## Findings

**Source:** Kieran Rails Reviewer Agent

**Location:** `internal/plugins/workspace/agent.go:539-548`

**Current Code:**
```go
func (p *Plugin) StartAgentWithOptions(wt *Worktree, agentType AgentType, skipPerms bool, prompt *Prompt) tea.Cmd {
    epoch := p.ctx.Epoch // Line 537: Capture epoch

    return func() tea.Msg {
        sessionName := tmuxSessionPrefix + sanitizeName(wt.Name) // Line 550: Session name

        // Lines 553-564: Session existence check
        if tmuxSessionExists(sessionName) {
            // ... handle existing session
        }

        // Lines 539-548: Plugin check happens HERE (too late!)
        if prompt != nil && prompt.RequiresPlugin != "" {
            if !IsPluginInstalled(prompt.RequiresPlugin) {
                return app.ToastMsg{
                    Message:  ErrPluginNotInstalled,
                    Duration: 6 * time.Second,
                    IsError:  true,
                }
            }
        }

        // ... rest of function (session creation, etc.)
    }
}
```

**Problem:** The check happens on line 539, AFTER:
- Epoch capture (line 537)
- Session name sanitization (line 550)
- Session existence check (lines 553-564)

**Impact:** Wasted CPU cycles and potential side effects (tmux queries) before detecting inevitable failure.

## Proposed Solutions

### Solution 1: Move Plugin Check to Top (Recommended)

**Implementation:**
```go
func (p *Plugin) StartAgentWithOptions(wt *Worktree, agentType AgentType, skipPerms bool, prompt *Prompt) tea.Cmd {
    epoch := p.ctx.Epoch // Capture epoch for stale detection

    // Check plugin FIRST - fail fast before any other work
    if prompt != nil && prompt.RequiresPlugin != "" {
        if !IsPluginInstalled(prompt.RequiresPlugin) {
            return func() tea.Msg {
                return AgentStartedMsg{
                    Epoch: epoch,
                    Err:   fmt.Errorf(ErrPluginNotInstalled),
                }
            }
        }
    }

    return func() tea.Msg {
        sessionName := tmuxSessionPrefix + sanitizeName(wt.Name)
        // ... rest of function
    }
}
```

**Pros:**
- Fail-fast principle (detect failure immediately)
- No wasted work (session checks, sanitization)
- Consistent error return type (`AgentStartedMsg`)

**Cons:**
- None (pure improvement)

**Effort:** Small (30 minutes)
**Risk:** Low (simple code move)

### Solution 2: Return Consistent Error Type

**Current Problem:** The plugin check returns `app.ToastMsg` directly instead of `AgentStartedMsg` like other error cases.

**Implementation:**
```go
// Instead of:
return app.ToastMsg{
    Message:  ErrPluginNotInstalled,
    Duration: 6 * time.Second,
    IsError:  true,
}

// Use:
return AgentStartedMsg{
    Epoch: epoch,
    Err:   fmt.Errorf(ErrPluginNotInstalled),
}
```

**Pros:**
- Consistent with other error returns (lines 577, 611)
- Easier to test (single message type expected)
- Caller can decide how to display error

**Cons:**
- Requires caller to handle error → toast conversion

**Effort:** Small (1 hour including caller updates)
**Risk:** Low

## Recommended Action

**Implement both solutions:**
1. Move plugin check to line 538 (immediately after epoch capture)
2. Return `AgentStartedMsg` with error instead of `ToastMsg`

This ensures:
- Fail-fast behavior (no wasted work)
- Consistent error handling pattern
- Easier testing

## Technical Details

**Affected Files:**
- `internal/plugins/workspace/agent.go` (move check + change return type)
- `internal/plugins/workspace/agent_test.go` (add test case)

**Database Changes:** None

**API Changes:** Internal only (message type change handled by caller)

## Acceptance Criteria

- [ ] Plugin check happens at line 538 (before session name computation)
- [ ] Plugin check returns `AgentStartedMsg{Epoch, Err}` not `ToastMsg`
- [ ] No other operations occur before plugin check if check will fail
- [ ] Test case added: `TestStartAgentWithOptions_PluginNotInstalled`
  - [ ] Verifies `AgentStartedMsg` returned
  - [ ] Verifies error contains "compound-engineering plugin required"
  - [ ] Verifies no tmux commands executed
- [ ] Existing tests still pass
- [ ] Performance: Plugin check overhead <1ms for failure case

## Work Log

**2026-02-20:** Created todo from code review findings (Kieran Rails Reviewer Agent)

## Resources

- PR: https://github.com/LucienMLD/sidecar/pull/1
- Code Review: Agent report (Kieran Rails Reviewer)
- Pattern: Fail-fast principle (detect errors before side effects)
- Reference: `agent.go:577, 611` for consistent error return pattern
