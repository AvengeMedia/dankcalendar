import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: root

    property date displayDate: new Date()
    property date today: new Date()
    property date selectedDate: new Date()
    property int eventsVersion: 0

    signal eventClicked(var event)

    readonly property int startHour: SettingsData.effectiveHourStart
    readonly property int endHour: SettingsData.effectiveHourEnd
    readonly property int hourCount: endHour - startHour
    readonly property real hourHeight: 48
    readonly property real timeColumnWidth: 60
    readonly property real allDayChipHeight: Math.max(18, SettingsData.weekEventTitleLines * 13 + 4)

    readonly property date startOfWeek: {
        const d = new Date(displayDate);
        d.setDate(d.getDate() - ((d.getDay() - SettingsData.effectiveFirstDayOfWeek + 7) % 7));
        d.setHours(0, 0, 0, 0);
        return d;
    }

    SystemClock {
        id: clock
        precision: SystemClock.Minutes
    }

    readonly property int nowColumn: {
        for (let i = 0; i < 7; i++) {
            if (isSameDay(dayAt(i), clock.date))
                return i;
        }
        return -1;
    }
    readonly property real nowHour: clock.date.getHours() + clock.date.getMinutes() / 60

    function hourLabel(hour) {
        if (hour === 0)
            return "";
        if (SettingsData.use24HourTime)
            return (hour < 10 ? "0" + hour : hour) + ":00";
        const h = hour % 12 === 0 ? 12 : hour % 12;
        return hour < 12 ? I18n.tr("%1 AM", "morning hour label in time gutter").arg(h) : I18n.tr("%1 PM", "afternoon hour label in time gutter").arg(h);
    }

    Connections {
        target: DankCalService
        function onEventsUpdated() {
            root.eventsVersion++;
        }
    }

    DankTooltipV2 {
        id: chipTooltip
    }

    function eventTooltip(ev) {
        if (ev.allDay)
            return ev.title + " · " + I18n.tr("All day", "all-day marker in event tooltip") + (ev.calendar ? " · " + ev.calendar : "");
        return ev.title + " · " + SettingsData.formatTime(ev.start) + " – " + SettingsData.formatTime(ev.end) + (ev.calendar ? " · " + ev.calendar : "");
    }

    function dayAt(index) {
        const d = new Date(startOfWeek);
        d.setDate(d.getDate() + index);
        return d;
    }

    function isSameDay(a, b) {
        return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
    }

    function timedEventsFor(day) {
        const dayStart = new Date(day.getFullYear(), day.getMonth(), day.getDate());
        const dayEnd = new Date(dayStart.getTime() + 86400000);
        const out = [];
        const list = DankCalService.eventsForDay(day);
        for (let i = 0; i < list.length; i++) {
            const ev = list[i];
            if (ev.allDay)
                continue;
            const coreStart = dayStart.getTime() + root.startHour * 3600000;
            const coreEnd = dayStart.getTime() + root.endHour * 3600000;
            const s = Math.max(ev.start.getTime(), dayStart.getTime(), coreStart);
            const e = Math.min(ev.end.getTime(), dayEnd.getTime(), coreEnd);
            if (e <= s)
                continue;
            const decorated = Object.assign({}, ev);
            decorated.startHour = (s - dayStart.getTime()) / 3600000 - root.startHour;
            decorated.durationHours = Math.max((e - s) / 3600000, 0.5);
            out.push(decorated);
        }
        return DankCalService.layoutTimedEvents(out);
    }

    function allDayEventsFor(day) {
        return DankCalService.eventsForDay(day).filter(ev => ev.allDay);
    }

    readonly property int allDayMax: {
        eventsVersion;
        let max = 0;
        for (let i = 0; i < 7; i++)
            max = Math.max(max, allDayEventsFor(dayAt(i)).length);
        return Math.min(max, 2);
    }

    function hiddenInfoFor(day) {
        const dayStart = new Date(day.getFullYear(), day.getMonth(), day.getDate());
        const dayEnd = new Date(dayStart.getTime() + 86400000);
        const coreStart = dayStart.getTime() + root.startHour * 3600000;
        const coreEnd = dayStart.getTime() + root.endHour * 3600000;
        const list = DankCalService.eventsForDay(day);
        let count = 0;
        let before = false;
        let after = false;
        for (let i = 0; i < list.length; i++) {
            const ev = list[i];
            if (ev.allDay)
                continue;
            const s0 = Math.max(ev.start.getTime(), dayStart.getTime());
            const e0 = Math.min(ev.end.getTime(), dayEnd.getTime());
            if (e0 <= s0)
                continue;
            if (s0 < coreStart)
                before = true;
            if (e0 > coreEnd)
                after = true;
            if (s0 < coreStart || e0 > coreEnd)
                count++;
        }
        return {
            count: count,
            before: before,
            after: after
        };
    }

    readonly property int hiddenEventTotal: {
        eventsVersion;
        let total = 0;
        for (let i = 0; i < 7; i++)
            total += root.hiddenInfoFor(root.dayAt(i)).count;
        return total;
    }

    readonly property bool anyHiddenBefore: {
        eventsVersion;
        for (let i = 0; i < 7; i++)
            if (root.hiddenInfoFor(root.dayAt(i)).before)
                return true;
        return false;
    }

    Column {
        anchors.fill: parent
        spacing: 0

        Rectangle {
            id: coreHoursWarning
            width: parent.width
            height: visible ? 36 : 0
            visible: root.hiddenEventTotal > 0
            color: Theme.withAlpha(Theme.warning, 0.18)

            Row {
                anchors.left: parent.left
                anchors.leftMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                spacing: Theme.spacingS

                DankIcon {
                    name: "warning"
                    size: Theme.iconSizeSmall
                    color: Theme.warning
                    anchors.verticalCenter: parent.verticalCenter
                }

                StyledText {
                    anchors.verticalCenter: parent.verticalCenter
                    text: root.hiddenEventTotal === 1 ? I18n.tr("1 appointment falls outside core hours", "core hours warning banner, singular") : I18n.tr("%1 appointments fall outside core hours", "core hours warning banner, plural").arg(root.hiddenEventTotal)
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.warning
                }
            }

            DankButton {
                anchors.right: parent.right
                anchors.rightMargin: Theme.spacingM
                anchors.verticalCenter: parent.verticalCenter
                text: I18n.tr("Disable core hours", "core hours warning banner disable action")
                backgroundColor: "transparent"
                textColor: Theme.warning
                buttonHeight: 28
                onClicked: SettingsData.coreHoursEnabled = false
            }
        }

        Row {
            width: parent.width
            height: 56

            Item {
                width: root.timeColumnWidth
                height: parent.height
            }

            Repeater {
                model: 7

                Item {
                    required property int index
                    readonly property date d: root.dayAt(index)
                    readonly property bool isToday: root.isSameDay(d, root.today)
                    readonly property bool isSelected: root.isSameDay(d, root.selectedDate)

                    width: (parent.width - root.timeColumnWidth) / 7
                    height: parent.height

                    Column {
                        anchors.centerIn: parent
                        spacing: 2

                        StyledText {
                            text: SettingsData.dayName(parent.parent.d.getDay())
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.surfaceVariantText
                            anchors.horizontalCenter: parent.horizontalCenter
                        }

                        Rectangle {
                            width: 32
                            height: 32
                            radius: 16
                            anchors.horizontalCenter: parent.horizontalCenter
                            color: parent.parent.isToday ? Theme.primary : "transparent"
                            border.color: Theme.primary
                            border.width: parent.parent.isSelected && !parent.parent.isToday ? 2 : 0

                            StyledText {
                                anchors.centerIn: parent
                                text: parent.parent.parent.d.getDate()
                                font.pixelSize: Theme.fontSizeMedium
                                font.weight: Font.Medium
                                color: parent.parent.parent.isToday ? Theme.primaryText : Theme.surfaceText
                            }
                        }
                    }

                    Rectangle {
                        anchors.bottom: parent.bottom
                        width: parent.width
                        height: 1
                        color: Theme.gridLine
                    }
                }
            }
        }

        Row {
            width: parent.width
            height: root.allDayMax > 0 ? root.allDayMax * (root.allDayChipHeight + 4) + 6 : 0
            visible: root.allDayMax > 0

            Item {
                width: root.timeColumnWidth
                height: parent.height
            }

            Repeater {
                model: 7

                Item {
                    id: allDayCell
                    required property int index
                    readonly property var dayEvents: {
                        root.eventsVersion;
                        return root.allDayEventsFor(root.dayAt(index));
                    }

                    width: (parent.width - root.timeColumnWidth) / 7
                    height: parent.height

                    Column {
                        anchors.fill: parent
                        anchors.margins: 2
                        spacing: 2

                        Repeater {
                            model: ScriptModel {
                                values: allDayCell.dayEvents.slice(0, 2)
                            }

                            Rectangle {
                                required property var modelData
                                width: parent.width
                                height: root.allDayChipHeight
                                radius: 4
                                clip: true
                                color: Theme.withAlpha(modelData.color, 0.22)
                                border.color: modelData.color
                                border.width: 1

                                StyledText {
                                    anchors.left: parent.left
                                    anchors.right: parent.right
                                    anchors.leftMargin: 4
                                    anchors.rightMargin: 4
                                    anchors.verticalCenter: parent.verticalCenter
                                    text: modelData.title
                                    font.pixelSize: 10
                                    color: Theme.surfaceText
                                    wrapMode: Text.WordWrap
                                    maximumLineCount: SettingsData.weekEventTitleLines
                                    elide: Text.ElideRight
                                }

                                MouseArea {
                                    anchors.fill: parent
                                    cursorShape: Qt.PointingHandCursor
                                    hoverEnabled: true
                                    onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                                    onExited: chipTooltip.hide()
                                    onClicked: {
                                        chipTooltip.hide();
                                        root.eventClicked(parent.modelData);
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        Row {
            id: hiddenBeforeStrip
            width: parent.width
            height: root.anyHiddenBefore ? 16 : 0
            visible: root.anyHiddenBefore

            Item {
                width: root.timeColumnWidth
                height: parent.height
            }

            Repeater {
                model: 7

                Item {
                    required property int index
                    readonly property date d: root.dayAt(index)
                    width: (parent.width - root.timeColumnWidth) / 7
                    height: parent.height

                    DankIcon {
                        visible: {
                            root.eventsVersion;
                            return root.hiddenInfoFor(parent.d).before;
                        }
                        anchors.centerIn: parent
                        name: "keyboard_arrow_up"
                        size: Theme.iconSizeSmall
                        color: Theme.warning
                    }
                }
            }
        }

        DankFlickable {
            width: parent.width
            height: parent.height - 56 - coreHoursWarning.height - hiddenBeforeStrip.height - (root.allDayMax > 0 ? root.allDayMax * (root.allDayChipHeight + 4) + 6 : 0)
            contentHeight: root.hourHeight * root.hourCount
            clip: true

            Item {
                width: parent.width
                height: root.hourHeight * root.hourCount

                Item {
                    anchors.left: parent.left
                    width: root.timeColumnWidth
                    height: parent.height

                    Repeater {
                        model: root.hourCount + 1

                        StyledText {
                            required property int index
                            anchors.right: parent.right
                            anchors.rightMargin: Theme.spacingS
                            y: Math.max(0, index * root.hourHeight - 6)
                            text: root.hourLabel(root.startHour + index)
                            font.pixelSize: 11
                            color: Theme.surfaceVariantText
                            isMonospace: true
                        }
                    }
                }

                Column {
                    anchors.right: parent.right
                    width: parent.width - root.timeColumnWidth
                    spacing: 0

                    Repeater {
                        model: root.hourCount

                        Rectangle {
                            width: parent.width
                            height: root.hourHeight
                            color: "transparent"
                            border.color: Theme.outlineLight
                            border.width: 0

                            Rectangle {
                                anchors.top: parent.top
                                width: parent.width
                                height: 1
                                color: Theme.gridLine
                            }
                        }
                    }
                }

                Rectangle {
                    anchors.right: parent.right
                    width: parent.width - root.timeColumnWidth
                    y: root.hourCount * root.hourHeight
                    height: 1
                    color: Theme.gridLine
                }

                Row {
                    anchors.right: parent.right
                    width: parent.width - root.timeColumnWidth
                    height: parent.height

                    Repeater {
                        model: 7

                        Item {
                            id: dayColumn
                            required property int index
                            readonly property var timedEvents: {
                                root.eventsVersion;
                                return root.timedEventsFor(root.dayAt(index));
                            }

                            width: parent.width / 7
                            height: parent.height

                            Rectangle {
                                anchors.left: parent.left
                                width: 1
                                height: parent.height
                                color: Theme.gridLine
                            }

                            DankIcon {
                                visible: {
                                    root.eventsVersion;
                                    return root.hiddenInfoFor(root.dayAt(dayColumn.index)).after;
                                }
                                anchors.horizontalCenter: parent.horizontalCenter
                                y: root.hourCount * root.hourHeight + 2
                                name: "keyboard_arrow_down"
                                size: Theme.iconSizeSmall
                                color: Theme.warning
                            }

                            Repeater {
                                model: ScriptModel {
                                    values: dayColumn.timedEvents
                                }

                                Rectangle {
                                    required property var modelData
                                    readonly property real laneGap: 2
                                    readonly property real usableWidth: parent.width - 8
                                    readonly property real laneWidth: (usableWidth - (modelData.columns - 1) * laneGap) / modelData.columns
                                    x: 4 + modelData.column * (laneWidth + laneGap)
                                    y: modelData.startHour * root.hourHeight
                                    width: laneWidth
                                    height: modelData.durationHours * root.hourHeight - 2
                                    radius: Theme.cornerRadiusSmall
                                    clip: true
                                    color: Theme.withAlpha(modelData.color, 0.22)
                                    border.color: modelData.color
                                    border.width: 1

                                    Column {
                                        anchors.fill: parent
                                        anchors.margins: 4
                                        spacing: 2

                                        StyledText {
                                            text: modelData.title
                                            font.pixelSize: 11
                                            font.weight: Font.Medium
                                            color: Theme.surfaceText
                                            width: parent.width
                                            wrapMode: Text.WordWrap
                                            maximumLineCount: Math.min(SettingsData.weekEventTitleLines, Math.max(1, Math.floor(parent.height / 14)))
                                            elide: Text.ElideRight
                                        }
                                    }

                                    MouseArea {
                                        anchors.fill: parent
                                        cursorShape: Qt.PointingHandCursor
                                        hoverEnabled: true
                                        onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                                        onExited: chipTooltip.hide()
                                        onClicked: {
                                            chipTooltip.hide();
                                            root.eventClicked(parent.modelData);
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                Item {
                    anchors.right: parent.right
                    width: parent.width - root.timeColumnWidth
                    height: parent.height
                    visible: root.nowColumn >= 0 && root.nowHour >= root.startHour && root.nowHour < root.endHour
                    z: 10

                    Item {
                        y: (root.nowHour - root.startHour) * root.hourHeight
                        width: parent.width

                        Rectangle {
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.verticalCenter: parent.top
                            height: 2
                            color: Theme.error
                        }

                        Rectangle {
                            x: root.nowColumn * (parent.width / 7)
                            anchors.verticalCenter: parent.top
                            width: 10
                            height: 10
                            radius: 5
                            color: Theme.error
                        }
                    }
                }
            }
        }
    }
}
