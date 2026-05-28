# Phase 2: internal/git 0% → 90% Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development

**Goal:** Raise internal/git coverage from 0% to 90%+ by refactoring for testability and writing tests with testcontainers for remote operations and local temp dirs for local operations.

**Architecture:** Make `Run`/`RunInternal` testable via a package-level `CmdFn` variable (same pattern as the old `execCommand` in root.go). For tests, use local temp dirs with native git for basic operations (worktree add/remove/list, porcelain parsing). Use testcontainers `alpine/git` container for remote operations (FetchOrigin, ListRemoteBranches).

**Spec:** `docs/superpowers/specs/2026-05-28-testing-strategy-design.md`

---

### Task 1: Add testcontainers-go dependency

```bash
go get github.com/testcontainers/testcontainers-go
go mod tidy
go build ./...
git commit -m "chore: add testcontainers-go dependency"
```

### Task 2: Make git command execution injectable

Refactor `internal/git/worktree.go`: extract `exec.Command` calls in `Run` and `RunInternal` into package-level `var CmdFn = exec.Command`.

Then `Run` uses `CmdFn("git", args...)` instead of `exec.Command("git", args...)`.

### Task 3: Write git_test.go — TestMain, porcelain parsing, URLToProjectName

Create `internal/git/git_test.go` with TestMain (starts alpine/git container for remote ops), tests for `Run`, `RunInternal`, `parsePorcelain`, `URLToProjectName`.

### Task 4: Write worktree operations tests

Add tests for `AddWorktree`, `RemoveWorktree`, `ListWorktrees`, `FindMainWorktree`, `CurrentWorktreePath`, `checkedOutBranches`.

### Task 5: Write remote operations tests with testcontainer

Add tests for `FetchOrigin`, `ListRemoteBranches` using the testcontainer as a remote.

### Task 6: Verify coverage >= 90%

Run coverage check, add any missing tests, commit.
