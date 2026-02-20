---
title: "feat: Integrate Compound-Engineering Workflow Prompts"
type: feat
date: 2026-02-19
brainstorm: docs/brainstorms/2026-02-19-compound-engineering-workflow-integration-brainstorm.md
---

# feat: Integrate Compound-Engineering Workflow Prompts

## Overview

Replace Sidecar's default prompts with prompts that leverage the compound-engineering Claude Code plugin workflows (`/workflows:brainstorm`, `/workflows:plan`, `/workflows:work`). Add two new prompts and a runtime guard that blocks workspace creation if the compound-engineering plugin is not installed.

## Problem Statement

The current default prompts ("Begin Work on Ticket", "Plan to Epic", etc.) use generic `td` instructions. Users who have the compound-engineering plugin installed get no guidance toward the structured `/workflows:*` commands that provide a significantly better development experience (brainstorm → plan → work cycle).

## Proposed Solution

1. **Extend the `Prompt` struct** with a non-serialized `RequiresPlugin` field
2. **Add a plugin detection function** `IsPluginInstalled(pluginID string) bool` reading `~/.claude/plugins/installed_plugins.json`
3. **Replace `DefaultPrompts()`** with 7 prompts (5 updated, 2 new)
4. **Add runtime check** in `StartAgentWithOptions` that blocks and shows a toast error if `RequiresPlugin` is set but the plugin is not installed

## Technical Approach

### Architecture

No new packages. All changes are within `internal/plugins/workspace/`:

- `prompts.go` — struct extension, new defaults, plugin detection function, error constants
- `agent.go` — plugin check in `StartAgentWithOptions`
- `prompts_test.go` — update counts, add detection tests
- `prompt_picker_test.go` — update count

### Plugin Detection

The `~/.claude/plugins/installed_plugins.json` file has the following schema:
```json
{
  "version": 1,
  "plugins": {
    "compound-engineering@every-marketplace": {
      "version": "2.31.1",
      "installPath": "..."
    }
  }
}
```

Detection function reads the file, unmarshals into a struct, and checks key presence. All error cases (file absent, malformed JSON) return `false` — treated as "plugin not installed".

### Plugin Check Timing

The check runs inside `StartAgentWithOptions` (`agent.go:533`), **before** any workspace creation side-effects (tmux pane, worktree creation). If the check fails, a `tea.Cmd` is returned that emits a `app.ToastMsg{IsError: true}` and no workspace is created.

## Acceptance Criteria

### Functional

- [x] `Prompt` struct has a `RequiresPlugin string` field with `json:"-"` tag
- [x] `IsPluginInstalled("compound-engineering@every-marketplace")` returns `true` when the key exists in `installed_plugins.json`
- [x] `IsPluginInstalled` returns `false` when the file is absent, empty, or malformed JSON
- [x] When `prompt.RequiresPlugin != ""` and `IsPluginInstalled` returns `false`, `StartAgentWithOptions` returns a blocking toast error and does NOT create the workspace
- [x] `DefaultPrompts()` returns 7 prompts with the correct names, TicketModes, RequiresPlugin values, and body content (see prompt spec below)
- [x] Prompts without `RequiresPlugin` ("Code Review Ticket", "TD Review Session") still work normally with no plugin check
- [x] `EnsureDefaultPrompts` and `WriteDefaultPromptsToConfig` continue to work (they call `DefaultPrompts()` internally — no changes needed)

### Non-Functional

- [x] `RequiresPlugin` field does not appear in JSON output when serializing a `Prompt`
- [x] `RequiresPlugin` field is silently ignored when deserializing JSON that lacks it (zero value = "")
- [x] Plugin check does not perform any network requests — local file read only

### Tests

- [x] `TestDefaultPrompts` updated: count changed from `5` to `7`, new prompts asserted by name and TicketMode
- [x] All `prompts_test.go` hardcoded count `5` changed to `7`
- [x] `prompt_picker_test.go` line 98: count changed from `5` to `7`
- [x] `TestIsPluginInstalled_Present` — file with key returns true
- [x] `TestIsPluginInstalled_Absent` — file without key returns false
- [x] `TestIsPluginInstalled_FileNotFound` — no file returns false
- [x] `TestIsPluginInstalled_MalformedJSON` — bad JSON returns false

## Prompt Specification

### Prompts with `RequiresPlugin: "compound-engineering@every-marketplace"`

**1. "Begin Work on Ticket"**
- `TicketMode: TicketRequired`
- `RequiresPlugin: "compound-engineering@every-marketplace"`
- Body:
```
td usage --new-session

Review ticket {{ticket}}.

If the ticket lacks clear acceptance criteria or has open questions, run:
/workflows:brainstorm {{ticket}}

Otherwise, run:
/workflows:plan {{ticket}}

Then execute the plan with:
/workflows:work
```

