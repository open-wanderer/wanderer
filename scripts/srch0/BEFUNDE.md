# SRCH0: Befunde und Merge-Blocker

Stand: 19. September 2026. Geprüfte Produktbaseline:
`e9b7a8cade980002acbcf2e2f5b2a083934f29d2`.

Die Tests auf `feat/srch0` verlangen korrekte Ergebnisse und müssen gegen
den bisherigen Produktcode rot bleiben. Bekannte Fehler innerhalb des
verbindlichen Abnahmeumfangs sind keine erlaubten Abweichungen. Die unten
dokumentierte Sortierabsicherung ist seit dem 19. September 2026 zurückgestellt
und kein SRCH0-Blocker. Produktkorrekturen werden als einzelne fachliche Fixes
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
| HTTP-Clientfehler wird zu 500 | Ein echter `MeilisearchApiError` mit HTTP 400 hat den Status unter `response.status`. Der Proxy liest `httpStatus` und antwortet mit 500. Die Fehlerklasse geht verloren. | `SRCH0-P-HTTP` in `web/src/lib/srch0/plausibility.test.ts`; reale API-Engine-Negativfälle |
| Radius bei Nullkoordinaten fehlt | `(46,7)` erzeugt einen Radiusfilter; `(0,7)` und `(46,0)` erzeugen keinen. Gültige geografische Koordinaten werden durch Truthiness verworfen. | `SRCH0-P-GEO-ZERO`; `SRCH0-COMPILER-017/018` |
| Falsche Abstiegslimite in der Karte | Bei `max_elevation_gain=800` und `max_elevation_loss=700` ist `elevationLossMax=700`, aber `elevationLossLimit=800`. Die Achse verwendet zwei widersprüchliche Grenzen. | `SRCH0-P-LOSS-LIMIT`; echter Map-Loader |
| Feldauswahl erreicht die Engine nicht | Der allgemeine Suchhelper legt `attributesToRetrieve` ausserhalb von `options` ab. Der Proxy reicht nur `options` weiter. Die vom Helper beabsichtigte Auswahl wird ignoriert. | `SRCH0-P-RETRIEVAL`; echter Requestbuilder und Proxycode |
| Negativer Thumbnailindex verursacht Panic | Das echte Produktionsschema akzeptiert `thumbnail=-1` bei vorhandenem, tatsächlich hochgeladenem JPEG. Der Projektor prüft nur die obere Grenze und greift auf `photos[-1]` zu. Ein speicherbarer Record bringt die Projektion zum Absturz. | `SRCH0-PROJECTION-021`; `db/srch0_projection_test.go` |
| Löschen eines Shares entfernt weitere Freigaben aus dem Suchindex | Alice und Bob haben je einen Share auf denselben Trail. Nach Löschen von Alices Share existiert Bobs Datenbankfreigabe weiterhin; der Hook schreibt dennoch `shares=[]` in den Suchindex. Bob verliert damit diesen Suchzugang. | `SRCH0-MUTATION-005`; echte Datenbankprobe und materialisierte Engineaufträge |
| Geänderte Metadaten bleiben im Trailindex alt | Änderungen an Actor, Tag oder Kategorie ändern die Quelldaten, aktualisieren aber die davon abhängigen Trail-Suchdokumente nicht entsprechend. Zusätzlich dürfen Metadatenupdates bei fehlenden Einträgen keine unvollständigen Treffer erzeugen. | `SRCH0-MUTATION-007/008/009`; echte Hooks und Datenbankproben |
| Fehlende oder ungültige Schwierigkeit wird erfunden | Go projiziert Unknown als `0`/leicht; die Web-Konvertierung weist sonstige Rohwerte teilweise als schwierig aus. Unknown muss ohne erfundene Stufe erhalten bleiben. | strikte Go-Projektions- und Web-DTO-Prüfungen |
| Ungültige Actor-Suchparameter werden falsch behandelt | Fehlendes `q` wird zu HTTP 500; `limit` erreicht den SDK-Auftrag als String. | strikte Actor-Parameterprüfungen |

Die Produktfixes gehören auf ihre separaten Produktbranches.
SRCH0 enthält die strikten Regressionstests und die korrekten aktiven
Erwartungen. Eine Änderung dieser Erwartungen darf keinen fachlichen
Propertytest umgehen. Die [Anleitung](README.md#eine-produktkorrektur-prüfen)
beschreibt den Vergleich mit der historischen Beobachtung.

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
rot werden. Ausschliesslich Fehler der genannten Sortierabsicherung gelten
fachlich als Diagnose ausserhalb der SRCH0-Abnahme. Die technische Trennung
von Diagnose und Abnahme muss vor der formalen Gesamtabnahme nachgeführt
werden; ein grüner Lauf wird hier nicht behauptet. Gemischte Fälle wie
`SRCH0-STATE-007` erhalten keine pauschale Ausnahme für andere Eigenschaften.

Gültige Sortierung bleibt verbindlich, insbesondere die neun Sortierfelder
in beiden Richtungen (`SRCH0-COMPILER-027` bis `044`) und die numerischen
Sortiereigenschaften. Textrelevanz und Gleichstände sind von dieser
Entscheidung nicht betroffen. Alle übrigen SRCH0-Blocker bleiben bestehen.

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
