pragma ComponentBehavior: Bound

import QtQuick
import qs.Common
import qs.Services
import qs.Widgets
import qs.DankCommon.Widgets

DankOverlayDialog {
    id: root

    signal dateSelected(date value)

    property date initialDate: new Date()
    property date pickerDate: new Date()
    property var targetDate: null

    readonly property int firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
    readonly property int gridYear: pickerDate.getFullYear()
    readonly property int gridMonth: pickerDate.getMonth()
    readonly property var dayNames: Array.from({
        "length": 7
    }, (_, i) => SettingsData.dayName((i + firstDayOfWeek) % 7))

    function show(date) {
        initialDate = date;
        pickerDate = new Date(date.getFullYear(), date.getMonth(), 1);
        targetDate = null;
        dateInput.text = "";
        open();
        dateInput.forceActiveFocus();
    }

    function validDate(year, month, day) {
        const d = new Date(year, month, day);
        if (d.getFullYear() !== year || d.getMonth() !== month || d.getDate() !== day)
            return null;
        return d;
    }

    function parseDate(text) {
        const s = text.trim();
        if (s.length === 0)
            return null;

        const ymd = s.match(/^(\d{4})[-/.](\d{1,2})(?:[-/.](\d{1,2}))?$/);
        if (ymd)
            return validDate(parseInt(ymd[1], 10), parseInt(ymd[2], 10) - 1, ymd[3] ? parseInt(ymd[3], 10) : 1);

        if (/^\d{4}$/.test(s))
            return validDate(parseInt(s, 10), 0, 1);

        const parsedDate = new Date(s);
        if (!isNaN(parsedDate.getTime()))
            return validDate(parsedDate.getFullYear(), parsedDate.getMonth(), parsedDate.getDate());

        return null;
    }

    function shiftMonth(delta) {
        pickerDate = new Date(gridYear, gridMonth + delta, 1);
        dateInput.text = "";
        targetDate = null;
    }

    function shiftYear(delta) {
        pickerDate = new Date(gridYear + delta, gridMonth, 1);
        dateInput.text = "";
        targetDate = null;
    }

    function commit(date) {
        dateSelected(date);
        close();
    }

    function submitInput() {
        const parsed = parseDate(dateInput.text);
        if (!parsed)
            return;
        commit(parsed);
    }

    maximumWidth: Theme.clockFaceSize + Theme.spacingXL * 3
    title: I18n.tr("Go to date", "go to date dialog title")
    iconName: "event"
    onAccepted: submitInput()

    DankTextField {
        id: dateInput
        width: parent.width
        outlined: true
        leftIconName: "keyboard"
        labelText: I18n.tr("Date", "go to date input label")
        placeholderText: I18n.tr("e.g. 2022-01-01", "go to date input placeholder example")
        onTextChanged: {
            const parsed = root.parseDate(text);
            if (!parsed) {
                root.targetDate = null;
                return;
            }
            root.targetDate = parsed;
            root.pickerDate = new Date(parsed.getFullYear(), parsed.getMonth(), 1);
        }
        onAccepted: root.submitInput()
        Keys.onReturnPressed: event => event.accepted = true
        Keys.onEnterPressed: event => event.accepted = true
    }

    Item {
        width: parent.width
        height: Theme.buttonHeightXS

        Row {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingXS

            DankActionButton {
                iconName: I18n.isRtl ? "keyboard_double_arrow_right" : "keyboard_double_arrow_left"
                iconSize: Theme.iconSizeSmall
                Accessible.name: I18n.tr("Previous year", "keyboard shortcut description")
                onClicked: root.shiftYear(-1)
            }

            DankActionButton {
                iconName: I18n.isRtl ? "chevron_right" : "chevron_left"
                Accessible.name: I18n.tr("Previous day, week or month", "keyboard shortcut description")
                onClicked: root.shiftMonth(-1)
            }
        }

        StyledText {
            anchors.centerIn: parent
            text: SettingsData.monthName(root.gridMonth) + " " + root.gridYear
            font.pixelSize: Theme.fontSizeMedium
            font.weight: Theme.fontWeightMedium
            color: Theme.surfaceText
        }

        Row {
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingXS

            DankActionButton {
                iconName: I18n.isRtl ? "chevron_left" : "chevron_right"
                Accessible.name: I18n.tr("Next day, week or month", "keyboard shortcut description")
                onClicked: root.shiftMonth(1)
            }

            DankActionButton {
                iconName: I18n.isRtl ? "keyboard_double_arrow_left" : "keyboard_double_arrow_right"
                iconSize: Theme.iconSizeSmall
                Accessible.name: I18n.tr("Next year", "keyboard shortcut description")
                onClicked: root.shiftYear(1)
            }
        }
    }

    DankMonthGrid {
        id: grid
        width: parent.width
        height: weekdayRowHeight + (Theme.buttonHeightS + cellGap) * rows
        displayDate: root.pickerDate
        selectedDate: root.targetDate ?? root.initialDate
        today: new Date()
        firstDayOfWeek: root.firstDayOfWeek
        dayNames: root.dayNames
        onDayClicked: date => root.commit(date)
    }
}
