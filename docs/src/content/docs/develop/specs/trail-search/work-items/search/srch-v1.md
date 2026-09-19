---
title: SRCH-V1 — Normativer Suchvertrag
description: Lieferartefakte und Abnahme für die gemeinsame typisierte Suchsprache von UI, API, App und Dialog.
editUrl: false
sidebar:
  order: 2
  badge: Blockiert
spec:
  id: SRCH-V1
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-foundation
  exposure: internal
  implementationConditions: [SRCH0 korrigierte Suchbasis abgenommen]
  normativeSources: [SRCH-V1-CONTRACT, ADR-0001, FEDERATION-SECURITY-V1]
  lastReviewed: '2026-09-19'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert bis zur abgenommenen korrigierten SRCH0-Suchbasis |
| Implementierungsabhängigkeiten | keine Work-Item-Kante; Implementierungsbedingung `SRCH0 korrigierte Suchbasis abgenommen` |
| Exposure | intern; noch kein Runtime-Cutover |
| Normative Quelle | [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/) |

## Nutzerergebnis und Releaseeinheit

SRCH-V1 schafft eine einzige, versionierte Sprache für denselben Suchauftrag in
Filterpanel, Karte, API, App und späterem Dialog. Nutzer sehen in diesem Work
Item noch keine neue Funktion; nachfolgende Slices müssen jedoch nicht mehr
Browserparameter oder Meilisearch-Ausdrücke als alternative Fachsemantik
interpretieren.

## Scope

- Den bestehenden normativen Vertrag gegen die vollständige SRCH0-Matrix prüfen.
- Request-, Response-, Treffer-, Count-, Options-, Fehler- und
  Capability-Schemas als versionierte Lieferartefakte bereitstellen.
- Normalisierung, Defaults, `unknown`, Sortierung, Cursorbindung und URL-Codec
  eindeutig und testbar halten.
- Korrigierte Legacyfälle der V1-Normalform zuordnen. Neue Vertragssemantik
  benötigt eigene Ziel-Overlays; historische Beobachtungen bleiben Evidenz
  und ersetzen die bereits geltenden SRCH0-Korrekturen nicht.
- Golden Fixtures für semantisch identische Browser-, API- und URL-Aufträge
  definieren.

## Nichtziele

- Kein Enginecompiler und kein produktiver Gateway.
- Kein Runtime-Produzent von `SearchReadinessV1`; für die heutigen
  Legacyindizes implementiert IDX0 die beiden Endpunkte, spätere Owner
  ersetzen nur deren interne Zustandsquelle.
- Keine zweite generische Filter-AST und keine freien Feld-/Operatornamen.
- Keine Implementierungsdetails aus Meilisearch als öffentliches Schema.

## Abnahme

- Jeder unterstützte SRCH0-Fall mit seiner aktiven Erwartung besitzt genau
  eine gültige V1-Normalform oder einen typisierten Fehler; Owner-Overlays
  benennen jede neue Vertragsabweichung separat.
- JSON-Schemas und Anwendungstypen lehnen unbekannte Felder und unzulässige
  Capability-Kombinationen ab.
- Browser-, App- und Chat-Beispiele erzeugen keine voneinander abweichenden
  Defaults.
- Der Vertrag verweist für Security, Cursor/Snapshot und Control Plane auf deren
  kanonische Dokumente, statt Regeln zu duplizieren.

## Review-Schnitte

Schemaartefakte, Fixture-Korpus und Dokumentabgleich dürfen getrennt reviewed
werden. SRCH-V1 gilt erst als abgeschlossen, wenn sie gemeinsam dieselbe
SRCH0-Matrix bestehen.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Der bestehende Trail-Suchvertrag ist normative Quelle; dieses Work Item operationalisiert ihn. |
| 2026-09-19 | Voraussetzung ist die korrigierte, abgenommene SRCH0-Suchbasis; ein vollständiger Korpus mit bekannten ungefixten Fehlern reicht nicht aus. |
