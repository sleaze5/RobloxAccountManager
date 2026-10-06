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

A Linux build links the system GTK 4 and WebKitGTK 6 libraries. Release builds come from Ubuntu 24.04 and need glibc 2.38 or later. Automatic unlock stores its encryption key in the Secret Service. The vault, settings, and logs stay in the application directory. Joining a game opens a `roblox-player:` link in Sober or Mocktail through that client's desktop entry. The Roblox setting chooses the client when both are installed. Automatic uses the only installed client, or the desktop default for `roblox-player` when both are installed. The managed browser runs Chrome for Testing from `storage/runtime/`. It needs the system libraries that Chrome requires and unprivileged user namespaces for the Chrome sandbox. Controlling Roblox processes is unavailable on Linux.

## Releases

`internal/appmeta/VERSION` is the only source of the application version. It uses `MAJOR.MINOR.PATCH` with an optional `-PRERELEASE` suffix. Go embeds it, the frontend reads it as `APP_VERSION`, and the build writes it into the Windows version resource. Do not write the version anywhere else.

To release, change `VERSION`, run `task fix`, push to `main`, and run the manual "Release" workflow in `.github/workflows/release.yml`. The workflow first checks that it runs on `main` and that tag `v<version>` does not exist. Only a confirmed "not found" answer passes the tag check. Any other error stops the release, because `gh release create` attaches a release to an existing tag and ignores `--target`. The check job defines the release targets once. They drive the build matrix and the targets that release preparation expects. The workflow builds Linux on Ubuntu 24.04 and Windows on Windows Server 2025 in parallel, through the reusable `.github/workflows/build-targets.yml`, and keeps the builds for 7 days. A build fails when it changes source files. The Windows build pauses Defender real-time scanning, because it slows builds on the disposable runners. After every build succeeds, the publish job on Ubuntu 24.04 runs `task release:prepare` and creates the release. The workflow publishes:

- `RobloxAccountManager.exe`: the Windows executable for manual download.
- `RobloxAccountManager-windows-x64.zip`: the Windows updater artifact.
- `RobloxAccountManager-linux-x64.tar.gz`: the Linux updater artifact and manual download. The archive keeps the executable permission that a bare download loses.
- `manifest.json`: the signed Wails update manifest. List only archives in it, because Wails treats every `.exe` as a Windows artifact.
- `THIRD_PARTY_LICENSES.txt`: the license texts of the bundled third-party works. It ships as its own asset, because each archive must contain only the executable.

Each archive must contain only the executable, because the updater accepts exactly one top-level entry. Wails reads the platform and architecture from each archive name, so keep the `<os>-x64` part.

The application reads `releases/latest/download/manifest.json`, so it never offers a prerelease.

### Release preparation

`task release:prepare` is the only definition of how release archives become a signed update manifest. It runs `scripts/release-prepare.ts` with these variables:

- `RELEASE_DIR`: the folder with the downloaded builds.
- `TARGETS`: the expected targets, such as `linux-amd64 windows-amd64`.
- `REPOSITORY`: the GitHub repository, for the download URLs.
- `SIGNING`: `production` or `rehearsal`.

The task reads the version from `VERSION` and the Wails CLI from `WAILS_CLI`. It builds the CLI without cgo, so it needs no system libraries. It finds every `.zip` and `.tar.gz` archive in `RELEASE_DIR` and generates the manifest. Wails reads each archive's platform and architecture from its name. The task fails unless each expected target has exactly one archive, and no archive belongs to another target or to no target. Then it verifies every signature.

- **Production** signing reads the private key from `UPDATER_PRIVATE_KEY` and verifies against `internal/appmeta/updater.key.pub`, the key that the application embeds.
- **Rehearsal** signing generates a throwaway key pair and verifies against its public key. It proves that preparation works, not that the production secret matches the embedded key.

The publish job uploads every archive, `.exe`, and `manifest.json` from `RELEASE_DIR`, plus `THIRD_PARTY_LICENSES.txt`. Asset names are defined only by the packaging in `build-targets.yml`.

To rehearse a release, run the "Release" workflow with **Rehearse** selected, from any branch. It builds every target and runs `task release:prepare` with rehearsal signing. It never reads `UPDATER_PRIVATE_KEY`, skips the branch and tag checks, and uploads the prepared files as the `release-rehearsal` artifact instead of creating a release.

A re-run of a workflow run uses that run's commit and workflow files. After a workflow fix, start a new run with **Run workflow**.

### Test builds

`.github/workflows/build-targets.yml` is the only definition of how a target is set up, built, checked, and packaged. The "Release" workflow calls it for each operating system, and the manual "Build" workflow in `.github/workflows/build.yml` calls it for one target. Add a target there once, and both workflows build it the same way.

To test a build without a release, run the "Build" workflow from the Actions tab and choose the branch and the target. The run keeps the packaged files for 7 days, with the same names as release assets. Test builds are not added to the update manifest.

### Checks

`task check` runs two scopes in parallel, and each runs its own steps in parallel:

- `task check:common` validates what does not depend on the operating system: frontend formatting, lint, and `svelte-check`, Go formatting, and whether `go.mod` and `go.sum` are tidy.
- `task check:native` runs `go vet`, staticcheck, and `go build`. Go compiles only the files of the current operating system, so this scope must run on each one.

`task fix` still applies every fix and then runs `task check`.

The "Check" workflow in `.github/workflows/check.yml` runs on every pull request and every push to `main`. A newer run cancels an older one for the same pull request or branch.

- **Scope** finds the changed files. The native scope runs only when Go files, `go.mod`, `go.sum`, a Taskfile, or `.github/` changed, or when there is no base to compare with.
- **Common** runs `task check:common` once on Linux.
- **Native** runs `task check:native` on each supported operating system at the same time, when the native scope runs.
- **CI** passes only when every other job passed or was skipped. Require only this check in branch protection.

`.github/actions/setup` is the only definition of how a runner is prepared for task commands. The check, build, and release workflows all use it.

### Update signing

- The `UPDATER_PRIVATE_KEY` repository secret signs `manifest.json`. Never commit the private key.
- `internal/appmeta/updater.key.pub` is the public key that every build embeds. The workflow verifies each manifest with it before it publishes.
- `internal/appupdate` rejects updates without a signature. Keep this check, because the Wails updater installs unsigned manifest entries.
- Replace `updater.key.pub` only to rotate a lost or exposed key. Installed copies with the old key cannot verify later updates.

### In-place updates

- `appupdate.HandleHelperMode()` must run first in `main()`, before the single-instance check. The update helper is this executable, and it starts while the old version still runs.
- The updater replaces only the executable. `storage/` and `logs/` stay unchanged.
- On Linux, the helper copies the downloaded update into a `wails-update-robloxaccountmanager-*` directory next to the executable before the swap. The Wails helper renames the update over the executable, and a rename fails across filesystems. The application removes the original download when it quits for the update.
- `internal/appupdate` removes `RobloxAccountManager.exe.old.*`, `wails-update-robloxaccountmanager-*` directories next to the executable, and the helper's `wails-update-*.log` when the new version starts. It deletes a downloaded update that was not installed when the application closes.

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
