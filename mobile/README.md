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

This is the main reason to go native, and **the backend is now wired up** using
Amazon SNS mobile push (same AWS account, near-$0). End to end:

- `src/native-bridge.js` requests permission, registers with APNs/FCM, and
  `POST`s the token to `/devices`.
- The Go Lambda (`backend/notify.go`) stores tokens in a `device-tokens`
  DynamoDB table and, on booking, sends a confirmation push via SNS.
- Terraform (`terraform/sns.tf`) creates the SNS platform applications — **only
  when you provide credentials**. With no credentials, tokens are still stored
  but nothing is sent, so a normal deploy is completely unchanged.

To turn push ON, set these Terraform variables (e.g. in `terraform.tfvars`) and
re-apply:

| Variable | Where it comes from |
|---|---|
| `apns_key` | Apple Developer → Keys → new APNs `.p8` key (file contents) |
| `apns_key_id` | the key's 10-char ID |
| `apns_team_id` | Apple Developer membership → team ID |
| `apns_bundle_id` | your app id (default `com.ozarks.schedulepro`) |
| `apns_sandbox` | `true` for dev/TestFlight builds, `false` for App Store |
| `fcm_service_account_json` | Firebase project → service account JSON (Android) |

You only need APNs for iOS or FCM for Android — set whichever you're shipping.
The `apns_sandbox = true` default is correct for the free device-testing methods
below; flip it to `false` only for a production App Store build.

## Testing on your own phone WITHOUT publishing

You do not need the App Store or Play Store to run this on a real device. Options
from easiest to most involved:

### Android — easiest
1. `npm run prepare:android && npm run open:android`
2. Enable **Developer options → USB debugging** on your phone, plug it in.
3. Press **Run ▶** in Android Studio — the app installs and launches directly.
   No Google Play account and no signing needed for your own device.
4. Or build a debug APK (Build → Build APK), email/AirDrop it to yourself, and
   install it (allow "install unknown apps"). Great for handing a test build to
   someone else.

Android push works immediately in this mode once FCM credentials are set.

### iOS — free, with one caveat
1. On a Mac: `npm run prepare:ios && npm run open:ios`.
2. In Xcode → Signing & Capabilities, sign in with your **free** Apple ID and let
   Xcode manage signing (a paid account is NOT required to run on your own
   device).
3. Plug in your iPhone, select it as the target, press **Run ▶**.
4. On the phone: Settings → General → VPN & Device Management → trust your
   developer certificate.

Caveat: with a **free** Apple ID the app runs for 7 days before you must
re-install from Xcode, and **push notifications require the paid Apple Developer
account** (APNs isn't available to free accounts). So you can test the whole app
and UI for free; to test iOS push specifically you'll need the $99/yr account.

### TestFlight — best for real beta testing (iOS)
Once you have the paid Apple account, upload a build to **TestFlight** and invite
up to 10,000 testers by email — full push support, installs like a normal app,
no public App Store listing. This is the standard way to test with real users
before publishing. (Android's equivalent is Play Console **Internal testing**.)

### Recommended path for you
Start with **Android via Android Studio** (fully free, push included) to validate
the whole flow, then use **iOS free-provisioning** for the UI on an iPhone. Add
the paid Apple account when you're ready to test iOS push or ship.

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
