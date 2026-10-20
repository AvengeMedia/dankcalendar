#!/usr/bin/env python3
"""Sync flatpak packaging with a release tag.

metainfo <tag>      ensure an AppStream release entry exists for the tag,
                    generated from commit subjects since the previous tag
pin <tag> <commit>  point the manifest at the tag, or at a bare commit when
                    packaging fixes landed after the tag
"""

import argparse
import html
import pathlib
import re
import subprocess
import sys

METAINFO = pathlib.Path("assets/dev.duxlabs.calendar.metainfo.xml")
MANIFEST = pathlib.Path("distro/flatpak/dev.duxlabs.calendar.yml")
SKIP_SUBJECT = re.compile(r"^(nix|flake|i18n|flatpak|ci|chore|build)[(:]", re.I)
RELEASE_LINE = re.compile(r'^ *<release version="([^"]+)"', re.M)


def version_key(version):
    return [int(n) for n in re.findall(r"\d+", version)]


def git(*args):
    return subprocess.run(["git", *args], capture_output=True, text=True, check=True).stdout


def release_notes(tag):
    prev = subprocess.run(
        ["git", "describe", "--tags", "--abbrev=0", f"{tag}^"],
        capture_output=True, text=True,
    )
    span = f"{prev.stdout.strip()}..{tag}" if prev.returncode == 0 else tag
    subjects = [
        s for s in git("log", "--no-merges", "--format=%s", span).splitlines()
        if s and not SKIP_SUBJECT.match(s)
    ]
    if not subjects:
        return "        <p>Maintenance release.</p>"
    items = "\n".join(f"          <li>{html.escape(s)}</li>" for s in subjects[:10])
    return f"        <ul>\n{items}\n        </ul>"


def cmd_metainfo(args):
    version = args.tag.removeprefix("v")
    text = METAINFO.read_text()
    if f'<release version="{version}"' in text:
        print(f"metainfo already has a {version} release entry")
        return

    date = git("log", "-1", "--format=%cs", args.tag).strip()
    entry = (
        f'    <release version="{version}" date="{date}">\n'
        f"      <description>\n{release_notes(args.tag)}\n      </description>\n"
        f"    </release>\n"
    )
    closing = "  </releases>\n"
    if closing not in text:
        sys.exit("metainfo has no <releases> section")
    at = next(
        (m.start() for m in RELEASE_LINE.finditer(text) if version_key(m.group(1)) < version_key(version)),
        text.index(closing),
    )
    METAINFO.write_text(text[:at] + entry + text[at:])
    print(f"added {version} release entry")


def cmd_pin(args):
    version = args.tag.removeprefix("v")
    text = MANIFEST.read_text()
    text = re.sub(r"main\.Version=[^\s\\]+", f"main.Version={version}", text)
    text = re.sub(r"main\.Commit=\w+", f"main.Commit={args.commit[:8]}", text)

    source = re.compile(
        r"(url: https://github\.com/duxlabs-dev/dux-calendar\.git\n)(\s*)(?:tag: \S+\n\s*)?commit: \S+"
    )
    m = source.search(text)
    if m is None:
        sys.exit("manifest has no dankcalendar git source")

    tag_commit = git("rev-parse", f"{args.tag}^{{commit}}").strip()
    if args.commit == tag_commit:
        block = f"{m.group(1)}{m.group(2)}tag: {args.tag}\n{m.group(2)}commit: {args.commit}"
    else:
        block = f"{m.group(1)}{m.group(2)}commit: {args.commit}"
    MANIFEST.write_text(text[: m.start()] + block + text[m.end():])
    print(f"pinned manifest to {args.tag} @ {args.commit[:8]}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(required=True)
    p = sub.add_parser("metainfo")
    p.add_argument("tag")
    p.set_defaults(func=cmd_metainfo)
    p = sub.add_parser("pin")
    p.add_argument("tag")
    p.add_argument("commit")
    p.set_defaults(func=cmd_pin)
    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
