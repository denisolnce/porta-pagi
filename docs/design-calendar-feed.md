# The calendar as a feed

Built 2026-11-03 on the owner's ask: "check if to the calendar it's possible
to add a subscription, basically, so that people could click somewhere there
and subscribe to this calendar in their Google calendars or their Apple
calendars, in the nicest way for it to integrate into their own calendars as
well."

**The villager job, in one line: "see Saturday's work party in the calendar
I already look at, beside the dentist, without opening the portal."**

## As built

- **One link per house** (`calendar_feeds`, migration `031`), made the first
  time anyone of the house opens 📅 *Add to your calendar* under the
  calendar's cards (`Subscribe` in `rooms/Calendar.tsx`): `GET
  /api/calendar/feed` only reads it, and the panel `POST`s the first one. The link is
  `<the portal>/api/ical/<secret>.ics`; the secret is 24 random bytes, kept
  plain like an invite because the house reads it again whenever a member
  adds the calendar. ♻ *New link* replaces it, and the old one answers 404 in
  every calendar that held it. One per person was offered and refused (owner,
  2026-11-03): more to manage, for a house that is one account anyway.
- **Each device sees the doors that open there.** *Apple Calendar*, on an
  iPhone, an iPad or a Mac only, opens the same address as `webcal://`, which
  they hand to their Calendar's subscribe sheet; elsewhere nothing handles
  it. *Google Calendar*, on a computer only, opens Google's own "add by URL"
  page with the address (`calendar.google.com/calendar/r?cid=…`), in a new
  tab with no referrer; on a phone (a coarse pointer) the panel says the
  words instead — Settings, Add calendar, From URL on a computer — because
  Google's phone app cannot add one. *Copy link* is the plain https address,
  for Outlook, Thunderbird or Google by hand, drawn where the browser has a
  clipboard; the address also sits in a read-only field, for a thumb in a
  browser without one (WhatsApp's own).
- **What the feed carries** (`ical.go`, RFC 5545 by hand, no dependency):
  events and work parties only — the owner's pick — sixty days back and every
  series, the window the calendar room reads. Per event: the title (a work
  party with 🤝 before it), the place, the notes, the crest and name of the
  house that called it, and a link back to the event in the portal. Times on
  the village's clock: `TZID=<TZ>` with a `VTIMEZONE` built from Go's zone
  data, so a weekly 09:00 stays 09:00 through the clock change; a village
  with no `TZ` sends floating times, read in each subscriber's own zone.
  A series goes out as one `RRULE`, the page's rule said in RFC terms (a
  monthly 31st and a yearly 29 February land on the last day of a shorter
  month: `BYMONTHDAY=d,-1;BYSETPOS=1`). `SEQUENCE` is the event's
  `time_version`, so a moved event moves in the calendar too.
- **What it never carries**: away notices (CLAUDE.md § Nothing is public, "no
  digests, no feeds"), birthdays (a person's data — the owner left them out),
  the sky, answers or a headcount.
- **Not in the demo**: its village is wiped on the hour, and every calendar
  that subscribed would hold a dead link.

## What it costs

- **A link is a credential that cannot be taken back from a copy.** A
  calendar cannot sign in, so whoever holds the address reads the village's
  events; ♻ ends it for everyone, and every house member then adds the new
  one. The panel says to keep it in the house.
- **Google keeps a copy.** It fetches the feed from its own servers, every
  few hours to a day at its own pace, and stores what it read. That is the
  house's choice, as an MCP key's provider is (CLAUDE.md § A door for
  agents); the panel says so, and the privacy note owes the village a line
  (`docs/later.md`). An iPhone or a Mac reads the feed itself.
- **Google's phone app cannot add a calendar by link.** The *Google
  Calendar* button works on a computer; once added, the calendar shows on the
  phone too. The panel says so.
- **Late.** An event moved at noon may show the old time in Google until
  evening. The portal stays the place that is right.

## Not verified here

The feed was checked against an independent parser (Python `icalendar` and
`recurring-ical-events`): the monthly 31st lands on 30 November and 28
February, a series keeps its wall-clock hour across the March change. Not
seen from the building session, whose network reaches neither: Google's
*add by URL* page taking the `webcal://` address, and an iPhone's subscribe
sheet. Owed: one subscription each, from the owner's phone and a computer,
after the roll.
