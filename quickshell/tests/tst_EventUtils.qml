import QtQuick
import QtTest
import "../Common/EventUtils.js" as EventUtils

TestCase {
    name: "EventUtils"

    function event(overrides) {
        return Object.assign({
            "id": "event-1",
            "uid": "uid-1",
            "calendarId": "calendar-1",
            "title": "Design review",
            "description": "Review, refine; ship\nBring notes",
            "location": "Studio A",
            "start": new Date(2026, 7, 10, 9, 30),
            "end": new Date(2026, 7, 10, 10, 45),
            "allDay": false,
            "recurringId": "",
            "recurrence": [],
            "reminders": [{
                "method": "popup",
                "minutes": 10
            }]
        }, overrides || {});
    }

    function test_moveFieldsPreserveLocalTimeAndDuration() {
        const original = event();
        const fields = EventUtils.moveFields(original, 3);
        const start = new Date(fields.start);
        const end = new Date(fields.end);

        compare(start.getDate(), 13);
        compare(start.getHours(), 9);
        compare(start.getMinutes(), 30);
        compare(end.getTime() - start.getTime(), 75 * 60000);
        verify(fields.occurrenceStart === undefined);
    }

    function test_recurringMoveIdentifiesOccurrence() {
        const original = event({
            "recurringId": "series-1",
            "recurrence": ["FREQ=WEEKLY"]
        });
        const fields = EventUtils.moveFields(original, 1);

        compare(fields.occurrenceStart, original.start.toISOString());
    }

    function test_clipboardRoundTripPreservesEditableFields() {
        const original = event({
            "title": "Déjeuner & planning"
        });
        const text = EventUtils.clipboardText([original]);
        const copied = EventUtils.clipboardEvents(text);

        verify(text.indexOf("BEGIN:VCALENDAR") !== -1);
        verify(text.indexOf("SUMMARY:Déjeuner & planning") !== -1);
        compare(copied.length, 1);
        compare(copied[0].summary, original.title);
        compare(copied[0].description, original.description);
        compare(copied[0].reminders[0].minutes, 10);
        verify(copied[0].recurrence === undefined);
    }

    function test_clipboardSkipsDuplicatesAndMalformedEntries() {
        const original = event();
        const text = EventUtils.clipboardText([original, original]) + "\r\nX-DANKCALENDAR-EVENT:not-json";
        const copied = EventUtils.clipboardEvents(text);

        compare(copied.length, 1);
    }

    function test_cutClipboardRetainsSourceForMoveAfterPaste() {
        const original = event({
            "recurringId": "series-1",
            "recurrence": ["FREQ=WEEKLY"]
        });
        const copied = EventUtils.clipboardEvents(EventUtils.clipboardText([original], true));

        compare(copied.length, 1);
        compare(copied[0]._dankCut.id, original.id);
        compare(copied[0]._dankCut.start, original.start.toISOString());
        compare(copied[0]._dankCut.recurrence[0], "FREQ=WEEKLY");
    }

    function test_pasteAnchorsGroupToTargetDay() {
        const first = EventUtils.createFields(event(), 0, "calendar-1", false);
        const second = EventUtils.createFields(event({
            "id": "event-2",
            "uid": "uid-2",
            "start": new Date(2026, 7, 12, 14, 0),
            "end": new Date(2026, 7, 12, 15, 0)
        }), 0, "calendar-1", false);
        const pasted = EventUtils.pasteFields([second, first], new Date(2026, 7, 20), "fallback");

        compare(new Date(pasted[0].start).getDate(), 20);
        compare(new Date(pasted[0].start).getHours(), 9);
        compare(new Date(pasted[1].start).getDate(), 22);
        compare(new Date(pasted[1].start).getHours(), 14);
    }

    function test_allDayCopyUsesDateValues() {
        const original = event({
            "allDay": true,
            "start": new Date(2026, 7, 10),
            "end": new Date(2026, 7, 11)
        });
        const text = EventUtils.clipboardText([original]);

        verify(text.indexOf("DTSTART;VALUE=DATE:20260810") !== -1);
        verify(text.indexOf("DTEND;VALUE=DATE:20260811") !== -1);
    }
}
