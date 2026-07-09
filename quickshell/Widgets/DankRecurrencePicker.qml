import QtQuick
import qs.Common
import qs.Services
import qs.Widgets

Column {
    id: root

    property var rules: []
    property date startDate: new Date()
    property bool allDay: false

    // mode is "" (no repeat), a plain FREQ preset, or "CUSTOM". Rules the
    // builder can't represent stay in extraParts/extraRules and survive a
    // save as long as the user stays in custom mode; picking a plain preset
    // deliberately replaces the whole rule.
    property string mode: ""
    property string customFreq: "WEEKLY"
    property int interval: 1
    property var byDays: []
    property string endMode: "never"
    property int count: 5
    property date untilDate: new Date()
    property var extraParts: []
    property var extraRules: []
    property bool dirty: false

    readonly property bool isRecurring: mode !== ""
    readonly property real numberFieldWidth: Theme.fontSizeMedium * 4

    readonly property var dayCodes: ["SU", "MO", "TU", "WE", "TH", "FR", "SA"]

    readonly property var presetOptions: [
        {
            label: I18n.tr("Does not repeat", "recurrence dropdown option"),
            value: ""
        },
        {
            label: I18n.tr("Daily", "recurrence dropdown option"),
            value: "DAILY"
        },
        {
            label: I18n.tr("Weekly", "recurrence dropdown option"),
            value: "WEEKLY"
        },
        {
            label: I18n.tr("Monthly", "recurrence dropdown option"),
            value: "MONTHLY"
        },
        {
            label: I18n.tr("Yearly", "recurrence dropdown option"),
            value: "YEARLY"
        },
        {
            label: I18n.tr("Custom", "recurrence dropdown option"),
            value: "CUSTOM"
        }
    ]

    readonly property var unitOptions: [
        {
            label: I18n.tr("days", "recurrence unit plural"),
            value: "DAILY"
        },
        {
            label: I18n.tr("weeks", "recurrence unit plural"),
            value: "WEEKLY"
        },
        {
            label: I18n.tr("months", "recurrence unit plural"),
            value: "MONTHLY"
        },
        {
            label: I18n.tr("years", "recurrence unit plural"),
            value: "YEARLY"
        }
    ]

    readonly property var endOptions: [
        {
            label: I18n.tr("Never", "recurrence end condition dropdown option"),
            value: "never"
        },
        {
            label: I18n.tr("After", "recurrence end condition dropdown option, followed by a number of times"),
            value: "count"
        },
        {
            label: I18n.tr("On date", "recurrence end condition dropdown option, followed by a date"),
            value: "until"
        }
    ]

    spacing: Theme.spacingM

    onRulesChanged: _load()
    Component.onCompleted: _load()

    function currentRules() {
        if (!dirty)
            return rules || [];
        if (mode === "")
            return [];

        const freq = mode === "CUSTOM" ? customFreq : mode;
        let rule = "FREQ=" + freq;
        if (mode !== "CUSTOM")
            return [rule];

        if (interval > 1)
            rule += ";INTERVAL=" + interval;
        if (freq === "WEEKLY" && byDays.length > 0)
            rule += ";BYDAY=" + byDays.slice().sort((a, b) => a - b).map(d => dayCodes[d]).join(",");
        switch (endMode) {
        case "count":
            rule += ";COUNT=" + Math.max(1, count);
            break;
        case "until":
            rule += ";UNTIL=" + _untilValue();
            break;
        }
        for (let i = 0; i < extraParts.length; i++)
            rule += ";" + extraParts[i];
        return [rule].concat(extraRules);
    }

    function _load() {
        dirty = false;
        mode = "";
        customFreq = "WEEKLY";
        interval = 1;
        byDays = [];
        endMode = "never";
        count = 5;
        untilDate = _defaultUntil();
        extraParts = [];
        extraRules = (rules || []).slice(1);

        const list = rules || [];
        if (list.length === 0)
            return;

        let freq = "";
        const extras = [];
        const parts = list[0].split(";");
        for (let i = 0; i < parts.length; i++) {
            const kv = parts[i].split("=");
            switch (kv[0]) {
            case "FREQ":
                freq = kv[1];
                break;
            case "INTERVAL":
                interval = parseInt(kv[1]) || 1;
                break;
            case "BYDAY":
                {
                    const days = _plainByDays(kv[1]);
                    if (days)
                        byDays = days;
                    else
                        extras.push(parts[i]);
                    break;
                }
            case "COUNT":
                endMode = "count";
                count = parseInt(kv[1]) || 1;
                break;
            case "UNTIL":
                {
                    const d = DankCalService.rruleUntilDate(kv[1]);
                    if (d) {
                        endMode = "until";
                        untilDate = d;
                    } else {
                        extras.push(parts[i]);
                    }
                    break;
                }
            default:
                extras.push(parts[i]);
            }
        }
        extraParts = extras;

        switch (freq) {
        case "DAILY":
        case "WEEKLY":
        case "MONTHLY":
        case "YEARLY":
            {
                customFreq = freq;
                const plain = interval <= 1 && byDays.length === 0 && endMode === "never" && extras.length === 0 && extraRules.length === 0;
                mode = plain ? freq : "CUSTOM";
                break;
            }
        default:
            mode = "CUSTOM";
        }
    }

    function _plainByDays(value) {
        const days = [];
        const tokens = value.split(",");
        for (let i = 0; i < tokens.length; i++) {
            const day = dayCodes.indexOf(tokens[i]);
            if (day === -1)
                return null;
            days.push(day);
        }
        return days;
    }

    function _defaultUntil() {
        const base = isNaN(startDate.getTime()) ? new Date() : startDate;
        return new Date(base.getFullYear(), base.getMonth() + 1, base.getDate());
    }

    function _untilValue() {
        const y = untilDate.getFullYear();
        const m = String(untilDate.getMonth() + 1).padStart(2, "0");
        const d = String(untilDate.getDate()).padStart(2, "0");
        if (allDay)
            return "" + y + m + d;
        return "" + y + m + d + "T235959Z";
    }

    function _optionLabel(options, value) {
        for (let i = 0; i < options.length; i++) {
            if (options[i].value === value)
                return options[i].label;
        }
        return options[0].label;
    }

    function _toggleDay(day) {
        dirty = true;
        const days = byDays.slice();
        const at = days.indexOf(day);
        if (at === -1)
            days.push(day);
        else
            days.splice(at, 1);
        byDays = days;
    }

    DankDropdown {
        width: parent.width
        options: root.presetOptions.map(o => o.label)
        currentValue: root._optionLabel(root.presetOptions, root.mode)
        onValueChanged: value => {
            for (let i = 0; i < root.presetOptions.length; i++) {
                if (root.presetOptions[i].label !== value)
                    continue;
                root.dirty = true;
                root.mode = root.presetOptions[i].value;
                break;
            }
        }
    }

    Row {
        width: parent.width
        spacing: Theme.spacingM
        visible: root.mode === "CUSTOM"

        StyledText {
            id: everyLabel
            text: I18n.tr("Every", "recurrence interval label, followed by a count and unit")
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceText
            anchors.verticalCenter: parent.verticalCenter
        }

        DankTextField {
            width: root.numberFieldWidth
            text: String(root.interval)
            onTextChanged: {
                const n = parseInt(text);
                const v = isNaN(n) || n < 1 ? 1 : n;
                if (v === root.interval)
                    return;
                root.dirty = true;
                root.interval = v;
            }
            anchors.verticalCenter: parent.verticalCenter
        }

        DankDropdown {
            width: parent.width - everyLabel.implicitWidth - root.numberFieldWidth - Theme.spacingM * 2
            options: root.unitOptions.map(o => o.label)
            currentValue: root._optionLabel(root.unitOptions, root.customFreq)
            onValueChanged: value => {
                for (let i = 0; i < root.unitOptions.length; i++) {
                    if (root.unitOptions[i].label !== value)
                        continue;
                    root.dirty = true;
                    root.customFreq = root.unitOptions[i].value;
                    break;
                }
            }
            anchors.verticalCenter: parent.verticalCenter
        }
    }

    Flow {
        width: parent.width
        spacing: Theme.spacingXS
        visible: root.mode === "CUSTOM" && root.customFreq === "WEEKLY"

        Repeater {
            model: 7

            StyledRect {
                id: dayChip

                required property int index

                readonly property int day: (SettingsData.effectiveFirstDayOfWeek + index) % 7
                readonly property bool selected: root.byDays.indexOf(day) !== -1

                width: Math.max(height, chipLabel.implicitWidth + Theme.spacingM * 2)
                height: 32
                radius: height / 2
                color: selected ? Theme.primary : Theme.surfaceContainerHigh

                StyledText {
                    id: chipLabel
                    anchors.centerIn: parent
                    text: SettingsData.dayName(dayChip.day)
                    font.pixelSize: Theme.fontSizeSmall
                    color: dayChip.selected ? Theme.primaryText : Theme.surfaceText
                }

                StateLayer {
                    onClicked: root._toggleDay(dayChip.day)
                }
            }
        }
    }

    Row {
        width: parent.width
        spacing: Theme.spacingM
        visible: root.mode === "CUSTOM"

        StyledText {
            id: endsLabel
            text: I18n.tr("Ends", "recurrence end condition label")
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceText
            anchors.verticalCenter: parent.verticalCenter
        }

        DankDropdown {
            id: endDropdown
            width: (parent.width - endsLabel.implicitWidth - Theme.spacingM * 2) * 0.4
            options: root.endOptions.map(o => o.label)
            currentValue: root._optionLabel(root.endOptions, root.endMode)
            onValueChanged: value => {
                for (let i = 0; i < root.endOptions.length; i++) {
                    if (root.endOptions[i].label !== value)
                        continue;
                    root.dirty = true;
                    root.endMode = root.endOptions[i].value;
                    break;
                }
            }
            anchors.verticalCenter: parent.verticalCenter
        }

        DankTextField {
            width: root.numberFieldWidth
            visible: root.endMode === "count"
            text: String(root.count)
            onTextChanged: {
                const n = parseInt(text);
                const v = isNaN(n) || n < 1 ? 1 : n;
                if (v === root.count)
                    return;
                root.dirty = true;
                root.count = v;
            }
            anchors.verticalCenter: parent.verticalCenter
        }

        StyledText {
            visible: root.endMode === "count"
            text: I18n.tr("times", "recurrence end condition, occurrence count unit")
            font.pixelSize: Theme.fontSizeMedium
            color: Theme.surfaceVariantText
            anchors.verticalCenter: parent.verticalCenter
        }

        DankDatePicker {
            visible: root.endMode === "until"
            width: parent.width - endsLabel.implicitWidth - endDropdown.width - Theme.spacingM * 2
            firstDayOfWeek: SettingsData.effectiveFirstDayOfWeek
            selectedDate: root.untilDate
            onDateSelected: value => {
                root.dirty = true;
                root.untilDate = value;
            }
            anchors.verticalCenter: parent.verticalCenter
        }
    }
}
