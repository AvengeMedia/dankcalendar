pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    property var fileBrowserSettings: ({})

    function saveCache() {
    }
}
