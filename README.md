# DankCalendar

Local, Google, Microsoft, CalDAV, and iCloud calendars in one standalone calendar app.

## Requirements

- Go 1.25+ (build only)
- [Quickshell](https://quickshell.org) (`qs`)
- Qt 6 declarative (including `Qt.labs.platform` for the tray icon)

## Install

```bash
sudo make install
```

This installs:

- `dcal` binary to `/usr/local/bin`
- Quickshell config to `/usr/local/share/quickshell/dankcal`
- Desktop entry + icon

Optional session service (starts the daemon at login, restarts on failure):

```bash
make install-systemd
systemctl --user enable --now dcal
# You can also configure auto-start from Dank Calendar settings
```

Override the prefix with `PREFIX=/usr sudo make install`.

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
Reopen from the tray icon, the desktop entry, or `dcal show`.
Quit entirely from the tray menu or `dcal kill`.

"Start at login" can be toggled in Settings → General (manages an XDG
autostart entry that launches `dcal run -d --hidden`).

The IPC surface is also scriptable: `dcal ipc ui.toggle`,
`dcal ipc events.list from=... to=...`, etc.

## Development

```bash
make run    # build and run against the in-repo quickshell config
make test
```

Install [prek](https://prek.j178.dev/) and activate the hooks:

```bash
prek install
```
