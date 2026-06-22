pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import qs.Services

Singleton {
    id: root

    // org.freedesktop.appearance color-scheme: 0 = no preference, 1 = prefer dark, 2 = prefer light
    property int systemColorScheme: 0
    property bool available: false

    // Desktops (including DMS) signal light mode as "default" (0), so only
    // prefer-dark counts as dark; without a portal the app stays dark.
    readonly property bool systemPrefersLight: available && systemColorScheme !== 1

    function _apply(data) {
        if (!data)
            return;
        available = data.available === true;
        systemColorScheme = data.colorScheme || 0;
    }

    function refresh() {
        if (typeof DankCalService === "undefined")
            return;
        DankCalService.sendRequest("system.colorScheme.get", null, response => {
            if (response.error)
                return;
            root._apply(response.result);
        });
    }

    Component.onCompleted: refresh()

    Connections {
        target: typeof DankCalService !== "undefined" ? DankCalService : null

        function onColorSchemeUpdate(data) {
            root._apply(data);
        }

        function onConnectionStateChanged() {
            root.refresh();
        }
    }
}
