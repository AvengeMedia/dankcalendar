import QtQuick
import QtQuick.Controls
import QtQuick.Window
import Quickshell
import qs.Common
import qs.Widgets

Item {
    id: root

    property var options: []
    property string currentValue: ""
    property int dropdownWidth: 180
    property int maxPopupHeight: 320
    property bool openUpwards: false

    signal valueChanged(string value)

    // updateDirection flips the popup above the button when there isn't room
    // below it within the window, so it never clips off the bottom edge.
    function updateDirection() {
        const winH = Window.height;
        if (winH <= 0) {
            openUpwards = false;
            return;
        }
        const topInWindow = root.mapToItem(null, 0, 0).y;
        const needed = Math.min(optionsColumn.implicitHeight + popup.padding * 2, root.maxPopupHeight + popup.padding * 2) + Theme.spacingXS;
        const spaceBelow = winH - (topInWindow + root.height);
        openUpwards = spaceBelow < needed && topInWindow > spaceBelow;
    }

    width: dropdownWidth
    height: 40

    Rectangle {
        id: button
        anchors.fill: parent
        radius: Theme.cornerRadius
        color: Theme.surfaceContainerHigh
        border.width: popup.visible ? 2 : 1
        border.color: popup.visible ? Theme.primary : Theme.outlineButton

        StyledText {
            anchors.left: parent.left
            anchors.right: arrow.left
            anchors.leftMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            text: root.currentValue
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceText
            wrapMode: Text.NoWrap
            elide: Text.ElideRight
        }

        DankIcon {
            id: arrow
            anchors.right: parent.right
            anchors.rightMargin: Theme.spacingS
            anchors.verticalCenter: parent.verticalCenter
            name: popup.visible ? "expand_less" : "expand_more"
            size: Theme.iconSize - 4
            color: Theme.surfaceVariantText
        }

        StateLayer {
            stateColor: Theme.primary
            cornerRadius: parent.radius
            onClicked: popup.visible ? popup.close() : popup.open()
        }
    }

    Popup {
        id: popup
        y: root.openUpwards ? -(height + Theme.spacingXS) : (button.height + Theme.spacingXS)
        width: root.width
        padding: Theme.spacingXS

        onAboutToShow: root.updateDirection()

        background: Rectangle {
            color: Theme.surfaceContainerHigh
            radius: Theme.cornerRadius
            border.width: 1
            border.color: Theme.outlineMedium
        }

        contentItem: DankFlickable {
            implicitHeight: Math.min(optionsColumn.implicitHeight, root.maxPopupHeight)
            contentHeight: optionsColumn.implicitHeight
            clip: true

            Column {
                id: optionsColumn
                width: parent.width
                spacing: 1

                LayoutMirroring.enabled: I18n.isRtl
                LayoutMirroring.childrenInherit: true

                Repeater {
                    model: ScriptModel {
                        values: root.options
                    }

                    Rectangle {
                        required property string modelData
                        readonly property bool active: modelData === root.currentValue

                        width: parent.width
                        height: 36
                        radius: Theme.cornerRadiusSmall
                        color: active ? Theme.primaryHover : "transparent"

                        StyledText {
                            anchors.left: parent.left
                            anchors.right: check.visible ? check.left : parent.right
                            anchors.leftMargin: Theme.spacingM
                            anchors.verticalCenter: parent.verticalCenter
                            text: parent.modelData
                            font.pixelSize: Theme.fontSizeMedium
                            color: parent.active ? Theme.primary : Theme.surfaceText
                            wrapMode: Text.NoWrap
                            elide: Text.ElideRight
                        }

                        DankIcon {
                            id: check
                            visible: parent.active
                            anchors.right: parent.right
                            anchors.rightMargin: Theme.spacingS
                            anchors.verticalCenter: parent.verticalCenter
                            name: "check"
                            size: Theme.iconSizeSmall
                            color: Theme.primary
                        }

                        StateLayer {
                            stateColor: Theme.primary
                            cornerRadius: parent.radius
                            // Close before emitting: a handler may rebuild the
                            // model that owns this delegate, destroying it
                            // mid-signal.
                            onClicked: {
                                const value = parent.modelData;
                                popup.close();
                                root.valueChanged(value);
                            }
                        }
                    }
                }
            }
        }
    }
}
