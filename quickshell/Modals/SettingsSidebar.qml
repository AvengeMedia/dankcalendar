import QtQuick
import Quickshell
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

Item {
    id: root

    property int currentIndex: 0

    signal tabSelected(int index)

    readonly property var tabs: [
        {
            label: I18n.tr("General", "settings sidebar tab label"),
            icon: "tune"
        },
        {
            label: I18n.tr("Appearance", "settings sidebar tab label"),
            icon: "palette"
        },
        {
            label: I18n.tr("Calendars", "settings sidebar tab label"),
            icon: "calendar_month"
        },
        {
            label: I18n.tr("Accounts", "settings sidebar tab label"),
            icon: "account_circle"
        },
        {
            label: I18n.tr("Notifications", "settings sidebar tab label"),
            icon: "notifications"
        },
        {
            label: I18n.tr("About", "settings sidebar tab label"),
            icon: "info"
        }
    ]

    implicitWidth: 220

    Rectangle {
        anchors.fill: parent
        color: Theme.surfaceContainer
        opacity: 0.6
    }

    Column {
        anchors.fill: parent
        anchors.margins: Theme.spacingM
        spacing: 2

        Repeater {
            model: ScriptModel {
                values: root.tabs
            }

            StyledRect {
                required property int index
                required property var modelData
                readonly property bool active: index === root.currentIndex

                width: parent.width
                height: 40
                radius: Theme.cornerRadius
                color: active ? Theme.primaryHover : "transparent"

                Row {
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingM
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: Theme.spacingM

                    DankIcon {
                        name: parent.parent.modelData.icon
                        size: Theme.iconSize - 4
                        color: parent.parent.active ? Theme.primary : Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: parent.parent.modelData.label
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: parent.parent.active ? Font.Medium : Font.Normal
                        color: parent.parent.active ? Theme.primary : Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                StateLayer {
                    stateColor: Theme.primary
                    cornerRadius: parent.radius
                    onClicked: root.tabSelected(parent.index)
                }
            }
        }
    }
}
