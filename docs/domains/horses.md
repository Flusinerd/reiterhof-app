# Pferde, Pferdeakte, Gesundheit und Dateien

Tickets: JAN-10 (Pferde und Reitbeteiligungen), JAN-48 (Pferdeakte), JAN-49 (Notfallkarte),
JAN-12 (Termine und Erinnerungen), JAN-53 (Dokumente). Code: `backend/internal/horses`,
`backend/internal/health`, `backend/internal/files`, `backend/internal/diary`, Migrations `0040` and `0310`, App: `mobile/app/horses/`,
`mobile/app/(tabs)/horses.tsx`, `mobile/components/horse-*.tsx`, `mobile/lib/api/horses.ts`,
`mobile/lib/upload.ts`.

All routes need a signed-in user with a stable (`auth.RequireStable`). A horse of another stable
answers `404 not_found`, a forbidden change `403 forbidden`, invalid input
`400 validation_failed` (see "Error format" in `architecture.md`). Text fields: `""` clears a value on
PATCH, numbers: `0` clears.

## Permissions

| Action | Who |
| --- | --- |
| list/read horses, emergency card, health items | every member of the stable |
| create a horse | every member (the creator becomes owner; only admins may set `owner_id`) |
| change a horse, emergency data and contacts, riders, health items | owner or admin (`auth.CanManageHorse`) |
| change the owner (`owner_id`) | admin only |
| read documents (list and file) | owner, riders of the horse, admins |
| write documents | owner or admin |

Decision for documents (JAN-53): passport, vaccination record and insurance papers contain personal
data, so plain members do not see them (`403`). The list returns a per-horse file URL that repeats
the check.

## Horses (`internal/horses`)

| Route | Notes |
| --- | --- |
| `GET /api/v1/horses` | all horses of the stable ordered by name: `id, name, box, sex, birth_year, breed, color_key, weight_kg, helper_note, photo_url, owner{id,name,color_key}, riders[{user_id,name,color_key,rules}], is_mine, i_ride, can_manage, my_rules` (`photo_url` is `null` without a photo) |
| `GET /api/v1/horses/{id}` | same shape |
| `POST /api/v1/horses` | `name` required; `box, sex (mare/gelding/stallion), birth_year (1970..now), breed, color_key, weight_kg (20..1500), helper_note`, emergency fields, `photo_path`, `owner_id` (admin) |
| `PATCH /api/v1/horses/{id}` | any of the above; emergency fields: `emergency_note, emergency_medication, permanent_medication, allergies, insurance, vet_name, vet_phone`; `photo_path` (see Photo) |
| `GET /api/v1/horses/{id}/diary` | see Diary |
| `PUT /api/v1/horses/{id}/riders/{userId}` | body `{rules: [...]}` (omitted: defaults); replaces the rules; the owner cannot be a rider |
| `DELETE /api/v1/horses/{id}/riders/{userId}` | `204` |
| `GET /api/v1/members` | `[{id, name, color_key, is_me}]` for pickers |

### Rider rules (positive list)

A rider may only do what is on the list. Owners and admins always may everything. The constants live in
`horses/rules.go` (`horses.Rule*`), other packages read them with `auth.RiderRules`.

| Rule | Meaning |
| --- | --- |
| `ride` | may ride the horse (from the seed data) |
| `groom` | may groom and care for the horse (from the seed data) |
| `log_sessions` | may log training sessions |
| `report_observations` | may report observations (Auffälligkeiten); not enforced, every member may report, see [observations.md](observations.md) |
| `take_week_slots` | may take slots in the week plan |
| `hack_alone` | may hack out alone |
| `shows` | may ride the horse at shows |

Default for a new rider: `log_sessions`, `report_observations`, `take_week_slots`. Unknown rules give `400`.
The app has the same list in `mobile/lib/horse-format.ts` (`RIDER_RULES`, German labels).

## Photo

