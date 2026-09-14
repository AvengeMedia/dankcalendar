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

    readonly property int cellSize: Theme.iconButtonSize

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

    height: Theme.fieldHeightLarge
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
        radius: Theme.cornerRadiusXS
        color: Theme.surfaceContainerHigh
        border.width: popup.visible || root.activeFocus ? Theme.outlineWidthFocused : Theme.outlineWidth
        border.color: popup.visible || root.activeFocus ? Theme.primary : Theme.outlineVariant

        FocusRing {
            visible: root.activeFocus && !popup.visible
        }

        Row {
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.leftMargin: Theme.spacingM
            anchors.rightMargin: Theme.spacingM
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingS

            DankIcon {
                name: root.iconName
                size: Theme.iconSizeMedium
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

        function focusDay(date) {
            for (const cell of grid.children) {
                if (cell.dayDate === undefined || !grid.sameDay(cell.dayDate, date))
                    continue;
                cell.forceActiveFocus(Qt.TabFocusReason);
                return;
            }
        }

        function focusedDay() {
            for (const cell of grid.children) {
                if (cell.dayDate !== undefined && cell.activeFocus)
                    return cell.dayDate;
            }
            return cursorDate;
        }

        function moveMonths(delta) {
            const d = new Date(focusedDay());
            d.setMonth(d.getMonth() + delta);
            cursorDate = d;
            displayDate = d;
            Qt.callLater(() => focusDay(d));
        }

        function selectDay(date) {
            root.dateSelected(date);
            close();
        }

        y: root.openUpwards ? -(height + Theme.spacingXS) : (field.height + Theme.spacingXS)
        width: root.cellSize * 7 + Theme.spacingXS * 6 + padding * 2
        padding: Theme.spacingS
        onAboutToShow: {
            displayDate = root.selectedDate;
            cursorDate = root.selectedDate;
            root.updateDirection();
        }
        onOpened: focusDay(root.selectedDate)
        onClosed: root.forceActiveFocus()

        background: Rectangle {
            color: Theme.surfaceContainerHigh
            radius: Theme.cornerRadiusM
        }

        contentItem: Column {
            id: calendarContent

            spacing: Theme.spacingXS

            LayoutMirroring.enabled: I18n.isRtl
            LayoutMirroring.childrenInherit: true

            Keys.onPressed: event => {
                switch (event.key) {
                case Qt.Key_PageUp:
                    popup.moveMonths(-1);
                    break;
                case Qt.Key_PageDown:
                    popup.moveMonths(1);
                    break;
                default:
                    return;
                }
                event.accepted = true;
            }

            Item {
                width: parent.width
                height: Theme.buttonHeightXS

                DankActionButton {
                    anchors.left: parent.left
                    anchors.verticalCenter: parent.verticalCenter
                    iconName: I18n.isRtl ? "chevron_right" : "chevron_left"
                    focusPolicy: Qt.NoFocus
                    Accessible.name: I18n.tr("Previous month", "date picker button that shows the previous month")
                    onClicked: popup.moveMonths(-1)
                }

                StyledText {
                    anchors.centerIn: parent
                    text: SettingsData.formatDate(popup.displayDate, "MMMM yyyy")
                    font.pixelSize: Theme.fontSizeMedium
                    font.weight: Theme.fontWeightMedium
                }

                DankActionButton {
                    anchors.right: parent.right
                    anchors.verticalCenter: parent.verticalCenter
                    iconName: I18n.isRtl ? "chevron_left" : "chevron_right"
                    focusPolicy: Qt.NoFocus
                    Accessible.name: I18n.tr("Next month", "date picker button that shows the next month")
                    onClicked: popup.moveMonths(1)
                }
            }

            DankMonthGrid {
                id: grid
                width: parent.width
                height: root.cellSize * 6 + cellGap * 6 + weekdayRowHeight
                displayDate: popup.displayDate
                selectedDate: root.selectedDate
                today: new Date()
                firstDayOfWeek: root.firstDayOfWeek
                dayNames: Array.from({
                    "length": 7
                }, (_, i) => SettingsData.dayName((i + root.firstDayOfWeek) % 7))
                onDayClicked: date => popup.selectDay(date)
            }
        }
    }
}
