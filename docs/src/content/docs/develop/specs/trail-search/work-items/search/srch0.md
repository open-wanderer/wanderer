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
Ein reproduzierbarer Fehler im verbindlichen Abnahmeumfang ist kein
zulässiges Regressionsergebnis.

Der Korpus unterscheidet drei Ebenen:

1. **Historische Beobachtung:** `observed` hält nachvollziehbar fest, was der
   gebundene Ausgangscommit tatsächlich getan hat.
2. **Aktive Solländerung:** Ein separat gespeicherter, an Case-ID und
   Basisdigest gebundener Korrekturfall beschreibt das fachlich richtige
   Ergebnis. Er ersetzt im Produktprüfungslauf die historische Erwartung.
3. **Unabhängige Eigenschaften:** Sichtbarkeit, Vollständigkeit, gültige
   Werte und konsistente Zustände werden zusätzlich ohne Ableitung aus den
   Goldens geprüft. Eine Solländerung darf diese Prüfungen nicht umgehen.

Bekannte, nachgewiesene Produktfehler im verbindlichen Abnahmeumfang
blockieren Merge und Abnahme, bis die Korrekturen integriert sind und die
betroffenen Tests bestehen. `known_gap`, ein `successor_ref` oder ein historisch grüner Lauf
sind keine Ausnahme. Erst diese korrigierte Ausgangsbasis bildet den
Kompatibilitätsvertrag für nachfolgende Arbeiten. Die Sortier-Robustheit
`SRCH0-GAP-SORT-001`, die Feldauswahl `SRCH0-GAP-DTO-001`, die globale
Tag-Umbenennung `SRCH0-MUTATION-008`, negative Thumbnailindizes
`SRCH0-PROJECTION-021`, die SDK-HTTP-Statusweitergabe (`SRCH0-P-HTTP`),
die unten abgegrenzten Actor-Suchparameter, die Upload-Duplikatprüfung
und die Vollständigkeit der Clustergrundlage oberhalb des bestehenden
Karten-Caps gehören nicht zu diesem Abnahmeumfang;
ihre Diagnosefälle bleiben nachvollziehbar erhalten.

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
| Abnahmevoraussetzung | alle nachgewiesenen Fehler im verbindlichen Abnahmeumfang behoben und am integrierten Zielcommit geprüft; die unten abgegrenzten Befunde zu Sortier-Robustheit, Feldauswahl, Tag-Umbenennung, negativen Thumbnailindizes, SDK-HTTP-Statusweitergabe, Actor-Suchparametern, Upload-Duplikatprüfung und Cluster-Cap sind keine Blocker |
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
im verbindlichen Abnahmeumfang blockiert den Lauf auch bei einem bislang als
`preserve` markierten Fall.

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
eigener Bestandsfall in den Korpus. Die Prüfung bleibt als technischer
Meilisearch-Consumer inventarisiert; ihre fachlich zum Import gehörende
Korrektur ist keine SRCH0-Abnahmevoraussetzung.

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
| `SRCH0-GAP-SEC-001` | nachgewiesene unzulässige Sichtbarkeit oder Umgehung im geprüften Suchpfad | keine unzulässigen Treffer oder Counts im betreffenden Principal-Kontext |
| `SRCH0-GAP-BOOT-001` | Startup leert live verwendete Indizes asynchron | Indexerhalt, terminal erfolgreiche Initialisierung, Fehlerweitergabe und Wiederaufnahme |

Der ausführbare Korpus ergänzt mindestens die auf `feat/srch0` belegten
Fehlerfälle: SDK-HTTP-Statusweitergabe, Actor-Suchparameter, negative
Thumbnailindizes, Erhalt verbleibender Shares, Aktualisierung abhängiger
Actor-/Tag-/Kategoriemetadaten. Die globale Tag-Umbenennung
`SRCH0-MUTATION-008`, negative Thumbnailindizes `SRCH0-PROJECTION-021`,
SDK-HTTP-Statusweitergabe und die unten abgegrenzten Actor-Suchparameter
bleiben dabei als nicht blockierende Diagnosefälle erhalten. Diese Ausnahmen
gelten weder pauschal für andere Metadatenmutationen oder Projektionsfehler
noch für Authentifizierung, Berechtigungen oder als Erfolg ausgegebene
Enginefehler.

