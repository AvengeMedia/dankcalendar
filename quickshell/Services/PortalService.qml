pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io

Singleton {
    id: root

    // org.freedesktop.appearance color-scheme: 0 = no preference, 1 = prefer dark, 2 = prefer light
    property int systemColorScheme: 0
    property bool available: false

    // Desktops (including DMS) signal light mode as "default" (0), so only
    // prefer-dark counts as dark; without a portal the app stays dark.
    readonly property bool systemPrefersLight: available && systemColorScheme !== 1

    function _applyScheme(raw) {
        const match = raw.match(/uint32 (\d+)/);
        if (!match)
            return;
        available = true;
        systemColorScheme = parseInt(match[1], 10);
    }

    Process {
        id: schemeRead
        command: ["gdbus", "call", "--session", "--dest", "org.freedesktop.portal.Desktop", "--object-path", "/org/freedesktop/portal/desktop", "--method", "org.freedesktop.portal.Settings.Read", "org.freedesktop.appearance", "color-scheme"]
        running: true

        stdout: StdioCollector {
            onStreamFinished: root._applyScheme(text)
        }
    }

    Process {
        id: schemeMonitor
        command: ["gdbus", "monitor", "--session", "--dest", "org.freedesktop.portal.Desktop", "--object-path", "/org/freedesktop/portal/desktop"]
        running: true

        stdout: SplitParser {
            onRead: line => {
                if (line.indexOf("SettingChanged") === -1)
                    return;
                if (line.indexOf("'org.freedesktop.appearance', 'color-scheme'") === -1)
                    return;
                root._applyScheme(line);
            }
        }
    }
}
