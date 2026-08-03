.pragma library

var clipboardProductId = "-//Dank Calendar//Event Clipboard 1.0//EN"
var clipboardDataPrefix = "X-DANKCALENDAR-EVENT:"

function eventKey(event) {
    return event.id + "|" + event.uid + "|" + new Date(event.start).getTime()
}

function startOfDay(value) {
    const date = new Date(value)
    return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function daysBetween(from, to) {
    const fromDay = startOfDay(from)
    const toDay = startOfDay(to)
    return Math.round((toDay.getTime() - fromDay.getTime()) / 86400000)
}

function shiftDate(value, days) {
    const shifted = new Date(value)
    shifted.setDate(shifted.getDate() + days)
    return shifted
}

function cloneValue(value) {
    return JSON.parse(JSON.stringify(value || []))
}

function createFields(event, dayOffset, calendarId, preserveRecurrence) {
    const fields = {
        "calendarId": calendarId || event.calendarId,
        "summary": event.title,
        "description": event.description || "",
        "location": event.location || "",
        "start": shiftDate(event.start, dayOffset).toISOString(),
        "end": shiftDate(event.end, dayOffset).toISOString(),
        "allDay": !!event.allDay,
        "reminders": cloneValue(event.reminders)
    }
    if (preserveRecurrence && event.recurrence && event.recurrence.length > 0)
        fields.recurrence = cloneValue(event.recurrence)
    return fields
}

function moveFields(event, dayOffset) {
    const fields = {
        "start": shiftDate(event.start, dayOffset).toISOString(),
        "end": shiftDate(event.end, dayOffset).toISOString()
    }
    if ((event.recurringId || "") !== "" || (event.recurrence || []).length > 0)
        fields.occurrenceStart = new Date(event.start).toISOString()
    return fields
}

function sortedUnique(events) {
    const seen = {}
    return (events || []).filter(event => {
        const key = eventKey(event)
        if (seen[key])
            return false
        seen[key] = true
        return true
    }).sort((left, right) => {
        const startDelta = new Date(left.start).getTime() - new Date(right.start).getTime()
        if (startDelta !== 0)
            return startDelta
        return eventKey(left).localeCompare(eventKey(right))
    })
}

function escapeICal(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/\r?\n/g, "\\n").replace(/,/g, "\\,").replace(/;/g, "\\;")
}

function formatICalDate(value, allDay) {
    const date = new Date(value)
    const pad = number => String(number).padStart(2, "0")
    if (allDay)
        return date.getFullYear() + pad(date.getMonth() + 1) + pad(date.getDate())
    return date.getUTCFullYear() + pad(date.getUTCMonth() + 1) + pad(date.getUTCDate()) + "T" + pad(date.getUTCHours()) + pad(date.getUTCMinutes()) + pad(date.getUTCSeconds()) + "Z"
}

function clipboardText(events) {
    const ordered = sortedUnique(events)
    const lines = ["BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:" + clipboardProductId, "CALSCALE:GREGORIAN"]
    for (let i = 0; i < ordered.length; i++) {
        const event = ordered[i]
        const fields = createFields(event, 0, event.calendarId, true)
        lines.push("BEGIN:VEVENT")
        lines.push("SUMMARY:" + escapeICal(event.title))
        if (event.description)
            lines.push("DESCRIPTION:" + escapeICal(event.description))
        if (event.location)
            lines.push("LOCATION:" + escapeICal(event.location))
        lines.push((event.allDay ? "DTSTART;VALUE=DATE:" : "DTSTART:") + formatICalDate(event.start, event.allDay))
        lines.push((event.allDay ? "DTEND;VALUE=DATE:" : "DTEND:") + formatICalDate(event.end, event.allDay))
        lines.push(clipboardDataPrefix + encodeURIComponent(JSON.stringify(fields)))
        lines.push("END:VEVENT")
    }
    lines.push("END:VCALENDAR")
    return lines.join("\r\n")
}

function clipboardEvents(text) {
    if (!text || text.indexOf("BEGIN:VCALENDAR") === -1)
        return []
    const lines = String(text).replace(/\r\n/g, "\n").split("\n")
    const events = []
    for (let i = 0; i < lines.length; i++) {
        if (lines[i].indexOf(clipboardDataPrefix) !== 0)
            continue
        try {
            const fields = JSON.parse(decodeURIComponent(lines[i].substring(clipboardDataPrefix.length)))
            if (!fields.summary || !fields.start || !fields.end)
                continue
            events.push(fields)
        } catch (error) {
            // Ignore malformed clipboard entries and keep scanning for others.
        }
    }
    return events
}

function pasteFields(copiedFields, targetDay, fallbackCalendarId) {
    if (!copiedFields || copiedFields.length === 0)
        return []
    const ordered = copiedFields.slice().sort((left, right) => new Date(left.start) - new Date(right.start))
    const offset = daysBetween(ordered[0].start, targetDay)
    return ordered.map(fields => {
        const copy = Object.assign({}, fields)
        copy.calendarId = copy.calendarId || fallbackCalendarId
        copy.start = shiftDate(copy.start, offset).toISOString()
        copy.end = shiftDate(copy.end, offset).toISOString()
        copy.reminders = cloneValue(copy.reminders)
        if (copy.recurrence)
            copy.recurrence = cloneValue(copy.recurrence)
        return copy
    })
}
