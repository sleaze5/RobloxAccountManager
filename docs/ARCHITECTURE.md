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

### Export files

An export file holds the vault data and settings so that the user can import them into any vault on any supported operating system. The user chooses where to save it and which file to open in the operating system's file dialogs. Both dialogs start in `export/` in the data root. The application creates that folder before it opens a dialog, and starts in the data root if it cannot.

- Keep everything an import needs inside the file. Do not depend on the file name, timestamps, permissions, or other filesystem metadata, because they can change when the file moves between computers or file systems. Write every number in a fixed byte order.
- `internal/storage/vault` owns the encrypted file format. Every version starts with the same prefix: the magic, the format version, and the creation time. The creation time never depends on filesystem timestamps. The rest of the version 1 header holds the Argon2id parameters, the salt, and the nonce. AES-256-GCM encrypts the data and authenticates the whole header. The passcode follows the master password rules.
- `internal/storage/transfer` owns the JSON data inside the file. Each migration transforms the raw JSON object of one version into the raw JSON object of the next version. The settings inside keep their own version and use the settings migrations.
- Export every portable setting, including options for other operating systems. Leave out values that belong to one computer, such as the Roblox Player path. On import, save options for other operating systems without using them.
- Import decrypts, migrates, and validates the data in memory before it writes anything.
- Import writes the vault data and the selected settings in one vault transaction. The settings go into the `pending_settings_import` table. After the commit, the application saves them to `settings.json` and clears the table. If that fails, or the application stops first, it saves them after the next unlock.
- Before an Overwrite import, back up the current vault. If the backup fails, import nothing.
- Reject a file or data version that is newer than the supported version, and tell the user to update the application.
- Keep every export decoder and migration, because an export file can exist indefinitely.

## Logging

- Log only through `internal/logging`. Do not create independent loggers or write log files directly.
- Get a module logger from `System.Module`. Do not log without a module.
- Add an `operation` attribute to entries. When one user action produces work in several modules, use one stable operation ID.
- Never log credentials, authentication tokens, cookies, account secrets, or other sensitive account data. The redacting handler is a safety net, not permission.
