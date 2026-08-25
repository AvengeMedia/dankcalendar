import QtQuick
import QtQuick.Controls
import QtQuick.Window
import qs.Common
import qs.Widgets
import qs.DankCommon.Widgets

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
    activeFocusOnTab: enabled

    Keys.onPressed: event => {
        switch (event.key) {
        case Qt.Key_Space:
        case Qt.Key_Return:
        case Qt.Key_Enter:
        case Qt.Key_Down:
            popup.open();
            event.accepted = true;
            break;
        }
    }

    Rectangle {
        id: field

        anchors.fill: parent
        radius: Theme.cornerRadius
        color: Theme.surfaceContainer
        border.width: popup.visible || root.activeFocus ? 2 : 1
        border.color: popup.visible || root.activeFocus ? Theme.primary : Theme.outlineLight

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
                text: SettingsData.formatDate(root.selectedDate, root.dateFormat)
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
        property date cursorDate: root.selectedDate

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

        function moveCursor(days) {
            const d = new Date(cursorDate);
            d.setDate(d.getDate() + days);
            cursorDate = d;
            if (d.getMonth() !== gridMonth || d.getFullYear() !== gridYear)
                displayDate = d;
        }

        function moveCursorMonths(delta) {
            const d = new Date(cursorDate);
            d.setMonth(d.getMonth() + delta);
            cursorDate = d;
            displayDate = d;
        }

        function selectCursor() {
            root.dateSelected(cursorDate);
            close();
        }

        y: root.openUpwards ? -(height + Theme.spacingXS) : (field.height + Theme.spacingXS)
        width: root.cellSize * 7 + padding * 2
        padding: Theme.spacingS
        onAboutToShow: {
            displayDate = root.selectedDate;
            cursorDate = root.selectedDate;
            root.updateDirection();
        }
        onOpened: calendarContent.forceActiveFocus()
        onClosed: root.forceActiveFocus()

        background: Rectangle {
            color: Theme.surfaceContainerHigh
            radius: Theme.cornerRadius
            border.width: 1
            border.color: Theme.outlineMedium
        }

        contentItem: Column {
            id: calendarContent

            spacing: Theme.spacingXS

            LayoutMirroring.enabled: I18n.isRtl
            LayoutMirroring.childrenInherit: true

            Keys.onPressed: event => {
                switch (event.key) {
                case Qt.Key_Left:
                    popup.moveCursor(I18n.isRtl ? 1 : -1);
                    break;
                case Qt.Key_Right:
                    popup.moveCursor(I18n.isRtl ? -1 : 1);
                    break;
                case Qt.Key_Up:
                    popup.moveCursor(-7);
                    break;
                case Qt.Key_Down:
                    popup.moveCursor(7);
                    break;
                case Qt.Key_PageUp:
                    popup.moveCursorMonths(-1);
                    break;
                case Qt.Key_PageDown:
                    popup.moveCursorMonths(1);
                    break;
                case Qt.Key_Space:
                case Qt.Key_Return:
                case Qt.Key_Enter:
                    popup.selectCursor();
                    break;
                default:
                    return;
                }
                event.accepted = true;
            }

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
                    text: SettingsData.formatDate(popup.displayDate, "MMMM yyyy")
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
                            text: SettingsData.dayName((index + root.firstDayOfWeek) % 7)
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
                        readonly property bool cursorDay: popup.sameDay(cellDay, popup.cursorDate)

                        width: root.cellSize
                        height: root.cellSize - 4
                        radius: height / 2
                        color: selected ? Theme.primary : (cursorDay ? Theme.primaryHover : "transparent")
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
