import QtQuick
import qs.Common

Item {
    id: root

    property string name: ""
    property int size: Theme.fontSizeMedium
    property alias color: icon.color

    width: size
    height: size
    visible: text.length > 0

    // This is for file browser, particularly - might want another map later for app IDs
    readonly property var iconMap: ({
            // --- Distribution logos ---
            "debian": "\u{f08da}",
            "arch": "\u{f08c7}",
            "archcraft": "\u{f345}",
            "guix": "\u{f325}",
            "fedora": "\u{f08db}",
            "nixos": "\u{f1105}",
            "ubuntu": "\u{f0548}",
            "gentoo": "\u{f08e8}",
            "endeavouros": "\u{f322}",
            "manjaro": "\u{f160a}",
            "opensuse": "\u{f314}",
            "artix": "\u{f31f}",
            "void": "\u{f32e}",

            // --- special types ---
            "folder": "\u{F024B}",
            "file": "\u{F0214}",

            // --- special filenames (no extension) ---
            "docker": "\u{F0868}",
            "makefile": "\u{F09EE}",
            "license": "\u{F09EE}",
            "readme": "\u{F0354}",

            // --- programming languages ---
            "rs": "\u{F1617}",
            "dart": "\u{e798}",
            "go": "\u{F07D3}",
            "py": "\u{F0320}",
            "js": "\u{F031E}",
            "jsx": "\u{F031E}",
            "ts": "\u{F06E6}",
            "tsx": "\u{F06E6}",
            "java": "\u{F0B37}",
            "c": "\u{F0671}",
            "cpp": "\u{F0672}",
            "cxx": "\u{F0672}",
            "h": "\u{F0672}",
            "hpp": "\u{F0672}",
            "cs": "\u{F031B}",
            "html": "\u{e60e}",
            "htm": "\u{e60e}",
            "css": "\u{E6b8}",
            "scss": "\u{F031C}",
            "less": "\u{F031C}",
            "md": "\u{F0354}",
            "markdown": "\u{F0354}",
            "json": "\u{eb0f}",
            "jsonc": "\u{eb0f}",
            "yaml": "\u{e8eb}",
            "yml": "\u{e8eb}",
            "xml": "\u{F09EE}",
            "sql": "\u{f1c0}",

            // --- scripts / shells ---
            "sh": "\u{f0bc1}",
            "bash": "\u{f0bc1}",
            "zsh": "\u{f0bc1}",
            "fish": "\u{f0bc1}",
            "ps1": "\u{f0bc1}",
            "bat": "\u{f0bc1}",

            // --- data / config ---
            "toml": "\u{e6b2}",
            "ini": "\u{F09EE}",
            "conf": "\u{F09EE}",
            "cfg": "\u{F09EE}",
            "csv": "\u{eefc}",
            "tsv": "\u{F021C}",

            // --- docs / office ---
            "pdf": "\u{F0226}",
            "doc": "\u{F09EE}",
            "docx": "\u{F09EE}",
            "rtf": "\u{F09EE}",
            "ppt": "\u{F09EE}",
            "pptx": "\u{F09EE}",
            "log": "\u{F09EE}",
            "xls": "\u{F021C}",
            "xlsx": "\u{F021C}",

            // --- images ---
            "ico": "\u{F021F}",

            // --- audio / video ---
            "mp3": "\u{e638}",
            "wav": "\u{e638}",
            "flac": "\u{e638}",
            "ogg": "\u{e638}",
            "mp4": "\u{f0567}",
            "mkv": "\u{f0567}",
            "webm": "\u{f0567}",
            "mov": "\u{f0567}",

            // --- archives / packages ---
            "zip": "\u{e6aa}",
            "tar": "\u{f003c}",
            "gz": "\u{f003c}",
            "bz2": "\u{f003c}",
            "7z": "\u{f003c}",

            // --- containers / infra / cloud ---
            "dockerfile": "\u{F0868}",
            "yml.k8s": "\u{F09EE}",
            "yaml.k8s": "\u{F09EE}",
            "tf": "\u{F09EE}",
            "tfvars": "\u{F09EE}",

            // --- moon phases
            "moon_new": "\u{F0F64}",
            "moon_waxing_crescent": "\u{F0F67}",
            "moon_first_quarter": "\u{F0F61}",
            "moon_waxing_gibbous": "\u{F0F68}",
            "moon_full": "\u{F0F62}",
            "moon_waning_gibbous": "\u{F0F66}",
            "moon_last_quarter": "\u{F0F63}",
            "moon_waning_crescent": "\u{F0F65}"
        })

    readonly property string text: iconMap[name] || iconMap["file"] || ""

    function getIconForFile(fileName) {
        const lowerName = fileName.toLowerCase();
        if (lowerName.startsWith("dockerfile")) {
            return "docker";
        }
        const ext = fileName.split('.').pop();
        return ext || "";
    }

    FontLoader {
        id: firaCodeFont
        source: Qt.resolvedUrl("../assets/fonts/nerd-fonts/FiraCodeNerdFont-Regular.ttf")
    }

    StyledText {
        id: icon

        anchors.centerIn: parent

        font.family: firaCodeFont.name
        font.pixelSize: root.size
        color: Theme.surfaceText
        text: root.text
        antialiasing: true
    }
}
