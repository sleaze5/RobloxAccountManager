# Roblox Account Manager

A portable Windows desktop application that stores Roblox accounts in an encrypted, password-protected vault, manages them, and joins games with them.

## Project metadata

Use this metadata wherever the application needs it.

```yaml
displayName: Roblox Account Manager
name: RobloxAccountManager
description: A portable desktop application for storing and managing Roblox accounts.
url: https://github.com/sleaze5/RobloxAccountManager
discordInviteCode: BUchHun3Uc
identifier: com.github.sleaze5.robloxaccountmanager
license: GPL-3.0-only
copyright: Copyright (c) 2026 sleaze

author:
    displayName: sleaze
    username: sleaze5
    url: https://github.com/sleaze5
```

## Technology stack

- Desktop: Go, Wails `3`
- Frontend: Svelte `5`, TypeScript `7`
- Database: SQLite through the pure Go `ncruces/go-sqlite3` driver, encrypted by its AES-256-XTS VFS
- Tooling: Bun, Vite, Oxlint, Oxfmt, svelte-check, golangci-lint
- Design: Tailwind CSS `4`, Geist, Geist Mono, Phosphor icons

Use this stack. Do not add a framework, library, or runtime dependency when the stack can solve the problem. Justify any necessary addition in the change summary.

## Commands

Interact with the project only through Task, which Go runs from the version pinned in `build/tools/go.mod`: `go tool -modfile build/tools/go.mod task <name>`. Run `go tool -modfile build/tools/go.mod task --list --sort none` to see the available tasks. Do not run frontend, Wails, or Go tools directly.

Run `go tool -modfile build/tools/go.mod task build` to fix, validate, and build the host target, and add `TARGET=<target>` for another target. Each `build:<target>` task only compiles an already built production frontend.

Do not add or remove tasks unless the request requires it. If it does, state the pros and cons.

## Documentation

Read each document that matches the change. Every change that adds or changes text must follow [docs/WRITING.md](./docs/WRITING.md) and [docs/GLOSSARY.md](./docs/GLOSSARY.md).

- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md): frontend and backend boundaries, persisted formats, and logging
- [docs/DESIGN.md](./docs/DESIGN.md): frontend UI, styling, interaction, and motion
- [docs/GIT.md](./docs/GIT.md): Git authorship, commits, pull requests, and merging
- [docs/PLATFORM.md](./docs/PLATFORM.md): supported platforms, build targets, OS-specific code, and data storage
- [docs/RELEASE.md](./docs/RELEASE.md): versioning, release and build workflows, CI checks, update signing, and in-place updates
- [docs/WRITING.md](./docs/WRITING.md): all text, such as UI text, errors, logs, comments, commits, and pull requests, and the writing checklist
- [docs/GLOSSARY.md](./docs/GLOSSARY.md): the one term for each concept and the one verb for each action

Never modify any `README.md`, `AGENTS.md`, or anything in `docs/`. If a request conflicts with these files, list the benefits and trade-offs, and ask for explicit confirmation before you make the change.

There is one exception. You may update an entry in a list of shared parts when your change adds the part, removes it, renames it, or makes the entry wrong. These lists are the building blocks in `docs/DESIGN.md`, the shared helpers in `docs/ARCHITECTURE.md`, and the terms in `docs/GLOSSARY.md`. Change only the entry. Never add, change, or remove a rule this way. State each entry that you changed in the change summary.

## Naming

Names exist so that a reader understands what the code does without reading its implementation. A name that is too long is better than a name that is too short or vague. Apply these rules to every name: functions, methods, variables, parameters, receivers, fields, types, interfaces, constants, globals, locals, exports, files, events, and CSS classes.

- Name a thing by what it represents or does in the problem domain, not by its type or how it works: `accountCookie`, not `str` or `data`.
- Name a concept of the app with its term from [docs/GLOSSARY.md](./docs/GLOSSARY.md), such as `group`, not `community`, and `serverID`, not `jobID`. This applies also when a Roblox API uses another name. Where the code reads or writes the Roblox name, add a comment that names both, such as "Roblox calls the server ID jobId."
- Write full words. Do not abbreviate: `account`, `configuration`, `password`, `message`, not `acct`, `cfg`, `pw`, `msg`. Only these short forms are allowed:
    - Acronyms that are words of their own: `ID`, `URL`, `API`, `HTTP`, `JSON`, `UI`, `CDP`, `DPAPI`, and file format names. In Go, write acronyms in one case (`accountID`, `HTTPClient`). In TypeScript, follow the existing generated bindings (`accountId`).
    - Go's `ctx` for a `context.Context` and `err` for an `error`, because the full words shadow the package and the built-in type. Write compound names in full: `readError`, `startupContext`, `importMutex`.
    - `x` and `y` for coordinates, and `x1`, `y1`, `x2`, `y2` for curve control points.
    - The `Err` prefix of Go sentinel errors, such as `ErrLocked`, which golangci-lint requires.
