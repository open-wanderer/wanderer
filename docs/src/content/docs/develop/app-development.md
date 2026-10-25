---
title: App development
description: How to set up a development environment for the wanderer mobile app
---

The <span class="-tracking-[0.075em]">wanderer</span> mobile app is a [Flutter](https://flutter.dev) application located in the `app/` folder of the repository. This guide walks you through setting up a development environment for it.

:::caution
The app has not been merged into the main release line yet. Two consequences:

- The `app/` folder only exists on the `feature/app` branch. Check that branch out.
- The backend the app talks to needs endpoints (region catalogue, health probe, navigation, and more) that are only in the **`-app` Docker images**: `flomp/wanderer-db:<version>-app` and `flomp/wanderer-web:<version>-app`, e.g. `v0.21.0-app`. Both images must carry the same tag, because the web image hosts the `/api/v1/regions` proxy the app relies on.
:::

## Prerequisites

- [Flutter SDK](https://docs.flutter.dev/get-started/install) with Dart `^3.11.5`
- A running <span class="-tracking-[0.075em]">wanderer</span> backend that supports the app: either a [local development setup](/develop/local-development) from the `feature/app` branch, or a self-hosted instance running the `-app` images (see above)
- For Android: Android Studio / SDK with a JDK 17, `compileSdk` 37 and `minSdk` 26 (Android 8.0)
- For iOS: Xcode with CocoaPods; an Apple development certificate for running on a physical device (the simulator needs none)

Verify your setup with:

```bash
cd app
flutter doctor
```

## Install dependencies & generate code

The app relies on generated code for immutable models ([freezed](https://pub.dev/packages/freezed)), JSON serialization, [Riverpod](https://riverpod.dev) providers, and the [ObjectBox](https://objectbox.io) local database. After cloning (and after every change to an annotated class) run:

```bash
cd app
flutter pub get
dart run build_runner build --delete-conflicting-outputs
```

`lib/objectbox-model.json` and `lib/objectbox.g.dart` are committed. Commit them whenever an ObjectBox entity changes; the model file records entity and property IDs that must stay stable across builds.

## Localization

User-facing strings live in ARB files in `lib/i18n/` (`app_en.arb` is the template). `flutter run` and `flutter build` regenerate the localization classes automatically (`generate: true` in `pubspec.yaml`). To regenerate them explicitly after adding or changing a string, and to refresh the untranslated-messages report, run:

```bash
flutter gen-l10n
```

This also updates `lib/i18n/untranslated_messages.json`, a committed report of strings that are still English-only in other locales. If your change makes this file grow, consider providing translations, or commit the updated report so the gap is at least visible in the diff.

## Run the app

Start the app on a connected device or emulator:

```bash
flutter run
```

### Connecting to a local backend

On first launch the app asks which instance to connect to. You can enter any URL, including your local development server.

The app talks to the **SvelteKit frontend**, which proxies to PocketBase. Point it at the frontend's port (`5173` for `npm run dev`, `3000` for a production build), never at PocketBase's `8090` directly.

For security reasons the app only permits unencrypted (`http://`) connections to `127.0.0.1` (iOS additionally accepts `localhost`); plain HTTP to LAN or emulator-bridge addresses (such as `10.0.2.2`) is blocked on both platforms. To connect to a backend running on your development machine:

**Android (emulator or USB device):** forward the port with adb, then connect to `127.0.0.1`:

  ```bash
  adb reverse tcp:5173 tcp:5173
  ```

  Then enter `http://127.0.0.1:5173` as the instance URL (or port `3000` for a production-mode frontend).

**iOS simulator:** the simulator shares the host network; enter `http://127.0.0.1:5173` directly.

**Physical device without adb/USB:** expose your dev server via HTTPS (e.g. a reverse proxy or tunnel) and use that URL.

:::caution[`ORIGIN` must match the URL you enter]
The app does not persist the URL you type. After signing in it stores the server address derived from your user's ActivityPub actor IRI, which the backend builds from its `ORIGIN` environment variable, and it re-reads that stored address on every restart. If `ORIGIN` differs from what you entered (for example `http://localhost:5173` from the [local development guide](/develop/local-development) versus `http://127.0.0.1:5173` in the app), the first sign-in succeeds but the session is gone after the next app restart: the auth cookie was stored for `127.0.0.1`, while the app now looks for it under `localhost`, and on Android plain HTTP to `localhost` is blocked anyway.

For app development set `ORIGIN` to exactly the URL you enter in the app, scheme, host and port included:

```bash
export ORIGIN=http://127.0.0.1:5173
```

Existing actors keep the IRI they were created with. Changing `ORIGIN` after the fact does not fix a user created under the old value; create a fresh user, or edit the user's `activitypub_actors` record (its `iri`, `inbox`, `outbox`, `followers` and `following` URLs) in the PocketBase dashboard.
:::

## Project layout

| Folder | Contents |
| ------ | -------- |
| `lib/routes/` | One file per screen, wired together in `lib/provider/router_provider.dart` (go_router) |
| `lib/components/` | Reusable widgets, grouped by feature (`trail/`, `map/`, `route_planner/`, …) |
| `lib/provider/` | Riverpod providers: app state, API access, settings |
| `lib/actions/` | Multi-step user flows shared between screens (launching navigation, importing a file, requesting permissions) |
| `lib/models/` | Immutable data models (freezed) |
| `lib/entities/` | ObjectBox entities for local persistence (offline trails, recordings, regions) |
| `lib/services/` | Long-running services (trail downloads, tile proxy, position sources) |
| `lib/i18n/` | ARB translation files and generated localizations |

## Analysis & tests

Before committing, make sure static analysis and the test suite pass:

```bash
flutter analyze
flutter test
```
