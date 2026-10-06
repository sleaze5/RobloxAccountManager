# Roblox Account Manager

A portable Windows desktop application that stores Roblox accounts in an encrypted, password-protected vault, manages them, and joins games with them.

## Project metadata

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

## Priorities

Apply these priorities in order when requirements conflict:

1. Protect account data and preserve correct behavior.
2. Keep the application responsive and resource-efficient.
3. Preserve the portable, self-contained distribution model.
4. Maintain a consistent, accessible user interface.
5. Keep platform-specific code isolated so that support for other operating systems can be added later.

Do not trade correctness, account safety, or maintainability for a speculative performance improvement.

## Technology stack

- Go for application logic and native integrations
- Wails `3.0.0-beta.27` for the desktop shell and the Go-to-frontend bridge
- Bun as the only JavaScript runtime and package manager
- Vite 8 with Lightning CSS for frontend builds
- Svelte 5 with TypeScript 7 in strict mode
- Geist and Geist Mono for fonts, bundled locally
- Phosphor for icons, with direct imports from `phosphor-svelte/lib/<Name>Icon`
- SQLite through SQLCipher (`go-sqlcipher`) for the encrypted vault
- Type-aware Oxlint, `svelte-check`, and Oxfmt for frontend validation and formatting

Use this stack. Do not add a framework, library, or runtime dependency when the stack can solve the problem. Justify any necessary addition in the change summary.

## Commands

Use only these commands:

- `task check`: validate frontend and backend source and compile the backend without artifacts
- `task fix`: apply safe formatting and lint fixes, then run `task check`
- `task update`: update unpinned dependencies, then run `task fix`
- `task update-cft-manifest`: update the pinned Chrome for Testing manifest
- `task dev`: start Wails development mode
- `task build:windows-amd64`: build `dist/windows-amd64/RobloxAccountManager.exe`
- `task release:prepare`: sign and verify the update manifest for downloaded release archives. The "Release" workflow runs it; see `docs/PLATFORM.md`.
- `task clean`: delete generated files, builds, logs, and all portable data, including the vault

The commands prepare their own dependencies and generated bindings. Do not run frontend tools, Wails binding generation, or Go formatting, validation, or builds directly.

The project has no tests. Do not add tests.

## Rules

- Do not edit `README.md`.
- Do not edit `AGENTS.md` unless the task asks for it.
- Do not use tables in Markdown. Use lists.
- Vault locking is manual-only. This is an intentional exception to the account-protection priority. Do not lock the vault when the Windows workstation or user session locks.

## Documentation

Read each document that matches the change:

- [docs/DESIGN.md](./docs/DESIGN.md): frontend UI, styling, interaction, and motion
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md): module boundaries, data flow, Wails bindings, persisted formats, logging, and new backend capabilities
- [docs/PLATFORM.md](./docs/PLATFORM.md): Windows integration, paths, processes, the registry, temporary files, builds, and packaging

## Checklist

Check each applicable item before you finish a change:

- The change does only what the task asks. It contains no unrelated refactoring.
- The UI follows `docs/DESIGN.md` and is usable without instructions.
- The UI works from the keyboard, with visible focus, sufficient contrast, labels, and accessible tooltips.
- The UI handles loading, empty, success, disabled, and error states where they apply.
- User-facing text is clear and short.
- Destructive actions require confirmation.
- Responsibilities stay in the correct layer, as described in `docs/ARCHITECTURE.md` and `docs/PLATFORM.md`.
- The application stays portable and self-contained.
- The frontend calls the backend through the Wails bridge. It does not duplicate backend logic.
- Public interfaces and data models are minimal, typed, and consistent with existing code.
- Logs go through the logging service, use the correct level, give actionable context, and contain no sensitive data.
- User notifications go through the notification service and are timely, actionable, and deduplicated.
