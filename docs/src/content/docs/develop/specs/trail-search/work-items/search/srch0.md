---
title: SRCH0 — Suchbestand und Testkorpus
description: Reproduzierbare Bestandsaufnahme, unabhängige Plausibilitätsprüfungen und gezielte Solländerungen für die Suche.
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
  lastReviewed: '2026-09-20'
---

## Zweck und Aussagekraft

SRCH0 beschreibt das Suchverhalten am Commit
`e9b7a8cade980002acbcf2e2f5b2a083934f29d2` vom 7. September 2026. Der Branch
`feat/srch0` setzt direkt auf diesem aktualisierten `origin/dev` auf. Der
UI-Prototyp und die übrigen Spezifikationen des früheren Spec-Branches sind
nicht Bestandteil dieser Implementierung.

SRCH0 darf bei fachlich falschen geprüften Ergebnissen nicht grün sein. Die
Tests werden auf `feat/srch0` verschärft. Produktkorrekturen werden als einzelne
fachliche Fixes mit ihren Regressionstests für separate PRs vorbereitet.
`fix/srch0-findings` bleibt die Sammelreferenz für die bisherigen Korrekturen.
`fix/search-index-startup` behandelt weiterhin das gesamte Startup-Paket:
Erhalt bestehender Indizes, synchrone Initialisierung vor Suchbereitschaft,
Fehlerweitergabe, Wiederaufnahme und Reparaturbefehl.

Die Absicherung ungültiger gespeicherter Sortierfelder und -richtungen sowie
die Vorgaberichtung bei fehlenden oder ungültigen Angaben sind seit der
Entscheidung vom 19. September 2026 **kein SRCH0-Blocker**. Diese kleine
Robustheitsverbesserung wird zurückgestellt; vorerst ist kein eigener PR
vorgesehen. Ein Fehler im normalen Gebrauch ist dafür bisher nicht
nachgewiesen. Gültige Sortierung und alle übrigen Blocker bleiben verbindlich.

Auch die ignorierte Feldauswahl (`SRCH0-GAP-DTO-001`) ist seit dieser
Entscheidung **kein SRCH0-Blocker**. Die separate Korrektur auf
`fix/search-retrieved-fields`, frisch ab `origin/dev` (`c73966d6c`), ist lokal
ohne Push oder PR vorbereitet und ist noch nicht in `dev` oder `feat/srch0`
integriert. Sie reicht die Standardauswahl als `options.attributesToRetrieve`
weiter und erhält ausdrücklich gewählte Felder des Aufrufers. Dies begrenzt
Antwortfelder wie `polyline`; Treffer, Filter, Sortierung und Zugriffsregeln
bleiben unverändert. Eine messbare Beschleunigung ist nicht nachgewiesen.
Commit und Einzelprüfungen stehen in `scripts/srch0/BEFUNDE.md` unter
„Kein Blocker: abgerufene Suchfelder“.

Ein veralteter Tag-Name nach einer globalen Umbenennung ist ebenfalls
**kein SRCH0-Merge- oder Abnahmeblocker**. Die normale Oberfläche erlaubt das
Anlegen und Zuordnen von Tags sowie das Entfernen einer Zuordnung, aber keine
Umbenennung. Reguläre Benutzer können Tags auch über die API nicht umbenennen
(`tags.updateRule: null`); die Administration als PocketBase-Superuser kann
dies dagegen. Der Befund betrifft einen administrativen Sonderfall.

Die Ausnahme gilt ausschliesslich für die Aktualität des Tag-Namens in
`SRCH0-MUTATION-008` und `GO-FIX-SRCH0-MUTATION-008`. Actor- und
Kategoriemetadaten sowie andere Assertions gemischter Fälle sind nicht
ausgenommen. Reguläre Tagfilter, Änderungen der Tag-Zuordnung einer Tour
und Zugriffsprüfungen bleiben verbindlich.
Der separate Fix `fix/search-tag-metadata`, Commit `122974380`,
ist frisch ab `origin/dev` (`c73966d6c`) lokal ohne Push oder PR vorbereitet
und noch nicht in `dev` oder `feat/srch0` integriert. Umfang und erfolgreiche
Einzelprüfungen stehen in `scripts/srch0/BEFUNDE.md` unter
„Kein Blocker: administrative Tag-Umbenennung“.

