# Privacy and consents (JAN-19, JAN-85, JAN-86)

Backend: `backend/internal/privacy`, migrations `0110_privacy.up.sql`, `0200_maps_consent.up.sql`,
`0210_age_confirmation.up.sql`; age confirmation in `backend/internal/auth/agegate.go`. Mobile: `app/settings/privacy.tsx`,
`app/legal/{privacy,licenses}.tsx`, `app/(auth)/age.tsx`, `lib/consent*.ts`, `lib/api/privacy.ts`, `components/consent-*.tsx`,
`components/map-consent-gate.tsx`.
Text: `docs/legal/datenschutz.md` (German draft, **Entwurf – vor Veröffentlichung rechtlich prüfen lassen**). No imprint:
the app is private and invite-only (owner's decision, 30.09.2026); the privacy text names the controller and the contact.
What the owner must sign and fill in: [docs/legal/avv-checkliste.md](../legal/avv-checkliste.md).

## Consents

Table `consents(user_id, stable_id, kind, version, granted_at, revoked_at)`, unique per `(user_id, kind)`:
a grant upserts the row (new version and time, `revoked_at` cleared), a revocation sets `revoked_at`. A kind
without a row was never granted. `stable_id` is nullable (a person may consent before joining a stable).

| Kind | Meaning | Server-side effect |
| --- | --- | --- |
| `location_geofence` | background location for "Automatisch erkennen" | `POST /presence/check-in` with `source: "geofence"` is refused with `403 consent_required` without a grant (`internal/presence`) |
| `location_tracking` | recording the GPS track of a training session | `POST /horses/{id}/sessions` with a `track` is refused with `403 consent_required` (`internal/trainingapi`) |
| `maps` | loading map tiles from Apple (iOS), Google (Android) or OpenFreeMap (web), which see the viewer's IP address and the map area | none; the app loads a map only with the grant (`MapConsentGate`) |
| `presence_sharing` | others may see that I am at the stable | revoking sets `users.presence_visibility` to `hidden` |
| `photos` | camera and photo library | none (device permission plus explanation) |
| `push` | push notifications | revoking deletes the user's `push_tokens` and `web_push_subscriptions`; `push.Notifier` sends only to users with a current grant (Expo and web alike) |

`privacy.TextVersion` (`"2026-09-30"`) is the version of the privacy text; a grant stores it. Changing the texts
means raising both `TextVersion` and the `**Textversion:**` line in `docs/legal/*.md` (a test checks this); existing
grants then show `up_to_date: false` and the app asks again, but nothing is locked.

**Age (Art. 8 GDPR):** a grant needs a confirmed age (`auth.AgeConfirmed`: the person stated to be 16 or older,
or a parent consented), else `403 age_unconfirmed`. Revoking always works. See "Age confirmation" below.

| Route (signed-in user, stable **not** required) | Notes |
| --- | --- |
| `GET /api/v1/me/consents` | `{current_version, items: [{kind, granted, version, granted_at, revoked_at, current_version, up_to_date}]}`, always all six kinds in the order above |
| `PUT /api/v1/me/consents/{kind}` | body `{granted: bool, version?: string}`; `version` is the text the app shows, a different one on a grant gives `409 version_mismatch`; unknown kind `404`; `403 age_unconfirmed` (grant only); returns the item |

For other packages: `privacy.Has(ctx, q, userID, kind)` tells whether a grant exists and is not revoked.

## Age confirmation and parental consent (JAN-86)

`backend/internal/auth/agegate.go`, columns `users.age_confirmed_at`, `users.parent_email`,
`users.parental_consent_at`, table `parental_consent_tokens` (token hash, parent email, 7 days, single use). No birth
date is stored. Nobody but the person and the parent is involved, so a stable stays self-administered.

| Route | Notes |
| --- | --- |
| `POST /api/v1/me/age` `{over_16: true}` | sets `age_confirmed_at`, clears a pending `parent_email`; returns `me` |
| `POST /api/v1/me/parental-consent` `{parent_email}` | stores the address, clears `age_confirmed_at`, revokes older links, mails the parent (`auth.MailContent`, link `<REITERHOF_PUBLIC_URL>/parental-consent?token=...`); `400` for the own address, `409 already_confirmed`, `429` after 3 mails per user and day, `502 mail_failed` |
| `GET /parental-consent?token=` | HTML page (own CSP like `/auth/verify`): child's name and email, what the app stores, link to `<REITERHOF_WEB_URL>/legal/privacy`, one form button; reading consumes nothing |
| `POST /parental-consent` (form `token`) | consumes the token (once), sets `parental_consent_at`; 20 per IP and 15 minutes; a "Danke" page |

`GET /api/v1/me` carries `age_status` (`unknown`, `parent_pending`, `confirmed`) and `parent_email`. Enforced
server-side: `POST /stables/join` and consent grants answer `403 age_unconfirmed` until confirmed. Seed users and
operator-created accounts: the seed sets `age_confirmed_at`; `stallfunk-admin user create` does not, the person
states it at the first start. App: `app/(auth)/age.tsx` after the name screen ("16 oder älter", or the parent's
address and a waiting screen that refetches `me` every 15 s); the root layout keeps the person there, the privacy
settings and the legal texts stay reachable.

## Data subject rights

**Export** `GET /api/v1/me/export` (attachment `reiterhof-export-<date>.json`, `Cache-Control: no-store`). All
personal data of the caller across tables, as whole table rows (`to_jsonb`, so new columns appear automatically):
`profile` (including the age statement and the parent's address), `stable`, `sign_in_identities`, `sign_in_sessions`
(no token hashes), `sign_in_attempts` (login tokens of the address, no hashes), `parental_consent_requests` (no hashes),
`consents`, `push_tokens` (masked to first and last four characters), `web_push_subscriptions` (push service host, user
agent and date only; the endpoint URL is a secret), `reminder_settings`, `reminders`, `presence_visits`,
`training_sessions` (including GPS tracks), `gait_windows` (raw sensor features of own rides), `week_slots`,
`observations_reported`, `requests_created`, `requests_helped`, `horses_owned`, `horse_rider_roles`, `blanket_changes`,
`reha_days_done`, `reha_plans_created`, `invites_created` (without the code), `documents_uploaded` and `uploaded_files`
(paths of documents and observation photos). Only rows the user owns or authored; rows of other people are never
included. **When you add a table that references `users`, add a section to `privacy.Export` and to
`privacy.DeleteAccount`.**

**Deletion** `POST /api/v1/me/delete {"confirm": true}` → `204`.

Decisions:

- **Owned horses block the deletion** (`409 owns_horses`, body lists `horses: [{id, name}]`). The alternative
  (keep the horse without owner) would leave the emergency card without a responsible person and the riders
  without someone who manages them. An admin changes the owner (`PATCH /horses/{id} owner_id`) or the horse is deleted;
  then deletion works.
- **The last admin of a stable is blocked** (`409 last_admin`), so a stable is never left without administration.
- **The `users` row is anonymised, not deleted**, because requests, training sessions, observations and week slots of a
  horse reference it (`NOT NULL` foreign keys) and the horse's history should survive: `name = "Gelöschtes Mitglied"`,
  `email = deleted-<id>@deleted.invalid` (the address can be registered again), `phone`, `avatar_color`, `stable_id`,
  `age_confirmed_at`, `parent_email`, `parental_consent_at` cleared, `is_admin = false`, `deleted_at` set. Without a
  stable the row does not show up in any member list.
- **Deleted:** sessions, sign-in identities, login tokens for the email, parental consent tokens, push tokens, web push
  subscriptions, reminder settings, reminders, presence visits, rider roles, helper assignments (a request that was
  `assigned` only to this user is `open` again), consents, GPS tracks (`sessions.track`) and raw gait windows
  (`gait_windows`), the media of reported observations (files are removed from disk after the commit; photos may show people).
- **Kept, now attributed to "Gelöschtes Mitglied":** observation text, training sessions (without track), requests
  (open and assigned ones are `cancelled`), week slots, document metadata, reha plans.
- Publishes `presence.changed` for the stable. Users without a stable can export and delete too.
- Backups still contain the data until they expire (14 daily, 8 weekly on the server; no offsite copy for now);
  documented in the privacy text. After a restore, repeat deletions.

## Retention

Scheduler job `privacy-retention`, daily 03:30 Europe/Berlin (`privacy.Job`, `privacy.Prune`; constants in
`retention.go`):

| Data | Kept | Then |
| --- | --- | --- |
| finished presence visits (`presence.left_at`) | 12 months | deleted (open visits are closed by `presence-close-stale` after 12 hours) |
| `sessions.track` (raw GPS) | 12 months after `started_at` | set to `NULL`, the session (duration, distance, gait shares) stays |
| `gait_windows` (raw gait features) | 12 months after the session's `started_at` | deleted |
| `reminders` | 12 months after `due_at` | deleted |
| `login_tokens` | until `expires_at` (15 minutes) | deleted when expired |
| `auth_sessions` | 90 days sliding | deleted when expired |
| `stable_invites` | 30 days after `expires_at` (`InviteGraceDays`) | deleted (they name their creator) |
| `parental_consent_tokens` | until `expires_at` (7 days) | deleted when expired |
| Backups | 14 daily + 8 weekly on the server; no offsite copy for now (`deploy/backup.sh` with `REITERHOF_SKIP_OFFSITE=1`; an offsite target must be an encrypting `crypt` remote) | deleted |

Raw gait windows (`gait_windows`, stored with tracked sessions to improve the gait model) follow the track: deleted
12 months after the session's `started_at`, and immediately when the account is deleted. Everything else (horse record, documents, requests) is kept until a user with
the right deletes it or the account is deleted.

## Files: metadata, least privilege, download links

`internal/files` (JAN-85): uploaded JPEG, PNG and WebP images are rewritten without metadata (EXIF with the GPS
position, XMP, IPTC, comments, vendor blocks, thumbnails, trailing data such as Motion Photo videos); a damaged
container is refused (`400 invalid_upload`), HEIC is not accepted at all (it cannot be cleaned), PDFs stay as they are.
`GET /api/v1/files/{path}` only serves files a record references, with that record's visibility (horse documents:
owner, riders, admins; observation and blanket photos: members); anything else is a 404. Session tokens are never
read from a URL; viewers that cannot send headers get a five-minute, path-bound link from
`POST /api/v1/files/download-link` (`openStoredFile()` in the app). Details in [horses.md](horses.md#files-internalfiles).

## Texts and how the app shows them

`docs/legal/datenschutz.md` is the **single source**. The app bundles it as
`mobile/lib/consent-legal-texts.ts`, generated by `node scripts/gen-legal.mjs` (the file says "GENERATED";
`lib/consent-legal.test.ts` fails while it is out of date, `--check` does the same on the command line). After editing
the text: run the script, raise `**Textversion:**` and `privacy.TextVersion` together, commit all of it.
The texts use only headings (`#` to `###`), paragraphs, `- ` lists, `> ` quotes and `**bold**`
(`lib/consent-markdown.ts` renders exactly that; the test rejects tables, code and links).
The consent explanations of the sheets (short, plain German) are UI copy in `lib/consent-core.ts` (`CONSENT_COPY`).

Open-source notices (JAN-87): `mobile/scripts/gen-licenses.mjs` writes `lib/licenses.generated.ts` (git-ignored) at
every `npm ci` from `package-lock.json` and the installed packages; a package with a license outside its allow-list or
without a usable text fails the install, and with it CI and the deploy. `app/legal/licenses.tsx` shows them.

## App

- `useConsentPrompt()` (`components/consent-prompt.tsx`): `if (!(await consent.ensure("photos"))) return;` shows the
  explanation sheet ("Erlauben", "Nicht jetzt", "Mehr erfahren" → `/legal/privacy`) when the consent is missing,
  revoked or for an old text. Render `{consent.sheet}` in the screen. Gated: GPS ride tracking (`location_tracking`, before the OS permission; the server answers
  `403 consent_required` to `POST /horses/{id}/sessions` with a `track` without it), the map of a ride (`maps`,
  `MapConsentGate` shows a placeholder with "Karte anzeigen" instead), the geofence switch and "Bin da"
  (`presence_sharing`, declining sets the visibility to hidden) on the presence screen, camera and photo library
  everywhere they are offered (horse documents, observation report, blanket photo; `photos`), push registration
  (`useDeviceSetup` registers the token only with the consent; the first-run sheet comes from `ConsentOnboarding` in the
  root layout, asked once per device).
  **New features that record location, show maps, take photos or send notifications must call `ensure` first.**
- `app/settings/privacy.tsx`: switch per consent (revoking the geofence consent also stops it on the device), data
  export (JSON file written to the cache and shared with the system share sheet), links to the texts and the
  open-source notices, "Konto löschen" with a confirmation. Reachable from the presence screen, the join screen and
  the age screen (no stable needed).
- `app/legal/privacy.tsx`, `app/legal/licenses.tsx`: reachable without a session (links on
  sign-in, join and age).

## Tests

`go test ./internal/privacy ./internal/presence ./internal/auth ./internal/files` (needs `REITERHOF_TEST_DATABASE_URL`):
consent grant, revoke, re-grant, version conflict, cross-user isolation, the age rule, consequences of revocation,
geofence check-in refused without consent, export contains only own data (also for the other stable), deletion
(anonymised row, data gone, other users and the other stable untouched, files removed, requests reopened/cancelled),
blocked deletion for owners and last admins, retention, a test that the legal texts carry `TextVersion` and the draft
marker, the age confirmation and parental consent flow (mail, page, single use, expiry, rate limit), metadata
stripping, reference-based file access and download links. Mobile: `lib/consent-core.test.ts`,
`lib/consent-markdown.test.ts`, `lib/consent-legal.test.ts`, `lib/upload-core.test.ts`, `lib/licenses.test.ts`.