Die Prüfungen fehlender Indexdokumente sichern die neu eingeführten
Metadaten-Teilaktualisierungen ab: Ein fehlender Eintrag darf durch sie nicht
als unvollständiger Treffer entstehen. Die Tests stellen diesen Ausgangszustand
gezielt her; daraus folgt kein zusätzlich nachgewiesener Bedienfehler in
`dev`. Diese Absicherung gehört zur jeweiligen Metadaten-Korrektur und erhält
kein separates Arbeitspaket, keinen eigenen PR und keinen zusätzlichen
SRCH0-Blockerstatus. Sobald eine solche Korrektur eingeführt wird, muss sie
diese Anforderung samt Regressionstests erfüllen. Für Tags ist dies bereits
in `fix/search-tag-metadata` (`122974380`) enthalten; die Absicherung macht
den nicht blockierenden Tag-Fix nicht zur Abnahmevoraussetzung. Bestehende
Startup- und andere Indexfehler werden dadurch nicht neu eingestuft. Tests
und Korpus bleiben unverändert; die noch ausstehende Trennung von Diagnose
und Abnahme muss diese Zuordnung ebenfalls berücksichtigen.

Die bisherige Forderung, für Cluster und Upload-Duplikatprüfung vor
SRCH0-Abnahme sämtliche Kandidaten über die vorhandenen Limits hinaus
zu erfassen, ist aufgehoben. Der Importfix wird separat umgesetzt;
das bestehende Karten-Cap wird vorerst als bekannte Begrenzung akzeptiert.
Die Nachweise mit einem Duplikat ausserhalb der ersten 20 Treffer und
mit tatsächlich indexierten 10'001 Trails bleiben als Diagnose erhalten.
Die folgenden Einstufungen begrenzen diese Ausnahmen; unzulässige
Zugriffe oder Treffer werden dadurch nicht akzeptiert.

### Zurückgestellte Sortier-Robustheit

**Entscheidung vom 19. September 2026:** `SRCH0-GAP-SORT-001` ist kein
Merge- oder Abnahmeblocker für SRCH0. Die Bereinigung ungültiger gespeicherter
Werte für `sort` und `sort_order` sowie der Fallback bei fehlender oder
ungültiger Richtung werden als kleine Robustheitsverbesserung zurückgestellt.
Vorerst wird dafür kein eigener Produkt-PR vorbereitet.

Die normale Oberfläche erzeugt gültige Sortierwerte. Der belegte negative
Testfall verwendet absichtlich ungültige Browser-Storagewerte; ein Fehler
im gewöhnlichen Gebrauch ist dafür bisher nicht nachgewiesen. Auch die
Fallback-Richtung wird derzeit nicht als Voraussetzung der Suchbasis gewertet.
Der gewünschte spätere Fallback auf die gültigen Vorgaben der jeweiligen
Ansicht bleibt dokumentiert, ist aber keine aktive SRCH0-Abnahmeforderung.

Gap-ID, Case-IDs und historische Beobachtungen bleiben erhalten. Dazu gehören
`SRCH0-BROWSER-009` und die Solländerung `WEB-FIX-SRCH0-BROWSER-009`.
Die betreffenden Proben dienen normativ nur der Diagnose. Die vorhandene
Testimplementierung ist noch nicht an diese Einordnung angepasst und kann
die Fälle weiterhin als verpflichtende Korrekturen prüfen. Ihre Trennung
von den verbindlichen Abnahmeprüfungen ist bei der nächsten Anpassung der
Suite nachzuführen; diese Dokumentänderung behauptet keinen grünen Lauf und
fordert keinen Sortierfix als zusätzliches Releasegate.

Die Tests der neun gültigen Sortierfelder in beiden Richtungen bleiben
verbindlich. Textrelevanz und Gleichstandsbehandlung sind eigenständige
Themen und von dieser Entscheidung nicht betroffen.

### Nicht blockierende Feldauswahl

**Entscheidung vom 19. September 2026:** `SRCH0-GAP-DTO-001` ist kein
Merge- oder Abnahmeblocker für SRCH0, wird aber separat auf
`fix/search-retrieved-fields` korrigiert. Der Suchhelper übermittelt seine
Standardfeldliste bisher ausserhalb von `options`; die Engine erhält sie
dadurch nicht. Die Korrektur setzt sie in `options.attributesToRetrieve`
und erhält eine ausdrücklich vom Aufrufer angegebene Feldliste.

Die Korrektur begrenzt die Antwort auf die vorgesehenen Felder und vermeidet
unter anderem unnötige `polyline`-Daten. Sie verändert weder Treffer, Filter,
Sortierung noch Sichtbarkeitsregeln. Eine messbare Beschleunigung ist bisher
nicht nachgewiesen; die Feldauswahl ist keine Voraussetzung der korrekten
SRCH0-Suchbasis.

