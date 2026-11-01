# Releases

Platform rules are in [PLATFORM.md](./PLATFORM.md). Git rules are in [GIT.md](./GIT.md).

`internal/appmeta/VERSION` is the only source of the application version. It uses `MAJOR.MINOR.PATCH` with an optional `-PRERELEASE` suffix. Go embeds it, the frontend reads it as `APP_VERSION`, and the build writes it into the Windows version resource. Do not write the version anywhere else.

To release, change `VERSION`, run `go tool -modfile build/tools/go.mod task fix`, get the change into `main` as [GIT.md](./GIT.md) describes, and run the manual "Release" workflow in `.github/workflows/release.yml`. The workflow first checks that it runs on `main` and that tag `v<version>` does not exist. Only a confirmed "not found" answer passes the tag check. Any other error stops the release, because `gh release create` attaches a release to an existing tag and ignores `--target`. The check job defines the release targets once. They drive the build matrix and the targets that release preparation expects. The workflow validates its own commit and does not rely on an earlier "Check" run: `.github/workflows/prepare-frontend.yml` checks the frontend and `.github/workflows/validate-go.yml` checks the Go source on every operating system. The frontend is prepared once on Linux and reaches each build as an artifact of the same run. The workflow builds Linux on Ubuntu 24.04, macOS on macOS 15, and Windows on Windows Server 2025 in parallel with Go validation, through the reusable `.github/workflows/build-targets.yml`, and keeps the builds for 7 days. The macOS job builds both architectures on Apple silicon. The Windows job builds x64 and cross-compiles ARM64 with a pinned, checksum-verified llvm-mingw release. A build fails when it changes source files. The Windows job pauses Defender real-time scanning, because it slows builds on the disposable runners. After validation and every build succeed, the publish job on Ubuntu 24.04 runs the `release:prepare` task and creates the release. The workflow publishes:

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

## Release preparation

The `release:prepare` task is the only definition of how release archives become a signed update manifest. It runs `scripts/release-prepare.ts` with these variables:

- `RELEASE_DIR`: the folder with the downloaded builds.
- `TARGETS`: the expected targets, such as `linux-amd64 windows-amd64`.
- `REPOSITORY`: the GitHub repository, for the download URLs.
- `SIGNING`: `production` or `rehearsal`.

The task reads the version from `VERSION` and runs the Wails CLI with `go tool wails3`, which `go.mod` pins to the same Wails version as the application. It builds the CLI without cgo, so it needs no system libraries. It finds every `.zip` and `.tar.gz` archive in `RELEASE_DIR` and generates the manifest. Wails reads each archive's platform and architecture from its name. The task fails unless each expected target has exactly one archive, and no archive belongs to another target or to no target. Then it verifies every signature.

- **Production** signing reads the private key from `UPDATER_PRIVATE_KEY` and verifies against `internal/appmeta/updater.key.pub`, the key that the application embeds.
- **Rehearsal** signing generates a throwaway key pair and verifies against its public key. It proves that preparation works, not that the production secret matches the embedded key.

The publish job uploads every archive, `.exe`, and `manifest.json` from `RELEASE_DIR`, plus `THIRD_PARTY_LICENSES.txt`. Asset names are defined only by the packaging in `build-targets.yml`.

To rehearse a release, run the "Release" workflow with **Rehearse** selected, from any branch. It builds every target and runs the `release:prepare` task with rehearsal signing. It never reads `UPDATER_PRIVATE_KEY`, skips the branch and tag checks, and uploads the prepared files as the `release-rehearsal` artifact instead of creating a release.

A re-run of a workflow run uses that run's commit and workflow files. After a workflow fix, start a new run with **Run workflow**.

## Test builds

`.github/workflows/build-targets.yml` is the only definition of how a target is set up, built, checked, and packaged. The "Release" workflow calls it for each operating system, and the manual "Build" workflow in `.github/workflows/build.yml` calls it for one target. Add a target there once, and both workflows build it the same way. It only compiles: it takes the `artifact` output of `prepare-frontend.yml` from the same run as its `frontend-artifact` input, so a caller must run `prepare-frontend.yml` first.

The frontend bindings and production assets are the same for every target, so they are prepared once on Linux. The Wails services and their models are defined in platform-neutral Go files. Keep them platform-neutral when you add an operating system, and keep native behavior behind them.

To test a build without a release, run the "Build" workflow from the Actions tab and choose the branch and the target. It validates the commit like a release does. The run keeps the packaged files for 7 days, with the same names as release assets. Test builds are not added to the update manifest.

The pinned llvm-mingw release for Windows ARM64 is set up by `.github/actions/setup-llvm-mingw`. Change its version and checksum there.

## Tasks

The `build` task is the complete local workflow: it runs `fix`, then `frontend:build`, then `build:<target>` for `TARGET`, which defaults to the host. CI runs the same steps as separate tasks, so it can prepare the frontend once and compile each target from it:

- `setup:frontend` installs the frontend dependencies from the lockfile.
- `generate` generates the TypeScript bindings for the Wails services.
- `frontend:build` builds the production frontend from the installed dependencies and the generated bindings.
- `build:<target>` compiles and packages one target from the production frontend. It never runs fixes, checks, generation, or a frontend build, and it fails when `frontend/dist/index.html` is missing or empty.

## Checks

The `check` task runs two scopes in parallel, and each runs its own steps in parallel:

- `check:common` installs the frontend dependencies, generates the bindings, and validates what does not depend on the operating system. It runs three tasks that CI also runs on their own:
    - `check:frontend:source`: frontend formatting and lint, and whether `@wailsio/runtime` in `frontend/package.json` matches the Wails version in `go.mod`.
    - `check:frontend:types`: `svelte-check` and the type check of the scripts. It needs the generated bindings.
    - `check:go`: Go formatting, and whether the `go.mod` and `go.sum` files of the application and of `build/tools` are tidy.