**2. "Brainstorm Feature"** *(new)*
- `TicketMode: TicketNone`
- `RequiresPlugin: "compound-engineering@every-marketplace"`
- Body:
```
td usage --new-session

/workflows:brainstorm
```

**3. "Plan Feature"** *(new)*
- `TicketMode: TicketOptional`
- `RequiresPlugin: "compound-engineering@every-marketplace"`
- Body:
```
td usage --new-session

/workflows:plan {{ticket || 'this feature'}}
```
*(Uses fallback syntax so the plan command receives meaningful context even without a ticket)*

**4. "Plan to Epic (No Impl)"**
- `TicketMode: TicketNone`
- `RequiresPlugin: "compound-engineering@every-marketplace"`
- Body:
```
td usage --new-session

/workflows:plan

Do not implement. Create sub-tasks with td after planning.
```

**5. "Plan to Epic + Implement"**
- `TicketMode: TicketNone`
- `RequiresPlugin: "compound-engineering@every-marketplace"`
- Body:
```
td usage --new-session

/workflows:plan

Then implement with:
/workflows:work
```

### Prompts unchanged (no `RequiresPlugin`)

**6. "Code Review Ticket"** — body unchanged, `TicketRequired`

**7. "TD Review Session"** — body unchanged, `TicketNone`

## Error Messages

Define as constants in `prompts.go`:

```go
const (
    // ErrPluginNotInstalled is shown when a compound-workflow prompt is selected
    // but the required Claude Code plugin is not installed.
    ErrPluginNotInstalled = "compound-engineering plugin required. Install it via: claude plugins install compound-engineering@every-marketplace"
)
```

Toast duration: 6 seconds (longer than the default 3s for error visibility).

## Implementation Steps

### Step 1 — Extend `Prompt` struct (`prompts.go`)

Add `RequiresPlugin string \`json:"-"\`` to the `Prompt` struct.

### Step 2 — Add plugin detection (`prompts.go`)

Add `IsPluginInstalled(pluginID string) bool` function:
- Reads `~/.claude/plugins/installed_plugins.json` via `os.UserHomeDir()` + `os.ReadFile`
- Unmarshals into a struct with `Plugins map[string]json.RawMessage`
- Returns `true` if `pluginID` key exists in the map
- Returns `false` on any error (file not found, parse failure)

Add `ErrPluginNotInstalled` constant.

### Step 3 — Update `DefaultPrompts()` (`prompts.go`)

Replace the function body with 7 prompts per the prompt specification above.

### Step 4 — Add plugin check to `StartAgentWithOptions` (`agent.go`)

After line 533 (function entry), before any workspace side-effects:

```go
if prompt != nil && prompt.RequiresPlugin != "" {
    if !IsPluginInstalled(prompt.RequiresPlugin) {
        return func() tea.Msg {
            return app.ToastMsg{
                Message:  ErrPluginNotInstalled,
                Duration: 6 * time.Second,
                IsError:  true,
            }
        }
    }
}
```

### Step 5 — Update tests (`prompts_test.go`, `prompt_picker_test.go`)

- Change all `5` hardcoded counts to `7`
- Add expected names "Brainstorm Feature" and "Plan Feature" to `expectedNames` maps
- Add `TestIsPluginInstalled_*` test cases using `t.TempDir()` and temporary JSON files

### Step 6 — Run tests

```bash
go test ./internal/plugins/workspace/...
```

## References

### Internal References

- `internal/plugins/workspace/prompts.go:22` — `Prompt` struct
- `internal/plugins/workspace/prompts.go:128` — `DefaultPrompts()`
- `internal/plugins/workspace/agent.go:427` — `buildAgentCommand()`
- `internal/plugins/workspace/agent.go:533` — `StartAgentWithOptions()` — plugin check insertion point
- `internal/plugins/workspace/template.go:13` — `ExpandPromptTemplate()` — handles `{{ticket || 'fallback'}}` syntax
- `internal/plugins/workspace/prompts_test.go:331` — `TestDefaultPrompts` — update count 5→7
- `internal/plugins/workspace/prompt_picker_test.go:98` — round-trip count 5→7
- `internal/app/commands.go` — `ToastMsg` type definition

### External References

- Brainstorm: `docs/brainstorms/2026-02-19-compound-engineering-workflow-integration-brainstorm.md`
- Plugin registry schema: `~/.claude/plugins/installed_plugins.json` (key: `"compound-engineering@every-marketplace"`)
