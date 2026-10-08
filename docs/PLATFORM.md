# Platform and portability

The rules in [ARCHITECTURE.md](./ARCHITECTURE.md) also apply to platform code. Release rules are in [RELEASE.md](./RELEASE.md).

## Supported platforms

- Windows 10 or later: AMD64 and ARM64
- Linux AMD64 with glibc 2.38 or later
- macOS 12 or later: ARM64 (Apple silicon) and AMD64 (Intel)

Do not expose other architectures.

## Build targets

- Name each target `<os>-<arch>` after Go's `GOOS` and `GOARCH` values.
- Write each target's output to `dist/<os>-<arch>/`. Development and production builds share this folder.
- Add a new target through its own platform modules. Do not change the shared application and frontend build.

## Platform isolation

A file name or package path must show whether its code is OS-specific. Put OS-specific code in one of two places:

- **`<name>_<os>.go` files in a feature package:** for native behavior that the package needs for its own work. Expose it through package-private functions with platform-neutral signatures. Example: `internal/gamelaunch/launch_windows.go`.
- **`internal/platform/<capability>/`:** for a native capability that the application uses as a feature of its own. Give it a platform-neutral API. Current packages: `protection`, `singleinstance`, `robloxmulti`, and `robloxlogs`.

Rules:

- Keep OS APIs out of shared files.
- To support another OS, add `_<os>.go` files that implement the same package-private functions and platform APIs. Do not change shared code.
- Platform code must not bypass the application service, storage, Roblox integration, or Wails binding boundaries.

## Data storage

The application needs no installer. It keeps all of its data in one data root:

- **Portable:** the folder that contains the executable. `storage/` and `logs/` sit next to the executable, so moving the folder, for example to a USB drive or another computer, moves the vault too.
- **Standard:** the per-user application data folder of the OS, defined by `StandardRoot` in `internal/appdata/standard_<os>.go`. The executable can live anywhere.

