import QtQuick
import qs.Common

Row {
    property var event: ({})
    property real margin: 3

    readonly property var others: (event && event.mergedFrom) || []

    visible: others.length > 0
    anchors.top: parent.top
    anchors.right: parent.right
    anchors.topMargin: margin
    anchors.rightMargin: margin
    spacing: -3
    z: 1

    Repeater {
        model: parent.others.slice(0, 3)

        Rectangle {
            required property var modelData
            width: 8
            height: 8
            radius: 4
            color: modelData.color
            border.color: Theme.surface
            border.width: 1
        }
    }
}
