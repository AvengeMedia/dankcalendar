#!/usr/bin/env python3
"""Exercise installed MIME registration without changing the user's desktop."""

import configparser
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
from urllib.parse import unquote, urlparse

ROOT = Path(__file__).resolve().parent.parent
DESKTOP_ID = "com.danklinux.dankcalendar.desktop"


def run(*args, env):
    completed = subprocess.run(args, cwd=ROOT, env=env, capture_output=True, text=True)
    if completed.returncode:
        print(completed.stdout, end="")
        print(completed.stderr, end="")
        completed.check_returncode()
    return completed.stdout


def check_registration(prefix, env):
    desktop = prefix / "share/applications" / DESKTOP_ID
    run("desktop-file-validate", str(desktop), env=env)
    settings = configparser.ConfigParser(interpolation=None)
    settings.read(desktop)
    assert settings["Desktop Entry"]["Exec"] == f"{prefix}/bin/dcal open %U"
    cache = configparser.ConfigParser()
    cache.read(desktop.parent / "mimeinfo.cache")
    for mime in ("text/calendar", "text/x-vcalendar", "application/ics",
                 "x-scheme-handler/webcal", "x-scheme-handler/webcals"):
        assert DESKTOP_ID in cache["MIME Cache"][mime].split(";"), mime
    assert (prefix / "share/icons/hicolor/scalable/apps/com.danklinux.dankcalendar.svg").is_file()
    assert (prefix / "share/metainfo/com.danklinux.dankcalendar.metainfo.xml").is_file()
    return desktop


def main():
    with tempfile.TemporaryDirectory(prefix="dcal-desktop-") as temporary:
        root = Path(temporary)
        prefix = root / "install"
        env = dict(os.environ, PREFIX=str(prefix), DESTDIR="",
                   XDG_DATA_HOME=str(prefix / "share"), XDG_CONFIG_HOME=str(root / "config"),
                   XDG_CACHE_HOME=str(root / "cache"), XDG_DATA_DIRS="/usr/local/share:/usr/share")
        run("make", "install-desktop", "install-icon", "install-metainfo", env=env)
        desktop = check_registration(prefix, env)

        # The same assets and installer ship in each Linux/FreeBSD architecture archive.
        for target in ("linux-amd64", "linux-arm64", "freebsd-amd64", "freebsd-arm64"):
            binary = root / ("dcal-" + target)
            binary.write_text("#!/bin/sh\nexit 0\n")
            stage = root / target
            staged_env = dict(env, PREFIX="/usr/local", DESTDIR=str(stage))
            run("sh", "scripts/install-release.sh", str(binary), env=staged_env)
            installed = stage / "usr/local"
            assert (installed / "bin/dcal").read_bytes() == binary.read_bytes()
            settings = configparser.ConfigParser(interpolation=None)
            settings.read(installed / "share/applications" / DESKTOP_ID)
            assert settings["Desktop Entry"]["Exec"] == "/usr/local/bin/dcal open %U"

        # Flatpak rewrites %U at installation time, enabling document-portal forwarding.
        build = root / "flatpak-build"
        (build / "files/bin").mkdir(parents=True)
        (build / "files/bin/dcal").write_text("#!/bin/sh\nexit 0\n")
        (build / "files/bin/dcal").chmod(0o755)
        (build / "metadata").write_text(
            "[Application]\nname=com.danklinux.dankcalendar\n"
            "runtime=org.kde.Platform/x86_64/6.11\nsdk=org.kde.Sdk/x86_64/6.11\ncommand=dcal\n")
        run("sh", "scripts/install-assets.sh", env=dict(
            env, PREFIX="", DATA_DIR="/share", DESTDIR=str(build / "files"), DCAL_EXEC="dcal"))
        run("flatpak", "build-finish", str(build), "--command=dcal", env=env)
        repository = root / "flatpak-repo"
        run("flatpak", "build-export", "--disable-sandbox", str(repository), str(build), env=env)
        flatpak_env = dict(env, XDG_DATA_HOME=str(root / "flatpak-data"))
        run("flatpak", "install", "--user", "--no-deps", "--noninteractive", "--assumeyes",
            str(repository), "com.danklinux.dankcalendar", env=flatpak_env)
        exported = root / "flatpak-data/flatpak/exports/share/applications" / DESKTOP_ID
        settings = configparser.ConfigParser(interpolation=None)
        settings.read(exported)
        launcher = settings["Desktop Entry"]
        assert "--file-forwarding" in launcher["Exec"]
        assert "open @@u %U @@" in launcher["Exec"]
        assert "text/calendar;text/x-vcalendar;application/ics;" in launcher["MimeType"]

        # GIO reads the installed handler and expands %U with both selected files.
        (prefix / "bin").mkdir(exist_ok=True)
        capture = root / "arguments.json"
        stub = prefix / "bin/dcal"
        stub.write_text("#!/usr/bin/env python3\nimport json, os, sys\n"
                        "with open(os.environ['DCAL_TEST_ARGUMENTS'], 'w') as f:\n"
                        "    json.dump(sys.argv[1:], f)\n")
        stub.chmod(0o755)
        env.update(PATH=str(prefix / "bin") + os.pathsep + env["PATH"], DCAL_TEST_ARGUMENTS=str(capture))
        files = [root / "Team planning.ics", root / "Appointment.vcs"]
        for path in files:
            path.write_text("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:tests\nEND:VCALENDAR\n")
            assert run("xdg-mime", "query", "filetype", str(path), env=env).strip() == "text/calendar"
        assert DESKTOP_ID in run("gio", "mime", "text/calendar", env=env)
        run("gio", "launch", str(desktop), *(str(path) for path in files), env=env)
        for _ in range(100):
            if capture.exists() and capture.stat().st_size:
                break
            time.sleep(0.02)
        arguments = json.loads(capture.read_text())
        assert arguments[0] == "open"
        paths = [Path(unquote(urlparse(arg).path)) if arg.startswith("file:") else Path(arg)
                 for arg in arguments[1:]]
        assert paths == files, arguments
    print("Native and Flatpak MIME registration, archive installation, and multi-file launch passed.")


if __name__ == "__main__":
    main()
