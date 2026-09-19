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

## Gesamtabnahme bleibt offen

Die genannten Testergebnisse gelten für den isolierten Radius-Fixcommit.
Sie sind keine Abnahme der vollständigen SRCH0-Suite auf `feat/srch0`.
Vor deren Abnahme müssen alle Produktkorrekturen im verbindlichen Umfang
einschliesslich des noch ausstehenden Startup-Pakets integriert und am
gemeinsamen Zielstand geprüft sein. Historische Beobachtungen und aktive
Solländerungen des SRCH0-Korpus werden durch diese Aufteilung nicht geändert.

Die übergreifende Planung liegt auf `spec/enhanced-trail-search` in
`docs/src/content/docs/develop/specs/trail-search/work-items/search/srch0.md`.
Der Testbranch `feat/srch0` führt die Befunde und den Integrationsstand in
`scripts/srch0/BEFUNDE.md` sowie den Testablauf in `scripts/srch0/README.md`.
