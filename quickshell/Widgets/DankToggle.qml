import QtQuick
import qs.Common

Item {
    id: root

    property bool checked: false
    property bool toggleEnabled: true

    signal toggled(bool checked)

    implicitWidth: 48
    implicitHeight: 28

    Rectangle {
        id: track
        anchors.fill: parent
        radius: height / 2
        color: root.checked ? Theme.primary : Theme.surfaceVariant
        opacity: root.toggleEnabled ? 1.0 : 0.5

        Behavior on color {
            ColorAnimation {
                duration: Theme.shortDuration
                easing.type: Theme.standardEasing
            }
        }
    }

    Rectangle {
        id: thumb
        width: 20
        height: 20
        radius: 10
        anchors.verticalCenter: parent.verticalCenter
        x: root.checked ? parent.width - width - 4 : 4
        color: root.checked ? Theme.primaryText : Theme.surfaceText

        Behavior on x {
            NumberAnimation {
                duration: Theme.shortDuration
                easing.type: Theme.standardEasing
            }
        }
    }

    MouseArea {
        anchors.fill: parent
        cursorShape: root.toggleEnabled ? Qt.PointingHandCursor : Qt.ArrowCursor
        enabled: root.toggleEnabled
        onClicked: {
            root.checked = !root.checked;
            root.toggled(root.checked);
        }
    }
}
