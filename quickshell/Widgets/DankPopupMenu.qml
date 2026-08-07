import QtQuick
import QtQuick.Controls
import Quickshell
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

Popup {
    id: root

    // items: [{ id, label, icon, shortcut?, danger?, enabled? }]
    // Non-action rows use { type: "header", label, subtitle? } or
    // { type: "separator" }.
    property var items: []
    property real preferredWidth: 228
    property int currentIndex: -1

    signal triggered(string itemId)

    function show(anchorItem, x, y) {
        parent = anchorItem;
        root.x = x;
        root.y = y;
        open();
    }

    function isAction(index) {
        if (index < 0 || index >= items.length)
            return false;
        const item = items[index];
        return (!item.type || item.type === "action") && item.enabled !== false;
    }

    function moveCurrent(direction) {
        if (items.length === 0)
            return;
        let next = currentIndex;
        for (let i = 0; i < items.length; i++) {
            next = (next + direction + items.length) % items.length;
            if (isAction(next)) {
                currentIndex = next;
                return;
            }
        }
    }

    function triggerCurrent() {
        if (!isAction(currentIndex))
            return;
        const itemId = items[currentIndex].id;
        close();
        triggered(itemId);
    }

    width: preferredWidth
    padding: Theme.spacingXS
    // Outside-press dismissal is armed shortly after opening, otherwise the
    // click that opened the menu can immediately close it. Presses inside the
    // anchor item never auto-close, so the opener can toggle deterministically.
    closePolicy: Popup.CloseOnEscape

    focus: true
    onOpened: {
        currentIndex = -1;
        moveCurrent(1);
        forceActiveFocus();
        armCloseTimer.start();
    }
    onClosed: {
        closePolicy = Popup.CloseOnEscape;
        currentIndex = -1;
    }

    Keys.onPressed: event => {
        switch (event.key) {
        case Qt.Key_Up:
            moveCurrent(-1);
            break;
        case Qt.Key_Down:
            moveCurrent(1);
            break;
        case Qt.Key_Return:
        case Qt.Key_Enter:
        case Qt.Key_Space:
            triggerCurrent();
            break;
        default:
            event.accepted = false;
            return;
        }
        event.accepted = true;
    }

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
                id: menuRow
                required property var modelData
                required property int index
                readonly property string itemType: modelData.type || "action"
                readonly property bool isAction: itemType === "action"
                readonly property bool itemEnabled: isAction && modelData.enabled !== false

                width: parent.width
                height: itemType === "separator" ? 9 : (itemType === "header" ? 48 : 38)
                radius: Theme.cornerRadiusSmall
                color: isAction && root.currentIndex === index ? Theme.primaryBackground : "transparent"
                opacity: !isAction || itemEnabled ? 1 : 0.4

                Rectangle {
                    visible: menuRow.itemType === "separator"
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.leftMargin: Theme.spacingS
                    anchors.rightMargin: Theme.spacingS
                    height: 1
                    color: Theme.outlineLight
                }

                Column {
                    visible: menuRow.itemType === "header"
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    anchors.leftMargin: Theme.spacingM
                    anchors.rightMargin: Theme.spacingM
                    spacing: 1

                    StyledText {
                        text: menuRow.modelData.label || ""
                        font.pixelSize: Theme.fontSizeMedium
                        font.weight: Font.Medium
                        color: Theme.surfaceText
                        width: parent.width
                        elide: Text.ElideRight
                    }

                    StyledText {
                        visible: text !== ""
                        text: menuRow.modelData.subtitle || ""
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceVariantText
                        width: parent.width
                        elide: Text.ElideRight
                    }
                }

                Row {
                    visible: menuRow.isAction
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.leftMargin: Theme.spacingM
                    anchors.rightMargin: Theme.spacingM
                    anchors.verticalCenter: parent.verticalCenter
                    spacing: Theme.spacingM

                    DankIcon {
                        visible: !!menuRow.modelData.icon
                        name: menuRow.modelData.icon || ""
                        size: Theme.iconSize - 6
                        color: menuRow.modelData.danger ? Theme.error : Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }

                    StyledText {
                        text: menuRow.modelData.label
                        font.pixelSize: Theme.fontSizeMedium
                        color: menuRow.modelData.danger ? Theme.error : Theme.surfaceText
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width - (menuRow.modelData.icon ? Theme.iconSize - 6 + Theme.spacingM : 0) - shortcutText.width
                        elide: Text.ElideRight
                    }

                    StyledText {
                        id: shortcutText
                        visible: text !== ""
                        text: menuRow.modelData.shortcut || ""
                        font.pixelSize: Theme.fontSizeSmall
                        color: Theme.surfaceVariantText
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }

                StateLayer {
                    visible: menuRow.isAction
                    stateColor: menuRow.modelData.danger ? Theme.error : Theme.primary
                    cornerRadius: parent.radius
                    enabled: menuRow.itemEnabled
                    disabled: !menuRow.itemEnabled
                    onClicked: {
                        root.currentIndex = menuRow.index;
                        root.close();
                        root.triggered(menuRow.modelData.id);
                    }
                }
            }
        }
    }
}
