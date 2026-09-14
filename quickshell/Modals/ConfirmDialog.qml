import QtQuick
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

DankOverlayDialog {
    id: root

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

    function confirm() {
        confirmed();
        close();
    }

    supportingText: message
    onAccepted: confirm()

    actions: [
        DankButton {
            text: I18n.tr("Cancel", "confirm dialog button to cancel")
            backgroundColor: Theme.secondaryContainer
            textColor: Theme.onSecondaryContainer
            onClicked: root.close()
        },
        DankButton {
            text: root.confirmText
            backgroundColor: root.danger ? Theme.errorContainer : Theme.primary
            textColor: root.danger ? Theme.onErrorContainer : Theme.primaryText
            onClicked: root.confirm()
        }
    ]
}
