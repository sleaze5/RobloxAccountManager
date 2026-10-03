# Architecture

Platform and portability rules are in [PLATFORM.md](./PLATFORM.md). UI rules are in [DESIGN.md](./DESIGN.md).

## System boundary

The Svelte frontend is a presentation layer. The Go backend owns application logic, sensitive operations, persistence, Roblox communication, process control, and operating-system integration.

## Frontend

The frontend renders the interface and collects user input.

- Organize code by feature under `frontend/src/lib/<feature>/`. Each feature holds its components and any feature state in `*-store.svelte.ts` or `*-state.svelte.ts` files. Shared UI primitives and helpers are in `lib/shared/`.
- Build small, focused components with one clear responsibility.
- Import backend bindings only through `frontend/src/lib/backend/bridge.ts`.
- Do not place business rules, credential handling, database access, Roblox API calls, launch logic, filesystem access, or operating-system behavior in the frontend.
- Keep frontend state limited to presentation and interaction state, such as open dialogs, selected items, form values, loading indicators, and display errors.
- Do not duplicate backend validation or business logic. Add lightweight input feedback only when it improves usability.
- Show user notifications through `lib/notifications/notification-center.svelte.ts`.

## Backend

Organize the backend by capability. Each package owns one responsibility and exposes a small, explicit interface.

- **Wails bindings:** `internal/bindings` holds thin, typed adapters that validate UI input, call application services, and return UI-safe results.
- **Application services:** `internal/appservice` holds account workflows, vault behavior, validation, orchestration, and events sent to the frontend.
- **Domain models:** `internal/accounts`, `internal/games`.
- **Storage:** `internal/storage/vault` (SQLCipher database, keys, backups, migrations), `internal/storage/accounts`, `internal/storage/games`.
- **Roblox integration:** `internal/roblox` (HTTP client, sessions, rate limits, error translation), `internal/roblox/services` (endpoint clients).
- **Other external services:** `internal/integration/rovalra`.
- **Game launching:** `internal/gamelaunch` handles launch preparation, command construction, process start, and launch results.
- **Application updates:** `internal/appupdate` checks, downloads, verifies, and installs updates through the Wails updater. See [PLATFORM.md](./PLATFORM.md#releases).
- **Managed browser:** `internal/browser` handles Chrome for Testing runtime installation and isolated, CDP-controlled browser sessions.
- **Roblox Player logs:** `internal/logsexplorer`.
- **Configuration and paths:** `internal/appdata` (portable paths and private files), `internal/appsettings` (`settings.json`), `internal/appmeta` (application identity, `VERSION`, and the update public key).
- **Logging:** `internal/logging`.
- **Native capabilities:** `internal/platform/*`. See [PLATFORM.md](./PLATFORM.md).

A new feature should have an obvious home in this list. Changing one subsystem should not require changes to unrelated frontend, storage, Roblox, or platform code.

## Dependency rules

- Wails bindings must not contain SQL.
- Storage code must not launch Roblox.
- Roblox API clients must not manipulate UI state.
- Platform code must not spread into account or persistence logic.
- Shared business logic depends on platform-neutral interfaces when practical.

## Persisted formats

Every persisted format stores a format version. The package that owns a format also owns its current version constant and its version scaffolding. A compile-time check fails when the number of migrations or decoders does not match the current version.

- `settings.json`: version constant `appsettings.FormatVersion`, scaffolding `migrations` in `internal/appsettings`
- `vault.db`: version constant `schemaVersion`, scaffolding `schemaMigrations` in `internal/storage/vault`
- `vault.key`: version constant `keyFormatVersion`, scaffolding `keyFileDecoders` in `internal/storage/vault`
- `autounlock.key`: version constant `autoFormatVersion`, scaffolding `autoUnlockDecoders` in `internal/storage/vault`

To change a persisted format, increase its version and add one migration or decoder for the previous version. Keep every decoder and migration for as long as an older backup can exist.

### Settings

The application loads `settings.json` in this order:

1. Read the file and parse it as a JSON object.
2. Read `version`. Reject a missing version, a version below 1, and a version above `FormatVersion`.
3. Run each migration in order, from the stored version to `FormatVersion`. Each migration receives the raw JSON object of one version and returns the raw JSON object of the next version.
4. Decode every known field into the current schema. Replace a field with its default when the field has the wrong type, is `null`, or fails validation. Drop unknown fields.
5. Write the canonical result atomically. Write a temporary file, flush it to disk, and replace `settings.json`.

If step 1, 2, or 3 fails, the application logs a warning, moves the file to `settings.json.bak`, and creates `settings.json` with the defaults. Replaced fields are logged as a warning. `appsettings.ReadLogging` runs steps 1 to 4 and does not write the file.

### Vault

The vault holds account data. Never replace vault data with defaults or delete it after a load failure. If a vault file is damaged or has an unsupported version, refuse to unlock, leave every file unchanged, and tell the user to restore a backup.

- **`vault.db`:** Unlock accepts schema versions from 1 to `schemaVersion`. After an unlock, if the stored version is older, the application creates a backup in `backups/`. It then runs every migration in one transaction, records the new version, runs a foreign key check and an integrity check, and commits. If a step fails, the transaction rolls back and the vault closes.
- **`vault.key` and `autounlock.key`:** Each format version has its own decoder. After an unlock, the application rewrites an older `autounlock.key` at the current version. It rewrites an older `vault.key` only after a password unlock, because the key file version is authenticated together with the wrapped database key. The older files stay readable, so a failed rewrite is logged and does not stop the unlock.

Backups keep the format versions that existed when they were created. A restored backup is migrated on the next unlock.

## Logging

All application logs go through `internal/logging`. Do not create independent loggers or write log files directly.

- Each launch writes one JSON log file, `<launch_id>.log`, to `logs/` in the portable application directory. The file rotates at 5 MiB and keeps 5 files.
- Every entry records its timestamp, level, module, and message, plus error details when they apply. Get a module logger from `System.Module`. Do not log without a module.
- Add an `operation` attribute to entries. Use one stable operation ID when one user action produces work in several modules.
- Never log credentials, authentication tokens, cookies, account secrets, or other sensitive account data. The redacting handler masks known sensitive keys, but it is a safety net, not permission.

The levels are `trace`, `debug`, `info`, `warn`, and `error`. `settings.json` stores each level as a boolean in `logging.enabledLevels`, so the configuration can enable all levels, any subset, or none. No enabled level means no log output. The defaults enable only `warn` and `error`. The settings UI controls each level and all levels together.
