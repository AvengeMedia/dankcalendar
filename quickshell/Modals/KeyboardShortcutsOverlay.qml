import QtQuick
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

DankOverlayDialog {
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
            "title": I18n.tr("Event actions", "keyboard shortcuts overlay section header"),
            "rows": [
                {
                    "keys": ["Ctrl+C"],
                    "label": I18n.tr("Copy selected events", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+X"],
                    "label": I18n.tr("Cut selected events", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+V"],
                    "label": I18n.tr("Paste events on the selected day", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+D"],
                    "label": I18n.tr("Duplicate selected events", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+Shift+D"],
                    "label": I18n.tr("Duplicate selected events to the next day", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+A"],
                    "label": I18n.tr("Select all events on the selected day", "keyboard shortcut description")
                },
                {
                    "keys": ["Ctrl+Click"],
                    "label": I18n.tr("Add or remove an event from the selection", "keyboard shortcut description")
                },
                {
                    "keys": ["Shift+Click"],
                    "label": I18n.tr("Select a range of events", "keyboard shortcut description")
                },
                {
                    "keys": ["Menu"],
                    "label": I18n.tr("Open event actions", "keyboard shortcut description")
                },
                {
                    "keys": ["Del", "Backspace"],
                    "label": I18n.tr("Delete selected events", "keyboard shortcut description")
                }
            ]
        },
        {
            "title": I18n.tr("General", "keyboard shortcuts overlay section header"),
            "rows": [
                {
                    "keys": ["c", "Ctrl+N"],
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

    takesFocus: false
    title: I18n.tr("Keyboard shortcuts", "keyboard shortcuts overlay title")
    onRejected: dismissed()

    Repeater {
        model: root.groups

        Column {
            id: group
            required property var modelData
            width: parent.width
            spacing: Theme.spacingS

            StyledText {
                text: group.modelData.title
                font.pixelSize: Theme.fontSizeSmall
                font.weight: Theme.fontWeightMedium
                color: Theme.primary
                width: parent.width
            }

            Column {
                width: parent.width
                spacing: Theme.groupedListGap

                Repeater {
                    model: group.modelData.rows

                    DankListRow {
                        id: row
                        required property var modelData
                        required property int index
                        width: parent.width
                        height: Theme.buttonHeightS
                        firstInGroup: index === 0
                        lastInGroup: index === group.modelData.rows.length - 1

                        StyledText {
                            anchors.left: parent.left
                            anchors.leftMargin: Theme.spacingM
                            anchors.right: keys.left
                            anchors.rightMargin: Theme.spacingM
                            anchors.verticalCenter: parent.verticalCenter
                            text: row.modelData.label
                            font.pixelSize: Theme.fontSizeMedium
                            color: row.contentColor
                            elide: Text.ElideRight
                        }

                        Row {
                            id: keys
                            anchors.right: parent.right
                            anchors.rightMargin: Theme.spacingM
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: Theme.spacingXS

                            Repeater {
                                model: row.modelData.keys

                                DankKeycap {
                                    required property string modelData
                                    text: modelData
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
