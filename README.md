# DankCalendar

<div align="center">
  <a href="https://danklinux.com">
    <img src="assets/dankcalendar.svg" alt="DankCalendar" width="180">
  </a>

### Local, Google, Microsoft, CalDAV, and iCloud calendars in one standalone app

Built with [Quickshell](https://quickshell.org/) and [Go](https://go.dev/)

[![GitHub stars](https://img.shields.io/github/stars/AvengeMedia/dankcalendar?style=for-the-badge&labelColor=101418&color=ffd700)](https://github.com/AvengeMedia/dankcalendar/stargazers)
[![GitHub License](https://img.shields.io/github/license/AvengeMedia/dankcalendar?style=for-the-badge&labelColor=101418&color=b9c8da)](https://github.com/AvengeMedia/dankcalendar/blob/master/LICENSE)
[![AUR version (bin)](<https://img.shields.io/aur/version/dankcalendar-bin?style=for-the-badge&labelColor=101418&color=9ccbfb&label=AUR%20(bin)>)](https://aur.archlinux.org/packages/dankcalendar-bin)
[![AUR version (git)](<https://img.shields.io/aur/version/dankcalendar-git?style=for-the-badge&labelColor=101418&color=9ccbfb&label=AUR%20(git)>)](https://aur.archlinux.org/packages/dankcalendar-git)
[![Discord](https://img.shields.io/badge/discord-join?style=for-the-badge&logo=discord&logoColor=ffffff&label=discord&labelColor=101418&color=5865f2)](https://discord.gg/ppWTpKmPgT)
[![Ko-Fi donate](https://img.shields.io/badge/donate-kofi?style=for-the-badge&logo=ko-fi&logoColor=ffffff&label=ko-fi&labelColor=101418&color=f16061&link=https%3A%2F%2Fko-fi.com%2Fdanklinux)](https://ko-fi.com/danklinux)

</div>

DankCalendar is a standalone calendar app that brings your Local, [Google](https://calendar.google.com/), [Microsoft](https://outlook.com/), [CalDAV](https://en.wikipedia.org/wiki/CalDAV), and [iCloud](https://www.icloud.com/calendar/) calendars together in one place. It runs as a lightweight daemon with a tray icon, keeps your accounts in sync, and reminds you about events — all with the look and feel of [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell).

## Repository Structure

This is a monorepo containing both the calendar interface and the core backend services:

```
dankcalendar/
├── quickshell/         # QML-based calendar interface
│   ├── Modules/        # UI components (calendar views, sidebar, tray)
│   ├── Modals/         # Settings, account, event, and search dialogs
│   ├── Services/       # IPC bridge and shell-side state
│   ├── Widgets/        # Reusable Dank UI controls
│   ├── Common/         # Shared resources, themes, and i18n
│   └── translations/   # POEditor-managed string catalogs
├── core/               # Go backend, daemon, and CLI
│   ├── cmd/dcal/       # dcal CLI and daemon entrypoint
│   ├── internal/       # Providers, sync engine, reminders, IPC, OAuth
│   ├── api/            # HUMA + chi HTTP API
│   ├── ent/            # Ent ORM schema and generated code
│   └── repo/           # Repository-pattern data access
├── assets/             # Icons, desktop entry, systemd unit
└── Makefile            # Build, install, and dev targets
```

## See it in Action

<div align="center">

<!-- Screenshots / demo video coming soon -->

</div>

## Installation

### Arch Linux (AUR)

```bash
# Prebuilt binary
yay -S dankcalendar-bin

# Build from latest git
yay -S dankcalendar-git
```

### From Source

```bash
sudo make install
```

This installs:

- `dcal` binary to `/usr/local/bin`
- Quickshell config to `/usr/local/share/quickshell/dankcal`
- Desktop entry + icon

Override the prefix with `PREFIX=/usr sudo make install`.

Optional session service (starts the daemon at login, restarts on failure):

```bash
make install-systemd
systemctl --user enable --now dcal
```

You can also toggle **Start at login** from Settings → General, which manages
an XDG autostart entry that launches `dcal run -d --hidden`.

### Requirements

- [Quickshell](https://quickshell.org) (`qs`)
- Qt 6 declarative (including `Qt.labs.platform` for the tray icon)
- Go 1.25+ (build only — the binary is pure Go, no CGO)

## Features

**Multiple Providers**
Connect Local, Google, Microsoft, CalDAV, and iCloud accounts and view every
calendar in a single unified agenda.

**Background Sync**
A lightweight daemon keeps your accounts in sync and serves the UI over IPC.
Closing the window only hides it — sync and reminders keep running.

**Event Reminders**
Native desktop notifications for upcoming events, with per-event and
per-calendar reminder configuration.

**Tray Integration**
A system tray icon to show, hide, and quit the app, plus quick access to
upcoming events.

**Secure Credentials**
OAuth tokens and account secrets are stored in your system keyring, never in
plaintext.

**Keyboard Navigation**
Full keyboard-driven navigation with a shortcuts overlay, plus spotlight-style
search across your events.

**Localized**
User-facing strings are translatable and managed through POEditor.

## Usage

```bash
dcal           # launch, or show/focus the window if already running
dcal show      # same as above
dcal toggle    # toggle window visibility
dcal run       # run in the foreground (logs to the terminal)
dcal run -d    # run as a background daemon
dcal restart   # restart the running shell
dcal kill      # stop everything
```

Closing the window only hides it — the daemon and tray icon keep running.
Reopen from the tray icon, the desktop entry, or `dcal show`. Quit entirely
from the tray menu or `dcal kill`.

Manage accounts and sync from the command line:

```bash
dcal account list                 # list connected accounts
dcal account add local ~/cal      # add a local calendar directory
dcal account add caldav           # add a CalDAV / iCloud account
dcal account setup google         # OAuth setup for Google
dcal account setup microsoft      # OAuth setup for Microsoft
dcal account remove <account-id>  # remove an account
dcal sync [account-id]            # force a sync
dcal reminders                    # list upcoming reminders
```

## IPC

The daemon exposes a scriptable IPC surface — handy for keybinds and automation:

```bash
dcal ipc ui.toggle
dcal ipc ui.show
dcal ipc events.list from=2026-06-01 to=2026-06-30
dcal ipc accounts.list
dcal ipc reminders.upcoming
dcal ipc system.autostart
```

## Development

```bash
make run    # build and run against the in-repo quickshell config
make test   # run the Go test suite
make fmt    # format Go code
```

Install [prek](https://prek.j178.dev/) and activate the hooks:

```bash
prek install
```

The hooks run `gofmt -s`, `go vet`, `go mod tidy`, and the Go test suite for
changes under `core/`, plus generic whitespace/YAML checks.

See [CONTRIBUTING.md](CONTRIBUTING.md) for editor setup, the i18n workflow, and
Go style guidelines.

## Contributing

Contributions welcome. Bug fixes, widgets, features, providers, and translations
all help.

1. Fork the repository
2. Make your changes
3. Test thoroughly (`make test`)
4. Open a pull request

Include screenshots or video in your PR when the change is user-facing.

## Credits

- [Quickshell](https://quickshell.org/) - Shell framework
- [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell) - Design language and widgets
- [Ent](https://entgo.io/) - Go ORM
- [HUMA](https://huma.rocks/) - HTTP API framework

## License

MIT License - See [LICENSE](LICENSE) for details.
