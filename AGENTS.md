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
license: MIT
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

Read each document that matches the change:

- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md): frontend and backend boundaries, persisted formats, and logging
- [docs/DESIGN.md](./docs/DESIGN.md): frontend UI, styling, interaction, and motion
- [docs/GIT.md](./docs/GIT.md): Git authorship, commits, pull requests, and merging
- [docs/PLATFORM.md](./docs/PLATFORM.md): supported platforms, build targets, OS-specific code, and data storage
- [docs/RELEASE.md](./docs/RELEASE.md): versioning, release and build workflows, CI checks, update signing, and in-place updates

Never modify any `README.md`, `AGENTS.md`, or anything in `docs/`. If a request conflicts with these files, list the benefits and trade-offs, and ask for explicit confirmation before you make the change.

## Naming

Names exist so that a reader understands what the code does without reading its implementation. A name that is too long is better than a name that is too short or vague. Apply these rules to every name: functions, methods, variables, parameters, receivers, fields, types, interfaces, constants, globals, locals, exports, files, events, and CSS classes.

- Name a thing by what it represents or does in the problem domain, not by its type or how it works: `accountCookie`, not `str` or `data`.
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
- Give each variable one purpose. Do not reuse a variable for a second meaning, and do not encode special meanings in values, such as `-1` for an error.
- Do not use generic type names such as `Manager`, `Helper`, `Util`, or `Data` for something new. Name it after its one capability.
- Do not add temporal or implementation words: `newAccountStore`, `accountStoreV2`, `accountsMap`, unless two variants exist in the same scope.
- If no accurate name fits, the code probably does too much. Split it before you name it.

## Design principles

Design the interface as one system, not as a set of components that each look good alone. Make every visual and interactive decision on purpose, take it from the shared tokens, and judge it against the whole screen. `docs/DESIGN.md` defines the concrete values: tokens, scales, building blocks, states, and motion limits. These principles say how to apply those values and how to decide a case they do not cover. When a principle and `docs/DESIGN.md` disagree on a value, follow `docs/DESIGN.md`.

1. **Geometric harmony.** Derive every corner radius, inset, border width, and proportion from the elements around it. Keep nested corners concentric: inner radius = outer radius − padding − border width. Give each component family one radius, and do not add a shape or radius that the radius scale does not contain.
2. **Optical precision.** Align by perceived edges and visual weight, not by bounding boxes. Center asymmetric glyphs, such as carets, arrows, and play icons, optically inside their containers. Align adjacent text on a shared baseline. Balance icon size and stroke against the weight of the text beside it. Adjust padding until it looks even, for example beside a leading icon, using only steps from the spacing scale.
3. **System-wide consistency.** Use one visual language: one spacing scale, one type scale, one set of color roles, one elevation order, one radius scale, and one set of durations and easing curves. Take every value from a token or stock scale. Treat a one-off value as a defect. Give the same role the same appearance and behavior on every screen.
4. **Intentional microdetails.** Treat hairline borders, separators, icon stroke weight, shadows, focus rings, and the hover, pressed, selected, focus, and disabled states as part of the design, not as finishing. Small inconsistencies compound into an interface that feels assembled.
5. **Environmental harmony.** Make the interface feel native to a dense desktop tool. Match desktop density, window chrome, pointer and keyboard conventions, context menus, and scroll behavior. Do not import web page patterns, such as hero sections, oversized cards, gradients, or translucent glass surfaces.
6. **Refinement over decoration.** Give every element a function. Leave out ornament: gradients, glows, blur, decorative illustration, and shadows used only for depth. Build hierarchy from weight, color step, spacing, and grouping. Restraint is the result of refinement, not a lack of detail.
7. **Interaction craftsmanship.** Make every interaction show clear cause and effect. Respond to input immediately. Keep spatial continuity: a surface appears from its trigger and returns to it. Let entrances decelerate and make exits faster than entrances. Make feedback proportionate to the action.
8. **Relentless quality control.** Inspect every change at the real window size and in a magnified screenshot. Find and fix inconsistent radii, uneven spacing, near-miss alignments, unbalanced visual weight, broken proportions, and patterns that do not match the rest of the application before you call the work done.

The goal is design inevitability: nothing looks arbitrary, disconnected, or accidentally assembled. Each component reads as a natural extension of the system, and each curve, space, interaction, and treatment follows the same logic. Do not optimize one component at the expense of the whole experience. Do not stop at an interface that works or looks good. Refine it until the whole experience is coherent, precise, and complete.

### Design review checklist

Review the whole screen first, then each component in each of its states. Check at 900px by 500px and 100% display scaling, then inspect details in a 2× screenshot.

System and tokens:

- [ ] Every color, spacing, size, radius, text size, shadow, duration, and easing comes from a token or a stock scale.
- [ ] The change reuses an existing building block or pattern. A new pattern is named and justified.
- [ ] Each role looks and behaves the same everywhere it appears.

Geometry:

- [ ] Each element uses the radius that its role defines.
- [ ] Nested corners are concentric: inner radius = outer radius − inset.
- [ ] Border weights are consistent for each role: 1px for boundaries and separators, 2px only for indicators.
- [ ] Control heights, row heights, and thumbnail sizes match across adjacent components.

