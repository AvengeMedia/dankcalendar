# DankCalendar Flatpak

The build provides `dcal` and the Quickshell runtime.
It installs the desktop file, metainfo, and icon as `com.danklinux.dankcalendar`.

## Build and install

From the repository root:

```bash
flatpak run --user org.flatpak.Builder \
  --force-clean --user --install --repo=repo \
  --mirror-screenshots-url=https://dl.flathub.org/media \
  --compose-url-policy=full \
  builddir distro/flatpak/com.danklinux.dankcalendar.yml
```

```bash
flatpak run com.danklinux.dankcalendar
```

To rebuild cached modules & tools:

```bash
flatpak run --user org.flatpak.Builder \
  --force-clean --disable-cache --user --install --repo=repo \
  --mirror-screenshots-url=https://dl.flathub.org/media \
  --compose-url-policy=full \
  builddir distro/flatpak/com.danklinux.dankcalendar.yml
```

## Uninstall

Flatpak stores app data under `~/.var/app/com.danklinux.dankcalendar/` separately from a native `make install`.

```bash
flatpak uninstall --delete-data com.danklinux.dankcalendar
```

## Lint

```bash
flatpak run --command=flatpak-builder-lint org.flatpak.Builder \
  --exceptions --user-exceptions distro/flatpak/exceptions.json \
  manifest distro/flatpak/com.danklinux.dankcalendar.yml
```

```bash
flatpak run --command=flatpak-builder-lint org.flatpak.Builder \
  --exceptions --user-exceptions distro/flatpak/exceptions.json \
  repo repo
```
