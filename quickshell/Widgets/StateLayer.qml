import QtQuick
import qs.Common

MouseArea {
    id: root

    property bool disabled: false
    property color stateColor: Theme.surfaceText
    property real cornerRadius: parent && parent.radius !== undefined ? parent.radius : Theme.cornerRadius

    readonly property real stateOpacity: {
        if (disabled)
            return 0;
        if (pressed)
            return 0.12;
        if (containsMouse)
            return 0.08;
        return 0;
    }

    anchors.fill: parent
    cursorShape: disabled ? Qt.ArrowCursor : Qt.PointingHandCursor
    hoverEnabled: true

    Rectangle {
        anchors.fill: parent
        radius: root.cornerRadius
        color: Qt.rgba(root.stateColor.r, root.stateColor.g, root.stateColor.b, root.stateOpacity)

        Behavior on color {
            ColorAnimation {
                duration: Theme.shorterDuration
                easing.type: Theme.standardEasing
            }
        }
    }
}
