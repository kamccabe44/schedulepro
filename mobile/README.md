# SchedulePro Mobile (iOS + Android)

Packages the existing SchedulePro web app as native iOS and Android apps using
[Capacitor](https://capacitorjs.com/). This folder is **fully self-contained** —
nothing here modifies `frontend/`, `backend/`, or the deploy pipeline. The web
app is copied in at build time by `scripts/sync-web.sh`, so the site keeps
working exactly as before until you decide to publish.

```
mobile/
├── package.json            # Capacitor deps + build scripts
├── capacitor.config.json   # App id, name, native settings
├── config.mobile.js        # Baked-in prod config (edit before building)
├── src/native-bridge.js    # Deep-link auth + push, injected only in the app
├── scripts/sync-web.sh      # Copies frontend/ → www/ (read-only toward frontend)
├── www/                    # BUILD OUTPUT (gitignored, regenerated)
├── ios/                    # Xcode project (gitignored, created by `cap add`)
└── android/                # Android Studio project (gitignored, created by `cap add`)
```

## How isolation works

- The only connection to the rest of the repo is `sync-web.sh`, which **reads**
  `frontend/` and writes into `mobile/www/`. It never edits the originals.
- `www/`, `ios/`, `android/`, and `node_modules/` are gitignored — only the
  source scaffolding is committed, so the mobile setup adds no build artifacts
  to the repo and can't affect a web deploy.
- Deleting the `mobile/` folder returns the repo to a pure web app.

## Prerequisites

| For | You need |
|---|---|
| Both | Node 18+, `npm` |
| iOS build | **macOS** + Xcode + [Apple Developer account](https://developer.apple.com/programs/) ($99/yr — unlimited apps) |
| Android build | Android Studio + JDK 17 + [Google Play account](https://play.google.com/console/signup) ($25 once — unlimited apps) |

## Quick start

```bash
cd mobile
npm install

# 1. Fill in your PRODUCTION values (from scripts/deploy.sh output or Terraform).
#    Edit config.mobile.js — apiUrl, cognitoDomain, cognitoClientId.

# 2. Generate the native projects.
npm run prepare:android    # creates android/, needs Android Studio to build
npm run prepare:ios        # creates ios/, macOS only

# 3. Open in the native IDE to run on a simulator/device.
npm run open:android
npm run open:ios
```

After changing anything in `frontend/` later, re-sync with:

```bash
npm run sync        # rebuilds www/ from frontend/ and pushes into ios/ + android/
```

## Auth on native (required before login works)

The web app signs in with Cognito Hosted UI + PKCE and redirects back to a web
URL. A native app can't receive a web redirect, so it uses a **custom URL
scheme** instead. Two things must line up:

1. In `config.mobile.js`, `cognitoRedirectUri` is set to
   `com.ozarks.schedulepro://oauth-callback` (matches the `appId`).
2. Add that **exact** URI to the Cognito app client's *Allowed callback URLs*
   (Cognito console → App client → Hosted UI, or via Terraform in
   `terraform/`). The custom scheme is registered automatically for iOS
   (`CFBundleURLTypes`) and Android (intent filter) when you run `cap add`;
   verify it if you change the `appId`.

`src/native-bridge.js` listens for the redirect deep link and hands the
`?code=...` back to the existing PKCE exchange in `app.js`.

> If you'd rather not touch Cognito yet, the app still builds and runs — only
> the sign-in round-trip needs this. Browsing works without it.

## Push notifications (booking reminders)

This is the main reason to go native. `src/native-bridge.js` already:
- requests notification permission,
- registers with APNs (iOS) / FCM (Android),
- POSTs the device token to `POST {apiUrl}/devices`.

**Backend work you still need to do** (in the Go Lambda, kept out of this folder
on purpose):
1. Add a `POST /devices` route that stores `{token, platform, userId}` in
   DynamoDB.
2. When a booking is created/approaching, send a push via APNs/FCM (e.g. AWS SNS
   mobile push, or a small APNs/FCM call) to the stored tokens.
3. iOS also needs an APNs key in your Apple Developer account; Android needs a
   Firebase project for FCM.

Until the backend route exists, the token upload just logs a warning — the app
is otherwise unaffected.

## Publishing

1. **Icons & splash:** drop a 1024×1024 PNG in `resources/` and run
   `npx @capacitor/assets generate` (add the tool if you want it), or set them
   in Xcode / Android Studio.
2. **iOS:** in Xcode set your Team (Apple Developer account), bump the version,
   Product → Archive → distribute to App Store Connect.
3. **Android:** in Android Studio, Build → Generate Signed Bundle (`.aab`),
   upload to the Play Console.
4. One Apple account and one Google account cover **unlimited** apps, so you can
   ship one SchedulePro app or one per customer under the same accounts.

## Notes

- The QR-code library loads from a CDN; the WebView has network access so it
  works, but for an offline-tolerant app consider vendoring it locally later.
- App id `com.ozarks.schedulepro` is a placeholder — change it in
  `capacitor.config.json` (and re-run `cap add`) to your real reverse-domain id
  before first publish, since it's permanent per app on the stores.
