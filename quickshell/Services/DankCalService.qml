pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Services

Singleton {
    id: root

    readonly property var log: Log.scoped("DankCalService")

    readonly property string socketPath: Quickshell.env("DANKCAL_SOCKET")

    property bool socketReady: false
    property bool connected: false
    property bool connecting: false
    property bool subscribed: false
    property int apiVersion: 0
    property string daemonVersion: ""
    property var capabilities: []

    property var accounts: []
    property var providers: []
    property var googleSetupSteps: []
    property var microsoftSetupSteps: []
    property string lastError: ""

    property bool autostartEnabled: false

    property var calendars: []
    property var events: []
    property bool eventsLoading: false
    property date focusDate: new Date()
    property var _loadedFrom: null
    property var _loadedTo: null

    readonly property var fallbackPalette: ["#7287fd", "#f38ba8", "#a6e3a1", "#fab387", "#cba6f7", "#94e2d5", "#f9e2af", "#89dceb"]

    property var pendingRequests: ({})
    property int requestCounter: 0

    signal connectionStateChanged
    signal googleFlowStarted(string state, string authUrl)
    signal googleFlowCompleted(string accountId, string email)
    signal googleFlowFailed(string state, string error)
    signal microsoftFlowStarted(string state, string authUrl)
    signal microsoftFlowCompleted(string accountId, string email)
    signal microsoftFlowFailed(string state, string error)
    signal accountAdded(string accountId)
    signal accountRemoved(string accountId)
    signal eventsUpdated
    signal windowActionRequested(string action)
    signal subscribeRequested(string url)

    onFocusDateChanged: _ensureWindow()

    Component.onCompleted: {
        if (socketPath && socketPath.length > 0) {
            socketProbe.running = true;
        }
    }

    Process {
        id: socketProbe
        command: ["test", "-S", root.socketPath]
        running: false
        onExited: code => {
            if (code === 0) {
                root.socketReady = true;
                root.connectSocket();
            } else {
                root.socketReady = false;
            }
        }
    }

    function connectSocket() {
        if (!socketReady || connected || connecting)
            return;

        connecting = true;
        requestSocket.connected = true;
    }

    DankSocket {
        id: requestSocket
        path: root.socketPath
        connected: false

        onConnectionStateChanged: {
            if (connected) {
                root.connected = true;
                root.connecting = false;
                root.connectionStateChanged();
                subscribeSocket.connected = true;
                root.refreshVersion();
                root.refreshAccounts();
                root.refreshProviders();
                root.refreshGoogleSetupSteps();
                root.refreshMicrosoftSetupSteps();
                root.refreshCalendars();
                root.reloadEvents();
                root.refreshAutostart();
            } else {
                root.connected = false;
                root.connecting = false;
                root.connectionStateChanged();
            }
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0)
                    return;

                let response;
                try {
                    response = JSON.parse(line);
                } catch (e) {
                    log.warn("bad response", line.substring(0, 200));
                    return;
                }
                root._handleResponse(response);
            }
        }
    }

    DankSocket {
        id: subscribeSocket
        path: root.socketPath
        connected: false

        onConnectionStateChanged: {
            root.subscribed = connected;
            if (connected)
                root._sendSubscribe();
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0)
                    return;

                let event;
                try {
                    event = JSON.parse(line);
                } catch (e) {
                    return;
                }
                root._handleEvent(event);
            }
        }
    }

    function _sendSubscribe() {
        const req = {
            "id": _nextId(),
            "method": "subscribe",
            "params": {
                "topics": ["accounts", "calendars", "events", "sync", "ui"]
            }
        };
        subscribeSocket.send(req);
    }

    function _nextId() {
        requestCounter++;
        return Date.now() + requestCounter;
    }

    function _handleResponse(response) {
        if (response.event) {
            _handleEvent(response);
            return;
        }

        const id = response.id;
        if (!id) {
            if (response.apiVersion !== undefined) {
                apiVersion = response.apiVersion;
                capabilities = response.capabilities || [];
            }
            return;
        }

        const cb = pendingRequests[id];
        if (cb) {
            delete pendingRequests[id];
            cb(response);
        }
    }

    function _handleEvent(event) {
        const topic = event.event;

        switch (topic) {
        case "accounts":
            refreshAccounts();
            refreshCalendars();
            break;
        case "calendars":
            refreshCalendars();
            break;
        case "events":
        case "sync":
            refreshDebounce.restart();
            break;
        case "ui":
            {
                const data = event.data || {};
                if (data.action === "subscribe")
                    subscribeRequested(data.url || "");
                else
                    windowActionRequested(data.action || "");
                break;
            }
        }
    }

    Timer {
        id: refreshDebounce
        interval: 400
        repeat: false
        onTriggered: {
            root.refreshCalendars();
            root.reloadEvents();
        }
    }

    function sendRequest(method, params, callback) {
        if (!connected) {
            if (callback)
                callback({
                    "error": "not connected to dankcalendar socket"
                });
            return;
        }

        const id = _nextId();
        const req = {
            "id": id,
            "method": method
        };
        if (params)
            req.params = params;
        if (callback)
            pendingRequests[id] = callback;
        requestSocket.send(req);
    }

    // Timestamps churn on every sync; ignore them so unchanged lists don't
    // get reassigned, which would recreate delegates and close their popups.
    function _stableStringify(list) {
        return JSON.stringify(list, (key, value) => (key === "updatedAt" || key === "createdAt") ? undefined : value);
    }

    // Keep the old object for unchanged entries so ScriptModel preserves
    // their delegates and only changed rows are recreated.
    function _mergeStable(next, prev) {
        const prevById = {};
        for (let i = 0; i < prev.length; i++)
            prevById[prev[i].id] = prev[i];
        return next.map(entry => {
            const old = prevById[entry.id];
            return (old && _stableStringify(old) === _stableStringify(entry)) ? old : entry;
        });
    }

    function refreshAccounts() {
        sendRequest("accounts.list", null, response => {
            if (response.error) {
                lastError = response.error;
                return;
            }
            const list = response.result || [];
            if (_stableStringify(list) === _stableStringify(accounts))
                return;
            accounts = _mergeStable(list, accounts);
        });
    }

    function refreshVersion() {
        sendRequest("version", null, response => {
            if (response.error)
                return;
            const info = response.result || {};
            daemonVersion = info.version || "";
            if (info.apiVersion)
                apiVersion = info.apiVersion;
        });
    }

    function refreshProviders() {
        sendRequest("accounts.providers", null, response => {
            if (!response.error)
                providers = response.result || [];
        });
    }

    function refreshGoogleSetupSteps() {
        sendRequest("accounts.google.setupGuide", null, response => {
            if (!response.error)
                googleSetupSteps = response.result || [];
        });
    }

    function refreshMicrosoftSetupSteps() {
        sendRequest("accounts.microsoft.setupGuide", null, response => {
            if (!response.error)
                microsoftSetupSteps = response.result || [];
        });
    }

    function refreshCalendars() {
        sendRequest("calendars.list", null, response => {
            if (response.error) {
                lastError = response.error;
                return;
            }
            const list = response.result || [];
            for (let i = 0; i < list.length; i++) {
                if (!list[i].color)
                    list[i].color = fallbackPalette[i % fallbackPalette.length];
            }
            if (_stableStringify(list) === _stableStringify(calendars))
                return;
            calendars = _mergeStable(list, calendars);
            eventsUpdated();
        });
    }

    function calendarById(id) {
        for (let i = 0; i < calendars.length; i++) {
            if (calendars[i].id === id)
                return calendars[i];
        }
        return null;
    }

    function accountById(id) {
        for (let i = 0; i < accounts.length; i++) {
            if (accounts[i].id === id)
                return accounts[i];
        }
        return null;
    }

    function writableCalendars() {
        return calendars.filter(c => !c.readOnly);
    }

    function defaultCalendar() {
        const writable = writableCalendars().filter(c => !c.hidden);
        return writable.length > 0 ? writable[0] : null;
    }

    function setCalendarHidden(calendarId, hidden, callback) {
        sendRequest("calendars.setHidden", {
            "calendarId": calendarId,
            "hidden": hidden
        }, response => {
            if (!response.error)
                refreshCalendars();
            if (callback)
                callback(response);
        });
    }

    function renameCalendar(calendarId, name, callback) {
        sendRequest("calendars.rename", {
            "calendarId": calendarId,
            "name": name
        }, response => {
            if (response.error)
                lastError = response.error;
            else
                refreshCalendars();
            if (callback)
                callback(response);
        });
    }

    function setCalendarReminders(calendarId, overrides, callback) {
        sendRequest("calendars.setReminders", {
            "calendarId": calendarId,
            "overrides": overrides || {}
        }, response => {
            if (response.error)
                lastError = response.error;
            else
                refreshCalendars();
            if (callback)
                callback(response);
        });
    }

    function deleteCalendar(calendarId, callback) {
        sendRequest("calendars.delete", {
            "calendarId": calendarId
        }, response => {
            if (response.error) {
                lastError = response.error;
            } else {
                refreshCalendars();
                reloadEvents();
            }
            if (callback)
                callback(response);
        });
    }

    function _ensureWindow() {
        if (!_loadedFrom || !_loadedTo) {
            reloadEvents();
            return;
        }
        const margin = 14 * 86400000;
        const t = focusDate.getTime();
        if (t < _loadedFrom.getTime() + margin || t > _loadedTo.getTime() - margin)
            reloadEvents();
    }

    function reloadEvents() {
        if (!connected)
            return;

        const from = new Date(focusDate.getTime() - 60 * 86400000);
        const to = new Date(focusDate.getTime() + 90 * 86400000);
        eventsLoading = true;
        sendRequest("events.list", {
            "from": from.toISOString(),
            "to": to.toISOString(),
            "limit": 5000
        }, response => {
            eventsLoading = false;
            if (response.error) {
                lastError = response.error;
                return;
            }
            _loadedFrom = from;
            _loadedTo = to;
            const raw = (response.result || {}).events || [];
            events = raw.map(e => _normalizeEvent(e));
            eventsUpdated();
        });
    }

    // All-day events are calendar dates stored at UTC midnight; pin them to
    // local midnights so they don't bleed into the previous day in negative
    // UTC offsets.
    function _dayBoundary(iso) {
        const d = new Date(iso);
        return new Date(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate());
    }

    function _normalizeEvent(e) {
        const allDay = !!e.allDay;
        return {
            "id": e.id,
            "calendarId": e.calendarId || "",
            "title": e.summary || "(untitled)",
            "description": e.description || "",
            "location": e.location || "",
            "url": e.url || "",
            "meetingUrl": e.meetingUrl || "",
            "start": allDay ? _dayBoundary(e.start) : new Date(e.start),
            "end": allDay ? _dayBoundary(e.end) : new Date(e.end),
            "allDay": allDay,
            "status": e.status || "confirmed",
            "recurringId": e.recurringId || "",
            "attendees": e.attendees || [],
            "organizer": e.organizer || null,
            "reminders": e.reminders || []
        };
    }

    function decorateEvent(ev) {
        const cal = calendarById(ev.calendarId);
        const out = Object.assign({}, ev);
        out.color = cal ? cal.color : fallbackPalette[0];
        out.calendar = cal ? cal.name : "";
        out.account = cal ? (cal.accountName || cal.accountId || "") : "";
        out.readOnly = cal ? !!cal.readOnly : false;
        return out;
    }

    function _hiddenCalendarIds() {
        const hidden = {};
        for (let i = 0; i < calendars.length; i++) {
            if (calendars[i].hidden)
                hidden[calendars[i].id] = true;
        }
        return hidden;
    }

    function eventsForRange(rangeStart, rangeEnd) {
        const hidden = _hiddenCalendarIds();
        const out = [];
        for (let i = 0; i < events.length; i++) {
            const ev = events[i];
            if (hidden[ev.calendarId])
                continue;
            if (ev.end <= rangeStart || ev.start >= rangeEnd)
                continue;
            out.push(decorateEvent(ev));
        }
        out.sort((a, b) => {
            if (a.allDay !== b.allDay)
                return a.allDay ? -1 : 1;
            return a.start - b.start;
        });
        return out;
    }

    function eventsForDay(day) {
        const dayStart = new Date(day.getFullYear(), day.getMonth(), day.getDate());
        const dayEnd = new Date(day.getFullYear(), day.getMonth(), day.getDate() + 1);
        return eventsForRange(dayStart, dayEnd);
    }

    function layoutTimedEvents(events) {
        const sorted = events.slice().sort((a, b) => {
            if (a.startHour !== b.startHour)
                return a.startHour - b.startHour;
            return b.durationHours - a.durationHours;
        });

        let cluster = [];
        let clusterEnd = -1;
        const columnEnds = [];

        const flushCluster = () => {
            let columns = 0;
            for (const ev of cluster)
                columns = Math.max(columns, ev.column + 1);
            for (const ev of cluster)
                ev.columns = columns;
            cluster = [];
            columnEnds.length = 0;
            clusterEnd = -1;
        };

        for (const ev of sorted) {
            const start = ev.startHour;
            const end = ev.startHour + ev.durationHours;

            if (cluster.length > 0 && start >= clusterEnd)
                flushCluster();

            let column = columnEnds.findIndex(slotEnd => start >= slotEnd);
            if (column === -1) {
                column = columnEnds.length;
                columnEnds.push(end);
            } else {
                columnEnds[column] = end;
            }

            ev.column = column;
            cluster.push(ev);
            clusterEnd = Math.max(clusterEnd, end);
        }

        if (cluster.length > 0)
            flushCluster();

        return sorted;
    }

    function searchEvents(query, callback) {
        sendRequest("events.list", {
            "query": query
        }, response => {
            if (response.error) {
                lastError = response.error;
                callback({
                    "error": response.error,
                    "events": []
                });
                return;
            }
            const hidden = _hiddenCalendarIds();
            const raw = (response.result || {}).events || [];
            const out = [];
            for (let i = 0; i < raw.length; i++) {
                const ev = _normalizeEvent(raw[i]);
                if (hidden[ev.calendarId])
                    continue;
                if (ev.status === "cancelled")
                    continue;
                out.push(decorateEvent(ev));
            }
            callback({
                "events": out
            });
        });
    }

    function createEvent(fields, callback) {
        sendRequest("events.create", fields, response => {
            if (response.error)
                lastError = response.error;
            else
                reloadEvents();
            if (callback)
                callback(response);
        });
    }

    function updateEvent(id, fields, callback) {
        const params = Object.assign({
            "id": id
        }, fields);
        sendRequest("events.update", params, response => {
            if (response.error)
                lastError = response.error;
            else
                reloadEvents();
            if (callback)
                callback(response);
        });
    }

    function deleteEvent(id, callback) {
        sendRequest("events.delete", {
            "id": id
        }, response => {
            if (response.error)
                lastError = response.error;
            else
                reloadEvents();
            if (callback)
                callback(response);
        });
    }

    function startGoogleFlow(displayName, clientId, clientSecret, callback) {
        const params = {
            "displayName": displayName || "",
            "clientId": clientId,
            "clientSecret": clientSecret
        };
        sendRequest("accounts.google.start", params, response => {
            if (response.error) {
                lastError = response.error;
                if (callback)
                    callback(response);
                return;
            }
            const result = response.result || {};
            googleFlowStarted(result.state, result.authUrl);
            if (callback)
                callback(response);
        });
    }

    function completeGoogleFlow(state, callback) {
        sendRequest("accounts.google.complete", {
            "state": state
        }, response => {
            if (response.error) {
                lastError = response.error;
                googleFlowFailed(state, response.error);
                if (callback)
                    callback(response);
                return;
            }
            const result = response.result || {};
            googleFlowCompleted(result.accountId, result.email);
            accountAdded(result.accountId);
            refreshAccounts();
            if (callback)
                callback(response);
        });
    }

    function cancelGoogleFlow(state, callback) {
        sendRequest("accounts.google.cancel", {
            "state": state
        }, callback);
    }

    function startMicrosoftFlow(clientId, callback) {
        sendRequest("accounts.microsoft.start", {
            "clientId": clientId
        }, response => {
            if (response.error) {
                lastError = response.error;
                if (callback)
                    callback(response);
                return;
            }
            const result = response.result || {};
            microsoftFlowStarted(result.state, result.authUrl);
            if (callback)
                callback(response);
        });
    }

    function completeMicrosoftFlow(state, callback) {
        sendRequest("accounts.microsoft.complete", {
            "state": state
        }, response => {
            if (response.error) {
                lastError = response.error;
                microsoftFlowFailed(state, response.error);
                if (callback)
                    callback(response);
                return;
            }
            const result = response.result || {};
            microsoftFlowCompleted(result.accountId, result.email);
            accountAdded(result.accountId);
            refreshAccounts();
            if (callback)
                callback(response);
        });
    }

    function cancelMicrosoftFlow(state, callback) {
        sendRequest("accounts.microsoft.cancel", {
            "state": state
        }, callback);
    }

    function reconnectAccount(acc, callback) {
        if (!acc)
            return;

        let startMethod = "";
        let completeMethod = "";
        switch (acc.kind) {
        case "google":
            startMethod = "accounts.google.reauth";
            completeMethod = "accounts.google.complete";
            break;
        case "microsoft":
            startMethod = "accounts.microsoft.reauth";
            completeMethod = "accounts.microsoft.complete";
            break;
        default:
            lastError = I18n.tr("This account cannot be reconnected — remove it and add it again.", "error when reconnect is unavailable for a provider");
            if (callback)
                callback({
                    "error": lastError
                });
            return;
        }

        sendRequest(startMethod, {
            "accountId": acc.id
        }, response => {
            if (response.error) {
                lastError = response.error;
                if (callback)
                    callback(response);
                return;
            }
            const result = response.result || {};
            Qt.openUrlExternally(result.authUrl);
            sendRequest(completeMethod, {
                "state": result.state
            }, done => {
                if (done.error)
                    lastError = done.error;
                else
                    refreshAccounts();
                if (callback)
                    callback(done);
            });
        });
    }

    function addCalDAVAccount(url, username, password, displayName, callback) {
        sendRequest("accounts.caldav.add", {
            "url": url,
            "username": username,
            "password": password,
            "displayName": displayName || ""
        }, response => {
            if (response.error) {
                lastError = response.error;
            } else {
                const result = response.result || {};
                accountAdded(result.accountId);
                refreshAccounts();
            }
            if (callback)
                callback(response);
        });
    }

    function addICalAccount(url, name, username, password, callback) {
        sendRequest("accounts.ical.add", {
            "url": url,
            "username": username || "",
            "password": password || "",
            "displayName": name || ""
        }, response => {
            if (response.error) {
                lastError = response.error;
            } else {
                const result = response.result || {};
                accountAdded(result.accountId);
                refreshAccounts();
            }
            if (callback)
                callback(response);
        });
    }

    function addLocalAccount(rootPath, displayName, callback) {
        sendRequest("accounts.local.add", {
            "root": rootPath,
            "displayName": displayName || ""
        }, response => {
            if (response.error) {
                lastError = response.error;
            } else {
                const result = response.result || {};
                accountAdded(result.accountId);
                refreshAccounts();
            }
            if (callback)
                callback(response);
        });
    }

    function addEvolutionAccount(displayName, callback) {
        sendRequest("accounts.evolution.add", {
            "displayName": displayName || ""
        }, response => {
            if (response.error) {
                lastError = response.error;
            } else {
                const result = response.result || {};
                accountAdded(result.accountId);
                refreshAccounts();
            }
            if (callback)
                callback(response);
        });
    }

    function createCalendar(accountId, name, callback) {
        sendRequest("calendars.create", {
            "accountId": accountId,
            "name": name
        }, response => {
            if (response.error)
                lastError = response.error;
            else
                refreshCalendars();
            if (callback)
                callback(response);
        });
    }

    function removeAccount(accountId, callback) {
        sendRequest("accounts.delete", {
            "accountId": accountId
        }, response => {
            if (!response.error) {
                accountRemoved(accountId);
                refreshAccounts();
                refreshCalendars();
                reloadEvents();
            }
            if (callback)
                callback(response);
        });
    }

    function sendTestReminder(callback) {
        sendRequest("reminders.test", null, response => {
            if (response.error)
                lastError = response.error;
            if (callback)
                callback(response);
        });
    }

    function refreshAutostart() {
        sendRequest("system.autostart.get", null, response => {
            if (!response.error)
                autostartEnabled = !!(response.result || {}).enabled;
        });
    }

    function setAutostart(enabled, callback) {
        sendRequest("system.autostart.set", {
            "enabled": enabled
        }, response => {
            if (response.error)
                lastError = response.error;
            else
                autostartEnabled = !!(response.result || {}).enabled;
            if (callback)
                callback(response);
        });
    }

    function quit() {
        if (connected) {
            sendRequest("ui.quit", null, null);
            return;
        }
        Qt.quit();
    }

    function refreshAccount(accountId, callback) {
        sendRequest("accounts.refresh", {
            "accountId": accountId
        }, callback);
    }

    function refreshAll(callback) {
        eventsLoading = true;
        sendRequest("accounts.refresh", {}, callback);
    }

    function accountFlavor(acc) {
        if (!acc)
            return "";
        const settings = acc.settings || {};
        if (acc.kind === "caldav" && (settings.preset === "icloud" || (settings.url || "").indexOf("icloud.com") !== -1))
            return "icloud";
        return acc.kind;
    }

    function accountLabel(acc) {
        if (!acc)
            return "";
        if (acc.displayName && acc.displayName !== acc.id)
            return acc.displayName;
        const settings = acc.settings || {};
        return settings.username || acc.displayName || acc.id;
    }
}
