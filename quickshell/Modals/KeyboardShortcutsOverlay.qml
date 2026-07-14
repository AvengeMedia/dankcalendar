import QtQuick
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

Item {
    id: root

    signal dismissed

    readonly property var groups: [
        {
            "title": I18n.tr("Navigation", "keyboard shortcuts overlay section header"),
            "rows": [
                {
                    "keys": ["t"],
                    "label": I18n.tr("Go to today", "keyboard shortcut description")
                },
                {
                    "keys": ["h", "←"],
                    "label": I18n.tr("Previous day", "keyboard shortcut description")
                },
                {
                    "keys": ["l", "→"],
                    "label": I18n.tr("Next day", "keyboard shortcut description")
                },
                {
                    "keys": ["j", "↓"],
                    "label": I18n.tr("Next event (next week in month view)", "keyboard shortcut description")
                },
                {
                    "keys": ["k", "↑"],
                    "label": I18n.tr("Previous event (previous week in month view)", "keyboard shortcut description")
                },
                {
                    "keys": ["[", "PgUp"],
                    "label": I18n.tr("Previous day, week or month", "keyboard shortcut description")
                },
                {
                    "keys": ["]", "PgDn"],
                    "label": I18n.tr("Next day, week or month", "keyboard shortcut description")
                },
                {
                    "keys": ["{"],
                    "label": I18n.tr("Previous year", "keyboard shortcut description")
                },
                {
                    "keys": ["}"],
                    "label": I18n.tr("Next year", "keyboard shortcut description")
                },
                {
                    "keys": ["g"],
                    "label": I18n.tr("Go to date", "keyboard shortcut description")
                },
                {
                    "keys": ["Tab"],
                    "label": I18n.tr("Next day with events", "keyboard shortcut description")
                },
                {
                    "keys": ["Shift+Tab"],
                    "label": I18n.tr("Previous day with events", "keyboard shortcut description")
                },
                {
                    "keys": ["Shift+←→↑↓"],
                    "label": I18n.tr("Select days for a new event (month view)", "keyboard shortcut description")
                },
                {
                    "keys": ["Enter"],
                    "label": I18n.tr("Open or create event", "keyboard shortcut description")
                },
                {
                    "keys": ["Esc"],
                    "label": I18n.tr("Clear event selection", "keyboard shortcut description")
                }
            ]
        },
        {
            "title": I18n.tr("View", "keyboard shortcuts overlay section header"),
            "rows": [
                {
                    "keys": ["d", "Ctrl+1"],
                    "label": I18n.tr("Day view", "keyboard shortcut description")
                },
                {
                    "keys": ["w", "Ctrl+2"],
                    "label": I18n.tr("Week view", "keyboard shortcut description")
                },
                {
                    "keys": ["m", "Ctrl+3"],
                    "label": I18n.tr("Month view", "keyboard shortcut description")
                },
                {
                    "keys": ["a", "Ctrl+4"],
                    "label": I18n.tr("Agenda view", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+5"],
                    "label": I18n.tr("Tasks view", "keyboard shortcut description")
                }
            ]
        },
        {
            "title": I18n.tr("Sidebar", "keyboard shortcuts overlay section header"),
            "rows": [
                {
                    "keys": ["s"],
                    "label": I18n.tr("Focus or leave the sidebar", "keyboard shortcut description")
                },
                {
                    "keys": ["j", "k"],
                    "label": I18n.tr("Move through sidebar items", "keyboard shortcut description")
                },
                {
                    "keys": ["h", "l"],
                    "label": I18n.tr("Collapse or expand a section", "keyboard shortcut description")
                },
                {
                    "keys": ["Enter", "Space"],
                    "label": I18n.tr("Toggle calendar, switch view or sync account", "keyboard shortcut description")
                },
                {
                    "keys": ["Del"],
                    "label": I18n.tr("Remove calendar or account", "keyboard shortcut description")
                },
                {
                    "keys": ["r"],
                    "label": I18n.tr("Rename calendar", "keyboard shortcut description")
                },
                {
                    "keys": ["Esc"],
                    "label": I18n.tr("Back to the calendar", "keyboard shortcut description")
                }
            ]
        },
        {
            "title": I18n.tr("General", "keyboard shortcuts overlay section header"),
            "rows": [
                {
                    "keys": ["c"],
                    "label": I18n.tr("Create event", "keyboard shortcut description")
                },
                {
                    "keys": ["/", "Ctrl+F"],
                    "label": I18n.tr("Search", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+,"],
                    "label": I18n.tr("Settings", "keyboard shortcut description")
                },
                {
                    "keys": ["?"],
                    "label": I18n.tr("Toggle this help", "keyboard shortcut description")
                }
            ]
        }
    ]

    Rectangle {
        anchors.fill: parent
        color: Qt.rgba(0, 0, 0, 0.55)

        MouseArea {
            anchors.fill: parent
            onClicked: root.dismissed()
        }
    }

    Rectangle {
        anchors.centerIn: parent
        width: Math.min(640, parent.width - Theme.spacingXL * 2)
        height: Math.min(card.implicitHeight + Theme.spacingXL * 2, parent.height - Theme.spacingXL * 2)
        radius: Theme.cornerRadius
        color: Theme.surfaceContainerHigh
        border.color: Theme.outlineLight
        border.width: 1

        MouseArea {
            anchors.fill: parent
        }

        DankFlickable {
            anchors.fill: parent
            anchors.margins: Theme.spacingXL
            contentWidth: width
            contentHeight: card.implicitHeight
            clip: true

            Column {
                id: card
                width: parent.width
                spacing: Theme.spacingL

                Row {
                    width: parent.width
                    spacing: Theme.spacingM

                    DankIcon {
                        name: "keyboard"
                        size: Theme.iconSize
                        color: Theme.primary
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: I18n.tr("Keyboard shortcuts", "keyboard shortcuts overlay title")
                        font.pixelSize: Theme.fontSizeLarge
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                Repeater {
                    model: root.groups

                    Column {
                        required property var modelData
                        width: parent.width
                        spacing: Theme.spacingS

                        StyledText {
                            text: parent.modelData.title
                            font.pixelSize: Theme.fontSizeSmall
                            font.weight: Font.Medium
                            color: Theme.surfaceVariantText
                            width: parent.width
                        }

                        Repeater {
                            model: parent.modelData.rows

                            Item {
                                required property var modelData
                                width: parent.width
                                height: 28

                                StyledText {
                                    anchors.left: parent.left
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: parent.modelData.label
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.surfaceText
                                }

                                Row {
                                    anchors.right: parent.right
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: Theme.spacingXS

                                    Repeater {
                                        model: parent.parent.modelData.keys

                                        Rectangle {
                                            required property var modelData
                                            height: 22
                                            width: Math.max(22, keyLabel.implicitWidth + Theme.spacingS * 2)
                                            radius: Theme.cornerRadiusSmall
                                            color: Theme.surfaceContainerHighest
                                            border.color: Theme.outlineLight
                                            border.width: 1

                                            StyledText {
                                                id: keyLabel
                                                anchors.centerIn: parent
                                                text: parent.modelData
                                                font.pixelSize: Theme.fontSizeSmall
                                                color: Theme.surfaceText
                                                isMonospace: true
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
