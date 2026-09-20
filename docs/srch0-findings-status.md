# SRCH0: Aufteilung der Produktkorrekturen

Stand: 20. September 2026.

`fix/srch0-findings` bleibt die Sammelreferenz der bereits implementierten
Suchkorrekturen. Der Stand `754456831` enthält den Merge von `dev`
(`c73966d6c`). Fachlich unabhängige Korrekturen werden einzeln auf frischen
`dev`-Branches vorbereitet, jeweils mit passenden Regressionstests und einem
eigenen PR. Zusammengehörige Änderungen zur Behebung desselben Fehlers
bleiben in einem PR.

Zurückgestellte Robustheitsverbesserungen ausserhalb des verbindlichen
SRCH0-Abnahmeumfangs benötigen dafür weder einen eigenen PR noch eine
Integration vor der Abnahme.

## Radiusfilter: separat vorbereitet

| Feld | Stand |
| --- | --- |
| Branch / Commit | `fix/search-radius-filter` / `398b45682` |
| Direkte Basis | `origin/dev` bei `c73966d6c` |
| Korrektur | gültige Nullkoordinaten erhalten; Koordinatengrenzen und endlichen positiven Radius prüfen; genau eine `_geoRadius`-Klausel erzeugen |
| Umfang | `web/src/lib/stores/trail_store.ts` und `web/src/lib/stores/trail_store.test.ts` |
| Gezielte Regressionen | 24 Fälle; vor der Korrektur 20 fehlgeschlagen, nach der Korrektur alle erfolgreich |
| Prüfung des Fixcommits | `npm run test:unit -- --run` im Webverzeichnis: 145 Tests erfolgreich; `npm run check`: keine Fehler oder Warnungen |
| Veröffentlichung / Integration | lokal, nicht gepusht, kein PR; weder in `dev` noch in `feat/srch0` integriert |

Die Auskopplung betrifft die SRCH0-Befunde `SRCH0-GAP-GEO-001` und
`SRCH0-GAP-GEO-002`, die Properties `SRCH0-P-GEO-DUPLICATE` und
`SRCH0-P-GEO-ZERO` sowie die Compilerfälle `SRCH0-COMPILER-016` bis `018`.
Die entsprechende Radiuskorrektur bleibt in diesem Sammelbranch enthalten;
der neue Branch erlaubt ihre unabhängige Prüfung und Integration.

## Sortiereinstellungen: zurückgestellt, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: `SRCH0-GAP-SORT-001` ist eine kleine
Robustheitsverbesserung und blockiert weder Merge noch Abnahme von SRCH0.
Dies betrifft die Validierung ungültiger gespeicherter Sortierfelder und
Sortierrichtungen sowie den Fallback bei fehlender oder ungültiger Richtung.
Die normale Oberfläche erzeugt gültige Sortierwerte; der belegte Randfall
verwendet absichtlich ungültige Browserspeicherwerte. Ein Fehler im normalen
Gebrauch ist dafür bisher nicht nachgewiesen.

Die Korrektur bleibt als Sammelreferenz in diesem Branch enthalten. Vorerst
wird dafür kein separater Produkt-PR vorbereitet. Prüfungen gültiger
Sortierungen sowie die anderen SRCH0-Blocker bleiben verbindlich.

Diese Änderung dokumentiert die Priorisierung. Produktcode, historische
Evidenz und bestehende Testimplementierung bleiben unverändert. Ein Fehler
ausschliesslich in den betreffenden Sortier-Robustheitsprüfungen ist nach
dieser Entscheidung diagnostisch und kein Abnahmehindernis; der aktuelle
Suite-Status ist damit nicht automatisch grün. Die ausführbare
Abnahmeauswertung muss diese begrenzte Einstufung berücksichtigen.

## Abgerufene Suchfelder: kein SRCH0-Blocker, separat korrigiert

Entscheidung vom 19. September 2026: `SRCH0-GAP-DTO-001` ist kein Merge- oder
Abnahmeblocker. Die Korrektur begrenzt die Antwort auf die vorgesehenen Felder
pro Treffer und vermeidet etwa die unnötige Übertragung der `polyline`.
Treffermenge, Ranking und Zugriffsregeln ändern sich dadurch nicht; eine
Beschleunigung ist bisher nicht gemessen.

