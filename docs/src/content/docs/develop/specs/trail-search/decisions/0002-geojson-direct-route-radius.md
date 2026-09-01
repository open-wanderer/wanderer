---
title: "ADR 0002: GeoJSON Direct für den Routenradius"
description: Angenommene Architektur- und Produktentscheidung für den approximativen Routenradius.
editUrl: false
sidebar:
  order: 2
  badge: Angenommen
spec:
  id: ADR-0002
  kind: decision
  status: accepted
  capability: CONTEXT
  lastReviewed: '2026-08-30'
---

- Status: angenommen
- Datum: 30. August 2026
- Freigabe: für den vorgesehenen Wanderer-Release qualifiziert
- Entscheidungsartefakt: G1
- Implementierungsbezug: G2 für die Suchgeometrie, G3a/G3b für Projektion
  und Runtime; die Codes bezeichnen veränderbare Bausteinschnitte im
  [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
- Betrifft: `route_radius_ux_v1`; nicht den bestehenden Startpunktradius
- Evidenzwerkzeug: `db/cmd/geobench/README.md`

## Kontext

Wanderer benötigt zusätzlich zur bestehenden Startpunktsuche einen Modus, der
eine Route findet, sobald eines ihrer realen Segmente einen Radius um einen
Punkt berührt. Startpunkt, GPX-Darstellung und Kartenpolyline beantworten
diesen Suchauftrag nicht.

Verglichen wurden ein direkter GeoJSON-Pfad in Meilisearch, GeoJSON als
konservativer Kandidatenindex mit exakter Nachprüfung, H3 mit exakter
Nachprüfung sowie SQLite RTree. Die zweistufigen Pfade können mathematisch
exakte Ergebnisse liefern, benötigen dafür aber die vollständige
Kandidatenmenge, Geometrie-Hydration und die exakte Distanzprüfung vor Count,
Sortierung und Pagination. Der Direct-Pfad vermeidet diese zweite Stufe, nimmt
dafür aber seltene, messbare räumliche Abweichungen in Kauf.

G1 entscheidet Architektur und Produktvertrag. Der produktförmige synthetische
Harness bildet die für diese Entscheidung benötigten Geo-, ACL-, Text-,
Filter-, Count-, Sortier-, Paging- und Parallelitätsfälle ab. Seine
dokumentierten Grenzen sind für den vorgesehenen Wanderer-Release akzeptiert;
eine zusätzliche Qualifikation auf einer bestimmten NAS-Hardware ist keine
Freigabevoraussetzung.

## Entscheidung

### 1. Produktvertrag

`route_radius_ux_v1` verwendet GeoJSON Direct. Meilisearchs gefilterte,
sortierte und paginierte Antwort ist das Endergebnis. Im Requestpfad gibt es
keine versteckte SQLite-Hydration und keine nachgelagerte exakte
Punkt-zu-Polyline-Prüfung.

Der Vertrag gilt für WGS84-Punktanker und

```text
0 < radius_m <= 100000
```

Grössere oder ungültige Radien werden typisiert abgelehnt. Der bestehende
Startpunktradius bleibt ein eigener Modus und fällt nie still auf den
Routenradius oder umgekehrt zurück.

Die Direct-Auswertung ist bewusst approximativ. `bounded_approximate`
bezeichnet dabei einen statistisch gegen ein festgelegtes Corpus
qualifizierten Produktvertrag, keine Fehlergrenze für eine einzelne Anfrage
und keine geometrische Distanzschranke. Bewertet wird sie gegen eine
unabhängige exakte sphärische Punkt-zu-Polyline-Referenz mit einem neutralen
Band von 50 Metern beidseits der angefragten Radiusgrenze. Pro verbindlichem
Qualifikationslauf gelten aggregiert mindestens 99 Prozent für:

- Raw- und Clear-Interior-Recall;
- materielle Precision ausserhalb des neutralen Bands;
- Top-10- und Nearest-Coverage;
- Erfolg von Anfragen mit klar erwarteten Treffern;
- Recall und Precision vergleichbarer Produktseiten.

Einzelne Boundary- oder materielle Abweichungen und False-Empty-Fälle bleiben
im Report sichtbar, entscheiden den Vertrag aber über diese Aggregate. Falsche
ACL-/Text-/Filtersemantik, semantische False Positives, abgeschnittene
Antworten und instabile Warm-Ergebnisse sind dagegen harte Nullfehler.

Das 50-Meter-Band ist eine Produkt- und Auswertungsregel, keine bewiesene
geometrische Worst-Case-Grenze. Insbesondere begründet weder die S50-
Vereinfachung noch ihre Kombination mit der r100-Kreisapproximation eine
globale Hausdorff-Garantie von 50 Metern. Eine spätere deterministische
Fehlerobergrenze wäre ein neuer, separat zertifizierter Geometrievertrag.

Das öffentliche `total`, die zurückgegebene Seite, `next_cursor` und künftige
Count-Gruppen müssen exakt zur tatsächlich gelieferten Direct-Menge desselben
Snapshots und Filterkontexts passen. V1 gibt keine Seiten- oder
Gesamtseitenzahl aus. Gegenüber der mathematisch exakten Rohgeometrie dürfen
die Werte wegen des räumlichen Direct-Prädikats abweichen. Das G1-Count-Gate
gilt ausschliesslich für das intern als `totalHits` gemessene öffentliche
`total`: Im Offline-Audit beträgt sein p95 höchstens
ein Prozent relative Countabweichung. Facetten- und Histogrammserien waren
nicht Teil dieser G1-Evidenz; sie erhalten erst bei ihrer Einführung durch
SRCH3 eine eigene Qualifikation gegen dasselbe Oracle. Diese theoretische
räumliche Abweichung wird einmal zentral in UI und API dokumentiert. Wanderer
setzt nicht vor jede ansonsten exakte Zahl ein `≈`.

Eine Sortierung nach der nächsten Stelle der Route ist in diesem Vertrag nicht
verfügbar. UI und Capability bieten sie nicht an; API, gespeicherte URL und
Chat-Compiler antworten mit einem typisierten 4xx. Eine unsortierte oder nach
Startpunktnähe sortierte Erfolgsantwort ist unzulässig.

### 2. Index- und Queryprofil

G1 qualifiziert das offizielle Meilisearch-Image 1.53.1 mit folgendem Profil:

- Route als segmenterhaltender GeoJSON-`MultiLineString` beziehungsweise
  `LineString` im separaten Route-Search-Pfad;
- Ramer-Douglas-Peucker-Vereinfachung mit 50 Metern Toleranz (`S50`) nur für
  die indexierte Suchprojektion;
- anschliessende geodätische Verdichtung auf höchstens 5 Kilometer lange
  Segmente und explizite Teilung am Antimeridian;
- `_geoRadius(lat, lon, radius_m, 100)` ohne Padding (`r100`);
- dieselben ACL-, Text-, Federation- und Metadatenprädikate in jeder
  Shard-Subquery;
- eine federierte Meilisearch-Multi-Search pro logischer Anfrage für globale
  Sortierung, zurückgegebene Seite, Cursorgrundlage und `totalHits`.

Die getrennte Route-Indexfamilie ist eine Paritätsanforderung und keine blosse
Skalierungsoptimierung. Meilisearch berücksichtigt bei `_geoRadius` sonst
`_geo` und `_geojson` desselben Dokuments. Läge die Routengeometrie im
Bestandsdokument, würde der bisherige Startpunktfilter still zu einem
Startpunkt-oder-Routenfilter. Deshalb enthält der Hauptpfad für
`start_point_radius` nur `_geo`, der Route-Pfad für `route_geometry_radius`
nur `_geojson`; beide binden denselben Geometry-Hash und dieselbe
Security-/Delivery-Watermark. Fehlende Routengeometrie fällt nie auf
Startpunktsemantik zurück.

Der historische Harness-Bezeichner `r100-exact` bedeutet lediglich „exakt der
angefragte Radius ohne konservatives Padding“. Er behauptet weder exakte
Routengeometrie noch exakte räumliche Resultate.

Es werden ausschliesslich offizielle Meilisearch-Releases eingesetzt. Ein
eigener Meilisearch-, Cellulite- oder h3o-Build ist ausgeschlossen. 1.53.1 ist
der G1-Evidenzstand. Eine neuere offizielle Version erhält ein neues
unveränderliches Engineprofil und eine neue Indexgeneration. Sie gilt erst
nach bestandenem GeoJSON-Contract, erneut bestandenem 99-Prozent-UX-Gate und
den normalen operativen Upgrade-, Dump/Restore-, Rollback- und Soak-Prüfungen
als gleichwertig oder besser. Allein die Engineversion ändert weder den
öffentlichen Spatial-Vertrag noch dessen Payloadschema.

### 3. Append-orientierte Shards

Ein Route-Index enthält höchstens 1.000 Trails. Die Zahl der Shards ist daher
nicht global fixiert: Ist der aktuelle offene Shard voll, wird der nächste
angelegt.

Die Erstzuordnung folgt `(local_created_at, trail_id)`. Massgeblich ist das
lokale, nach der Aufnahme unveränderliche Erstellungsdatum. Ein originales
föderiertes Erstellungs- oder Publikationsdatum darf die Zuordnung nie
beeinflussen. `geo_shard_id` und Shardmanifest werden persistent gespeichert.
Update, Delete oder Federation-Sync verschieben keinen vorhandenen Trail;
gelöschte Plätze werden nicht durch Repacking aufgefüllt.

Die Indexfamilie ist versioniert und generationengebunden. Wanderer löscht
oder baut sie beim normalen Start nicht vollständig neu. Backfill,
Schemawechsel und Vollrebuild laufen als Schattenindex; erst nach bestätigtem
Cutoff und vollständigen Watermarks wird atomar umgeschaltet.

### 4. Known Limitations und Upgradevertrag

Die Contract-Fixtures für `LineString` und `MultiLineString` bestehen auf
Meilisearch 1.53.1. Zwei zusätzliche reproduzierbare Diagnosen zeigen dennoch
seltene Lücken im offiziellen GeoJSON-Leser:

- `geojson_cell_coverage`: Eine Mischung aus sphärischer H3-Zellzuordnung und
  planaren Zellpolygonen kann gespeicherte Zellen auslassen. Der reduzierte
  Fall tritt auch bei gewöhnlicher Breite auf und ist deshalb weder ein reiner
  Polar-, Pentagon- noch Local-IJ-Fehler.
- `geojson_hierarchical_split`: In einer dicht geteilten Zelle liefert ein
  GeoJSON-Polygon 4.980 von 5.000 Kontrollen, während der native
  Bounding-Box-Pfad 5.000 von 5.000 liefert.

Beide sind unter dem aggregierten 99-Prozent-Vertrag akzeptierte, sichtbare
Engine-Limitierungen. Sie werden bei jedem Meilisearch-Upgrade erneut
ausgeführt. `known_engine_limitation` bleibt nicht-gating; kann eine Diagnose
selbst nicht vollständig ausgeführt werden, ist `diagnostic_error` gating und
der Contractlauf schlägt fehl. Eine neue Version verbessert den Vertrag nicht
allein durch ihre Versionsnummer, sondern erst durch die Messung.

## Evidenz

Der angenommene 10k-G1-Lauf verwendete Seed `20260830`, 60 Querypunkte, die
Radien 0,5/5/25/100 Kilometer, vier Clients, 240 vollständige räumliche Fälle
und 240 Warm-Produktsamples.

| Merkmal | 10k-G1-Ergebnis |
|---|---:|
| Shards / Trails | 10 / 10.000 |
| räumlicher Raw Recall | 99,97 % |
| Clear Recall / materielle Precision | 100 % / 100 % |
| Top-10 / Nearest / Non-Empty | 100 % / 100 % / 100 % |
| materielle FN / FP | 0 / 0 |
| Produkt p50 / p95 / p99 | 65 / 183 / 338 ms |
| Durchsatz | 49 Queries/s |
| exakter Count / innerhalb 1 % / abgeleitete Seitengrenze | 57/60 / 60/60 / 60/60 |
| vollständiger Indexaufbau | 10,2 s |
| Meilisearch-Datenbank | 52,7 MiB |
| Engine-Peak-RSS-Delta | ca. 1,2 GiB |

Die historische dritte Harnessmetrik wurde ursprünglich `totalPages` genannt.
Sie prüfte eine intern aus Count und Seitengröße abgeleitete Seitengrenze und
ist kein Feld der öffentlichen V1-Response.

Evidenz- und Identitätsanker:

- Harness:
  `sha256:b65bfd9c31f15edb2d697743551237951ab6642a3a1da5251a2d86d1c82c4867`
- Dataset:
  `sha256:9f94f324539c0c231f83b1c21f7fd3a2ec59eacc5c9ef375230329f0e62bbdc0`
- Executable:
  `sha256:acc700fe190faf67371b6224b22bd241c3349257a7c2eb2dc049e1b0637fe535`
- Meilisearch-Image:
  `getmeili/meilisearch@sha256:8d6643d86d71fad6ad3cba92cde7ccfce9e4d6c384bda67598eb553571c32431`

Der Lauf hatte `performance_target=none`, führte im Capacity-Search-Modus
keine Mutationen aus und meldete `gitDirty=true`. Die obigen Digests
identifizieren die dabei verwendeten Harness-Quellen, das Dataset, das
Executable und das Image; ein Digest ersetzt nicht die Archivierung der
zugehörigen Bytes. Diese Eigenschaften sind für den beschlossenen
Releaseumfang dokumentierte Evidenzgrenzen und keine nachgelagerten
Freigabeblocker. Die
anschliessende Harness-Finalisierung korrigiert Planmetadaten,
Reproduzierbarkeitsfelder und
Contract-Gating, nicht den gemessenen Query- oder Indexalgorithmus. Sie erhält
bei neuen Läufen folgerichtig einen neuen Harness- und Executable-Digest.

Ein separater synthetischer 50k-Lauf mit 50 Shards bestätigt die
Skalierungsrichtung: 99,95 Prozent räumlicher Raw Recall, 100 Prozent Clear
Recall und materielle Precision, 8,5 Queries/s, 827 ms Produkt-p95, 85,6
Sekunden Rebuild inklusive Setup und rund 262 MiB Datenbank. Er lief jedoch
mit 32 GiB RAM, nur 120 Warm-Samples, ohne Performance-Target und nicht aus
einem sauberen Source-Stand. Er ist zusätzliche Skalierungsevidenz, aber keine
Voraussetzung für die Releaseentscheidung.

## Folgen

- Der Routenradius kann ohne zweite Geometriedatenbank im Requestpfad
  umgesetzt werden.
- Räumliche Resultate und ihre mathematischen Oracle-Counts können selten
  voneinander abweichen. Die Abweichung wird zentral erklärt und fortlaufend
  gemessen.
- Der bestehende `_geo`-Startpunktindex und die GPX-/Polyline-
  Kartendarstellung bleiben unverändert.
- Eine Heatmap ist keine kostenlose Folge des GeoJSON-Index. Sie darf später
  als getrennte, asynchron abgeleitete Analyseprojektion entstehen; H3 kann
  dafür neu bewertet werden, ohne heute einen parallelen Suchindex zu
  rechtfertigen.
- „Route verläuft durch den Kartenausschnitt“ kann denselben kanonischen
  GeoJSON-Datenpfad nutzen, benötigt wegen Bounding-Box-, Antimeridian- und
  Engine-Limitierungen aber einen eigenen Contract- und Release-Test.
- Ein Radius über 100 Kilometer oder eine Suche gegen echte Regionsflächen
  benötigt einen neuen Spatial-Vertrag.

## Verworfene Alternativen

### H3 parallel zu GeoJSON

Verworfen. Ein paralleler H3-Suchindex verdoppelt Projektions-, Write-,
Speicher- und Betriebslogik, ohne für den entschiedenen Routenradius einen
Produktvorteil zu liefern.

H3 mit vollständiger Geometrie-Hydration und exakter Prüfung ist der erste
geordnete Fallback, falls der Direct-Pfad später seine verbindlichen
Regressionstests oder den dokumentierten UX-Vertrag nicht mehr besteht. Der
Wechsel erfolgt als eigener kompletter Vertical Slice und nie als stiller
Mischbetrieb.

### SQLite RTree

Verworfen. RTree ist ein SQLite-Modul für räumliche Bounding-Box-Indizes. Für
den Wanderer-Suchauftrag bliebe es nur ein Kandidatengenerator; ACL, Text,
Counts, globale Sortierung und Pagination müssten mit Meilisearch und exakter
Segmentprüfung orchestriert werden. Die zusätzliche persistente Projektion und
die gemessene Last in dichten grossen Radien rechtfertigen diese Komplexität
nicht.

### GeoJSON Safe/Exact als Standard

Nicht als V1-Standard gewählt. Der konservativ erweiterte Radius plus exakte
Nachprüfung bleibt ein vermessener möglicher späterer Exact-Modus. Er führt
aber genau die Hydration, Vollmengenprüfung und Count-/Paging-Orchestrierung
ein, die Direct vermeidet, und beseitigt die vorgelagerten Engine-Limitierungen
nicht automatisch.

### Eigener Meilisearch-Build

Verworfen. Ein Fork würde Security-, Upgrade-, Image- und Supportkosten in
Wanderer verlagern. Known Limitations werden dokumentiert und offizielle
Releases durch Contract-, UX-Regression und operative Upgradeprüfungen
qualifiziert.

## Releaseentscheidung und Regression

Mit diesem ADR ist G1 abgeschlossen und GeoJSON Direct für den vorgesehenen
Release qualifiziert. Ein zusätzlicher 50k-Lauf, ein bestimmter RAM-Ausbau oder
eine bestimmte NAS-Hardware sind keine Release-Gates. Die vorhandenen `nas`-
und DS923+-Profile des Harness bleiben optionale Last- und
Kapazitätsdiagnosen für Betreiber mit entsprechend grossen Katalogen.

Verbindlich bleiben die in diesem ADR festgelegten fachlichen und technischen
Verträge: das aggregierte 99-Prozent-UX-Gate, harte Nullfehler für ACL-/Text-/
Filtersemantik und Truncation, die GeoJSON-Contracts samt ausführbaren
Diagnosen, höchstens 100 Kilometer Radius, stabile Capacity-Shards sowie ein
nicht blockierender persistenter Index-Lifecycle. Diese Regeln werden durch
normale Unit-, Contract- und Integrationstests geschützt, nicht durch eine
einmalige Hardwareabnahme.

Scheitert GeoJSON Direct später an diesen Verträgen, wird H3 plus exakte
Prüfung als nächster vollständiger Pfad bewertet. Es gibt keinen automatischen
Fallback pro Anfrage.

Dieses ADR ersetzt für G1 widersprechende Vorentscheidungen in den
Konzeptentwürfen, insbesondere S0 statt S50, 99,5 statt 99 Prozent und harte
Nullfehler für jede einzelne räumliche Abweichung. Die betreffenden Dokumente
werden separat an diese Entscheidung angepasst.

## Referenzen

- [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
- Geo-Benchmark-Harness und Messvertrag: `db/cmd/geobench/README.md`
- [Meilisearch GeoJSON](https://www.meilisearch.com/docs/capabilities/geo_search/how_to/use_geojson_format)
- [Meilisearch-Issue 6575](https://github.com/meilisearch/meilisearch/issues/6575)
