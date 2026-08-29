# Boop for iOS

The native client for a self-hosted [Boop](../README.md) server. You build and sign it yourself with your own Apple Developer account; nothing is distributed through Boop infrastructure.

Requires Xcode 26 and iOS 26. App version 1.2.0 needs server 1.2.0 or newer for grouping, actions and the share menu (older servers still work; those features simply do not appear).

## Build it

```bash
open ios/Boop.xcodeproj
```

Then in Xcode:

1. Select the **Boop** target → **Signing & Capabilities**.
2. Pick your **Team**.
3. Change the **Bundle Identifier** from `com.example.Boop` to your own (for example `com.yourname.Boop`). This exact value must also be set as `APNS_BUNDLE_ID` on your server. Do the same for the **BoopNotificationService** target: its identifier must be your app id plus `.NotificationService` (for example `com.yourname.Boop.NotificationService`).
4. Make sure the **Push Notifications** capability is present (it comes from `Boop.entitlements`). Xcode will register the App ID with Apple automatically when signing is set up.
5. Plug in your iPhone, select it as the run destination, and press Run. Push notifications do not work in the simulator.

`Boop.entitlements` ships with `aps-environment: development`; Xcode switches it to `production` automatically when you archive. Match the server: `APNS_ENVIRONMENT=sandbox` for a debug build installed from Xcode, `production` for TestFlight or an archive.

The project file is generated from `project.yml` with [XcodeGen](https://github.com/yonaskolb/XcodeGen). If you edit `project.yml`, run `xcodegen generate` in `ios/`. Editing the bundle id and team in Xcode is fine too.

## Pair it

1. In the Boop web UI open **Devices** → **Pair iPhone**.
2. In the app tap **Pair server** and scan the code. There is also **Enter details manually** for the simulator or when the camera is unavailable; the values are under **Show payload** in the web UI.
3. Allow notifications when asked. The app registers with APNs and sends the token to your server; the device then shows as "Registered" in the web UI.
4. **Settings → Send test notification** in the web UI should now reach your phone.

The server address in the pairing code comes from `BOOP_BASE_URL`. It must be reachable from the phone (HTTPS in production). For local development, `http://<your-mac-ip>:8080` works: `Info.plist` allows plain-HTTP local networking.

## What it does

- Inbox grouped by day, pull to refresh, cursor pagination, project and level filters.
- Group repeats: events sharing a fingerprint show as one row (`KeyError ×47 · First seen 09:31 · Last seen 10:42`) that opens the occurrences. Toggle in the filter menu.
- Event detail with exception, stacktrace (in-app frames highlighted), tags, context, breadcrumbs, raw JSON, and action buttons.
- Share menu: **Copy** (plain text), **Copy as Markdown** (sectioned, ready for an AI assistant) and **Share** (straight into another app).
- Notification actions: an event's `actions` become buttons on the push (long-press it). The `BoopNotificationService` extension registers them just in time; tapping one opens its URL.
- Notification tap opens the event and fetches the latest data; if the server is unreachable the push's title and body are shown with a retry.
- Device credential is stored in the Keychain. The app never sees project API keys or APNs keys.
- Re-registers with APNs on every launch and updates the server when the token changes.

## Simulator testing

APNs registration fails in the simulator (expected; the app treats "no token" as normal), but everything else can be exercised:

```bash
# Pair via "Enter details manually" using a token minted from the server:
curl -s -X POST http://localhost:8080/api/v1/pairing | jq -r .token

# Deliver a fake push to the simulator to test notification tap → event detail:
cat > /tmp/push.json <<'JSON'
{"aps":{"alert":{"title":"Uini · KeyError","body":"key :can_palette? not found"},"sound":"default"},"event_id":"evt_REPLACE","project_id":"prj_REPLACE"}
JSON
xcrun simctl push booted com.example.Boop /tmp/push.json

# With action buttons (long-press the banner; simctl does not run the service
# extension, so on the simulator the buttons only appear on a real device):
cat > /tmp/push-actions.json <<'JSON'
{"aps":{"alert":{"title":"Shop · Payment received","body":"£19.99"},"sound":"default","category":"boop.event.actions","mutable-content":1},
 "event_id":"evt_REPLACE","project_id":"prj_REPLACE","actions":[{"label":"Open in Stripe","url":"https://dashboard.stripe.com"}]}
JSON
xcrun simctl push booted com.example.Boop /tmp/push-actions.json
```

## Tests

```bash
xcodebuild -project Boop.xcodeproj -scheme Boop -destination 'platform=iOS Simulator,name=iPhone 17 Pro' test
```