- `check:native` runs golangci-lint and `go build`. Go compiles only the files of the current operating system, so this scope must run on each one.

The `fix` task still applies every fix, including golangci-lint fixes, and then runs the `check` task. It first sets `@wailsio/runtime` to the Wails version in `go.mod`, so `go.mod` is the only place to change the Wails version. Dependabot and the `update` task update Wails only in `go.mod`.

golangci-lint runs the `go vet` analyzers, staticcheck, and the other linters that `.golangci.yml` enables. Its version is pinned in the separate tool module `build/tools`, and the tasks run it with `go tool -modfile=build/tools/go.mod`. The separate module keeps the linter's dependencies out of the application's `go.mod`, so they cannot change the dependency versions of the application. Dependabot updates golangci-lint, and the `update` task upgrades it.

Task is pinned in `build/tools` the same way and runs with `go tool -modfile build/tools/go.mod task`, so contributors and runners need Go but not a separate Task installation. Dependabot updates Task, and the `update` task upgrades it.

The "Check" workflow in `.github/workflows/check.yml` runs on every pull request and every push to `main`. A newer run cancels an older one for the same pull request or branch.

- **Scope** compares the changed files, including deleted and renamed ones. A push compares its before and after commits. A pull request compares its merge commit with the base it merges into, so changes on `main` that the branch does not have are not counted. It selects the other jobs:
    - Changes in `frontend/` or `scripts/` run **Frontend** with every frontend check.
    - Changes in `internal/`, to a Go file, or to `go.sum` run **Go**, and **Frontend** without formatting and lint, because the bindings and `APP_VERSION` can change.
    - Changes only to documentation, `AGENTS.md`, the license files, `.claude/`, or `assets/showcase/` run neither.
    - Any other change, including `go.mod`, which `check:frontend:source` compares with `@wailsio/runtime`, a missing base, and a manual run, runs everything.
- **Frontend** runs `.github/workflows/prepare-frontend.yml`, which is described under "Caches".
- **Go** runs `.github/workflows/validate-go.yml`: the `check:go` task once on Linux, and the `check:native` task on each supported operating system at the same time.
- **CI** passes only when every selected job passed and every other job was skipped, so a selected job that was skipped or cancelled fails it. Require only this check in branch protection.

Each tool has one setup action, and a job uses only the ones it needs. `.github/actions/setup-go` sets up Go, which runs the pinned Task, so no job installs Task. `.github/actions/setup-bun` sets up Bun. `.github/actions/setup-native` prepares the operating system for cgo: it installs the GTK and WebKitGTK libraries on Linux and pauses Defender on Windows. The "Update dependencies" workflow runs the complete local workflow, so it uses all three.

## Caches

`.github/actions/setup-go` is the only owner of the Go module and build caches. Each job passes a cache name for the kind of work it does, because `actions/setup-go` saves a cache only once per key, and the first job to save it decides what every later job restores:

- `validation`: Go formatting and module checks.
- `native`: golangci-lint and `go build` on each operating system, and the "Update dependencies" workflow.
- `frontend`: binding generation.
- `production-<targets>`: production builds of one target group, such as `production-windows-amd64 windows-arm64`. A single-target test build has its own cache, so it cannot leave out the second architecture of a release build.
- `none`: no cache, for the publish job, which only builds Task.

The key also contains the operating system, the architecture, the Go version, and both `go.sum` files. Go's build cache already tells apart architectures, build flags, tags, and C compiler versions, so a new runner image or compiler only causes cache misses. A cache never changes after it is saved. It gets the packages that changed since then only when a `go.sum` file or the Go version changes.

`.github/workflows/prepare-frontend.yml` caches the bindings and the production frontend together, with an exact key and no fallback to a similar key. The key hashes every file that binding generation or the frontend build reads: everything in `frontend/` and `internal/`, `main.go`, `go.mod`, `go.sum`, the two Taskfiles that define the commands, and the workflow itself. `frontend/package.json` pins the Bun version and `go.mod` pins Wails, so both are in the key. A hit skips generation and the frontend build, but the frontend checks still run. A second exact cache holds only the bindings, keyed without `frontend/`. When only frontend files changed, it skips the native libraries and binding generation. Caches are saved only after every check passes. Add a file to both keys when generation or the build starts to read it.

## Update signing

- The `UPDATER_PRIVATE_KEY` repository secret signs `manifest.json`. Never commit the private key.
- `internal/appmeta/updater.key.pub` is the public key that every build embeds. The workflow verifies each manifest with it before it publishes.
- `internal/appupdate` rejects updates without a signature. Keep this check, because the Wails updater installs unsigned manifest entries.
- Replace `updater.key.pub` only to rotate a lost or exposed key. Installed copies with the old key cannot verify later updates.

## In-place updates

- `appupdate.HandleHelperMode()` must run first in `main()`, before the single-instance check. The update helper is this executable, and it starts while the old version still runs.
- The updater replaces only the executable, or on macOS only the application bundle. The data root stays unchanged in both storage modes, because it is never inside what the updater replaces.
- On Linux, the helper copies the downloaded update into a `wails-update-robloxaccountmanager-*` directory next to the executable before the swap. The Wails helper renames the update over the executable, and a rename fails across filesystems. The application removes the original download when it quits for the update.
- On macOS, the helper copies the extracted bundle into a `wails-update-robloxaccountmanager-*` directory next to the installed bundle for the same reason.
- `internal/appupdate` removes `RobloxAccountManager.exe.old.*`, `wails-update-robloxaccountmanager-*` directories next to the executable or bundle, and the helper's `wails-update-*.log` when the new version starts. It deletes a downloaded update that was not installed when the application closes.
