---
title: SRCH-COMP — Bestandsadapter
description: Serverseitiger Normalisierer, Compiler, URL-Codec und Legacyadapter auf dem heutigen Backend und Index.
editUrl: false
sidebar:
  order: 3
  badge: Blockiert
spec:
  id: SRCH-COMP
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-foundation
  exposure: cutover
  implementationDependsOn: [SRCH-V1]
  releaseGates: [SEC-VIS-0]
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED, FEDERATION-SECURITY-V1]
  lastReviewed: '2026-09-01'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch SRCH-V1 |
| Implementierungsabhängigkeiten | SRCH-V1 |
| Releasegate | SEC-VIS-0 für den produktiven gemischten Suchpfad |
| Ergebnis | V1-Vertrag auf heutigem Backend mit genau den drei in SRCH0 klassifizierten Bestandskorrekturen |

## Nutzerergebnis und Releaseeinheit

Nach dem Cutover erzeugt der Browser keine freien Meilisearch-Ausdrücke mehr.
Alle heutigen Suchwege durchlaufen einen serverseitigen, allowlisteten V1-
Normalisierer und Compiler. Für Nutzer bleibt der unterstützte Bestand ausser
den drei ausdrücklich klassifizierten SRCH0-Korrekturen semantisch gleich:
doppelte Karten-Radiusklausel, erfundener Difficulty-Default und die nun rein
lokal aus `ExpandedAll("trails")` berechneten fünf Listenaggregate. Die sichtbare
`0`-Statistik eines nicht lokal materialisierten Federation-Stubs ist damit
kein stilles Paritätsversprechen. Spätere Panel- und Geo-Slices erhalten eine
erprobte API.

## Scope

- `SearchRequestV1` validieren und kanonisieren.
- Normalisierte V1-Filter auf den heutigen Index und seine realen Felder
  kompilieren.
- Bestehende URL-/Store-Zustände über einen expliziten Legacyadapter übersetzen.
- Responses auf die erlaubten V1-DTOs begrenzen.
- Alte und neue Eingabe für alle SRCH0-Fixtures gegeneinander testen.
- Alle heutigen serverseitigen Engine-Consumer einschliesslich Einzel-/Multi-
  Search, Actor-, Profil-, Cluster-, Bounding-Box-, Upload-Duplikat- und
  sonstigen Legacyhilfsrouten über einen gemeinsamen Readiness-/Tenant-Token-
  Adapter führen. Nur dieser Adapter darf nach einem Engine-`403` den intern
  gecachten Token invalidieren, einmal neu holen und denselben normalisierten
  Auftrag genau einmal wiederholen. Bei `search_initializing` und
  `search_manual_intervention` muss derselbe Adapter vor Tokenausgabe und
  Enginezugriff mit `search_unavailable` abbrechen.
- Typisierte Fehler für ungültige Filter, Sortierungen, Radien, Cursor und nicht
  verfügbare Capabilities liefern.

## Nichtziele

- Kein neuer Shadow-Index, kein Federation-Gateway und keine neuen Felder.
- Keine Counts oder Histogramme, die der heutige Pfad nicht vertragsgemäß
  berechnen kann.
- Kein stilles Durchreichen generischer Enginefilter.

## Migration und Rollback

Der neue Adapter läuft zunächst gegen Golden Fixtures und optional im
Response-Shadow ohne ausstellbare Antwort. Der Cutover schaltet alle Browser-
und API-Aufrufe gemeinsam auf den V1-Pfad. Rollback darf auf den alten Adapter
zurückschalten, solange kein neuer URL-State veröffentlicht wurde, den dieser
nicht versteht. Nach öffentlicher Ausgabe neuer V1-Zustände bleibt der
Legacyparser als Eingangsadapter erhalten.

## Abnahme

- Vollständige Parität der SRCH0-Matrix einschließlich ACL-Kontexten.
- Keine freie Engine-Syntax erreicht den Compiler.
- Normalisierung ist deterministisch und idempotent.
- Kanonische URL, API-Auftrag und ausgeführter Suchauftrag bleiben nach
  Roundtrip gleich.
- Negativtests beweisen, dass alte Direktzugänge den Adapter nicht umgehen.
- Jede oben genannte Consumerfamilie verwendet nach der `SEC-VIS-0`-Parent-Key-
  Rotation den gemeinsamen Adapter. Ein alter 24-Stunden-Cookie wird wegen der
  erhöhten Ziel-/Rollback-Epoche vor Verwendung ersetzt; ein erzwungener erster
  `403` führt zu genau einem erfolgreichen Refresh/Retry, ein zweiter endet ohne
  Schleife.
- Jede Consumerfamilie läuft zusätzlich unter
  `search_manual_intervention`: Search-, Token-, Proxy- und Hilfsantworten
  bleiben fail-closed und der Engine-Mock beobachtet exakt null Requests,
  während ein unabhängiger Fachwrite und die Federation-Inbox weiterlaufen.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Der erste reale Vertragsverbraucher läuft auf dem heutigen Backend, nicht erst auf G3b. |
| 2026-08-31 | Alle First-Party-Engine-Consumer teilen Readiness, Tokeninvalidierung und höchstens einen 403-Refresh; Einzelrouten dürfen keinen eigenen Retrypfad besitzen. |
| 2026-08-31 | `search_manual_intervention` öffnet keine versteckte Engine-Nebenroute; jeder First-Party-Consumer scheitert im gemeinsamen Guard vor Token und Engine. |
