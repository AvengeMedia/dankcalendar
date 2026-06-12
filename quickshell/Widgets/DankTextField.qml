import QtQuick
import QtQuick.Controls
import qs.Common

Rectangle {
    id: root

    property alias text: input.text
    property alias placeholderText: input.placeholderText
    property alias echoMode: input.echoMode
    property alias readOnly: input.readOnly
    property string label: ""
    property string iconName: ""
    property bool ignoreLeftRightKeys: false
    property alias topPadding: input.topPadding
    property alias bottomPadding: input.bottomPadding

    signal accepted

    function forceActiveFocus() {
        input.forceActiveFocus();
    }

    implicitHeight: 48
    radius: Theme.cornerRadius
    color: Theme.surfaceContainer
    border.color: input.activeFocus ? Theme.primary : Theme.outlineLight
    border.width: 1

    Behavior on border.color {
        ColorAnimation {
            duration: Theme.shortDuration
            easing.type: Theme.standardEasing
        }
    }

    Row {
        anchors.fill: parent
        anchors.leftMargin: Theme.spacingM
        anchors.rightMargin: Theme.spacingM
        spacing: Theme.spacingS

        DankIcon {
            visible: root.iconName !== ""
            name: root.iconName
            size: Theme.iconSize - 6
            color: input.activeFocus ? Theme.primary : Theme.surfaceVariantText
            anchors.verticalCenter: parent.verticalCenter
        }

        TextField {
            id: input
            width: parent.width - (root.iconName !== "" ? Theme.iconSize - 6 + Theme.spacingS : 0)
            anchors.verticalCenter: parent.verticalCenter
            background: null
            color: Theme.surfaceText
            placeholderTextColor: Theme.surfaceVariantText
            font.family: Theme.fontFamily
            font.pixelSize: Theme.fontSizeMedium
            selectionColor: Theme.primarySelected
            selectedTextColor: Theme.surfaceText
            onAccepted: root.accepted()
            Keys.onLeftPressed: event => {
                event.accepted = root.ignoreLeftRightKeys;
            }
            Keys.onRightPressed: event => {
                event.accepted = root.ignoreLeftRightKeys;
            }
        }
    }
}