Die Korrektur liegt dennoch separat auf `fix/search-retrieved-fields`, Commit
`8211e5598`, frisch ab `origin/dev` bei `c73966d6c`: Der Suchhelper setzt
`attributesToRetrieve` unter `options`; eigene Feldlisten des Aufrufers bleiben
erhalten. Lokal geprüft sind die Requests mit Standardfeldern, eigener und
leerer Feldliste, alle 121 Webtests sowie `npm run check` ohne Fehler oder
Warnungen. Kein Push, kein PR und keine Integration in `dev` oder `feat/srch0`.

Der bestehende Nachweis `SRCH0-P-RETRIEVAL` bleibt erhalten. Wie bei der
Sortierabsicherung betrifft die Nicht-Blockerentscheidung nur diesen Befund;
die noch unveränderte Testauswertung muss Diagnose und Abnahme entsprechend
trennen. Alle übrigen Blocker bleiben verbindlich.

## Tag-Umbenennung: administrativer Sonderfall, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: Veraltete Tag-Namen im Suchindex nach
einer globalen Tag-Umbenennung (`SRCH0-MUTATION-008`) sind **kein Merge- oder
Abnahmeblocker für SRCH0**. Die normale Wanderer-Oberfläche erlaubt das
Anlegen und Zuordnen von Tags sowie das Entfernen einer Zuordnung, aber keine
globale Umbenennung. Auch normale API-Benutzer dürfen bestehende Tags nicht
ändern: Die PocketBase-Collection `tags` hat weiterhin `updateRule: null`.
Eine Umbenennung ist beispielsweise als PocketBase-Superuser in der
Administration oder durch interne Backend-Schreibzugriffe möglich. Die
bisherige Einstufung als praxisrelevanter Fehler im normalen Bedienablauf
wird damit korrigiert.

Der separat vorbereitete Fix bleibt erhalten: `fix/search-tag-metadata`,
Commit `122974380`, frisch ab `origin/dev` bei `c73966d6c`. Er aktualisiert
ausschliesslich die Tag-Namen der betroffenen Touren; Autoren- und
Kategoriemetadaten sind nicht Teil dieses Fixbranches. Die Regressionen
prüfen unter anderem 201 betroffene Touren, den Erhalt anderer Suchfelder,
fehlende Indexdokumente sowie abgewiesene und zurückgerollte Änderungen.
Gezielte Regressionstests und die gesamte Backend-Testsuite sind erfolgreich.
Der Branch ist lokal, ohne Push oder PR, und nicht in `dev` oder `feat/srch0`
integriert. Seine Integration ist keine Voraussetzung für die SRCH0-Abnahme.

„Fehlende Indexdokumente“ bezeichnet dabei die Absicherung der neu eingeführten
Metadaten-Teilaktualisierung. Die Tests stellen einen fehlenden Indexeintrag
gezielt her; eine Teilaktualisierung darf daraus keinen unvollständigen
Treffer erzeugen. Dies ist kein separat nachgewiesener Bedienfehler in `dev`
und kein eigenes Arbeitspaket, eigener PR oder zusätzlicher SRCH0-Blocker.
Die Absicherung bleibt Bestandteil der jeweiligen Metadaten-Korrektur und
ihrer Regressionstests. Für Tags ist sie bereits in `122974380` enthalten;
sie erzwingt keine Integration dieses nicht blockierenden Fixes vor SRCH0.
Damit werden weder andere Indexlücken allgemein repariert noch bestehende
Startup-, Freigabe- oder andere Indexbefunde von der Abnahme ausgenommen.
Tests und Korpus bleiben unverändert; die noch ausstehende Trennung von
Diagnose und Abnahme muss diese Zuordnung ebenfalls berücksichtigen.

Diese Ausnahme betrifft nur den Befund der administrativen Tag-Umbenennung.
Tagfilter, das Zuordnen und Entfernen von Tags an Touren sowie Zugriffsregeln
bleiben verbindlich. Historische Evidenz, Korpus, aktive Sollwerte und
Testimplementierung bleiben unverändert. Die ausführbare Auswertung muss
den betreffenden Befund noch als Diagnose von der Abnahme trennen; gemischte
Prüfungen erhalten keine pauschale Ausnahme. Damit wird weder ein grüner
SRCH0-Lauf noch eine Freigabe anderer Metadatenbefunde behauptet.

