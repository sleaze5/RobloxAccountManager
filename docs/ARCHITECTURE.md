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
- Every store action that calls the backend catches its errors and shows them through the store's error state or a notification. A component may call an action without awaiting it, as in `void store.action()`, only when the action handles its own errors.
- Ignore a response that arrives after a newer request or after the vault locked. Check a version or request counter before you apply the response.
- Keep state only when the UI reads it.
- Reuse the shared helpers in `frontend/src/lib/shared/` instead of writing the behavior again: `coalesce.ts` for refreshes that must not overlap, `search.ts` for search terms and matching, `clipboard.ts` for copying and the copy notification, `errors.ts` for error messages, `cancellable-request.ts` for the type of a backend call that can be cancelled, and `text-filter-check.svelte.ts` for Roblox text filter results that apply only while the text is unchanged.
- Give state that belongs to one account's page an `AccountScope` from `frontend/src/lib/accounts/account-scope.ts`. Apply a response only while `isCurrent` is true, pass each request through `track` so that `dispose` cancels it, and call `dispose` when the page closes.

## Backend

- Give each package one capability and a small, explicit interface. A new feature needs an obvious home.
- A change to one subsystem must not require changes to unrelated frontend, storage, Roblox, or platform code.
- Keep Wails bindings thin. They validate UI input, call application services, and return UI-safe results. They contain no SQL.
- Storage code does not launch Roblox.
- Roblox API clients do not manipulate UI state.
- Keep platform code out of account and persistence logic. Shared business logic depends on platform-neutral interfaces when practical.
- Every exported method of the binding service is called by the frontend, and every field of a returned type is read by the frontend or used by the backend. Remove a binding, field, or event in the same change that stops using it.
- `main.go` wires every dependency of a service. Do not check those dependencies for `nil`, and do not add a fallback for a dependency that is always present.
- Give each behavior one implementation. Code that two packages or two operating systems need goes into one shared function or one `_unix.go` or `_other.go` file, as [PLATFORM.md](./PLATFORM.md) describes.
- Add an interface only when it has two implementations or crosses a package boundary that must not import the other package.

## Errors

- Return errors that the UI shows as `*roblox.Error` with a short, complete sentence in `Message` that tells the user what to do. `Error()` returns only `Message`, because the desktop runtime shows it as is. Do not convert or strip error text per binding.
- Never put paths, URLs, cookies, tokens, or internal error text in `Message`. Put the underlying error in `Cause`. The frontend shows only `Message`.
- Log an error once, where it is handled, with an `operation` attribute and the `error`. A `*roblox.Error` logs its endpoint, kind, status, and Roblox code itself.
- Map storage and vault errors to user messages in one place per package, such as `mapRepositoryError` and `safeVaultError`, instead of at each call.

## Concurrency and crash safety

- A resource that several goroutines use, such as the vault database, the backup folder, or a staging folder, has one owner that serializes access to it. Never close, replace, or delete a resource while another operation holds it. Use the owner's existing lock or exclusive operation instead of adding a second one.
- A change that writes several files must leave a state that the next startup can finish or roll back after a crash at any point. Write new files next to the old ones, replace them in an order that keeps one complete set readable, and say why the order is safe in a comment.
- Bound every wait, retry, and poll with a context, a timeout, or a maximum count, and comment why a retry or delay exists.

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
- Outside `internal/storage/vault`, change vault data only through `vault.RunWriteTransaction`. Report whether the write changed data, so that the storage revision increases and the next backup check sees the change.
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
