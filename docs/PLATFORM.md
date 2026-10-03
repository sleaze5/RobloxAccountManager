# Platform and portability

Architecture rules are in [ARCHITECTURE.md](./ARCHITECTURE.md). They also apply to platform code.

## Platform support

Support only Windows AMD64. Do not expose other architectures or operating systems as build targets.

Name each build target `<os>-<arch>` with Go's `GOOS` and `GOARCH` values. Expose it as `task build:<os>-<arch>` and write its output to a matching subdirectory of `dist/`. Development and production builds share the target directory:

```text
dist/
    windows-amd64/
        RobloxAccountManager.exe
```

Add a future target through its own platform tasks and modules. Keep the shared application and frontend task graph.

## Platform isolation

Keep a clear line between OS-specific code and shared code. A reader must be able to tell from the file name or package path whether code is OS-specific.

OS-specific code lives in one of two places:

- **OS-specific files in a feature package.** When a feature package needs native behavior for its own work, put that behavior in `<name>_windows.go` files in the same package. Expose it to the rest of the package through package-private functions with platform-neutral signatures. Examples: `internal/appdata/permissions_windows.go`, `internal/browser/process_windows.go`, `internal/gamelaunch/launch_windows.go`, `internal/logging/crash_windows.go`.
- **Packages under `internal/platform/`.** A native capability that the application uses as a feature of its own goes in `internal/platform/<capability>/`, with a platform-neutral API. Current packages: `protection` (protects data for the current Windows user), `singleinstance`, `robloxmulti` (Roblox multi-instance support and process control), and `robloxlogs` (Roblox Player log directory).

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