Die Absicherung negativer Thumbnailindizes ist ebenfalls **kein SRCH0-Merge-
oder Abnahmeblocker**. Der Befund `SRCH0-PROJECTION-021` und seine aktive
Solländerung `GO-FIX-SRCH0-PROJECTION-021` bleiben erhalten: Das reale
PocketBase-Schema erlaubt einen negativen Index bei vorhandenem Foto; der
Projektor greift damit ausserhalb des Arrays zu und bricht mit einer Panic
ab. Die normale Fotoauswahl und JSON-API erzeugen oder erlauben diesen Wert
nicht; Multipart-, PocketBase- und interne Schreibpfade können ihn dagegen
speichern. Der separate Fix `fix/search-thumbnail-index`, Commit `e6861358b`,
frisch ab `origin/dev` (`c73966d6c`), fällt bei einem negativen Index auf das erste Foto
zurück. Er ist lokal ohne Push oder PR vorbereitet und noch nicht in `dev`
oder `feat/srch0` integriert. Er ändert weder Eingabevalidierung noch Schema
oder Fotoanordnung. Andere Projektions-, Zugriffs- und Startup-Prüfungen
bleiben verbindlich. Sechs Grenzfälle und die gesamte Backend-Testsuite sind
erfolgreich; dies ersetzt keine SRCH0-Gesamtabnahme. Details stehen in
`scripts/srch0/BEFUNDE.md` unter
„Kein Blocker: negativer Thumbnailindex“.

Die Weitergabe des API-Fehlerstatus und die Validierung der Actor-Suchparameter
sind ebenfalls **keine SRCH0-Merge- oder Abnahmeblocker**. Die unabhängigen
Branches `fix/search-api-error-status` (`23d6b0204`) und
`fix/search-actor-parameters` (`a72ff18df`) sind
auf der damaligen `origin/dev`-Basis `c73966d6c` lokal ohne Push oder PR
vorbereitet und noch nicht in `dev` oder `feat/srch0` integriert. Die
API-Reviewrevision ist ein Folgecommit. Nach der Reviewrevision
vom 20. September erhält der erste Fix nur einen echten SDK-400 mit
`invalid_search_filter` als HTTP 400. Andere Engine-, Transport- und
Timeoutfehler werden generisches HTTP 502; unerwartete lokale Fehler bleiben
500. Eigene HTTP- und PocketBase-Fehlerbehandlung bleiben erhalten. Der zweite
liefert bei fehlendem `q` HTTP 400, übergibt gültige Limits als Zahlen und
weist ungültige Limits mit HTTP 400 zurück; der Standard bleibt `3`.
Fehlendes `q` ergab in der historischen Baseline HTTP 500, auf aktuellem
`dev` dank `isHttpError` bereits HTTP 404. Die normale Oberfläche übergibt
`q` und kein eigenes `limit`; eine Störung dieser regulären Aufrufe ist nicht
nachgewiesen. Die Ausnahmen betreffen nur diese Statusweitergabe und
Parameterbehandlung. Authentifizierung, Berechtigungen, Sichtbarkeit und die
Ablehnung unberechtigter Anfragen bleiben verbindlich. Evidenz, erfolgreiche
Einzelprüfungen und abgegrenzte SRCH0-Prüfungen stehen in
`scripts/srch0/BEFUNDE.md` unter „Kein Blocker:
API-Fehlerstatus“ und „Kein Blocker: Actor-Suchparameter“.

Die Vollständigkeit der Upload-Duplikatprüfung und die Vervollständigung der
Clustergrundlage sind ebenfalls **keine SRCH0-Merge- oder Abnahmeblocker**.
Der separate Fix `fix/upload-duplicate-check` basiert auf dem damaligen
`origin/dev` bei `c73966d6c`; `d0ede4520` ist der Reviewfolgecommit. Der Branch
ist lokal ohne Push oder PR vorbereitet und noch nicht in `dev` oder
`feat/srch0` integriert. Die Korrektur gehört fachlich zum Upload;
dessen Meilisearch-Nutzung bleibt im SRCH0-Inventar und in der Evidenz.
Seit der Reviewrevision vom 20. September wird genau eine Suchanfrage mit
`_geoRadius(lat, lon, 100)`, strikt offenen ±50-Bereichen für Distanz, Auf-
und Abstieg sowie `limit: 1` gestellt. Abgerufen werden nur `id`, `name`,
`author_name` und `domain`; der Batchhelper samt Paging und `NOT IN` entfällt.
Der inklusive, millimetergerundete Engine-Rand ersetzt bewusst das bisherige
JS-`< 100`. Tenant-Client und erzwungener Upload bleiben unverändert.
Die Status-/Batch-Goldens der unveränderten Suite müssen gezielt nachgeführt
werden; die historische Evidenz und Nicht-Blocker-Einstufungen bleiben erhalten.
Sie verändert keine Clusterabfrage. Für Cluster ist die bestehende
Begrenzung als bekannte Grenze akzeptiert. Das Nachladen des Sammelfixes
bleibt zurückgestellt, ohne separaten Clusterbranch oder Änderung an
Clusterabfrage und Oberfläche. Bei 10'001 Touren und `maxTotalHits=1000`
werden nur 1'000 repräsentiert; vollständige Abdeckung ist nicht zugesagt
und kein SRCH0-Pflichtnachweis. `SRCH0-SEARCH-129` betrifft
Cluster, `SRCH0-SEARCH-130` dagegen die unveränderte Listenpagination an der
Enginegrenze. Die Entscheidung gibt keine allgemeine Ausnahme für
Listenfehler, Authentifizierung, Zugriffsregeln oder Sichtbarkeit. Evidenz,
erfolgreiche Einzelprüfungen und Umfang stehen in `scripts/srch0/BEFUNDE.md`
unter „Kein Blocker: Upload-Duplikatprüfung“ und „Kein Blocker: begrenzte
Clustergrundlage“.

