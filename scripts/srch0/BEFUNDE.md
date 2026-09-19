# SRCH0: Befunde und Merge-Blocker

Stand: 19. September 2026. Geprüfte Produktbaseline:
`e9b7a8cade980002acbcf2e2f5b2a083934f29d2`.

Die Tests auf `feat/srch0` verlangen korrekte Ergebnisse und müssen gegen
den bisherigen Produktcode rot bleiben. Bekannte Fehler innerhalb des
verbindlichen Abnahmeumfangs sind keine erlaubten Abweichungen. Die unten
dokumentierten Befunde zur Sortierabsicherung, Feldauswahl, administrativen
Tag-Umbenennung, Absicherung negativer Thumbnailindizes, Weitergabe des
API-Fehlerstatus und Validierung der Actor-Suchparameter sind seit dem
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

## Kein Blocker: API-Fehlerstatus

Entscheidung vom 19. September 2026: Die Weitergabe des Meilisearch-Fehlerstatus
ist **kein SRCH0-Merge- oder Abnahmeblocker**. Ein echter
`MeilisearchApiError` mit HTTP 400 stellt seinen Status unter `response.status`
bereit. Der bisherige Zugriff auf `httpStatus` führt im Proxy zu HTTP 500.
Die separate Korrektur auf `fix/search-api-error-status` liest den richtigen
SDK-Status in den Routen für Einzel-, Mehrfach-, Cluster- und
Bounding-Box-Suche sowie im gemeinsamen Fehlerhandler. Erfolgreiche
Suchanfragen und deren Ergebnisse werden dadurch nicht verändert.

Die Evidenz bleibt erhalten: `SRCH0-P-HTTP` in
`web/src/lib/srch0/plausibility.test.ts` sowie die API-Statusprüfungen in
`SRCH0-SEARCH-116` und `SRCH0-SEARCH-124` bis `127` mit ihren
`API-FIX-`-Solländerungen. Die Ausnahme betrifft ausschliesslich die falsche
Weitergabe des Fehlerstatus. Authentifizierung, Berechtigungen, Sichtbarkeit
und die tatsächliche Ablehnung unberechtigter Anfragen bleiben verbindlich;
gemischte Negativfälle erhalten keine pauschale Ausnahme.

Commit `8bcfe61df` ist frisch ab `origin/dev` (`c73966d6c`) lokal vorbereitet,
ohne Push oder PR und ohne Integration in `dev` oder `feat/srch0`. Die
Regression reproduziert den Fehler auf `dev`; mit dem Fix bestehen alle 25
neuen Regressionstests und alle 146 Web-Unit-Tests. `npm run check` meldet
0 Fehler und 0 Warnungen. Dies ersetzt keine SRCH0-Gesamtabnahme. SRCH0-Tests, Korpus
und aktive Erwartungen bleiben unverändert; ihre technische Einordnung als
Diagnose ausserhalb der Abnahme steht noch aus. Ein grüner SRCH0-Lauf wird
hier nicht behauptet.

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
Absicherung negativer Thumbnailindizes, der Weitergabe des API-Fehlerstatus
und der Validierung der Actor-Suchparameter gelten
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
| Merge-Blocker: unvollständige Clusterresultate | Der Generator materialisiert tatsächlich 10'001 Trails. Bei `maxTotalHits=1000` liefert die bisherige Clusterabfrage nur 1'000 davon. | Produktprüfungen verlangen die vollständige sichtbare Clustergrundlage. Ein Engine-Cap darf kein erfolgreiches unvollständiges Produktresultat ergeben. Fälle `SRCH0-SEARCH-129/130`, `completeness.test.ts`. |
| Merge-Blocker: unvollständige Duplikatprüfung | Der bisherige Uploadhelper untersucht nur die erste Engineantwort mit höchstens 20 Trails. | Eine kontrollierte Probe legt das passende Duplikat in eine spätere Antwort und verlangt dessen Erkennung. Die echte API-/Engineprüfung ergänzt diesen Nachweis. `SRCH0-SEARCH-117`, `completeness.test.ts`. |
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
