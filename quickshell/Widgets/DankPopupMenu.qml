import QtQuick
import QtQuick.Controls
import Quickshell
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

Popup {
    id: root

    // items: [{ id, label, icon, danger?, enabled? }]
    property var items: []

    signal triggered(string itemId)

    function show(anchorItem, x, y) {
        parent = anchorItem;
        root.x = x;
        root.y = y;
        open();
    }

    width: 200
    padding: Theme.spacingXS
    // Outside-press dismissal is armed shortly after opening, otherwise the
    // click that opened the menu can immediately close it. Presses inside the
    // anchor item never auto-close, so the opener can toggle deterministically.
    closePolicy: Popup.CloseOnEscape

    onOpened: armCloseTimer.start()
    onClosed: closePolicy = Popup.CloseOnEscape

    Timer {
        id: armCloseTimer
        interval: 100
        onTriggered: root.closePolicy = Popup.CloseOnEscape | Popup.CloseOnPressOutsideParent
    }

    background: Rectangle {
        color: Theme.surfaceContainerHigh
        radius: Theme.cornerRadius
        border.width: 1
        border.color: Theme.outlineMedium
    }

    contentItem: Column {
        spacing: 1

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        Repeater {
            model: ScriptModel {
                values: root.items
            }

            Rectangle {
                required property var modelData
                readonly property bool itemEnabled: modelData.enabled !== false

                width: parent.width
                height: 36
                radius: Theme.cornerRadiusSmall
                color: "transparent"
                opacity: itemEnabled ? 1 : 0.4

                Row {
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingM
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: Theme.spacingM

                    DankIcon {
                        visible: !!parent.parent.modelData.icon
                        name: parent.parent.modelData.icon || ""
                        size: Theme.iconSize - 6
                        color: parent.parent.modelData.danger ? Theme.error : Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: parent.parent.modelData.label
                        font.pixelSize: Theme.fontSizeMedium
                        color: parent.parent.modelData.danger ? Theme.error : Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                StateLayer {
                    stateColor: parent.modelData.danger ? Theme.error : Theme.primary
                    cornerRadius: parent.radius
                    enabled: parent.itemEnabled
                    disabled: !parent.itemEnabled
                    onClicked: {
                        root.close();
                        root.triggered(parent.modelData.id);
                    }
                }
            }
        }
    }
}