„Fehlende Indexdokumente“ ist kein separat nachgewiesener `dev`-Fehler und
kein zusätzlicher SRCH0-Blocker, sondern eine Implementierungsanforderung an
neue Metadaten-Teilupdates. Die Tests stellen den fehlenden Zustand gezielt
her. Werden solche Updates geliefert, müssen sie vollständige Suchdokumente
herstellen oder bei unzureichenden Daten die Indexaktualisierung mit einem
Fehler abbrechen. Die Absicherung gehört in denselben Fix und
erhält keinen eigenen Branch, PR oder Arbeitspunkt. Der Tag-Fix enthält sie
bereits (`TestTagRenameRebuildsMissingSearchDocument`). Ohne diesen optionalen
Fix entsteht daraus keine zusätzliche SRCH0-Abnahmevoraussetzung. Andere
Indexfehler bleiben unverändert eingeordnet. Tests, Korpus und Erwartungen
bleiben unverändert; die technische Zuordnung dieser Implementierungsprüfungen
ist vor der Gesamtabnahme nachzuführen. Details stehen in
`scripts/srch0/BEFUNDE.md` unter „Bestandteil der Metadatenkorrektur: fehlende
Indexdokumente“.

Betroffen sind insbesondere `SRCH0-BROWSER-009`, dessen aktive Solländerung
`WEB-FIX-SRCH0-BROWSER-009` und der Plausibilitätstest „verwendet für ungültige
Sortwerte gültige Vorgaben“ sowie für die Feldauswahl `SRCH0-P-RETRIEVAL`,
`SRCH0-COMPILER-049` und `WEB-FIX-SRCH0-COMPILER-049`.
Historische Evidenz, aktive Erwartungen und
strikte Tests bleiben unverändert; sie können deshalb weiterhin rot werden.
Nur Fehler dieser Sortierabsicherung, Feldauswahl, administrativen
Tag-Umbenennung, Absicherung negativer Thumbnailindizes, Weitergabe des
API-Fehlerstatus, Validierung der Actor-Suchparameter, Vollständigkeit der
Upload-Duplikatprüfung und Vervollständigung der Clustergrundlage sind im jeweils
beschriebenen Umfang fachlich als Diagnose ausserhalb
der Abnahme zu werten. Die technische Trennung von Diagnose und Abnahme ist
vor der formalen Gesamtabnahme nachzuführen; ein grüner Lauf wird hier nicht
behauptet. Gemischte Fälle erhalten keine Ausnahme für andere Eigenschaften.
Details stehen in `scripts/srch0/BEFUNDE.md` unter „Zurückgestellt: Absicherung
gespeicherter Sortwerte“, „Kein Blocker: abgerufene Suchfelder“,
„Kein Blocker: administrative Tag-Umbenennung“,
„Kein Blocker: negativer Thumbnailindex“, „Kein Blocker: API-Fehlerstatus“,
„Kein Blocker: Actor-Suchparameter“, „Kein Blocker: Upload-Duplikatprüfung“ und
„Kein Blocker: begrenzte Clustergrundlage“.

SRCH0 bleibt bis zur Integration aller übrigen erforderlichen Korrekturen und
zur erfolgreichen Prüfung seines tatsächlichen Branchstands blockiert. Solange
dieser Produktstand abnahmerelevante Fehler enthält, müssen seine betroffenen
Tests rot bleiben. Ein kombinierter Testlauf dient der Fixprüfung und macht den
ungefixten SRCH0-Stand nicht mergebar. Der bisherige grüne kombinierte Lauf
enthielt auch das Startup-Paket und belegt keinen grünen SRCH0-Lauf ohne diese
Änderungen.

Stand vom 19. September 2026: Die Radiuskorrektur liegt als erster einzelner Fix
auf `fix/search-radius-filter`, Commit `398b45682`, frisch ab `origin/dev`
(`c73966d6c`). Sie behandelt Nullkoordinaten, gültige Koordinaten und Radien
sowie die doppelte Radiusklausel. Alle 24 Radius-Regressionstests und alle 145
Web-Unit-Tests sind erfolgreich; `npm run check` meldet 0 Fehler und 0 Warnungen.
Der Branch ist lokal und ungepusht, es gibt keinen PR und keine Integration in
`dev` oder `feat/srch0`. Diese Einzelprüfung ersetzt keine SRCH0-Gesamtabnahme.
Historische Beobachtungen und aktive Erwartungen bleiben unverändert. Details
stehen im Abschnitt „Lieferstand der Radiuskorrektur“ in
`scripts/srch0/BEFUNDE.md`.