Gap-ID, Case-IDs und historische Beobachtungen bleiben erhalten. Die
betreffenden Diagnoseprüfungen und aktiven Solländerungen sind in der
vorhandenen Suite noch unverändert; ihre Trennung vom verbindlichen
Abnahmeumfang muss dort nachgeführt werden. Diese Einstufung behauptet
keinen grünen Gesamtlauf und hebt keine übrigen Blocker auf.

### Nicht blockierende Tag-Umbenennung

**Entscheidung vom 19. September 2026:** Die veralteten Tagnamen nach einer
globalen Umbenennung (`SRCH0-MUTATION-008`) sind kein Merge- oder
Abnahmeblocker für SRCH0. Der Befund bleibt ein Fehler: Die Datenbank enthält
den neuen Namen, während die betroffenen Suchdokumente den alten behalten;
ein Filter mit dem neuen Tagnamen kann deshalb Touren übersehen.

Die normale Wanderer-Oberfläche bietet jedoch keine globale Tag-Umbenennung
an. Benutzer können Tags anlegen sowie Touren zuordnen oder die Zuordnung
entfernen. Die Collection `tags` hat `updateRule: null`; auch über die
Records-API dürfen normale Benutzer bestehende Tags nicht umbenennen.
Ein PocketBase-Superuser kann den Namen global ändern. Der Fall prüft damit
einen administrativen Sonderfall und ist keine Voraussetzung der aktuellen
SRCH0-Suchbasis.

Die Ausnahme betrifft ausschliesslich diese Tag-Umbenennung. Reguläre
Tagfilter, das Anlegen von Tags, Änderungen der Tagzuordnung an Touren und
die zugehörigen Aktualisierungs- und Berechtigungsprüfungen bleiben
verbindlich. Andere Actor-, Kategorie- oder Metadatenbefunde werden damit
nicht neu eingestuft.

Case-ID, historische Evidenz und die gewünschte Korrektur bleiben erhalten.
Tests, Korpus und Erwartungen werden durch diese Dokumentänderung nicht
geändert; die Suite kann den Fall weiterhin als Pflichtprüfung behandeln.
Die Trennung der Diagnose von der verbindlichen Abnahmeprüfung ist dort
noch nachzuführen. Die Einstufung behauptet keinen grünen Gesamtlauf und
lässt die übrigen SRCH0-Blocker bestehen.

Die isolierte Korrektur ist auf `fix/search-tag-metadata` (`122974380`)
lokal ab `origin/dev` (`c73966d6c`) vorbereitet. Sie führt ausschliesslich
Tag-Umbenennungen in den Suchdokumenten betroffener Touren nach. Der
Regressionstest mit 201 betroffenen Touren und die gesamte Backend-Testsuite
sind erfolgreich; Push, PR und Integration in `dev` oder `feat/srch0` sind
noch nicht erfolgt. Dieser Nachweis betrifft den separaten Fix, nicht die
SRCH0-Gesamtabnahme.

### Nicht blockierende Vorschaubild-Absicherung

**Entscheidung vom 19. September 2026:** Der negative Thumbnailindex
(`SRCH0-PROJECTION-021`) ist kein Merge- oder Abnahmeblocker für SRCH0,
wird aber separat auf `fix/search-thumbnail-index` korrigiert. Der Fehler
ist nachgewiesen: Ein im tatsächlichen PocketBase-Schema gespeicherter
Datensatz mit Foto und `thumbnail: -1` verursacht bei der Suchprojektion
eine Go-Panic. Es handelt sich nicht bloss um ein falsches Vorschaubild.

Die normale Fotoauswahl erzeugt keine negativen Indizes; die JSON-API weist
sie zurück. Über den Multipart-Pfad oder direkte beziehungsweise interne
PocketBase-Schreibzugriffe kann der ungültige Wert jedoch gespeichert
werden. Ein Auslöser durch normale Bedienung der Oberfläche ist bisher
nicht nachgewiesen. Deshalb wird die Absicherung separat behandelt und
nicht zur Voraussetzung der SRCH0-Suchbasis gemacht.

Der Fix erweitert ausschliesslich die vorhandene Bereichsprüfung: Bei
einem negativen Index wird wie bei einem zu grossen Index das erste Foto
verwendet. API-Validierung, Datenbankschema und Fotoreihenfolge ändern sich
dadurch nicht. Die Ausnahme gilt nur für diesen negativen Index; andere
Projektions-, Sichtbarkeits- und Startup-Prüfungen bleiben verbindlich.

