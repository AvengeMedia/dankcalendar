pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell

// ToastService is a lightweight, in-window snackbar. Unlike DMS it does not own
// a layer-shell surface; a Toast item is mounted inside each window that should
// render it. The latest toast replaces any current one (no queue) which suits
// brief confirmations and undo prompts.
Singleton {
    id: root

    property string message: ""
    property string actionLabel: ""
    property var action: null
    property bool active: false

    // show displays msg. opts may carry { actionLabel, action, duration }. An
    // action keeps the toast up longer so it can be clicked before it fades.
    function show(msg, opts) {
        opts = opts || {};
        message = msg;
        actionLabel = opts.actionLabel || "";
        action = (typeof opts.action === "function") ? opts.action : null;
        active = true;
        timer.interval = opts.duration || (actionLabel !== "" ? 6000 : 2500);
        timer.restart();
    }

    function info(msg) {
        show(msg, {});
    }

    function runAction() {
        const pending = action;
        dismiss();
        if (pending)
            pending();
    }

    function dismiss() {
        active = false;
        actionLabel = "";
        action = null;
        timer.stop();
    }

    Timer {
        id: timer
        repeat: false
        onTriggered: root.dismiss()
    }
}
