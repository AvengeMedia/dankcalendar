import QtQuick
import Quickshell
import qs.Common
import qs.Services
import "../Common/EventUtils.js" as EventUtils

Item {
    id: root

    property var selectedKeys: []
    property string anchorKey: ""
    property bool busy: false

    readonly property int count: selectedKeys.length
    readonly property bool hasSelection: count > 0
    readonly property var clipboardEvents: EventUtils.clipboardEvents(Quickshell.clipboardText)
    readonly property int clipboardCount: clipboardEvents.length

    visible: false
    width: 0
    height: 0

    function contains(event) {
        return selectedKeys.indexOf(DankCalService.eventKey(event)) !== -1;
    }

    function clear() {
        selectedKeys = [];
        anchorKey = "";
    }

    function replace(events) {
        selectedKeys = events.map(event => DankCalService.eventKey(event));
        anchorKey = selectedKeys.length > 0 ? selectedKeys[selectedKeys.length - 1] : "";
    }

    function ensureSelected(event) {
        if (contains(event))
            return;
        replace([event]);
    }

    function selectRange(event, additive) {
        const events = DankCalService.visibleEvents();
        const targetKey = DankCalService.eventKey(event);
        let anchorIndex = -1;
        let targetIndex = -1;
        for (let i = 0; i < events.length; i++) {
            const key = DankCalService.eventKey(events[i]);
            if (key === anchorKey)
                anchorIndex = i;
            if (key === targetKey)
                targetIndex = i;
        }
        if (anchorIndex < 0 || targetIndex < 0) {
            replace([event]);
            return;
        }

        const start = Math.min(anchorIndex, targetIndex);
        const end = Math.max(anchorIndex, targetIndex);
        const rangeKeys = events.slice(start, end + 1).map(item => DankCalService.eventKey(item));
        if (!additive) {
            selectedKeys = rangeKeys;
            return;
        }

        const merged = selectedKeys.slice();
        for (let i = 0; i < rangeKeys.length; i++) {
            if (merged.indexOf(rangeKeys[i]) === -1)
                merged.push(rangeKeys[i]);
        }
        selectedKeys = merged;
    }

    function select(event, modifiers) {
        const toggle = (modifiers & (Qt.ControlModifier | Qt.MetaModifier)) !== 0;
        const extend = (modifiers & Qt.ShiftModifier) !== 0;
        const key = DankCalService.eventKey(event);

        if (extend && anchorKey !== "") {
            selectRange(event, toggle);
            return;
        }
        if (toggle) {
            const keys = selectedKeys.slice();
            const index = keys.indexOf(key);
            if (index === -1)
                keys.push(key);
            else
                keys.splice(index, 1);
            selectedKeys = keys;
            anchorKey = key;
            return;
        }
        selectedKeys = [key];
        anchorKey = key;
    }

    function selectDay(day) {
        replace(DankCalService.eventsForDay(day));
    }

    function events(fallback) {
        const resolved = DankCalService.eventsByKeys(selectedKeys);
        if (resolved.length > 0)
            return resolved;
        return fallback ? [fallback] : [];
    }

    function allWritable(fallback) {
        const selected = events(fallback);
        return selected.length > 0 && selected.every(event => !event.readOnly);
    }

    function writableCalendarId(preferredId) {
        const writable = DankCalService.writableCalendars();
        for (let i = 0; i < writable.length; i++) {
            if (writable[i].id === preferredId)
                return preferredId;
        }
        const fallback = DankCalService.defaultCalendar();
        return fallback ? fallback.id : "";
    }

    function finishOperation(verb, total, response) {
        busy = false;
        const completed = (response.results || []).length;
        if (response.error) {
            ToastService.info(I18n.tr("%1 of %2 events %3", "partial batch event action result; %1 completed count, %2 total count, %3 action verb").arg(completed).arg(total).arg(verb));
            return;
        }
        ToastService.info(total === 1 ? I18n.tr("1 event %1", "single event action result; %1 is an action verb").arg(verb) : I18n.tr("%1 events %2", "multiple event action result; %1 is event count, %2 is an action verb").arg(total).arg(verb));
    }

    function copy(fallback) {
        if (fallback)
            ensureSelected(fallback);
        const selected = events(fallback);
        if (selected.length === 0)
            return;
        Quickshell.clipboardText = EventUtils.clipboardText(selected);
        ToastService.info(selected.length === 1 ? I18n.tr("Copied 1 event", "clipboard confirmation for one event") : I18n.tr("Copied %1 events", "clipboard confirmation for multiple events; %1 is event count").arg(selected.length));
    }

    function paste(targetDay) {
        const copied = EventUtils.clipboardEvents(Quickshell.clipboardText);
        if (copied.length === 0 || busy)
            return;
        const fallbackId = writableCalendarId("");
        if (fallbackId === "") {
            ToastService.info(I18n.tr("No writable calendar available", "event paste error when no calendar allows creating events"));
            return;
        }
        const fields = EventUtils.pasteFields(copied, targetDay, fallbackId);
        for (let i = 0; i < fields.length; i++)
            fields[i].calendarId = writableCalendarId(fields[i].calendarId);
        busy = true;
        DankCalService.createEvents(fields, response => {
            root.finishOperation(I18n.tr("pasted", "past-tense event action used in a result message"), fields.length, response);
            root.clear();
        });
    }

    function duplicate(dayOffset, fallback) {
        if (busy)
            return;
        if (fallback)
            ensureSelected(fallback);
        const selected = events(fallback);
        const fields = [];
        for (let i = 0; i < selected.length; i++) {
            const calendarId = writableCalendarId(selected[i].calendarId);
            if (calendarId !== "")
                fields.push(EventUtils.createFields(selected[i], dayOffset, calendarId, false));
        }
        if (fields.length === 0) {
            ToastService.info(I18n.tr("No writable calendar available", "event duplicate error when no calendar allows creating events"));
            return;
        }
        busy = true;
        DankCalService.createEvents(fields, response => root.finishOperation(I18n.tr("created", "past-tense event action used in a result message"), fields.length, response));
    }

    function moveTo(anchorEvent, targetDay) {
        if (busy)
            return;
        ensureSelected(anchorEvent);
        const selected = events(anchorEvent);
        if (!allWritable(anchorEvent)) {
            ToastService.info(I18n.tr("Read-only events can't be moved", "event move error for a read-only calendar"));
            return;
        }
        const offset = EventUtils.daysBetween(anchorEvent.start, targetDay);
        if (offset === 0)
            return;
        busy = true;
        DankCalService.moveEvents(selected, offset, response => {
            root.finishOperation(I18n.tr("moved", "past-tense event action used in a result message"), selected.length, response);
            root.clear();
        });
    }

    function remove(fallback) {
        if (busy)
            return;
        if (fallback)
            ensureSelected(fallback);
        const selected = events(fallback);
        if (!allWritable(fallback)) {
            ToastService.info(I18n.tr("Read-only events can't be deleted", "event delete error for a read-only calendar"));
            return;
        }
        busy = true;
        DankCalService.deleteEvents(selected, response => {
            root.finishOperation(I18n.tr("deleted", "past-tense event action used in a result message"), selected.length, response);
            root.clear();
        });
    }

    Connections {
        target: DankCalService
        function onEventsUpdated() {
            if (root.selectedKeys.length === 0)
                return;
            const available = {};
            const events = DankCalService.visibleEvents();
            for (let i = 0; i < events.length; i++)
                available[DankCalService.eventKey(events[i])] = true;
            const next = root.selectedKeys.filter(key => available[key]);
            if (next.length !== root.selectedKeys.length)
                root.selectedKeys = next;
            if (root.anchorKey !== "" && !available[root.anchorKey])
                root.anchorKey = next.length > 0 ? next[next.length - 1] : "";
        }
    }
}
