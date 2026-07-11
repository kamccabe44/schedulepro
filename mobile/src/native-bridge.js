// ─────────────────────────────────────────────────────────────────────────────
// Native bridge — loaded ONLY in the mobile app (appended to index.html by
// scripts/sync-web.sh). The web build never includes this file, so frontend/
// behavior is unchanged.
//
// Responsibilities:
//   1. Route the Cognito OAuth redirect (custom scheme deep link) back into the
//      web app so the existing PKCE code-exchange in app.js can run.
//   2. Register for push notifications and hand the device token to the backend.
//   3. Use the native Share sheet for the "Share site" button when available.
//
// All Capacitor plugins are accessed via the global `Capacitor` object that the
// runtime injects — no bundler required.
// ─────────────────────────────────────────────────────────────────────────────
(function () {
  const Cap = window.Capacitor;
  if (!Cap || !Cap.isNativePlatform || !Cap.isNativePlatform()) return;

  const Plugins = Cap.Plugins || {};

  // ── 1. OAuth deep-link handling ────────────────────────────────────────────
  // When Cognito redirects to `com.ozarks.schedulepro://oauth-callback?code=...`,
  // the OS reopens the app with that URL. We rewrite the browser location so the
  // existing app.js handler picks up the `code`/`state` query params on load.
  if (Plugins.App) {
    Plugins.App.addListener("appUrlOpen", (event) => {
      try {
        const url = new URL(event.url);
        const params = url.search || "";
        if (params.includes("code=") || params.includes("error=")) {
          // Preserve query so the PKCE exchange in app.js can complete.
          window.history.replaceState({}, "", "/" + params);
          window.dispatchEvent(new Event("popstate"));
          // If app.js reads params only on initial load, a reload is the safe
          // fallback (verifier is persisted in sessionStorage before redirect).
          if (typeof window.handleAuthRedirect !== "function") {
            window.location.replace("/index.html" + params);
          }
        }
      } catch (e) {
        console.warn("[native-bridge] failed to parse deep link", e);
      }
    });
  }

  // ── 2. Push notifications ──────────────────────────────────────────────────
  // Registers with APNs/FCM and POSTs the token to the backend so it can send
  // booking reminders. The backend endpoint is a stub you implement server-side
  // (see mobile/README.md → "Push notifications").
  async function registerPush() {
    const Push = Plugins.PushNotifications;
    if (!Push) return;

    const perm = await Push.requestPermissions();
    if (perm.receive !== "granted") return;

    await Push.register();

    Push.addListener("registration", async (token) => {
      const cfg = window.SCHEDPRO_CONFIG || {};
      if (!cfg.apiUrl) return;
      try {
        await fetch(cfg.apiUrl + "/devices", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            token: token.value,
            platform: Cap.getPlatform(),
          }),
        });
      } catch (e) {
        console.warn("[native-bridge] device token upload failed", e);
      }
    });

    Push.addListener("registrationError", (err) =>
      console.warn("[native-bridge] push registration error", err)
    );
  }

  // Register after auth so the token can be associated with a signed-in user.
  window.addEventListener("load", () => {
    // Small delay lets app.js finish restoring the session first.
    setTimeout(registerPush, 1500);
  });

  // ── 3. Native share sheet ──────────────────────────────────────────────────
  // Progressive enhancement: if the app calls navigator.share, it already works
  // via the Web Share API inside the WebView. This block is a no-op placeholder
  // in case you want to force the native Share plugin later.
})();
