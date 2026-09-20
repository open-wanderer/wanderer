# SRCH0: Befunde und Merge-Blocker

Stand: 19. September 2026. Geprüfte Produktbaseline:
`e9b7a8cade980002acbcf2e2f5b2a083934f29d2`.

Die Tests auf `feat/srch0` verlangen korrekte Ergebnisse und müssen gegen
den bisherigen Produktcode rot bleiben. Bekannte Fehler innerhalb des
verbindlichen Abnahmeumfangs sind keine erlaubten Abweichungen. Die unten
dokumentierten Befunde zur Sortierabsicherung, Feldauswahl, administrativen
Tag-Umbenennung, Absicherung negativer Thumbnailindizes, Weitergabe des
API-Fehlerstatus, Validierung der Actor-Suchparameter, Vollständigkeit der
Upload-Duplikatprüfung und Vervollständigung der Clustergrundlage sind seit dem
19. September 2026 keine SRCH0-Blocker.
Produktkorrekturen werden als einzelne fachliche Fixes
mit ihren Regressionstests für separate PRs vorbereitet. `fix/srch0-findings` bleibt die
Sammelreferenz für die bisherigen Korrekturen. `fix/search-index-startup`
behandelt weiterhin das gesamte Startup-Paket einschliesslich Indexerhalt,
synchroner Initialisierung, Fehlerweitergabe, Wiederaufnahme und
Reparaturbefehl. Erst nach Integration aller erforderlichen Korrekturen und
erfolgreichem Lauf des tatsächlichen SRCH0-Branchstands ist SRCH0 mergebar.
Historische Goldens dürfen diesen Nachweis nicht ersetzen. Der bisherige grüne
kombinierte Prüfbaum enthielt die Findings-Korrekturen und das Startup-Paket;
sein Ergebnis gilt nicht für den Korrekturbranch ohne Startup-Paket.

## Lieferstand der Radiuskorrektur

Als erster einzelner Fix liegt die Radiuskorrektur auf
`fix/search-radius-filter`, Commit `398b45682`, frisch ab `origin/dev`
(`c73966d6c`). Sie erhält gültige Nullkoordinaten, prüft die Koordinatengrenzen
und einen endlichen positiven Radius und erzeugt genau eine `_geoRadius`-Klausel.
Der Stand vom 19. September 2026 ist ausschliesslich lokal: nicht gepusht, kein
PR erstellt und weder in `dev` noch in `feat/srch0` integriert.

Auf dem Radiusbranch sind alle 24 gezielten Regressionstests und alle 145
Web-Unit-Tests erfolgreich; `npm run check` meldet 0 Fehler und 0 Warnungen.
Diese Prüfung betrifft den einzelnen Fix und ersetzt keine SRCH0-Gesamtabnahme.
SRCH0 bleibt blockiert. Seine historischen Beobachtungen und aktiven
Erwartungen bleiben unverändert; die Radiusbefunde sind im geprüften
SRCH0-Produktstand weiterhin offen.

## Bestätigte Fehler im verbindlichen Abnahmeumfang

| Befund | Reproduktion und Bedeutung | Nachweis |
| --- | --- | --- |
| Radius bei Nullkoordinaten fehlt | `(46,7)` erzeugt einen Radiusfilter; `(0,7)` und `(46,0)` erzeugen keinen. Gültige geografische Koordinaten werden durch Truthiness verworfen. | `SRCH0-P-GEO-ZERO`; `SRCH0-COMPILER-017/018` |
| Falsche Abstiegslimite in der Karte | Bei `max_elevation_gain=800` und `max_elevation_loss=700` ist `elevationLossMax=700`, aber `elevationLossLimit=800`. Die Achse verwendet zwei widersprüchliche Grenzen. | `SRCH0-P-LOSS-LIMIT`; echter Map-Loader |
| Löschen eines Shares entfernt weitere Freigaben aus dem Suchindex | Alice und Bob haben je einen Share auf denselben Trail. Nach Löschen von Alices Share existiert Bobs Datenbankfreigabe weiterhin; der Hook schreibt dennoch `shares=[]` in den Suchindex. Bob verliert damit diesen Suchzugang. | `SRCH0-MUTATION-005`; echte Datenbankprobe und materialisierte Engineaufträge |
| Geänderte Actor- und Kategoriemetadaten bleiben im Trailindex alt | Änderungen an Actor oder Kategorie ändern die Quelldaten, aktualisieren aber die davon abhängigen Trail-Suchdokumente nicht entsprechend. Die Entscheidung zur Tag-Umbenennung nimmt diese Befunde nicht von der Abnahme aus. | `SRCH0-MUTATION-007/009`; echte Hooks und Datenbankproben |
| Fehlende oder ungültige Schwierigkeit wird erfunden | Go projiziert Unknown als `0`/leicht; die Web-Konvertierung weist sonstige Rohwerte teilweise als schwierig aus. Unknown muss ohne erfundene Stufe erhalten bleiben. | strikte Go-Projektions- und Web-DTO-Prüfungen |

