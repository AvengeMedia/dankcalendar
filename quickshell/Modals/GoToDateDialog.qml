pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import qs.Common
import qs.Services
import qs.Widgets
import qs.DankCommon.Widgets

Popup {
    id: root

    signal dateSelected(date value)

    property date initialDate: new Date()
    property date pickerDate: new Date()
    property var targetDate: null

    readonly property int firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
    readonly property int cellSize: 34
    readonly property int gridYear: pickerDate.getFullYear()
    readonly property int gridMonth: pickerDate.getMonth()
    readonly property int leadingDays: {
        const offset = new Date(gridYear, gridMonth, 1).getDay() - firstDayOfWeek;
        return offset < 0 ? offset + 7 : offset;
    }

    function show(date) {
        initialDate = date;
        pickerDate = new Date(date.getFullYear(), date.getMonth(), 1);
        targetDate = null;
        dateInput.text = "";
        open();
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

    function cellDate(index) {
        return new Date(gridYear, gridMonth, 1 + index - leadingDays);
    }

    function sameDay(a, b) {
        return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
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

    parent: Overlay.overlay
    x: Math.round((parent.width - width) / 2)
    y: Math.round(parent.height * 0.14)
    width: root.cellSize * 7 + Theme.spacingL * 2
    modal: true
    padding: Theme.spacingL
    closePolicy: Popup.CloseOnEscape | Popup.CloseOnPressOutside

    onOpened: dateInput.forceActiveFocus()

    Overlay.modal: Rectangle {
        color: Qt.rgba(0, 0, 0, 0.4)
    }

    background: Rectangle {
        color: Theme.surfaceContainer
        radius: Theme.cornerRadiusLarge
        border.width: 1
        border.color: Theme.outlineMedium
    }

    contentItem: Column {
        spacing: Theme.spacingM

        LayoutMirroring.enabled: I18n.isRtl
        LayoutMirroring.childrenInherit: true

        Row {
            width: parent.width
            spacing: Theme.spacingS

            DankIcon {
                name: "event"
                size: Theme.iconSize - 4
                color: Theme.primary
                anchors.verticalCenter: parent.verticalCenter
            }

            StyledText {
                text: I18n.tr("Go to date", "go to date dialog title")
                font.pixelSize: Theme.fontSizeLarge
                font.weight: Font.Medium
                color: Theme.surfaceText
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        Rectangle {
            width: parent.width
            height: 44
            radius: Theme.cornerRadius
            color: Theme.surfaceContainerHigh
            border.width: 1
            border.color: dateInput.activeFocus ? Theme.primary : Theme.outlineLight

            Behavior on border.color {
                ColorAnimation {
                    duration: Theme.shortDuration
                    easing.type: Theme.standardEasing
                }
            }

            Row {
                anchors.fill: parent
                anchors.leftMargin: Theme.spacingM
                anchors.rightMargin: Theme.spacingM
                spacing: Theme.spacingS

                DankIcon {
                    name: "keyboard"
                    size: Theme.iconSize - 6
                    color: dateInput.activeFocus ? Theme.primary : Theme.surfaceVariantText
                    anchors.verticalCenter: parent.verticalCenter
                }

                TextInput {
                    id: dateInput
                    width: parent.width - Theme.iconSize + 6 - Theme.spacingS
                    anchors.verticalCenter: parent.verticalCenter
                    font.family: Theme.fontFamily
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.surfaceText
                    selectionColor: Theme.primary
                    selectedTextColor: Theme.primaryText
                    clip: true
                    focus: true
                    onTextChanged: {
                        const parsed = root.parseDate(text);
                        if (!parsed) {
                            root.targetDate = null;
                            return;
                        }
                        root.targetDate = parsed;
                        root.pickerDate = new Date(parsed.getFullYear(), parsed.getMonth(), 1);
                    }
                    Keys.onPressed: event => {
                        switch (event.key) {
                        case Qt.Key_Return:
                        case Qt.Key_Enter:
                            root.submitInput();
                            event.accepted = true;
                            return;
                        case Qt.Key_Escape:
                            root.close();
                            event.accepted = true;
                            return;
                        }
                        event.accepted = false;
                    }

                    Text {
                        anchors.fill: parent
                        visible: dateInput.text.length === 0
                        text: I18n.tr("e.g. 2022-01-01", "go to date input placeholder example")
                        color: Theme.surfaceVariantText
                        font.family: Theme.fontFamily
                        font.pixelSize: Theme.fontSizeMedium
                        verticalAlignment: Text.AlignVCenter
                        elide: Text.ElideRight
                    }
                }
            }
        }

        Item {
            width: parent.width
            height: 32

            Row {
                anchors.left: parent.left
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingXS

                DankActionButton {
                    iconName: I18n.isRtl ? "keyboard_double_arrow_right" : "keyboard_double_arrow_left"
                    iconSize: Theme.iconSize - 8
                    iconColor: Theme.surfaceVariantText
                    onClicked: root.shiftYear(-1)
                }

                DankActionButton {
                    iconName: I18n.isRtl ? "chevron_right" : "chevron_left"
                    iconColor: Theme.surfaceVariantText
                    onClicked: root.shiftMonth(-1)
                }
            }

            StyledText {
                anchors.centerIn: parent
                text: SettingsData.monthName(root.gridMonth) + " " + root.gridYear
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.Medium
                color: Theme.surfaceText
            }

            Row {
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingXS

                DankActionButton {
                    iconName: I18n.isRtl ? "chevron_left" : "chevron_right"
                    iconColor: Theme.surfaceVariantText
                    onClicked: root.shiftMonth(1)
                }

                DankActionButton {
                    iconName: I18n.isRtl ? "keyboard_double_arrow_left" : "keyboard_double_arrow_right"
                    iconSize: Theme.iconSize - 8
                    iconColor: Theme.surfaceVariantText
                    onClicked: root.shiftYear(1)
                }
            }
        }

        Row {
            LayoutMirroring.enabled: I18n.isRtl
            LayoutMirroring.childrenInherit: true

            Repeater {
                model: 7

                Item {
                    required property int index

                    width: root.cellSize
                    height: 22

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

            LayoutMirroring.enabled: I18n.isRtl
            LayoutMirroring.childrenInherit: true

            Repeater {
                model: 42

                Rectangle {
                    id: dayCell

                    required property int index
                    readonly property date cellDay: root.cellDate(index)
                    readonly property bool inMonth: cellDay.getMonth() === root.gridMonth
                    readonly property bool target: root.targetDate && root.sameDay(cellDay, root.targetDate)
                    readonly property bool current: root.sameDay(cellDay, root.initialDate)
                    readonly property bool isToday: root.sameDay(cellDay, new Date())

                    width: root.cellSize
                    height: root.cellSize - 4
                    radius: height / 2
                    color: target ? Theme.primary : current ? Theme.primarySelected : "transparent"
                    border.width: isToday && !target ? 1 : 0
                    border.color: Theme.primary

                    StyledText {
                        anchors.centerIn: parent
                        text: dayCell.cellDay.getDate()
                        font.pixelSize: Theme.fontSizeSmall
                        color: dayCell.target ? Theme.primaryText : (dayCell.inMonth ? Theme.surfaceText : Theme.surfaceVariantText)
                    }

                    StateLayer {
                        stateColor: Theme.primary
                        cornerRadius: parent.radius
                        onClicked: root.commit(dayCell.cellDay)
                    }
                }
            }
        }
    }
}
