<div align="center">

<img src="assets/icon.svg" alt="Roblox Account Manager icon" width="96" height="96">

# Roblox Account Manager

A portable desktop application to store, manage, and launch Roblox accounts from an encrypted, password-protected vault.

[![Latest release](https://img.shields.io/github/v/release/sleaze5/RobloxAccountManager?label=release)](https://github.com/sleaze5/RobloxAccountManager/releases/latest) [![Checks](https://img.shields.io/github/actions/workflow/status/sleaze5/RobloxAccountManager/check.yml?branch=main&label=checks)](https://github.com/sleaze5/RobloxAccountManager/actions/workflows/check.yml) [![License](https://img.shields.io/github/license/sleaze5/RobloxAccountManager)](LICENSE)

[![Download for Windows x64](https://img.shields.io/badge/Windows-x64-0969da?style=for-the-badge&labelColor=24292f&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI%2BPHBhdGggZmlsbD0iI2ZmZiIgZD0iTTAgMGgxMS40djExLjRIMHpNMTIuNiAwSDI0djExLjRIMTIuNnpNMCAxMi42aDExLjRWMjRIMHpNMTIuNiAxMi42SDI0VjI0SDEyLjZ6Ii8%2BPC9zdmc%2B)](https://github.com/sleaze5/RobloxAccountManager/releases/latest/download/RobloxAccountManager.exe) [![Download for Windows ARM64](https://img.shields.io/badge/Windows-ARM64-0969da?style=for-the-badge&labelColor=24292f&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI%2BPHBhdGggZmlsbD0iI2ZmZiIgZD0iTTAgMGgxMS40djExLjRIMHpNMTIuNiAwSDI0djExLjRIMTIuNnpNMCAxMi42aDExLjRWMjRIMHpNMTIuNiAxMi42SDI0VjI0SDEyLjZ6Ii8%2BPC9zdmc%2B)](https://github.com/sleaze5/RobloxAccountManager/releases/latest/download/RobloxAccountManager-arm64.exe) [![Download for Linux x64](https://img.shields.io/badge/Linux-x64-0969da?style=for-the-badge&labelColor=24292f&logo=linux&logoColor=white)](https://github.com/sleaze5/RobloxAccountManager/releases/latest/download/RobloxAccountManager-linux-x64.tar.gz) [![Download for macOS on Apple silicon](https://img.shields.io/badge/macOS-Apple%20silicon-0969da?style=for-the-badge&labelColor=24292f&logo=apple&logoColor=white)](https://github.com/sleaze5/RobloxAccountManager/releases/latest/download/RobloxAccountManager-macos-arm64.zip) [![Download for macOS on Intel](https://img.shields.io/badge/macOS-Intel-0969da?style=for-the-badge&labelColor=24292f&logo=apple&logoColor=white)](https://github.com/sleaze5/RobloxAccountManager/releases/latest/download/RobloxAccountManager-macos-x64.zip)

[All releases](https://github.com/sleaze5/RobloxAccountManager/releases) | [Installation](#installation) | [Platform support](#platform-support) | [Building from source](#building-from-source)

</div>

This app is built primarily for Windows but also works on Linux and macOS. Some features may be unavailable on those platforms.

## Features

- **Encrypted vault:** SQLCipher encryption, protected by your master password through Argon2id.
    - Unlock automatically on your device.
    - Restore from automatic backups.
- **Account import:** Paste multiple `.ROBLOSECURITY` cookies, or sign in through a managed browser.
- **Account management:** Search, tag, favorite, and reorder accounts.
    - View profiles, Robux balances, online status, and account details.
    - Change Roblox privacy and account settings.
    - Read and send Roblox chats.
    - Renew cookies and open signed-in browsers.
- **Game launching:** Join by Place ID, Job ID, user, or Roblox link with one or more accounts.
    - Join private servers, share links, and experience invites.
    - Run multiple Roblox clients at once on Windows.
- **Games and servers:** Search games, save favorite places, and browse servers by ping and player count.
    - Find the nearest joinable server with [RoValra](https://www.rovalra.com).
- **Logs Explorer:** Browse Roblox logs by session and game visit, and rejoin past servers.
- **Portable storage:** Run without an installer, with the vault next to the executable or in your user data folder.

RoValra features are on by default. Turn them off in Settings > Integrations.

## Showcase

Select a screenshot to view it at full size.

<table>
<tr>
<td width="50%"><a href="assets/showcase/01-accounts.png"><img src="assets/showcase/01-accounts.png" alt="Account profile with presence, Robux balance, tags, and launch options" width="100%"></a><br><sub><b>Accounts:</b> profiles, presence, and launch options</sub></td>
<td width="50%"><a href="assets/showcase/02-multi-launch.png"><img src="assets/showcase/02-multi-launch.png" alt="Several selected accounts with additional launch options open" width="100%"></a><br><sub><b>Multi-account launch:</b> join with several accounts at once</sub></td>
</tr>
<tr>
<td width="50%"><a href="assets/showcase/03-processes.png"><img src="assets/showcase/03-processes.png" alt="Dialog that lists running Roblox processes to close" width="100%"></a><br><sub><b>Process control:</b> close running Roblox clients</sub></td>
<td width="50%"><a href="assets/showcase/04-add-account.png"><img src="assets/showcase/04-add-account.png" alt="Paste cookies dialog with a validated preview of three accounts" width="100%"></a><br><sub><b>Account import:</b> validate cookies before saving</sub></td>
</tr>
<tr>
<td width="50%"><a href="assets/showcase/05-account-settings.png"><img src="assets/showcase/05-account-settings.png" alt="Account communication and privacy settings" width="100%"></a><br><sub><b>Account settings:</b> Roblox privacy and communication</sub></td>
<td width="50%"><a href="assets/showcase/06-chats.png"><img src="assets/showcase/06-chats.png" alt="Chat list and an open conversation" width="100%"></a><br><sub><b>Chats:</b> read and send Roblox messages</sub></td>
</tr>
<tr>
<td width="50%"><a href="assets/showcase/07-game-details.png"><img src="assets/showcase/07-game-details.png" alt="Favorite games list and game details" width="100%"></a><br><sub><b>Games:</b> favorites and game details</sub></td>
<td width="50%"><a href="assets/showcase/08-servers.png"><img src="assets/showcase/08-servers.png" alt="Server browser with player counts, ping, region, and uptime" width="100%"></a><br><sub><b>Servers:</b> browse by ping, region, and player count</sub></td>
</tr>
<tr>
<td width="50%"><a href="assets/showcase/09-logs-explorer.png"><img src="assets/showcase/09-logs-explorer.png" alt="Logs Explorer with a session timeline of game visits" width="100%"></a><br><sub><b>Logs Explorer:</b> sessions, visits, and rejoins</sub></td>
<td width="50%"><a href="assets/showcase/10-unlock.png"><img src="assets/showcase/10-unlock.png" alt="Unlock vault dialog asking for the master password" width="100%"></a><br><sub><b>Encrypted vault:</b> unlock with your master password</sub></td>
</tr>
</table>

The screenshots use sample data. The accounts, chats, and logs shown are fictional and do not represent real Roblox users.

## Installation

Download for your system using the buttons above or the [latest release](https://github.com/sleaze5/RobloxAccountManager/releases/latest).

### Windows

1. Download `RobloxAccountManager.exe` for x64 or `RobloxAccountManager-arm64.exe` for ARM64.
2. Put the executable in its own folder. Portable storage creates `storage/` and `logs/` next to it.
3. Run the executable.
4. Choose portable or standard storage.
5. Create a master password.

### Linux

1. Download `RobloxAccountManager-linux-x64.tar.gz`.
2. Extract the archive into its own folder:

    ```sh
    tar -xzf RobloxAccountManager-linux-x64.tar.gz
    ```

3. Install Sober or Mocktail to join games.
4. Run `./RobloxAccountManager`.
5. Choose portable or standard storage.
6. Create a master password.

The application needs the GTK 4 and WebKitGTK 6 libraries. Automatic unlock needs a Secret Service provider, such as GNOME Keyring or KWallet. The managed browser needs the system libraries of Chrome and unprivileged user namespaces for the Chrome sandbox.

### macOS

1. Download `RobloxAccountManager-macos-arm64.zip` for Apple silicon or `RobloxAccountManager-macos-x64.zip` for Intel.
2. Extract the archive.
3. Open `RobloxAccountManager.app`. Gatekeeper blocks the first launch, because release bundles are signed ad hoc and are not notarized. Allow the application in System Settings > Privacy & Security, or open it from the Finder context menu.
4. Create a master password.

To join games, install Roblox. The application opens `Roblox.app` from `/Applications` or `~/Applications`.

> [!IMPORTANT]\
> Before each launch on macOS, the application deletes the cookie file of the Roblox Player. Otherwise, Roblox joins with the account that last signed in to the Roblox app. As a result, every launch signs the Roblox app out.

## Platform support

Windows is the primary platform. The minimum supported versions are:

- Windows 10 or later: x64 and ARM64
- Linux x64 with glibc 2.38 or later
- macOS 12 or later: Apple silicon and Intel

Feature differences:

- **Windows:** Every feature is available, including multi-instance, Roblox process control, and direct launch of `RobloxPlayerBeta.exe`. Automatic unlock uses DPAPI. On ARM64, the managed browser runs x64 Chrome through Windows 11 emulation, because Chrome for Testing has no ARM64 build for Windows.
- **Linux:** Joins games through Sober or Mocktail. Settings > Roblox chooses the client. Its Automatic option uses the installed client, or the desktop default when both are installed. Automatic unlock uses the Secret Service. Multi-instance and Roblox process control are unavailable.
- **macOS:** Only standard storage is available, because a portable vault would sit inside the app bundle. Automatic unlock uses the login Keychain. Roblox process control is available. Multi-instance is unavailable.

## Data and privacy

All application data stays in one folder:

```text
storage/
    settings.json
    vault/       # Encrypted vault, key file, and automatic unlock key
    backups/     # Vault backups
    runtime/     # Managed browser
    temp/        # Temporary browser data
logs/
```

- Portable storage uses the executable's folder. Standard storage uses your OS user's application data folder. Settings > About shows the storage mode and data folder.
- Keep the data folder on a local or removable drive. The application rejects network folders. Cloud-synchronized folders are unsupported, because another process can replace the vault files.
- `vault.db` and `vault.key` work on every supported operating system. `autounlock.key` works only on the device and operating system user that created it. After a move to another device, unlock the vault with the master password and turn on automatic unlock again.
- The application connects to Roblox, to the RoValra API when the integration is on, to GitHub to check for updates, and to Google storage to download the managed browser. Requests to the RoValra API include only Place IDs and server IDs.
- Application logs never contain cookies, passwords, or other credentials.

## Building from source

### Requirements

- [Go](https://go.dev) 1.26 or later
- [Bun](https://bun.sh)
- A C compiler for cgo, because SQLCipher needs cgo:
    - **Windows x64:** MinGW-w64 GCC on `PATH`.
    - **Windows ARM64:** MinGW-w64 GCC and `aarch64-w64-mingw32-clang` from [llvm-mingw](https://github.com/mstorsjo/llvm-mingw) on `PATH`. Put llvm-mingw after the x64 MinGW on `PATH`, because llvm-mingw also ships a `gcc`.
    - **Linux:** A C compiler and the GTK 4 and WebKitGTK 6 development libraries. On Ubuntu 24.04, install `libgtk-4-dev` and `libwebkitgtk-6.0-dev`.
    - **macOS:** Xcode command line tools. On macOS, the Wails runtime also uses cgo.

### Build

Clone the repository:

```sh
git clone https://github.com/sleaze5/RobloxAccountManager.git
cd RobloxAccountManager
```

Run every command from the repository root. Go runs the [Task](https://taskfile.dev) version pinned in `build/tools/go.mod`, so you do not install Task yourself.

Run the build task for your target:

- `go tool -modfile build/tools/go.mod task build:windows-amd64`: Windows x64. Runs on Windows.
- `go tool -modfile build/tools/go.mod task build:windows-arm64`: Windows ARM64. Cross-compiles on x64 Windows with llvm-mingw.
- `go tool -modfile build/tools/go.mod task build:linux-amd64`: Linux x64. Runs on Linux.
- `go tool -modfile build/tools/go.mod task build:darwin-arm64`: macOS app bundle for Apple silicon. Runs on macOS.
- `go tool -modfile build/tools/go.mod task build:darwin-amd64`: macOS app bundle for Intel. Runs on macOS.

Each build first runs `go tool -modfile build/tools/go.mod task fix` to format, apply lint fixes, and validate the source. Output goes to `dist/<os>-<arch>/`.

Other tasks:

- `go tool -modfile build/tools/go.mod task dev`: Start development mode with hot reload.
- `go tool -modfile build/tools/go.mod task check`: Validate the source without changing files.
- `go tool -modfile build/tools/go.mod task --list --sort none`: List every task.

> [!WARNING]\
> `go tool -modfile build/tools/go.mod task clean` deletes builds, dependencies, generated files, logs, and all portable data, including any account vault in `dist/`. This cannot be undone. The task asks for confirmation, and `-y` skips it.

## License

Roblox Account Manager is released under the [MIT License](LICENSE).

Copyright (c) 2026 sleaze

The application includes third-party works that keep their own licenses. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and [THIRD_PARTY_LICENSES.txt](THIRD_PARTY_LICENSES.txt).

## Disclaimer

Roblox Account Manager is an unofficial, independent project. It is not affiliated with, endorsed by, or sponsored by Roblox Corporation. Roblox is a trademark of Roblox Corporation.

- You are responsible for your use of this software and for following the [Roblox Terms of Use](https://en.help.roblox.com/hc/en-us/articles/115004647846-Roblox-Terms-of-Use).
- A `.ROBLOSECURITY` cookie gives full access to its account. Never share cookies, vault files, or copied launch commands.
- The software is provided "as is", without warranty of any kind. See the [MIT License](LICENSE).