## Vorschaubild: Robustheitskorrektur, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: Der negative Vorschaubildindex
(`SRCH0-PROJECTION-021`, `GO-FIX-SRCH0-PROJECTION-021`) ist **kein Merge- oder
Abnahmeblocker für SRCH0**. Die normale Fotoauswahl erzeugt keine negativen
Indizes; Formularvalidierung und JSON-API lehnen sie ab. Das PocketBase-Schema
und der Multipart-API-Pfad erlauben solche Werte jedoch. Der Nachweis mit
tatsächlich gespeichertem Datensatz und Foto bleibt gültig: `thumbnail=-1`
verursacht eine Panic bei der Erzeugung des Suchdokuments. Die Entscheidung
priorisiert diese Robustheitskorrektur; sie erklärt den Fehler nicht für
harmlos und behauptet keinen nachgewiesenen normalen UI-Auslöser.

Der isolierte Fix liegt auf `fix/search-thumbnail-index`, Commit `e6861358b`,
frisch ab `origin/dev` bei `c73966d6c`. Er ergänzt ausschliesslich die untere
Indexgrenze im Suchprojektor: Bei negativen Werten wird wie bei zu grossen
Werten das erste Foto verwendet. API-Validierung, Datenbankschema und
Foto-Reihenfolge werden nicht geändert. Der neue Regressionstest reproduziert
die Panic ohne Fix; mit Fix bestehen alle sechs Vorschaubildfälle und die
gesamte Backend-Testsuite. Lokal, ohne Push oder PR und ohne Integration in
`dev` oder `feat/srch0`; diese Integration ist keine SRCH0-Abnahmevoraussetzung.

Historische Evidenz, Korpus, aktive Sollwerte und bestehende SRCH0-Tests
bleiben unverändert. Die automatische Auswertung muss die eng begrenzte
Nicht-Blocker-Einstufung noch berücksichtigen. Andere Projektionsprüfungen,
Zugriffsregeln und Startup-Anforderungen bleiben verbindlich; ein grüner
SRCH0-Gesamtlauf wird hier nicht behauptet.

## API-Fehlerstatus: separat korrigiert, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: Die Weitergabe des HTTP-Status von
Suchfehlern ist **kein SRCH0-Merge- oder Abnahmeblocker**. Der bisherige
Proxy liest `httpStatus`, obwohl das installierte Meilisearch-SDK den Status
unter `response.status` liefert; der gemeinsame Fehlerhandler behandelt
SDK-Fehler pauschal als 500. Dadurch erscheinen beispielsweise abgelehnte
Anfragen mit Engine-Status 400 oder 403 als interne Serverfehler. Die
direkte Weitergabe aller SDK-Statuscodes im ersten Fixstand wurde im Review
durch eine gezielte Gateway-Policy ersetzt.

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

Die Ausnahme betrifft `SRCH0-P-HTTP` und ausschliesslich die Statusweitergabe
in `API-FIX-SRCH0-SEARCH-116` sowie `124` bis `127` und den zugehörigen
Suchendpunkt-/Fehlerhandlerprüfungen. Authentifizierung, Berechtigungen,
Sichtbarkeit und die Ablehnung fehlerhafter Anfragen bleiben verbindlich;
die betroffenen Fälle erhalten keine pauschale Ausnahme.

Der unabhängige Fix liegt auf `fix/search-api-error-status`, ursprünglich
abgezweigt vom damaligen `origin/dev` bei `c73966d6c`. `23d6b0204` ist der
Reviewfolgecommit. Er umfasst die
Einzel- und Mehrfachsuche, Kartencluster, Trail-Bounding-Box und den gemeinsamen
Fehlerhandler. Er benötigt die Actor-Parameterkorrektur nicht.
Die 80 gezielten Tests und sämtliche 194 Webtests bestehen;
`npm run check` meldet keine Fehler oder Warnungen.
Die früheren 25 Regressionstests und 146 Webtests qualifizierten `8bcfe61df`,
nicht die revidierte Gateway-Policy. Lokal, ohne Push, PR oder Integration.