Windows and Linux offer both modes. macOS offers only Standard storage. See [macOS](#macos). If the executable folder is the Standard root, only Standard storage is offered.

Both modes use this layout:

```text
storage/
    settings.json
    vault/          # vault.db, vault.key, autounlock.key, recovery files, and vault creation staging
    backups/
    runtime/        # Managed browser runtime
    temp/           # Browser installation staging and temporary browser profiles
logs/
```

Missing subdirectories are created when needed.

### Choosing the data root

`internal/appdata` classifies each root by its contents. There is no marker file.

- **Empty:** no `storage/` folder, or none of `settings.json`, `vault/`, `backups/`, `runtime/`, and `temp/` in it.
- **Structure:** at least one of those entries, but no vault files. `settings.json` and `autounlock.key` do not count as vault files.
- **Incomplete:** only one of `vault.db` and `vault.key`, or files in the vault creation staging folder. Keep it for recovery. Never replace it with a new vault in another root.
- **Vault:** `vault.db` and `vault.key`, with no unfinished vault creation.

At startup, when the OS offers both modes:

1. Neither root has data: the first-run dialog offers Portable or Standard storage.
2. Only one root has a vault: use it without asking. Also use a Standard root that has only an incomplete vault, so the vault dialog can recover it.
3. The Portable root has data but no usable vault: offer to continue there, to create or recover a vault, or to switch to Standard storage.
4. Only the Standard root has data, without a vault: use it.
5. Both roots have vault data, complete or incomplete: never choose. Ask on every launch until the user moves or deletes one copy. Do not change either copy.

Until the user confirms a root:

- The application uses the Portable root provisionally.
- Log entries and default settings stay in memory. `settings.json` is not read or written.
- Vault startup does not run, so automatic unlock, migrations, and settings recovery never touch an unconfirmed root.
- Only stale temporary browser files in `storage/temp/` may be removed.

Choosing an empty provisional root continues in the same process. Any other choice writes the held log entries to the chosen root and restarts with `--storage=<mode>`. The new process waits for the single-instance lock and opens that root without asking.

Deleting `settings.json` resets settings to their defaults. Deleting `autounlock.key` turns off automatic unlock. Logs, backups, runtime data, and temporary data are re-created when needed.

### Moving data

- `vault.db` and `vault.key` work in either mode and on any supported OS. Moving `storage/` keeps the accounts.
- `autounlock.key` is sealed to the device and OS user. A copied `autounlock.key` fails, and the vault asks for the master password until automatic unlock is turned on again.

When the OS offers both modes, Settings > Vault > Storage moves the data to the other root. It moves instead of copying, because a vault in both roots triggers the storage question on every launch.

1. `appdata.CheckMove` requires a complete vault in the current root and no vault files in the target root.
2. The old process closes the vault and releases the single-instance lock. The application restarts with `--move-storage=<mode>`.
3. Before it opens settings, logs, or the vault, the new process copies `settings.json`, `vault/`, and `backups/` to a staging folder in the target `storage/`. It verifies each file with SHA-256.
4. It renames the staged backups, settings, and finally the vault into place. Backups that already exist in the target keep their target copy.
5. It deletes the source vault, then the source settings, backups, browser runtime, and temporary files. Logs stay where they are.

If a step before the vault rename fails, the data stays in the source root and the application starts there. If only the deletion fails, the application starts from the target and asks the user to delete the old copy. A notification shows the result. Automatic unlock keeps working, because the device and user stay the same.

The About page shows the storage mode and data folder. Each launch records the mode, root state, and reason for any storage question in the `application.location` component record. The record never contains the folder path, because paths usually contain the user name.

### Rules

- Resolve every data path through `internal/appdata`. Do not rely on the working directory or on a fixed name or location for the executable folder or data root.
- Do not require machine-wide installation, global environment variables, or system services.
- Write outside the data root only where the OS or Roblox requires it. Remove any temporary data when the operation finishes or at the next safe cleanup.
- Keep the live vault on a local or removable drive. The application rejects network data roots. Cloud-synchronized folders are unsupported, because another process can replace vault files outside the application lock.

## Windows

- Put `golang.org/x/sys/windows`, registry access, Win32 calls, and Windows path assumptions only in `_windows.go` files.
- Do not create persistent registry entries. Reading existing registry values is allowed.
- Automatic unlock protects its key with DPAPI for the current user.
- Joining a game starts the Roblox Player directly or through its protocol.
- Multi-instance support and Roblox process control are available.
- ARM64 is cross-compiled on x64 Windows. SQLCipher needs cgo, so the build needs `aarch64-w64-mingw32-clang` from [llvm-mingw](https://github.com/mstorsjo/llvm-mingw) on `PATH`. Put it after the x64 MinGW, because llvm-mingw also ships a `gcc`.
- Chrome for Testing has no ARM64 build. On ARM64, the managed browser runs x64 Chrome through Windows 11 emulation.

## Linux

- The build links the system GTK 4 and WebKitGTK 6 libraries. Release builds come from Ubuntu 24.04 and need glibc 2.38 or later.
- The GTK program name is the application identifier. WebKitGTK stores its data in `$XDG_DATA_HOME/<program name>`, and the executable name would put that data inside the Standard root.
- Automatic unlock stores its key in the Secret Service.
- Joining a game opens a `roblox-player:` link in Sober or Mocktail through its desktop entry. The Roblox setting chooses the client. Automatic uses the only installed client, or the desktop default for `roblox-player` if both are installed.
- Multi-instance support and Roblox process control are unavailable.
- The managed browser runs Chrome for Testing from `storage/runtime/`. It needs Chrome's system libraries and unprivileged user namespaces for the Chrome sandbox.

## macOS

- The application is a `.app` bundle. Build it on macOS with Xcode command line tools, because the Wails runtime and SQLCipher use cgo.
- `build/darwin/Info.plist` is the bundle template. The build adds the version from `VERSION` and `assets/icon.icns`, then signs the bundle ad hoc.
- Regenerate `assets/icon.icns` from `assets/icon.svg` when the icon changes.
- Portable storage is unavailable (`portableSupported` is `false` in `internal/appdata/standard_darwin.go`). The executable is in `RobloxAccountManager.app/Contents/MacOS/`, so a Portable vault would sit inside the bundle. Writing there breaks the bundle signature, and an in-place update replaces the bundle and would delete the vault.
- The application creates the Standard root on first launch without asking.
- Never write data into the bundle.
- Automatic unlock stores its key in the login Keychain. The key never leaves the Keychain.
- Joining a game opens a `roblox-player:` link with `open -a` in `Roblox.app` from `/Applications` or `~/Applications`. This keeps the authentication ticket away from a bootstrapper that registered the scheme. Without a known installation, Launch Services chooses the app.
- Before each launch, the application deletes the Roblox Player's cookie file. Otherwise, Roblox joins with the account that last signed in to the Roblox app. This signs the Roblox app out.
- Roblox process control lists and closes the current user's `RobloxPlayer`, `RobloxCrashHandler`, and `RobloxStudio` processes. Multi-instance support is unavailable.
- The managed browser runs the macOS Chrome for Testing bundle from `storage/runtime/`. Extraction accepts only relative link targets without `..`, and creates links after all files.
- Chrome keeps running after its last window closes, so the browser coordinator treats closing the last page as a user close. An application crash does not stop the managed browser.
- Single-instance handling locks a file in the per-user temporary folder.
- Release bundles are signed ad hoc and are not notarized. Gatekeeper blocks the first launch until the user allows it in System Settings > Privacy & Security, or opens the bundle from the Finder context menu. Updates installed by the application are not quarantined.
- Developer ID signing and notarization can replace the ad hoc signature without other changes, because the bundle never contains user data.
