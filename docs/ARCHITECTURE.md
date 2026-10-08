# Architecture

Platform rules are in [PLATFORM.md](./PLATFORM.md). UI rules are in [DESIGN.md](./DESIGN.md).

The Svelte frontend is a presentation layer. The Go backend owns application logic, sensitive operations, persistence, Roblox communication, process control, and operating-system integration.

## Frontend

- Keep frontend state to presentation and interaction state, such as open dialogs, selected items, form values, loading indicators, and display errors.
- Do not put business rules, credential handling, database access, Roblox API calls, launch logic, filesystem access, or operating-system behavior in the frontend.
- Do not duplicate backend validation. Add lightweight input feedback only when it improves usability.
- Import backend bindings only through `frontend/src/lib/backend/bridge.ts`.
- Show notifications only through `frontend/src/lib/notifications/notification-center.svelte.ts`.
- Build small components with one responsibility.

## Backend

- Give each package one capability and a small, explicit interface. A new feature needs an obvious home.
- A change to one subsystem must not require changes to unrelated frontend, storage, Roblox, or platform code.
- Keep Wails bindings thin. They validate UI input, call application services, and return UI-safe results. They contain no SQL.
- Storage code does not launch Roblox.
- Roblox API clients do not manipulate UI state.
- Keep platform code out of account and persistence logic. Shared business logic depends on platform-neutral interfaces when practical.

## Persisted formats

- Every persisted format stores a format version. The package that owns the format owns its version constant and its migrations or decoders. A compile-time check fails when their count does not match the version.
- To change a format, increase its version and add one migration or decoder for the previous version.
- Keep every migration and decoder for as long as an older backup can exist. A restored backup migrates on the next unlock.

### Settings

- Each migration transforms the raw JSON object of one version into the raw JSON object of the next version.
- Replace a field with its default when the field has the wrong type, is `null`, or fails validation. Drop unknown fields.
- If the file cannot be parsed, has an unsupported version, or fails a migration, log a warning, move the file to `settings.json.bak`, and start from the defaults.
- Write `settings.json` atomically. Write a temporary file, flush it to disk, and replace the original.

### Vault

The vault holds account data. Never replace vault data with defaults or delete it after a load failure.

- If a vault file is damaged or has an unsupported version, refuse to unlock, leave every file unchanged, and tell the user to restore a backup.
- Before `vault.db` migrates, create a backup in `backups/`. Run every migration in one transaction, then run a foreign key check and an integrity check. If a step fails, roll back and close the vault.
- Rewrite an older `vault.key` only after a password unlock, because the key file version is authenticated together with the wrapped database key.
- If a key file rewrite fails, log it and continue the unlock. The older file stays readable.

## Logging

- Log only through `internal/logging`. Do not create independent loggers or write log files directly.
- Get a module logger from `System.Module`. Do not log without a module.
- Add an `operation` attribute to entries. When one user action produces work in several modules, use one stable operation ID.
- Never log credentials, authentication tokens, cookies, account secrets, or other sensitive account data. The redacting handler is a safety net, not permission.
