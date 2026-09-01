---
title: SRCH2 — Bestehende und billige Suchfelder
description: Bestehende Filterparität, Subkategorien, Sortierungen und belegte Suchkorrekturen auf dem heutigen Indexpfad.
editUrl: false
sidebar:
  order: 4
  badge: Blockiert
spec:
  id: SRCH2
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-panel
  exposure: user-visible
  implementationDependsOn: [SRCH-COMP]
  releaseGates: [SEC-VIS-0]
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch SRCH-COMP |
| Implementierungsabhängigkeiten | SRCH-COMP |
| Sichtbares Releasegate | SEC-VIS-0 |
| Ergebnis | vollständiger Bestandsumfang plus echte Subkategorien und Sortierung |

## Nutzerergebnis und Releaseeinheit

Nutzer können den heutigen Datenbestand über die erhaltenen Filter und
Sortierungen sowie eine echte Kategorie-/Subkategoriehierarchie durchsuchen.
Belegte Fehlsemantiken werden korrigiert, ohne den vorhandenen
Startpunktradius oder die gespeicherte Quellschwierigkeit zu entfernen.

## Scope

- Alle in SRCH0 bestätigten Filter und Sortierungen über SRCH-COMP anbieten.
- Kategorien und Subkategorien mit stabilen IDs und vererbter Auswahl abbilden.
- Vorhandene Taxonomie-, Tag- und Waypoint-Terme nach der vertraglichen
  Freitextregel indexieren.
- Lokale Likes und vorhandene Sortierfelder korrekt projizieren.
- Den Startpunktradius und die in SRCH0 korrigierte exakt einmalige
  Radiusbedingung unverändert übernehmen und erneut gegen die Paritätsmatrix
  prüfen.
- Die in SRCH0 migrierte Quellschwierigkeit mit expliziter
  Presence-/Unknown-Semantik erhalten, ohne eine zweite Korrekturmigration zu
  definieren.

## Nichtziele

- Keine persönliche Schwierigkeit, Ratings, Surface- oder Wetterbewertung.
- Keine disjunktiven Optionscounts oder Histogramme; diese gehören zu SRCH3.
- Keine räumliche Suche entlang der Route; diese gehört zu G2–L4.

## Migration und Rückweg

Neue Projektionsfelder werden versioniert aufgebaut und erst nach vollständigem
Backfill aktiviert. Die Radius- und Difficulty-Korrekturen samt Reindex gehören
kanonisch zu SRCH0 und werden hier nur als Bestand vorausgesetzt und geprüft.
Category-/Subcategory-IDs bleiben über den URL-Codec stabil. Rollback darf die
neue Projektion deaktivieren, aber nie bereits bestehende Bestandsfilter aus der
Oberfläche entfernen.

## Abnahme

- SRCH0-Parität bleibt vollständig grün.
- Parent-, Child- und kombinierte Kategorieauswahl haben deterministische
  Semantik und stabile URL-Roundtrips.
- Unknown-Schwierigkeit wird weder als `easy` angezeigt noch so gefiltert.
- Alle veröffentlichten Sortierungen sind global stabil und besitzen einen
  eindeutigen Tie-Breaker.
- Kein Feld erscheint als Filter, bevor sein Backfill vollständig aktiviert ist.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | SRCH2 bleibt auf dem heutigen Indexpfad und liefert den Datenumfang für SRCH4a. |