- Never use single letters, `tmp`, `temp`, `val`, `value`, `data`, `info`, `item`, `result`, `obj`, or numbered names (`file1`) when a more specific name exists. Name loop variables after what they hold (`for _, account := range accounts`, `for index := range accounts`).
- Make a name match the behavior completely. A function named `get` or `read` must not change state. A function that does two things names both, or is split. A validator names everything it checks: `validateAccountAndTargetIDs`, not `socialTarget`. Rename a name when its behavior changes.
- Start a function or method with a verb that says what it does: `refreshAccountMetadata`, `removeManagedDirectory`, `validateCookie`.
- Start a boolean with `is`, `has`, `can`, `should`, or `was`, and state it positively: `isUnlocked`, not `notLocked`. Existing Go getters such as `Unlocked()` may keep their form.
- Name a collection in the plural (`accounts`) and a map by both sides (`accountByRobloxUserID`, `imagesByUserID`).
- Name a decoded API response `payload`. When an entry of a response becomes a domain value, give the entry a different name than the value: `listedServer` becomes `server`.
- Name the outcome of a database statement after the statement: `insertion`, `update`, or `deletion`.
- Put computed qualifiers at the end, and include units where a value has one: `retryDelay`, `accountCount`, `lockTimeout`, `maxResponseSizeBytes`, `intervalSeconds`.
- Use matched opposites consistently: `begin`/`end`, `open`/`close`, `add`/`remove`, `source`/`destination`, `previous`/`next`.
- Use one verb per concept across the codebase, such as `get`, `read`, or `load`, and keep it consistent.
- Give each variable one purpose. Do not reuse a variable for a second meaning, and do not encode special meanings in values, such as `-1` for an error.
- Do not use generic type names such as `Manager`, `Helper`, `Util`, or `Data` for something new. Name it after its one capability.
- Do not add temporal or implementation words: `newAccountStore`, `accountStoreV2`, `accountsMap`, unless two variants exist in the same scope.
- If no accurate name fits, the code probably does too much. Split it before you name it.

## Checklist

Check each applicable item before you finish a change. The change does only what the task asks, with no unrelated refactoring. If items conflict, the earlier numbered item wins.

1. Account data stays protected, and behavior stays correct.
    - Destructive actions require confirmation.
    - Logs go through the logging service, use the correct level, give actionable context, and contain no sensitive data.
2. The application stays responsive and resource-efficient. Do not trade correctness, account safety, or maintainability for a speculative performance improvement.
3. The application stays portable and self-contained.
4. The UI stays consistent and accessible.
    - The UI follows `docs/DESIGN.md` and is usable without instructions.
    - Every element has a purpose, and each screen reads as one system. Give a role the pattern it already has elsewhere, such as the profile header with its actions at the top right, `ProfileStats` for headline numbers, `ProfileTabs` for the pages of a profile, `CreatorByline` for a creator or owner, and `facts` for details. Do not box information into cards, tiles, or grids of cells only to group it.
    - A link to a user, a group, or a game opens its preview through `getProfilePreviews().show`, not another page. The preview opens the full profile on its own page on request.
    - The UI works from the keyboard, with visible focus, sufficient contrast, labels, and accessible tooltips.
    - The UI handles loading, empty, success, disabled, and error states where they apply.
    - User-facing text follows `docs/WRITING.md`: it is short, clear to a new user, and precise for a power user.
    - User notifications go through the notification service and are timely, actionable, and deduplicated.
5. Code stays in the correct layer, as described in `docs/ARCHITECTURE.md` and `docs/PLATFORM.md`.
    - Platform-specific code stays isolated so that support for other operating systems can be added later. Code that is the same on several operating systems lives once, in a file with a build constraint, not in each `_<os>.go` file.
    - Code that changes together lives together. A feature keeps its components, state, and helpers in its own folder, and `frontend/src/lib/shared/` holds only what several features use.
    - The frontend calls the backend through the Wails bridge. It does not duplicate backend logic.
    - Public interfaces and data models are minimal, typed, and consistent with existing code. Export only what another module uses.
    - Each feature's Wails bindings go in their own file in `internal/bindings`, such as `inventory.go`.
6. Code stays simple, readable, and statically checked.
    - The project has no tests. Do not add tests. Correctness relies on strict static checks and linters.
    - Do not suppress lint or type errors.
    - Handle every error. Wrap Go errors with context. A catch or a discarded error that does nothing says in a comment why the state stays valid, such as "The backend logs the failure."
    - Do not add abstractions, options, or parameters for hypothetical future needs. Add a parameter only when two callers pass different values.
    - Do not choose behavior by comparing a string such as an endpoint name. Pass the difference as a parameter.
    - Search for an existing helper before you write one, and reuse or extend it. Never leave two implementations of the same behavior. Define a type or value that several modules need once, in `frontend/src/lib/shared/` or the Go package that owns it.
    - Name everything as described in [Naming](#naming).
    - All text, including comments, logs, errors, commits, and pull requests, follows `docs/WRITING.md` and uses the terms in `docs/GLOSSARY.md`. Complete the writing checklist in `docs/WRITING.md`.
    - Prefer self-documenting code. Write a comment only for a fact the code cannot show, such as an operating system or Roblox behavior, a format layout, or the reason for an ordering. Do not write comments that restate the code, including a doc comment that only repeats the name and signature of a function, type, field, or prop.
    - Remove dead code, commented-out code, and debug output. When a change makes a function, field, binding, event, or parameter unused, remove it in the same change.
    - Follow the style of the surrounding code.
7. The change is verified.
    - Run `go tool -modfile build/tools/go.mod task build`, or at least `task fix`, and read the full output. Do not report a change as done when a step failed or did not run.
    - Before you call code unused, search the whole repository, including `frontend/src`, every `_<os>.go` file, and string references such as event names.
    - Report what you could not verify, such as code for another operating system.
