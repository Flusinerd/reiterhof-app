# Session tracking (M7)

Tickets: JAN-63 GPS ride tracking, JAN-65 tracking in hall and arena, the exercise library screens of
JAN-58. Gait detection itself is JAN-64 ([`mobile/lib/gait/README.md`](../../mobile/lib/gait/README.md)).

**Status: implemented and unit-tested, not tested on a device.** GPS, background location, the
accelerometer, the map and the foreground service could not be exercised in the development
environment (no phone or emulator). Everything below the "Logic" line is covered by `node --test`;
everything that touches a sensor or the OS is untested. The gait model is validated on synthetic
data only (see the gait README).

## Flow

"Starten" in "Was heute?" opens `app/training/session.tsx` with `horse`, `activity`, `minutes`, `exercise`.
That screen is a chooser: hack (Ausritt) recommends GPS, everything else recommends indoor, both stay
available. It also offers to continue or discard a session that was still stored when the app was
closed, and links the exercise library.

| Screen | Purpose |
| --- | --- |
| `app/training/track/gps.tsx` | Ride with GPS (JAN-63) |
| `app/training/track/indoor.tsx` | Hall, arena, lunge without GPS (JAN-65) |
| `app/training/exercises/index.tsx`, `[id].tsx` | Library: filter by discipline, level and goal; steps and next progression |
| `app/training/session/finish.tsx` | Unchanged API; additionally posts `track` and `gait_windows` (see below) |

### GPS mode

- `expo-location` updates through one task (`reiterhof-tracking`, `lib/tracking-location.ts`), foreground and
  background (Android foreground service with a notification, iOS background location indicator). The
  geofence of the check-in (`lib/geofence.ts`, task `reiterhof-geofence`) is a separate task and API and is
  not touched. The task is defined at import time; `app/_layout.tsx` imports the module so the OS can wake it.
- Fixes are filtered (accuracy over 35 m, jumps above 90 km/h, old or duplicate timestamps, while paused).
  Distance is the haversine sum per uninterrupted run, so a lift home after a pause adds nothing. Average
  speed is distance over active time. Elevation gain: moving average over 5 fixes, then a 3 m hysteresis.
- Gait per point comes from the `GaitStream` windows (accelerometer, 50 Hz, fused with GPS speed); where no
  window exists (sensor stopped in the background) the OS speed decides, then the speed between fixes.
- Map: `react-native-maps`, polyline pieces colored by gait (walk `#86efac`, trot `#fbbf24`, canter
  `#c2410c` on the light map, halt grey). Pauses are not drawn. The camera follows the newest point.
- `expo-keep-awake` keeps the screen on while the tracking screen is open.
- Permission denial: German messages for denied, blocked ("in den Einstellungen erlauben"), location services
  off. Without the background permission the ride is still recorded while the app is open (hint shown).
- Heart rate is not implemented (needs a BLE sensor or a wearable API); it is not part of the data model.

### Indoor mode

- GPS is off. The accelerometer feeds `GaitStream`; the current gait is the newest window.
- The big buttons Schritt, Trab, Galopp only *correct*: a tap says "the gait is X now". The windows from the
  last 6 s that the detector called something else (`wrong`) are labelled X, and so is every following window
  until the detector itself changes its mind or the next tap (`applyCorrections`). The label wins in the
  gait shares and is exported as `c` with the raw window, which is the training label.
- "Handwechsel" toggles the rein. Time per rein is *active* time (pauses excluded); segments under 5 s are
  folded into their neighbour. The result is the ordered list `[{rein, minutes}]` of the API.
- The exercise checklist shows the steps ("Ablauf") of the recommended exercise; ticks are local and stored in
  the snapshot but not sent. Target duration: progress bar and "noch X Min." from `minutes`.
- Pause stops the clock, the sensor and (GPS) the location updates; resume starts a fresh `GaitStream`.

### Finish

`finish()` closes the session and navigates to `/training/session/finish` with the params the finish screen
already reads: `horse`, `activity`, `minutes` (active time, at least 1), `started_at`, `exercise`, `gait`
(JSON, fractions of `halt|walk|trot|canter`, rounded down so the sum stays at most 1), `rein` (JSON) and
`distance_m`. Track and windows are too large for router params: they go to `lib/tracking-store.ts`
(`saveFinishExtras`) and `finish.tsx` reads them (`loadFinishExtras`, keyed by `started_at`), adds them to the
POST body and deletes them after saving. Until the extras are loaded the save button shows its spinner.