Der Korpus unterscheidet drei Dinge:

1. **Historische Beobachtung:** Was erzeugte die Baseline bei einer konkreten
   Eingabe? Dieser Wert steht im unveränderten Basisfall unter `observed`.
2. **Aktive Solländerung:** Welche vollständige Beobachtung verlangt eine spätere
   Produktkorrektur? Sie steht in `changes.json` und ersetzt im Test die
   historische Erwartung, gebunden an deren Basisdigest.
3. **Plausibilitätsprüfung:** Erfüllt das Ergebnis eine unabhängig formulierte
   Eigenschaft, etwa Sichtbarkeit, gültige Koordinaten oder eine konsistente
   Sortierung? Diese Prüfung liest keine erwartete Treffermenge aus `observed`.

Jede Verletzung einer geprüften Eigenschaft lässt den Test scheitern. Innerhalb
des verbindlichen Abnahmeumfangs gibt es keine Ausnahme für bekannte Verstösse.
Historische Beobachtungen sind weder eine fachliche Freigabe noch ein Ersatz
für korrekte aktive Erwartungen.

SRCH0 führt keine neuen Suchfunktionen und keine Korrektur der gefundenen
Produktfehler ein. Die einzige produktive Testnaht erlaubt es, den bereits
asynchronen Startup-Auftrag im Test zuverlässig abzuwarten; die normale
Ausführung bleibt asynchron.

## Umfang und Testgrenzen

Der gemeinsame JSON-Korpus liegt unter `testdata/trail-search/srch0/v1`.

| Familie | Inhalt | Ausführung |
| --- | --- | --- |
| `state` | Listen- und Kartendefaults, rohe URL-/Storagewerte, Präzedenz, Sanitizing | echte SvelteKit-Loader und Filterfunktionen in Vitest |
| `projection` | Trail-, Listen- und Actordokumente, Relationsauflösung, defensive Eingaben | echte Go-Projektoren und reale PocketBase-Migrationen |
| `compiler` | Filtertexte, Sortierung, DTOs und bekannte API-Aufträge | echte Requestbuilder und Handler in Vitest |
| `search` | Volltext, ACL, Filter, Sortierung, Paging und Nebenconsumer | Meilisearch 1.11.3 und 1.36.0; repräsentative Fälle zusätzlich über echte API-Handler |
| `mutation` | Create, Update, Delete, Shares, Likes, Relationen, Tenant-Token und Startup | echte Go-Hooks; Aufträge zusätzlich gegen beide Engines |
| `browser` | URL, Storage, Snapshot, Navigation, Liste, Karte und History | Playwright mit echter SvelteKit-App und synthetischen externen Diensten |

Browsertests prüfen den Zustand und die Navigation. Die Enginejobs prüfen
die echte Filterung. Kein Test lädt Benutzer oder Trails aus einer
Entwicklerinstanz.

Die Go-Tests verwenden das Produktionsschema einschliesslich Auth-Collection,
Feldvalidierung und Relationen. Nur drei Migrationen, die ausschliesslich eine
externe Meilisearch-Instanz umkonfigurieren, sind beim Aufbau der Testdatenbank
ausgenommen. Lesbare Fixture-IDs werden an der Datenbankgrenze deterministisch
auf gültige PocketBase-IDs abgebildet. Tatsächlich unzulässige Eingaben werden
als Ablehnung geprüft; direkte Projektorproben mit solchen Records sind
explizit defensive Tests.

Die Engine-Suite übernimmt Tenant-Suchregeln aus den erfolgreich geprüften
Go-Tokenfällen. Sie erzeugt keine zweite handgeschriebene Kopie dieser Regeln.
Eine separate Sichtbarkeitsprüfung vergleicht Treffer und Counts mit den
Beziehungen im synthetischen Quelldataset.

Startup-Evidenz verbindet zwei Tests: Go pausiert die echten HTTP-Aufträge
und verlangt, dass Suchbereitschaft erst nach erfolgreichem Abschluss
hergestellt wird. Die API-Suite prüft den vollständigen Ready-Endzustand.
Eine erfolgreiche Suchantwort während eines Teilaufbaus ist kein akzeptiertes
Golden. Fehler- und Wiederanlauffälle ergänzen die normalen Starts.

## Fachliche Bewertung

Die ausführliche, reproduzierbare Bewertung steht in
`scripts/srch0/BEFUNDE.md`. Bereits bestätigte Fehler betreffen den verlorenen
HTTP-Clientstatus, Radiusfilter bei Nullkoordinaten, die Abstiegslimite der
Karte und die nicht weitergereichte Feldauswahl. Der falsche API-Fehlerstatus
und die Actor-Parameterbehandlung sind gemäss Entscheidung vom
19. September 2026 keine SRCH0-Blocker und werden unabhängig korrigiert.
Ein negatives Thumbnail mit
vorhandenem Foto ist im Produktionsschema speicherbar und bringt den
Projektor zum Absturz; seine Absicherung ist gemäss Entscheidung vom
19. September 2026 kein SRCH0-Blocker und wird separat korrigiert.

