//@ pragma Env QSG_RENDER_LOOP=threaded
//@ pragma Env QT_WAYLAND_DISABLE_WINDOWDECORATION=1
//@ pragma Env QT_QUICK_CONTROLS_STYLE=Material
//@ pragma UseQApplication
//@ pragma AppId com.danklinux.dankcalendar

import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.Common
import qs.Modules
import qs.Services

ShellRoot {
    id: root

    readonly property var log: Log.scoped("Shell")
    readonly property bool enableHotReload: Quickshell.env("DCAL_ENABLE_HOTRELOAD") === "1" || Quickshell.env("DCAL_ENABLE_HOTRELOAD") === "true"

    Component.onCompleted: {
        Quickshell.watchFiles = enableHotReload;
    }

    function ownToplevel() {
        const toplevels = ToplevelManager.toplevels;
        if (!toplevels || !toplevels.values)
            return null;

        for (const t of toplevels.values) {
            if (t.appId === "com.danklinux.dankcalendar")
                return t;
        }
        return null;
    }

    function focusToplevel() {
        const t = ownToplevel();
        if (!t)
            return;
        if (t.minimized)
            t.minimized = false;
        t.activate();
    }

    function showAndFocus() {
        if (windowLoader.active) {
            focusToplevel();
            return;
        }
        windowLoader.active = true;
        focusRetry.restart();
    }

    property string pendingSubscribeUrl: ""
    property var pendingImport: null
    property string pendingView: ""
    property var pendingEvent: null
    property var pendingNewEvent: null

    function handleWindowAction(action, view) {
        view = view || "";
        switch (action) {
        case "show":
            pendingView = view;
            showAndFocus();
            break;
        case "hide":
            windowLoader.active = false;
            break;
        case "toggle":
            {
                const t = windowLoader.active ? ownToplevel() : null;
                if (t && !t.minimized) {
                    windowLoader.active = false;
                    return;
                }
                pendingView = view;
                showAndFocus();
                break;
            }
        }
        applyPendingView();
    }

    function applyPendingView() {
        if (!windowLoader.item || pendingView === "")
            return;
        windowLoader.item.currentView = pendingView;
        pendingView = "";
    }

    function handleSubscribe(url) {
        pendingSubscribeUrl = url;
        showAndFocus();
        applyPendingSubscribe();
    }

    function applyPendingSubscribe() {
        if (!windowLoader.item || pendingSubscribeUrl === "")
            return;
        windowLoader.item.openSubscribe(pendingSubscribeUrl);
        pendingSubscribeUrl = "";
    }

    function handleImportIcs(ics, name) {
        if (ics === "")
            return;
        pendingImport = {
            "ics": ics,
            "name": name
        };
        showAndFocus();
        applyPendingImport();
    }

    function applyPendingImport() {
        if (!windowLoader.item || !pendingImport)
            return;
        windowLoader.item.openImport(pendingImport.ics, pendingImport.name);
        pendingImport = null;
    }

    function handleOpenEvent(uid, start) {
        if (uid === "")
            return;
        pendingEvent = {
            "uid": uid,
            "start": start
        };
        showAndFocus();
        applyPendingEvent();
    }

    function applyPendingEvent() {
        if (!windowLoader.item || !pendingEvent)
            return;
        windowLoader.item.openEventByUid(pendingEvent.uid, pendingEvent.start);
        pendingEvent = null;
    }

    function handleNewEvent(start) {
        pendingNewEvent = {
            "start": start || ""
        };
        showAndFocus();
        applyPendingNewEvent();
    }

    function applyPendingNewEvent() {
        if (!windowLoader.item || !pendingNewEvent)
            return;
        const start = pendingNewEvent.start;
        pendingNewEvent = null;
        if (start !== "") {
            const d = new Date(start);
            if (!isNaN(d.getTime())) {
                windowLoader.item.openCreateEvent(d);
                return;
            }
        }
        windowLoader.item.openCreateEvent();
    }

    Connections {
        target: DankCalService
        function onWindowActionRequested(action, view) {
            root.handleWindowAction(action, view);
        }
        function onSubscribeRequested(url) {
            root.handleSubscribe(url);
        }
        function onImportIcsRequested(ics, name) {
            root.handleImportIcs(ics, name);
        }
        function onOpenEventRequested(uid, start) {
            root.handleOpenEvent(uid, start);
        }
        function onNewEventRequested(start) {
            root.handleNewEvent(start);
        }
    }

    Connections {
        target: windowLoader
        function onItemChanged() {
            root.applyPendingView();
            root.applyPendingSubscribe();
            root.applyPendingImport();
            root.applyPendingEvent();
            root.applyPendingNewEvent();
        }
    }

    Timer {
        id: focusRetry
        interval: 150
        repeat: false
        onTriggered: root.focusToplevel()
    }

    Loader {
        id: trayLoader
        active: SettingsData.showTrayIcon && Quickshell.env("DANKCAL_NO_TRAY") !== "1"
        source: Qt.resolvedUrl("Modules/TrayIcon.qml")
        onLoaded: item.shell = root
        onStatusChanged: {
            if (status === Loader.Error)
                root.log.warn("tray icon unavailable (Qt.labs.platform missing?)");
        }
    }

    LazyLoader {
        id: windowLoader
        active: Quickshell.env("DANKCAL_START_HIDDEN") !== "1"

        CalendarWindow {
            onHideRequested: windowLoader.active = false
        }
    }
}
