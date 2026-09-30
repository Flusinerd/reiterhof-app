# Setting up sign-in

What the owner has to configure once. API and data model: [architecture.md](architecture.md#authentication-and-roles).

## Local development

```sh
REITERHOF_DEV_LOGIN=true just api      # enables POST /api/v1/auth/dev-login
curl -s localhost:8080/api/v1/auth/dev-login -d '{"email":"jan@example.org"}'   # {"token": "...", "user": {...}}
curl -s localhost:8080/api/v1/me -H "Authorization: Bearer <token>"
```

Seed users are `<name>@example.org` (`jan` is admin and owns Luna). Without SMTP, magic-link mails are
only written to the API log (`msg="mail (not sent, no SMTP configured)"`, the 6-digit code is in `subject`
and `body`, the link in `body`). Sign in with the code:

```sh
curl -s localhost:8080/api/v1/auth/magic-link -d '{"email":"jan@example.org"}'   # 204; read the code from the API log
curl -s localhost:8080/api/v1/auth/verify-code -d '{"email":"jan@example.org","code":"123456"}'   # {"token": "...", "user": {...}}
```

To sign in on a device with a real link, open `stallfunk://auth/verify?token=...` from the log
(`adb shell am start -a android.intent.action.VIEW -d "stallfunk://auth/verify?token=..."`).

## Email (SMTP)

Production uses **Resend** (any SMTP provider works). Set in `/etc/reiterhof/api.env`:

```
REITERHOF_SMTP_HOST=smtp.resend.com
REITERHOF_SMTP_PORT=587            # 587 = STARTTLS, 465 = implicit TLS
REITERHOF_SMTP_USER=resend         # literally "resend"
REITERHOF_SMTP_PASSWORD=re_...     # Resend API key with "Sending access" only
REITERHOF_SMTP_FROM=Stallfunk <login@stallfunk.de>
REITERHOF_PUBLIC_URL=https://api.stallfunk.de
```

Resend setup: add the domain `stallfunk.de` in the Resend dashboard (choose the EU region), create the DNS records
it shows (DKIM `TXT` on `resend._domainkey`, SPF `TXT` and `MX` on the `send` subdomain), wait for "Verified", then
create an API key restricted to "Sending access" for that domain. Add a DMARC record
(`_dmarc.stallfunk.de TXT "v=DMARC1; p=none;"` to start), otherwise login mails may land in spam.
`REITERHOF_PUBLIC_URL` adds an https fallback link (`/auth/verify?token=...`) to the mail; it opens a page that
links to the app. It does not need Universal Links / App Links (not configured yet, see open points in the JAN-6 report).
The same base URL carries the link in the mail to parents (`/parental-consent?token=...`, see
[domains/privacy.md](domains/privacy.md#age-confirmation-and-parental-consent-jan-86)); `REITERHOF_WEB_URL`
(`https://stallfunk.de`) lets that mail and page link the privacy text of the web app.

## Login code key

The mail contains a 6-digit code next to the link (needed for the iPhone PWA, where the link opens in Safari and
not in the installed app). The API stores only an HMAC of it. Set a secret key once in `/etc/reiterhof/api.env`:

```
REITERHOF_LOGIN_CODE_KEY=...       # openssl rand -hex 32
```

Without it the API generates a random key per start (and logs a warning): codes from mails sent before a restart
then stop working, links keep working. Changing the key has the same effect, so it is safe to rotate. Details:
[architecture.md](architecture.md#authentication-and-roles).

## Sign in with Google

1. Google Cloud Console, project for Stallfunk, "APIs & Services" > "OAuth consent screen": configure (external), app name, support email.
2. "Credentials" > "Create credentials" > "OAuth client ID", three clients:
   - Web application (its ID is also used by Expo's auth proxy in development),
   - iOS, bundle ID `de.flusinerd.stallfunk`,
   - Android, package `de.flusinerd.stallfunk` and the SHA-1 of the signing key (debug key for development, Play App Signing key for release).
3. Put the IDs into `mobile/app.json` under `expo.extra`: `googleWebClientId`, `googleIosClientId`, `googleAndroidClientId`.
   Until the ID of the current platform is set, the button shows "Diese Anmeldung ist noch nicht eingerichtet."
4. Put **all** client IDs, comma separated, into `REITERHOF_GOOGLE_CLIENT_IDS` on the server; the token's `aud` is the ID of the platform that requested it.
5. Google sign-in needs a development build or a release build (not Expo Go on Android).

## Sign in with Apple (iOS only)

1. Apple Developer account > Identifiers > App ID `de.flusinerd.stallfunk` > enable "Sign in with Apple".
2. `app.json` already sets `ios.usesAppleSignIn` and the `expo-apple-authentication` plugin. It needs a development/release build, not Expo Go.
3. Server: `REITERHOF_APPLE_CLIENT_IDS=de.flusinerd.stallfunk` (the bundle ID is the token's `aud` for native sign-in).
4. Apple may hide the real address behind `...@privaterelay.appleid.com`; that is accepted as a normal address and never merged with other accounts.
5. The button is hidden on Android.
