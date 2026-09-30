# Pferde, Pferdeakte, Gesundheit und Dateien

Tickets: JAN-10 (Pferde und Reitbeteiligungen), JAN-48 (Pferdeakte), JAN-49 (Notfallkarte),
JAN-12 (Termine und Erinnerungen), JAN-53 (Dokumente). Code: `backend/internal/horses`,
`backend/internal/health`, `backend/internal/files`, Migration `0040`, App: `mobile/app/horses/`,
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
| `GET /api/v1/horses` | all horses of the stable ordered by name: `id, name, box, sex, birth_year, breed, color_key, weight_kg, helper_note, owner{id,name,color_key}, riders[{user_id,name,color_key,rules}], is_mine, i_ride, can_manage, my_rules` |
| `GET /api/v1/horses/{id}` | same shape |
| `POST /api/v1/horses` | `name` required; `box, sex (mare/gelding/stallion), birth_year (1970..now), breed, color_key, weight_kg (20..1500), helper_note`, emergency fields, `owner_id` (admin) |
| `PATCH /api/v1/horses/{id}` | any of the above; emergency fields: `emergency_note, emergency_medication, permanent_medication, allergies, insurance, vet_name, vet_phone` |
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
| `report_observations` | may report observations (Auffälligkeiten) |
| `take_week_slots` | may take slots in the week plan |
| `hack_alone` | may hack out alone |
| `shows` | may ride the horse at shows |

Default for a new rider: `log_sessions`, `report_observations`, `take_week_slots`. Unknown rules give `400`.
The app has the same list in `mobile/lib/horse-format.ts` (`RIDER_RULES`, German labels).

## Emergency card (JAN-49)

`GET /api/v1/horses/{id}/emergency` (every member): `horse_id, horse_name, box, weight_kg, owner{id,name,phone}`
(phone from `users.phone`), `vet_name, vet_phone, emergency_note, emergency_medication,
permanent_medication, allergies, insurance, contacts[{id,label,name,phone}], can_manage`.
Extra contacts (owner or admin): `POST /api/v1/horses/{id}/emergency-contacts {label,name,phone}`,
`PATCH .../emergency-contacts/{contactId}`, `DELETE .../emergency-contacts/{contactId}`.

## Health items and reminders (JAN-12, `internal/health`)

| Route | Notes |
| --- | --- |
| `GET /api/v1/horses/{id}/health` | `{today, items, summary, can_manage}`; items ordered by due date (undated last); `summary` has the keys `vaccination, farrier, deworming, dentist`, each the item of that kind due first or `null`; every item has `days_until_due` (negative = overdue), counted from `today` in the stable's time zone |
| `POST /api/v1/horses/{id}/health-items` | `kind` (vaccination, farrier, deworming, dentist, physio, medication, vet), `label`, `due_date` (YYYY-MM-DD), `interval_days`, `note`, `daily_time` (HH:MM) |
| `PATCH /api/v1/health-items/{id}`, `DELETE /api/v1/health-items/{id}` | owner or admin |
| `POST /api/v1/health-items/{id}/done` | "erledigt": `due_date` moves forward by `interval_days`; if that is still not after today (very overdue), by further intervals; without a due date the interval counts from today; without interval the due date is cleared |

**Reminder job** (`health.Reminders`, scheduler job `health-reminders` in `cmd/api`, every 15 minutes):

- `health_due`: items with a due date and no `daily_time`, 7 days and 1 day before the due date, from 08:00
  stable-local time on. Text: "Fällig: Luna" / "Hufschmied ist in 7 Tagen fällig (08.10.2026).".
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
`GET .../documents/{docId}/file` (role check, accepts `?access_token=`). Kinds: `passport`,
`vaccination_record`, `insurance`, `other`. Flow in the app: pick a photo or PDF, `uploadFile()`, then
`POST .../documents`.

## Files (`internal/files`)

Shared storage for photos and documents; see also "Files" in `architecture.md`.

- Directory `REITERHOF_UPLOAD_DIR` (default `./uploads`, production `/var/lib/reiterhof/uploads`, see
  `deploy/api.env.example`), files at `<stable_id>/<32 hex random>.<ext>`.
- `POST /api/v1/files` (multipart, field `file`): JPEG, PNG, WebP, HEIC, PDF, at most 20 MB. The content is sniffed
  (a declared type that is not allowed is rejected, a wrong but allowed one is corrected), the client file name is
  never used. `201 {path, url, content_type, size}`; `413 file_too_large`, `415 unsupported_type`,
  `400 invalid_upload`. Caddy allows 25 MB bodies (`deploy/Caddyfile`).
- `GET /api/v1/files/{path...}`: members of the stable only (`401` without session, `404` for other stables and for
  every path that is not exactly `stable_id/random.ext`). Sent with `Content-Type` from the extension,
  `X-Content-Type-Options: nosniff`, `Cache-Control: private`.
- `?access_token=<session token>` is accepted on file downloads for consumers that cannot set headers. Tradeoff:
  the full session token ends up in the URL and can appear in proxy or access logs and history. Use
  `Authorization` where possible (React Native `<Image source={{ uri, headers }}>` works). Signed short-lived URLs
  would be the next step.
- The stable check is the only access check of `/files`. File names are unguessable, but anybody in the stable who
  knows a path can load it; where the path is sensitive (horse documents) the domain package serves the file
  through its own role-checked route and never hands out the raw path.

## App

- Tab "Pferde": hero with the number of horses, "Meine Pferde", "Ich reite", then the rest, button "Pferd anlegen".
- Horse record `/horses/[id]`: header, emergency card preview, four health tiles, "Auffälligkeiten" (placeholder
  `components/horse-observations.tsx`, the extension point for the observations feature), riders with rule
  toggles (owner/admin), documents, links "Trainingsprofil", "Deckenplan", "Reha"
  (`horseRoutes` in `lib/horse-format.ts`).
- Sub screens: `emergency` (tap to call), `health` (due list, add/edit sheet, "Erledigt"), `documents`
  (photo/PDF upload), `edit`, and `/horses/new`.
- Upload helper `lib/upload.ts`: `uploadFile({uri, name, mimeType, size})`, `fileSource(url)`,
  `fileUrlWithToken(url)`, `uploadErrorMessage(err)`.

## Seed

`internal/seed/horsecare.go` gives the demo horses palette color keys, breed and weight, emergency data for
Luna and Fanta, three contacts, nine health items relative to the current date (overdue, soon, later, daily
medication), and phone numbers for Jan and Anna.
