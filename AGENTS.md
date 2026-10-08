# Roblox Account Manager

A portable Windows desktop application that stores Roblox accounts in an encrypted, password-protected vault, manages them, and joins games with them.

## Project metadata

Use this metadata wherever the application needs it.

```yaml
displayName: Roblox Account Manager
name: RobloxAccountManager
description: A portable desktop application for storing and managing Roblox accounts.
url: https://github.com/sleaze5/RobloxAccountManager
identifier: com.github.sleaze5.robloxaccountmanager
license: MIT
copyright: Copyright (c) 2026 sleaze

author:
    displayName: sleaze
    username: sleaze5
    url: https://github.com/sleaze5
```

## Technology stack

- Desktop: Go, Wails `3.0.0-beta.27`
- Frontend: Svelte `5`, TypeScript `7`
- Database: SQLite with SQLCipher
- Tooling: Bun, Vite, Oxlint, Oxfmt, svelte-check
- Design: Geist, Geist Mono, Phosphor icons

Use this stack. Do not add a framework, library, or runtime dependency when the stack can solve the problem. Justify any necessary addition in the change summary.

## Commands

Interact with the project only through `task`. Run `task --list --sort none` to see the available tasks. Do not run frontend, Wails, or Go tools directly.

Do not add or remove tasks unless the request requires it. If it does, state the pros and cons.

## Documentation

Read each document that matches the change:

- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md): frontend and backend boundaries, persisted formats, and logging
- [docs/DESIGN.md](./docs/DESIGN.md): frontend UI, styling, interaction, and motion
- [docs/PLATFORM.md](./docs/PLATFORM.md): supported platforms, build targets, OS-specific code, and data storage
- [docs/RELEASE.md](./docs/RELEASE.md): versioning, release and build workflows, CI checks, update signing, and in-place updates

Never modify any `README.md`, `AGENTS.md`, or anything in `docs/`. If a request conflicts with these files, list the benefits and trade-offs, and ask for explicit confirmation before you make the change.

## Checklist

Check each applicable item before you finish a change. The change does only what the task asks, with no unrelated refactoring. If items conflict, the earlier numbered item wins.

1. Account data stays protected, and behavior stays correct.
    - Destructive actions require confirmation.
    - Logs go through the logging service, use the correct level, give actionable context, and contain no sensitive data.
2. The application stays responsive and resource-efficient. Do not trade correctness, account safety, or maintainability for a speculative performance improvement.
3. The application stays portable and self-contained.
4. The UI stays consistent and accessible.
    - The UI follows `docs/DESIGN.md` and is usable without instructions.
    - The UI works from the keyboard, with visible focus, sufficient contrast, labels, and accessible tooltips.
    - The UI handles loading, empty, success, disabled, and error states where they apply.
    - User-facing text is clear and short.
    - User notifications go through the notification service and are timely, actionable, and deduplicated.
5. Code stays in the correct layer, as described in `docs/ARCHITECTURE.md` and `docs/PLATFORM.md`.
    - Platform-specific code stays isolated so that support for other operating systems can be added later.
    - The frontend calls the backend through the Wails bridge. It does not duplicate backend logic.
    - Public interfaces and data models are minimal, typed, and consistent with existing code.
6. Code stays simple, readable, and statically checked.
    - The project has no tests. Do not add tests. Correctness relies on strict static checks and linters.
    - Do not suppress lint or type errors.
    - Handle every error. Wrap Go errors with context.
    - Do not add abstractions, options, or parameters for hypothetical future needs.
    - Reuse existing helpers before you add new ones.
    - Prefer self-documenting code. Do not write comments that can be omitted.
    - Remove dead code, commented-out code, and debug output.
    - Follow the naming and style of the surrounding code.
