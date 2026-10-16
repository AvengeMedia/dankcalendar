# Opening calendar files

Open an `.ics` or `.vcs` file with Dank Calendar in your file manager, or run:

```sh
dcal open invitation.ics appointment.vcs
dcal open 'file:///home/me/Team%20meeting.ics'
dcal open webcals://example.com/calendar.ics
```

The installed desktop entry advertises `text/calendar`, `text/x-vcalendar`,
`application/ics`, `webcal`, and `webcals`. It accepts multiple URLs with `%U`.
Installation updates the desktop database; choosing Dank Calendar as the default
handler remains a desktop preference. Flatpak exports the same desktop entry and
forwards selected files through its document portal.

Linux and FreeBSD source installs use `make install`. Their release tarballs ship
`scripts/install-release.sh`, which installs the binary and the same desktop entry,
icon, and AppStream metadata. Nix and Flatpak packages install the desktop entry
as part of their normal package output.

## Event imports

A normal file opens a preview with a destination calendar picker. Only writable
calendars that support events appear. If none exist, add a calendar and return to
the same import. Nothing is written until **Import** is pressed.

Each event shows its title, location, time, and overlapping busy events with their
calendar names and times. The preview uses currently synced data across calendars,
including recurring occurrences and all-day events. Adjacent events and events
marked free or cancelled do not conflict. Overlaps are advisory; importing is
still allowed. Up to 20 overlaps are displayed per event, with the total count.

Recurring imports check one year from their first occurrence; the preview states
the ending date. Recurrence exceptions are preserved as exclusions plus standalone
replacement events, so moved occurrences retain their time across providers.
Cancelled exceptions only exclude the original occurrence.

An event already present in the selected destination is skipped. Changing the
destination refreshes duplicate detection and the preview. Replies and
cancellations offer the existing meeting instead of creating a new event.
Importing a private copy does not send an invitation response.

Multiple opened files are reviewed in order. Cancelling one advances to the next.
If an import fails partway through, the dialog remains open and refreshes its
preview; retry skips events already stored in the destination.

## Subscriptions

`webcal`, `webcals`, HTTP, and HTTPS links open subscription setup. A file with a
calendar-level `SOURCE` or `X-WR-CALURL` also opens subscription setup instead of
an event conflict preview. Scheduling invitations remain event imports even if
they carry source metadata. Neither an event's `URL`, the number of events, nor a
recurrence rule alone makes a file a subscription.

The source can be reviewed before adding the subscribed calendar. Existing feed
sync manages subsequent refreshes. A local event file without a source is a
snapshot, so it cannot supply subscription updates.

`SOURCE` follows [RFC 7986 section 5.8](https://www.rfc-editor.org/rfc/rfc7986.html#section-5.8).
Desktop associations follow the [Desktop Entry specification](https://specifications.freedesktop.org/desktop-entry/latest-single/).

## Legacy compatibility

Both extensions accept iCalendar 2.0. vCalendar 1.0 additionally supports
quoted-printable text, declared character sets, fixed-offset `TZ`, stable generated
IDs for UID-less events, and daily, weekly, monthly and yearly recurrence forms.
Legacy `DAYLIGHT` rules and bounded recurrence rules containing multiple days are
rejected with an explicit request for an iCalendar 2.0 export rather than changing
the schedule silently. Calendar files are limited to 256 KiB.

## Manual verification

Use a temporary local account and calendars; no network account is needed.

1. Place a busy event at 12:30–13:30 and open an invitation for 12:00–13:00.
   Confirm the preview names the overlap and its calendar; import, then reopen
   the file to verify it is marked as already present.
2. Switch to another writable calendar and verify import becomes available after
   the new preview completes. Repeat with calendars having the same name.
3. Check an adjacent 13:30 event, a free event, an all-day event, and an occurrence
   of a recurring event. Verify busy overlaps only and exclusive end boundaries.
4. Open two files together with the app stopped, then while it is running. Cancel
   the first and confirm the second is presented without overwriting the first.
5. Open a `.vcs` file with `VERSION:1.0`, quoted-printable text, and a `TZ` offset.
   Verify the title and local time before and after import.
6. Open a file containing `SOURCE:https://example.com/feed.ics`: verify subscription
   setup appears. The same metadata in `METHOD:REQUEST` must show event import.
7. Check an empty file, malformed data, a directory, an oversized file, and a
   remote-host `file://` URL. Confirm an error and no calendar mutation.
8. In native and Flatpak installs, choose **Open With → Dank Calendar** on files
   with spaces in their names. For Flatpak, verify the exported desktop entry
   forwards file URLs and does not require general home-directory access.