Die Produktfixes gehören auf ihre separaten Produktbranches.
SRCH0 enthält die strikten Regressionstests und die korrekten aktiven
Erwartungen. Eine Änderung dieser Erwartungen darf keinen fachlichen
Propertytest umgehen. Die [Anleitung](README.md#eine-produktkorrektur-prüfen)
beschreibt den Vergleich mit der historischen Beobachtung.

## Kein Blocker: Upload-Duplikatprüfung

Entscheidung vom 19. September 2026: Die Vollständigkeit der
Upload-Duplikatprüfung ist **kein SRCH0-Merge- oder Abnahmeblocker**.
Die Korrektur gehört fachlich zum Upload und wird unabhängig auf
`fix/upload-duplicate-check` vorbereitet. Die Duplikatprüfung bleibt als
Meilisearch-Consumer im SRCH0-Inventar und in der Evidenz enthalten.

Der bisherige Uploadhelper prüft nur die erste Engineantwort mit höchstens
20 Trails. Eine kontrollierte Probe legt das passende Duplikat hinter diese
erste Antwort. Der separate Fix lässt die Engine passende Kandidaten
auswählen. Nachweis und aktive Solländerungen bleiben erhalten: `SRCH0-SEARCH-117` mit
`API-FIX-SRCH0-SEARCH-117`, `SRCH0-COMPILER-083/084` mit ihren `WEB-FIX-`
Änderungen und der Duplikattest in `web/src/lib/srch0/completeness.test.ts`.
Die Ausnahme betrifft diese Vollständigkeitskorrektur, keine
Authentifizierung oder Zugriffsregeln; insbesondere ist der Tenant-Negativfall
`SRCH0-SEARCH-118` davon nicht ausgenommen.

**Reviewrevision vom 20. September 2026:** Die Uploadroute stellt genau
eine Suchanfrage mit `_geoRadius(lat, lon, 100)` und offenen Wertebereichen
für `distance`, `elevation_gain` und `elevation_loss`: jeweils strikt grösser
als der Uploadwert minus 50 und strikt kleiner als der Uploadwert plus 50.
`limit: 1` genügt, weil die Engine bereits die passenden Kandidaten auswählt.
Abgerufen werden nur `id`, `name`, `author_name` und `domain`. Der bisherige
Batchhelper entfällt; es gibt weder Paging noch wachsende `NOT IN`-Listen
oder einen Vollscan im Uploadcode. Der Tenant-Client und das erzwungene
Hochladen über `ignoreDuplicates` bleiben unverändert.

Die Geo-Regel bedeutet den 100-m-Radius der Engine, einschliesslich seines
Rands. Meilisearch 1.36.0 vergleicht `_geo`-Punkte mit `<=` und verwendet eine
auf Millimeter gerundete Haversine-Distanz. Das ist nicht bitgleich zur
bisherigen JavaScript-Prüfung `< 100`: Auch unmittelbar am Rand und durch
Rundung knapp darüber liegende Punkte können passen. Die drei numerischen
Toleranzen bleiben strikt; ±50 selbst ist ausgeschlossen. Dies ist keine
Garantie gegen parallele Imports oder verzögert aktualisierte Suchindizes.
Die [Filterimplementierung von Meilisearch 1.36.0](https://github.com/meilisearch/meilisearch/blob/v1.36.0/crates/milli/src/search/facet/filter.rs#L685-L692)
und die [verwendete Distanzfunktion](https://docs.rs/crate/geoutils/0.5.1/source/src/formula.rs)
belegen diese Geo-Randsemantik.

Der Branch basiert auf dem damaligen `origin/dev` bei `c73966d6c`;
die Reviewrevision `d0ede4520` ist ein Folgecommit. Er bleibt lokal, ohne
Push, PR oder Integration in `dev` oder `feat/srch0`. Mit aktivierter Engineintegration bestehen sämtliche 140
Webtests, darunter
8 Upload-Unitfälle und 11 Tests gegen eine isolierte Meilisearch-Instanz
1.53.2. Die echte Uploadroute findet bei `maxTotalHits=1000` passende Treffer
neben 20'000 unpassenden und 11 weiteren Testdokumenten mit genau einer
Anfrage und `limit: 1`. Der Nachweis verwendet einen echten Tenant-Token und
prüft private Treffer, Nullkoordinaten, alle sechs strikten ±50-Metrikgrenzen,
einen weit entfernten Start und den inklusiven 100-m-Rand. Der bestandene 140er-Lauf umfasst 129 reguläre Tests und die
11 nur
bei aktivierter Engineintegration ausgeführten Fälle. `npm run check` meldet keine Fehler oder Warnungen. Dieser Lauf
mit 1.53.2 ist vom obigen Quellbeleg für 1.36.0 getrennt; er ersetzt weder
einen vollständigen Importablauf noch die SRCH0-Gesamtabnahme.
Die früheren 15 Uploadtests und die Batchprobe gegen Meilisearch 1.53.2
qualifizierten `8f50fe9ac`, nicht den nun ersetzten Anfrageansatz.
Clusterabfragen werden durch den isolierten Uploadfix nicht verändert.

Historische Evidenz, SRCH0-Korpus, aktive Sollwerte und strikte Tests
bleiben in dieser Dokumentationsrevision unverändert. Die bisherigen
Status-Goldens zur direkten SDK-Weitergabe und die Batch-/Nachlade-Goldens
bilden die revidierte Policy noch nicht ab und müssen gezielt nachgeführt
werden, ohne historische Beobachtungen umzuschreiben. Die technische
Trennung von Diagnose und verbindlicher Abnahme bleibt ebenfalls offen.
Ein grüner SRCH0-Gesamtlauf wird nicht behauptet; die bisherigen
Nicht-Blocker-Einstufungen und alle übrigen Anforderungen bleiben bestehen.

## Kein Blocker: begrenzte Clustergrundlage

Entscheidung vom 19. September 2026: Die Vervollständigung der Clustergrundlage
ist **kein SRCH0-Merge- oder Abnahmeblocker**. Die bestehende Begrenzung bleibt
als bekannte Grenze akzeptiert. Der Generator erzeugt tatsächlich 10'001
Trails; bei `maxTotalHits=1000` liefert die bisherige Clusterabfrage nur
1'000 davon. Daraus folgt keine Zusage, dass die Clusterantwort alle
sichtbaren Touren oder deren Gesamtzahl vollständig repräsentiert.

Das Nachladen aus dem Sammelfix ist zurückgestellt. Es wird kein separater
Cluster-Fixbranch vorbereitet; Clusterabfrage und Oberfläche bleiben
unverändert, einschliesslich des bisherigen fehlenden Begrenzungshinweises.
Das diskutierte Nachladen in 500er-Paketen verursacht zusätzliche
Engineanfragen und überträgt weitere Treffer. Eine Laufzeit- oder
Performancewirkung ist ohne Benchmarks nicht gemessen.

Die Evidenz bleibt erhalten: `SRCH0-SEARCH-129` mit
`API-FIX-SRCH0-SEARCH-129`, `SRCH0-COMPILER-072` mit
`WEB-FIX-SRCH0-COMPILER-072` und der Clustertest in
`web/src/lib/srch0/completeness.test.ts`. Die vollständige Darstellung aller
10'001 Touren ist keine SRCH0-Abnahmevoraussetzung. `SRCH0-SEARCH-130`
prüft dagegen die **Listenpagination** jenseits der Enginegrenze, keine
Clusteränderung. Diese separate Grenzprobe bleibt unverändert und erhält
durch diese Entscheidung keine allgemeine Ausnahme für Listenfehler.
Authentifizierung, Zugriffsregeln, Sichtbarkeit und andere Assertions
gemischter Fälle bleiben verbindlich.

SRCH0-Tests, Korpus und aktive Erwartungen bleiben unverändert; insbesondere
werden die bisherigen Solländerungen zum Nachladen nicht stillschweigend
entfernt. Ihre technische Einordnung als Diagnose ausserhalb der Abnahme
steht vor der formalen Gesamtabnahme noch aus. Ein grüner SRCH0-Lauf wird
hier nicht behauptet.

## Kein Blocker: API-Fehlerstatus

Entscheidung vom 19. September 2026: Die Weitergabe des Meilisearch-Fehlerstatus
ist **kein SRCH0-Merge- oder Abnahmeblocker**. Ein echter
`MeilisearchApiError` mit HTTP 400 stellt seinen Status unter `response.status`
bereit. Der bisherige Zugriff auf `httpStatus` führt im Proxy zu HTTP 500.

**Reviewrevision vom 20. September 2026:** Die Engine ist eine interne
Abhängigkeit. Nur ein echter `MeilisearchApiError` mit `response.status = 400`
und `code = invalid_search_filter` wird als HTTP 400 weitergegeben. Andere
Meilisearch-API-Fehler, einschliesslich Engine-403/404/503, sowie Transport-
und Timeoutfehler ergeben HTTP 502 mit der generischen Meldung
`Search service unavailable`; ihre internen Engineangaben werden nicht
weitergereicht. Der erlaubte Filterfehler mit HTTP 400 behält seine Details.
Unerwartete lokale
Fehler bleiben HTTP 500; eigene SvelteKit-HTTP-Fehler und die bestehende
PocketBase-Fehlerbehandlung bleiben erhalten. Der Tenant-Client wird nicht
geändert. Erfolgreiche Suchantworten und Zugriffsregeln bleiben unverändert.

Die Evidenz bleibt erhalten: `SRCH0-P-HTTP` in
`web/src/lib/srch0/plausibility.test.ts` sowie die API-Statusprüfungen in
`SRCH0-SEARCH-116` und `SRCH0-SEARCH-124` bis `127` mit ihren
`API-FIX-`-Solländerungen. Die Ausnahme betrifft ausschliesslich die falsche
Weitergabe des Fehlerstatus. Authentifizierung, Berechtigungen, Sichtbarkeit
und die tatsächliche Ablehnung unberechtigter Anfragen bleiben verbindlich;
gemischte Negativfälle erhalten keine pauschale Ausnahme.

Der Branch basiert auf dem damaligen `origin/dev` bei `c73966d6c`;
die Reviewrevision `23d6b0204` ist ein Folgecommit. Er bleibt lokal, ohne
Push, PR oder Integration in `dev` oder `feat/srch0`. Die 80 gezielten Tests und sämtliche 194 Webtests bestehen;
`npm run check` meldet keine Fehler oder Warnungen.
Die früheren 25 Regressionstests und 146 Webtests qualifizierten den
vorherigen Stand `8bcfe61df`, nicht die revidierte Gateway-Policy.

Historische Evidenz, SRCH0-Korpus, aktive Sollwerte und strikte Tests
bleiben in dieser Dokumentationsrevision unverändert. Die bisherigen
Status-Goldens zur direkten SDK-Weitergabe und die Batch-/Nachlade-Goldens
bilden die revidierte Policy noch nicht ab und müssen gezielt nachgeführt
werden, ohne historische Beobachtungen umzuschreiben. Die technische
Trennung von Diagnose und verbindlicher Abnahme bleibt ebenfalls offen.
Ein grüner SRCH0-Gesamtlauf wird nicht behauptet; die bisherigen
Nicht-Blocker-Einstufungen und alle übrigen Anforderungen bleiben bestehen.

## Kein Blocker: Actor-Suchparameter

Entscheidung vom 19. September 2026: Die Validierung der Actor-Suchparameter
ist **kein SRCH0-Merge- oder Abnahmeblocker**. Fehlendes `q` ergibt in der
historischen Baseline `e9b7a8cad` HTTP 500. Auf dem aktuellen `dev`
(`c73966d6c`) bleibt dank `isHttpError` bereits HTTP 404 erhalten; fachlich
vorgesehen ist HTTP 400. Ein übergebenes `limit` erreicht den SDK-Auftrag
bisher als String. Die normale Oberfläche übergibt `q` und kein eigenes
`limit`; eine Störung dieser regulären Aufrufe ist nicht nachgewiesen.

Der unabhängige Fix `fix/search-actor-parameters` liefert bei fehlendem `q`
HTTP 400, übergibt gültige Limits als Zahlen und weist ungültige Limits mit
HTTP 400 zurück. Der Standardwert bleibt `3`. Die Evidenz in
`SRCH0-COMPILER-062/064`, ihren `WEB-FIX-`-Solländerungen und den strikten
Actor-Parameterprüfungen in `web/src/lib/srch0/plausibility.test.ts` bleibt
erhalten. Die Ausnahme betrifft nur diese Parametervalidierung und
Typumwandlung; Authentifizierung, Berechtigungen und Sichtbarkeit bleiben
verbindlich.

Commit `a72ff18df` ist frisch ab `origin/dev` (`c73966d6c`) lokal vorbereitet,
ohne Push oder PR und ohne Integration in `dev` oder `feat/srch0`. Er benötigt
den separaten Fehlerstatus-Fix nicht. Auf `dev` scheitern 14 der 20 neuen
Regressionstests; mit dem Fix bestehen alle 20 und die gesamte Web-Testsuite
mit 141 Tests. `npm run check` meldet 0 Fehler und 0 Warnungen. Dies ersetzt
keine SRCH0-Gesamtabnahme. SRCH0-Tests, Korpus und aktive Erwartungen bleiben
unverändert; die technische Trennung von Diagnose und Abnahme steht vor der
formalen Gesamtabnahme noch aus. Ein grüner SRCH0-Lauf wird hier nicht behauptet.

## Kein Blocker: negativer Thumbnailindex

Entscheidung vom 19. September 2026: Die Absicherung negativer
Thumbnailindizes wird als Robustheitsverbesserung eingeordnet und ist
**kein SRCH0-Merge- oder Abnahmeblocker**. Sie wird separat auf
`fix/search-thumbnail-index` korrigiert. Die normale Fotoauswahl erzeugt
nichtnegative Indizes; auch die JSON-API validiert sie entsprechend.
Multipart-Anfragen, die PocketBase-Records-API oder interne Schreibzugriffe
können dagegen negative Werte speichern. Der Fehler ist nachgewiesen und
betrifft mehr als die Bildanzeige: Das echte Produktionsschema akzeptiert
`thumbnail=-1` bei vorhandenem, tatsächlich hochgeladenem JPEG. Der Projektor
prüft nur die obere Grenze, greift auf `photos[-1]` zu und bricht mit einer
Panic ab.

Der Fix fällt bei einem negativen Index auf das erste Foto zurück. Commit
`e6861358b` ist frisch ab `origin/dev` (`c73966d6c`) lokal vorbereitet, ohne
Push oder PR und ohne Integration in `dev` oder `feat/srch0`. Eingabevalidierung,
PocketBase-Schema und Fotoanordnung in der Oberfläche gehören nicht zu diesem
Fix. Andere Projektions-, Zugriffs- und Startup-Prüfungen bleiben verbindlich.

Die gezielte Regression reproduziert auf `dev` die Panic bei `-1`. Mit dem
Fix bestehen alle sechs Grenzfälle (negativer Index, erstes und letztes Foto,
Index gleich und grösser als die Fotoanzahl sowie keine Fotos) und die gesamte
Backend-Testsuite. Diese Einzelprüfung ersetzt keine SRCH0-Gesamtabnahme.

Die Evidenz bleibt erhalten: `SRCH0-PROJECTION-021`, die aktive Solländerung
`GO-FIX-SRCH0-PROJECTION-021` und die Prüfung in
`db/srch0_projection_test.go`. Tests, Korpus und aktive Erwartungen bleiben
unverändert; die strikten Prüfungen können daher weiterhin rot werden.
Ihre technische Trennung von Diagnose und Abnahme steht vor der formalen
Gesamtabnahme noch aus. Ein grüner SRCH0-Lauf wird hier nicht behauptet.

## Kein Blocker: administrative Tag-Umbenennung

Entscheidung vom 19. September 2026: Der veraltete Tag-Name nach einer globalen
Umbenennung ist **kein SRCH0-Merge- oder Abnahmeblocker**. Die normale
Wanderer-Oberfläche erlaubt das Anlegen und Zuordnen von Tags sowie das
Entfernen einer Zuordnung, aber keine globale Umbenennung. Reguläre Benutzer
können Tags auch über die API nicht umbenennen (`tags.updateRule: null`).
Eine Umbenennung durch einen PocketBase-Superuser in der Administration ist
möglich; der Befund betrifft somit einen administrativen Sonderfall.

Die Ausnahme betrifft ausschliesslich die Aktualität des Tag-Namens in
`SRCH0-MUTATION-008` und der aktiven Solländerung
`GO-FIX-SRCH0-MUTATION-008`. Sie nimmt weder Actor- und Kategoriemetadaten noch
andere Assertions desselben Falls pauschal aus. Reguläre Tagfilter, das
Aktualisieren der Tag-Zuordnung einer Tour und Zugriffsprüfungen bleiben
verbindlich. Der Schutz vor unvollständigen Suchdokumenten gehört zur unten
beschriebenen Implementierungsanforderung an neue Metadatenupdates.

Der separate Fix auf `fix/search-tag-metadata`, Commit `122974380`, ist frisch
ab `origin/dev` (`c73966d6c`) lokal vorbereitet, ohne Push oder PR und ohne
Integration in `dev` oder `feat/srch0`. Er aktualisiert ausschliesslich die
Tags betroffener Touren nach Tag-Umbenennungen. Seine Regressionstests,
einschliesslich einer Umbenennung mit 201 betroffenen Touren, und die gesamte
Backend-Testsuite sind erfolgreich. Diese Einzelprüfung ersetzt keine
SRCH0-Gesamtabnahme.

Tests, Korpus und aktive Erwartungen bleiben unverändert. Die strikten
Tag-Aktualitätsprüfungen können deshalb weiterhin rot werden; ihre technische
Trennung von Diagnose und Abnahme steht vor der formalen Gesamtabnahme noch
aus. Ein grüner SRCH0-Lauf wird hier nicht behauptet.

## Bestandteil der Metadatenkorrektur: fehlende Indexdokumente

Entscheidung vom 19. September 2026: „Fehlende Indexdokumente“ ist kein
eigenständiger nachgewiesener `dev`-Fehler und kein zusätzlicher SRCH0-Blocker.
Die Tests stellen den fehlenden Indexeintrag gezielt her; sie belegen keine
Häufigkeit im Alltag. Neue Metadaten-Teilupdates müssen in diesem Zustand
vollständige Suchdokumente herstellen oder die Indexaktualisierung mit einem
Fehler abbrechen, wenn die nötigen Daten fehlen. Sie dürfen keine
unvollständigen Treffer erzeugen. Diese Absicherung gehört in denselben Fix;
ein eigener Branch, PR oder Arbeitspunkt ist nicht vorgesehen. Der Tag-Fix
`fix/search-tag-metadata` (`122974380`) enthält die vollständige Neuerstellung
bereits, geprüft durch `TestTagRenameRebuildsMissingSearchDocument`. Ohne
Lieferung dieses optionalen Tag-Fixes entsteht daraus keine zusätzliche
SRCH0-Abnahmevoraussetzung. Andere Indexpfade, etwa bei Startup, Freigaben
oder Likes, werden damit weder allgemein repariert noch von ihren Prüfungen
ausgenommen.
Tests, Korpus und aktive Erwartungen bleiben unverändert; auch diese
Zuordnung der Implementierungsprüfungen ist vor der formalen Gesamtabnahme
technisch nachzuführen.

## Kein Blocker: abgerufene Suchfelder

Entscheidung vom 19. September 2026 zu `SRCH0-GAP-DTO-001`: Die ignorierte
Feldauswahl ist **kein SRCH0-Blocker**, wird aber separat auf
`fix/search-retrieved-fields` korrigiert, Commit `8211e5598`, frisch ab
`origin/dev` (`c73966d6c`). Der Branch ist ausschliesslich lokal vorbereitet,
ohne Push oder PR; die
Korrektur ist noch nicht in `dev` oder `feat/srch0` integriert.

Der allgemeine Suchhelper legt `attributesToRetrieve` ausserhalb von `options`
ab; der Proxy reicht nur `options` weiter. Die Korrektur übermittelt die
vorgesehene Standardauswahl innerhalb von `options` und erhält ausdrücklich
gewählte Felder des Aufrufers. Dadurch entfallen unnötige Antwortfelder wie
`polyline`. Treffer, Filter, Sortierung und Zugriffsregeln bleiben unverändert;
eine messbare Beschleunigung ist bisher nicht nachgewiesen.

Die gezielte Requestprobe bestätigt die 32 Standardfelder, eine eigene
Auswahl `['id']` und die ausdrücklich leere Auswahl `[]`. Auf dem Fixbranch
sind alle 121 Web-Unit-Tests erfolgreich; `npm run check` meldet 0 Fehler und
0 Warnungen. Diese Einzelprüfung ersetzt keine SRCH0-Gesamtabnahme.

Die Evidenz bleibt erhalten: `SRCH0-P-RETRIEVAL` in
`web/src/lib/srch0/plausibility.test.ts`, `SRCH0-COMPILER-049` und dessen aktive
Solländerung `WEB-FIX-SRCH0-COMPILER-049`. Für diese Feldauswahl gilt dieselbe
noch ausstehende technische Trennung von Diagnose und Abnahme wie für die
folgende Sortierabsicherung.

## Zurückgestellt: Absicherung gespeicherter Sortwerte

Entscheidung vom 19. September 2026 zu `SRCH0-GAP-SORT-001`: Die Absicherung ungültiger gespeicherter
Sortierfelder und -richtungen sowie die Übernahme der Ansichtsvorgabe bei
fehlender oder ungültiger Richtung sind kleine Robustheitsverbesserungen und
**kein SRCH0-Blocker**. Dafür wird vorerst kein eigener PR vorbereitet. Die
normale Oberfläche erzeugt gültige Sortierwerte; ein Fehler im regulären
Gebrauch ist für diesen Befund bisher nicht nachgewiesen.

Der Nachweis verwendet absichtlich ungültige Werte: `SRCH0-BROWSER-009`
setzt `sort="unknown"` und `sort_order="raw"` im Storage. Dazu gehören die
aktive Solländerung `WEB-FIX-SRCH0-BROWSER-009` und der Plausibilitätstest
„verwendet für ungültige Sortwerte gültige Vorgaben“ in
`web/src/lib/srch0/plausibility.test.ts`. Diese Evidenz bleibt erhalten.

Tests, Korpus und Sollwerte werden durch diese Dokumentationsentscheidung
nicht geändert; die bisherigen strikten Assertions können deshalb weiterhin
rot werden. Ausschliesslich Fehler der genannten Sortierabsicherung, der
oben beschriebenen Feldauswahl, der administrativen Tag-Umbenennung, der
Absicherung negativer Thumbnailindizes, der Weitergabe des API-Fehlerstatus,
der Validierung der Actor-Suchparameter, der Vollständigkeit der
Upload-Duplikatprüfung und der Vervollständigung der Clustergrundlage gelten
im jeweils abgegrenzten Umfang fachlich als Diagnose ausserhalb der
SRCH0-Abnahme. Die technische Trennung
von Diagnose und Abnahme muss vor der formalen Gesamtabnahme nachgeführt
werden; ein grüner Lauf wird hier nicht behauptet. Gemischte Fälle wie
`SRCH0-STATE-007` erhalten keine pauschale Ausnahme für andere Eigenschaften.

Gültige Sortierung bleibt verbindlich, insbesondere die neun Sortierfelder
in beiden Richtungen (`SRCH0-COMPILER-027` bis `044`) und die numerischen
Sortiereigenschaften. Textrelevanz und Gleichstände sind von diesen
Entscheidungen nicht betroffen. Alle übrigen SRCH0-Blocker bleiben bestehen.

## Weitere geprüfte Eigenschaften und Grenzen

| Einordnung | Beobachtung | Aussagegrenze |
| --- | --- | --- |
| Listenpagination an der Enginegrenze | Bei `maxTotalHits=1000` und 100 Treffern pro Seite bleibt Seite 11 leer, obwohl das Dataset 10'001 Trails enthält. | `SRCH0-SEARCH-130` ist eine unveränderte Listen-Grenzprobe, kein Clusterfall und kein Nachweis einer Clusterkorrektur. |
| Merge-Blocker: Startup-Verfügbarkeit | Normale Starts löschen und befüllen die live verwendeten Indizes asynchron neu. Tokenroute und Suche können währenddessen erreichbar sein. Das gesamte Startup-Paket wird im eigenen PR auf `fix/search-index-startup` behandelt. | Strikte Tests bleiben in SRCH0 und verlangen den Erhalt vorhandener Daten, terminal erfolgreiche Initialisierung vor Suchbereitschaft, Fehlerweitergabe und Wiederaufnahme abgebrochener Erstinitialisierung. Fälle `SRCH0-MUTATION-090` bis `093`. Die Auslagerung erlaubt keine Fehlerausnahme. |
| Redundanz | Derselbe `_geoRadius` wird zweimal per AND verknüpft. | Logisch dieselbe Treffermenge; kein belegter Ergebnisfehler allein durch die Wiederholung. `SRCH0-P-GEO-DUPLICATE`, `SRCH0-COMPILER-016`. |
| Merge-Blocker: Enddatum | Ein Enddatum `2026-09-07` wird bisher zu einer inklusiven Grenze am Tagesbeginn. Der lokale Mittag liegt dahinter. | Das ganze lokale Kalenderdatum muss eingeschlossen sein, auch an 23-/25-Stunden-Tagen. Diese Zielsemantik war bereits im Spec festgelegt. `SRCH0-P-DATE-END` und strikte Kalender-/DST-Prüfungen. |
| Ranking ist keine globale numerische Sortierung | Bei `q=Alice` und `distance:asc` steht ein Treffer mit Autorname „Alice Aurora“ und Distanz 4'242 vor einem Treffer mit Alice im Trailnamen und Distanz 36. | Die konfigurierte Attributrelevanz kommt vor der Sortierregel. Kein Enginefehler; eine UI-Zusage rein aufsteigender Distanz wäre damit nicht erfüllt. `SRCH0-SEARCH-131`. |
| Föderierte Listenprojektion | Ein unvollständiger Remote-Listenrecord liest Aggregate live aus der Origininstanz. | Abhängigkeit von fremder Verfügbarkeit und fremden Aggregaten; der Fall allein beweist keinen unzulässigen Datenabfluss. |

## Unabhängig geprüfte Eigenschaften

Die Engineprüfungen leiten zulässige Treffer aus Public-/Autor-/Sharebeziehungen
im synthetischen Quelldataset ab. Sie kopieren dafür weder die erwarteten
Treffer noch den Tenant-Filterstring. Geprüft werden:

- Sichtbarkeit jedes identifizierbaren Treffers und eine zur zulässigen Menge
  passende obere Countgrenze;
- eindeutige IDs, passende Seitenkapazität und zum Trefferumfang passende Counts;
- unveränderte abgerufene Quelldaten, mit eng begrenzter Toleranz für
  Gleitkomma-Rundläufe;
- monotone numerische Sortierung bei leerem Suchtext; und
- Ergebnismengen, die beim Verengen eines Bereichs nicht wachsen.

Die ursprüngliche Enginequalifikation erfüllt diese allgemeinen Eigenschaften.
Die fachlichen Blocker werden zusätzlich gegen den aktuellen Produktcode
geprüft und sind dadurch nicht freigegeben.
Die Access-Matrix umfasst drei Principals und zehn konkrete Kontexte. Das
ersetzt keine Prüfung beliebiger ACL-Ausdrücke, gleichzeitiger Mutationen,
aller Föderationszustände oder bösartiger Freitexteingaben.

Die Web-Proben formulieren die gewünschte Eigenschaft direkt. Jeder Verstoss
schlägt fehl; der frühere `knownViolation`-Ausweg ist entfernt. Go prüft die
fachlichen Eigenschaften ebenfalls vor dem Goldenvergleich. Die ursprüngliche
Beobachtung bleibt historische Evidenz und kann keinen Fehler freigeben.

## Korrigierte Annahmen der ersten Testfassung

Die erste Go-Testdatenbank ersetzte das Produktionsschema durch vereinfachte
Collections. Damit waren einige angeblich speicherbare Ausgangsdaten fiktiv.
Die neue Fassung führt die echten Migrationen aus:

- `users` bleibt eine Auth-Collection. IDs, Relationen und Pflichtfelder werden
  tatsächlich validiert.
- Der Text `unknown` ist als Difficulty nicht speicherbar. Fehlende oder leere
  Difficulty ist dagegen zulässig. Die unbekannte Ausprägung bleibt ein klar
  gekennzeichneter defensiver Rohdaten-/Indexfall.
- `thumbnail` ist auch im Produktionsschema numerisch. Die Kritik am
  vereinfachten Schema war berechtigt; die Zahl als Feldtyp war jedoch kein
  Testartefakt. Erst die Prüfung seiner tatsächlichen Untergrenze zeigte den
  neuen Panic-Fehler.
- Ungültige Mutationen werden als Ablehnung erfasst. Ein zusätzlicher gültiger
  Updatefall `SRCH0-MUTATION-017` prüft die reguläre Core-/Sichtbarkeitsänderung.
- Subcategory-Iconänderungen sind kein automatisch belegter Fehler: Das Icon
  kommt in dieser Form gar nicht im Trail-Suchdokument vor.

Auch der SDK-Befund wurde präzisiert: Für die installierte Version ist
`response.status` der relevante Zugriff, nicht ein direktes `status` am Fehler.

## Reproduzieren und bewerten

```sh
make srch0-check
make srch0-unit
make srch0-engine
make srch0-browser
```

`web/test-results/srch0-plausibility.json` enthält die direkten
Web-Proben. Die Engineberichte enthalten profilgebundene Ergebnisse und die
tatsächlichen Datasetgrössen. Go meldet Datenbankvalidierung, Mutation und
Projektionsdiagnosen bei den jeweiligen Fällen. Alle Berichte sind zusammen
mit ihrem Manifest- und Änderungsdigest zu bewerten.

Die Untergrenzen und das handgepflegte Inventar verhindern Lücken im
deklarierten Umfang. Sie ersetzen weder das Lesen eines neuen Falls noch eine
fachliche Begründung. Weitere Fehler können ausserhalb der geprüften
Eigenschaften weiterhin vorhanden sein.
