import QtQuick
import Quickshell
import qs.Common
import qs.Modals
import qs.Services
import qs.Widgets
import qs.DankCommon.Widgets

Item {
    id: root

    property string currentView: "month"
    property date selectedDate: new Date()
    property var actionCalendar: null
    property var actionAccount: null
    property bool calendarsExpanded: true
    property bool accountsExpanded: true
    property bool tasksExpanded: true
    property bool keyboardActive: false
    property int navIndex: -1

    signal viewChanged(string view)
    signal todayRequested
    signal createEventRequested
    signal createTaskRequested
    signal taskClicked(var task)
    signal addAccountRequested

    implicitWidth: 240

    readonly property var viewItems: {
        const items = [
            {
                view: "day",
                label: I18n.tr("Day", "view switcher option in sidebar"),
                icon: "calendar_view_day"
            },
            {
                view: "week",
                label: I18n.tr("Week", "view switcher option in sidebar"),
                icon: "calendar_view_week"
            },
            {
                view: "month",
                label: I18n.tr("Month", "view switcher option in sidebar"),
                icon: "calendar_view_month"
            },
            {
                view: "agenda",
                label: I18n.tr("Agenda", "view switcher option in sidebar"),
                icon: "view_agenda"
            }
        ];
        if (SettingsData.showTasks && DankCalService.hasTaskLists())
            items.push({
                view: "tasks",
                label: I18n.tr("Tasks", "view switcher option in sidebar"),
                icon: "task_alt"
            });
        return items;
    }

    // Flat keyboard navigation order: view switcher, then each section header
    // followed by its rows when expanded.
    readonly property var navItems: {
        const items = [];
        for (let i = 0; i < viewItems.length; i++)
            items.push({
                type: "view",
                key: "view:" + viewItems[i].view,
                view: viewItems[i].view
            });
        items.push({
            type: "section",
            key: "section:calendars",
            section: "calendars"
        });
        if (calendarsExpanded) {
            const cals = DankCalService.eventCalendars();
            for (let i = 0; i < cals.length; i++)
                items.push({
                    type: "calendar",
                    key: "cal:" + cals[i].id,
                    data: cals[i]
                });
        }
        if (SettingsData.showTasks && DankCalService.hasTaskLists()) {
            items.push({
                type: "section",
                key: "section:tasks",
                section: "tasks"
            });
            if (tasksExpanded) {
                const tasks = tasksPanel.openTasks;
                for (let i = 0; i < tasks.length; i++)
                    items.push({
                        type: "task",
                        key: "task:" + tasks[i].id,
                        data: tasks[i]
                    });
            }
        }
        items.push({
            type: "section",
            key: "section:accounts",
            section: "accounts"
        });
        if (accountsExpanded) {
            const accs = DankCalService.accounts;
            for (let i = 0; i < accs.length; i++)
                items.push({
                    type: "account",
                    key: "acc:" + accs[i].id,
                    data: accs[i]
                });
        }
        return items;
    }

    readonly property string navSelectedKey: keyboardActive && navIndex >= 0 && navIndex < navItems.length ? navItems[navIndex].key : ""

    onKeyboardActiveChanged: {
        if (keyboardActive && (navIndex < 0 || navIndex >= navItems.length))
            navIndex = 0;
    }

    function currentNav() {
        if (navIndex < 0 || navIndex >= navItems.length)
            return null;
        return navItems[navIndex];
    }

    function moveNav(delta) {
        if (navItems.length === 0)
            return;
        if (navIndex < 0) {
            navIndex = 0;
            return;
        }
        navIndex = Math.max(0, Math.min(navItems.length - 1, navIndex + delta));
    }

    function sectionOf(item) {
        switch (item.type) {
        case "section":
            return item.section;
        case "calendar":
            return "calendars";
        case "task":
            return "tasks";
        case "account":
            return "accounts";
        default:
            return "";
        }
    }

    function isSectionExpanded(name) {
        switch (name) {
        case "calendars":
            return calendarsExpanded;
        case "tasks":
            return tasksExpanded;
        case "accounts":
            return accountsExpanded;
        default:
            return false;
        }
    }

    function setSectionExpanded(name, expanded) {
        switch (name) {
        case "calendars":
            calendarsExpanded = expanded;
            break;
        case "tasks":
            tasksExpanded = expanded;
            break;
        case "accounts":
            accountsExpanded = expanded;
            break;
        default:
            return;
        }
        if (!expanded)
            navIndex = navItems.findIndex(i => i.key === "section:" + name);
    }

    function activateNav() {
        const item = currentNav();
        if (!item)
            return;
        switch (item.type) {
        case "view":
            viewChanged(item.view);
            return;
        case "section":
            setSectionExpanded(item.section, !isSectionExpanded(item.section));
            return;
        case "calendar":
            DankCalService.setCalendarHidden(item.data.id, !item.data.hidden);
            return;
        case "task":
            taskClicked(item.data);
            return;
        case "account":
            DankCalService.refreshAccount(item.data.id);
            return;
        }
    }

    function removeNav() {
        const item = currentNav();
        if (!item)
            return;
        switch (item.type) {
        case "calendar":
            confirmDeleteCalendar(item.data);
            return;
        case "account":
            confirmRemoveAccount(item.data);
            return;
        }
    }

    function renameNav() {
        const item = currentNav();
        if (!item || item.type !== "calendar")
            return;
        openRenameCalendar(item.data);
    }

    function handleKey(event) {
        switch (event.key) {
        case Qt.Key_J:
        case Qt.Key_Down:
            moveNav(1);
            break;
        case Qt.Key_K:
        case Qt.Key_Up:
            moveNav(-1);
            break;
        case Qt.Key_H:
        case Qt.Key_Left:
            {
                const item = currentNav();
                if (item)
                    setSectionExpanded(sectionOf(item), false);
                break;
            }
        case Qt.Key_L:
        case Qt.Key_Right:
            {
                const item = currentNav();
                if (item)
                    setSectionExpanded(sectionOf(item), true);
                break;
            }
        case Qt.Key_Space:
        case Qt.Key_Return:
        case Qt.Key_Enter:
            activateNav();
            break;
        case Qt.Key_Delete:
        case Qt.Key_Backspace:
            removeNav();
            break;
        case Qt.Key_R:
            renameNav();
            break;
        default:
            event.accepted = false;
            return;
        }
        event.accepted = true;
    }

    function revealNav(item) {
        const y = item.mapToItem(scrollColumn, 0, 0).y;
        if (y < scroll.contentY) {
            scroll.contentY = Math.max(0, y - Theme.spacingM);
            return;
        }
        const bottom = y + item.height;
        if (bottom > scroll.contentY + scroll.height)
            scroll.contentY = Math.max(0, Math.min(scroll.contentHeight - scroll.height, bottom - scroll.height + Theme.spacingM));
    }

    function openRenameCalendar(cal) {
        actionCalendar = cal;
        renameLoader.active = true;
        renameLoader.item.show(cal);
    }

    function confirmDeleteCalendar(cal) {
        actionCalendar = cal;
        deleteConfirmLoader.active = true;
        deleteConfirmLoader.item.show({
            title: I18n.tr("Delete \"%1\"?", "confirm dialog title for deleting a calendar, %1 is the calendar name").arg(cal.name),
            message: I18n.tr("Removes this calendar and its events from Dank Calendar. If the provider still offers it, it will come back on the next sync.", "confirm dialog body for deleting a calendar"),
            confirmText: I18n.tr("Delete", "confirm button for deleting a calendar"),
            danger: true
        });
    }

    function confirmRemoveAccount(acc) {
        actionAccount = acc;
        accountRemoveLoader.active = true;
        accountRemoveLoader.item.show({
            title: I18n.tr("Remove \"%1\"?", "confirm dialog title for removing an account, %1 is the account label").arg(DankCalService.accountLabel(acc)),
            message: I18n.tr("Removes this account and its calendars and events from Dank Calendar. Nothing is deleted from the provider.", "confirm dialog body for removing an account"),
            confirmText: I18n.tr("Remove", "confirm button for removing an account"),
            danger: true
        });
    }

    function providerIcon(flavor) {
        switch (flavor) {
        case "google":
            return "mail";
        case "microsoft":
            return "business";
        case "icloud":
            return "cloud_circle";
        case "caldav":
            return "cloud";
        case "local":
            return "folder";
        default:
            return "account_circle";
        }
    }

    function providerLabel(flavor) {
        return DankCalService.providerLabel(flavor);
    }

    component SectionHeader: StyledRect {
        id: sectionHeader
        property string title: ""
        property bool expanded: true
        property string navKey: ""
        readonly property bool navSelected: root.navSelectedKey === navKey
        onNavSelectedChanged: {
            if (navSelected)
                root.revealNav(sectionHeader);
        }

        signal toggled

        width: parent.width
        height: 28
        radius: Theme.cornerRadiusSmall
        color: "transparent"
        border.color: navSelected ? Theme.primary : "transparent"
        border.width: navSelected ? 2 : 0

        Row {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            spacing: Theme.spacingXS

            DankIcon {
                name: "expand_more"
                size: Theme.iconSize - 6
                color: Theme.surfaceVariantText
                rotation: sectionHeader.expanded ? 0 : -90
                anchors.verticalCenter: parent.verticalCenter

                Behavior on rotation {
                    NumberAnimation {
                        duration: Theme.shortDuration
                        easing.type: Theme.standardEasing
                    }
                }
            }

            StyledText {
                text: sectionHeader.title
                font.pixelSize: Theme.fontSizeSmall
                font.weight: Font.Medium
                color: Theme.surfaceVariantText
                anchors.verticalCenter: parent.verticalCenter
            }
        }

        StateLayer {
            stateColor: Theme.primary
            cornerRadius: parent.radius
            onClicked: sectionHeader.toggled()
        }
    }

    DankTooltipV2 {
        id: calTooltip
    }

    DankTooltipV2 {
        id: authTooltip
    }

    Rectangle {
        anchors.fill: parent
        color: Theme.surfaceContainer
        opacity: 0.6
    }

    Column {
        id: topSection
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: Theme.spacingM
        spacing: Theme.spacingL

        DankButton {
            width: parent.width
            text: I18n.tr("Create event", "sidebar button to create a new event")
            iconName: "add"
            buttonHeight: 44
            backgroundColor: Theme.primary
            textColor: Theme.primaryText
            onClicked: root.createEventRequested()
        }

        Column {
            width: parent.width
            spacing: 2

            Repeater {
                model: ScriptModel {
                    values: root.viewItems
                }

                StyledRect {
                    required property var modelData
                    readonly property bool active: root.currentView === modelData.view
                    readonly property bool navSelected: root.navSelectedKey === "view:" + modelData.view

                    width: parent.width
                    height: 40
                    radius: Theme.cornerRadius
                    color: active ? Theme.primaryHover : "transparent"
                    border.color: navSelected ? Theme.primary : "transparent"
                    border.width: navSelected ? 2 : 0

                    Row {
                        anchors.left: parent.left
                        anchors.leftMargin: Theme.spacingM
                        anchors.verticalCenter: parent.verticalCenter
                        spacing: Theme.spacingM

                        DankIcon {
                            name: parent.parent.modelData.icon
                            size: Theme.iconSize - 4
                            color: parent.parent.active ? Theme.primary : Theme.surfaceVariantText
                            anchors.verticalCenter: parent.verticalCenter
                        }

                        StyledText {
                            text: parent.parent.modelData.label
                            font.pixelSize: Theme.fontSizeMedium
                            font.weight: parent.parent.active ? Font.Medium : Font.Normal
                            color: parent.parent.active ? Theme.primary : Theme.surfaceText
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }

                    StateLayer {
                        stateColor: Theme.primary
                        cornerRadius: parent.radius
                        onClicked: root.viewChanged(parent.modelData.view)
                    }
                }
            }
        }
    }

    DankFlickable {
        id: scroll
        anchors.top: topSection.bottom
        anchors.topMargin: Theme.spacingL
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.leftMargin: Theme.spacingM
        anchors.rightMargin: Theme.spacingM
        anchors.bottomMargin: Theme.spacingM
        clip: true
        contentWidth: width
        contentHeight: scrollColumn.implicitHeight

        Column {
            id: scrollColumn
            width: parent.width
            spacing: Theme.spacingL

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.outlineLight
            }

            Column {
                width: parent.width
                spacing: Theme.spacingS

                SectionHeader {
                    title: I18n.tr("My calendars", "sidebar section header for the calendar list")
                    expanded: root.calendarsExpanded
                    navKey: "section:calendars"
                    onToggled: root.calendarsExpanded = !root.calendarsExpanded
                }

                StyledText {
                    visible: root.calendarsExpanded && DankCalService.eventCalendars().length === 0
                    text: DankCalService.connected ? I18n.tr("No calendars yet", "sidebar placeholder when the calendar list is empty") : I18n.tr("Daemon offline", "sidebar placeholder when the daemon is not connected")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.surfaceVariantText
                    width: parent.width
                }

                Repeater {
                    model: ScriptModel {
                        values: DankCalService.eventCalendars()
                    }

                    Item {
                        id: calRow
                        required property var modelData
                        readonly property string accountTooltip: {
                            const acc = DankCalService.accountById(modelData.accountId);
                            if (!acc)
                                return modelData.accountName || I18n.tr("Local calendar", "fallback tooltip for a calendar without an account");
                            const provider = root.providerLabel(DankCalService.accountFlavor(acc));
                            const label = DankCalService.accountLabel(acc);
                            if (!label || label === provider)
                                return provider;
                            return provider + " · " + label;
                        }
                        readonly property string rowTooltip: modelData.name + "  —  " + accountTooltip
                        readonly property bool navSelected: root.navSelectedKey === "cal:" + modelData.id
                        onNavSelectedChanged: {
                            if (navSelected)
                                root.revealNav(calRow);
                        }
                        width: parent.width
                        height: 32
                        visible: root.calendarsExpanded

                        function openMenu(x, y) {
                            root.actionCalendar = modelData;
                            calendarMenu.show(calRow, x, y);
                        }

                        Rectangle {
                            anchors.fill: parent
                            radius: Theme.cornerRadiusSmall
                            color: "transparent"
                            border.color: calRow.navSelected ? Theme.primary : "transparent"
                            border.width: calRow.navSelected ? 2 : 0
                        }

                        Row {
                            anchors.left: parent.left
                            anchors.right: moreButton.left
                            anchors.leftMargin: Theme.spacingXS
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: Theme.spacingM

                            Rectangle {
                                width: 14
                                height: 14
                                radius: 4
                                color: calRow.modelData.hidden ? "transparent" : calRow.modelData.color
                                border.color: calRow.modelData.color
                                border.width: 2
                                anchors.verticalCenter: parent.verticalCenter
                            }

                            StyledText {
                                text: calRow.modelData.name
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.surfaceText
                                opacity: calRow.modelData.hidden ? 0.6 : 1.0
                                width: parent.width - 14 - Theme.spacingM
                                wrapMode: Text.NoWrap
                                maximumLineCount: 1
                                elide: Text.ElideRight
                                anchors.verticalCenter: parent.verticalCenter
                            }
                        }

                        StateLayer {
                            id: calRowState
                            stateColor: Theme.primary
                            cornerRadius: Theme.cornerRadiusSmall
                            acceptedButtons: Qt.LeftButton | Qt.RightButton
                            onEntered: calTooltip.show(calRow.rowTooltip, calRow)
                            onExited: calTooltip.hide()
                            onClicked: mouse => {
                                calTooltip.hide();
                                if (mouse.button === Qt.RightButton) {
                                    calRow.openMenu(mouse.x, mouse.y);
                                    return;
                                }
                                calendarMenu.close();
                                DankCalService.setCalendarHidden(calRow.modelData.id, !calRow.modelData.hidden);
                            }
                        }

                        DankActionButton {
                            id: moreButton
                            property bool hovered: false
                            readonly property bool menuOpenHere: calendarMenu.opened && (root.actionCalendar ? root.actionCalendar.id : "") === calRow.modelData.id
                            anchors.right: parent.right
                            anchors.verticalCenter: parent.verticalCenter
                            buttonSize: 26
                            circular: false
                            iconName: "more_horiz"
                            iconSize: Theme.iconSizeSmall
                            iconColor: Theme.surfaceVariantText
                            opacity: calRowState.containsMouse || hovered || menuOpenHere ? 1 : 0
                            visible: opacity > 0
                            onEntered: hovered = true
                            onExited: hovered = false
                            onClicked: {
                                if (menuOpenHere) {
                                    calendarMenu.close();
                                    return;
                                }
                                calRow.openMenu(calRow.width - calendarMenu.width, calRow.height);
                            }
                        }
                    }
                }
            }

            Rectangle {
                visible: tasksPanel.visible
                width: parent.width
                height: 1
                color: Theme.outlineLight
            }

            Column {
                id: tasksPanel
                width: parent.width
                spacing: Theme.spacingS
                visible: SettingsData.showTasks && DankCalService.hasTaskLists()

                property var openTasks: {
                    const buckets = DankCalService.taskBuckets();
                    return buckets.overdue.concat(buckets.today, buckets.upcoming, buckets.someday);
                }

                SectionHeader {
                    title: I18n.tr("Tasks", "sidebar section header for the task list")
                    expanded: root.tasksExpanded
                    navKey: "section:tasks"
                    onToggled: root.tasksExpanded = !root.tasksExpanded
                }

                StyledText {
                    visible: root.tasksExpanded && tasksPanel.openTasks.length === 0
                    text: I18n.tr("All done", "sidebar placeholder when there are no open tasks")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.surfaceVariantText
                    width: parent.width
                }

                Repeater {
                    model: ScriptModel {
                        values: tasksPanel.openTasks
                    }

                    Item {
                        id: taskRow
                        required property var modelData
                        readonly property string rowTooltip: {
                            const acct = modelData.accountSummary || "";
                            if (modelData.calendar === "")
                                return acct;
                            return acct === "" ? modelData.calendar : modelData.calendar + "  —  " + acct;
                        }
                        readonly property bool navSelected: root.navSelectedKey === "task:" + modelData.id
                        onNavSelectedChanged: {
                            if (navSelected)
                                root.revealNav(taskRow);
                        }
                        width: parent.width
                        height: 42
                        visible: root.tasksExpanded

                        Rectangle {
                            anchors.fill: parent
                            radius: Theme.cornerRadiusSmall
                            color: "transparent"
                            border.color: taskRow.navSelected ? Theme.primary : "transparent"
                            border.width: taskRow.navSelected ? 2 : 0
                        }

                        Rectangle {
                            id: taskCheck
                            width: 16
                            height: 16
                            radius: 8
                            anchors.left: parent.left
                            anchors.leftMargin: Theme.spacingXS
                            anchors.verticalCenter: parent.verticalCenter
                            color: "transparent"
                            border.color: taskRow.modelData.color
                            border.width: 2

                            StateLayer {
                                stateColor: taskRow.modelData.color
                                cornerRadius: parent.radius
                                enabled: !taskRow.modelData.readOnly
                                onClicked: DankCalService.completeTaskWithUndo(taskRow.modelData)
                            }
                        }

                        Column {
                            anchors.left: taskCheck.right
                            anchors.leftMargin: Theme.spacingM
                            anchors.right: parent.right
                            anchors.rightMargin: Theme.spacingXS
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: 1

                            StyledText {
                                width: parent.width
                                text: taskRow.modelData.title
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.surfaceText
                                wrapMode: Text.NoWrap
                                maximumLineCount: 1
                                elide: Text.ElideRight
                                horizontalAlignment: Text.AlignLeft
                            }

                            StyledText {
                                width: parent.width
                                visible: taskRow.modelData.calendar !== ""
                                text: taskRow.modelData.calendar
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.surfaceVariantText
                                wrapMode: Text.NoWrap
                                maximumLineCount: 1
                                elide: Text.ElideRight
                                horizontalAlignment: Text.AlignLeft
                            }
                        }

                        StateLayer {
                            anchors.fill: parent
                            anchors.leftMargin: 16 + Theme.spacingM + Theme.spacingXS
                            stateColor: Theme.primary
                            cornerRadius: Theme.cornerRadiusSmall
                            onEntered: {
                                if (taskRow.rowTooltip !== "")
                                    calTooltip.show(taskRow.rowTooltip, taskRow);
                            }
                            onExited: calTooltip.hide()
                            onClicked: {
                                calTooltip.hide();
                                root.taskClicked(taskRow.modelData);
                            }
                        }
                    }
                }

                DankButton {
                    visible: root.tasksExpanded && DankCalService.taskListCalendars().length > 0
                    width: parent.width
                    text: I18n.tr("Add task", "sidebar button to add a task")
                    iconName: "add"
                    buttonHeight: 36
                    backgroundColor: "transparent"
                    textColor: Theme.primary
                    onClicked: root.createTaskRequested()
                }
            }

            Rectangle {
                width: parent.width
                height: 1
                color: Theme.outlineLight
            }

            Column {
                width: parent.width
                spacing: Theme.spacingS

                SectionHeader {
                    title: I18n.tr("Accounts", "sidebar section header for the account list")
                    expanded: root.accountsExpanded
                    navKey: "section:accounts"
                    onToggled: root.accountsExpanded = !root.accountsExpanded
                }

                Repeater {
                    model: ScriptModel {
                        values: DankCalService.accounts
                    }

                    Item {
                        id: accRow
                        required property var modelData
                        readonly property bool authorized: modelData.authorized !== false && modelData.needsReauth !== true
                        readonly property string flavor: DankCalService.accountFlavor(modelData)
                        readonly property bool reconnectable: modelData.kind === "google" || modelData.kind === "microsoft"
                        readonly property string authReason: {
                            const detail = (modelData.authError || "").trim();
                            if (modelData.needsReauth === true) {
                                const head = I18n.tr("Sign-in expired — click to reconnect", "tooltip on the account warning icon when re-authentication is required");
                                return detail === "" ? head : head + " · " + detail;
                            }
                            if (modelData.authorized === false)
                                return I18n.tr("Not signed in — click to reconnect or re-add this account", "tooltip on the account warning icon when credentials are missing");
                            return I18n.tr("Account problem — click for options", "tooltip on the account warning icon for an unspecified problem");
                        }
                        readonly property bool navSelected: root.navSelectedKey === "acc:" + modelData.id
                        onNavSelectedChanged: {
                            if (navSelected)
                                root.revealNav(accRow);
                        }
                        width: parent.width
                        height: 36
                        visible: root.accountsExpanded

                        function openMenu(x, y) {
                            root.actionAccount = modelData;
                            accountMenu.show(accRow, x, y);
                        }

                        Rectangle {
                            anchors.fill: parent
                            radius: Theme.cornerRadiusSmall
                            color: "transparent"
                            border.color: accRow.navSelected ? Theme.primary : "transparent"
                            border.width: accRow.navSelected ? 2 : 0
                        }

                        MouseArea {
                            id: accRowArea
                            anchors.fill: parent
                            hoverEnabled: true
                            acceptedButtons: Qt.RightButton
                            onClicked: mouse => accRow.openMenu(mouse.x, mouse.y)
                        }

                        DankIcon {
                            id: authWarning
                            visible: !parent.authorized
                            anchors.right: accMoreButton.left
                            anchors.rightMargin: Theme.spacingXS
                            anchors.verticalCenter: parent.verticalCenter
                            name: "warning"
                            size: Theme.iconSize - 8
                            color: Theme.error

                            MouseArea {
                                anchors.fill: parent
                                anchors.margins: -Theme.spacingXS
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onEntered: authTooltip.show(accRow.authReason, authWarning)
                                onExited: authTooltip.hide()
                                onClicked: {
                                    authTooltip.hide();
                                    if (accRow.reconnectable) {
                                        DankCalService.reconnectAccount(accRow.modelData);
                                        return;
                                    }
                                    accRow.openMenu(accRow.width - accountMenu.width, accRow.height);
                                }
                            }
                        }

                        DankActionButton {
                            id: accMoreButton
                            property bool hovered: false
                            readonly property bool menuOpenHere: accountMenu.opened && (root.actionAccount ? root.actionAccount.id : "") === accRow.modelData.id
                            anchors.right: parent.right
                            anchors.verticalCenter: parent.verticalCenter
                            buttonSize: 26
                            circular: false
                            iconName: "more_horiz"
                            iconSize: Theme.iconSizeSmall
                            iconColor: Theme.surfaceVariantText
                            opacity: accRowArea.containsMouse || hovered || menuOpenHere ? 1 : 0
                            visible: opacity > 0
                            onEntered: hovered = true
                            onExited: hovered = false
                            onClicked: {
                                if (menuOpenHere) {
                                    accountMenu.close();
                                    return;
                                }
                                accRow.openMenu(accRow.width - accountMenu.width, accRow.height);
                            }
                        }

                        Row {
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.leftMargin: Theme.spacingXS
                            anchors.rightMargin: parent.authorized ? Theme.spacingXS : Theme.iconSize
                            anchors.verticalCenter: parent.verticalCenter
                            spacing: Theme.spacingM

                            DankIcon {
                                name: root.providerIcon(parent.parent.flavor)
                                size: Theme.iconSize - 6
                                color: parent.parent.authorized ? Theme.surfaceVariantText : Theme.error
                                anchors.verticalCenter: parent.verticalCenter
                            }

                            Column {
                                anchors.verticalCenter: parent.verticalCenter
                                spacing: 0
                                width: parent.width - (Theme.iconSize - 6) - Theme.spacingM

                                StyledText {
                                    text: root.providerLabel(parent.parent.parent.flavor)
                                    font.pixelSize: Theme.fontSizeSmall
                                    font.weight: Font.Medium
                                    color: Theme.surfaceText
                                    width: parent.width
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
                                    elide: Text.ElideRight
                                }

                                StyledText {
                                    text: DankCalService.accountLabel(parent.parent.parent.modelData)
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.surfaceVariantText
                                    width: parent.width
                                    wrapMode: Text.NoWrap
                                    maximumLineCount: 1
                                    elide: Text.ElideRight
                                }
                            }
                        }
                    }
                }

                DankButton {
                    visible: root.accountsExpanded
                    width: parent.width
                    text: I18n.tr("Add account", "sidebar button to add a provider account")
                    iconName: "add"
                    buttonHeight: 36
                    backgroundColor: "transparent"
                    textColor: Theme.primary
                    onClicked: root.addAccountRequested()
                }
            }
        }
    }

    DankPopupMenu {
        id: calendarMenu
        items: {
            const cal = root.actionCalendar;
            if (!cal)
                return [];
            return [
                {
                    id: "toggle",
                    label: cal.hidden ? I18n.tr("Show", "calendar context menu action to show a hidden calendar") : I18n.tr("Hide", "calendar context menu action to hide a calendar"),
                    icon: cal.hidden ? "visibility" : "visibility_off"
                },
                {
                    id: "rename",
                    label: I18n.tr("Rename…", "calendar context menu action to rename a calendar"),
                    icon: "edit"
                },
                {
                    id: "delete",
                    label: I18n.tr("Delete…", "calendar context menu action to delete a calendar"),
                    icon: "delete_outline",
                    danger: true
                }
            ];
        }
        onTriggered: itemId => {
            const cal = root.actionCalendar;
            if (!cal)
                return;
            switch (itemId) {
            case "toggle":
                DankCalService.setCalendarHidden(cal.id, !cal.hidden);
                break;
            case "rename":
                root.openRenameCalendar(cal);
                break;
            case "delete":
                root.confirmDeleteCalendar(cal);
                break;
            }
        }
    }

    Loader {
        id: renameLoader
        active: false
        sourceComponent: RenameCalendarDialog {
            onClosed: renameLoader.active = false
        }
    }

    Loader {
        id: newCalendarLoader
        active: false
        sourceComponent: NewCalendarDialog {
            onClosed: newCalendarLoader.active = false
        }
    }

    Loader {
        id: deleteConfirmLoader
        active: false
        sourceComponent: ConfirmDialog {
            onConfirmed: {
                if (root.actionCalendar)
                    DankCalService.deleteCalendar(root.actionCalendar.id);
            }
            onClosed: deleteConfirmLoader.active = false
        }
    }

    DankPopupMenu {
        id: accountMenu
        items: {
            const entries = [];
            if (root.actionAccount && root.actionAccount.needsReauth === true)
                entries.push({
                    id: "reauth",
                    label: I18n.tr("Reconnect", "account context menu action to re-authorize the account"),
                    icon: "login"
                });
            if (root.actionAccount && root.actionAccount.kind === "local")
                entries.push({
                    id: "newCalendar",
                    label: I18n.tr("New calendar…", "account context menu action to create a calendar in a local folder"),
                    icon: "create_new_folder"
                });
            entries.push({
                id: "sync",
                label: I18n.tr("Sync now", "account context menu action to sync the account"),
                icon: "refresh"
            });
            entries.push({
                id: "remove",
                label: I18n.tr("Remove…", "account context menu action to remove the account"),
                icon: "delete_outline",
                danger: true
            });
            return entries;
        }
        onTriggered: itemId => {
            const acc = root.actionAccount;
            if (!acc)
                return;
            switch (itemId) {
            case "reauth":
                DankCalService.reconnectAccount(acc);
                break;
            case "newCalendar":
                newCalendarLoader.active = true;
                newCalendarLoader.item.show(acc);
                break;
            case "sync":
                DankCalService.refreshAccount(acc.id);
                break;
            case "remove":
                root.confirmRemoveAccount(acc);
                break;
            }
        }
    }

    Loader {
        id: accountRemoveLoader
        active: false
        sourceComponent: ConfirmDialog {
            onConfirmed: {
                if (root.actionAccount)
                    DankCalService.removeAccount(root.actionAccount.id);
            }
            onClosed: accountRemoveLoader.active = false
        }
    }
}