Die Clusterabfrage erreicht beim grossen Dataset die Indexgrenze von 1'000
Treffern; die Duplikatprüfung untersucht nur die erste Engineantwort mit
höchstens 20 Trails. Beide Befunde sind seit dem 19. September 2026 keine
SRCH0-Blocker. Die Duplikatprüfung wird unabhängig im Upload korrigiert;
Cluster behalten ihre bekannte Begrenzung. Die bisherigen strikten
Vollständigkeitsprüfungen bleiben als Evidenz erhalten und können weiterhin
scheitern. Ihre technische Einordnung als Diagnose steht noch aus; ein
Nachweis aller 10'001 Touren ist keine SRCH0-Abnahmevoraussetzung.

Die doppelte `_geoRadius`-Klausel ist eine Redundanz ohne belegten Unterschied
in der Treffermenge. Das Enddatum umfasst den ganzen lokalen Kalendertag;
Tests prüfen dies auch an Tagen mit Zeitumstellung. Bei nichtleerem Suchtext steht die
Attributrelevanz vor der expliziten Sortierung; ein eigener Fall demonstriert
den Konflikt mit weit auseinanderliegenden Distanzen.

Der geprüfte ACL-Ausschnitt und die Sortier-/Bereichseigenschaften sind keine
allgemeine Sicherheits- oder Korrektheitszusage für beliebige Suchausdrücke.

## Artefakte und Wartung

| Datei | Aufgabe |
| --- | --- |
| `schema.json` | geschlossene Form der expandierten Basisfälle |
| `manifest.json` | Digests der Fälle, Datasets, Profile und Metadaten |
| `inventory.json` | unabhängig gepflegte Consumer und konkrete Abdeckungspflichten |
| `sources.json` | einmalige Quellenprovenienz am Baselinecommit |
| `changes.json` | konkrete aktive Solländerungen; anfangs leer |
| `profiles/*.json` | beobachtete Settings für Trails, Listen und Actors |
| `datasets/*.json` | synthetischer Referenzbestand und Generator für 10'001 Trails |
| `cases/<family>/*.json` | thematische Gruppen mit gemeinsamen Metadaten und individuellen Fällen |
| `README.md` | kompakte Übersicht über Gruppen und Consumer |

Gruppen speichern Familie, Dataset, Kontext und Codeanker einmal. Die Fälle
enthalten ID, Eingabe, Beobachtung und Einordnung sowie nötige Abweichungen.
`loadCorpus` ergänzt Baseline und Formatversion aus dem Manifest und expandiert
die Gruppen für alle Adapter. Gruppenbytes und kanonische expandierte Fälle
sind jeweils per Digest gebunden.

`inventory.json` enthält voneinander abgegrenzte Consumer. Aliase wie
`recommendation` und `recommendations` werden nicht doppelt gezählt.
Neue Consumer ohne ausführbaren Fall lassen den Validator scheitern.
Familien benötigen mindestens einen positiven, negativen und Randfall;
diese Untergrenze steigt nicht bei jedem neuen Test automatisch an.

Konkrete Pflichten umfassen URL-/Storagefehler, Taxonomie, Datum und DST,
Ranges, Difficulty, Geo, Projektion, Mutation und Startup. Die Access-Matrix
prüft zusätzlich die tatsächlichen Anfragen für drei Principals und zehn
Sichtbarkeitskontexte. Alle neun Sortierkeys benötigen beide Richtungen in
Compiler- und Enginefällen. Tags machen die übrigen Pflichten auffindbar;
dass ein Fall sein angegebenes Merkmal wirklich prüft, bleibt Reviewaufgabe.

Die Quellenhashes dokumentieren Herkunft. `--verify-baseline` kann sie
explizit gegen den historischen Commit prüfen. Das ist keine Erkennung von
Änderungen im Arbeitsbaum. Die normalen Tests führen den aktuellen Produktcode
aus; sie verlangen weder unveränderte Quelldateien noch die ganze Git-Historie.

Die Standardmethoden sind je Adapter zentral in `scripts/srch0/README.md`
beschrieben. `evidence.method` ist ein optionaler, rein dokumentarischer Hinweis
zu Besonderheiten eines Falls. Ein Textbaustein ist weder erforderlich noch
ein zusätzlicher Nachweis; das Feld steuert keine Ausführung oder Freigabe.

## Erwartungen gezielt ändern

Eine veröffentlichte Beobachtung wird nicht überschrieben. Der SRCH0-Testbranch
verlangt die korrekte Erwartung bereits vor Integration der separat
entwickelten Produktkorrektur. Dazu ergänzt er `changes.json` mit:

- einer eindeutigen Änderungs-ID und der betroffenen `case_id`;
- `basis_digest`: SHA-256 des kanonischen expandierten Basisfalls;
- einer fachlichen Begründung und prüfbaren Evidenzdateien;
- `observed`: der vollständigen neuen Erwartung.

Pro Basisfall ist eine aktive Ersatzbeobachtung erlaubt. Weitere Korrekturen
bearbeiten denselben Eintrag und bleiben an der ursprünglichen Basis gebunden.
Unbekannte Fälle, doppelte Einträge und veraltete Digests werden abgewiesen.

Go-, Web-, Engine-, API- und Browseradapter verwenden diese Erwartung im
Speicher; der Basisfall bleibt unverändert. Ein vollständiges Beispiel steht
in `scripts/srch0/README.md`. Startup-Settings können in der Ersatzbeobachtung
über `settings_overrides` gezielt geändert werden. Die Engine-Bestandssuite
behält ihre historischen Profile; ein neues Produktprofil benötigt eigene
versionierte Qualifikationsfälle.

Der Validator prüft Form und Bindung. Er kann eine fachlich
unbegründete Solländerung nicht beurteilen. Begründung und Evidenz sind
Reviewhilfen, keine automatische Freigabe. Ein geänderter Evidenztext erlaubt
keine Änderung der historischen Beobachtung.

Neue Eingaben oder korrigierte historische Erfassungen erhalten eine neue
Case-ID. Entfallene veröffentlichte IDs bleiben unter `manifest.retired`
reserviert. Geänderte Datasets und Profile benötigen eine neue Artefaktrevision;
neue oder entfernte Fälle sowie geänderte Artefakte eine neue Manifestrevision.
Eine inkompatible Bedeutung des Fixtureformats verlangt eine neue Schemaversion.

## Baseline erneuern

Vor einem stabilen Release mit aktiven Solländerungen, spätestens jedoch bei
20 betroffenen Basisfällen, wird eine neue Korpusgeneration qualifiziert.
Ihre Beobachtungen werden am festgelegten Releasecommit frisch reproduziert;
Solländerungen dürfen nicht automatisch als neue historische Beobachtungen
übernommen werden. Die neue Generation beginnt mit leeren Solländerungen.

Die alte Generation einschliesslich `changes.json` bleibt unverändert
archiviert. Eine Zuordnung verbindet alte Case-IDs und Basisdigests mit neuen
Case-IDs und Digests; entfallene Fälle erhalten eine Begründung. Der Wechsel
setzt eine erneute Qualifikation aller sechs Familien einschliesslich API und
beider Engineprofile sowie ein Review dieser Zuordnung voraus.

Eine Korpusgeneration erneuert die historische Bezugsbasis. Das ist unabhängig
von einer inkompatiblen Schemaversion: Bei unveränderter Feldbedeutung kann die
Formatversion gleich bleiben. Die heutigen Schema-Konstanten für Commit und
Datum sowie die Runner sind jedoch an `srch0/v1` gebunden. Diese Bindungen und
die Auswahl einer weiteren Generation müssen vor einem Wechsel ausdrücklich
angepasst und geprüft werden.

## Zufall und Vertragsgrenzen

`preserve` bezeichnet einen stabilen Bestandsfall. `known_gap` kennzeichnet
einen Befund mit benannter Folgearbeit. `non_contract` beschreibt defensive
oder nicht zugesagte Eingaben und Resultatanteile.

Diese Einordnung schaltet nicht pauschal einen ganzen Test aus: Auch bei einer
zufälligen Empfehlung bleiben Sichtbarkeit, Antwortform und Kapazität
prüfbar. Unspezifizierte Identitäten und Reihenfolgen werden jedoch gar nicht
als Erwartung gespeichert. `hit_groups` erlaubt Tausch innerhalb echter
Gleichstände. Unsortierte, abgeschnittene Mengen verwenden nur Anzahl,
Dokumentform und unabhängige Eignungsprüfung.

Task-IDs, signierte Tokens, Laufzeiten, lokale Pfade und zufällige
Empfehlungsreihenfolgen gehören nicht in die Goldens. Eine unbenutzte
Versionsweiche gibt es nicht: Beide gebundenen Engines reproduzieren derzeit
dieselben Beobachtungen. Bei einer belegten künftigen Versionsabweichung muss
die Unterscheidung zuerst ausdrücklich spezifiziert und getestet werden.

## Ausführung und Abnahme

Vom Repositoryroot aus:

```sh
make srch0-check
make srch0-unit
make srch0-engine
make srch0-browser
```

Die Anleitung in `scripts/srch0/README.md` beschreibt Voraussetzungen,
Einzelläufe und Reports. Jeder Ausführungsbericht bindet das Manifest und die
aktiven Änderungen. Testbefehle schreiben keine beobachteten Werte neu.