Case-ID, historische Evidenz und die Solländerung
`GO-FIX-SRCH0-PROJECTION-021` bleiben erhalten. Tests, Korpus und
Erwartungen werden durch diese Dokumentänderung nicht geändert. Die
vorhandene Suite kann den Fall weiterhin als Pflichtprüfung behandeln;
die Trennung der Diagnose von der verbindlichen Abnahme ist noch
nachzuführen. Diese Einstufung behauptet keinen grünen Gesamtlauf und
hebt keine übrigen SRCH0-Blocker auf.

Die isolierte Korrektur ist auf `fix/search-thumbnail-index` (`e6861358b`)
lokal direkt ab `origin/dev` (`c73966d6c`) vorbereitet. Der gezielte Test
reproduziert vor der Korrektur die Go-Panic bei `-1`. Nach der Korrektur
bestehen alle sechs Grenzfälle (negativer Index, erstes und letztes Foto,
Index gleich oder grösser als die Fotoanzahl sowie keine Fotos) und die
gesamte Backend-Testsuite. Push, PR und Integration sind noch nicht
erfolgt. Dieser Nachweis betrifft den isolierten Fix, nicht die
SRCH0-Gesamtabnahme.

### Nicht blockierende API-Fehlerstatusweitergabe

**Entscheidung vom 19. September 2026:** Die fehlerhafte Weitergabe des
HTTP-Status eines Meilisearch-SDK-Fehlers ist kein Merge- oder
Abnahmeblocker für SRCH0. Sie erhält mit `fix/search-api-error-status`
(`8bcfe61df`) einen separaten Fixbranch direkt ab `origin/dev` (`c73966d6c`).

Der SDK-Fehler stellt den Status unter `response.status` bereit; die
bisherigen Fehlerhandler lesen `httpStatus`. Dadurch wird etwa ein
Enginefehler mit HTTP 400 als HTTP 500 ausgegeben. Der Fix erhält den
tatsächlichen 4xx-/5xx-Status in den betroffenen Suchendpunkten und im
gemeinsamen Fehlerhandler. Er ändert keine Suchergebnisse oder
Berechtigungsregeln. Die Einordnung priorisiert diese Fehlerklassifikation
separat; der nachgewiesene API-Fehler bleibt dokumentiert.

Die Ausnahme betrifft `SRCH0-P-HTTP` und die Statusassertionen in
`SRCH0-SEARCH-116` sowie `SRCH0-SEARCH-124` bis `127` samt ihren
`API-FIX-…`-Solländerungen. Authentifizierung, Berechtigungsprüfung und die
Anforderung, fehlgeschlagene Engineanfragen nicht als erfolgreichen
Suchlauf auszugeben, bleiben verbindlich; andere Assertions derselben
Fälle sind nicht pauschal ausgenommen.

Historische Evidenz, Tests, Korpus und aktive Erwartungen bleiben
unverändert. Die Suite kann die Statuskorrektur weiterhin als Pflicht
prüfen; ihre technische Trennung von Diagnose und verbindlicher Abnahme
steht noch aus. Die Dokumentationsentscheidung behauptet weder einen
grünen Gesamtlauf noch die Behebung anderer SRCH0-Blocker.

Der isolierte Fix ist mit 25 neuen Regressionstests geprüft: vier
Suchrouten mit SDK-Status 400, 403, 404 und 503 sowie Transportfehlern,
unveränderte Erfolgsantworten und der gemeinsame Fehlerhandler.
Vor der Korrektur schlugen im gezielten Lauf 19 Tests fehl; danach
bestehen sämtliche 146 Webtests und `npm run check` ohne Fehler oder
Warnungen. Die Actor-Route wird in diesem Branch nicht geändert. Push,
PR und Integration sind noch nicht erfolgt; dies ist keine
SRCH0-Gesamtabnahme.

### Nicht blockierende Actor-Suchparameter

**Entscheidung vom 19. September 2026:** Die Behandlung eines fehlenden
`q` und expliziter `limit`-Werte in der Actor-Suche ist kein Merge- oder
Abnahmeblocker für SRCH0. Die Korrektur wird unabhängig vom Statusfix auf
`fix/search-actor-parameters` (`a72ff18df`) direkt ab `origin/dev` (`c73966d6c`)
vorbereitet. Die normale Oberfläche setzt `q` stets und verwendet ohne
eigenen Limitparameter das funktionierende numerische Standardlimit `3`.

