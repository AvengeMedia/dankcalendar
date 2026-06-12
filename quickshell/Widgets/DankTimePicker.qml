import QtQuick
import QtQuick.Controls
import qs.Common
import qs.Widgets

Item {
    id: root

    property int minutes: 600
    property bool use24Hour: false
    property int stepMinutes: 30
    property string iconName: "schedule"

    signal timeSelected(int value)

    readonly property int slotCount: Math.ceil(1440 / stepMinutes)

    function formatMinutes(value) {
        const d = new Date(2000, 0, 1, Math.floor(value / 60), value % 60);
        return Qt.formatTime(d, use24Hour ? "HH:mm" : "h:mm AP");
    }

    height: 48

    Rectangle {
        id: field

        anchors.fill: parent
        radius: Theme.cornerRadius
        color: Theme.surfaceContainer
        border.width: 1
        border.color: popup.visible ? Theme.primary : Theme.outlineLight

        Row {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.leftMargin: Theme.spacingM
            anchors.rightMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS

            DankIcon {
                name: root.iconName
                size: Theme.iconSize - 6
                color: popup.visible ? Theme.primary : Theme.surfaceVariantText
                anchors.verticalCenter: parent.verticalCenter
            }

            StyledText {
                text: root.formatMinutes(root.minutes)
                font.pixelSize: Theme.fontSizeMedium
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        StateLayer {
            stateColor: Theme.primary
            cornerRadius: parent.radius
            onClicked: popup.visible ? popup.close() : popup.open()
        }
    }

    Popup {
        id: popup

        y: field.height + Theme.spacingXS
        width: root.width
        height: 240
        padding: Theme.spacingXS
        onOpened: list.positionViewAtIndex(Math.min(Math.floor(root.minutes / root.stepMinutes), root.slotCount - 1), ListView.Center)

        background: Rectangle {
            color: Theme.surfaceContainerHigh
            radius: Theme.cornerRadius
            border.width: 1
            border.color: Theme.outlineMedium
        }

        contentItem: DankListView {
            id: list

            clip: true
            spacing: 1
            model: root.slotCount

            delegate: Rectangle {
                id: slot

                required property int index
                readonly property int slotMinutes: index * root.stepMinutes
                readonly property bool active: slotMinutes === root.minutes

                width: list.width
                height: 32
                radius: Theme.cornerRadiusSmall
                color: active ? Theme.primaryHover : "transparent"

                StyledText {
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingM
                    anchors.verticalCenter: parent.verticalCenter
                    text: root.formatMinutes(slot.slotMinutes)
                    font.pixelSize: Theme.fontSizeMedium
                    color: slot.active ? Theme.primary : Theme.surfaceText
                }

                StateLayer {
                    stateColor: Theme.primary
                    cornerRadius: parent.radius
                    onClicked: {
                        root.timeSelected(slot.slotMinutes);
                        popup.close();
                    }
                }
            }
        }
    }
}
