---
name: debugging
description: Systematic debugging and troubleshooting methodology. Use when encountering errors, bugs, unexpected behavior, build failures, test failures, or runtime exceptions in any programming language or framework.
license: MIT
compatibility: Requires read_file, exec_shell, grep tools
metadata:
  author: danqing-teams
  version: "1.0"
---

# Debugging Skill

Systematic approach to identifying and fixing bugs in software projects.

## Debugging Process

### 1. Reproduce the Issue

- Read any error messages, stack traces, or logs the user has provided.
- Run the failing command or test to reproduce the error yourself.
- Note the exact error message, file, and line number.

### 2. Isolate the Problem

- Identify the precise code path that leads to the error.
- Use `grep` to find related definitions, callers, or configuration.
- Narrow down to the smallest reproducible case possible.

### 3. Analyze Root Cause

- Read the relevant source files thoroughly to understand the logic.
- Check recent changes or commits that might have introduced the issue.
- Look for common patterns: null/nil pointer, type mismatch, race condition, missing imports, environment differences.

### 4. Formulate and Test a Fix

- Propose a minimal fix that addresses the root cause, not just the symptom.
- Prefer `edit` for targeted changes over rewriting entire files.
- After applying the fix, re-run the failing command to verify.

### 5. Validate No Regression

- Run existing tests if available: `make test` or equivalent.
- Check that related functionality still works as expected.
- If the project has lint/typecheck, run those too.

## Common Patterns

### Interpreting Stack Traces

- Start from the bottom (your code) and work up.
- Identify the first frame in your project's code base.
- Read the error message carefully — it often tells you exactly what's wrong.

### Build Failures

- Check the compiler/linker output for the first error (subsequent errors are often cascading).
- Verify dependencies are installed correctly.
- Check for missing build tags, platform-specific code, or version mismatches.

### Test Failures

- Read the test name and assertion message.
- Understand what the test is verifying before making changes.
- Consider whether the test expectation or the implementation needs to change.

### Runtime Errors

- Check log files for context surrounding the error.
- Verify configuration files and environment variables.
- For time-sensitive bugs, add timing-related checks.

## Reference Material

See `references/patterns.md` for language-specific debugging patterns.
