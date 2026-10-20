#!/bin/sh
# Shared desktop assets for source installs and Linux/FreeBSD binary archives.
set -eu

asset_dir=$(CDPATH= cd -- "$(dirname -- "$0")/../assets" && pwd)
desktop_id=dev.duxlabs.calendar
legacy_id=com.danklinux.dankcalendar
prefix=${PREFIX:-/usr/local}
dcal_exec=${DCAL_EXEC:-$prefix/bin/dcal}
staging=${DESTDIR:-}
data_dir=${DATA_DIR:-$prefix/share}
applications_dir=${APPLICATIONS_DIR:-$data_dir/applications}
icon_dir=${ICON_DIR:-$data_dir/icons/hicolor/scalable/apps}
metainfo_dir=${METAINFO_DIR:-$data_dir/metainfo}

install_asset() {
    mkdir -p "$staging$2"
    install -m 644 "$asset_dir/$1" "$staging$2/$1"
}

install_desktop() {
    if [ -z "$staging" ] && ! command -v update-desktop-database >/dev/null 2>&1; then
        echo 'Desktop registration requires desktop-file-utils (update-desktop-database).' >&2
        exit 1
    fi
    mkdir -p "$staging$applications_dir"
    sed "s|^Exec=dcal |Exec=$dcal_exec |" \
        "$asset_dir/$desktop_id.desktop" > "$staging$applications_dir/$desktop_id.desktop"
    chmod 644 "$staging$applications_dir/$desktop_id.desktop"
    rm -f "$staging$applications_dir/$legacy_id.desktop"
    if [ -z "$staging" ]; then
        update-desktop-database -q "$applications_dir"
    fi
}

install_metainfo() {
    install_asset "$desktop_id.metainfo.xml" "$metainfo_dir"
    rm -f "$staging$metainfo_dir/$legacy_id.metainfo.xml"
}

install_icon() {
    install_asset "$desktop_id.svg" "$icon_dir"
    rm -f "$staging$icon_dir/$legacy_id.svg"
    if [ -z "$staging" ] && command -v gtk-update-icon-cache >/dev/null 2>&1; then
        gtk-update-icon-cache -q "$data_dir/icons/hicolor" 2>/dev/null || true
    fi
}

case "${1:-all}" in
    desktop) install_desktop ;;
    icon) install_icon ;;
    metainfo) install_metainfo ;;
    all)
        install_desktop
        install_icon
        install_metainfo
        ;;
    *) echo "Usage: $0 [all|desktop|icon|metainfo]" >&2; exit 2 ;;
esac
