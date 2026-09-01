---
title: G3b — Search-Integration Core
description: Föderierte räumliche Suche mit ACL, Bestandsfiltern, globaler Sortierung, korrektem Direct-total, Pagination und Cursor.
editUrl: false
sidebar:
  order: 3
  badge: Blockiert
spec:
  id: G3b
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: CONTEXT
  productSlice: geo-discovery
  exposure: cutover
  implementationDependsOn: [G3a, SRCH1b]
  releaseGates: [SEC-VIS-1, SEC-AUTH-1, SEC-DUR-1, SEC-SCOPE-1]
  normativeSources: [ADR-0001, ADR-0002, SRCH-V1-CONTRACT, STATE1-CONTRACT, FEDERATION-SECURITY-V1]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch G3a und SRCH1b |
| Implementierungsabhängigkeiten | G3a, SRCH1b |
| Releasegates | SEC-VIS-1, SEC-AUTH-1, SEC-DUR-1, SEC-SCOPE-1 |
| Nicht enthalten | räumliche Optionscounts/Histogramme und Proximity-Sortierung |

## Nutzerergebnis und Releaseeinheit

G3b stellt den vollständigen serverseitigen Punkt-Radius entlang der Route
bereit. Ein normalisierter Suchauftrag kombiniert Raumprädikat, Text,
Bestandsfilter, ACL, Federation und globale Sortierung und liefert Treffer,
korrektes `total` der tatsächlich berechneten Direct-Menge sowie stabile
Pagination. L4 kann diesen Vertrag anschließend sichtbar machen.

## Scope

- Eine snapshotgebundene Meilisearch-Multi-Search über alle gebundenen
  Route-Shards orchestrieren.
- Den V1-Auftrag einschließlich ACL, Federation, Text, Bestandsfiltern,
  räumlichem Prädikat und Sortierung auf jeden Shard anwenden.
- Shardergebnisse global deduplizieren, stabil sortieren und paginieren, ohne
  unbeschränkt den Katalog im Speicher zu materialisieren.
- `total`, zurückgegebene Seite, `next_cursor` und Treffer aus demselben
  Direct-Snapshot liefern. V1 gibt keine Seitenzahl oder Gesamtseitenzahl aus.
- Cursor, Suchkontext, Generation, Spatial-Vertrag, Geometrieversion,
  Neutralband und Gültigkeitsgrenzen binden.
- Cache- und Fehlerkontext vollständig auf Principal, Scope, Generation,
  Revision, Spatial-Profil und Request normalisieren.
- Route-Proximity-Sortierung mit typisiertem 4xx ablehnen.

## Genauigkeitsvertrag

`total` und Seite sind exakt für die von Meilisearch berechnete Direct-Menge.
Das räumliche Prädikat selbst folgt dem statistisch qualifizierten
`route_radius_ux_v1` und ist gegenüber der mathematischen Rohgeometrie nicht
garantiert exakt. Diese Unterscheidung wird einmal zentral erklärt; einzelne
Zahlen erhalten kein irreführendes `≈`.

## Nichtziele

- Keine SRCH3-Facetten oder Histogramme; diese folgen in G3b-AGG.
- Keine exakte Punkt-zu-Linie-Nachprüfung und kein stiller H3-Fallback.
- Keine Startpunktnähe als Ersatz für nicht unterstützte Routen-Proximity.
- Keine partielle Seite oder Count-Antwort nach Shardfehler.

## Abnahme

- ACL-, Federation-, Text- und Filterabweichungen sind harte Nullfehler.
- Direct-`total`, globale Reihenfolge, zurückgegebene Seite und `next_cursor`
  stimmen mit der vollständigen gebundenen Direct-Menge überein.
- Cursorwechsel über Revision, Generation, Principal, Radius oder Spatial-Profil
  werden abgelehnt.
- Shardfehler, Retentionablauf und Restriction-Fence liefern typisierte,
  fail-closed Fehler statt partieller Erfolge.
- G1-Produktgate, Pagination-/Top-10-/Nearest-Metriken und Warm-Stabilität laufen
  im echten ACL-/Federation-Kontext.

## Mögliche Review-Schnitte

Compiler/Plan, Shard-Orchestrierung, globales Merge/Paging und Gatewaybindung
können getrennte PRs sein. Kein Teilschnitt wird als räumliche Produktsuche
freigegeben, bevor die vollständige Abnahme gemeinsam besteht.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | G3b-Core hängt nicht von SRCH3 ab; Aggregationen bleiben G3b-AGG. |