## Actor-Suchparameter: separat korrigiert, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: Die Korrektur fehlender Suchparameter
und expliziter Trefferlimits ist **kein SRCH0-Merge- oder Abnahmeblocker**.
Die normale Oberfläche übermittelt immer `q` und kein eigenes `limit`;
der numerische Standardwert ist drei. Direkte API-Aufrufe sind betroffen:
ein gültiges `limit=7` wird bisher als Text an die Engine weitergegeben.
Die Korrektur wandelt gültige Limits in nichtnegative sichere Ganzzahlen
um und lehnt ungültige Werte mit 400 ab. Fehlendes `q` ergibt ebenfalls 400;
`q=` bleibt zulässig. Der historische SRCH0-Ausgangsstand meldete fehlendes
`q` mit 500. Aktuelles `dev` meldet seit `5058af64d` bereits 404, weil der
gemeinsame Fehlerhandler SvelteKit-HTTP-Fehler unverändert weitergibt.

Diese Ausnahme umfasst die Parameterkorrekturen von `SRCH0-COMPILER-062`
und `064` samt `WEB-FIX-SRCH0-COMPILER-062` und `064` sowie die gezielten
Actor-Parameterprüfungen. Authentifizierung, Selbst-Ausschluss,
föderierte Handle-Auflösung und andere Sucheigenschaften bleiben verbindlich.

Der unabhängige Fix liegt auf `fix/search-actor-parameters`, Commit
`a72ff18df`, frisch ab `origin/dev` bei `c73966d6c`. Er ändert ausschliesslich die Actor-Route und
deren Regressionstests; die allgemeine SDK-Fehlerweitergabe gehört zum
anderen Fix. Vor der Korrektur schlagen 14 der 20 neuen Tests fehl; danach
bestehen alle 20 und sämtliche 141 Webtests. `npm run check` meldet keine
Fehler oder Warnungen. Beide Branches sind lokal, ohne Push oder PR und ohne
Integration in `dev` oder `feat/srch0` vorbereitet.

Historische Evidenz, Korpus, aktive Sollwerte und bestehende SRCH0-Tests
bleiben unverändert. Die ausführbare Auswertung muss die beiden eng
begrenzten Nicht-Blocker-Einstufungen noch technisch berücksichtigen;
ein grüner SRCH0-Gesamtlauf wird damit nicht behauptet.

## Upload-Duplikatprüfung: separater Import-Fix, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: Die unvollständige Duplikatprüfung
beim Datei-/URL-Upload ist **kein SRCH0-Merge- oder Abnahmeblocker**. Sie
bleibt als Meilisearch-Consumer inventarisiert, ihre Korrektur gehört jedoch
fachlich zum Import und ist keine Voraussetzung der erweiterten Suche.
Der Fehler bleibt nachgewiesen: Bisher wird nur die erste Suchantwort mit
dem Standardlimit 20 geprüft. Ein passender sichtbarer Kandidat ausserhalb
dieser Antwort kann deshalb übersehen werden.

Der unabhängige Branch `fix/upload-duplicate-check` basiert auf dem damaligen
`origin/dev` bei `c73966d6c`; `d0ede4520` ist der Reviewfolgecommit.

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

`SRCH0-SEARCH-117`, `API-FIX-SRCH0-SEARCH-117`, die Traversierungsanteile
von `SRCH0-COMPILER-083/084` samt `WEB-FIX-`-Solländerungen sowie die
kontrollierte Vollständigkeitsprobe und der API-/Engine-Nachweis bleiben erhalten.
Die Ausnahme betrifft ausschliesslich die Duplikaterkennung ausserhalb
der ersten Antwort; Zugriffsregeln und andere Upload-Eigenschaften werden
nicht pauschal aus der Abnahme genommen. Der Fix ist lokal vorbereitet,
ohne Push oder PR und ohne Integration in `dev` oder `feat/srch0`.

Mit aktivierter Engineintegration bestehen sämtliche 140 Webtests, darunter
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
Die früheren 15 Uploadregressionen, 136 Webtests und die Batchprobe gegen
Meilisearch 1.53.2 qualifizierten `8f50fe9ac`, nicht den revidierten
Anfrageansatz. Der alte Vollscan bleibt in diesem Sammelbranch nur als
Implementierungsreferenz erhalten; diese Dokumentationsrevision übernimmt
keine Produktänderung aus den isolierten Fixbranches.

