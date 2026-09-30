# Setting up sign-in

What the owner has to configure once. API and data model: [architecture.md](architecture.md#authentication-and-roles).

## Local development

```sh
REITERHOF_DEV_LOGIN=true just api      # enables POST /api/v1/auth/dev-login
curl -s localhost:8080/api/v1/auth/dev-login -d '{"email":"jan@example.org"}'   # {"token": "...", "user": {...}}
curl -s localhost:8080/api/v1/me -H "Authorization: Bearer <token>"
```

Seed users are `<name>@example.org` (`jan` is admin and owns Luna). Without SMTP, magic-link mails are
only written to the API log (`msg="mail (not sent, no SMTP configured)"`, the link is in `body`).
To sign in on a device with a real link, open `stallfunk://auth/verify?token=...` from the log
(`adb shell am start -a android.intent.action.VIEW -d "stallfunk://auth/verify?token=..."`).

## Email (SMTP)

Any SMTP provider works (Netcup mail, Mailgun, Brevo, ...). Set in `/etc/reiterhof/api.env`:

```
REITERHOF_SMTP_HOST=smtp.example.org
REITERHOF_SMTP_PORT=587            # 587 = STARTTLS, 465 = implicit TLS
REITERHOF_SMTP_USER=login@example.org
REITERHOF_SMTP_PASSWORD=...
REITERHOF_SMTP_FROM=Stallfunk <login@example.org>
REITERHOF_PUBLIC_URL=https://api.example.org
```

Set SPF, DKIM and DMARC for the sender domain, otherwise login mails land in spam.
`REITERHOF_PUBLIC_URL` adds an https fallback link (`/auth/verify?token=...`) to the mail; it opens a page that
links to the app. It does not need Universal Links / App Links (not configured yet, see open points in the JAN-6 report).

## Sign in with Google

1. Google Cloud Console, project for Stallfunk, "APIs & Services" > "OAuth consent screen": configure (external), app name, support email.
2. "Credentials" > "Create credentials" > "OAuth client ID", three clients:
   - Web application (its ID is also used by Expo's auth proxy in development),
   - iOS, bundle ID `org.datenlotse.stallfunk`,
   - Android, package `org.datenlotse.stallfunk` and the SHA-1 of the signing key (debug key for development, Play App Signing key for release).
3. Put the IDs into `mobile/app.json` under `expo.extra`: `googleWebClientId`, `googleIosClientId`, `googleAndroidClientId`.
   Until the ID of the current platform is set, the button shows "Diese Anmeldung ist noch nicht eingerichtet."
4. Put **all** client IDs, comma separated, into `REITERHOF_GOOGLE_CLIENT_IDS` on the server; the token's `aud` is the ID of the platform that requested it.
5. Google sign-in needs a development build or a release build (not Expo Go on Android).

## Sign in with Apple (iOS only)

1. Apple Developer account > Identifiers > App ID `org.datenlotse.stallfunk` > enable "Sign in with Apple".
2. `app.json` already sets `ios.usesAppleSignIn` and the `expo-apple-authentication` plugin. It needs a development/release build, not Expo Go.
3. Server: `REITERHOF_APPLE_CLIENT_IDS=org.datenlotse.stallfunk` (the bundle ID is the token's `aud` for native sign-in).
4. Apple may hide the real address behind `...@privaterelay.appleid.com`; that is accepted as a normal address and never merged with other accounts.
5. The button is hidden on Android.
