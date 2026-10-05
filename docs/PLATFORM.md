# Platform and portability

Architecture rules are in [ARCHITECTURE.md](./ARCHITECTURE.md). They also apply to platform code.

## Platform support

Support Windows AMD64, Windows ARM64, Linux AMD64, macOS ARM64 (Apple silicon), and macOS AMD64 (Intel). Do not expose other architectures.

Name each build target `<os>-<arch>` with Go's `GOOS` and `GOARCH` values. Expose it as `task build:<os>-<arch>` and write its output to a matching subdirectory of `dist/`. Development and production builds share the target directory:

```text
dist/
    windows-amd64/
        RobloxAccountManager.exe
    windows-arm64/
        RobloxAccountManager.exe
    linux-amd64/
        RobloxAccountManager
    darwin-arm64/
        RobloxAccountManager.app
    darwin-amd64/
        RobloxAccountManager.app
```

Add another target through its own platform tasks and modules. Keep the shared application and frontend task graph.

### Windows

Automatic unlock protects its key with DPAPI for the current Windows user. Joining a game can start the Roblox Player directly or through its protocol. Multi-instance support and Roblox process control are available.

The ARM64 build is cross-compiled on x64 Windows. SQLCipher needs cgo, so `task build:windows-arm64` needs `aarch64-w64-mingw32-clang` from [llvm-mingw](https://github.com/mstorsjo/llvm-mingw) on `PATH`. Put it after the x64 MinGW, because llvm-mingw also ships a `gcc`. Chrome for Testing has no Windows ARM64 build, so the managed browser runs the x64 Chrome through the emulation of Windows 11.

### Linux

A Linux build links the system GTK 4 and WebKitGTK 6 libraries. Release builds come from Ubuntu 24.04 and need glibc 2.38 or later. Automatic unlock stores its encryption key in the Secret Service. Joining a game opens a `roblox-player:` link in Sober or Mocktail through that client's desktop entry. The Roblox setting chooses the client when both are installed. Automatic uses the only installed client, or the desktop default for `roblox-player` when both are installed. The managed browser runs Chrome for Testing from `storage/runtime/`. It needs the system libraries that Chrome requires and unprivileged user namespaces for the Chrome sandbox. Multi-instance support and Roblox process control are unavailable on Linux. The GTK program name is the application identifier, because WebKitGTK keeps its own data in `$XDG_DATA_HOME/<program name>`, and the executable name would place it in the Standard root.

### macOS

The application is a normal `.app` bundle and needs macOS 12 or later. Builds run on macOS with Xcode command line tools, because the Wails runtime and SQLCipher use cgo. `build/darwin/Info.plist` is the bundle template. The build writes the version from `VERSION` into it, copies `assets/icon.icns`, and signs the bundle ad hoc. `assets/icon.icns` is generated from `assets/icon.svg`. Regenerate it when the icon changes.

- Data always uses Standard storage in `~/Library/Application Support/RobloxAccountManager`. macOS does not offer Portable storage and never writes data into the bundle, so the bundle stays replaceable and signable.
- Automatic unlock stores its encryption key in the login Keychain. The key never leaves that Keychain.
- Joining a game opens a `roblox-player:` link with `open -a` in `Roblox.app` from `/Applications` or `~/Applications`, so a bootstrapper that registered the scheme does not receive the authentication ticket. Without a known installation, Launch Services chooses the app. Before each launch, the application deletes `~/Library/HTTPStorages/com.roblox.RobloxPlayer.binarycookies`. Roblox otherwise joins with the account that last signed in to the Roblox app instead of the selected account. This signs the Roblox app itself out.
- Roblox Player logs are read from `~/Library/Logs/Roblox`.
- Roblox process control lists and closes the current user's `RobloxPlayer`, `RobloxCrashHandler`, and `RobloxStudio` processes. Multi-instance support is unavailable.
- The managed browser runs the macOS Chrome for Testing bundle from `storage/runtime/`. Its archive contains relative links inside the bundle. Extraction accepts only relative link targets without `..`, and creates them after all files. Chrome on macOS keeps running after its last window closes, so the browser coordinator treats the close of the last page as a user close. An application crash does not stop a running managed browser.
- Single-instance handling holds an exclusive lock on a file in the per-user temporary folder.

Release bundles carry an ad hoc signature and are not notarized. Gatekeeper blocks the first launch of a downloaded bundle until the user allows it in System Settings → Privacy & Security, or opens it from the Finder context menu. Updates installed by the application are not quarantined. A Developer ID signature and notarization can replace the ad hoc signature without other changes, because the bundle never contains user data.

## Releases

`internal/appmeta/VERSION` is the only source of the application version. It uses `MAJOR.MINOR.PATCH` with an optional `-PRERELEASE` suffix. Go embeds it, the frontend reads it as `APP_VERSION`, and the build writes it into the Windows version resource. Do not write the version anywhere else.

To release, change `VERSION`, run `task fix`, push to `main`, and run the manual "Release" workflow in `.github/workflows/release.yml`. The workflow first checks that it runs on `main` and that tag `v<version>` does not exist. Then it builds Linux on Ubuntu 24.04, macOS on macOS 15, and Windows on Windows Server 2025 in parallel. The macOS job builds both architectures on Apple silicon. The Windows job builds x64 and cross-compiles ARM64 with a pinned, checksum-verified llvm-mingw release. A build fails when it changes source files. The publish job signs the manifest and creates the release only after every build succeeds. The Windows jobs pause Defender real-time scanning, because it slows builds on the disposable runners. The workflow publishes:

- `RobloxAccountManager.exe`: the Windows executable for manual download.
- `RobloxAccountManager-windows-x64.zip`: the Windows updater artifact.
- `RobloxAccountManager-arm64.exe`: the Windows ARM64 executable for manual download.
- `RobloxAccountManager-windows-arm64.zip`: the Windows ARM64 updater artifact.
- `RobloxAccountManager-linux-x64.tar.gz`: the Linux updater artifact and manual download. The archive keeps the executable permission that a bare download loses.
- `RobloxAccountManager-macos-arm64.zip` and `RobloxAccountManager-macos-x64.zip`: the macOS updater artifacts and manual downloads. Each contains `RobloxAccountManager.app`, packed with `ditto` to keep modes, links, and the signature.
- `manifest.json`: the signed Wails update manifest. List only archives in it, because Wails treats every `.exe` as a Windows artifact.
- `THIRD_PARTY_LICENSES.txt`: the license texts of the bundled third-party works. It ships as its own asset, because each archive must contain only the executable.

Each archive must contain only the executable, or on macOS only the application bundle, because the updater accepts exactly one top-level entry. Wails reads the platform and architecture from each archive name, so keep the `<os>-x64`, `windows-arm64`, and `macos-<arch>` parts. Each installed copy updates within its own architecture. An x64 copy that runs through emulation on ARM64 keeps updating to x64.

The application reads `releases/latest/download/manifest.json`, so it never offers a prerelease.

### Update signing

- The `UPDATER_PRIVATE_KEY` repository secret signs `manifest.json`. Never commit the private key.
- `internal/appmeta/updater.key.pub` is the public key that every build embeds. The workflow verifies each manifest with it before it publishes.
- `internal/appupdate` rejects updates without a signature. Keep this check, because the Wails updater installs unsigned manifest entries.
- Replace `updater.key.pub` only to rotate a lost or exposed key. Installed copies with the old key cannot verify later updates.

### In-place updates

- `appupdate.HandleHelperMode()` must run first in `main()`, before the single-instance check. The update helper is this executable, and it starts while the old version still runs.
- The updater replaces only the executable, or on macOS only the application bundle. The data root stays unchanged in both storage modes, because it is never inside what the updater replaces.
- On Linux, the helper copies the downloaded update into a `wails-update-robloxaccountmanager-*` directory next to the executable before the swap. The Wails helper renames the update over the executable, and a rename fails across filesystems. The application removes the original download when it quits for the update.
- On macOS, the helper copies the extracted bundle into a `wails-update-robloxaccountmanager-*` directory next to the installed bundle for the same reason.
- `internal/appupdate` removes `RobloxAccountManager.exe.old.*`, `wails-update-robloxaccountmanager-*` directories next to the executable or bundle, and the helper's `wails-update-*.log` when the new version starts. It deletes a downloaded update that was not installed when the application closes.

## Platform isolation

Keep a clear line between OS-specific code and shared code. A reader must be able to tell from the file name or package path whether code is OS-specific.

OS-specific code lives in one of two places:

- **OS-specific files in a feature package.** When a feature package needs native behavior for its own work, put that behavior in `<name>_<os>.go` files in the same package. Expose it to the rest of the package through package-private functions with platform-neutral signatures. Examples: `internal/appdata/permissions_windows.go`, `internal/appdata/permissions_linux.go`, `internal/appdata/standard_darwin.go`, `internal/browser/process_windows.go`, `internal/browser/process_linux.go`, `internal/gamelaunch/launch_windows.go`, `internal/gamelaunch/launch_linux.go`, `internal/logging/crash_windows.go`, `internal/logging/crash_linux.go`.
- **Packages under `internal/platform/`.** A native capability that the application uses as a feature of its own goes in `internal/platform/<capability>/`, with a platform-neutral API. Current packages: `protection` (protects data for the current user with DPAPI, the Secret Service, or the Keychain), `singleinstance`, `robloxmulti` (Roblox multi-instance support and process control), and `robloxlogs` (Roblox Player log directories).

Rules:

- Keep OS APIs out of shared files. `golang.org/x/sys/windows`, registry access, Win32 calls, and Windows path assumptions belong only in `_windows.go` files.
- To support another OS, add `_<os>.go` files that implement the same package-private functions and platform package APIs. Do not change shared code to do it.
- Platform code must not bypass the application service, storage, Roblox integration, or Wails binding boundaries.

## Data storage

The application needs no installer. It keeps all of its data in one data root, in one of two storage modes:

- **Portable:** the data root is the folder that contains the executable. Moving the folder moves the application and its data.
- **Standard:** the data root is the per-user application data folder of the operating system:
    - Windows: `%LOCALAPPDATA%\RobloxAccountManager`
    - Linux: `$XDG_DATA_HOME/RobloxAccountManager`, or `~/.local/share/RobloxAccountManager` when `XDG_DATA_HOME` is unset or not absolute
    - macOS: `~/Library/Application Support/RobloxAccountManager`

Windows and Linux offer both modes. macOS offers only Standard storage. When the executable folder is the Standard root, only Standard storage is offered.

Both modes use the same layout, relative to the data root:

```text
storage/
    settings.json
    vault/          # vault.db, vault.key, autounlock.key, recovery files, and vault creation staging
    backups/
    runtime/        # Managed browser runtime
    temp/           # Browser installation staging and temporary browser profiles
logs/
```

Managed subdirectories are created when needed.

### Choosing the data root

There is no marker file. `internal/appdata` detects each candidate root from its existing layout and vault files:

- **Empty:** no `storage/` folder, or one that contains none of `settings.json`, `vault/`, `backups/`, `runtime/`, and `temp/`.
- **Structure:** a `storage/` folder with at least one of those entries, but without vault files. `settings.json`, `autounlock.key`, logs, backups, runtime data, and temporary data are optional and are re-created when needed, so they do not make a vault.
- **Incomplete:** some vault files, but not a usable vault. This is only one of `vault.db` and `vault.key`, or files in the vault creation staging folder. An incomplete vault is kept for recovery and is never replaced by a new vault in another root.
- **Vault:** `vault.db` and `vault.key` without unfinished vault creation.

At startup on Windows and Linux:

1. Neither root has data: the first-run dialog offers Portable or Standard storage.
2. Only one root has a vault: that root is used without a question. A Standard root with only an incomplete vault is also used, so the vault dialog can recover it.
3. The Portable root has data but no usable vault: the dialog offers to continue there, to create or recover a vault, or to start using Standard storage.
4. Only the Standard root has data without a vault: the Standard root is used.
5. Both roots hold vault data, complete or incomplete: the application never chooses. The dialog asks every launch until the user moves or deletes one copy. The other copy stays unchanged.

macOS always uses the Standard root and creates it on first launch without a question.

Until the user confirms a root, the application uses the Portable root provisionally. It holds log entries in memory, keeps default settings in memory when the root is empty, and does not start vault startup, so automatic unlock and migrations never touch a vault that the user has not chosen. Choosing the provisional root continues in the same process. Choosing the other root writes the held log entries to that root, starts a new process with `--storage=<mode>`, and quits. The new process waits for the single-instance lock and uses that root without a question.

Deleting `settings.json` resets settings to their defaults. Deleting `autounlock.key` turns off automatic unlock, and the vault asks for the master password. Logs, backups, runtime data, and temporary data are re-created when needed.

### Moving data

The vault is portable. `vault.db` and `vault.key` work in either mode and on any supported operating system, so moving `storage/` between roots or computers keeps the accounts. Automatic unlock is bound to the device and user. `autounlock.key` is sealed with DPAPI, the Secret Service, or the Keychain, so a copied `autounlock.key` fails, and the vault asks for the master password until automatic unlock is turned on again.

### Rules

- Resolve every data path through `internal/appdata`. Do not depend on the current working directory, and do not assume that the executable folder or the data root has a fixed name or location.
- Do not require machine-wide installation, global environment variables, system services, or persistent registry entries. Reading existing registry values is allowed.
- Outside the data root, write only where the operating system or Roblox requires it for a specific operation. If such an operation creates temporary data, remove the data when the operation finishes or during the next safe cleanup.
- Never write data into the macOS application bundle.
- Keep the live vault on a local or removable drive. The application rejects network data roots. Cloud-synchronized directories are unsupported, because another process can replace vault files outside the application lock.

`task clean` is destructive. It deletes generated files, dependencies, build outputs, and all portable `storage/` and `logs/` data in the repository and in `dist/`, including the account vault. It never deletes a Standard data root. The task asks for confirmation before it deletes anything. It must keep every path that it does not list.