Der API-Fehler bleibt real: Ein explizites `limit=7` wird bisher als String
an das SDK weitergereicht. Der Fix übergibt Limits als sichere,
nichtnegative Ganzzahlen, behält den Standardwert `3` und weist ungültige
Werte mit HTTP 400 zurück. Ein fehlendes `q` wird ebenfalls als HTTP 400
gemeldet. Am historischen Ausgangscommit `e9b7a8cad` ergibt dieser Fall
HTTP 500; auf aktuellem `dev` bereits HTTP 404, weil der gemeinsame
Fehlerhandler SvelteKit-HTTP-Fehler inzwischen erhält. Die historische
Beobachtung wird dadurch nicht umgeschrieben.

Die Ausnahme betrifft die Parameterprüfungen zu `SRCH0-COMPILER-062`
und `SRCH0-COMPILER-064` sowie die zugehörigen `WEB-FIX-…`-Solländerungen
und strikten Parameterproben. Authentifizierung, Berechtigungen,
`includeSelf` und die sonstige Actor-Suchsemantik bleiben verbindlich.
Tests, Korpus und Erwartungen bleiben unverändert; die technische
Trennung der Diagnose von der verbindlichen Abnahme steht noch aus.
Die Einstufung behauptet keinen grünen Gesamtlauf und hebt keine übrigen
SRCH0-Blocker auf.

Der isolierte Fix ändert die Actor-Route samt API-Dokumentation und
ergänzt 20 Regressionstests. Vor der Korrektur schlugen davon 14 fehl;
danach bestehen alle 20 sowie sämtliche 141 Webtests.
`npm run check` meldet keine Fehler oder Warnungen. Der Branch enthält
keine Änderungen des separaten Statusfixes. Push, PR und Integration
sind noch nicht erfolgt; der Nachweis ersetzt keine SRCH0-Gesamtabnahme.

### Nicht blockierende Upload-Duplikatprüfung

**Entscheidung vom 19. September 2026:** Die unvollständige Duplikatprüfung
beim Trail-Upload (`SRCH0-SEARCH-117`) ist kein Merge- oder Abnahmeblocker
für SRCH0. Sie wird als eigenständiger Importfix auf
`fix/upload-duplicate-check` (`8f50fe9ac`) direkt ab `origin/dev` (`c73966d6c`)
korrigiert. SRCH0 führt den technischen Meilisearch-Consumer und seine
Evidenz weiter; die Integration dieses Fixes ist keine Voraussetzung
der Suchbaseline.

Die bisherige Prüfung betrachtet lediglich die erste Engineantwort mit
dem Standardlimit 20. Ein passender vorhandener Trail ausserhalb dieser
Antwort kann damit übersehen werden. Der separate Fix prüft weitere
Kandidaten mit den bisherigen Vergleichskriterien und erhält den
authentifizierten Suchscope. Der echte Engine-Regressionstest muss den
Vergleichskandidaten aus tatsächlich ausserhalb der ersten 20 Treffer
liegenden Dokumenten wählen; ein Name wie `duplicate-21` allein belegt
keine solche Position. Die Ausnahme betrifft neben `SRCH0-SEARCH-117`
auch die Nachlade- und Anforderungsform in `SRCH0-COMPILER-083/084`
samt `WEB-FIX-SRCH0-COMPILER-083/084`; Berechtigungsassertionen sind
dadurch nicht ausgenommen.

Historische Evidenz, Tests, Korpus und Solländerungen bleiben durch diese
Dokumentationsentscheidung unverändert. Die technische Trennung der
Duplikatdiagnose von der verbindlichen SRCH0-Abnahme steht noch aus.
Authentifizierung, Berechtigungen und korrekter Zugriffsscope bleiben
verbindlich. Die Einstufung erklärt weder die vorhandene Lücke für
behoben noch einen SRCH0-Gesamtlauf für grün.

Der separate Fix enthält nur den Batchhelper, die Uploadroute und deren
Tests. Von 15 neuen Uploadregressionen schlugen vor der Korrektur sieben
fehl; danach bestehen alle 15 sowie sämtliche 136 Webtests.
`npm run check` meldet keine Fehler oder Warnungen. Ein zusätzlicher
Batchhelper-Test gegen isoliertes Meilisearch 1.53.2 bestätigt die
vollständige Erfassung der zulässigen Testbestände: bei `maxTotalHits=1000`
alle 1'101 Treffer in Paketen von 500/500/101, bei `maxTotalHits=7` alle
17 Treffer in Paketen von 7/7/3, jeweils vier Anfragen einschliesslich
der abschliessenden leeren Antwort. Tenant-Token und zusätzliche Filter
bleiben wirksam; private oder anderweitig ausgeschlossene Dokumente
werden nicht geliefert. Das qualifiziert den separaten Produktfix,
nicht die SRCH0-Gesamtabnahme oder ein neues Engineprofil des Korpus.
Push, PR und Integration sind noch nicht erfolgt.

