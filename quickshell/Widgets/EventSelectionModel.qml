import QtQuick
import qs.Services

Item {
    id: root

    property var selectedKeys: []
    property string anchorKey: ""
    property bool busy: false

    readonly property int count: selectedKeys.length
    readonly property bool hasSelection: count > 0

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
