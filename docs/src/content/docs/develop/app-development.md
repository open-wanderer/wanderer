---
title: App development
description: How to set up a development environment for the <span class="-tracking-[0.075em]">wanderer</span> mobile app
---

The <span class="-tracking-[0.075em]">wanderer</span> mobile app is a [Flutter](https://flutter.dev) application located in the `app/` folder of the repository. This guide walks you through setting up a development environment for it.

:::caution
The app has not been merged into the main release line yet. To develop against a compatible backend you must run the current release version with the `-app` suffix, e.g. `v0.20.0-app`, instead of the plain release tag. These tags contain the additional backend endpoints (region catalogue, health probe, navigation, and more) that the app depends on.
:::

## Prerequisites

- [Flutter SDK](https://docs.flutter.dev/get-started/install) with Dart `^3.11.5`
- A running <span class="-tracking-[0.075em]">wanderer</span> backend at the matching `-app` release (see above) — either a [local development setup](/develop/local-development) checked out at that tag or a self-hosted instance running it
- For Android: Android Studio / SDK with `minSdk` 26 (Android 8.0) or higher
- For iOS: Xcode and an Apple development certificate

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

## Localization

User-facing strings live in ARB files in `lib/i18n/` (`app_en.arb` is the template). After adding or changing a string, regenerate the localization classes with:

```bash
flutter gen-l10n
```

This also updates `lib/i18n/untranslated_messages.json`, a committed report of strings that are still English-only in other locales. If your change makes this file grow, consider providing translations — or commit the updated report so the gap is at least visible in the diff.

## Run the app

Start the app on a connected device or emulator:

```bash
flutter run
```

### Connecting to a local backend

On first launch the app asks which instance to connect to. You can enter any URL — including your local development server.

For security reasons the app only permits unencrypted (`http://`) connections to `127.0.0.1`; plain HTTP to LAN or emulator-bridge addresses (such as `10.0.2.2`) is blocked on both platforms. To connect to a backend running on your development machine:

- **Android (emulator or USB device):** forward the port with adb, then connect to `127.0.0.1`:

  ```bash
  adb reverse tcp:5173 tcp:5173
  ```

  Then enter `http://127.0.0.1:5173` as the instance URL (or port `3000` for a production-mode frontend).

- **iOS simulator:** the simulator shares the host network — enter `http://127.0.0.1:5173` directly.

- **Physical device without adb/USB:** expose your dev server via HTTPS (e.g. a reverse proxy or tunnel) and use that URL.

## Project layout

| Folder | Contents |
| ------ | -------- |
| `lib/routes/` | One file per screen, wired together in `lib/provider/router_provider.dart` (go_router) |
| `lib/components/` | Reusable widgets, grouped by feature (`trail/`, `map/`, `route_planner/`, …) |
| `lib/provider/` | Riverpod providers — app state, API access, settings |
| `lib/models/` | Immutable data models (freezed) |
| `lib/entities/` | ObjectBox entities for local persistence (offline trails, recordings) |
| `lib/services/` | Long-running services (trail downloads, tile proxy, position sources) |
| `lib/i18n/` | ARB translation files and generated localizations |

## Analysis & tests

Before committing, make sure static analysis and the test suite pass:

```bash
flutter analyze
flutter test
```