### Nicht blockierende Clusterbegrenzung

**Entscheidung vom 19. September 2026:** Die begrenzte Clustergrundlage
der aktuellen Karte (`SRCH0-SEARCH-129`) ist kein Merge- oder
Abnahmeblocker für SRCH0. Das bestehende Karten-Cap bleibt als bekannte
Begrenzung akzeptiert; oberhalb dieses Caps besteht für den aktuellen
Pfad kein Versprechen vollständiger Cluster. Der Nachladefix aus
`fix/srch0-findings` bleibt zurückgestellt und dient nur als Referenz.
Es wird kein eigener Clusterbranch vorbereitet und keine entsprechende
Produktänderung, kein neues UI-Signal und keine Änderung von
`maxTotalHits` übernommen. Die Ausnahme umfasst die Vollmengen- und
Nachladeanforderung aus `SRCH0-SEARCH-129` sowie `SRCH0-COMPILER-072`
und `WEB-FIX-SRCH0-COMPILER-072`; die Prüfung des Zugriffsscopes und
der Nutzerfilter bleibt verbindlich.

Der Test indexiert 10'001 Trails. Bei `maxTotalHits=1000` liefert die
Clusterabfrage trotz `limit: 10000` höchstens 1'000 Treffer. Das ist die
Wirkung einer konfigurierbaren Enginebegrenzung mit Standardwert 1'000,
kein Meilisearch-Fehler. Der vorgeschlagene Vollscan in Paketen von 500
mit wachsenden ID-Ausschlüssen verursacht zusätzliche Anfragen und
Filteraufwand; er ist keine kostenneutrale Voraussetzung der Suchbaseline.

`SRCH0-SEARCH-130` prüft dagegen die gewöhnliche Listenpagination: Bei
Seitengrösse 100 ist Seite 11 am selben Engine-Cap leer. Der Fall ist
kein Clusterfix-Nachweis und wird nicht als solcher umgedeutet. Diese
Entscheidung ändert weder die Listenpagination noch die späteren
Vollständigkeitsanforderungen des V1-Cursorvertrags.

Evidenz und Tests der Clusterbegrenzung bleiben erhalten. Die strikte
Testsuite kann weiterhin Vollständigkeit oberhalb des Caps verlangen;
die technische Diagnose-/Abnahmetrennung ist noch nachzuführen.
Zugriffsscope, Sichtbarkeit und Berechtigungen der gelieferten Treffer
bleiben verbindlich. Die Begrenzung ist keine allgemeine Ausnahme für
ACL-Fehler, andere Suchpfade oder erfolgreiche Antworten auf Enginefehler.
Die Einstufung behauptet keinen grünen SRCH0-Gesamtlauf.

### Weitergehende Folgearbeiten

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
| Upload-Duplikat | Tenant-Scope, leeres `q`, Offset 0, Defaultlimit 20, acht gelesene Felder; Diagnose eines Duplikats ausserhalb der ersten 20 Treffer |
| Projektion | Trail-, Listen- und Actordokumente samt Relationsauflösung |
| Mutationen | Create, Update, Delete, Sichtbarkeit, Shares, Likes, Actor, Tags und Taxonomie |
| Startup | vorhandene, fehlende und leere Indizes; Settings-, Delete- und Add-Taskfolge; Request während jeder Zwischenstufe |
| Negative Proxyfälle | unbekannter Index, Feld, Sortkey und freie Retrievaloption als nicht zugesagte Eingaben |
| Browser | URL, Storage, Snapshot, Navigation, Liste, Karte und History |

Jeder produktive First-Party-Consumer muss mindestens einem Fall zugeordnet
sein. Ein Consumer ohne Fixture ist ein Manifestfehler, nicht implizit durch
die allgemeine Suchmatrix abgedeckt.

Die Pflicht zur Inventarisierung verlangt keine Korrektur der oben
ausgenommenen Diagnosefälle vor SRCH0-Abnahme. Insbesondere bleiben
Upload-Duplikatprüfung und Cluster-Cap erfasst, ohne dadurch erneut zu
Merge- oder Abnahmeblockern zu werden.

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

Die oben abgegrenzten nicht blockierenden Diagnosefälle, einschliesslich
Upload-Duplikatprüfung und Cluster-Cap, sind dabei von den verbindlichen
Abnahmeprüfungen zu trennen; diese technische Umstellung steht noch aus.

