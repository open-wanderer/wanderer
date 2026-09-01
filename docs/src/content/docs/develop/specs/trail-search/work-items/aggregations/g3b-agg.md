---
title: G3b-AGG — Räumliche Aggregationen
description: SRCH3-Optionscounts und Histogramme im selben Spatial-, Snapshot-, ACL- und Federation-Kontext wie G3b.
editUrl: false
sidebar:
  order: 3
  badge: Blockiert
spec:
  id: G3b-AGG
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: CONTEXT
  productSlice: geo-aggregations
  exposure: operational
  implementationDependsOn: [G3b, SRCH3]
  releaseGates: [SEC-VIS-1, SEC-AUTH-1, SEC-DUR-1, SEC-SCOPE-1]
  normativeSources: [ADR-0002, SRCH-V1-CONTRACT, STATE1-CONTRACT]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch G3b und SRCH3 |
| Releasebeziehung | additiv nach L4; kein L4-Gate |
| Ergebnis | Optionscounts und Histogramme für die aktive räumliche Ergebnismenge |

## Nutzerergebnis und Releaseeinheit

Bei aktivem Routenradius beziehen sich sichtbare Filterzahlen und Histogramme
auf dasselbe räumliche Trefferuniversum wie Treffer und `total`. Wanderer zeigt
keine Zahlen aus einer bloß textuell oder nach Startpunkt gefilterten Menge.

## Scope

- SRCH3-Disjunktion auf den vollständigen G3b-Spatial-Plan anwenden.
- Spatial-Vertrag, Radius, Punktanker, Geometrieversion, Generation, Revision,
  ACL, Federation und Policy in Suchkontext, Cursor und Cache binden.
- Options- und Bucketgruppen global über die gebundene Shardfamilie
  zusammenführen.
- Gruppengrenzen, Budget und Fehlervertrag aus SRCH3 unverändert übernehmen.
- Prädikatsgenauigkeit des Direct-Vertrags zentral ausweisen, ohne Counts als
  intern nur ungefähr berechnet darzustellen.

## Nichtziele

- Keine zweite Kandidaten- oder Count-Semantik neben G3b/SRCH3.
- Keine exakte Rohgeometrie-Nachprüfung nur für Aggregationen.
- Keine Freigabe partieller Shard- oder Bucketzahlen.
- Keine Blockade des vollständigen L4-Slices aus Treffer, `total` und Pagination.

## Abnahme

- Erfolgreiche Gruppen sind exhaustiv für die tatsächlich berechnete
  G3b-Direct-Menge.
- Oracle- und Integrationsfälle kombinieren Spatial, Text, Bestandsfilter, ACL,
  Federation, Sortierung und mehr als 1.000 Treffer.
- Shard-, Fence-, Cursor- oder Gruppenfehler erzeugen keine Zahlen aus einem
  Teiluniversum.
- SRCH4b kann bei aktivem Routenradius ausschließlich diese räumlich gebundenen
  Gruppen anzeigen.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | G3b-Core und L4 bleiben ohne Aggregationspflicht vollständig; G3b-AGG folgt separat. |
