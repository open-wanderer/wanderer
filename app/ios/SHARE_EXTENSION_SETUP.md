# iOS Share Extension — manual Xcode setup

The Dart code, Android intent filters, and all iOS **source files** for receiving
shared trail files are in place. The remaining steps must be done in Xcode
because they modify `Runner.xcodeproj/project.pbxproj` (adding a build target,
entitlements, and signing), which cannot be safely hand-edited.

Files already created (add them to the target you create below):

- `Share Extension/ShareViewController.swift`
- `Share Extension/Info.plist`
- `Share Extension/ShareExtension.entitlements`
- `Share Extension/MainInterface.storyboard`
- `Runner/Runner.entitlements` (App Group for the host app)

## Steps

1. **Open** `ios/Runner.xcworkspace` in Xcode.

2. **Add the App Group to Runner**
   - Select the **Runner** target → **Signing & Capabilities** → **+ Capability** → **App Groups**.
   - Add `group.com.openwanderer.wanderer`.
   - Confirm **Code Signing Entitlements** points to `Runner/Runner.entitlements`.

3. **Create the Share Extension target**
   - **File → New → Target… → Share Extension**. Name it exactly **`Share Extension`** (matches the Podfile target and folder).
   - When prompted, do **not** activate the scheme.
   - Delete the boilerplate `ShareViewController.swift`, `Info.plist`, and
     `MainInterface.storyboard` Xcode generated, then **Add Files…** the four
     files from the existing `Share Extension/` folder instead (so the versioned
     ones are used). Ensure they're added to the **Share Extension** target.

4. **Add the App Group to the extension**
   - Select the **Share Extension** target → **Signing & Capabilities** → **App Groups** → add the same `group.com.openwanderer.wanderer`.
   - Set its **Code Signing Entitlements** to `Share Extension/ShareExtension.entitlements`.

5. **Match deployment target & signing**
   - Set the extension's **iOS Deployment Target** to 15.0 (same as Runner).
   - Assign the same development team.

6. **Install pods**
   - `cd ios && pod install` (the Podfile already declares the `Share Extension` target).

7. **Bundle id note**
   - The app bundle id is `com.openwanderer.wanderer`. If you change it again,
     update the App Group id (`group.<new-bundle-id>`) in **both** entitlements
     files and both capability screens to keep them in sync.

## Verify

Run on a real device: share a `.gpx` from Files/Mail → Wanderer appears in the
share sheet → tapping it opens the trail create/edit screen with the route drawn.
