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
  implementationDependsOn: [SRCH-COMP, IDX0]
  releaseGates: [SEC-VIS-0]
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-09-01'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch SRCH-COMP und IDX0 |
| Implementierungsabhängigkeiten | SRCH-COMP, IDX0 |
| Sichtbares Releasegate | SEC-VIS-0 |
| Ergebnis | vollständiger Bestandsumfang, korrekte Unknown-Difficulty, echte Subkategorien und Sortierung |

## Nutzerergebnis und Releaseeinheit

Nutzer können den heutigen Datenbestand über die erhaltenen Filter und
Sortierungen sowie eine echte Kategorie-/Subkategoriehierarchie durchsuchen.
Belegte Fehlsemantiken werden korrigiert, ohne den vorhandenen
Startpunktradius oder die gespeicherte Quellschwierigkeit zu entfernen. SRCH2
besitzt dabei die Ende-zu-Ende-Korrektur für eine fehlende, leere oder
ungültige Quellschwierigkeit: Sie bleibt unbekannt und wird weder als `easy`
angezeigt noch so gefiltert.

## Scope

- Alle in SRCH0 bestätigten Filter und Sortierungen über SRCH-COMP anbieten.
- Kategorien und Subkategorien mit stabilen IDs und vererbter Auswahl abbilden.
- Vorhandene Taxonomie-, Tag- und Waypoint-Terme nach der vertraglichen
  Freitextregel indexieren.
- Lokale Likes und vorhandene Sortierfelder korrekt projizieren.
- Den von SRCH-COMP gelieferten Startpunktradius unverändert übernehmen; SRCH2
  besitzt dafür kein zweites Korrektur-Overlay.
- Die Quellschwierigkeit mit expliziter Presence-/Unknown-Semantik projizieren,
  filtern, sortieren und in allen Trefferansichten darstellen.
- Fehlende, leere und unbekannte Quellstrings als `difficulty: null` sowie mit
  einem ausschliesslich internen Presence-Sortkey abbilden; bekannte Werte
  bleiben `easy`, `moderate` und `difficult` beziehungsweise `0`, `1` und `2`.

## Nichtziele

- Keine persönliche Schwierigkeit, Ratings, Surface- oder Wetterbewertung.
- Keine disjunktiven Optionscounts oder Histogramme; diese gehören zu SRCH3.
- Keine räumliche Suche entlang der Route; diese gehört zu G2–L4.

## SRCH0-Delta-Overlay

SRCH2 konsumiert den unveränderten SRCH0-Referenzbestand, bindet sein Overlay
an `SRCH0-GAP-DIFF-001` und hält die korrigierten Erwartungen in einem eigenen
Ziel-Overlay. Es überschreibt keine Baseline-Goldens und deklariert
ausschliesslich die Difficulty-Deltas:

| Fallfamilie | SRCH2-Ziel |
| --- | --- |
| bekannte Quellwerte | `easy`, `moderate`, `difficult` bleiben `0`, `1`, `2` |
| missing, leer oder ungültig | Projektion `difficulty: null`, Treffer und UI `unknown`, nie `easy` |
| Bestandsdefault `[0,1,2]` | keine Difficulty-Klausel; Unknown bleibt im Suchuniversum |
| echte Teilmenge | numerischer Filter; Unknown trifft nicht |
| Sortierung auf-/absteigend | bekannte Werte vor Unknown, danach die gewählte Difficulty-Richtung |
| Full-, Create- und Patch-Projektion | ein früherer Zahlenwert wird bei Wechsel auf Unknown gelöscht |

Das Overlay trägt `delivery_owner: SRCH2`. Das interne Presence-Feld ist kein
öffentliches DTO-Feld, kein akzeptierter Clientfilter und kein öffentlicher
Sortkey. API-Negativfälle aus dem SRCH0-Korpus müssen seine Abfrage und Ausgabe
vor dem Enginezugriff verhindern.

## Migration und Rückweg

SRCH2 ergänzt das interne Presence-Sortierfeld in einem versionierten
Trailprofil und baut die korrigierte Projektion vor Aktivierung vollständig
über den [IDX0-Offline-Rebuild](/develop/specs/trail-search/work-items/engine/idx0/#quieszenz-und-offline-rebuild)
neu auf. SRCH2 liefert dafür Builder und versioniertes Profil; IDX0 besitzt
nur Ausführung, terminale Task-Waits und das mechanische Abschlussgate. Den
vollständigen Dokument- und Settingsverifier liefert und besitzt ebenfalls
SRCH2. Der Backfill muss Full-, Create- und Patch-Projektion auf denselben
Endzustand bringen; ein blosses Weglassen von `difficulty` darf keinen
früheren Zahlenwert im Engine-Dokument zurücklassen. Aktiviert wird erst nach
vollständigem Dokument- und Settingsvergleich sowie erfülltem SEC-VIS-0-Gate.
Rollback führt denselben IDX0-Offline-Rebuild mit dem vor dem Cutover
erhaltenen Builder, Profil und Verifier aus; ein blosses Zurückschalten der
Settings auf den bereits umgeschriebenen Dokumenten ist verboten. Der Radius
besitzt hier weder Migration noch Rückweg. Category-/Subcategory-IDs bleiben über den URL-
Codec stabil, und ein Rollback darf nie bereits bestehende Bestandsfilter aus
der Oberfläche entfernen.

Die im supersedierten SRCH0-Designarchiv durchgearbeitete
[Unknown-Difficulty-Korrektur](/develop/specs/trail-search/evidence/srch0-design-archive-2026-09-01/#technische-korrektur-unknown-difficulty)
und [Reindex-, Vergleichs- und Rückwegexploration](/develop/specs/trail-search/evidence/srch0-design-archive-2026-09-01/#migration-backfill-cutover-und-rollback)
sind dafür ausdrücklich nicht bindende Vorarbeit; SRCH2 und IDX0 entscheiden
ihre normative Umsetzung selbst.

## Abnahme

- Alle SRCH0-`preserve`-Fälle bleiben grün; ausserhalb des Difficulty-Overlays
  entspricht jeder `known_gap`-Fall weiterhin seiner Baselinebeobachtung.
- Das SRCH2-Delta-Overlay ist für bekannte, fehlende, leere und ungültige
  Quellwerte sowie Default, Teilmenge, beide Sortierrichtungen und alle
  Trefferansichten grün.
- Full-, Create- und Patch-Projektion konvergieren nach bekannt→Unknown ohne
  stehengebliebenen numerischen Difficulty-Wert; das interne Presence-Feld ist
  über Request und Response nicht erreichbar.
- Der IDX0-Offline-Rebuild führt auch bei vorher countgleichem, gefülltem
  Trailindex genau einen vollständigen SRCH2-Pass aus und wird erst nach
  terminalen Tasks, IDX0-Strukturread und grünem SRCH2-Dokument-/Settings-
  Verifier freigegeben.
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
| 2026-09-01 | SRCH2 besitzt die Unknown-Difficulty-Korrektur samt Trailprofil, Backfill und SRCH0-Delta-Overlay; die Radiusbereinigung gehört SRCH-COMP. |
| 2026-09-01 | Der vollständige Difficulty-Backfill verwendet IDX0s Offline-Rebuildmechanik; SRCH2 bleibt Owner von Builder, Profil und Zielsemantik. |
