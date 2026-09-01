---
title: SRCH0 — Bestandsvertrag und Testkorpus
description: Datierter, ausführbarer Vertrag des beobachteten First-Party-Suchverhaltens vor der V1-Migration.
editUrl: false
sidebar:
  order: 1
  badge: Review
spec:
  id: SRCH0
  kind: work-item
  status: reviewable
  deliveryStatus: candidate
  capability: FOUNDATION
  productSlice: search-foundation
  exposure: internal
  implementationDependsOn: []
  releaseGates: []
  normativeSources: [TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-09-01'
---

## Zweck

SRCH0 friert das beobachtete Suchverhalten von Wanderer an einer datierten
Ausgangsrevision als sprachneutralen, ausführbaren Testkorpus ein. Der Korpus
macht sichtbar, welche Eingaben die vorhandenen First-Party-Oberflächen
erzeugen, welche Dokumente der heutige Projektor schreibt und welche Treffer
die vorhandenen Suchpfade zurückgeben.

Der Vertrag ist absichtlich beobachtend. Ein festgehaltener Zustand ist nicht
automatisch fachlich erwünscht. Bekannte Lücken werden mit stabiler Case-ID und
einem `successor_ref` markiert, aber in SRCH0 weder korrigiert noch mit einer
Zielsemantik überschrieben.

Nachgelagerte Verträge, Compiler, Sicherheitsarbeiten und Indexprojektionen
können dieselben Fälle konsumieren. Ihre erwarteten Änderungen werden als
eigene Overlays neben dem unveränderten SRCH0-Basisfall beschrieben.

SRCH0 verändert keine produktive Suche. Insbesondere definiert es keinen
Reindex, keinen Cutover, keine Readinesslogik und keine Runtime-Control-Plane.

## Metadaten und Bestandsanker

| Feld | Wert |
| --- | --- |
| Work Item | `SRCH0` |
| Ergebnis | kanonischer Fixturekorpus und ausführbarer Bestandsbericht |
| Ausgangsrevision | Git-Commit `e6db729cb73a490a939420e256b176acf9fd8fc3` |
| Beobachtungsdatum | 30. August 2026 |
| Dokumentrevision | 1. September 2026 |
| Engine-Ausgangsprofile | Meilisearch 1.11.3 und 1.36.0 mit den Settings der Ausgangsrevision |
| Exposure | intern; Tests und Dokumentation, keine Produktaktivierung |
| Implementierungsabhängigkeiten | keine |
| Nachfolger | IDX0, SRCH-V1, SRCH-COMP, SRCH2, SRCH4a, SEC-VIS-0 und IDX1 |

Der Bestandsanker besteht aus Commit, Engineprofil, Fixture-Schema und
Datasetrevision. Ein Ergebnis ohne diese vier Bindungen ist keine
SRCH0-Evidenz.

### Bedeutung normativer Sprache

`MUSS`, `DARF NICHT` und vergleichbare Formulierungen regeln in diesem
Dokument ausschliesslich Form, Reproduzierbarkeit und Auswertung des Korpus.
Sie verpflichten nicht die spätere Produktimplementierung, beobachtete Lücken
beizubehalten.

`observed` bezeichnet das am Ausgangscommit tatsächlich reproduzierte
Ergebnis. Ein Nachfolgetask darf ein anderes Ergebnis fordern, aber den
SRCH0-Basiswert nicht umschreiben.

## Scope

SRCH0 umfasst:

- die produktiv verwendeten Suchoberflächen Trail-Liste, Karte und Profil;
- URL-, LocalStorage-, Snapshot-, History- und Paginationzustände dieser
  Oberflächen;
- alle im First-Party-Code erzeugten Trailfilter und sichtbaren Sortierungen;
- die bekannten Einzel- und Multi-Search-Aufträge;
- die PocketBase-zu-Meilisearch-Projektionen für Trails, Listen und Actors;
- die am Prozessstart beobachtete Settings-, Lösch- und Neuaufbaufolge dieser
  drei Indizes samt währenddessen erreichbarer Suchpfade;
- Tenant-Token-Kontext und serverseitige Kategoriepräferenzen als beobachtete
  Eingaben der Treffermenge;
- Trefferreihenfolge, Gleichstandsgruppen, `total`, Seite und Seitengrösse;
- den authentifizierten Actor-Suchpfad;
- die Meilisearch-gestützte Duplikatprüfung beim Trail-Upload;
- die bekannten Profil-, Empfehlungs-, Bounding-Box- und Clusterabfragen;
- Create-, Update-, Delete- und Relationsmutationen, soweit sie heute
  Suchdokumente erzeugen oder verändern; und
- einen gemeinsamen JSON-Korpus für Go-, Vitest-, Engine-, API- und
  Browsertests.

Der rohe Proxy `POST /api/v1/search/{index}` kann technisch mehr ausdrücken
als die Anwendung selbst verwendet. SRCH0 inventarisiert nur nachweisbare
First-Party-Aufträge. Zufällig von Meilisearch akzeptierte freie Optionen sind
kein Bestandsvertrag.

## Nichtziele

- Keine Korrektur eines beobachteten Verhaltens.
- Keine neue öffentliche Such-API oder V1-Requestgrammatik.
- Keine V1-Normalisierung und kein kanonischer V1-URL-Codec.
- Kein neues Indexschema und keine neue fachliche Projektion.
- Kein Reindex-, Migrations-, Backfill-, Swap- oder Rollbackverfahren.
- Keine Startup-, Liveness-, Readiness- oder Write-Admission-Regel.
- Keine Taskbarriere, Epochenattestierung oder Engine-Tasküberwachung.
- Keine Credentialrotation oder Festlegung produktiver Netzwerkgrenzen.
- Keine neuen Filter, Sortierungen, Counts, Histogramme oder Geo-Funktionen.
- Keine Paritätszusage für beliebige freie Meilisearch-Ausdrücke.
- Keine Aufwertung des sichtbaren `TrailFilterPreview` zu einer produktiven
  Capability.

Die frühere Langfassung mit Korrektur-, Migrations-, Recovery- und
Runtimeüberlegungen bleibt als
[nichtnormatives Designarchiv](/develop/specs/trail-search/evidence/srch0-design-archive-2026-09-01/)
erhalten.

## Evidenzmodell

Die Evidenz wird in folgender Reihenfolge bewertet:

1. ausführbarer Code am gebundenen Ausgangscommit;
2. reproduzierbare Engineantwort am gebundenen Engineprofil;
3. eingechecktes Dataset und schema-validierter Fixturefall;
4. menschenlesbare, aus den Fixtures erzeugte Bestandsmatrix; und
5. erläuternde historische Dokumentation.

Widersprechen sich Prosa und reproduzierbarer Fall, ist der Fall massgeblich
und die Prosa wird korrigiert. Widersprechen sich Code und ein Golden, darf das
Golden nicht automatisch aus der neuen Ausgabe aktualisiert werden. Zuerst
muss geklärt werden, ob Ausgangscommit, Profil, Dataset oder Beobachtung falsch
gebunden sind.

Ein sicherheitswidriges Ergebnis wird als beobachteter `known_gap` erfasst.
Die Erfassung macht es weder zulässig noch zu einer zu erhaltenden
Produkteigenschaft.

## Beobachteter Datenfluss

```text
URL / LocalStorage / SvelteKit-Snapshot / UI
  -> oberflächenspezifische Defaults und sanitizeTrailFilter
  -> TrailFilter
  -> buildFilterText + q + sort + page
  -> SvelteKit-Suchproxy
  -> Kategoriepräferenzen + Meilisearch-Tenant-Token
  -> Index trails
  -> TrailSearchResult -> Trail
  -> Liste, Tabelle, Kartenliste oder Cluster

PocketBase-Record + expandierte Relationen
  -> documentFromTrailRecord / weiterer Indexadapter
  -> Meilisearch-Dokument
```

| Grenze | Codeanker am Ausgangscommit | Zu beobachtender Anteil |
| --- | --- | --- |
| Listen-Defaults und URL | `web/src/routes/trails/+page.ts`, `+page.svelte` | Defaultfilter, URL-Werte, Storagepräzedenz, Seite |
| Karten-Defaults | `web/src/routes/map/+page.ts`, `+page.svelte` | Sortierung, Bounds und Kartenkoordinaten |
| Sanitizing | `web/src/lib/util/trail_filter_util.ts` | Typkonversion, Clamp, Fallback, Min/Max |
| Legacycompiler | `web/src/lib/stores/trail_store.ts` | Filtertext, Rundung, Gruppen und Engineauftrag |
| Suchproxy | `web/src/routes/api/v1/search/[index]/+server.ts` und `multi/+server.ts` | bekannte Requestformen |
| Präferenzfilter | `web/src/lib/server/category_preference_filter.ts` | actorbezogene Taxonomieausblendung |
| ACL-Kontext | `db/routes/search_token.go` | anonymer und authentifizierter Tenant-Scope |
| Trailprojektion | `db/util/meilisearch.go` | Dokumentfelder und Relationsauflösung |
| Engineprofil | `db/main.go` | Attribute und Rankingregeln |
| Startup-Indexinitialisierung | `db/main.go`, `onBeforeServeHandler`, `initData`, `initMeilisearchDocuments` | Auftragsfolge, Task-Waits, Fehlerbehandlung und gleichzeitig erreichbare Consumer |
| Trefferkonvertierung | `web/src/lib/stores/trail_store.ts` | gelesene Felder und UI-Abbildung |
| Actor-Suche | `web/src/routes/api/v1/search/actor/+server.ts` | Auth, Handle-Zweig, `includeSelf`, Limit |
| Upload-Duplikatprüfung | `web/src/routes/api/v1/trail/upload/+server.ts` | leerer Suchauftrag, Scope, Limit, gelesene Felder |

### Startup-Indexinitialisierung

Am Ausgangscommit ruft jeder Serve-Start nach der Routenregistrierung
`initData` auf. Die Funktion submitttet die Settings der logischen Indizes
`trails`, `lists` und `actors` und startet danach eine Goroutine. Diese ruft
für jeden der drei live verwendeten Indizes zuerst `DeleteAllDocuments` und
danach paginierte Add-Aufträge aus PocketBase auf.

Nur eine gegebenenfalls nötige Indexerstellung wird terminal abgewartet.
Settings-, Delete- und Add-Aufträge werden nicht terminal bestätigt; der
Rückgabefehler von `initMeilisearchDocuments` wird in der Goroutine verworfen.
`se.Next()` öffnet den Server unabhängig davon, und die beobachteten
First-Party-Suchpfade besitzen keinen gemeinsamen Readiness-Guard. Damit kann
ein Request bei jedem normalen Neustart einen leeren oder teilweise neu
befüllten Index erreichen.

Der Basisfall vergoldet ausschliesslich diese Auftrags- und
Erreichbarkeitsfolge. Ihre Ablösung und die Produktion des normativen
Readinesszustands gehören `IDX0`; SRCH0 definiert weder Ensure-Algorithmus noch
Startupgate. Die zugehörigen Fixtures verwenden `family: mutation` und einen
eigenen Startup-Consumer; sie führen keine siebte Fixturefamilie ein.

## Beobachtete Legacy-Semantik

### Zustand und Defaults

Die Trail-Liste startet ohne gespeicherten Zustand mit `created ASC`; die
Karte verwendet `created DESC`. Ohne gespeicherte Seitengrösse startet die
Liste mit 25 Treffern.

Ein vorhandenes `paginationItems` wird beim Mount nach Darstellungsmodus
gebucketet: Karten verwenden `12/24/48/96`, andere Ansichten
`10/25/50/100`. Ein gespeicherter Wert 25 wird in Kartenansicht zu 24.

Die alte Listen-URL liest nur `author`, den jeweils ersten `category`- und
`subcategory`-Wert sowie `page`. Sobald einer dieser Filter in der URL
vorhanden ist, wird das gespeicherte `trailListFilter` vollständig verworfen.
Die separaten Storagewerte `sort` und `sort_order` überschreiben danach den
ermittelten Filter ohne erneutes Sanitizing. Die Präzedenz ist deshalb
feldbezogen und nicht pauschal „URL vor Storage“.

Ein `q`- oder `sort`-Queryparameter ohne V1-Kennung besitzt in dieser URL keine
Suchwirkung. Defektes JSON, doppelte Querykeys und getrennte Page-Keys werden
als rohe Eingaben im Korpus erhalten.

### Filter und Suche

| Eingabe | Beobachtete Auswertung |
| --- | --- |
| Freitext `q` | unverändert an Meilisearch |
| Suchattribute | `author_name`, `name`, `description`, `location`, `tags` in dieser Reihenfolge |
| Distanz/Aufstieg/Abstieg Minimum | immer `floor`, Grenze inklusive |
| Distanz/Aufstieg/Abstieg Maximum | unter dynamischem Oberlimit `ceil`, Grenze inklusive |
| Difficulty | `0`, `1`, `2` innerhalb der Gruppe OR; auch Default `[0,1,2]` erzeugt einen Filter |
| Autor | exakte Actor-ID |
| Public/Private/Shared | Browserbranches plus autoritativer Tenant-Token-Scope |
| Von mir geliked | `true` filtert auf Actor-ID; `false` ist wirkungslos |
| Datum | JavaScript-Datum zu Unixsekunden; Ende bezeichnet den Beginn des Tages |
| Kategorie/Subkategorie | Kategoriebranches OR; Child oder `__no_subcategory__:<category>` verengt den Parent |
| Tags | ausgewählte Namen OR; gegenüber anderen Gruppen AND |
| Abschlussstatus | `true` und `false` filtern exakt; ausgelassen ist neutral |
| Startpunktnähe | nur Liste und nur bei truthy Latitude und Longitude |
| Kartenbounds | Bounding-Box-Prädikat für Liste und Cluster |

Bei aktiver Startpunktnähe erzeugt der beobachtete Compiler dieselbe
`_geoRadius`-Klausel zweimal. Der Korpus speichert sowohl den bytegenauen
Filter als auch Treffer, `total`, Reihenfolge und Seite.

Die Karte kombiniert Liste, Cluster und ID-Details aus mehreren Requests ohne
gemeinsamen Snapshot. SRCH0 prüft diese Ergebnismengen auf einem unveränderlichen
Dataset getrennt und leitet daraus keine Snapshotgarantie ab.

### Sortierung, Ranking und Pagination

Beobachtete sichtbare Sortierkeys sind:

```text
name, distance, duration, difficulty, elevation_gain, elevation_loss,
like_count, created, date
```

Jeder Key wird auf- und absteigend erfasst. Bei nichtleerem `q` steht die
explizite Sortierung in den heutigen Rankingregeln nach Wort-, Tippfehler-,
Nähe- und Attributrelevanz. Der Korpus behauptet deshalb nicht, der sichtbare
Sortkey sei immer der Primärschlüssel der Gesamtreihenfolge.

Pagination verwendet `page` und `hitsPerPage`. Wo der Bestand keinen
eindeutigen Tie-Breaker besitzt, speichert das Fixture eine geordnete Liste von
Gleichstandsgruppen statt eine erfundene Totalordnung.

### Sichtbarkeitskontext

Der anonyme und der authentifizierte Tenant-Token begrenzen die beobachtete
Treffermenge zusätzlich zum Browserfilter. „Geteilt“ bezeichnet in der
Suchmatrix einen Actor-Share, der in `shares` und im Tenant-Scope vorkommt.

Ein Link-Share-Token erlaubt beobachtet PocketBase-Detailzugriff auf genau
einen Trail, erweitert aber nicht das Meilisearch-Suchuniversum. Lokale,
Remote-, private, public-, owned-, shared-, revoked- und deleted-Kontexte
erhalten getrennte Fälle.

Der Korpus speichert beobachtete Hits, `total` und DTO gemeinsam. Eine
Sicherheitslücke darf nicht durch ein Golden verborgen werden; sie wird als
`known_gap` an SEC-VIS-0 verwiesen.

### Projektion und Trefferabbildung

Das beobachtete Traildokument umfasst unter anderem Identität, Autor,
Beschreibung, Ort, Distanz, Dauer, Auf-/Abstieg, Difficulty, Taxonomie, Tags,
Datum, Sichtbarkeit, Abschlussstatus, Assets, Bounds, `_geo`, Shares und
Likes. Das eingecheckte Baselineprofil ist für die vollständige Feldliste und
Settingsreihenfolge massgeblich.

Create und Vollprojektion sowie Core-, Share- und Like-Updates verwenden
heute nicht durchgehend dieselbe Payloadform. Mutationfixtures speichern
deshalb Vorzustand, konkrete Mutation, beobachtete Engineaufträge und den
materialisierten Endzustand getrennt.

Eine fehlende, leere oder unbekannte Quellschwierigkeit wird am Ausgangsstand
nicht durchgängig als eigenständige Unknown-Ausprägung erhalten. Projektion,
Defaultfilter, Sortierung, Trefferkonvertierung und UI werden als getrennte
Beobachtungsgrenzen erfasst.

Die Listenprojektion besitzt für bestimmte föderierte Records einen
Remotezweig. Dieser setzt während der Dokumentbildung einen Live-Request auf
`/api/v1/search/lists` der Origininstanz ab und verwendet dessen Aggregate.
Lokale und voll materialisierte Listen werden davon getrennt erfasst.

Actor-, Listen- und Trailprofile sind getrennte Baselineartefakte. SRCH0
definiert daraus kein gemeinsames Zielprofil.

### Weitere Engine-Consumer

`GET /api/v1/search/actor` weist nicht authentifizierte Requests an der Route
mit `401` ab. Authentifizierte Requests suchen im `actors`-Index;
`includeSelf=false` ergänzt den Filter gegen die eigene ID. Ein syntaktisch
gültiger Remote-Handle nimmt zuerst den direkten ActivityPub-Zweig. Diese
Fälle werden nicht mit der allgemeinen Trailmatrix verschmolzen.

Die Upload-Duplikatprüfung sendet beobachtet eine authentifizierte,
tenantgescopte Suche mit leerem `q`, Offset 0, ohne Sortierung und ohne
explizites Limit. Damit gilt das Engine-Defaultlimit 20. Gelesen werden `id`,
`name`, `author_name`, `domain`, `distance`, `elevation_gain`,
`elevation_loss` und `_geo`. Ein passender Datensatz an Position 21 gehört als
eigener Bestandsfall in den Korpus.

Profil-, Empfehlungs-, Bounding-Box-, Cluster- und bekannte Multi-Search-
Aufträge erhalten je eigene Consumerkennung. Zufallsausgaben werden nicht als
bytegenaue Reihenfolge vergoldet; nur ihre beobachtete Eligibility ist
stabiler Fixtureinhalt.

## Bekannte Lücken und Nachfolger

Eine Zeile dieser Tabelle beschreibt ausschliesslich den beobachteten Gap und
seinen nächsten fachlichen Owner. Sie legt kein Zielresultat fest.

| Gap-ID | Beobachtung | `successor_refs` |
| --- | --- | --- |
| `SRCH0-GAP-GEO-001` | Startpunktsuche erzeugt zwei identische `_geoRadius`-Klauseln | `SRCH-COMP` |
| `SRCH0-GAP-DIFF-001` | Missing, leer und unbekannt bleiben nicht an allen Grenzen als Unknown unterscheidbar | `SRCH2` |
| `SRCH0-GAP-LIST-001` | Projektion bestimmter föderierter Listen liest Aggregate live von der Origininstanz | `SEC-VIS-0` |
| `SRCH0-GAP-DATE-001` | inklusives Datumsende erfasst nur den Tagesbeginn | `SRCH-COMP` |
| `SRCH0-GAP-GEO-002` | Latitude oder Longitude `0` deaktiviert den Startpunktradius durch Truthiness | `SRCH-COMP` |
| `SRCH0-GAP-MAP-001` | Kartendefault verwendet für `elevationLossLimit` den Gain-Grenzwert | `SRCH-COMP` |
| `SRCH0-GAP-DTO-001` | `attributesToRetrieve` liegt im allgemeinen Helper ausserhalb von `options` | `SRCH-COMP` |
| `SRCH0-GAP-SORT-001` | unbekannte Storagewerte für Sortkey/-richtung gelangen raw zur Engine | `SRCH-COMP` |
| `SRCH0-GAP-SEC-001` | beobachtete ACL- oder Bypassabweichungen gegen den Securityvertrag | `SEC-VIS-0` |
| `SRCH0-GAP-UI-001` | `TrailFilterPreview` zeigt wirkungslose Beispielwerte | `SRCH4a` |
| `SRCH0-GAP-BOOT-001` | jeder Serve-Start leert und befüllt die drei live verwendeten Indizes asynchron ohne Search-Admission neu | `IDX0` |

Die drei ursprünglich gemeinsam behandelten Bestandskorrekturen besitzen
bewusst keinen gemeinsamen Sammel-Task. SRCH-COMP, SRCH2 und SEC-VIS-0
referenzieren jeweils die betroffene Case-ID und halten Zielsemantik,
Migration und Freigabe in ihrem eigenen Overlay. Dadurch kann keine Korrektur
die anderen beiden oder die Abnahme des neutralen Korpus blockieren.

## Kanonisches Artefaktlayout

```text
testdata/trail-search/srch0/v1/
  schema.json
  manifest.json
  README.md
  profiles/trails-legacy-v0.json
  profiles/lists-legacy-v0.json
  profiles/legacy-actor-v0.json
  datasets/*.json
  cases/state/*.json
  cases/projection/*.json
  cases/compiler/*.json
  cases/search/*.json
  cases/mutation/*.json
  cases/browser/*.json
```

`schema.json` beschreibt nur SRCH0-Basisfälle. `manifest.json` bindet
Dataset-, Profil- und Case-Digests, zählt die Pflichtfamilien und verhindert,
dass ein leeres Verzeichnis als grün gilt.

Die drei Profile enthalten ausschliesslich am Ausgangscommit beobachtete
Settings und Dokumentformen. Zielprofile, Reindexpläne, Verifikationsreports
und Migrationsfälle gehören nicht in diesen Namensraum.

`README.md` ist eine aus Manifest und Fixtures erzeugte menschenlesbare
Bestandsmatrix. CI lehnt Drift zwischen Darstellung und JSON-Artefakten ab.

## Fixture-Vertrag `wanderer.srch0/v1`

Jede Fixturedatei ist genau ein JSON-Objekt mit
`additionalProperties: false`.

| Feld | Vertrag |
| --- | --- |
| `schema_version` | exakt `wanderer.srch0/v1` |
| `case_id` | stabile ID `SRCH0-<FAMILIE>-NNN` |
| `family` | `state`, `projection`, `compiler`, `search`, `mutation` oder `browser` |
| `baseline` | Commit, Beobachtungsdatum, Engineprofile und Settings-Fingerprint |
| `dataset_ref` | Referenz auf genau ein versioniertes synthetisches Dataset |
| `context` | Principal, Locale, Zeitzone, Präferenzen, Surface und Consumer |
| `input` | rohe Eingabe ohne vorweggenommene Normalisierung |
| `observed` | am Ausgangsstand reproduziertes Ergebnis |
| `stability` | `preserve`, `known_gap` oder `non_contract` |
| `successor_refs` | kanonisch sortierte eindeutige Nachfolger-IDs |
| `evidence` | Codeanker, Capture oder begründeter Evidenzverweis |

Für `known_gap` ist mindestens ein `successor_ref` Pflicht. Bei `preserve`
ist die Liste leer. `non_contract` darf einen Nachfolger nennen, wenn dieser
den wirkungslosen oder zufälligen Zustand entfernt, erhebt dessen Ausgabe aber
nicht zum Bestandsversprechen.

SRCH0-Basisfälle enthalten ausdrücklich keine Felder `expected`,
`allowed_delta`, `delivery_owner` oder `activation_gate`. Ein Folgetask legt
solche Werte in einem eigenen, auf `case_id` und Basisdigest gebundenen Overlay
ab.

Roh-URL und LocalStorage bleiben Strings. Doppelte Querykeys, ungültiges JSON
und historische Präzedenz dürfen nicht durch vorgeparste Ersatzobjekte
verloren gehen.

Je nach Familie enthält `observed` unter anderem:

- `legacy_state` mit ausgewertetem Filter, Surface-Default und Pagination;
- `engine_request` mit `q`, bytegenauem Filter, Sortierung und Pagination;
- `projection` mit vollständigen beobachteten Dokumentfeldern;
- `result` mit `hit_groups`, `total`, `page` und `total_pages`;
- `browser_state` mit URL-, Storage-, Snapshot- und Historywerten;
- `mutation_state` mit Vorzustand, Mutation, Engineauftrag und Endzustand; und
- `diagnostics` mit stabiler Kategorie statt lokalisierter Freitextmeldung.

`hit_groups` ist eine geordnete Liste von ID-Gruppen. Eine einelementige
Gruppe ist exakt geordnet. Mehrere IDs in derselben Gruppe dürfen nur dann
tauschen, wenn das beobachtete Profil keinen Tie-Breaker besitzt.

Processing-Zeiten, Task-IDs, zufällige Empfehlungen, signierte Tokens,
Dateipfade und lokalisierte Texte sind keine Goldens.

### Case-ID- und Versionsregel

Eine veröffentlichte `case_id` wird nie für eine andere Eingabe oder Bedeutung
wiederverwendet. Entfällt ein Fall, bleibt seine ID im Manifest als
`retired` mit Begründung und Nachfolgereferenz reserviert.

Eine inkompatible Änderung der Fixtureform oder Feldbedeutung erzeugt
`wanderer.srch0/v2`. Neue Fälle und neue Werte in ausdrücklich offenen
Aufzählungen erhöhen nur Manifest- und Datasetrevision.

### Golden-Regel

Ein Golden wird nie aus der Ausgabe genau des Systems überschrieben, das der
Test gerade prüft. Jede Änderung an `observed` benötigt im selben Review:

1. den reproduzierbaren Ausgangscommit oder einen begründeten neuen
   Bestandsanker;
2. den kleinsten strukturellen Diff;
3. den aktualisierten Code- oder Capturebeleg;
4. die Entscheidung, ob eine neue Korpusversion erforderlich ist; und
5. unveränderte Case-ID oder eine explizit neue ID bei Bedeutungswechsel.

Ein Zieloverlay darf das Basisfixture weder erzeugen noch ersetzen.

## Synthetischer Referenzbestand

Der gemeinsame Datasetkorpus enthält keine Produktionsdaten und mindestens:

- einen anonymen Principal sowie lokale Actors Alice und Bob;
- lokale public-, eigene private-, fremde private- und Actor-geteilte Trails;
- synthetische Remote-, unverified-, revoked- und deleted-Kontexte;
- eine nur als Federation-Stub vorhandene Liste und eine materialisierte Liste;
- zwei Kategorien, mehrere Subkategorien und einen Trail ohne Subkategorie;
- überlappende Tagnamen und actorbezogene ausgeblendete Taxonomien;
- je einen bekannten Difficultywert sowie missing, leer und unbekannt;
- eindeutige Werte und Gleichstände für jede Sortierachse;
- numerische Werte auf, unter und über Rundungs- und Slidergrenzen;
- Datumswerte an einem normalen Tag, am DST-Wechsel und am Tagesende;
- Startpunkte im normalen Bereich sowie Latitude oder Longitude `0`;
- Actor-Self-/Other- und syntaktische Remote-Handle-Fälle;
- mindestens 21 geeignete Duplikatprüfungskandidaten; und
- mindestens 2,5 Seiten bei Seitengrösse 2.

IDs, Timestamps, Locale, Zeitzone und Katalogwerte sind feste, lesbare
Testwerte. Kein Test bezieht Benutzer, Kategorien oder Zeit aus einer
laufenden Entwicklerinstanz.

Ein separat versionierter deterministischer Generator erzeugt 10'001
abgeleitete Trails für Paging-, Cluster- und Lastgrenzen. Diese Masse wird
nicht als handgeschriebenes Golden dupliziert und definiert keine
Produktions-Performancegrenze.

## Mindestabdeckung

| Familie | Pflichtfälle |
| --- | --- |
| State | Listen-/Kartendefault, URL-/Storage-Präzedenz, defektes Storage, Snapshot, Buckets, Page-Keys |
| Volltext | leer, eindeutiger Treffer je Suchattribut, mehrere Tokens, Rankinggleichstand |
| Access | anonymous/authenticated × local/remote × public/private/owned/shared/revoked/deleted |
| Taxonomie | Parent, Child, gleicher/fremder Parent, kein Child, unbekannte ID, Präferenz |
| Tags/Booleans | ein/mehrere Tags, `liked` neutral/aktiv, `completed` ausgelassen/true/false |
| Ranges | jede Achse einzeln/kombiniert, Floor/Ceil, offenes Maximum, Min grösser Max |
| Datum | normaler Tag, DST, Tagesbeginn und Tagesende |
| Difficulty | drei bekannte Werte, missing/leer/unbekannt, Default, Teilmenge, Anzeige, Sortierung |
| Geo | kein Radius, normaler Punkt, Koordinate `0`, Liste, Karte, Bounds und Cluster |
| Sort/Paging | neun Keys in beide Richtungen, Defaults, Gleichstandsgruppen, erste/mittlere/leere Seite |
| Surfaces | Liste, Karte, Profil, Cluster, Bounding Box, Multi-Search und Empfehlung |
| Actor | unauthentifiziert, Self ein/aus, lokaler Lookup und Remote-Handle-Zweig |
| Upload-Duplikat | Tenant-Scope, leeres `q`, Offset 0, Defaultlimit 20, acht gelesene Felder, Position 21 |
| Projektion | Trail-, Listen- und Actordokumente samt Relationsauflösung |
| Mutationen | Create, Update, Delete, Sichtbarkeit, Shares, Likes, Actor, Tags und Taxonomie |
| Startup | vorhandene, fehlende und leere Indizes; Settings-, Delete- und Add-Taskfolge; Request während jeder Zwischenstufe |
| Negative Proxyfälle | unbekannter Index, Feld, Sortkey und freie Retrievaloption als nicht zugesagte Eingaben |
| Browser | URL, Storage, Snapshot, Navigation, Liste, Karte und History |

Jeder produktive First-Party-Consumer muss mindestens einem Fall zugeordnet
sein. Ein Consumer ohne Fixture ist ein Manifestfehler, nicht implizit durch
die allgemeine Suchmatrix abgedeckt.

## Testharness

| Schicht | Aufgabe | Darf nicht ersetzen |
| --- | --- | --- |
| Schema-/Manifesttest | Form, Referenzen, IDs, Digests und Mindestabdeckung | fachliche Ausführung |
| Vitest | Zustand, Sanitizing, Legacycompiler und Trefferkonvertierung | reale Enginefilterung |
| Go-Unit | beobachtete Projektoren und Payloadformen | Browser- und ACL-Verhalten |
| Meilisearch-Integration | Settings, Filter, Rankinggruppen, `total`, Paging und Dokumentform | serverseitigen Principal-Kontext |
| PocketBase/Meili-Mutation | beobachtete Hook- und Relationsfolgen | allgemeine Recoverygarantien |
| SvelteKit/API-Integration | Tokenkontext, Präferenzfilter und bekannte Proxyformen | Browser-History |
| Playwright | URL, Storage, Snapshot, Navigation, Liste und Karte | Engineversionsqualifikation |

Alle Schichten konsumieren dieselben Basisfixtures. Eine Schicht darf
familienfremde Felder ignorieren, aber keine eigene abweichende Kopie des
Goldens pflegen.

Engineintegration läuft getrennt gegen 1.11.3 und 1.36.0. Der Bericht bindet
Engineversion und Settings-Fingerprint. Eine versionsabhängige Beobachtung
wird im Fixture explizit nach Profil diskriminiert und nicht durch eine
gemeinsame Erwartung geglättet.

Tests melden mindestens Case-ID, Familie, Basisdigest, Engineprofil und den
kleinsten strukturellen Diff. Reports enthalten keine privaten
Produktionswerte, Secrets oder aus einer realen Instanz übernommenen Actor-IDs.

Schema-, Manifest-, Go- und Vitest-Suite laufen auf jedem Pull Request. Reale
Engine-, API- und Browsertests dürfen in getrennten Jobs laufen, müssen aber
aus demselben Manifest berichten. Ein alter grüner Lauf ist kein Beleg für
ein geändertes Manifest.

## Abnahme

SRCH0 ist inhaltlich abgenommen, wenn:

1. Schema, Manifest und generierte Bestandsmatrix ohne Drift vorliegen;
2. alle Pflichtfamilien mindestens einen positiven, negativen und relevanten
   Randfall besitzen;
3. alle inventarisierten First-Party-Consumer auf stabile Case-IDs zeigen;
4. die gebundenen Engineprofile ihre jeweils deklarierten Beobachtungen
   reproduzieren;
5. State-, Compiler-, Projektions-, Search-, Mutation- und Browserharness
   denselben Datasetdigest konsumieren;
6. jeder `known_gap` mindestens einen konkreten `successor_ref` besitzt;
7. kein Basisfixture Zielwerte oder Freigabegates eines Nachfolgetasks enthält;
8. ein absichtlich verändertes Golden ohne Evidenzänderung von CI abgewiesen
   wird;
9. eine entfernte Case-Datei nicht als verringerte Mindestabdeckung grün wird;
10. der komplette Aufbau ausschliesslich synthetische Daten verwendet; und
11. die Ausführung keine produktive Runtime, keinen Indexcutover und keine
    Betreiberaktion voraussetzt.

Die Abnahme von SRCH0 sagt ausschliesslich, dass der Bestand reproduzierbar
beschrieben ist. Sie sagt nicht, dass bekannte Lücken behoben oder irgendein
neuer Suchpfad freigegeben ist.

## Abhängige Folgearbeiten

| Nachfolger | Nutzung des SRCH0-Korpus |
| --- | --- |
| IDX0 | ersetzt den beobachteten ungeschützten Startup-Rebuild und bindet sein Ziel an `SRCH0-GAP-BOOT-001` |
| SRCH-V1 | vergleicht die neue fachliche Sprache mit inventarisierten Bestandsfällen |
| SRCH-COMP | bindet Legacyzustände an den typisierten Adapter und besitzt das `_geoRadius`-Overlay |
| SRCH2 | verwendet Feld- und Projektionsfälle und besitzt das Unknown-Difficulty-Overlay samt Backfill |
| SRCH4a | verwendet Browserfälle und besitzt das Overlay zum Entfernen wirkungsloser Previewwerte |
| SEC-VIS-0 | verwendet Access- und Listenfälle, besitzt das lokale Listenaggregat-Overlay und definiert selbst die zulässige Zielmenge |
| IDX1 | verwendet Projektionsfixtures als Paritätsinput für das typisierte Read Model |

Die Kante zeigt vom abgenommenen SRCH0-Korpus zum Nachfolger. SRCH0 wartet
nicht auf Implementierung, Migration oder Livefreigabe dieser Bausteine.

## Beitrags- und Änderungsregel

SRCH0 besitzt drei reviewbare Schnitte:

1. Fixture-Schema, Manifestvalidator und Case-ID-Regel;
2. synthetische Datasets und vollständige Bestandsmatrix; und
3. gemeinsame Harnessadapter und reproduzierbarer Evidenzbericht.

Ein Pull Request, der produktive Semantik ändert, darf sein Ziel nicht durch
Anpassung des SRCH0-`observed`-Werts legitimieren. Er referenziert den
Basisfall, liefert ein Nachfolge-Overlay und lässt den historischen
Bestandsbeleg unverändert.

Wird ein neuer bisher übersehener First-Party-Consumer am Ausgangscommit
gefunden, wird er mit neuer Case-ID ergänzt. Wird dagegen erst durch einen
Nachfolgetask eine neue Capability eingeführt, gehört ihr Fixture nicht in
SRCH0.

## Entscheidungsprotokoll

| Datum | Entscheidung | Begründung |
| --- | --- | --- |
| 2026-08-30 | Commit `e6db729c` ist die Ausgangsrevision | Codeanker und Beobachtungen benötigen denselben reproduzierbaren Stand |
| 2026-08-30 | Ein sprachneutraler JSON-Korpus speist alle Testschichten | getrennte Goldens würden unbemerkt divergieren |
| 2026-08-30 | Case-IDs bleiben stabil und werden nie neu verwendet | Nachfolgetasks brauchen dauerhafte Referenzen |
| 2026-09-01 | SRCH0 enthält nur `observed`, keine Zielerwartung | Bestandsvertrag und Korrekturvertrag bleiben unabhängig |
| 2026-09-01 | Die drei Bestandskorrekturen wechseln getrennt zu SRCH-COMP, SRCH2 und SEC-VIS-0 | ihre Semantik und Auslieferung sind kein Bestandteil der Bestandsaufnahme und besitzen keinen gemeinsamen natürlichen Owner |
| 2026-09-01 | Der beobachtete ungeschützte Startup-Rebuild verweist auf IDX0 | Bootstrap und Readiness erhalten einen Runtimeowner, ohne Teil von SRCH0 zu werden |
| 2026-09-01 | Runtime-, Reindex- und Cutoverdesign ist nicht normativ archiviert | die technische Exploration bleibt erhalten, ohne SRCH0 aufzublähen |