Verletzungen im verbindlichen Abnahmeumfang schlagen immer fehl.
`knownViolation`, erwartetes Fehlschlagen,
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
6. alle nachgewiesenen Fehler im verbindlichen Abnahmeumfang eine begründete
   aktive Korrektur und einen grünen unabhängigen Regressionstest besitzen;
7. historische Basisfälle und aktive Solländerungen getrennt validiert sind;
8. ein absichtlich verändertes Golden ohne Evidenzänderung von CI abgewiesen
   wird;
9. eine entfernte Case-Datei nicht als verringerte Mindestabdeckung grün wird;
10. der komplette Aufbau ausschliesslich synthetische Daten und für
    Datenbankproben die tatsächlichen Produktionsregeln verwendet;
11. der Testaufbau weder eine reale Betreiberinstanz noch deren Daten benötigt;
12. Startup-, API-, Engine- und Browsertests einschließlich der betroffenen
    Bestandsreparaturen am tatsächlich integrierten Zielstand grün sind; und
13. kein bekannter Fehler im verbindlichen Abnahmeumfang durch eine Ausnahme
    oder ausschliesslich durch den Vergleich mit einem ebenfalls fehlerhaften
    Golden freigegeben wird.

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
Behebung desselben Fehlers bleiben in einem PR. Die zurückgestellte
Sortier-Robustheit `SRCH0-GAP-SORT-001` wird vorerst nicht ausgekoppelt und
blockiert SRCH0 nicht. Die ebenfalls nicht blockierende Feldauswahl
`SRCH0-GAP-DTO-001` ist dagegen auf einem eigenen lokalen Fixbranch
vorbereitet: `fix/search-retrieved-fields` (`8211e5598`), direkt ab
`origin/dev` bei `c73966d6c`, ohne Push oder PR und noch ohne Integration
in `dev` oder `feat/srch0`. Gezielte Requestproben bestätigen die
Standardliste mit 32 Feldern sowie eigene und explizit leere Feldlisten;
alle 121 Webtests und `npm run check` sind ohne Fehler oder Warnungen
erfolgreich. Dies belegt den isolierten Fix, keine SRCH0-Gesamtabnahme.

