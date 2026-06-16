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

    readonly property int hourCount: 24
    readonly property real hourHeight: 48
    readonly property real timeColumnWidth: 60

    readonly property date startOfWeek: {
        const d = new Date(displayDate);
        d.setDate(d.getDate() - ((d.getDay() - SettingsData.effectiveFirstDayOfWeek + 7) % 7));
        d.setHours(0, 0, 0, 0);
        return d;
    }

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
            const s = Math.max(ev.start.getTime(), dayStart.getTime());
            const e = Math.min(ev.end.getTime(), dayEnd.getTime());
            const decorated = Object.assign({}, ev);
            decorated.startHour = (s - dayStart.getTime()) / 3600000;
            decorated.durationHours = Math.max((e - s) / 3600000, 0.5);
            out.push(decorated);
        }
        return out;
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

    Column {
        anchors.fill: parent
        spacing: 0

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
            height: root.allDayMax > 0 ? root.allDayMax * 22 + 6 : 0
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
                                height: 18
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
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
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

        DankFlickable {
            width: parent.width
            height: parent.height - 56 - (root.allDayMax > 0 ? root.allDayMax * 22 + 6 : 0)
            contentHeight: root.hourHeight * root.hourCount
            clip: true

            Item {
                width: parent.width
                height: root.hourHeight * root.hourCount

                Column {
                    width: root.timeColumnWidth
                    spacing: 0

                    Repeater {
                        model: root.hourCount

                        Item {
                            required property int index
                            width: parent.width
                            height: root.hourHeight

                            StyledText {
                                anchors.right: parent.right
                                anchors.rightMargin: Theme.spacingS
                                anchors.top: parent.top
                                anchors.topMargin: -6
                                text: root.hourLabel(index)
                                font.pixelSize: 11
                                color: Theme.surfaceVariantText
                                isMonospace: true
                            }
                        }
                    }
                }

                Column {
                    x: root.timeColumnWidth
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

                Row {
                    x: root.timeColumnWidth
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

                            Repeater {
                                model: ScriptModel {
                                    values: dayColumn.timedEvents
                                }

                                Rectangle {
                                    required property var modelData
                                    x: 4
                                    y: modelData.startHour * root.hourHeight
                                    width: parent.width - 8
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
                                            wrapMode: Text.NoWrap
                                            maximumLineCount: 1
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
            }
        }
    }
}
