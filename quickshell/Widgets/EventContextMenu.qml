import QtQuick
import qs.Common
import qs.Services
import qs.Widgets
import "../Common/EventUtils.js" as EventUtils

Item {
    id: root

    property var controller: null
    property var actionEvent: null
    property date targetDay: new Date()
    property date selectedDay: new Date()

    readonly property bool eventMode: actionEvent !== null
    readonly property int selectionCount: controller ? controller.count : 0

    signal openRequested(var event)
    signal createRequested(date day)
    signal deleteRequested(var event)

    visible: false
    width: 0
    height: 0

    function showForEvent(event, anchorItem, x, y) {
        actionEvent = event;
        targetDay = new Date(event.start);
        controller.ensureSelected(event);
        menu.show(anchorItem, x, y);
    }

    function showForDay(day, anchorItem, x, y) {
        actionEvent = null;
        targetDay = new Date(day);
        menu.show(anchorItem, x, y);
    }

    function countLabel(action, count) {
        if (count === 1)
            return action;
        return action + " " + count + " " + I18n.tr("events", "plural event noun in context menu actions");
    }

    function eventSubtitle(event) {
        const when = event.allDay ? I18n.tr("All day", "all-day marker in event context menu") : SettingsData.formatTime(event.start) + " – " + SettingsData.formatTime(event.end);
        return event.calendar ? when + " · " + event.calendar : when;
    }

    function menuItems() {
        const items = [];
        if (!eventMode) {
            items.push({
                type: "header",
                label: Qt.formatDate(targetDay, "dddd, MMMM d"),
                subtitle: I18n.tr("Day actions", "subtitle for a calendar day context menu")
            });
            items.push({
                id: "create",
                label: I18n.tr("New event", "calendar day context menu action"),
                icon: "add"
            });
            items.push({
                id: "paste",
                label: controller && controller.clipboardCount === 1 ? I18n.tr("Paste 1 event", "calendar day context menu paste action for one event") : I18n.tr("Paste %1 events", "calendar day context menu paste action; %1 is event count").arg(controller ? controller.clipboardCount : 0),
                icon: "content_paste",
                shortcut: "Ctrl+V",
                enabled: controller && controller.clipboardCount > 0 && !controller.busy
            });
            const dayCount = DankCalService.eventsForDay(targetDay).length;
            items.push({
                id: "selectDay",
                label: dayCount === 1 ? I18n.tr("Select event", "calendar day context menu action for one event") : I18n.tr("Select all %1 events", "calendar day context menu selection action; %1 is event count").arg(dayCount),
                icon: "select_all",
                enabled: dayCount > 0
            });
            return items;
        }

        if (selectionCount === 1) {
            items.push({
                type: "header",
                label: actionEvent.title,
                subtitle: eventSubtitle(actionEvent)
            });
            items.push({
                id: "open",
                label: I18n.tr("Open event", "event context menu action"),
                icon: "open_in_new",
                shortcut: "Enter"
            });
        } else {
            items.push({
                type: "header",
                label: I18n.tr("%1 events selected", "event context menu selection summary; %1 is event count").arg(selectionCount),
                subtitle: I18n.tr("Actions apply to the whole selection", "event context menu multi-selection hint")
            });
        }

        items.push({
            type: "separator"
        });
        items.push({
            id: "copy",
            label: selectionCount === 1 ? I18n.tr("Copy event", "event context menu copy action for one event") : I18n.tr("Copy %1 events", "event context menu copy action; %1 is event count").arg(selectionCount),
            icon: "content_copy",
            shortcut: "Ctrl+C",
            enabled: !controller.busy
        });
        items.push({
            id: "duplicate",
            label: countLabel(I18n.tr("Duplicate", "event context menu duplicate action"), selectionCount),
            icon: "file_copy",
            shortcut: "Ctrl+D",
            enabled: !controller.busy
        });
        items.push({
            id: "tomorrow",
            label: countLabel(I18n.tr("Repeat tomorrow", "event context menu quick repeat action"), selectionCount),
            icon: "update",
            enabled: !controller.busy
        });
        items.push({
            id: "nextWeek",
            label: countLabel(I18n.tr("Repeat next week", "event context menu quick repeat action"), selectionCount),
            icon: "event_repeat",
            enabled: !controller.busy
        });
        items.push({
            id: "moveSelected",
            label: I18n.tr("Move to selected day", "event context menu move action"),
            icon: "drive_file_move",
            enabled: controller.allWritable(actionEvent) && EventUtils.daysBetween(actionEvent.start, selectedDay) !== 0 && !controller.busy
        });
        if (controller.clipboardCount > 0) {
            items.push({
                id: "paste",
                label: controller.clipboardCount === 1 ? I18n.tr("Paste 1 event here", "event context menu paste action for one event") : I18n.tr("Paste %1 events here", "event context menu paste action; %1 is event count").arg(controller.clipboardCount),
                icon: "content_paste",
                shortcut: "Ctrl+V",
                enabled: !controller.busy
            });
        }
        items.push({
            type: "separator"
        });
        items.push({
            id: "delete",
            label: selectionCount === 1 ? I18n.tr("Delete event…", "event context menu delete action for one event") : I18n.tr("Delete %1 events…", "event context menu delete action; %1 is event count").arg(selectionCount),
            icon: "delete_outline",
            danger: true,
            enabled: controller.allWritable(actionEvent) && !controller.busy
        });
        return items;
    }

    DankPopupMenu {
        id: menu
        preferredWidth: 264
        items: root.menuItems()
        onTriggered: itemId => {
            switch (itemId) {
            case "open":
                root.openRequested(root.actionEvent);
                break;
            case "create":
                root.createRequested(root.targetDay);
                break;
            case "selectDay":
                root.controller.selectDay(root.targetDay);
                break;
            case "copy":
                root.controller.copy(root.actionEvent);
                break;
            case "duplicate":
                root.controller.duplicate(0, root.actionEvent);
                break;
            case "tomorrow":
                root.controller.duplicate(1, root.actionEvent);
                break;
            case "nextWeek":
                root.controller.duplicate(7, root.actionEvent);
                break;
            case "moveSelected":
                root.controller.moveTo(root.actionEvent, root.selectedDay);
                break;
            case "paste":
                root.controller.paste(root.targetDay);
                break;
            case "delete":
                root.deleteRequested(root.actionEvent);
                break;
            }
        }
    }
}