## Local persistence

The running session is a JSON snapshot (`tracking-session.json` in the app's document directory, via
`expo-file-system`): state, gait windows, checked steps. It is written every 15 s, on pause, on rein changes,
on corrections and when the app goes to the background, and deleted when the session is finished or discarded.
Files were chosen over `expo-secure-store` because a ride is hundreds of kilobytes and the secure store is
meant for small values; AsyncStorage is not a dependency.

After an app kill the chooser offers "Fortsetzen". The session comes back **paused** at the last sign of life
(the dead time does not count). Fixes that the background task received while no screen listened are buffered
in `tracking-pending.json` and folded in first. Not persisted: the live `GaitStream` buffer (the last few
seconds of samples).

## API additions (`POST /horses/{id}/sessions`)

Additive; a session with any of these has `source = tracked`.

| Field | Content | Limits |
| --- | --- | --- |
| `track` | `[{lat, lon, t, alt?, g?}]`. `t` seconds since `started_at`, ascending; `g` is the gait of the stretch that ends at the point (`halt|walk|trot|canter`). The first point after a pause has `g: halt` (it bridges the pause). Stored in `sessions.track` (jsonb, NULL without track). | at most 5000 points, coordinates in range, altitude -500..9000 m |
| `gait_windows` | `[{t, f, p, a, v?, c?}]`, the app's `WindowRecord`: `t` ms since `started_at`, `f` six features in the order of `FEATURE_ORDER`, `p` predicted gait, `a` accelerometer-only gait, `v` GPS km/h, `c` user-corrected gait (the label). Stored in table `gait_windows` (migration `0090`, one row per session, cascade delete). | at most 3000 windows (about 100 min); the app keeps corrected windows first and thins the rest evenly |
| `distance_m` | already existed | 0..500000 |

The 1 MiB body limit of `httpx.ReadJSON` holds with a full track plus full windows (about 300 KB + 330 KB).
`track` and `gait_windows` are write-only for now: the session responses do not return them (a ride detail
screen would add `GET /sessions/{id}`). The windows are not visible to riders or owners through any route;
they are for improving the model.

## Logic (unit-tested, `node --test`)

`lib/geo.ts` (haversine, elevation gain, Douglas-Peucker with forced points and a point cap),
`lib/tracking.ts` (state, pause, fix filter, stats, rein accounting, corrections, gait timeline, segmenting,
API payloads, target progress), `lib/tracking-persist.ts` (snapshot and restore rule),
`lib/tracking-exercises.ts` (filters and labels), `lib/tracking-format.ts` (German number formats).
Wiring lives in `lib/tracking-session.ts` (hook), `lib/tracking-location.ts`, `lib/tracking-store.ts` and
`components/tracking-*.tsx`, `components/exercise-*.tsx`; those are not unit-tested.

## Setup and open points

- **Google Maps API key (Android).** `react-native-maps` on Android uses Google Maps and needs a key in
  `app.json` (`android.config.googleMaps.apiKey`); without it the map stays blank. iOS uses Apple Maps. The key
  is not in the repository. Expo Go cannot run this; use a development build.
- `app.json` changes: foreground service for `expo-location` (`FOREGROUND_SERVICE`,
  `FOREGROUND_SERVICE_LOCATION`), iOS `UIBackgroundModes: location`, the `expo-sensors` plugin, and permission
  texts that now also mention ride recording. The store review needs a justification for background location.
- Privacy: a track is precise location data that is stored in the cloud. GPS tracking needs the
  `location_tracking` consent (JAN-19): the GPS screen calls `useConsentPrompt().ensure("location_tracking")`
  before the OS permission is requested, and `POST /horses/{id}/sessions` with a `track` answers
  `403 consent_required` without it (`privacy.Has`). Indoor tracking sends no coordinates and needs no location
  consent. See [privacy.md](privacy.md); a per-session "hide route" option is not built.
- Sensors in the background: Android may throttle or stop the accelerometer when the screen is off even with a
  foreground service; the GPS speed then decides the gait. Unverified. Phone placement matters (see the gait
  README); the learned calibration (`learnCalibration`) has no UI yet.
- The exercise library is linked from the Training tab and from the session chooser ("Übungsbibliothek").
- The rein starts on the left; a change within the first 5 s folds into the start, so "I start on the right
  rein" works by tapping "Handwechsel" right away.