Auch die nicht blockierende globale Tag-Umbenennung `SRCH0-MUTATION-008`
besitzt mit `fix/search-tag-metadata` (`122974380`) einen separaten lokalen
Fixbranch ab `origin/dev` (`c73966d6c`), ohne Push, PR oder Integration.
Die [Tag-Einstufung](#nicht-blockierende-tag-umbenennung) beschreibt den
administrativen Sonderfall, den begrenzten Fix und die Prüfungsnachweise.

Die ebenfalls nicht blockierende Vorschaubild-Absicherung
`SRCH0-PROJECTION-021` erhält mit `fix/search-thumbnail-index` einen
eigenen lokalen Fixbranch (`e6861358b`) direkt ab `origin/dev`
(`c73966d6c`), ohne Push, PR oder Integration. Sechs Grenzfälle und die
gesamte Backend-Testsuite bestehen. Die [Vorschaubild-Einstufung](#nicht-blockierende-vorschaubild-absicherung)
beschreibt den nachgewiesenen Absturz, die begrenzte Ausnahme und den
isolierten Prüfungsnachweis.

Die nicht blockierenden API-Befunde erhalten zwei unabhängige lokale
Fixbranches direkt ab `origin/dev` (`c73966d6c`):
`fix/search-api-error-status` (`8bcfe61df`) für die SDK-HTTP-Statusweitergabe und
`fix/search-actor-parameters` (`a72ff18df`) für fehlendes `q` und numerische Limits.
Push, PR und Integration sind noch nicht erfolgt. Die oben dokumentierten
Ausnahmen gelten nur für diese Korrekturen, nicht für den gesamten
API-Prüfungsumfang.

Die Upload-Duplikatprüfung wird als Importfix unabhängig auf
`fix/upload-duplicate-check` (`8f50fe9ac`) direkt ab `origin/dev` (`c73966d6c`)
vorbereitet, ohne Push, PR oder Integration. Die
[Duplikat-Einstufung](#nicht-blockierende-upload-duplikatprüfung) erklärt
den nicht blockierenden Status und die isolierten Prüfungsnachweise
(136 Webtests, fehlerfreier Check und echte Meilisearch-Batchprüfung). Für die
[Clusterbegrenzung](#nicht-blockierende-clusterbegrenzung) bleibt die
Sammelkorrektur zurückgestellt; es gibt dafür keinen separaten Fixbranch
und keine übernommene Produktänderung.

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
| 2026-09-19 | `SRCH0-GAP-SORT-001` wird als Robustheitsverbesserung zurückgestellt und ist kein SRCH0-Blocker; vorerst kein eigener PR | ungültige Storagewerte sind ein defensiver Testfall ohne nachgewiesenen Fehler im gewöhnlichen Gebrauch; Diagnosefälle bleiben erhalten, die Suite muss ihre nicht blockierende Einordnung noch übernehmen |
| 2026-09-19 | `SRCH0-GAP-DTO-001` ist kein SRCH0-Blocker; separate lokale Korrektur auf `fix/search-retrieved-fields` | die vorgesehene Feldauswahl reduziert unnötige Antwortdaten ohne Änderung der Suchergebnisse; Diagnosefälle bleiben erhalten, die Suite muss ihre nicht blockierende Einordnung noch übernehmen |
| 2026-09-19 | Die globale Tag-Umbenennung `SRCH0-MUTATION-008` ist kein SRCH0-Blocker; separater lokaler Fix auf `fix/search-tag-metadata` (`122974380`) bleibt vorbereitet | Umbenennung ist nur als administrativer Sonderfall möglich, nicht in der normalen UI oder über die Records-API normaler Benutzer; reguläre Tagfilter und Zuordnungsänderungen bleiben verbindlich, die Diagnose-/Abnahmetrennung in der Suite steht noch aus |
| 2026-09-19 | Schutz vor unvollständigen neuen Indexdokumenten gehört zur jeweiligen Metadaten-Korrektur, kein separates Arbeitspaket oder zusätzlicher SRCH0-Blocker | Die Tests prüfen gezielt fehlende Dokumente als Ausgangslage der neuen Teilaktualisierung; der Tag-Fix enthält die Absicherung bereits. Tests und Korpus bleiben unverändert; andere Indexfehler sind nicht ausgenommen. |
| 2026-09-19 | Negative Thumbnailindizes `SRCH0-PROJECTION-021` sind kein SRCH0-Blocker; separate lokale Korrektur auf `fix/search-thumbnail-index` (`e6861358b`) | Der Absturz ist mit einem gespeicherten Datensatz und Foto nachgewiesen, normale Fotoauswahl und JSON-API lassen negative Werte jedoch nicht zu. Der Fix fällt auf das erste Foto zurück; Diagnosefall und Solländerung bleiben erhalten, die Diagnose-/Abnahmetrennung in der Suite steht noch aus. |
| 2026-09-19 | SDK-HTTP-Statusweitergabe ist kein SRCH0-Blocker; separater Fix auf `fix/search-api-error-status` (`8bcfe61df`) | Der tatsächliche SDK-Status ersetzt den irrtümlichen Fallback auf 500. Die Ausnahme betrifft nur die Fehlerklassifikation, nicht Authentifizierung, Berechtigungen oder als Erfolg ausgegebene Enginefehler; die Diagnose-/Abnahmetrennung bleibt offen. |
| 2026-09-19 | Fehlendes `q` und explizite Actor-Limits sind kein SRCH0-Blocker; unabhängiger Fix auf `fix/search-actor-parameters` (`a72ff18df`) | Die normale UI setzt `q` und nutzt das numerische Standardlimit 3. Explizite Limits als String und der falsche Status für fehlendes `q` bleiben nachgewiesene API-Fehler; Tests und historische Erwartungen bleiben unverändert, die Diagnose-/Abnahmetrennung steht aus. |
| 2026-09-19 | Upload-Duplikatprüfung `SRCH0-SEARCH-117` ist kein SRCH0-Blocker; separater Importfix auf `fix/upload-duplicate-check` (`8f50fe9ac`) | SRCH0 inventarisiert den technischen Meilisearch-Consumer weiter, die Importkorrektur ist keine Abnahmevoraussetzung. Evidenz und Tests bleiben erhalten; die Diagnose-/Abnahmetrennung steht aus. |
| 2026-09-19 | Cluster-Cap `SRCH0-SEARCH-129` ist kein SRCH0-Blocker; bestehende Begrenzung akzeptiert und Nachladefix zurückgestellt | Kein Vollständigkeitsversprechen oberhalb des Caps, kein eigener Clusterbranch, keine Produkt-, UI- oder `maxTotalHits`-Änderung. Die konfigurierbare Enginebegrenzung ist kein Meilisearch-Fehler; ein Vollscan ist nicht kostenneutral. `SRCH0-SEARCH-130` bleibt ein separater Listenpagination-Fall. |
