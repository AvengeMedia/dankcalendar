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

    function focusToplevel() {
        const toplevels = ToplevelManager.toplevels;
        if (!toplevels || !toplevels.values)
            return;

        for (const t of toplevels.values) {
            if (t.appId === "com.danklinux.dankcalendar") {
                t.activate();
                return;
            }
        }
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
    property string pendingView: ""

    function handleWindowAction(action, view) {
        switch (action) {
        case "show":
            pendingView = view;
            showAndFocus();
            break;
        case "hide":
            windowLoader.active = false;
            break;
        case "toggle":
            if (windowLoader.active) {
                windowLoader.active = false;
                return;
            }
            pendingView = view;
            showAndFocus();
            break;
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

    Connections {
        target: DankCalService
        function onWindowActionRequested(action, view) {
            root.handleWindowAction(action, view);
        }
        function onSubscribeRequested(url) {
            root.handleSubscribe(url);
        }
    }

    Connections {
        target: windowLoader
        function onItemChanged() {
            root.applyPendingView();
            root.applyPendingSubscribe();
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
        active: Quickshell.env("DANKCAL_NO_TRAY") !== "1"
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