Der SRCH0-Workflow prüft das Korpus, die strikten Go-/Web-Tests und die
veröffentlichten Fälle gegenüber der PR-Basis. Damit kann ein reiner
Strukturcheck nicht als fachliche SRCH0-Abnahme erscheinen. Die normalen Go-
und Web-Workflows führen ihre vollständigen Tests weiterhin aus. Engine-/API- und Browserjobs laufen
bei Änderungen an den relevanten Code- und Korpuspfaden. Reine Änderungen
an der allgemeinen Dokumentation starten keine Engine-Matrix. Ein separater
Docs-Workflow führt den Dokumentationsbuild aus und erkennt dabei auch
ungültiges Frontmatter.

Die Code-Pfadfilter bleiben bewusst breit, weil Suchverhalten auch durch
indirekte Abhängigkeiten beeinflusst wird. Ein Cache vorbereiteter
Testdatenbank-Vorlagen ist eine zurückgestellte optionale Laufzeitoptimierung.

Abnahme verlangt ein valides Schema und Manifest, vollständiges Inventar,
reproduzierbare Fälle auf beiden Engines, erfolgreiche Go-/Web-/Browserläufe,
getestete Änderungsregeln und eine ausdrückliche fachliche Bewertung. Ein
alter Bericht deckt kein später geändertes Manifest ab.

Bezeichnungen wie `IDX0`, `SRCH-COMP`, `SRCH2` oder `SEC-VIS-0` in
`successor_refs` sind Planungsreferenzen aus dem Suchentwurf. Ihre
Spezifikationen liegen nicht auf diesem Branch. Sie sind keine auflösbaren
Docs-Links und keine Voraussetzung, einen hier belegten Fehler zu korrigieren.

## Entscheidungen

