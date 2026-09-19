# SRCH0: Aufteilung der Produktkorrekturen

Stand: 19. September 2026.

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
