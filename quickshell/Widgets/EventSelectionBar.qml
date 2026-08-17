import QtQuick
import Quickshell
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

Item {
    id: root

    property var controller: null

    signal deleteRequested

    readonly property var actions: [
        {
            id: "copy",
            label: I18n.tr("Copy", "multi-event action shelf label"),
            icon: "content_copy"
        },
        {
            id: "cut",
            label: I18n.tr("Cut", "multi-event action shelf label"),
            icon: "content_cut"
        },
        {
            id: "duplicate",
            label: I18n.tr("Again", "multi-event action shelf duplicate label"),
            icon: "file_copy"
        },
        {
            id: "tomorrow",
            label: I18n.tr("+1 day", "multi-event action shelf repeat tomorrow label"),
            icon: "update"
        },
        {
            id: "week",
            label: I18n.tr("+1 week", "multi-event action shelf repeat next week label"),
            icon: "event_repeat"
        },
        {
            id: "delete",
            label: I18n.tr("Delete", "multi-event action shelf label"),
            icon: "delete_outline",
            danger: true
        },
        {
            id: "clear",
            label: I18n.tr("Clear", "multi-event action shelf clear selection label"),
            icon: "close"
        }
    ]

    function run(actionId) {
        switch (actionId) {
        case "copy":
            controller.copy();
            break;
        case "cut":
            controller.cut();
            break;
        case "duplicate":
            controller.duplicate(0);
            break;
        case "tomorrow":
            controller.duplicate(1);
            break;
        case "week":
            controller.duplicate(7);
            break;
        case "delete":
            deleteRequested();
            break;
        case "clear":
            controller.clear();
            break;
        }
    }

    visible: controller && controller.count > 1
    implicitWidth: selectionLabel.width + actionRow.width + Theme.spacingL * 2 + Theme.spacingM
    implicitHeight: 54
    opacity: visible ? 1 : 0
    scale: visible ? 1 : 0.96

    Behavior on opacity {
        NumberAnimation {
            duration: Theme.shortDuration
            easing.type: Theme.standardEasing
        }
    }

    Behavior on scale {
        NumberAnimation {
            duration: Theme.shortDuration
            easing.type: Theme.standardEasing
        }
    }

    StyledRect {
        anchors.fill: parent
        color: Theme.surfaceContainerHigh
        radius: Theme.cornerRadiusLarge
        border.color: Theme.primary
        border.width: 1
    }

    Row {
        anchors.centerIn: parent
        spacing: Theme.spacingM

        Row {
            id: selectionLabel
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS

            DankIcon {
                name: "select_all"
                size: Theme.iconSize - 2
                color: Theme.primary
                anchors.verticalCenter: parent.verticalCenter
            }

            StyledText {
                text: I18n.tr("%1 selected", "multi-event action shelf selection count; %1 is event count").arg(root.controller ? root.controller.count : 0)
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.Medium
                color: Theme.surfaceText
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        Row {
            id: actionRow
            spacing: 2

            Repeater {
                model: ScriptModel {
                    values: root.actions
                }

                Item {
                    id: action
                    required property var modelData
                    width: 48
                    height: 44

                    Column {
                        anchors.centerIn: parent
                        spacing: 0

                        DankIcon {
                            name: action.modelData.icon
                            size: Theme.iconSize - 4
                            color: action.modelData.danger ? Theme.error : Theme.surfaceVariantText
                            anchors.horizontalCenter: parent.horizontalCenter
                        }

                        StyledText {
                            text: action.modelData.label
                            font.pixelSize: 9
                            color: action.modelData.danger ? Theme.error : Theme.surfaceVariantText
                            anchors.horizontalCenter: parent.horizontalCenter
                        }
                    }

                    StateLayer {
                        stateColor: action.modelData.danger ? Theme.error : Theme.primary
                        cornerRadius: Theme.cornerRadiusSmall
                        disabled: root.controller ? root.controller.busy : true
                        onClicked: root.run(action.modelData.id)
                    }
                }
            }
        }
    }
}
