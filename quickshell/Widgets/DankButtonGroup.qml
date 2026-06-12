import QtQuick
import Quickshell
import qs.Common

Row {
    id: root

    property var model: []
    property int currentIndex: -1
    property bool checkEnabled: true
    property int buttonHeight: 40
    property int minButtonWidth: 64
    property int buttonPadding: Theme.spacingL
    property int checkIconSize: Theme.iconSizeSmall
    property int textSize: Theme.fontSizeMedium
    property bool userInteracted: false

    signal selectionChanged(int index, bool selected)

    spacing: Theme.spacingXS

    Timer {
        id: animationTimer
        interval: Theme.shortDuration
        onTriggered: root.userInteracted = false
    }

    function selectItem(index) {
        if (index === currentIndex)
            return;
        userInteracted = true;
        selectionChanged(index, true);
        animationTimer.restart();
    }

    Repeater {
        id: repeater
        model: ScriptModel {
            values: root.model
        }

        delegate: Rectangle {
            id: segment

            property bool selected: index === root.currentIndex
            property bool hovered: mouseArea.containsMouse
            property bool pressed: mouseArea.pressed
            property bool isFirst: index === 0
            property bool isLast: index === repeater.count - 1
            property bool visualFirst: I18n.isRtl ? isLast : isFirst
            property bool visualLast: I18n.isRtl ? isFirst : isLast

            width: Math.max(contentRow.implicitWidth + root.buttonPadding * 2, root.minButtonWidth) + (selected ? 4 : 0)
            height: root.buttonHeight

            color: selected ? Theme.buttonBg : Theme.surfaceVariant

            topLeftRadius: (visualFirst || selected) ? Theme.cornerRadius : Math.min(4, Theme.cornerRadius)
            bottomLeftRadius: (visualFirst || selected) ? Theme.cornerRadius : Math.min(4, Theme.cornerRadius)
            topRightRadius: (visualLast || selected) ? Theme.cornerRadius : Math.min(4, Theme.cornerRadius)
            bottomRightRadius: (visualLast || selected) ? Theme.cornerRadius : Math.min(4, Theme.cornerRadius)

            Behavior on width {
                enabled: root.userInteracted
                NumberAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Behavior on topLeftRadius {
                enabled: root.userInteracted
                NumberAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Behavior on topRightRadius {
                enabled: root.userInteracted
                NumberAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Behavior on bottomLeftRadius {
                enabled: root.userInteracted
                NumberAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Behavior on bottomRightRadius {
                enabled: root.userInteracted
                NumberAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Behavior on color {
                enabled: root.userInteracted
                ColorAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Rectangle {
                anchors.fill: parent
                topLeftRadius: parent.topLeftRadius
                bottomLeftRadius: parent.bottomLeftRadius
                topRightRadius: parent.topRightRadius
                bottomRightRadius: parent.bottomRightRadius
                color: {
                    if (segment.pressed)
                        return segment.selected ? Theme.buttonPressed : Theme.surfaceTextHover;
                    if (segment.hovered)
                        return segment.selected ? Theme.buttonHover : Theme.surfaceTextHover;
                    return "transparent";
                }

                Behavior on color {
                    ColorAnimation {
                        duration: Theme.shorterDuration
                        easing.type: Theme.standardEasing
                    }
                }
            }

            Row {
                id: contentRow
                anchors.centerIn: parent
                spacing: Theme.spacingS

                DankIcon {
                    name: "check"
                    size: root.checkIconSize
                    color: segment.selected ? Theme.buttonText : Theme.surfaceVariantText
                    visible: root.checkEnabled && segment.selected
                    opacity: segment.selected ? 1 : 0
                    scale: segment.selected ? 1 : 0.6
                    anchors.verticalCenter: parent.verticalCenter

                    Behavior on opacity {
                        enabled: root.userInteracted
                        NumberAnimation {
                            duration: Theme.shortDuration
                            easing.type: Theme.standardEasing
                        }
                    }

                    Behavior on scale {
                        enabled: root.userInteracted
                        NumberAnimation {
                            duration: Theme.shortDuration
                            easing.type: Theme.emphasizedEasing
                        }
                    }
                }

                StyledText {
                    text: typeof modelData === "string" ? modelData : modelData.text || ""
                    font.pixelSize: root.textSize
                    font.weight: segment.selected ? Font.Medium : Font.Normal
                    color: segment.selected ? Theme.buttonText : Theme.surfaceVariantText
                    anchors.verticalCenter: parent.verticalCenter
                }
            }

            MouseArea {
                id: mouseArea
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: root.selectItem(index)
            }
        }
    }
}
