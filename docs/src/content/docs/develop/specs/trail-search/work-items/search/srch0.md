---
title: SRCH0 — Korrekte Suchbasis und Testkorpus
description: Reproduzierbarer Suchbestand, verbindliche Fehlerkorrekturen und gemeinsame Regressionstests vor der V1-Migration.
editUrl: false
sidebar:
  order: 1
  badge: Blockiert
spec:
  id: SRCH0
  kind: work-item
  status: reviewable
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-foundation
  exposure: internal
  implementationDependsOn: []
  releaseGates: []
  normativeSources: [TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-09-19'
---

## Zweck

SRCH0 schafft eine fachlich geprüfte, korrigierte Ausgangsbasis für die
Weiterentwicklung der Suche. Der gemeinsame Testkorpus prüft Eingaben der
First-Party-Oberflächen, Indexprojektionen und tatsächliche Suchergebnisse.
Ein reproduzierbarer Fehler ist kein zulässiges Regressionsergebnis.

Der Korpus unterscheidet drei Ebenen:

1. **Historische Beobachtung:** `observed` hält nachvollziehbar fest, was der
   gebundene Ausgangscommit tatsächlich getan hat.
2. **Aktive Solländerung:** Ein separat gespeicherter, an Case-ID und
   Basisdigest gebundener Korrekturfall beschreibt das fachlich richtige
   Ergebnis. Er ersetzt im Produktprüfungslauf die historische Erwartung.
3. **Unabhängige Eigenschaften:** Sichtbarkeit, Vollständigkeit, gültige
   Werte und konsistente Zustände werden zusätzlich ohne Ableitung aus den
   Goldens geprüft. Eine Solländerung darf diese Prüfungen nicht umgehen.

Bekannte, nachgewiesene Produktfehler im geprüften Umfang blockieren Merge
und Abnahme, bis die Korrekturen integriert sind und die betroffenen Tests
bestehen. `known_gap`, ein `successor_ref` oder ein historisch grüner Lauf
sind keine Ausnahme. Erst diese korrigierte Ausgangsbasis bildet den
Kompatibilitätsvertrag für nachfolgende Arbeiten.

Tests und Produktkorrekturen dürfen in getrennten PRs entstehen. SRCH0
verantwortet den Korrektheitsnachweis; die Produkt-PRs liefern die dazu
notwendigen Änderungen samt gegebenenfalls erforderlicher Bestandsreparatur.
SRCH0 führt keine neue Suchfunktion oder Runtime-Control-Plane ein.

## Metadaten und Bestandsanker

| Feld | Wert |
| --- | --- |
| Work Item | `SRCH0` |
| Ergebnis | gemeinsamer Testkorpus und nachweislich korrigierte Suchbasis |
| Ausgangsrevision | Git-Commit `e9b7a8cade980002acbcf2e2f5b2a083934f29d2` |
| Beobachtungsdatum | 7. September 2026 |
| Dokumentrevision | 19. September 2026 |
| Engine-Ausgangsprofile | Meilisearch 1.11.3 und 1.36.0 mit den Settings der Ausgangsrevision |
| Exposure | Testpaket intern; notwendige Produktkorrekturen in separaten PRs |
| Implementierungsabhängigkeiten | keine für den Aufbau des Korpus und der Tests |
| Abnahmevoraussetzung | alle nachgewiesenen Fehler im Prüfungsumfang behoben und am integrierten Zielcommit geprüft |
| Nachfolger | IDX0, SRCH-V1, SRCH-COMP, SRCH2, SRCH4a, SEC-VIS-0 und IDX1 |

Der historische Bestandsanker besteht aus Commit, Engineprofil,
Fixture-Schema und Datasetrevision. Die Produktabnahme bindet zusätzlich den
tatsächlich geprüften Zielcommit, den Manifestdigest und den Digest der
aktiven Solländerungen. Ein Ergebnis ohne diese Bindungen ist keine
SRCH0-Abnahme. Der frühere Anker `e6db729c` bleibt historische Vorarbeit;
Beobachtungen verschiedener Ausgangsrevisionen dürfen nicht vermischt werden.

### Bedeutung normativer Sprache

`MUSS`, `DARF NICHT` und vergleichbare Formulierungen regeln sowohl die
Reproduzierbarkeit des Korpus als auch die fachliche Produktabnahme.
`observed` ist historische Evidenz, keine Autorität über ein korrektes
Sollresultat. Ohne Solländerung gilt der beobachtete Wert nur, sofern er die
unabhängigen fachlichen Prüfungen besteht. Ein dabei neu gefundener Fehler
blockiert den Lauf auch bei einem bislang als `preserve` markierten Fall.

## Scope

SRCH0 umfasst:

- die produktiv verwendeten Suchoberflächen Trail-Liste, Karte und Profil;
- URL-, LocalStorage-, Snapshot-, History- und Paginationzustände dieser
  Oberflächen;
- alle im First-Party-Code erzeugten Trailfilter und sichtbaren Sortierungen;
- die bekannten Einzel- und Multi-Search-Aufträge;
- die PocketBase-zu-Meilisearch-Projektionen für Trails, Listen und Actors;
- Indexerhalt, erfolgreiche Erstinitialisierung, Fehlerweitergabe und
  Wiederaufnahme beim Prozessstart dieser drei Indizes;
- Tenant-Token-Kontext und serverseitige Kategoriepräferenzen als beobachtete
  Eingaben der Treffermenge;
- Trefferreihenfolge, Gleichstandsgruppen, `total`, Seite und Seitengrösse;
- den authentifizierten Actor-Suchpfad;
- die Meilisearch-gestützte Duplikatprüfung beim Trail-Upload;
- die bekannten Profil-, Empfehlungs-, Bounding-Box- und Clusterabfragen;
- Create-, Update-, Delete- und Relationsmutationen, soweit sie heute
  Suchdokumente erzeugen oder verändern; und
- einen gemeinsamen JSON-Korpus für Go-, Vitest-, Engine-, API- und
  Browsertests einschließlich aktiver Solländerungen und unabhängiger
  Korrektheitsprüfungen.

Der rohe Proxy `POST /api/v1/search/{index}` kann technisch mehr ausdrücken
als die Anwendung selbst verwendet. SRCH0 inventarisiert nur nachweisbare
First-Party-Aufträge. Zufällig von Meilisearch akzeptierte freie Optionen sind
kein Bestandsvertrag.

## Nichtziele

- Keine neuen Produktfähigkeiten über die Korrektur der geprüften Suche hinaus.
- Keine neue öffentliche Such-API oder V1-Requestgrammatik.
- Keine V1-Normalisierung und kein kanonischer V1-URL-Codec.
- Kein neues typisiertes Zielindexschema oder allgemeines Migrationssystem;
  notwendige Reparaturen bestehender Suchdokumente gehören zur jeweiligen
  Produktkorrektur und müssen mitgeprüft werden.
- Kein vollständiger IDX0-Readinessproduzent, Wrapper-Rollout oder
  generationsgebundener Index-Lifecycle; verbindlich bleibt die unten
  beschriebene Startup-Korrektheit.
- Keine Epochenattestierung oder allgemeine Write-Admission-Control-Plane.
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

Für die historische Reproduktion wird die Evidenz in folgender Reihenfolge
bewertet:

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

Für die Produktabnahme gelten die fachlichen Eigenschaften und begründeten
aktiven Solländerungen. Ein historischer Goldenvergleich kann ihre Verletzung
nicht freigeben. Ein sicherheitswidriges Ergebnis wird als `known_gap`
dokumentiert und blockiert die Abnahme bis zur nachgewiesenen Behebung.

Eine noch nicht umgesetzte neue Fähigkeit ist dagegen nicht allein deshalb
ein Produktfehler. Beispielsweise beweist ein Remoteaufruf der heutigen
Listenprojektion für sich keinen unzulässigen Datenabfluss. Neue
Federation-, Snapshot- oder Routenradiusverträge behalten ihre eigenen
Abnahmen; nachgewiesene Fehler des bereits geprüften Produkts dürfen nicht
unter Berufung auf diese Folgearbeiten vertagt werden.

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

Die historische Auftragsfolge bleibt als Diagnose erhalten. Der aktive
Produkttest verlangt dagegen:

- Ein normaler Neustart erhält vorhandene korrekte Indexdaten.
- Fehlende oder unvollständig initialisierte Indizes werden kontrolliert
  hergestellt; Suchbereitschaft setzt terminal erfolgreiche notwendige
  Settings- und Dokumentaufträge voraus.
- Fehler und Timeouts werden weitergegeben und dürfen keine vorgetäuschte
  Suchbereitschaft erzeugen.
- Ein abgebrochener Erstaufbau kann beim nächsten Start wiederaufgenommen
  werden; Teilbestände gelten nicht still als vollständig.

Die Produktkorrektur wird im Startup-Paket geliefert. `IDX0` besitzt darüber
hinaus den normativen Readinessproduzenten, den Offline-Rebuild und den
Betriebsrollout. Dessen vollständige Umsetzung ist keine Voraussetzung für
den Aufbau der SRCH0-Tests; die genannten Startup-Eigenschaften sind jedoch
Voraussetzung für ihre Abnahme. Die Fixtures verwenden `family: mutation`
und einen eigenen Startup-Consumer, keine siebte Fixturefamilie.

## Historisch beobachtete Legacy-Semantik

Die folgenden Abschnitte dokumentieren den Ausgangscommit. Beschriebene
Fehler sind keine Sollwerte der Produktabnahme; die aktiven Korrekturen und
fachlichen Eigenschaften stehen im anschliessenden Fehlerkatalog.

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
`known_gap` dokumentiert und muss vor SRCH0-Abnahme geschlossen werden.
SEC-VIS-0 behält seine weitergehenden Freigabegates.

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

## Bekannte Fehler und verbindliche Korrekturen

Die stabilen Gap-IDs verbinden historische Beobachtungen, konkrete Case-IDs
und Produktkorrekturen. Die folgenden Sollwerte gehören bereits zur
SRCH0-Abnahme; Nachfolgetasks müssen sie erhalten.

| Gap-ID | Historischer Befund | Aktive Erwartung |
| --- | --- | --- |
| `SRCH0-GAP-GEO-001` | doppelte identische `_geoRadius`-Klausel | genau eine Klausel; diese strukturelle Bereinigung ändert Treffer, `total`, Reihenfolge und Seite nicht |
| `SRCH0-GAP-DIFF-001` | fehlende oder ungültige Schwierigkeit wird als bekannte Stufe erfunden | Unknown bleibt in Projektion und Trefferdarstellung unbekannt; Defaultfilter schliesst Unknown ein, echte Teilmengen bekannter Stufen nicht |
| `SRCH0-GAP-DATE-001` | inklusives Datumsende erfasst nur den Tagesbeginn | vollständiger lokaler Kalendertag einschliesslich 23-/25-Stunden-Tagen |
| `SRCH0-GAP-GEO-002` | Koordinate `0` deaktiviert den Radius | Latitude und Longitude `0` bleiben gültige Anker |
| `SRCH0-GAP-MAP-001` | Abstiegslimit verwendet den Aufstiegsgrenzwert | Abstieg verwendet den Abstiegsgrenzwert |
| `SRCH0-GAP-DTO-001` | Feldauswahl erreicht die Engine nicht | beabsichtigte Retrievalfelder werden tatsächlich angewendet |
| `SRCH0-GAP-SORT-001` | ungültige Storagewerte erreichen die Engine | gültiger dokumentierter Fallback vor dem Enginezugriff |
| `SRCH0-GAP-SEC-001` | nachgewiesene unzulässige Sichtbarkeit oder Umgehung im geprüften Suchpfad | keine unzulässigen Treffer oder Counts im betreffenden Principal-Kontext |
| `SRCH0-GAP-BOOT-001` | Startup leert live verwendete Indizes asynchron | Indexerhalt, terminal erfolgreiche Initialisierung, Fehlerweitergabe und Wiederaufnahme |

Der ausführbare Korpus ergänzt mindestens die auf `feat/srch0` belegten
Fehlerfälle: SDK-HTTP-Statusweitergabe, Actor-Suchparameter, negative
Thumbnailindizes, Erhalt verbleibender Shares, Aktualisierung abhängiger
Actor-/Tag-/Kategoriemetadaten und vollständige Materialisierung fehlender
Indexdokumente. Clustergrundlage und Upload-Duplikatprüfung müssen die gesamte
zulässige Kandidatenmenge berücksichtigen; Engine-Defaultlimits und
`maxTotalHits` dürfen kein erfolgreiches unvollständiges Ergebnis erzeugen.
Ein Duplikat hinter Position 20 und tatsächlich indexierte 10'001 Trails
gehören zum Nachweis.

Zwei bestehende Referenzen bezeichnen weitergehende Folgearbeiten:

- `SRCH0-GAP-LIST-001`: SEC-VIS-0 stellt Listenaggregate auf die lokale
  Relation um und besitzt dafür Migration und Freigabe. Ein dabei in SRCH0
  belegter aktueller ACL- oder Projektionsfehler muss bereits vor dessen
  Abnahme behoben werden; der Remoteaufruf allein belegt keinen solchen Fehler.
- `SRCH0-GAP-UI-001`: Der UI-Prototyp aus dem Spec-Branch ist nicht Teil des
  Ausgangscommits `e9b7a8cad` und keine geprüfte Produktfähigkeit. SRCH4a
  ersetzt ihn durch ein echtes Panel; Beispielwerte werden nicht als
  korrekte Suchergebnisse in die Baseline aufgenommen.

`successor_refs` dokumentieren die spätere Nutzung, keine Fristverlängerung
für einen bekannten Fehler. Die Korrekturen des bisherigen Produkts werden
vor den darauf aufbauenden Vertrags- und UI-Arbeiten geliefert. Damit
entsteht keine zyklische Abhängigkeit zu SRCH-COMP, SRCH2 oder SEC-VIS-0.

## Kanonisches Artefaktlayout

```text
testdata/trail-search/srch0/v1/
  schema.json
  manifest.json
  changes.json
  inventory.json
  sources.json
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

`schema.json` beschreibt die historischen SRCH0-Basisfälle. `changes.json`
enthält die aktiven Solländerungen. `manifest.json` bindet Dataset-, Profil-,
Case- und Änderungsdigests, zählt die Pflichtfamilien und verhindert, dass ein
leeres Verzeichnis als grün gilt. Inventar und Quellanker machen fehlende
Consumer und veränderte Produktionspfade prüfbar.

Die drei Profile enthalten ausschliesslich am Ausgangscommit beobachtete
Settings und Dokumentformen. Korrekturprüfungen verwenden die Produktsettings
des Zielcommits; historische Profile dürfen keine zur Korrektur notwendige
Produktänderung verdecken. Neue Generationenprofile und allgemeine
Migrationspläne gehören weiterhin zum jeweiligen Folge-Work-Item.

`README.md` ist eine aus Manifest und Fixtures erzeugte menschenlesbare
Bestandsmatrix. CI lehnt Drift zwischen Darstellung und JSON-Artefakten ab.

## Fixture-Vertrag `wanderer.srch0/v1`

Jeder normalisierte Basisfall ist genau ein JSON-Objekt mit
`additionalProperties: false`. Physische Dateien dürfen mehrere Fälle einer
Gruppe mit gemeinsamen Metadaten enthalten; der Validator expandiert sie
deterministisch vor Schema- und Digestprüfung.

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

SRCH0-Basisfälle enthalten weiterhin keine Felder `expected`,
`allowed_delta`, `delivery_owner` oder `activation_gate`. Aktive Korrekturen
liegen bereits in SRCH0 separat in `changes.json`; sie sind keine optionalen
Erwartungen eines erst später zu liefernden Folgetasks.

### Aktive Solländerungen

`changes.json` verwendet `schema_version: wanderer.srch0.changes/v1`.
Jeder Eintrag enthält eine eindeutige `id`, `case_id`, `basis_digest`,
fachliche `reason`, `evidence` und die vollständige korrigierte Ergebnisform
unter `observed`. Dieser Feldname bezeichnet hier den Ersatzwert für den
Produktprüfungslauf, keine nachträgliche historische Beobachtung.

Pro Fall ist höchstens eine aktive Änderung zulässig. Fehlende Referenzen,
falsche Digests, mehrere konkurrierende Änderungen oder unvollständige
Ersatzwerte lassen die Prüfung scheitern. Eine neue Korrektur verändert
weder Eingabe noch historischen Basisfall. Im Produktprüfungslauf gilt der
Ersatzwert; ohne Änderung gilt der historische Wert unter dem Vorbehalt der
unabhängigen Eigenschaften. Ein nachgewiesener Fehler ohne Solländerung ist
ein offener Blocker, kein implizit freigegebener Fall.

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

Eine Solländerung darf das historische Basisfixture weder erzeugen noch
überschreiben. Auch ihre Änderung benötigt eine fachliche Begründung und
unabhängige Evidenz; der aktuelle Produktoutput allein genügt nicht.

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

PocketBase-Proben verwenden das reale Produktionsschema und seine
Validierungsregeln. Nicht speicherbare Rohwerte werden als defensive
Projektor-/Indexfälle gekennzeichnet und nicht als regulär erzeugbare
Datenbankzustände ausgegeben. Ungültige Mutationen müssen ihre tatsächliche
Ablehnung prüfen; vereinfachte Testcollections dürfen keine fiktiven
Produktfehler erzeugen.

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

Alle Schichten konsumieren dieselben Basisfixtures und aktiven Solländerungen.
Eine Schicht darf familienfremde Felder ignorieren, aber keine eigene
abweichende Kopie des Goldens pflegen.

Historische Reproduktion und Produktabnahme sind getrennte Laufarten. Ein
erfolgreicher historischer Lauf sagt nur, dass die Beobachtung reproduziert
wurde; er kann den aktuellen Produktlauf nicht ersetzen. Dieser prüft die
integrierten Produktkorrekturen mit aktiven Erwartungen und unabhängigen
Eigenschaften. Sichtbarkeit wird aus den Public-/Autor-/Sharebeziehungen des
synthetischen Quelldatasets abgeleitet, nicht aus erwarteten Trefferlisten
oder dem vom Produkt erzeugten Tenant-Filter. Weitere Properties prüfen
Vollständigkeit, eindeutige IDs, konsistente Counts und Seiten, Werterhalt,
Range-Monotonie und numerische Sortierung bei leerem Suchtext.

Verletzungen schlagen immer fehl. `knownViolation`, erwartetes Fehlschlagen,
Überspringen oder ein Golden-Update dürfen keinen bekannten Produktfehler
grün machen. Eine nachgewiesene Redundanz ist von einem Ergebnisfehler zu
unterscheiden; eine angenommene strukturelle Bereinigung erhält einen
eigenen Delta-Test.

Engineintegration läuft getrennt gegen 1.11.3 und 1.36.0. Der Bericht bindet
Engineversion und Settings-Fingerprint. Eine versionsabhängige Beobachtung
wird im Fixture explizit nach Profil diskriminiert und nicht durch eine
gemeinsame Erwartung geglättet.

Tests melden mindestens Case-ID, Familie, Basisdigest, Änderungsdigest,
Zielcommit, Engineprofil und den kleinsten strukturellen Diff. Reports enthalten keine privaten
Produktionswerte, Secrets oder aus einer realen Instanz übernommenen Actor-IDs.

Schema-, Manifest-, Go- und Vitest-Suite laufen auf jedem Pull Request. Reale
Engine-, API- und Browsertests dürfen in getrennten Jobs laufen, müssen aber
aus demselben Manifest und demselben Zielcommit berichten. Ein alter grüner
Lauf ist kein Beleg für einen geänderten Korpus oder Produktstand.

## Abnahme

SRCH0 ist erst mergebar und abgenommen, wenn:

1. Schema, Manifest und generierte Bestandsmatrix ohne Drift vorliegen;
2. alle Pflichtfamilien mindestens einen positiven, negativen und relevanten
   Randfall besitzen;
3. alle inventarisierten First-Party-Consumer auf stabile Case-IDs zeigen;
4. historische Beobachtungen reproduzierbar gebunden sind und beide
   Engineprofile die aktiven Produktprüfungen bestehen;
5. State-, Compiler-, Projektions-, Search-, Mutation- und Browserharness
   denselben Datasetdigest konsumieren;
6. alle nachgewiesenen Fehler im Prüfungsumfang eine begründete aktive
   Korrektur und einen grünen unabhängigen Regressionstest besitzen;
7. historische Basisfälle und aktive Solländerungen getrennt validiert sind;
8. ein absichtlich verändertes Golden ohne Evidenzänderung von CI abgewiesen
   wird;
9. eine entfernte Case-Datei nicht als verringerte Mindestabdeckung grün wird;
10. der komplette Aufbau ausschliesslich synthetische Daten und für
    Datenbankproben die tatsächlichen Produktionsregeln verwendet;
11. der Testaufbau weder eine reale Betreiberinstanz noch deren Daten benötigt;
12. Startup-, API-, Engine- und Browsertests einschließlich der betroffenen
    Bestandsreparaturen am tatsächlich integrierten Zielstand grün sind; und
13. kein bekannter Fehler durch eine Ausnahme oder ausschliesslich durch den
    Vergleich mit einem ebenfalls fehlerhaften Golden freigegeben wird.

Die Abnahme bestätigt Korrektheit im dokumentierten Prüfungsumfang. Sie ist
weder ein Beweis für beliebige ACL-/Federationszustände noch die Freigabe
eines neuen Suchpfads. Korpusvollständigkeit allein ist kein Abschluss.

### Aktueller Umsetzungsbezug

Stand 19. September 2026 liegt die Testimplementierung auf `feat/srch0`
(`fc6e6c62b`). `fix/srch0-findings` bleibt die Sammelreferenz für die bereits
implementierten Produktkorrekturen; dessen Stand `754456831` enthält den
Merge von `dev` (`c73966d6c`). Fachlich unabhängige Korrekturen werden daraus
einzeln auf frischen `dev`-Branches vorbereitet, jeweils mit passenden
Regressionstests und einem eigenen PR. Zusammengehörige Änderungen zur
Behebung desselben Fehlers bleiben in einem PR.

Als erste Auskopplung ist die Radiuskorrektur lokal vorbereitet:

| Feld | Stand |
| --- | --- |
| Branch / Commit | `fix/search-radius-filter` / `398b45682` |
| Direkte Basis | `origin/dev` bei `c73966d6c` |
| Befunde | `SRCH0-GAP-GEO-001` und `SRCH0-GAP-GEO-002`; genau eine Radiusklausel, gültige Nullkoordinaten, Prüfung der Koordinatengrenzen und eines endlichen positiven Radius |
| Gezielte Regressionen | 24 Fälle in `web/src/lib/stores/trail_store.test.ts`; vor der Korrektur 20 fehlgeschlagen, nach der Korrektur alle erfolgreich |
| Prüfung des Fixcommits | `npm run test:unit -- --run` im Webverzeichnis: 145 Tests erfolgreich; `npm run check`: keine Fehler oder Warnungen |
| Veröffentlichung / Integration | lokal, nicht gepusht, kein PR; weder in `dev` noch in `feat/srch0` integriert |

Die Radiusprüfung belegt den isolierten Fix, keine SRCH0-Gesamtabnahme. Der
Befund bleibt für den ungefixten Produktstand offen. Das vorgesehene
Startup-Paket auf `fix/search-index-startup` ist weiterhin nicht
implementiert; der Branch zeigt auf die Ausgangsrevision. Alle erforderlichen
Produktkorrekturen einschliesslich Startup müssen vor SRCH0-Abnahme
integriert und gemeinsam geprüft sein. Ein früherer grüner kombinierter
Prüfbaum belegt nicht den verbliebenen Stand ohne Startup-Paket. Die Branches
sind Arbeitsbezüge, keine zusätzlichen fachlichen Work Items oder bereits
erteilte Freigaben.

## Abhängige Folgearbeiten

| Nachfolger | Nutzung des SRCH0-Korpus |
| --- | --- |
| IDX0 | erhält die geprüfte Startup-Korrektheit und ergänzt Readinessproduzent, Offline-Rebuild und Betriebsrollout |
| SRCH-V1 | vergleicht die neue fachliche Sprache mit inventarisierten Bestandsfällen |
| SRCH-COMP | bindet korrigierte Legacyzustände an den typisierten Adapter und erhält die bestandenen Radius-, Datums- und Requestprüfungen |
| SRCH2 | erhält korrekte Unknown-Behandlung und ergänzt typisierte Feld-/Presence-Semantik samt erforderlichem Backfill |
| SRCH4a | verwendet Browserfälle und besitzt das Overlay zum Entfernen wirkungsloser Previewwerte |
| SEC-VIS-0 | verwendet Access- und Listenfälle, besitzt das lokale Listenaggregat-Overlay und definiert selbst die zulässige Zielmenge |
| IDX1 | verwendet Projektionsfixtures als Paritätsinput für das typisierte Read Model |

Die Kante zeigt von der abgenommenen korrigierten Suchbasis zum Nachfolger.
Notwendige Korrekturen des geprüften Bestands werden vorgezogen; sie dürfen
nicht erst einen solchen Nachfolger voraussetzen. Neue Fähigkeiten und
weitergehende Runtimearchitektur bleiben unabhängige Folgearbeiten. IDX0
kann ohne fertigen SRCH0-Korpus begonnen werden.

## Beitrags- und Änderungsregel

SRCH0 besitzt vier reviewbare Schnitte:

1. Fixture-Schema, Manifestvalidator und Case-ID-Regel;
2. synthetische Datasets und vollständige Bestandsmatrix;
3. gemeinsame Harnessadapter, aktive Solländerungen und Properties; und
4. separate Produktkorrekturen mit gemeinsamem grünem Integrationsnachweis.

Ein Pull Request, der produktive Semantik ändert, darf sein Ziel nicht durch
Anpassung des SRCH0-`observed`-Werts legitimieren. Er referenziert den
Basisfall, liefert eine aktive Solländerung und lässt den historischen
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
| 2026-09-19 | Fachlich korrekte aktive Erwartungen und unabhängige Properties bestimmen die Abnahme; historische Beobachtungen bleiben Evidenz | die rein beobachtende Abnahmeregel vom 1. September würde bekannte Fehler als grüne Ausgangsbasis festschreiben und ist ersetzt |
| 2026-09-19 | Ausgangsrevision ist `e9b7a8cad` vom 7. September; Tests und Produktfixes bleiben getrennt reviewbar | die Spezifikation folgt dem ausführbaren Korpus auf `feat/srch0`, ohne dessen noch blockierten Stand als abgenommen auszugeben |
| 2026-09-19 | Nachgewiesene Bestandsfehler einschließlich Startup werden vor SRCH0-Abnahme behoben | spätere Owner erhalten diese Korrekturen; ihre vollständige neue Architektur wird dadurch nicht zur zyklischen Voraussetzung |
| 2026-09-19 | Fachlich unabhängige Produktkorrekturen erhalten eigene PRs samt Regressionstests; erster lokaler Fix ist `fix/search-radius-filter` (`398b45682`) | die Sammelkorrekturen bleiben Referenz; der isolierte Radiusnachweis ersetzt weder Integration noch SRCH0-Gesamtabnahme |
