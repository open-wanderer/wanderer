---
title: G3a — Spatial Runtime
description: Generationengebundene append-orientierte Route-Indexfamilie für den GeoJSON-Direct-Vertrag.
editUrl: false
sidebar:
  order: 2
  badge: Blockiert
spec:
  id: G3a
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: CONTEXT
  productSlice: geo-discovery
  exposure: shadow
  implementationDependsOn: [G2, IDX3]
  normativeSources: [ADR-0002, STATE1-CONTRACT, FEDERATION-SECURITY-V1]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch G2 und IDX3 |
| Implementierungsabhängigkeiten | G2, IDX3 |
| Exposure | Shadow/Runtime, noch kein Benutzervertrag |
| Ergebnis | produktionsfähige räumliche Indexfamilie gemäß ADR 0002 |

## Scope

- Separate generationengebundene Route-Indexfamilie mit `_geojson`, aber ohne
  `_geo`, aufbauen.
- Höchstens 1.000 dauerhaft zugeordnete Trails pro append-orientiertem Shard.
- Persistentes Shardmanifest und `geo_shard_id`; gelöschte Plätze nicht durch
  stilles Repacking füllen.
- G2-Geometrie und erforderliche Filter-/ACL-/Federationfelder atomar
  projizieren.
- Add, Update und Delete einschließlich Tombstones idempotent abbilden.
- Vollaufbau, Cutoff-Replay, Shadow-Verifikation und Familien-Swap nach IDX3.
- Direct-Anfragen mit `_geoRadius(lat, lon, meter, 100)` ohne versteckten
  Exact-Finalizer ausführen können.
- Generation, Spatial-Profil, Geometrieversion und Watermarks diagnostizierbar
  machen.

## Nichtziele

- Keine Search-API, globale Pagination oder UI; das gehört zu G3b/L4.
- Keine `_geo`-Startpunktfelder in der Route-Indexfamilie.
- Kein H3-, RTree-, Buffer- oder Safe-Exact-Pfad neben Direct.
- Kein unbegrenztes Repacking oder Shard-Fan-out ohne Manifestvertrag.

## Migration und Rollback

Der Aufbau erfolgt ausschließlich als neue Generation. Livegeneration und
Rollbackkandidat werden nicht geleert oder in-place umgebaut. Erst vollständige
Projektion, Replay, Taskbestätigung, Contracttests und Watermarkkonvergenz
erlauben den Pointerwechsel. Eine versiegelte Altgeneration bleibt nur für ihre
gebundenen Cursor lesbar und wird nie reaktiviert. Für einen Rollback wird nach
dem Zustandsvertrag ein getrennter Kandidat auf neuen physischen IDs bis zum
aktuellen Cutoff aufgeholt und erst danach per Pointer-CAS aktiviert.

## Abnahme

- Shardkapazität, dauerhafte Zuordnung und Manifest überstehen Add/Update/Delete,
  Rebuild und Neustart.
- Hauptindex und Route-Index stammen aus demselben Fach-/Geometriehash und
  Visibility-Fence.
- Kein nicht autorisiertes oder tombstonetes Dokument erscheint in der
  gebundenen Generation.
- G0/G1-Contracts und Known-Limitation-Diagnosen bestehen auf dem gepinnten
  Engineprofil.
- Swap, Recovery und Rollback werden mit realen Meilisearch-Tasks getestet.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Startpunkt- und Routengeometrie bleiben wegen Meilis `_geoRadius`-Semantik in getrennten Indexfamilien. |
