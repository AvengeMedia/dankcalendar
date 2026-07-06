import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets

Item {
    id: root

    property date displayDate: new Date()
    property string selectedEventKey: ""
    property int eventsVersion: 0

    signal eventClicked(var event)

    function revealHours(start, duration) {
        const top = start * hourHeight;
        const bottom = top + duration * hourHeight;
        if (top < dayFlickable.contentY) {
            dayFlickable.contentY = Math.max(0, top - hourHeight / 2);
            return;
        }
        if (bottom > dayFlickable.contentY + dayFlickable.height)
            dayFlickable.contentY = Math.max(0, Math.min(dayFlickable.contentHeight - dayFlickable.height, bottom - dayFlickable.height + hourHeight / 2));
    }

    readonly property int startHour: SettingsData.effectiveHourStart
    readonly property int endHour: SettingsData.effectiveHourEnd
    readonly property int hourCount: endHour - startHour
    readonly property real hourHeight: 56
    readonly property real timeColumnWidth: 72

    SystemClock {
        id: clock
        precision: SystemClock.Minutes
    }

    readonly property bool showsNow: clock.date.getFullYear() === displayDate.getFullYear() && clock.date.getMonth() === displayDate.getMonth() && clock.date.getDate() === displayDate.getDate()
    readonly property real nowHour: clock.date.getHours() + clock.date.getMinutes() / 60

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

    function hourLabel(hour) {
        if (hour === 0)
            return "";
        if (SettingsData.use24HourTime)
            return (hour < 10 ? "0" + hour : hour) + ":00";
        const h = hour % 12 === 0 ? 12 : hour % 12;
        return hour < 12 ? I18n.tr("%1 AM", "morning hour label in time gutter").arg(h) : I18n.tr("%1 PM", "afternoon hour label in time gutter").arg(h);
    }

    readonly property var dayEvents: {
        eventsVersion;
        return DankCalService.eventsForDay(displayDate);
    }

    readonly property var allDayEvents: dayEvents.filter(ev => ev.allDay)

    readonly property var timedEvents: {
        const dayStart = new Date(displayDate.getFullYear(), displayDate.getMonth(), displayDate.getDate());
        const dayEnd = new Date(dayStart.getTime() + 86400000);
        const out = [];
        for (let i = 0; i < dayEvents.length; i++) {
            const ev = dayEvents[i];
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

    readonly property var hiddenInfo: {
        eventsVersion;
        const dayStart = new Date(displayDate.getFullYear(), displayDate.getMonth(), displayDate.getDate());
        const dayEnd = new Date(dayStart.getTime() + 86400000);
        const coreStart = dayStart.getTime() + root.startHour * 3600000;
        const coreEnd = dayStart.getTime() + root.endHour * 3600000;
        let count = 0;
        let before = false;
        let after = false;
        for (let i = 0; i < dayEvents.length; i++) {
            const ev = dayEvents[i];
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

    readonly property int hiddenEventCount: hiddenInfo.count

    Rectangle {
        id: coreHoursWarning
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        height: visible ? 36 : 0
        visible: root.hiddenEventCount > 0
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
                text: root.hiddenEventCount === 1 ? I18n.tr("1 appointment falls outside core hours", "core hours warning banner, singular") : I18n.tr("%1 appointments fall outside core hours", "core hours warning banner, plural").arg(root.hiddenEventCount)
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

    Column {
        id: allDayStrip
        anchors.top: coreHoursWarning.visible ? coreHoursWarning.bottom : parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.leftMargin: root.timeColumnWidth
        spacing: 2
        visible: root.allDayEvents.length > 0

        Repeater {
            model: ScriptModel {
                values: root.allDayEvents
            }

            Rectangle {
                required property var modelData
                readonly property bool isSelected: root.selectedEventKey === DankCalService.eventKey(modelData)
                width: parent.width
                height: 22
                radius: 4
                clip: true
                color: Theme.withAlpha(modelData.color, 0.22)
                border.color: isSelected ? Theme.primary : modelData.color
                border.width: isSelected ? 2 : 1

                StyledText {
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.leftMargin: Theme.spacingS
                    anchors.rightMargin: Theme.spacingS
                    anchors.verticalCenter: parent.verticalCenter
                    text: modelData.title + "  ·  " + I18n.tr("all day", "suffix on all-day event chip in day view")
                    font.pixelSize: Theme.fontSizeSmall
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

    Item {
        id: hiddenBeforeStrip
        anchors.top: allDayStrip.visible ? allDayStrip.bottom : (coreHoursWarning.visible ? coreHoursWarning.bottom : parent.top)
        anchors.topMargin: allDayStrip.visible ? Theme.spacingS : (coreHoursWarning.visible ? Theme.spacingS : 0)
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.leftMargin: root.timeColumnWidth
        height: root.hiddenInfo.before ? 16 : 0
        visible: root.hiddenInfo.before

        DankIcon {
            anchors.centerIn: parent
            name: "keyboard_arrow_up"
            size: Theme.iconSizeSmall
            color: Theme.warning
        }
    }

    DankFlickable {
        id: dayFlickable
        anchors.top: hiddenBeforeStrip.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
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

            Item {
                anchors.right: parent.right
                width: parent.width - root.timeColumnWidth
                height: parent.height

                Repeater {
                    model: root.hourCount + 1

                    Rectangle {
                        required property int index
                        y: index * root.hourHeight
                        width: parent.width
                        height: 1
                        color: Theme.gridLine
                    }
                }

                DankIcon {
                    visible: root.hiddenInfo.after
                    anchors.horizontalCenter: parent.horizontalCenter
                    y: root.hourCount * root.hourHeight + 2
                    name: "keyboard_arrow_down"
                    size: Theme.iconSizeSmall
                    color: Theme.warning
                }

                Item {
                    visible: root.showsNow && root.nowHour >= root.startHour && root.nowHour < root.endHour
                    y: (root.nowHour - root.startHour) * root.hourHeight
                    width: parent.width
                    z: 10

                    Rectangle {
                        id: nowDot
                        anchors.left: parent.left
                        anchors.verticalCenter: parent.top
                        width: 10
                        height: 10
                        radius: 5
                        color: Theme.error
                    }

                    Rectangle {
                        anchors.left: nowDot.right
                        anchors.right: parent.right
                        anchors.verticalCenter: parent.top
                        height: 2
                        color: Theme.error
                    }
                }

                Repeater {
                    model: ScriptModel {
                        values: root.timedEvents
                    }

                    Rectangle {
                        required property var modelData
                        readonly property bool isSelected: root.selectedEventKey === DankCalService.eventKey(modelData)
                        onIsSelectedChanged: {
                            if (isSelected)
                                root.revealHours(modelData.startHour, modelData.durationHours);
                        }
                        readonly property real laneGap: 3
                        readonly property real usableWidth: parent.width - 16
                        readonly property real laneWidth: (usableWidth - (modelData.columns - 1) * laneGap) / modelData.columns
                        x: 8 + modelData.column * (laneWidth + laneGap)
                        y: modelData.startHour * root.hourHeight
                        width: laneWidth
                        height: modelData.durationHours * root.hourHeight - 4
                        radius: Theme.cornerRadiusSmall
                        clip: true
                        color: Theme.withAlpha(modelData.color, 0.22)
                        border.color: isSelected ? Theme.primary : modelData.color
                        border.width: isSelected ? 2 : 1

                        Row {
                            anchors.fill: parent
                            anchors.margins: Theme.spacingS
                            spacing: Theme.spacingS

                            Rectangle {
                                width: 3
                                height: parent.height - 4
                                anchors.verticalCenter: parent.verticalCenter
                                radius: 1.5
                                color: parent.parent.modelData.color
                            }

                            Column {
                                anchors.verticalCenter: parent.verticalCenter
                                width: parent.width - 12

                                StyledText {
                                    text: parent.parent.parent.modelData.title
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.Medium
                                    color: Theme.surfaceText
                                    width: parent.width
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
                                    elide: Text.ElideRight
                                }

                                StyledText {
                                    visible: parent.parent.parent.modelData.durationHours >= 0.75
                                    text: parent.parent.parent.modelData.calendar
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    width: parent.width
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
                                    elide: Text.ElideRight
                                }
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
