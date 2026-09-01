---
title: Evidenz und Kalibrierung
description: Nicht normative Messhistorie, Diagnosen, Alternativen und offene Kalibrierung der Trail-Suche.
editUrl: false
sidebar:
  order: 1
  badge: Evidenz
spec:
  id: EVIDENCE
  kind: evidence
  status: draft
  lastReviewed: '2026-08-30'
---

:::caution[Nicht normativ]
Diese Seite protokolliert reproduzierbare Evidenz, verworfene Hypothesen und
noch zu kalibrierende Werte. Sie überschreibt weder die angenommene
[GeoJSON-Direct-ADR](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
noch den [CONTEXT-Vertrag](/develop/specs/trail-search/capabilities/context/)
oder die allgemeinen Such- und Control-Plane-Verträge. Historische Resultate
behalten den Vertrag und die Grenzen, unter denen sie gemessen wurden.
:::

## Bestandsaudit vom 30. August 2026

Diese Momentaufnahme beschreibt den vor dem Suchumbau vorhandenen Stand auf
Commit
[`e6db729c`](https://github.com/open-wanderer/wanderer/tree/e6db729cb73a490a939420e256b176acf9fd8fc3).
Sie ist bewusst nicht normativ: SRCH0 muss die Aussagen vor einer Umsetzung
erneut prüfen und als ausführbare Regressionsmatrix festhalten.

### Meilisearch-Dokument und Suchkonfiguration

Das heutige Trail-Dokument wird in
[`db/util/meilisearch.go:68-107`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/util/meilisearch.go#L68-L107)
gebildet. Es enthält Name, Beschreibung, Ort, Autor, Tags, Kategorie- und
Subkategorie-ID, föderierte Taxonomienamen, Distanz, Auf-/Abstieg, Dauer,
angegebene Schwierigkeit, Datum, Erstellzeit, Sichtbarkeit, Abschlussstatus,
Likes, Shares, Thumbnail, GPX-Referenz, Bounds, vereinfachte Polyline und
`_geo` mit `trail.lat/lon`. `_geo` repräsentiert damit nur den Startpunkt und
nicht die gesamte Route.

Die produktiven Einstellungen in
[`db/main.go:327-343`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/main.go#L327-L343)
durchsuchen in dieser Reihenfolge `author_name`, `name`, `description`,
`location` und `tags`. Wegen der Attributrelevanz kann ein Autorentreffer damit
vor einem Routennamens-Treffer liegen. Kategorie- und Subkategorienamen,
föderierte Taxonomienamen sowie Waypoint-Namen sind nicht durchsuchbar.

Weitere belegte Grenzen:

- Der Browser hängt denselben `_geoRadius` derzeit zweimal an
  ([`web/src/lib/stores/trail_store.ts:918-923`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/web/src/lib/stores/trail_store.ts#L918-L923)).
- Unbekannte Difficulty-Strings werden beim Indexieren zu `0` und damit zu
  „leicht“
  ([`db/util/meilisearch.go:143-153`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/util/meilisearch.go#L143-L153)).
  Der vorhandene Filter bleibt als angegebene Schwierigkeit erhalten; nur
  diese Unknown-Zuordnung ist zu korrigieren.
- Beim Start wird der Trail-Index vollständig geleert und neu aufgebaut
  ([`db/main.go:387-395`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/main.go#L387-L395)).
  Dabei expandiert der Builder Tags, Kategorie, Shares, Likes und Autor für
  jeden Trail erneut
  ([`db/util/meilisearch.go:320-343`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/util/meilisearch.go#L320-L343)).
  Das ist für grosse GeoJSON-Dokumente und zahlreiche Analysefelder weder ein
  inkrementeller noch ein atomarer Rolloutpfad.
- Der heutige `plugin-sync` ist ein direkter Cron-Callback
  ([`db/main.go:215-227`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/main.go#L215-L227)).
  Sein Seitencursor lebt ausdrücklich nur innerhalb eines Laufs
  ([`db/routes/plugin_system_sync.go:277-299`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/routes/plugin_system_sync.go#L277-L299));
  ein persistenter Fortschrittscursor und ein Lease gegen überlappende
  Cron-Läufe fehlen. Diese Ausgangsschwäche begründet die gemeinsame
  JOB1-/SURF3R-Queue, ohne den heutigen Cron zum Zielmodell zu erklären.

### Dauer und Analysefelder

`duration` ist sortierbar, aber nicht filterbar und besitzt keine einheitliche
Provenienz. Eine geroutete Valhalla-Strecke übernimmt die Providerzeit; eine
manuell verbundene Gerade erhält `0`
([`web/src/lib/stores/valhalla_store.svelte.ts:88-93`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/web/src/lib/stores/valhalla_store.svelte.ts#L88-L93)).
GPX-Uploads berechnen Dauer und Höhenmeter clientseitig
([`web/src/lib/models/gpx/gpx.ts:97-155`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/web/src/lib/models/gpx/gpx.ts#L97-L155));
der Pluginimport besitzt daneben eine serverseitige GPX-Berechnung und kann
Providerwerte übernehmen
([`db/plugins/importer/importer.go:204-213`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/plugins/importer/importer.go#L204-L213),
[`db/plugins/importer/importer.go:377-392`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/db/plugins/importer/importer.go#L377-L392)).
Eine kanonische, versionierte serverseitige Routen- und Anstiegsanalyse fehlt.

Routenform, höchster Punkt, isolierte Anstiege, Sternebewertungen, Transit,
Parkplätze, Surface-/Wegtypfakten und Leistungsprofile fehlen ebenfalls im
heutigen Suchmodell. Föderations- und Visibility-Befunde stehen getrennt im
[Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/),
damit diese Momentaufnahme keine zweite Security-Norm erzeugt.

### Nominatim und Ortssuche

`NOMINATIM_URL` erlaubt bereits einen alternativen oder selbst gehosteten
Endpoint. Die Erkennung des öffentlichen OSMF-Dienstes erfolgt aber über einen
URL-Substring
([`web/src/lib/server/nominatim.ts:9-15`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/web/src/lib/server/nominatim.ts#L9-L15))
und ist keine explizite Provider- oder Capability-Erkennung. Der Serverendpoint
akzeptiert `q` und `limit`
([`web/src/routes/api/v1/geocoding/search/+server.ts:5-27`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/web/src/routes/api/v1/geocoding/search/+server.ts#L5-L27));
der Store erhält zwar ein `limit`, überträgt es aber nicht und bildet nur
Freitextparameter ab
([`web/src/lib/stores/search_store.ts:104-127`](https://github.com/open-wanderer/wanderer/blob/e6db729cb73a490a939420e256b176acf9fd8fc3/web/src/lib/stores/search_store.ts#L104-L127)).
Die Antwort enthält bereits rohe `category`-/`type`-Werte, doch ein gewählter
harter Unterkunfts-, Sehenswürdigkeits- oder Parkplatz-Scope kann nicht
providerseitig durchgesetzt werden.

## Chronik der Geo-Benchmarks

Der reproduzierbare Spike verglich die bestehende `_geo`-Startpunktbaseline,
direktes Meilisearch-`_geojson`, konservative GeoJSON-Kandidaten mit exakter
Punkt-zu-Polyline-Prüfung, H3 plus exakter Prüfung, gepufferte Routenkorridore
und SQLite-RTree plus exakter Prüfung. Das Oracle erhielt GPX-Segmentgrenzen
und verwendete sphärische Punkt-zu-Polyline-Distanz.

### Erster Engine-Lauf mit 5'000 Trails

Der `standard`-Lauf vom 28. August 2026 enthielt 5'000 synthetische Trails,
218'748 Vertices, 213'248 Segmente, 36 Querypunkte, Radien von
0,5/5/25/100 km und zwei Wiederholungen. Er verglich Meilisearch 1.36.0 und
1.44.0.

| Pfad | Beobachtung | Damalige Aussage |
| --- | --- | --- |
| Startpunkt `_geo` | schnell, aber bei `alpine_dense` und 0,5 km nur 3,17 % Recall sowie bei `long_outlier` für alle Radien 0 % | ungeeignet für routeweite Semantik; weiterhin gültig für den getrennten Startpunktfilter |
| GeoJSON Direct auf 1.44 | ca. 43,5 s Vollaufbau, 1,14 s für 50 Updates, gemessenes p95 höchstens ca. 69 ms; drei False Negatives ausserhalb der damaligen Grenztoleranz | der unveränderte Radius verletzte den damaligen Exact-Vertrag; kein Beweis für einen defekten GeoJSON-Index |
| H3 plus exakt | nach exakter Prüfung in diesem Korpus bis auf fünf ausgeschlossene Grenzfälle vollständig; ca. 3,8 s Aufbau, 1,17 s für 50 Updates, ca. 50 ms maximales gemessenes p95; ca. 97 MiB und 1,4 GiB Engine-Peak-RSS-Delta | plausibler Kandidatengenerator; Produktionsload, Fan-out und Caps offen |
| SQLite-RTree plus exakt | nach exakter Prüfung bis auf dieselben Grenzfälle vollständig; ca. 20,1 s Aufbau, 0,32 s für 50 Updates, 42,5 MiB; p95 ca. 183/187 ms bei dichten 25-/100-km-Queries | plausibler Kandidatengenerator; Suchintersection und Produktionsintegration offen |

Das war nur ein Zwischenstand. H3 wurde über eine bereits im RAM gehaltene Map
verifiziert, RTree lud eigene Geometrie-BLOBs, `maxTotalHits` entsprach der
Kataloggrösse und reale ACL, Freitext, Federation, Counts, globale Sortierung,
Pagination und Parallelität fehlten.

### Produktförmiger Preview mit 5'000 Trails

Danach trennte der Harness Engine- und Produktworkloads. Alle Pfade verwendeten
dieselben synthetischen Anwendungsdokumente und dieselbe 19-Feld-Projektion.
Eine exhaustive `ProductQuery` ausserhalb der Zeitmessung war Referenz für
synthetische ACL, deterministischen Text, Federationflags, Taxonomie- und
Zahlenfilter, exakte Geometrie, Count, stabile Sortierung und Pagination. H3
und GeoJSON-safe drückten kompatible Prädikate in Meilisearch, luden alle
verbleibenden SQLite-Kandidaten und prüften sie exakt. GeoJSON Direct behandelte
Meilis gefilterte, sortierte und paginierte Antwort als Endergebnis.

Der G1-Preview vom 28. August verwendete Meilisearch 1.53.1, 36 Produktfälle,
acht parallele Clients und je zwei Warm-Wiederholungen:

| Pfad | Korrektheit | Produkt-p50/p95/p99 und Durchsatz | Diagnose |
| --- | --- | --- | --- |
| H3 plus exakt | 36/36 Fälle und 72/72 Warm-Samples einschliesslich Counts/Reihenfolge/Seiten korrekt | 9,3/321,7/333,3 ms; 68,8 q/s | SQLite-Fetch/Decode dominierte mit ca. 255 ms p95; durchschnittlich 1'165 Zeilen beziehungsweise 3,48 MB geladen |
| GeoJSON Direct | 34/36 Fälle unter dem damaligen Exact-Vertrag; zwei Fälle mit je einem fehlenden Kandidaten/Count und vier abweichende Warm-Samples | 9,7/315,5/327,1 ms; 82,7 q/s | für damaligen Exact-Vertrag abgelehnt, aber offen für einen eigenen approximativen Produktvertrag |
| RTree plus exakt | 36/36 Fälle und alle Warm-Samples korrekt | 1,8/1'148,3/1'177,3 ms; 29,3 q/s | dichte parallele Kandidatenphase ca. 884,5 ms p95 |

Isolierte Vollaufbau- und Update/Add/Delete-Zeiten betrugen bei H3 3,20 s und
0,674/1,183/1,156 s, bei GeoJSON 39,00 s und 1,145/1,149/0,650 s und bei RTree
18,02 s und 0,149/0,040/0,064 s. Mutations-Readback war stichprobenartig;
Vollkorpuskorrektheit, parallele Queries, Recovery und Reconciliation während
Mutationen blieben ungemessen.

Der Preview machte H3/exakt zunächst zum führenden Kandidaten, aber nicht zur
Produktentscheidung. Anwendungszeilen waren synthetisch, Text modellierte
weder Meili-Relevanz noch Typotoleranz und Federation war ein Flag statt des
echten Visibility-Modells. Eine Query lud maximal 3'401 Zeilen beziehungsweise
10'177'009 Bytes und führte 2'384 exakte Geometrieprüfungen aus. Der Report
hatte zudem `gitDirty=true` und `fail_on_incorrect=false`.

### GeoJSON-Grenzdiagnose und Safe-Sweep

Fünf reproduzierte Near-Boundary-Paare lagen nur 0,038 bis 2,416 m innerhalb
ihrer 0,5-/5-/25-km-Radien. Meilisearch delegiert `_geoRadius` an Cellulite,
dessen Standardkreis polygonal angenähert wird. Dies erklärt Query-Kreis-Misses,
beweist aber keinen defekten gespeicherten GeoJSON-Index.

Der Harness prüfte explizite Auflösungen und konservative Varianten
`r / cos(pi/n) + padding` auf demselben Index mit rotierender Reihenfolge. Ein
Safe-Finalist verlangte die vollständige numerische Grenzzone, keine
Truncation, vollständige Geometrie-Hydration, exakte Prüfung und dieselbe
synthetische Produktpipeline. Auswahlpriorität waren Korrektheit und Caps, dann
End-to-End-p95; Differenzen bis fünf Prozent galten als Gleichstand und wurden
über kleinere Radiusaufweitung beziehungsweise geringeren Fan-out entschieden.

Ein lokaler Lauf vom 29. August wählte vorläufig Auflösung 16 mit
Secans-Erweiterung und 0,5 m Padding: 36/36 Erstfälle und 72/72 Warm-Samples
waren korrekt, p50/p95/p99 lagen bei 9,9/312,0/325,6 ms, der Durchsatz bei
80,7 q/s und der GeoJSON-Aufbau bei 44,0 s. Im Mittel lud der Lauf 855
Kandidaten aus SQLite; nach synthetischen ACL-, Text- und Fachfiltern blieben
358 exakte Geometrieprüfungen. Wegen nur 72 Full-Workload-Samples war dies
kein universeller Parameterentscheid.

Der spätere `calibration-5k-v2`-Lauf verwendete ein Ryzen-R1600-NAS, zwei
Kerne/vier Threads, 31,3 GiB nutzbaren RAM, vier Clients und 144 Warm-Samples
je Direct-Plan. Er bestand auf Meilisearch 1.53.1 `--fail-on-incorrect` und
`--fail-on-ux`, hatte aber kein Performance-Target und weiterhin
`gitDirty=true`. RTree und Point-Baseline wurden nicht gemessen. Die Ergebnisse
qualifizieren weder 8 GiB RAM noch unabhängig eine DS923+. Diese Grenzen müssen
jede zitierte Index-, H3-, Safe-GeoJSON- oder Latenzzahl begleiten.

| Pfad | Vertrag und Ergebnis | p50/p95/p99 | Durchsatz | Fan-out/Transfer | damalige Kalibrierung |
| --- | --- | ---: | ---: | --- | --- |
| GeoJSON Direct r100 | approximativer UX-Vertrag bestanden; 12'852/12'853 Produktmatches, ein Boundary-Miss 3,8 cm innerhalb, keine materiellen FN/FP und keine ACL-/Text-/Filterabweichung; vergleichbare Seiten stabil | 22/94/119 ms | 120,5 q/s | kein SQLite-Fetch; ca. 3,0 KiB finale Response | stärkste Hypothese für den Punkt-Radius |
| GeoJSON-safe r8 plus exakt | 36/36 Produktfälle und 144/144 Samples gegen die synthetische exakte Referenz korrekt | 25/636/726 ms | 28,7 q/s | durchschnittlich 363 Kandidaten beziehungsweise 1,17 MB | führender vollständig exakter Fallback dieses Laufs |
| H3 plus exakt | 36/36 Produktfälle und 144/144 Samples gegen die synthetische exakte Referenz korrekt | 39/700/812 ms | 21,0 q/s | durchschnittlich 542 Kandidaten beziehungsweise 1,74 MB | nur noch Skalierungs-/Indexaufbau-Fallback; Heatmaps ungemessen |

Im vollständigen r100-Screen blieben fünf Boundary-Misses bis höchstens
2,30 m innerhalb sichtbar; alle klar inneren Treffer, die zehn nächsten klar
inneren Kandidaten und alle 19 vergleichbaren Produktseiten waren vollständig.
Sechs von 36 absichtlich eingestreuten Linien-Proximity-Sortierungen waren
sichtbar nicht unterstützt und wurden nicht als vergleichbare Seiten gewertet.
r125 war beim Auswahl-p95 langsamer und hatte im Produkt-Audit zwei
Boundary-Misses. r250 zeigte in diesem Korpus keine Produkt- oder
Screenabweichung zur exakten Referenz, benötigte aber 113,7/148,4 ms p95/p99
statt 93,9/118,6 ms bei r100; dies war ein Sicherheitsvergleich, kein
allgemeiner Exaktheitsbeweis.

Der GeoJSON-Vollaufbau dauerte 53,57 s gegenüber 10,35 s für H3. H3 belegte
99,8 MiB logisch beziehungsweise 116,3 MiB im Dateisystem, GeoJSON 35,6
beziehungsweise 38,5 MiB. Der Engine-Peak-RSS-Delta lag bei beiden ungefähr
zwischen 1,1 und 1,2 GiB. Mutationswerte bezogen sich nur auf je 50 isolierte
Änderungen und waren keine Aussage zu gleichzeitiger Last oder Skalierung.

### Nicht normative Harness- und Regressionsmatrix

Folgende Messdimensionen erhalten die Architekturentscheidung reproduzierbar
und grenzen künftige Engine-, Korpus- und Kapazitätsprofile ein. Ihre konkreten
Budgets werden erst durch den jeweils angenommenen Vertrag normativ:

- Vollindex sowie inkrementelles Add/Update/Delete unter gleichzeitiger
  Suchlast, einschliesslich Crash, Rebuild und Reconciliation;
- Peak-RAM, CPU, I/O und Plattenwachstum getrennt nach Engine, Projektion und
  gegebenenfalls Geometriehydration;
- kalte und warme p50/p95/p99-Latenz je Radius, Parallelität, Shardzahl und
  Katalogprofil sowie lange Steady-State-Läufe;
- kombinierte Geo-, Text-, ACL-, Federation-, Facetten-, Count-, Sortier- und
  Deep-Pagination-Queries unter Parallelität;
- bei Direct interne Snapshot-/Seiten-/Countkonsistenz sowie Abweichung vom
  unabhängigen Rohgeometrie-Oracle; bei zweistufigen Pfaden Kandidatenzahl,
  DB-Fetch/-Bytes, Decode und vollständige exakte Prüfung vor Count/Paging;
- Precision und Recall einschliesslich klarer Innen-/Aussenfälle, neutralem
  Grenzband, Antimeridian, Polnähe, H3-Pentagonen, getrennten Segmenten,
  Capacity-Caps, dichten Korridoren und adversarialen langen Segmenten;
- Responsegrösse, Warm-Stabilität, Truncation, False-Empty, Top-10,
  Nearest-Coverage, globale Sortierung und vollständige Traversierung;
- reproduzierbare Harness-, Dataset-, Executable- und Image-Digests,
  Dirty-State, Hardwareprofil, aktive Targets sowie archivierte Rohreports.

Ein bestandener isolierter Mutationslauf belegt keine Mutationskorrektheit
unter Suchlast; ein synthetisches ACL-/Federationflag belegt nicht den realen
Produktpfad; ein Hardwarelauf ohne aktives Target bleibt Diagnose. Diese
Grenzen werden in jedem Report zusammen mit seinen Zahlen ausgewiesen.

### Angenommene Evidenz mit 10'000 Trails

Die ADR dokumentiert den entscheidenden Lauf: Seed `20260830`, 60 Querypunkte,
vier Radien, vier Clients, 240 vollständige räumliche Fälle und 240
Warm-Produktsamples.

Das G1-Korpus kombiniert reale Geometrieverteilungen für p50-, p95-, p99- und
Maximalrouten mit deterministischen synthetischen dichten Alpenkorridoren,
weltweit verteilten Routen, getrennten Segmenten und adversarialen langen
Ausreissern. Das unabhängige Oracle misst die kanonische Rohgeometrie; der
Direct-Index verwendet die angenommene S50-Suchprojektion mit höchstens 5 km
langen Segmenten. Dieselben Klassen bleiben normale Contract- und
Regressionfixtures.

| Metrik | Ergebnis |
| --- | ---: |
| Shards / Trails | 10 / 10'000 |
| räumlicher Raw Recall | 99,97 % |
| Clear Recall / materielle Precision | 100 % / 100 % |
| Top-10 / Nearest / Non-Empty | 100 % / 100 % / 100 % |
| materielle False Negatives / Positives | 0 / 0 |
| Produkt-p50 / p95 / p99 | 65 / 183 / 338 ms |
| Durchsatz | 49 q/s |
| exakter Count / innerhalb 1 % / abgeleitete Seitengrenze | 57/60 / 60/60 / 60/60 |
| vollständiger Indexaufbau | 10,2 s |
| Meilisearch-Datenbank | 52,7 MiB |
| Engine-Peak-RSS-Delta | ca. 1,2 GiB |

Die im historischen Harness `totalPages` genannte dritte Metrik war eine intern
aus Count und Seitengröße abgeleitete Seitengrenze. Sie ist kein Feld der
öffentlichen V1-Response.

Identitätsanker:

- Harness: `sha256:b65bfd9c31f15edb2d697743551237951ab6642a3a1da5251a2d86d1c82c4867`;
- Dataset: `sha256:9f94f324539c0c231f83b1c21f7fd3a2ec59eacc5c9ef375230329f0e62bbdc0`;
- Executable: `sha256:acc700fe190faf67371b6224b22bd241c3349257a7c2eb2dc049e1b0637fe535`;
- Image: `getmeili/meilisearch@sha256:8d6643d86d71fad6ad3cba92cde7ccfce9e4d6c384bda67598eb553571c32431`.

Der Lauf hatte kein Performance-Target, führte im Capacity-Search-Modus keine
Mutationen aus und meldete einen Dirty-Source-Stand. Digests identifizieren
Inputs, archivieren deren Bytes aber nicht. Das sind Evidenzgrenzen und keine
Behauptung, Produktpipeline, Recovery oder Deployment seien über den
dokumentierten Workload hinaus bewiesen.

Ein separater synthetischer 50k-/50-Shard-Lauf zeigte die Skalierungsrichtung:
99,95 % Raw Recall, 100 % Clear Recall und materielle Precision, 8,5 q/s,
827 ms Produkt-p95, 85,6 s Build inklusive Setup und ca. 262 MiB Datenbank. Er
lief mit 32 GiB RAM, 120 Warm-Samples, ohne Performance-Target und aus einem
Dirty-Stand. Er ist optionale Kapazitätsevidenz, kein Releasegate und keine
DS923+-/8-GiB-Qualifikation.

## Engine-Diagnosen

Zwei reproduzierbare Diagnosen bleiben für Meilisearch 1.53.1 sichtbar:

- `geojson_cell_coverage`: Die Kombination aus sphärischer Zellzuordnung und
  planarem Zellpolygon kann gespeicherte Zellen auch in gewöhnlicher Breite
  auslassen.
- `geojson_hierarchical_split`: In einer dicht geteilten Zelle liefert ein
  GeoJSON-Polygon 4'980 von 5'000 Kontrollen, der native Bounding-Box-Pfad
  5'000 von 5'000.

Der angenommene Vertrag behandelt diese Engine-Limitierungen über seine
aggregierten UX-Gates und deutet sie nicht als deterministische 50-m-Grenze.
Jedes Engineupgrade führt die Diagnosen erneut aus. Eine bekannte beobachtete
Limitierung ist nicht selbst gating; kann die Diagnose nicht ausgeführt werden,
ist `diagnostic_error` gating.

## Bewertete Alternativen

### H3 plus exakte Prüfung

H3 war im Preview nur mit vollständiger Kandidatenhydration und exakter Prüfung
vollständig. Ein Parallelbetrieb würde Write-, Projektions-, Speicher- und
Betriebslogik duplizieren. H3 bleibt der erste geordnete Ersatz, falls Direct
seinen Vertrag später verfehlt; der Ersatz ist ein vollständiger Vertical Slice
und nie ein stiller Fallback pro Request.

### SQLite-RTree

RTree liefert Bounding-Box-Kandidaten, nicht das finale Wanderer-Ergebnis. ACL,
Text, exakte Segmentdistanz, Counts, globale Sortierung und Pagination müssten
weiter orchestriert werden. Zusätzliche Persistenz und dichte Radiuslast
rechtfertigten dies für den angenommenen Vertrag nicht.

### GeoJSON Safe/Exact

Konservative Radiusaufweitung plus exakter Prüfung bleibt ein vermessener
möglicher Exact-Modus. Sie führt dieselbe Vollmengenhydration und Orchestrierung
ein, die Direct vermeidet, und beseitigt die vorgelagerten Engine-Diagnosen
nicht automatisch.

### Korridor, Startpunkt und eigener Enginebuild

Ein gepufferter Polygonkorridor erhöht Geometrie-/Indexkomplexität und braucht
einen eigenen Vertrag. `_geo` beantwortet nur die getrennte Startpunktfrage.
Ein eigener Meilisearch-/Cellulite-/H3-Build wurde verworfen, weil er Security-,
Upgrade-, Image- und Supportkosten zu Wanderer verlagert.

## Surface-Proof-of-Concept

Der verworfene `surfaceTypes`-POC (Commits `cd82ad01f` bis `e01735d48`) rief
Valhalla `trace_attributes` auf, speicherte Surface-/Wegtypwechsel und zeigte
sie im Höhenprofil; MTB-Skalen stammten aus einer Browser-Overpass-Abfrage. UI
und Distanzrollups bleiben nützliche Evidenz, die Implementierung wird nicht
übernommen.

Der entscheidende Fehler war stille Matcher-Unterdeckung: Valhallas
Standard-`pedestrian`-Costing schloss anspruchsvollere `sac_scale`-Wege aus.
OSM-Weg `39669166` matchte nur ca. 16 m statt ca. 1,09 km, bis
`costing_options.pedestrian.max_hiking_difficulty: 6` gesetzt wurde. Weitere
Defizite waren clientseitige Provideraufrufe, ungecachte breite
Overpass-Bounding-Box-Queries, abgeflachte GPX-Segmente, Punkt- statt
Liniendistanz, fehlende Dataset-/Versions-/Coverage-Semantik und das
mehrdeutige `unknown = 0` bei MTB S0.

Diese Evidenz begründet die persistierte, versionierte und coverage-bewusste
Surface-Pipeline im CONTEXT-Vertrag. Sie legt weder finale Taxonomie, Matcher,
Schwellen, Refreshintervall noch Produktionskapazität fest.

## Kalibrierhypothesen für die Cycle-Referenzdauer

[ADR 0003](/develop/specs/trail-search/decisions/0003-walk-hike-cycle-personal-evaluation/)
legt Cycle als Bestandteil der ersten produktiven Bewertung, die getrennten
Disziplinen und den Cycle-Kombinator fest. Die folgenden Werte sind dagegen nur
Seeds für ein ausdrücklich nicht auslieferbares Kalibrierprofil,
Kalibrierfixtures und den Aufbau eines realen Korpus:

| Disziplin | Distanzrate | Steigrate |
| --- | ---: | ---: |
| Touring | 15 km/h | 400 m/h |
| Road | 22 km/h | 600 m/h |
| Gravel | 16 km/h | 500 m/h |
| MTB | 12 km/h | 500 m/h |

Der MTB-Seed lehnt sich an die veröffentlichte DAV-Planungsreferenz von 12 km/h
und 500 bis 600 Höhenmetern pro Stunde an. Touring, Road und Gravel sind
Produkt- und Korpushypothesen, keine extern belegten universellen
Leistungswerte. Keiner dieser Seeds ist ein Produktionsdefault oder eine
Aussage über eine konkrete Person.

Die Qualifikation stratifiziert mindestens nach Disziplin, Distanz,
Gesamtaufstieg, flacher Route, Rundweg/Punkt-zu-Punkt, Datenquelle,
Unterstützungsclaim und tatsächlicher Unterstützung. Provider-Planzeiten und
beobachtete Bewegungszeiten bleiben getrennte Vergleichsreihen. Jeder
freigegebene Parametersatz erhält eine unveränderliche Modellversion und einen
Fingerprint; eine Parameteränderung erzeugt eine neue Version. Elektrisch
unterstützte Beobachtungen sind vom Fit der neutralen menschlichen Distanz- und
Steigraten ausgeschlossen und dürfen nur einen getrennten Unterstützungseffekt
kalibrieren.

## Herstellerreferenzen für Anstiege

Garmin ClimbPro liefert für Radfahren eine reproduzierbare Kalibrierreferenz:
mindestens 500 m Länge, mindestens 3 % Durchschnittssteigung und mindestens
1'500 Punkte mit
`Score = Länge in Metern * Steigung in Prozentpunkten`. Flachstücke und
Gegenabfahrten dürfen enthalten sein, solange der Gesamtdurchschnitt genügt.
Die veröffentlichten Schwellen für Cat 4/3/2/1/HC liegen über
8'000/16'000/32'000/64'000/80'000 Punkten.

Hammerhead nennt mindestens 400 m und 3 %, veröffentlicht aber keine
vollständigen numerischen Grenzen für „mittel“ und „gross“. Diese Zahlen sind
Mess- und Kalibrierreferenzen, keine übernommene Wanderer-Skala und kein
normativer Default. Wanderer kann eine gemeinsame Basissegmentierung nutzen,
hält Anerkennung, Score und Klasse aber über Aktivitätsfamilie, gegebenenfalls
Disziplin, Klassifikationsprofil/-version und Scoremodell eindeutig getrennt.

## Noch zu kalibrierende Werte

Folgende Werte benötigen repräsentative reale Korpora, Providerprofile, ADRs
und Lasttests. Der angenommene `route_radius_ux_v1` wird dadurch nicht wieder
geöffnet:

- Walk-/Hike-Koeffizienten, Cycle-Geschwindigkeiten und -Steigraten je
  Disziplin sowie Profildefaults und Grenzen der persönlichen Konditionsklassen;
- Climb-Resampling/-Glättung, Prominenz, Merge-Regeln, Mindestcoverage und
  aktivitätsspezifische Erkennungs-/Klassifikationsschwellen;
- Surface-Taxonomie, Mindest-Match- und Dimensionscoverage, Anteil-Buckets,
  zulässiger Matcher-/Tag-Snapshot-Abstand und Confidence-Regeln;
- `refresh_due_at`-/`usable_until`-Policies je exakter und opaker Quelle;
- Surface-Schedule, Claim-Batch, Workerparallelität sowie Zeit-, Punkt-, Meter-,
  Provider-, Retry- und Dead-Job-Budgets gegen grössten Katalog und Freshnessziel;
- Notwendigkeit einer Diff-Invalidierung, falls der begrenzte TTL-Sweep das
  Freshnessziel nicht erfüllt;
- je Geocoding-Adapter Datasetumfang, RAM/Storage, Updatepfad, maximales Alter,
  Kategorie-Fixtures und Readiness-/Verfügbarkeitsziele für Managed/Self-hosted;
- Transit-Gehschwellen, Endpunktregeln, Modi, statischer Servicehorizont,
  Realtime-Kompatibilität und Journey-Providerquoten;
- Saison-/Forecast-Auflösung, Cachegültigkeit, Unsicherheit, Konvergenz,
  Kandidaten- und Latenzbudgets;
- optionale Kapazitätsdiagnosen für grössere Kataloge und beschränkte NAS;
- jedes künftige Radiusprofil, Exact-/Proximity-Produkt, echter Flächenvertrag
  oder Engineupgrade. Sie benötigen ein neues unveränderliches Profil und einen
  vollständigen Contractlauf; S50/r100, 100-km-Cap und bestehende
  Startpunktsemantik bleiben im angenommenen V1-Vertrag stabil.

## Quellen und externe Referenzen

Die Links belegen einzelne Ausgangsaussagen und Kalibrierreferenzen; sie sind
keine eigenständigen Wanderer-Produktverträge.

### Suche, GeoJSON und Hardware

- [Meilisearch: Reihenfolge durchsuchbarer Attribute](https://www.meilisearch.com/docs/capabilities/full_text_search/relevancy/attribute_ranking_order)
- [Meilisearch: GeoJSON](https://www.meilisearch.com/docs/capabilities/geo_search/how_to/use_geojson_format)
- [Meilisearch 1.53.1: `_geoRadius`-Auflösung 3–1'000](https://github.com/meilisearch/meilisearch/blob/v1.53.1/crates/milli/src/search/facet/filter/index_filter.rs#L481-L527)
- [Cellulite 0.3.2: approximierter Kreis und Default-Auflösung 125](https://github.com/meilisearch/cellulite/blob/v0.3.2/src/reader.rs#L139-L176)
- [Meilisearch: offener GeoJSON-Korrektheitsfall #6575](https://github.com/meilisearch/meilisearch/issues/6575)
- [Synology: DS923+ Datenblatt](https://global.download.synology.com/download/Document/Hardware/DataSheet/DiskStation/23-year/DS923%2B/enu/Synology_DS923%2B_Data_Sheet_enu.pdf)
- [AMD: Ryzen Embedded R1000 Product Brief](https://www.amd.com/content/dam/amd/en/documents/products/embedded/ryzen/R1000-product-brief.pdf)

### Geocoding und Places

- [OpenStreetMap Foundation: Nominatim Usage Policy](https://operations.osmfoundation.org/policies/nominatim/)
- [Nominatim Search API: Layer und Feature-Typen](https://nominatim.org/release-docs/latest/api/Search/)
- [Photon: API und kategorisierte Suche](https://github.com/komoot/photon/blob/master/docs/api-v1.md)
- [Photon: Kategorien und `include`/`exclude`](https://github.com/komoot/photon/blob/master/docs/categories.md)

### Routing, Surface, Anstiege und Transit

- [Valhalla: Map Matching / Trace Attributes](https://valhalla.github.io/valhalla/api/map-matching/api-reference/)
- [Valhalla: Status, Tileset-Zeit und Dataset-ID](https://valhalla.github.io/valhalla/api/status/)
- [OpenStreetMap: PBF-Replikationsmetadaten](https://wiki.openstreetmap.org/wiki/PBF#Definition_of_the_OSMHeader_fileblock)
- [OpenStreetMap: Replikationsdiffs und State-Semantik](https://wiki.openstreetmap.org/wiki/Planet.osm/diffs)
- [OpenStreetMap: Surface, Smoothness und Tracktype](https://wiki.openstreetmap.org/wiki/Surface)
- [Verworfener Surface-POC, letzter Commit](https://github.com/open-wanderer/wanderer/commit/e01735d488b8d22b9d81aa8ca30c4020a9569fb6)
- [Spätere Surface-Ursachenanalyse](https://github.com/open-wanderer/wanderer/blob/feature/app/.planning/phases/37-way-types-surfaces-breakdown-mobile-first/37-RESEARCH-SOURCE.md)
- [Garmin: ClimbPro-Klassifikation](https://support.garmin.com/en-US/?faq=KKRLD2Fo6MAlCXOzUZb1e9)
- [Hammerhead: Karoo Climber](https://support.hammerhead.io/hc/en-us/articles/25602163627163-Karoo-OS-Climber)
- [DAV: Fahrzeitberechnung für Mountainbike-Touren](https://www.alpenverein.de/artikel/wie-berechne-ich-meine-fahrzeit_be657eb5-6195-4b01-ad27-606fa0934647)
- [Schweizer Wanderwege: Wanderzeitberechnung](https://www.schweizer-wanderwege.ch/de/wissen/signalisation/wanderzeit)
- [GTFS](https://gtfs.org/documentation/overview/)
