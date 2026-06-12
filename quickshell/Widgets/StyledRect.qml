import QtQuick
import qs.Common

Rectangle {
    color: "transparent"
    radius: Theme.cornerRadius

    Behavior on radius {
        NumberAnimation {
            duration: Theme.shortDuration
            easing.type: Theme.standardEasing
        }
    }

    Behavior on opacity {
        NumberAnimation {
            duration: Theme.shortDuration
            easing.type: Theme.standardEasing
        }
    }
}
