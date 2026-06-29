import QtQuick
import QtQuick.Controls
import QtQuick.Window
import qs.Common
import qs.Widgets

Item {
    id: root

    property date selectedDate: new Date()
    property int firstDayOfWeek: 0
    property string iconName: "today"
    property string dateFormat: "ddd, MMM d, yyyy"
    property bool openUpwards: false

    signal dateSelected(date value)

    readonly property int cellSize: 36

    // updateDirection flips the calendar above the field when it would clip off
    // the bottom of the window.
    function updateDirection() {
        const winH = Window.height;
        if (winH <= 0) {
            openUpwards = false;
            return;
        }
        const topInWindow = root.mapToItem(null, 0, 0).y;
        const popH = popup.contentItem ? popup.contentItem.implicitHeight + popup.padding * 2 : 0;
        const spaceBelow = winH - (topInWindow + root.height);
        openUpwards = spaceBelow < popH + Theme.spacingXS && topInWindow > spaceBelow;
    }

    height: 48

    Rectangle {
        id: field

        anchors.fill: parent
        radius: Theme.cornerRadius
        color: Theme.surfaceContainer
        border.width: 1
        border.color: popup.visible ? Theme.primary : Theme.outlineLight

        Row {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.leftMargin: Theme.spacingM
            anchors.rightMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS

            DankIcon {
                name: root.iconName
                size: Theme.iconSize - 6
                color: popup.visible ? Theme.primary : Theme.surfaceVariantText
                anchors.verticalCenter: parent.verticalCenter
            }

            StyledText {
                text: Qt.formatDate(root.selectedDate, root.dateFormat)
                font.pixelSize: Theme.fontSizeMedium
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        StateLayer {
            stateColor: Theme.primary
            cornerRadius: parent.radius
            onClicked: popup.visible ? popup.close() : popup.open()
        }
    }

    Popup {
        id: popup

        property date displayDate: root.selectedDate

        readonly property int gridYear: displayDate.getFullYear()
        readonly property int gridMonth: displayDate.getMonth()
        readonly property int leadingDays: {
            const offset = new Date(gridYear, gridMonth, 1).getDay() - root.firstDayOfWeek;
            return offset < 0 ? offset + 7 : offset;
        }

        function cellDate(index) {
            return new Date(gridYear, gridMonth, 1 + index - leadingDays);
        }

        function sameDay(a, b) {
            return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
        }

        y: root.openUpwards ? -(height + Theme.spacingXS) : (field.height + Theme.spacingXS)
        width: root.cellSize * 7 + padding * 2
        padding: Theme.spacingS
        onAboutToShow: {
            displayDate = root.selectedDate;
            root.updateDirection();
        }

        background: Rectangle {
            color: Theme.surfaceContainerHigh
            radius: Theme.cornerRadius
            border.width: 1
            border.color: Theme.outlineMedium
        }

        contentItem: Column {
            spacing: Theme.spacingXS

            LayoutMirroring.enabled: I18n.isRtl
            LayoutMirroring.childrenInherit: true

            Item {
                width: parent.width
                height: 32

                DankActionButton {
                    anchors.left: parent.left
                    anchors.verticalCenter: parent.verticalCenter
                    iconName: I18n.isRtl ? "chevron_right" : "chevron_left"
                    iconColor: Theme.surfaceVariantText
                    onClicked: popup.displayDate = new Date(popup.gridYear, popup.gridMonth - 1, 1)
                }

                StyledText {
                    anchors.centerIn: parent
                    text: Qt.formatDate(popup.displayDate, "MMMM yyyy")
                    font.pixelSize: Theme.fontSizeMedium
                    font.weight: Font.Medium
                }

                DankActionButton {
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    iconName: I18n.isRtl ? "chevron_left" : "chevron_right"
                    iconColor: Theme.surfaceVariantText
                    onClicked: popup.displayDate = new Date(popup.gridYear, popup.gridMonth + 1, 1)
                }
            }

            Row {
                Repeater {
                    model: 7

                    Item {
                        required property int index

                        width: root.cellSize
                        height: 24

                        StyledText {
                            anchors.centerIn: parent
                            text: Qt.locale().dayName((index + root.firstDayOfWeek) % 7, Locale.ShortFormat)
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.surfaceVariantText
                        }
                    }
                }
            }

            Grid {
                columns: 7

                Repeater {
                    model: 42

                    Rectangle {
                        id: dayCell

                        required property int index
                        readonly property date cellDay: popup.cellDate(index)
                        readonly property bool inMonth: cellDay.getMonth() === popup.gridMonth
                        readonly property bool selected: popup.sameDay(cellDay, root.selectedDate)
                        readonly property bool isToday: popup.sameDay(cellDay, new Date())

                        width: root.cellSize
                        height: root.cellSize - 4
                        radius: height / 2
                        color: selected ? Theme.primary : "transparent"
                        border.width: isToday && !selected ? 1 : 0
                        border.color: Theme.primary

                        StyledText {
                            anchors.centerIn: parent
                            text: dayCell.cellDay.getDate()
                            font.pixelSize: Theme.fontSizeSmall
                            color: dayCell.selected ? Theme.primaryText : (dayCell.inMonth ? Theme.surfaceText : Theme.surfaceVariantText)
                        }

                        StateLayer {
                            stateColor: Theme.primary
                            cornerRadius: parent.radius
                            onClicked: {
                                root.dateSelected(dayCell.cellDay);
                                popup.close();
                            }
                        }
                    }
                }
            }
        }
    }
}
