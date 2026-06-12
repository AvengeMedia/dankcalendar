pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    function set(key, value) {
    }
    function get(key, fallback) {
        return fallback;
    }
}