Layout and spacing:

- [ ] Spacing follows the 4px scale, and related items sit closer together than unrelated ones.
- [ ] Edges share alignment axes across components, with no near-misses of 1px or 2px.
- [ ] Sibling surfaces use the same gutters and inner padding.
- [ ] The layout holds at the minimum window size, truncates in a deliberate order, and does not shift while content loads.

Typography:

- [ ] Every text element uses the size, weight, and color that its role defines.
- [ ] The hierarchy still reads when you squint at the screen.
- [ ] Adjacent text shares a baseline, and technical values use the monospace font.

Optical balance:

- [ ] Icons look centered in their containers, including asymmetric glyphs.
- [ ] Icon size and stroke weight balance the text beside them.
- [ ] Controls in one group carry equal visual weight, whether they show an icon, a label, or both.
- [ ] Padding looks even, not only measures even.

Color, contrast, and elevation:

- [ ] Surfaces follow the elevation order from content to chrome to floating surfaces to raised controls.
- [ ] Text has 4.5:1 contrast, and icons, boundaries, and focus rings have 3:1 contrast.
- [ ] Semantic color carries meaning only, and never as the only cue.

States:

- [ ] Every interactive element has each state that applies: default, hover, pressed, focus-visible, selected, disabled, loading, and error.
- [ ] The same state looks the same across components.
- [ ] The focus ring is visible on every surface and is never clipped.
- [ ] Loading, empty, success, and error states are designed, not left to defaults.

Motion and feedback:

- [ ] Every state change gives immediate, proportionate feedback.
- [ ] Each transition uses the duration and easing of its role. Entrances decelerate, and exits are faster.
- [ ] Surfaces enter from and exit toward their trigger.
- [ ] Under reduced motion, every state stays understandable.

Platform fit and restraint:

- [ ] Density, chrome, and keyboard behavior feel native to a desktop application, and nothing looks like exposed browser UI.
- [ ] Every element has a purpose. Decoration, redundant labels, and repeated values are gone.
- [ ] Each surface has at most one dominant action.

Whole experience:

- [ ] The screen shares its anatomy, alignment axes, and spacing rhythm with neighboring screens.
- [ ] No component was refined in isolation at the cost of the whole.

## Checklist

Check each applicable item before you finish a change. The change does only what the task asks, with no unrelated refactoring. If items conflict, the earlier numbered item wins.

1. Account data stays protected, and behavior stays correct.
    - Destructive actions require confirmation.
    - Logs go through the logging service, use the correct level, give actionable context, and contain no sensitive data.
2. The application stays responsive and resource-efficient. Do not trade correctness, account safety, or maintainability for a speculative performance improvement.
3. The application stays portable and self-contained.
4. The UI stays consistent and accessible.
    - The UI follows `docs/DESIGN.md` and the [Design principles](#design-principles), passes the [Design review checklist](#design-review-checklist), and is usable without instructions.
    - The UI works from the keyboard, with visible focus, sufficient contrast, labels, and accessible tooltips.
    - The UI handles loading, empty, success, disabled, and error states where they apply.
    - User-facing text is clear and short.
    - User notifications go through the notification service and are timely, actionable, and deduplicated.
5. Code stays in the correct layer, as described in `docs/ARCHITECTURE.md` and `docs/PLATFORM.md`.
    - Platform-specific code stays isolated so that support for other operating systems can be added later.
    - The frontend calls the backend through the Wails bridge. It does not duplicate backend logic.
    - Public interfaces and data models are minimal, typed, and consistent with existing code. Export only what another module uses.
    - Each feature's Wails bindings go in their own file in `internal/bindings`, such as `inventory.go`.
6. Code stays simple, readable, and statically checked.
    - The project has no tests. Do not add tests. Correctness relies on strict static checks and linters.
    - Do not suppress lint or type errors.
    - Handle every error. Wrap Go errors with context.
    - Do not add abstractions, options, or parameters for hypothetical future needs. Add a parameter only when two callers pass different values.
    - Do not choose behavior by comparing a string such as an endpoint name. Pass the difference as a parameter.
    - Search for an existing helper before you write one, and reuse or extend it. Never leave two implementations of the same behavior. Define a type or value that several modules need once, in `frontend/src/lib/shared/` or the Go package that owns it.
    - Name everything as described in [Naming](#naming).
    - Prefer self-documenting code. Write a comment only for a fact the code cannot show, such as an operating system or Roblox behavior, a format layout, or the reason for an ordering. Do not write comments that restate the code.
    - Remove dead code, commented-out code, and debug output. When a change makes a function, field, binding, event, or parameter unused, remove it in the same change.
    - Follow the style of the surrounding code.
7. The change is verified.
    - Run `go tool -modfile build/tools/go.mod task build`, or at least `task fix`, and read the full output. Do not report a change as done when a step failed or did not run.
    - Before you call code unused, search the whole repository, including `frontend/src`, every `_<os>.go` file, and string references such as event names.
    - Report what you could not verify, such as code for another operating system.
