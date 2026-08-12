import QtQuick
import Quickshell
import qs.Common
import qs.Services
import qs.Widgets
import qs.DankCommon.Widgets

Item {
    id: root

    property date displayDate: new Date()
    property date today: new Date()
    property date selectedDate: new Date()
    property string selectedEventKey: ""
    property var selectedEventKeys: []
    property int eventsVersion: 0
    property bool eventPointerDown: false
    property bool eventDragging: false
    property var draggedEvent: null
    property real dragTargetTime: 0
    property point dragPosition: Qt.point(0, 0)

    signal eventClicked(var event, int modifiers)
    signal eventContextRequested(var event, var anchorItem, real x, real y)
    signal dayContextRequested(date day, var anchorItem, real x, real y)
    signal eventDropRequested(var event, date targetDay)
    signal shiftDaysRequested(int days)

    function isEventSelected(event) {
        const key = DankCalService.eventKey(event);
        return selectedEventKey === key || selectedEventKeys.indexOf(key) !== -1;
    }

    function updateEventDrag(pointerItem, x, y) {
        const position = pointerItem.mapToItem(root, x, y);
        dragPosition = Qt.point(position.x, position.y);
        if (position.x < timeColumnWidth || position.x >= width) {
            dragTargetTime = 0;
            return;
        }
        const visualIndex = Math.floor((position.x - timeColumnWidth - slidePx) / dayWidth);
        const dayIndex = I18n.isRtl ? 6 - visualIndex : visualIndex;
        dragTargetTime = dayAt(dayIndex).getTime();
    }

    function finishEventDrag(event) {
        const targetTime = dragTargetTime;
        eventPointerDown = false;
        eventDragging = false;
        draggedEvent = null;
        dragTargetTime = 0;
        if (targetTime > 0)
            eventDropRequested(event, new Date(targetTime));
    }

    function revealHours(start, duration) {
        const top = start * hourHeight;
        const bottom = top + duration * hourHeight;
        if (top < weekFlickable.contentY) {
            weekFlickable.contentY = Math.max(0, top - hourHeight / 2);
            return;
        }
        if (bottom > weekFlickable.contentY + weekFlickable.height)
            weekFlickable.contentY = Math.max(0, Math.min(weekFlickable.contentHeight - weekFlickable.height, bottom - weekFlickable.height + hourHeight / 2));
    }

    readonly property int startHour: SettingsData.effectiveHourStart
    readonly property int endHour: SettingsData.effectiveHourEnd
    readonly property int hourCount: endHour - startHour
    readonly property real hourHeight: 48
    readonly property real timeColumnWidth: 60
    readonly property real allDayChipHeight: Math.max(18, SettingsData.weekEventTitleLines * 13 + 4)

    readonly property date firstDay: {
        const d = new Date(displayDate);
        d.setHours(0, 0, 0, 0);
        return d;
    }

    readonly property real dayWidth: width > timeColumnWidth ? (width - timeColumnWidth) / 7 : 0
    property real slidePx: 0

    function applySlide(dx) {
        if (dayWidth <= 0)
            return;
        snapAnim.stop();
        slidePx += dx;
        while (slidePx >= dayWidth) {
            slidePx -= dayWidth;
            shiftDaysRequested(I18n.isRtl ? 1 : -1);
        }
        while (slidePx <= -dayWidth) {
            slidePx += dayWidth;
            shiftDaysRequested(I18n.isRtl ? -1 : 1);
        }
    }

    function settleSlide() {
        if (dayWidth <= 0)
            return;
        if (slidePx >= dayWidth / 2) {
            slidePx -= dayWidth;
            shiftDaysRequested(I18n.isRtl ? 1 : -1);
        } else if (slidePx <= -dayWidth / 2) {
            slidePx += dayWidth;
            shiftDaysRequested(I18n.isRtl ? -1 : 1);
        }
        snapAnim.restart();
    }

    function slideDays(days) {
        if (dayWidth <= 0) {
            shiftDaysRequested(days);
            return;
        }
        snapAnim.stop();
        shiftDaysRequested(days);
        slidePx += (I18n.isRtl ? -1 : 1) * days * dayWidth;
        snapAnim.restart();
    }

    readonly property int nowIndex: {
        const nowDay = new Date(today.getFullYear(), today.getMonth(), today.getDate());
        return Math.round((nowDay.getTime() - firstDay.getTime()) / 86400000);
    }
    readonly property real nowHour: today.getHours() + today.getMinutes() / 60

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
        const suffix = (ev.location ? " · " + ev.location : "") + (ev.calendar ? " · " + ev.calendar : "");
        if (ev.allDay)
            return ev.title + " · " + I18n.tr("All day", "all-day marker in event tooltip") + suffix;
        return ev.title + " · " + SettingsData.formatTime(ev.start) + " – " + SettingsData.formatTime(ev.end) + suffix;
    }

    function dayAt(index) {
        const d = new Date(firstDay);
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
        for (let i = -1; i <= 7; i++)
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
        for (let i = -1; i <= 7; i++)
            total += root.hiddenInfoFor(root.dayAt(i)).count;
        return total;
    }

    readonly property bool anyHiddenBefore: {
        eventsVersion;
        for (let i = -1; i <= 7; i++)
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
            height: root.hiddenEventTotal > 0 ? 36 : 0
            visible: height > 0
            clip: true
            color: Theme.withAlpha(Theme.warning, 0.18)

            Behavior on height {
                NumberAnimation {
                    duration: Theme.shorterDuration
                    easing.type: Theme.standardEasing
                }
            }

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

            Item {
                width: parent.width - root.timeColumnWidth
                height: parent.height
                clip: true

                Row {
                    x: root.slidePx - root.dayWidth
                    height: parent.height

                    Repeater {
                        model: 9

                        Item {
                            required property int index
                            readonly property date d: root.dayAt(index - 1)
                            readonly property bool isToday: root.isSameDay(d, root.today)
                            readonly property bool isSelected: root.isSameDay(d, root.selectedDate)

                            width: root.dayWidth
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

                            TapHandler {
                                acceptedButtons: Qt.RightButton
                                onTapped: eventPoint => root.dayContextRequested(parent.d, parent, eventPoint.position.x, eventPoint.position.y)
                            }
                        }
                    }
                }
            }
        }

        Row {
            id: allDayRow
            width: parent.width
            height: Math.max(1, root.allDayMax) * (root.allDayChipHeight + 4) + 6
            clip: true

            Behavior on height {
                NumberAnimation {
                    duration: Theme.shorterDuration
                    easing.type: Theme.standardEasing
                }
            }

            Item {
                width: root.timeColumnWidth
                height: parent.height
            }

            Item {
                width: parent.width - root.timeColumnWidth
                height: parent.height
                clip: true

                Row {
                    x: root.slidePx - root.dayWidth
                    height: parent.height

                    Repeater {
                        model: 9

                        Item {
                            id: allDayCell
                            required property int index
                            readonly property date day: root.dayAt(index - 1)
                            readonly property bool isDropTarget: root.eventDragging && day.getTime() === root.dragTargetTime
                            readonly property var dayEvents: {
                                root.eventsVersion;
                                return root.allDayEventsFor(root.dayAt(index - 1));
                            }

                            width: root.dayWidth
                            height: parent.height

                            Rectangle {
                                visible: allDayCell.isDropTarget
                                anchors.fill: parent
                                anchors.margins: 1
                                color: Theme.withAlpha(Theme.primary, 0.12)
                                border.color: Theme.primary
                                border.width: 2
                                radius: Theme.cornerRadiusSmall
                            }

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
                                        readonly property bool isSelected: root.isEventSelected(modelData)
                                        width: parent.width
                                        height: root.allDayChipHeight
                                        radius: 4
                                        clip: true
                                        color: Theme.withAlpha(modelData.color, 0.22)
                                        border.color: isSelected ? Theme.primary : modelData.color
                                        border.width: isSelected ? 2 : 1

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

                                        EventMouseArea {
                                            anchors.fill: parent
                                            eventData: parent.modelData
                                            dragEnabled: !parent.modelData.readOnly
                                            onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                                            onExited: chipTooltip.hide()
                                            onActivated: (event, modifiers) => {
                                                chipTooltip.hide();
                                                root.eventClicked(event, modifiers);
                                            }
                                            onContextRequested: (event, anchorItem, x, y) => root.eventContextRequested(event, anchorItem, x, y)
                                            onDragPressed: root.eventPointerDown = true
                                            onDragStarted: (event, pointerItem, x, y) => {
                                                chipTooltip.hide();
                                                root.draggedEvent = event;
                                                root.eventDragging = true;
                                                root.updateEventDrag(pointerItem, x, y);
                                            }
                                            onDragMoved: (event, pointerItem, x, y) => root.updateEventDrag(pointerItem, x, y)
                                            onDropped: (event, pointerItem, x, y) => {
                                                root.updateEventDrag(pointerItem, x, y);
                                                root.finishEventDrag(event);
                                            }
                                            onDragReleased: {
                                                root.eventPointerDown = false;
                                                if (root.eventDragging)
                                                    root.finishEventDrag(parent.modelData);
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

        Row {
            id: hiddenBeforeStrip
            width: parent.width
            height: root.anyHiddenBefore ? 16 : 0
            visible: height > 0
            clip: true

            Behavior on height {
                NumberAnimation {
                    duration: Theme.shorterDuration
                    easing.type: Theme.standardEasing
                }
            }

            Item {
                width: root.timeColumnWidth
                height: parent.height
            }

            Item {
                width: parent.width - root.timeColumnWidth
                height: parent.height
                clip: true

                Row {
                    x: root.slidePx - root.dayWidth
                    height: parent.height

                    Repeater {
                        model: 9

                        Item {
                            required property int index
                            readonly property date d: root.dayAt(index - 1)
                            width: root.dayWidth
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
            }
        }

        DankFlickable {
            id: weekFlickable
            width: parent.width
            height: parent.height - 56 - coreHoursWarning.height - hiddenBeforeStrip.height - allDayRow.height
            contentHeight: root.hourHeight * root.hourCount
            clip: true

            DankSlideDragHandler {
                slideArea: slidePager
            }

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

                Item {
                    anchors.right: parent.right
                    width: parent.width - root.timeColumnWidth
                    height: parent.height
                    clip: true

                    Row {
                        x: root.slidePx - root.dayWidth
                        height: parent.height

                        Repeater {
                            model: 9

                            Item {
                                id: dayColumn
                                required property int index
                                readonly property date day: root.dayAt(index - 1)
                                readonly property bool isDropTarget: root.eventDragging && day.getTime() === root.dragTargetTime
                                readonly property var timedEvents: {
                                    root.eventsVersion;
                                    return root.timedEventsFor(root.dayAt(index - 1));
                                }

                                width: root.dayWidth
                                height: parent.height

                                Rectangle {
                                    visible: dayColumn.isDropTarget
                                    anchors.fill: parent
                                    anchors.margins: 1
                                    color: Theme.withAlpha(Theme.primary, 0.1)
                                    border.color: Theme.primary
                                    border.width: 2
                                    radius: Theme.cornerRadiusSmall
                                }

                                Rectangle {
                                    anchors.left: parent.left
                                    width: 1
                                    height: parent.height
                                    color: Theme.gridLine
                                }

                                DankIcon {
                                    visible: {
                                        root.eventsVersion;
                                        return root.hiddenInfoFor(root.dayAt(dayColumn.index - 1)).after;
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
                                        readonly property bool isSelected: root.isEventSelected(modelData)
                                        onIsSelectedChanged: {
                                            if (isSelected)
                                                root.revealHours(modelData.startHour, modelData.durationHours);
                                        }
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
                                        border.color: isSelected ? Theme.primary : modelData.color
                                        border.width: isSelected ? 2 : 1

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

                                            StyledText {
                                                visible: text !== "" && modelData.durationHours >= 1
                                                text: modelData.location
                                                font.pixelSize: 10
                                                color: Theme.surfaceVariantText
                                                width: parent.width
                                                wrapMode: Text.NoWrap
                                                maximumLineCount: 1
                                                elide: Text.ElideRight
                                            }
                                        }

                                        EventMouseArea {
                                            anchors.fill: parent
                                            eventData: parent.modelData
                                            dragEnabled: !parent.modelData.readOnly
                                            onEntered: chipTooltip.show(root.eventTooltip(parent.modelData), parent)
                                            onExited: chipTooltip.hide()
                                            onActivated: (event, modifiers) => {
                                                chipTooltip.hide();
                                                root.eventClicked(event, modifiers);
                                            }
                                            onContextRequested: (event, anchorItem, x, y) => root.eventContextRequested(event, anchorItem, x, y)
                                            onDragPressed: root.eventPointerDown = true
                                            onDragStarted: (event, pointerItem, x, y) => {
                                                chipTooltip.hide();
                                                root.draggedEvent = event;
                                                root.eventDragging = true;
                                                root.updateEventDrag(pointerItem, x, y);
                                            }
                                            onDragMoved: (event, pointerItem, x, y) => root.updateEventDrag(pointerItem, x, y)
                                            onDropped: (event, pointerItem, x, y) => {
                                                root.updateEventDrag(pointerItem, x, y);
                                                root.finishEventDrag(event);
                                            }
                                            onDragReleased: {
                                                root.eventPointerDown = false;
                                                if (root.eventDragging)
                                                    root.finishEventDrag(parent.modelData);
                                            }
                                        }
                                    }
                                }

                                TapHandler {
                                    acceptedButtons: Qt.RightButton
                                    onTapped: eventPoint => root.dayContextRequested(dayColumn.day, dayColumn, eventPoint.position.x, eventPoint.position.y)
                                }
                            }
                        }
                    }
                }

                Item {
                    anchors.right: parent.right
                    width: parent.width - root.timeColumnWidth
                    height: parent.height
                    clip: true
                    visible: root.nowHour >= root.startHour && root.nowHour < root.endHour
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
                            x: root.slidePx + (I18n.isRtl ? 6 - root.nowIndex : root.nowIndex) * root.dayWidth
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

    NumberAnimation {
        id: snapAnim
        target: root
        property: "slidePx"
        to: 0
        duration: Theme.shortDuration
        easing.type: Theme.standardEasing
    }

    DankSlideArea {
        id: slidePager
        anchors.fill: parent
        z: 50
        dragEnabled: !root.eventPointerDown && !root.eventDragging
        onMoved: dx => root.applySlide(dx)
        onStepped: direction => root.slideDays((I18n.isRtl ? 1 : -1) * direction)
        onSettled: root.settleSlide()
    }

    DankSlideDragHandler {
        slideArea: slidePager
    }

    Rectangle {
        visible: root.eventDragging && root.draggedEvent
        x: Math.min(root.width - width - Theme.spacingS, Math.max(Theme.spacingS, root.dragPosition.x + 12))
        y: Math.min(root.height - height - Theme.spacingS, Math.max(Theme.spacingS, root.dragPosition.y + 12))
        width: Math.min(220, dragLabel.implicitWidth + Theme.spacingL * 2)
        height: 34
        radius: Theme.cornerRadiusSmall
        color: Theme.surfaceContainerHigh
        border.color: Theme.primary
        border.width: 2
        z: 100

        StyledText {
            id: dragLabel
            anchors.centerIn: parent
            text: root.selectedEventKeys.indexOf(root.draggedEvent ? DankCalService.eventKey(root.draggedEvent) : "") !== -1 && root.selectedEventKeys.length > 1 ? I18n.tr("Move %1 events", "event drag preview label; %1 is event count").arg(root.selectedEventKeys.length) : (root.draggedEvent ? root.draggedEvent.title : "")
            font.pixelSize: Theme.fontSizeSmall
            font.weight: Font.Medium
            color: Theme.surfaceText
            elide: Text.ElideRight
            width: Math.min(196, implicitWidth)
        }
    }
}
