import QtQuick
import QtQuick.Controls
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

Popup {
    id: root

    property string title: ""
    property string message: ""
    property string confirmText: I18n.tr("Confirm", "confirm dialog default confirm button")
    property bool danger: false

    signal confirmed

    function show(opts) {
        title = opts.title || "";
        message = opts.message || "";
        confirmText = opts.confirmText || I18n.tr("Confirm", "confirm dialog default confirm button");
        danger = opts.danger === true;
        open();
    }

    parent: Overlay.overlay
    anchors.centerIn: parent
    modal: true
    width: Math.min(400, parent.width - Theme.spacingXL * 2)
    padding: Theme.spacingL
    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside

    Overlay.modal: Rectangle {
        color: Qt.rgba(0, 0, 0, 0.4)
    }

    background: Rectangle {
        color: Theme.surfaceContainerHigh
        radius: Theme.cornerRadiusLarge
        border.width: 1
        border.color: Theme.outlineMedium
    }

    contentItem: Column {
        spacing: Theme.spacingM

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        StyledText {
            text: root.title
            font.pixelSize: Theme.fontSizeLarge
            font.weight: Font.Medium
            color: Theme.surfaceText
            width: parent.width
            wrapMode: Text.WordWrap
        }

        StyledText {
            visible: text !== ""
            text: root.message
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceVariantText
            width: parent.width
            wrapMode: Text.WordWrap
        }

        Row {
            anchors.right: parent.right
            spacing: Theme.spacingS

            DankButton {
                text: I18n.tr("Cancel", "confirm dialog button to cancel")
                backgroundColor: "transparent"
                textColor: Theme.surfaceText
                onClicked: root.close()
            }

            DankButton {
                text: root.confirmText
                backgroundColor: root.danger ? Theme.error : Theme.primary
                textColor: root.danger ? Theme.surface : Theme.primaryText
                onClicked: {
                    root.close();
                    root.confirmed();
                }
            }
        }
    }
}