Historische Evidenz, SRCH0-Korpus, aktive Sollwerte und strikte Tests
bleiben in dieser Dokumentationsrevision unverändert. Die bisherigen
Status-Goldens zur direkten SDK-Weitergabe und die Batch-/Nachlade-Goldens
bilden die revidierte Policy noch nicht ab und müssen gezielt nachgeführt
werden, ohne historische Beobachtungen umzuschreiben. Die technische
Trennung von Diagnose und verbindlicher Abnahme bleibt ebenfalls offen.
Ein grüner SRCH0-Gesamtlauf wird nicht behauptet; die bisherigen
Nicht-Blocker-Einstufungen und alle übrigen Anforderungen bleiben bestehen.

## Cluster-Trefferlimit: akzeptierte Begrenzung, kein SRCH0-Blocker

Entscheidung vom 19. September 2026: Die Karte bleibt unverändert. Ihre
begrenzte Clustergrundlage oberhalb des Engine-Caps ist **kein SRCH0-Merge-
oder Abnahmeblocker**. `maxTotalHits` ist ein konfigurierbares Engine-Limit
mit Standardwert 1000. Der bisherige Clusterrequest mit `limit=10000`
überwindet diese Grenze nicht; die Cluster repräsentieren in diesem Fall
nur die zurückgelieferten Touren. Das ist eine akzeptierte bekannte
Begrenzung und kein Versprechen einer vollständigen Darstellung.

Die Cluster-Vervollständigung in diesem Sammelbranch bleibt historische
Implementierungsreferenz und wird zurückgestellt. Sie wird weder auf einen
separaten Cluster-Fixbranch ausgekoppelt noch in den Import-Fix übernommen.
Kartenimplementierung, Engine-Einstellungen und UI bleiben unverändert.
Der Nachladeansatz erzeugt zusätzliche sequenzielle Suchanfragen, wachsende
ID-Ausschlusslisten und eine vollständige Sammlung im Arbeitsspeicher;
eine solche Vollständigkeitsanforderung ist keine SRCH0-Voraussetzung mehr.

Die Ausnahme betrifft die Cluster-Vervollständigung in `SRCH0-SEARCH-129`,
`API-FIX-SRCH0-SEARCH-129`, die Traversierungsanteile von
`SRCH0-COMPILER-072` samt `WEB-FIX-`-Solländerung und die zugehörigen
Vollständigkeitsprüfungen.
`SRCH0-SEARCH-130` dokumentiert dagegen das Engine-Cap der Listenpagination
und ist kein Nachweis einer separaten Clusterkorrektur. Er bleibt als
Beobachtung erhalten; eine Änderung der Listenpagination gehört nicht zu
dieser Arbeit. Sichtbarkeit und Zugriffsregeln bleiben verbindlich.

Die Dokumentationsentscheidung ändert weder historische Evidenz noch
SRCH0-Korpus, aktive Sollwerte oder Testimplementierungen. Ihre technische
Trennung von Diagnose und Abnahme steht weiterhin aus. Ein grüner
SRCH0-Gesamtlauf wird nicht behauptet.

## Gesamtabnahme bleibt offen

Die genannten Testergebnisse gelten für die jeweils isolierten Fixcommits.
Sie sind keine Abnahme der vollständigen SRCH0-Suite auf `feat/srch0`.
Vor deren Abnahme müssen alle Produktkorrekturen im verbindlichen Umfang
einschliesslich des noch ausstehenden Startup-Pakets integriert und am
gemeinsamen Zielstand geprüft sein. Historische Beobachtungen und aktive
Solländerungen des SRCH0-Korpus werden durch diese Aufteilung nicht geändert.

Die übergreifende Planung liegt auf `spec/enhanced-trail-search` in
`docs/src/content/docs/develop/specs/trail-search/work-items/search/srch0.md`.
Der Testbranch `feat/srch0` führt die Befunde und den Integrationsstand in
`scripts/srch0/BEFUNDE.md` sowie den Testablauf in `scripts/srch0/README.md`.