Optional (migration `0310`, `horses.photo_path`). The app uploads an image with `POST /api/v1/files` and sends the returned
`path` as `photo_path` in `PATCH /api/v1/horses/{id}` (owner or admin like every horse field); `""` removes the photo. The path
must be an existing JPEG, PNG or WebP of the caller's stable (`400 validation_failed` otherwise, a PDF is refused). The horse JSON
carries `photo_url` (`/api/v1/files/<path>`, `null` without photo); every member of the stable may load the file, like blanket
photos. When the photo is replaced or removed, the old file is deleted unless another record (document, observation, blanket,
horse) still points to it (`files.Referenced`). There is no route that deletes a horse, so there is no file clean-up on deletion.
A horse photo shows the stable's animal, not a person; it needs no entry of its own in the processing register. The privacy export
(`to_jsonb(horses)` of owned horses) contains `photo_path`. Without a photo the app shows the horse's color with its initial.

## Diary (`internal/diary`)

`GET /api/v1/horses/{id}/diary?before=<RFC 3339>&limit=<1..100, default 30>`: training sessions and observations of the horse in
one list, newest first. Answer: `{items, next_before}`; `next_before` is the `before` of the next page (`null` at the end), so the
app loads older entries with "Mehr laden". An item has `type` (`session` or `observation`), `id`, `at` (UTC), `day` (the
stable's local calendar day), `user{id,name,color_key}` and exactly one of `session{activity, label, minutes, note, feel,
exercise_title}` or `observation{category, body_part, description, urgency, status, media_url (first photo), media_count}`.

Who sees what: observations every member of the stable (like `GET /horses/{id}/observations`); sessions follow the training rule
(owner and admin all, riders the sessions marked visible plus their own, other members none). A horse of another stable is a `404`.
Health items keep no history (marking one done moves its due date), so they are not part of the diary.

## Emergency card (JAN-49)

`GET /api/v1/horses/{id}/emergency` (every member): `horse_id, horse_name, box, weight_kg, owner{id,name,phone}`
(phone from `users.phone`), `vet_name, vet_phone, emergency_note, emergency_medication,
permanent_medication, allergies, insurance, contacts[{id,label,name,phone}], can_manage`.
Extra contacts (owner or admin): `POST /api/v1/horses/{id}/emergency-contacts {label,name,phone}`,
`PATCH .../emergency-contacts/{contactId}`, `DELETE .../emergency-contacts/{contactId}`.

## Health items and reminders (JAN-12, `internal/health`)

| Route | Notes |
| --- | --- |
| `GET /api/v1/horses/{id}/health` | `{today, items, summary, can_manage}`; items ordered by due date (undated last); `summary` has the keys `vaccination, farrier, deworming, dentist`, each the item of that kind due first or `null`; every item has `days_until_due` (negative = overdue), counted from `today` in the stable's time zone; items of the kinds `vet`, `vaccination`, `dentist` that are due soon also carry `bundle {kind, count, horses[{id,name}]}` (other horses with the same kind due within 14 days, see [observations.md](observations.md#bundling-appointments-jan-54)) |
| `POST /api/v1/horses/{id}/health-items` | `kind` (vaccination, farrier, deworming, dentist, physio, medication, vet), `label`, `due_date` (YYYY-MM-DD), `interval_count` + `interval_unit` (`day`, `week`, `month`, `year`; at most about ten years, `0` clears), `note`, `daily_time` (HH:MM), `remind_days` (0..365 days before the due date, at most 5, default `[7, 1]`, `[]` = no reminder), `remind_time` (HH:MM, default `08:00`). The older `interval_days` is still accepted (= `interval_count` with unit `day`) and returned for day and week intervals (else `null`). |
| `PATCH /api/v1/health-items/{id}`, `DELETE /api/v1/health-items/{id}` | owner or admin |
| `POST /api/v1/health-items/{id}/done` | "erledigt": `due_date` moves forward by the interval; if that is still not after today (very overdue), by further intervals; without a due date the interval counts from today; without interval the due date is cleared. Months and years keep the day of the month and fall back to the last day of a shorter month (31.08. + 6 months = 28.02.), always counted from the old due date, so the day does not drift (`health.Interval.After`) |

**Reminder job** (`health.Reminders`, scheduler job `health-reminders` in `cmd/api`, every 15 minutes):

- `health_due`: items with a due date and no `daily_time`, on each of the item's `remind_days` before the due date
  (default 7 and 1, `0` = on the day), from its `remind_time` (default 08:00) stable-local time on until the end of that
  day. Text: "Fällig: Luna" / "Hufschmied: in 7 Tagen (08.10.2026)." ("morgen", "heute"). Migration `0260` turned
  `interval_days` into `interval_count` + `interval_unit` (whole years and weeks converted) and added the reminder columns.
- `medication`: items with `daily_time`, daily from that time for up to 3 hours (a missed reminder after longer
  downtime is skipped, not sent late). `due_date` of such an item is the last day of the course.
- Recipients: owner and all riders of the horse; users can switch a kind off in `reminder_settings`
  (handled by `push.Notifier`).
- Idempotency: every reminder is claimed by a row in `reminders` (`source_table = 'health_items'`,
  unique index `reminders_health_once` on user, kind, item and scheduled time, migration `0040`). Running the
  job repeatedly sends nothing twice; after "erledigt" the new due date starts a new cycle. If the push fails,
  the claim is released and the next run retries.

Farrier items show "Termin begleiten" in the app, which opens
`/requests/new?type=appointment_companion&horse=<id>` (screen of the requests feature).

## Documents (JAN-53)

`GET /api/v1/horses/{id}/documents`, `POST` (`{kind, title, file_path}`, `file_path` comes from the files API of
the same stable), `PATCH .../documents/{docId}` (`kind`, `title`), `DELETE` (also removes the file),
`GET .../documents/{docId}/file` (role check; viewers that cannot send headers use a download link, see Files). Kinds: `passport`,
`vaccination_record`, `insurance`, `other`. Flow in the app: pick a photo or PDF, `uploadFile()`, then
`POST .../documents`.

## Files (`internal/files`)

Shared storage for photos and documents; see also "Files" in `architecture.md`.

- Directory `REITERHOF_UPLOAD_DIR` (default `./uploads`, production `/var/lib/reiterhof/uploads`, see
  `deploy/api.env.example`), files at `<stable_id>/<32 hex random>.<ext>`.
- `POST /api/v1/files` (multipart, field `file`): JPEG, PNG, WebP, PDF, at most 20 MB. The content is sniffed
  (a declared type that is neither allowed nor some `image/*` is rejected; a wrong one is corrected, since pickers
  report the type of the original photo: Android's image picker re-encodes a HEIF photo to JPEG and still says
  `image/heif`), the client file name is never used. Images are stored without metadata (`metadata.go`: EXIF, XMP,
  IPTC, comments, unknown application segments, PNG text and time chunks, WebP EXIF/XMP chunks, trailing data); only
  a JPEG's EXIF orientation is carried over, alone in a fresh EXIF block, so upright phone photos do not show up on
  their side. A damaged container is refused. HEIC is recognised but refused (its metadata cannot be rewritten; the
  app asks the iOS picker for the compatible representation, which is JPEG). `201 {path, url,
  content_type, size}` (the size after stripping); `413 file_too_large`, `415 unsupported_type`, `400 invalid_upload`
  (malformed upload or damaged image). Caddy allows 25 MB bodies (`deploy/Caddyfile`).
- `GET /api/v1/files/{path...}` follows the record that references the file (least privilege): a
  `horse_documents.file_path` only for the owner, riders and admins of the horse, an `observations.media` entry or a
  `blankets.photo_path` or a `horses.photo_path` for every member of the stable. `401` without session, `404` for other stables, malformed
  paths and files nothing points to (never attached, or detached since). Sent with `Content-Type` from the
  extension, `X-Content-Type-Options: nosniff`, `Cache-Control: private`.
- Session tokens are never read from the URL. Viewers that cannot send headers (Linking.openURL of a PDF or a full-size
  photo) ask `POST /api/v1/files/download-link {url}` for `{url: "<same path>?dl=<token>", expires_at}`: the token is
  bound to the user and the exact path, expires after five minutes (`files.DownloadLinkTTL`) and is signed with a key
  derived from `REITERHOF_LOGIN_CODE_KEY` (random per process without it). `files.DownloadLink(deps)` wraps the two
  file routes and puts the user into the context; the role checks run as usual. Everything else uses `Authorization`
  (React Native `<Image source={{ uri, headers }}>` works).

## App

- Horse list `/horses` (also the horse tab for users with several horses): header with the number of horses and an outlined
  "Pferd anlegen", then "Meine Pferde", "Ich reite" and the rest as hairline rows (`ListRow`, [design-system.md](../design-system.md))
  with the photo as avatar.
- Horse page `/horses/[id]` (`HorseRecord`, also the horse tab for the user's only horse, named after it), built from
  `components/horse-hero.tsx`, `horse-today-rows.tsx`, `horse-week-strip.tsx`, `horse-diary.tsx`, `horse-health-list.tsx`,
  `horse-file-tab.tsx`; the logic is in `lib/horse-page.ts` (tested):
  - Hero, full-bleed: with a photo it fades into the background, without one the horse's color is a field with its big italic initial
    and "+ Foto hinzufügen" (owner and admin only; with a photo they get a small camera button). The photo sheet
    (`components/horse-photo-sheet.tsx`) takes or picks a photo (photos consent), `uploadFile()`, then PATCH `photo_path`; it can remove it.
    Eyebrow "Stute · 11 Jahre · Box 12", the name in display italic, below it "Dein Pferd · Hannoveraner".
  - "Heute für <Name>": up to three rows with hairlines, each with one action: training today from the week plan ("Starten" when it is the
    user's, one primary button), tonight's blanket ("Ändern" for owner and admin, opens the blanket choice), the next health item
    ("Erledigt" for owner and admin when overdue or due within a week). Empty rows are left out; the training row is only for owner, admin
    and riders (the week endpoint says `403` to others).
  - "Diese Woche": seven marks (done, planned, show day in brass, rest, empty), tap opens `/training/week?horse=`.
  - `Segmented` with Tagebuch (the diary, "Mehr laden", notes as italic quotes, observation photos as thumbnails, "Auffälligkeit melden"),
    Gesundheit (compact health list, "Alle Termine", the observations with "Reha-Plan erstellen" for owner and admin, see
    [observations.md](observations.md) and [reha.md](reha.md)) and Akte (emergency card preview, riders with rule toggles, documents,
    links "Trainingsprofil", "Deckenplan", "Reha", helper note, "Pferd bearbeiten"; `horseRoutes` in `lib/horse-format.ts`).
  - Stored images need the session token: `components/auth-image.tsx` sends it as a header (native), `auth-image.web.tsx` fetches the file
    with the header and shows a blob URL (React Native Web ignores `source.headers`). `Avatar` takes `photoUrl`.
- Sub screens are rows with hairlines, no card stacks: `emergency` (tap a row to call; the number big in Fraunces brass, the
  preview in the Akte tab shows the same), `health` (rows with the kind as eyebrow, the due time right in brass, danger when overdue,
  "Erledigt" as a quiet button; due list, add/edit sheet with date picker, repeat as one-tap presets
  per kind such as "6 Monate" / "1 Jahr" or "Eigenes" with number and unit, reminder days as chips "4 Wochen vorher" …
  "am Tag" plus the time; "Erledigt"), `documents`
  (photo/PDF upload), `edit`, and `/horses/new`.
- Upload helper `lib/upload.ts`: `uploadFile({uri, name, mimeType, size, file})`, `fileSource(url)`,
  `openStoredFile(url)` (download link, then `Linking.openURL`), `uploadErrorMessage(err)`. Every picker call uses
  `PHOTO_PICKER_OPTIONS` (images, iOS "compatible" representation, so no HEIC) and turns the asset into the upload
  input with `pickedFile(asset, fallbackName)`, which keeps the web `File`. `uploadFile` declares the type from the
  extension of the file it sends, not from the picker's label, and appends the file the way the platform needs
  (`upload-core.ts`, `uploadSource`): native `{uri, name, type}` objects, on the web the `File` (or the blob: URL
  fetched into a `Blob`); a browser turns the native object into the text "[object Object]".

## Seed

`internal/seed/horsecare.go` gives the demo horses palette color keys, breed and weight, emergency data for
Luna and Fanta, three contacts, nine health items relative to the current date (overdue, soon, later, daily
medication), and phone numbers for Jan and Anna.
