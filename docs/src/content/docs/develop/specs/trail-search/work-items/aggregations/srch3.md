---
title: SRCH3 — Counts, Ranges und Histogramme
description: Exhaustive disjunktive Optionscounts und versionierte numerische Verteilungen im gebundenen Suchkontext.
editUrl: false
sidebar:
  order: 1
  badge: Blockiert
spec:
  id: SRCH3
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-aggregations
  exposure: operational
  implementationDependsOn: [IDX3, SRCH1b, SRCH2, A0]
  releaseGates: [SEC-VIS-1, SEC-AUTH-1, SEC-DUR-1, SEC-SCOPE-1]
  normativeSources: [SRCH-V1-CONTRACT, STATE1-CONTRACT, FEDERATION-SECURITY-V1]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch IDX3, SRCH1b, SRCH2 und A0 |
| Releasegates für föderierte Counts | SEC-VIS-1, SEC-AUTH-1, SEC-DUR-1, SEC-SCOPE-1 |
| Exposure | API/Optionen; sichtbare Darstellung folgt in SRCH4b |
| Nicht enthalten | räumliche Bindung, die separat in G3b-AGG erfolgt |

## Nutzerergebnis und Releaseeinheit

SRCH3 liefert belastbare Optionszahlen und numerische Verteilungen für die
jeweils berechnete Ergebnismenge. Nutzer können dadurch erkennen, welche
Auswahl noch Treffer besitzt und wo reale Distanz-, Höhen- oder Dauerwerte
liegen, ohne durch `maxTotalHits`, Top-k-Facetten oder Ausreißer getäuscht zu
werden.

## Semantik

- Ein Optionscount ist disjunktiv: Er verwendet alle aktiven Filter außer der
  eigenen Optionsgruppe und simuliert anschließend die betreffende Option.
- Ausgewählte Optionen bleiben mit Count 0 sichtbar.
- Erfolgreiche Gruppen sind exhaustiv für ihre gebundene Ergebnismenge.
- Eine fehlgeschlagene Gruppe liefert einen typisierten Gruppenfehler und keine
  teilberechneten Zahlen.
- Counts, Histogramme, Treffer und Suchkontext binden Principal, ACL,
  Federation, Generation, Revision, Capability- und Policyversion.
- `unknown` und Presence besitzen eigene, vertraglich definierte Zustände.

## Scope

- Options-Lookup und Aggregationsplan nach Trail-Suchvertrag implementieren.
- Maximal 16 angeforderte Aggregationsgruppen insgesamt, 32 intern kompilierte
  Search-Queries insgesamt und 16 Buckets je Histogrammgruppe einschließlich
  `unknown_bucket` erzwingen. Optionen oder Buckets erzeugen nicht je eine
  eigene Query.
- Stabile, versionierte Bucket-IDs und offene Endbereiche verwenden.
- Aktivitätsfamilien-spezifische p99-/Cap-Domains aus A0 anwenden.
- Distanz, Aufstieg und Abstieg als ersten Range-Slice qualifizieren.
- Cache-Key und Invalidation über alle Policy-, Actor-, Snapshot-, Generation-
  und Capabilityfelder vollständig halten.
- Federation-, Expiry-, Swap- und Katalogtests oberhalb von 1.000 Treffern.

## Nichtziele

- Keine UI; diese gehört zu SRCH4b.
- Keine dynamisch pro Request neu erfundenen Bucketgrenzen.
- Keine abgeschnittenen Facetten als exhaustive Counts deklarieren.
- Keine räumlichen Aggregationen ohne G3b-AGG.

## Abnahme

- Oracle-Tests vergleichen jede Count-Gruppe mit einer vollständigen
  disjunktiven Auswertung der gebundenen Ergebnismenge.
- Ausgewählte Nulloptionen, Unknown/Presence und offene Endbereiche bleiben
  stabil über URL/Request-Roundtrips.
- Cache-, Cursor-, Federation-, ACL- oder Generationwechsel können keine Zahlen
  aus einem anderen Kontext wiederverwenden.
- Budgetüberschreitungen und Gruppenfehler liefern keine partiellen Zahlen.
- p95/p99 und Payload bleiben innerhalb der vor Freigabe kalibrierten Budgets.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Counts sind ein eigener Slice und weder Voraussetzung für SRCH4a noch für L4. |
