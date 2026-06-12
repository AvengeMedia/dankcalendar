import QtQuick
import QtQuick.Controls
import qs.Common
import qs.Services
import qs.Widgets

Popup {
    id: root

    property var calendar: null
    readonly property bool hasOverride: !!(calendar && calendar.providerName && calendar.name !== calendar.providerName)

    function show(cal) {
        calendar = cal;
        nameField.text = cal.name || "";
        open();
        nameField.forceActiveFocus();
    }

    function submit() {
        if (!calendar)
            return;
        const trimmed = nameField.text.trim();
        if (trimmed === "")
            return;

        // Storing the provider's own name is the same as clearing the override
        const next = trimmed === calendar.providerName ? "" : trimmed;
        DankCalService.renameCalendar(calendar.id, next);
        close();
    }

    parent: Overlay.overlay
    anchors.centerIn: parent
    modal: true
    width: 400
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

        StyledText {
            text: I18n.tr("Rename calendar", "rename calendar dialog header")
            font.pixelSize: Theme.fontSizeLarge
            font.weight: Font.Medium
            color: Theme.surfaceText
        }

        DankTextField {
            id: nameField
            width: parent.width
            placeholderText: I18n.tr("Calendar name", "rename calendar dialog placeholder for name input")
            onAccepted: root.submit()
        }

        StyledText {
            visible: root.hasOverride
            text: I18n.tr("Synced as \"%1\". The name only changes in Dank Calendar.", "rename calendar dialog note showing provider name").arg(root.calendar ? root.calendar.providerName : "")
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.surfaceVariantText
            width: parent.width
            wrapMode: Text.WordWrap
        }

        Row {
            anchors.right: parent.right
            spacing: Theme.spacingS

            DankButton {
                visible: root.hasOverride
                text: I18n.tr("Use synced name", "rename calendar dialog button to revert to provider name")
                backgroundColor: "transparent"
                textColor: Theme.surfaceVariantText
                onClicked: {
                    DankCalService.renameCalendar(root.calendar.id, "");
                    root.close();
                }
            }

            DankButton {
                text: I18n.tr("Cancel", "rename calendar dialog button to cancel")
                backgroundColor: "transparent"
                textColor: Theme.surfaceText
                onClicked: root.close()
            }

            DankButton {
                text: I18n.tr("Save", "rename calendar dialog button to save name")
                backgroundColor: Theme.primary
                textColor: Theme.primaryText
                onClicked: root.submit()
            }
        }
    }
}