| Datum | Entscheidung | Begründung |
| --- | --- | --- |
| 2026-09-07 | Frischer Branch direkt auf aktualisiertem `origin/dev` | SRCH0 übernimmt keinen UI-Prototyp und keine weiteren Spec-Änderungen |
| 2026-09-08 | Beobachtung, Solländerung und Plausibilität getrennt auswerten | Reproduzierbarkeit darf Fehler weder legitimieren noch ihre Korrektur blockieren |
| 2026-09-08 | Bekannte Fehler lassen SRCH0 rot und blockieren seinen Merge; Produktfixes entstehen auf einem separaten Branch | Historische Beobachtungen und Fehlerausnahmen sind keine zulässige fachliche Abnahme |
| 2026-09-08 | Das gesamte Startup-Paket erhält mit `fix/search-index-startup` einen eigenen PR neben `fix/srch0-findings`; beide blockieren SRCH0 | Der Startup-Umbau überschreitet den Umfang des Korrekturbranches. Die fachlichen SRCH0-Prüfungen bleiben unverändert verbindlich. |
| 2026-09-08 | Produktionsmigrationen, unabhängiges Inventar und kleine Adapter | Die Tests sollen reale Zustände abbilden und überprüfbar bleiben |
| 2026-09-08 | Dokumentation bleibt Deutsch | Vorgabe für diesen Branch |
| 2026-09-19 | Fachliche Korrekturen mit ihren Regressionstests einzeln für separate PRs vorbereiten; Radiusfix zuerst auf `fix/search-radius-filter` | Einzelne Fixes lassen sich unabhängig prüfen. Dies ersetzt die bisherige Aufteilung in genau zwei Produkt-PRs; `fix/srch0-findings` bleibt Sammelreferenz und Startup bleibt separat. Der Radiusfix ist nur lokal vorbereitet und noch nicht integriert; SRCH0 bleibt blockiert. |
| 2026-09-19 | Absicherung gespeicherter Sortwerte und Vorgaberichtung zurückstellen; kein SRCH0-Blocker und vorerst kein eigener PR | Der belegte Fall nutzt absichtlich ungültige Werte; ein Fehler im regulären Gebrauch ist nicht nachgewiesen. Die Entscheidung grenzt die frühere pauschale Blockerregel ein. Gültige Sortierung und übrige Blocker bleiben verbindlich; die technische Einordnung der unveränderten strikten Proben ist vor der Gesamtabnahme nachzuführen. |
| 2026-09-19 | Ignorierte Feldauswahl (`SRCH0-GAP-DTO-001`) ist kein SRCH0-Blocker; separat auf `fix/search-retrieved-fields` korrigieren | Die Korrektur begrenzt Antwortfelder, ohne Treffer, Filter, Sortierung oder Zugriffsregeln zu ändern; eine Beschleunigung ist nicht gemessen. Der Fix ist lokal ohne Push oder PR vorbereitet. Die unveränderten strikten Proben sind vor der Gesamtabnahme technisch als Diagnose einzuordnen; übrige Blocker bleiben verbindlich. |
| 2026-09-19 | Veralteter Tag-Name nach administrativer Umbenennung ist kein SRCH0-Merge- oder Abnahmeblocker; separater Fix auf `fix/search-tag-metadata` | Die normale Oberfläche bietet keine Umbenennung und die PocketBase-Records-API erlaubt sie nur Superusern. Die Ausnahme betrifft ausschliesslich die Tag-Aktualität in `SRCH0-MUTATION-008`, keine übrigen Metadaten oder Assertions gemischter Fälle. Der lokale Fix ist ungepusht und nicht integriert. Strikte Tests und Erwartungen bleiben unverändert; die technische Trennung von Diagnose und Abnahme steht aus. |
| 2026-09-19 | Fehlende Indexdokumente als Bestandteil neuer Metadatenupdates behandeln; kein eigener Fix oder zusätzlicher SRCH0-Blocker | Der fehlende Zustand wird in Tests gezielt hergestellt und belegt keinen eigenständigen `dev`-Fehler. Der Tag-Fix enthält die Absicherung bereits. Ihre Implementierungsprüfungen begründen ohne diesen optionalen Fix keine zusätzliche Abnahmevoraussetzung; die technische Zuordnung der unveränderten Prüfungen steht aus. |
| 2026-09-19 | Negative Thumbnailindizes sind kein SRCH0-Merge- oder Abnahmeblocker; separat auf `fix/search-thumbnail-index` korrigieren | Die Panic mit einem im Produktionsschema speicherbaren negativen Index bleibt belegt. Normale Fotoauswahl und JSON-API erzeugen oder erlauben diesen Wert nicht. Der lokale Fix verwendet bei negativem Index das erste Foto, ohne Eingabevalidierung oder Fotoanordnung zu ändern. Tests und Erwartungen bleiben unverändert; die technische Trennung von Diagnose und Abnahme steht aus. |
| 2026-09-19 | API-Fehlerstatus ist kein SRCH0-Merge- oder Abnahmeblocker; unabhängig auf `fix/search-api-error-status` korrigieren | Der SDK-Status unter `response.status` muss korrekt weitergereicht werden. Die Ausnahme betrifft nur dessen Weitergabe, keine Authentifizierung, Berechtigungen oder Sichtbarkeitsregeln. Evidenz, Tests und Erwartungen bleiben erhalten; die technische Einordnung als Diagnose steht aus. |
| 2026-09-19 | Actor-Suchparameter sind kein SRCH0-Merge- oder Abnahmeblocker; unabhängig auf `fix/search-actor-parameters` korrigieren | Fehlendes `q` soll HTTP 400 ergeben und `limit` als validierte Zahl übergeben werden; Standard bleibt `3`. Die normale Oberfläche sendet `q` ohne eigenes `limit`. Historische Baseline und aktuelles `dev` liefern bei fehlendem `q` unterschiedliche falsche Statuscodes. Tests und Erwartungen bleiben unverändert; die technische Einordnung als Diagnose steht aus. |
| 2026-09-19 | Upload-Duplikatprüfung ist kein SRCH0-Merge- oder Abnahmeblocker; unabhängig auf `fix/upload-duplicate-check` korrigieren | Die Erkennung hinter der ersten Engineantwort gehört fachlich zum Upload. Der Meilisearch-Consumer bleibt inventarisiert. Tests, Korpus und Erwartungen bleiben unverändert; die technische Einordnung als Diagnose steht aus. |
| 2026-09-19 | Bestehende Clusterbegrenzung beibehalten; Vervollständigung ist kein SRCH0-Merge- oder Abnahmeblocker | Das Nachladen des Sammelfixes ist zurückgestellt. Kein separater Clusterbranch und keine Änderung an Clusterabfrage oder Oberfläche. Die begrenzte Antwort verspricht keine vollständige Abdeckung; alle 10'001 Touren nachzuweisen ist keine Abnahmevoraussetzung. `SRCH0-SEARCH-129` bleibt Clusterevidenz, `130` eine separate Listen-Grenzprobe. Tests und Erwartungen bleiben unverändert; die technische Einordnung als Diagnose steht aus. |
| 2026-09-20 | API-Statusfix im Review auf eine Gateway-Policy begrenzen; nur SDK-400 mit `invalid_search_filter` bleibt 400, andere Engine-/Transport-/Timeoutfehler werden generisches 502 | Engine-403/404/503 beschreiben die interne Abhängigkeit und dürfen nicht unverändert als Status der Wanderer-Anfrage erscheinen. Eigene HTTP- und PocketBase-Fehlerbehandlung bleiben erhalten; die Nicht-Blocker-Einstufung gilt weiter. |
| 2026-09-20 | Upload-Duplikatfix verwendet eine gefilterte Meilisearch-Anfrage mit 100-m-Radius, strikt offenen ±50-Metrikbereichen und `limit: 1`; Batch-Vollscan und Helper entfallen | Die Engine wählt passende sichtbare Kandidaten aus. Der inklusive, millimetergerundete Geo-Rand ersetzt bewusst das bisherige JS-`< 100`; Tenant-Client und erzwungener Upload bleiben unverändert. Alte Status-/Batch-Goldens müssen gezielt nachgeführt werden, die SRCH0-Abnahme bleibt offen. |
