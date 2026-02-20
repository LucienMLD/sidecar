---
status: pending
priority: p3
issue_id: "006"
tags: [code-review, agent-native, feature-request, mcp]
dependencies: []
---

# Add Workspace MCP Tools for Agent Accessibility

## Problem Statement

The 6 workflow prompts added in this PR are **UI-only capabilities**. While the prompts invoke compound-engineering plugin slash commands (which ARE agent-accessible), the workspace creation mechanism itself is not agent-accessible. Agents cannot programmatically create workspaces, select prompts, or invoke these workflows without user intervention through the TUI.

## Findings

**Source:** Agent-Native Reviewer Agent

**Gap Analysis:**

| UI Action | Agent Tool Status |
|-----------|------------------|
| Create workspace with prompt | ❌ UI-only |
| Select prompt from list | ❌ UI-only |
| List available prompts | ❌ UI-only |
| List active workspaces | ❌ UI-only |
| Delete workspace | ❌ UI-only |

**Impact:** Agents cannot initiate workflows autonomously. When asked "help me start work on a feature", agents can't create the appropriate workspace - users must manually create it first.

**Current Workaround:** Users create workspaces via UI, then agents work within them.

## Proposed Solutions

### Solution 1: Comprehensive MCP Tool Suite (Recommended)

**Implementation:**

```go
// MCP Tool: create_workspace
{
  "name": "create_workspace",
  "description": "Create a new workspace with a prompt",
  "parameters": {
    "name": "string (workspace name)",
    "prompt_id": "string (prompt name from list_workspace_prompts)",
    "task_id": "string (optional - TD task ID)",
    "agent_type": "string (optional - claude/codex/gemini, default: claude)"
  },
  "returns": {
    "workspace_name": "string",
    "tmux_session": "string",
    "git_branch": "string",
    "prompt_used": "string"
  }
}

// MCP Tool: list_workspace_prompts
{
  "name": "list_workspace_prompts",
  "description": "List available workspace prompts",
  "parameters": {},
  "returns": [
    {
      "name": "Begin Work on Ticket",
      "ticket_mode": "required",
      "requires_plugin": "compound-engineering@every-marketplace",
      "description": "Invokes /workflows:plan with ticket"
    },
    // ... 5 more prompts
  ]
}

// MCP Tool: list_workspaces
{
  "name": "list_workspaces",
  "description": "List active workspaces with status",
  "parameters": {},
  "returns": [
    {
      "name": "feature-auth",
      "task_id": "td-123",
      "git_branch": "worktree/feature-auth",
      "agent_status": "running",
      "tmux_session": "sidecar-ws-feature-auth"
    }
  ]
}

// MCP Tool: delete_workspace
{
  "name": "delete_workspace",
  "description": "Delete a workspace and clean up git worktree",
  "parameters": {
    "workspace_name": "string"
  },
  "returns": {
    "success": true,
    "deleted_worktree": "path"
  }
}
```

**Pros:**
- Full agent-workspace parity (agents can do everything users can)
- Enables autonomous workflow initiation
- Supports multi-workspace management by agents

**Cons:**
- Significant implementation effort (10-15 hours)
- Requires MCP server integration

**Effort:** Large (2-3 days)
**Risk:** Medium (new API surface)

### Solution 2: Minimal MCP Tool (Single Create)

**Implementation:**

```go
// MCP Tool: create_workspace (minimal version)
{
  "name": "create_workspace",
  "description": "Create workspace with prompt",
  "parameters": {
    "name": "string",
    "prompt_name": "string (from: Begin Work on Ticket, Brainstorm Feature, ...)"
  }
}
```

**Pros:**
- Smaller scope (easier to implement)
- Unblocks primary use case (agents starting workflows)

**Cons:**
- No workspace discovery (agents can't list existing workspaces)
- No cleanup capability (agents can't delete when done)

**Effort:** Medium (1 day)
**Risk:** Low

### Solution 3: Document Limitation

**Implementation:**

Add to system prompt:
```markdown
## Workspace Limitations

Sidecar workspaces are user-created contexts. You cannot create or manage workspaces programmatically.

When asked to start work on a feature:
1. Suggest the user create a workspace with the appropriate prompt
2. Recommend: "Begin Work on Ticket" for tickets, "Brainstorm Feature" for exploratory work
3. Once the workspace exists, you can run workflows within it using /workflows:* commands
```

**Pros:**
- Zero implementation effort
- Clear expectation setting

**Cons:**
- Agents remain second-class citizens
- Workflow friction (user must context-switch)

**Effort:** Trivial (documentation only)
**Risk:** None

## Recommended Action

**Solution 2** (minimal MCP tool) as first iteration, **Solution 1** (full suite) as future enhancement.

Rationale:
- Unblocks primary agent-driven workflow use case
- Manageable implementation scope
- Can iterate to full MCP suite based on usage

## Technical Details

**Affected Files:**
- `internal/mcp/workspace_tools.go` (new file - MCP tool definitions)
- `internal/plugins/workspace/plugin.go` (expose CreateWorkspace method)
- `cmd/sidecar/mcp_server.go` (register MCP tools)
- System prompt (document workspace MCP tools)

**Database Changes:** None

**API Changes:** New MCP tools (additive only)

## Acceptance Criteria

- [ ] MCP tool `create_workspace` implemented
- [ ] Validates plugin installation before creation (same check as UI)
- [ ] Returns structured workspace info (name, branch, session)
- [ ] Error handling:
  - [ ] Plugin not installed: returns actionable error
  - [ ] Invalid prompt name: returns available prompts
  - [ ] Workspace already exists: returns existing workspace info
- [ ] System prompt documents usage:
  - [ ] How to discover prompts
  - [ ] When to use each prompt type
  - [ ] Example workflow
- [ ] Integration test:
  - [ ] Agent creates workspace with "Brainstorm Feature" prompt
  - [ ] Workspace is functional (agent can run /workflows:brainstorm)
- [ ] Documentation in README.md

## Work Log

**2026-02-20:** Created todo from code review findings (Agent-Native Reviewer)

## Resources

- PR: https://github.com/LucienMLD/sidecar/pull/1
- Agent-Native Review: Agent report
- Related: Agent-native architecture principle (any user action should be agent-accessible)
- MCP Protocol: https://modelcontextprotocol.io/
