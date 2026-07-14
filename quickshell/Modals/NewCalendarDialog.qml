import QtQuick
import QtQuick.Controls
import qs.Common
import qs.Services
import qs.Widgets

Popup {
    id: root

    property var account: null
    property string errorText: ""

    function show(acc) {
        account = acc;
        errorText = "";
        nameField.text = "";
        open();
        nameField.forceActiveFocus();
    }

    function submit() {
        if (!account)
            return;
        const trimmed = nameField.text.trim();
        if (trimmed === "") {
            errorText = I18n.tr("Enter a name for the calendar.", "new calendar dialog validation when the name is empty");
            return;
        }
        DankCalService.createCalendar(account.id, trimmed, response => {
            if (response.error) {
                errorText = response.error;
                return;
            }
            root.close();
        });
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
            width: parent.width
            text: I18n.tr("New calendar", "new calendar dialog header")
            font.pixelSize: Theme.fontSizeLarge
            font.weight: Font.Medium
            color: Theme.surfaceText
        }

        StyledText {
            text: root.account ? I18n.tr("Adds a calendar to \"%1\".", "new calendar dialog subtitle naming the local account").arg(DankCalService.accountLabel(root.account)) : ""
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.surfaceVariantText
            width: parent.width
            wrapMode: Text.WordWrap
        }

        DankTextField {
            id: nameField
            width: parent.width
            placeholderText: I18n.tr("Calendar name", "new calendar dialog placeholder for name input")
            onTextChanged: root.errorText = ""
            onAccepted: root.submit()
        }

        StyledText {
            visible: root.errorText !== ""
            text: root.errorText
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.error
            width: parent.width
            wrapMode: Text.WordWrap
        }

        Row {
            anchors.right: parent.right
            spacing: Theme.spacingS

            DankButton {
                text: I18n.tr("Cancel", "new calendar dialog button to cancel")
                backgroundColor: "transparent"
                textColor: Theme.surfaceText
                onClicked: root.close()
            }

            DankButton {
                text: I18n.tr("Create", "new calendar dialog button to create the calendar")
                backgroundColor: Theme.primary
                textColor: Theme.primaryText
                onClicked: root.submit()
            }
        }
    }
}
