import QtQuick
import QtQuick.Controls
import qs.Common
import qs.DankCommon.Widgets

Popup {
    id: root

    property bool takesFocus: true
    property alias title: dialog.title
    property alias supportingText: dialog.supportingText
    property alias maximumWidth: dialog.maximumWidth
    property alias acceptEnabled: dialog.acceptEnabled
    property alias closeEnabled: dialog.closeEnabled
    default property alias content: dialog.content
    property alias actions: dialog.actions

    signal accepted
    signal rejected

    parent: Overlay.overlay
    width: parent?.width ?? 0
    height: parent?.height ?? 0
    padding: 0
    modal: true
    dim: false
    focus: takesFocus
    closePolicy: Popup.NoAutoClose
    background: null

    onAboutToShow: dialog.opened = true
    onAboutToHide: dialog.opened = false
    onRejected: close()

    exit: Transition {
        PauseAnimation {
            duration: Theme.animationsEnabled ? Theme.expressiveDurations.expressiveEffects : 0
        }
    }

    contentItem: DankDialog {
        id: dialog
        embedded: false
        opened: false
        focus: root.takesFocus
        onAccepted: root.accepted()
        onRejected: root.rejected()

        windowControls: QtObject {
            readonly property bool canMinimize: false
            readonly property bool canMaximize: false
            readonly property var targetWindow: null

            function tryStartMove() {
            }

            function tryToggleMaximize() {
            }

            function tryMinimize() {
            }
        }
    }
}
