import QtQuick
import qs.Common
import qs.Services
import qs.Widgets

// Toast renders ToastService's current message as a bottom-centered snackbar
// within whatever window mounts it. Mount it last (high z) with anchors.fill so
// it overlays the window's content.
Item {
    id: root

    z: 1000
    visible: bar.opacity > 0

    Rectangle {
        id: bar

        readonly property bool showing: ToastService.active

        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: Theme.spacingL + (showing ? 0 : Theme.spacingS)
        height: 48
        width: Math.min(parent.width - Theme.spacingL * 2, contentRow.implicitWidth + Theme.spacingL * 2)
        radius: Theme.cornerRadius
        color: Theme.surfaceContainerHighest
        border.width: 1
        border.color: Theme.outlineMedium
        opacity: showing ? 1 : 0

        Behavior on opacity {
            NumberAnimation {
                duration: Theme.shortDuration
                easing.type: Theme.standardEasing
            }
        }
        Behavior on anchors.bottomMargin {
            NumberAnimation {
                duration: Theme.shortDuration
                easing.type: Theme.standardEasing
            }
        }

        Row {
            id: contentRow
            anchors.centerIn: parent
            spacing: Theme.spacingM

            StyledText {
                text: ToastService.message
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.surfaceText
                wrapMode: Text.NoWrap
                maximumLineCount: 1
                elide: Text.ElideRight
                anchors.verticalCenter: parent.verticalCenter
            }

            DankButton {
                visible: ToastService.actionLabel !== ""
                text: ToastService.actionLabel
                buttonHeight: 32
                backgroundColor: "transparent"
                textColor: Theme.primary
                anchors.verticalCenter: parent.verticalCenter
                onClicked: ToastService.runAction()
            }
        }
    }
}
