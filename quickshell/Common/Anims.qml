pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

Singleton {
    id: root

    readonly property int durShort: 200
    readonly property int durMed: 450
    readonly property int durLong: 600

    readonly property var standard: [0.20, 0.00, 0.00, 1.00, 1.00, 1.00]
    readonly property var standardDecel: [0.00, 0.00, 0.00, 1.00, 1.00, 1.00]
    readonly property var standardAccel: [0.30, 0.00, 1.00, 1.00, 1.00, 1.00]
    readonly property var emphasized: [0.05, 0.00, 0.133333, 0.06, 0.166667, 0.40, 0.208333, 0.82, 0.25, 1.00, 1.00, 1.00]
}
