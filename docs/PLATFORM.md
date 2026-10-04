# Platform and portability

Architecture rules are in [ARCHITECTURE.md](./ARCHITECTURE.md). They also apply to platform code.

## Platform support

Support Windows AMD64 and Linux AMD64. Do not expose other architectures.

Name each build target `<os>-<arch>` with Go's `GOOS` and `GOARCH` values. Expose it as `task build:<os>-<arch>` and write its output to a matching subdirectory of `dist/`. Development and production builds share the target directory:

```text
dist/
    windows-amd64/
        RobloxAccountManager.exe
    linux-amd64/
        RobloxAccountManager
```

Add another target through its own platform tasks and modules. Keep the shared application and frontend task graph.

A Linux build links the system GTK 4 and WebKitGTK 6 libraries. Automatic unlock stores its encryption key in the Secret Service. The vault, settings, and logs stay in the application directory. Joining a game opens a `roblox-player:` link in Sober or Mocktail through that client's desktop entry. The Roblox setting chooses the client when both are installed. Automatic uses the only installed client, or the desktop default for `roblox-player` when both are installed. The managed browser runs Chrome for Testing from `storage/runtime/`. It needs the system libraries that Chrome requires and unprivileged user namespaces for the Chrome sandbox. Controlling Roblox processes is unavailable on Linux.

## Releases

`internal/appmeta/VERSION` is the only source of the application version. It uses `MAJOR.MINOR.PATCH` with an optional `-PRERELEASE` suffix. Go embeds it, the frontend reads it as `APP_VERSION`, and the build writes it into the Windows version resource. Do not write the version anywhere else.

To release, change `VERSION`, run `task fix`, push to `main`, and run the manual "Release" workflow in `.github/workflows/release.yml`. The workflow builds Linux on Ubuntu 24.04 and Windows on Windows Server 2025. It fails when tag `v<version>` exists or when either build changes source files. It publishes:

- `RobloxAccountManager.exe`: the Windows executable for manual download.
- `RobloxAccountManager-windows-x64.zip`: the Windows updater artifact.
- `RobloxAccountManager-linux-x64.tar.gz`: the Linux updater artifact and manual download. The archive keeps the executable permission that a bare download loses.
- `manifest.json`: the signed Wails update manifest. List only archives in it, because Wails treats every `.exe` as a Windows artifact.

Each archive must contain only the executable, because the updater accepts exactly one top-level entry. Wails reads the platform and architecture from each archive name, so keep the `<os>-x64` part.

The application reads `releases/latest/download/manifest.json`, so it never offers a prerelease.

### Update signing

- The `UPDATER_PRIVATE_KEY` repository secret signs `manifest.json`. Never commit the private key.
- `internal/appmeta/updater.key.pub` is the public key that every build embeds. The workflow verifies each manifest with it before it publishes.
- `internal/appupdate` rejects updates without a signature. Keep this check, because the Wails updater installs unsigned manifest entries.
- Replace `updater.key.pub` only to rotate a lost or exposed key. Installed copies with the old key cannot verify later updates.

### In-place updates

- `appupdate.HandleHelperMode()` must run first in `main()`, before the single-instance check. The update helper is this executable, and it starts while the old version still runs.
- The updater replaces only the executable. `storage/` and `logs/` stay unchanged.
- On Linux, the helper copies the downloaded update into a `wails-update-*` directory next to the executable before the swap. The Wails helper renames the update over the executable, and a rename fails across filesystems.
- `internal/appupdate` removes `RobloxAccountManager.exe.old.*`, `wails-update-*` directories next to the executable, and the helper's `wails-update-*.log` when the new version starts. It deletes a downloaded update that was not installed when the application closes.

## Platform isolation

Keep a clear line between OS-specific code and shared code. A reader must be able to tell from the file name or package path whether code is OS-specific.

OS-specific code lives in one of two places:

- **OS-specific files in a feature package.** When a feature package needs native behavior for its own work, put that behavior in `<name>_<os>.go` files in the same package. Expose it to the rest of the package through package-private functions with platform-neutral signatures. Examples: `internal/appdata/permissions_windows.go`, `internal/appdata/permissions_linux.go`, `internal/browser/process_windows.go`, `internal/browser/process_linux.go`, `internal/gamelaunch/launch_windows.go`, `internal/gamelaunch/launch_linux.go`, `internal/logging/crash_windows.go`, `internal/logging/crash_linux.go`.
- **Packages under `internal/platform/`.** A native capability that the application uses as a feature of its own goes in `internal/platform/<capability>/`, with a platform-neutral API. Current packages: `protection` (protects data for the current user), `singleinstance`, `robloxmulti` (Roblox multi-instance support and process control), and `robloxlogs` (Roblox Player log directories).

Rules:

- Keep OS APIs out of shared files. `golang.org/x/sys/windows`, registry access, Win32 calls, and Windows path assumptions belong only in `_windows.go` files.
- To support another OS, add `_<os>.go` files that implement the same package-private functions and platform package APIs. Do not change shared code to do it.
- Platform code must not bypass the application service, storage, Roblox integration, or Wails binding boundaries.

## Portable distribution

The application runs from its own directory without an installer.

- Store binaries, assets, configuration, logs, and the vault in the application directory.
- Do not require machine-wide installation, global environment variables, system services, or persistent registry entries. Reading existing registry values is allowed.
- Do not write to user-profile or system directories unless Windows or Roblox requires it for a specific operation.
- If an operation must create temporary data outside the application directory, remove the data when the operation finishes or during the next safe cleanup.
- Resolve paths from the executable directory through `internal/appdata`. Do not depend on the current working directory.
- Do not assume that the application directory has a fixed name or location.
- Keep the live vault on a local or removable drive. The application rejects network paths. Cloud-synchronized directories are unsupported, because another process can replace vault files outside the application lock.

The runtime layout, relative to the executable directory:

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

`task clean` is destructive. It deletes generated files, dependencies, build outputs, and all portable `storage/` and `logs/` data, including the account vault. The task asks for confirmation before it deletes anything. It must keep every path that it does not list.
