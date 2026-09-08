# SRCH0: Suchbestand prüfen und weiterentwickeln

SRCH0 verlangt fachlich korrektes Suchverhalten. Historische Beobachtungen
helfen beim Vergleich, dürfen aber keinen bekannten Fehler legitimieren.
Jede verletzte geprüfte Eigenschaft lässt den Test scheitern; Ausnahmen für
bekannte Fehler sind verboten. Die [Befunde](BEFUNDE.md) sind Merge-Blocker.

Tests und Sollwerte werden auf `feat/srch0` gepflegt. Die Produktkorrekturen
entstehen getrennt auf `fix/srch0-findings`. Solange der von SRCH0 geprüfte
Produktstand die Fehler enthält, müssen die betroffenen Tests rot bleiben.
SRCH0 darf erst nach Integration der Fixes und erfolgreichen fachlichen
Prüfungen gemergt werden. Ein grüner Lauf in einer kombinierten Prüfkopie
ersetzt diese Abnahme des tatsächlichen SRCH0-Branchstands nicht.

Die Baseline ist `e9b7a8cade980002acbcf2e2f5b2a083934f29d2` vom 7. September
2026, direkt aus dem aktualisierten `origin/dev`. Die
[deutsche Spezifikation](../../docs/src/content/docs/develop/specs/trail-search/work-items/search/srch0.md)
ist eigenständig lesbar. Weitere Suchentwürfe und der UI-Prototyp sind nicht
Teil dieses Branches.

## Einstieg

Voraussetzungen: Node 22, Go 1.26, Docker und Playwright-Chromium.
Vom Repositoryroot aus:

```sh
npm ci --prefix web
make srch0-check
make srch0-unit
make srch0-engine
(cd web && npx playwright install chromium)
make srch0-browser
```

| Befehl | Was er prüft |
| --- | --- |
| `srch0-check` | Schema, Digests, Inventar, konkrete Abdeckung und Negativtests der Schutzregeln |
| `srch0-unit` | Go-Projektion/Mutation/Token/Startup sowie Web-State/Compiler/API und fachliche Proben |
| `srch0-engine` | echte Suche und unabhängige Eigenschaften auf beiden Engines; zusätzlich API, echte Mutationsaufträge und aktuelle Unknown-Projektionen |
| `srch0-browser` | URL, Storage, Snapshot und History in der echten SvelteKit-App |

Einzelne Schichten lassen sich getrennt ausführen:

```sh
node scripts/srch0/engine.mjs --version 1.11.3 --integration
node scripts/srch0/engine.mjs --version 1.36.0 --integration
(cd web && npm run test:unit -- --run src/lib/srch0/plausibility.test.ts)
```

Die Engine startet eigene Container ausschliesslich an Loopback-Adressen.
Die Browser-Suite startet ihre Dienste auf vom Betriebssystem vergebenen
Ports und beendet ihre eigenen Prozesse im Teardown. Ein vorhandener
Entwicklungsserver wird nicht benötigt. `GOCACHE` wird vom Runner nicht
überschrieben; für lokale Sonderfälle kann es ausdrücklich gesetzt werden.

Engineberichte stehen standardmässig in `/tmp/wanderer-srch0-reports`,
Web-/Browserberichte in `web/test-results`. `SRCH0_REPORT_DIR` überschreibt
das Berichtsverzeichnis der Engine und fachlichen Web-Proben. Berichte binden
das Manifest und den Änderungsdigest. Der SRCH0-Workflow lädt Engine- und
Browserberichte hoch; sein Korpusjob sowie die normalen Go-/Web-Jobs führen
die Unit-Tests aus. Der
separate Docs-Workflow baut geänderte Dokumentation und erzeugt vorher die im
Checkout noch fehlende OpenAPI-Datei. Er startet keine Engine-Matrix.

API-Vitest und Browser-Suite benötigen denselben generierten SvelteKit-Ordner.
Im gleichen Checkout deshalb nacheinander ausführen; in CI haben sie getrennte
Runner. Die oben aufgeführten Make-Befehle sind entsprechend sequenziell.

## Wo Änderungen hingehören

| Bereich | Einstieg | Verantwortung |
| --- | --- | --- |
| Korpus lesen | `corpus.mjs` | Datei-/Digestfunktionen und Auswahl der Basis- oder aktiven Erwartung |
| Korpus prüfen | `manifest.mjs`, `inventory.mjs`, `evolution.mjs` | Artefaktbindung, unabhängige Abdeckung und historische Identität |
| Solländerungen | `expectations.mjs` | vollständige Ersatzbeobachtungen, gebunden an den Basisdigest |
| Engine | `engine.mjs` | kurzer Ablauf: starten, ausführen, berichten, aufräumen |
| Enginegrenzen | `engine-lib.mjs`, `engine-results.mjs`, `engine-integration.mjs` | Lebenszyklus/HTTP, Ergebnisse/Eigenschaften, Go-/API-Ausführung |
| Web | `web/tests/srch0/*-adapter.ts` | getrennte State-, Compiler- und API-Adapter statt eines grossen Testblocks |
| Go | `db/srch0_*_test.go`, `db/routes/srch0_search_token_test.go` | echte Produktionsfunktionen; Hilfen unter `db/internal/srch0` |
| Browser | `web/tests/srch0/browser.spec.ts`, `browser-setup.ts` | Browserbeobachtungen und Prozessverwaltung |

Der Korpus liegt in
[`testdata/trail-search/srch0/v1`](../../testdata/trail-search/srch0/v1/README.md).
Thematische Gruppen wie `cases/compiler/sorting.json` enthalten gemeinsame
Angaben zu Familie, Dataset, Kontext und Codeankern sowie ein Array `cases`.
Jeder Fall behält seine ID, rohe Eingaben und Beobachtung. Abweichender Kontext
überschreibt einzelne Gruppenfelder; fallspezifische Codeanker ersetzen die
gemeinsame Liste. Baseline und Formatversion stammen zentral aus dem Manifest.

`loadCorpus` expandiert diese Angaben für alle Adapter und prüft Gruppen-,
Fall-, Dataset- und Änderungsdigests. Der Fall-Digest bezieht sich auf die
kanonische expandierte Basis. Das handgepflegte `inventory.json` beschreibt
unabhängig die erforderlichen Consumer und Merkmale; die Korpus-README bietet
eine kompakte Übersicht statt einer zweiten vollständigen Fallliste.

`sources.json` dokumentiert den ursprünglichen Commit und einmalig die
Dateihashes. Diese Herkunft kann bei Bedarf mit vorhandenem Baselinecommit
geprüft werden:

```sh
node scripts/srch0/manifest.mjs --verify-baseline
```

Das ist eine Provenienzprüfung, kein Schutz gegen Quellcodeänderungen. Die
laufenden Adapter prüfen den aktuellen Produktcode. Codeanker und
Abdeckungstags helfen beim Review; sie beweisen allein keine Semantik.

Die Standardmethode steht zentral in der folgenden Tabelle. `evidence.method`
ist optional und rein dokumentarisch: Es ergänzt nur fallspezifische Hinweise,
steuert weder Ausführung noch Freigabe und benötigt keinen wiederholten
Standardtext.

| Adapter | Standardmethode |
| --- | --- |
| `route-load`, `sanitize` | Produktive Loader und Filterprüfung mit den rohen Eingaben und kontrollierten externen Antworten ausführen. |
| `trail-dto` | Die produktive Umwandlung auf das gebundene Suchdokument anwenden. |
| `list-search`, `map-search`, `generic-search`, `global-multi`, `list-index-search` | Produktive Helper ausführen und ihre HTTP-Aufträge aufzeichnen. |
| `api-*` | Den jeweiligen produktiven Handler mit Principal und Präferenzen aufrufen; kontrollierte externe Antworten begrenzen die Unit-Probe. |
| `engine-search`, `engine-multi` | Anfragen auf beiden echten Engines ausführen; eine begründete Auswahl zusätzlich über die produktiven API-Handler prüfen. |
| `go-projection` | Produktive Projektoren mit Daten aus dem migrierten Testschema ausführen; defensive Rohdaten ausdrücklich kennzeichnen. |
| `go-mutation`, `go-search-token`, `go-startup` | Registrierte Hooks und Routen ausführen, Suchaufträge beziehungsweise kontrollierte Startup-Zwischenstände beobachten. |
| `browser` | In der echten SvelteKit-App navigieren und URL, Storage und History mit synthetischen externen Diensten beobachten. |

Der eingefrorene Referenzindex enthält historische Projektionen. Seine
Schwierigkeitswerte `0` prüfen den Transport von `0` nach `easy`, nicht die
Richtigkeit der damaligen Ableitung aus fehlenden PB-Werten. Dafür verlangen
`PROJECTION-008/010` unabhängig `null` aus den aktuellen PB-Projektoren und
materialisieren diese Aufträge auf beiden Engines; die Web-DTO-Prüfung
verlangt daraus Unknown ohne erfundene Schwierigkeitsstufe.

Die folgenden Adaptergrenzen erläutern die Aussagekraft dieser Methoden;
ein Methodentext ersetzt keinen ausführbaren Fall.

## Eine Produktkorrektur prüfen

Für eine Korrektur bleiben die historischen Basisfälle unverändert. Der
Testbranch erhält in `changes.json` pro betroffenem Fall eine vollständige
korrekte Ersatzbeobachtung unter `observed`, gebunden an dessen `basis_digest`.
Diese Erwartung gilt bereits vor Integration des separaten Produktfixes. Alle
Adapter verwenden diesen Ersatz im Speicher; die ursprüngliche Beobachtung
bleibt unter `baseline_observed` zugänglich.

Das folgende Beispiel gibt einen vollständigen Änderungsvorschlag für den
doppelten Radius aus. Es verändert keine Dateien und korrigiert den
Produktcode noch nicht:

```sh
node --input-type=module <<'JS'
import { loadCorpus } from './scripts/srch0/corpus.mjs';
const { entry, fixture } = loadCorpus().cases.find(
  ({ fixture }) => fixture.case_id === 'SRCH0-COMPILER-016'
);
const observed = structuredClone(fixture.observed);
const request = observed.engine_requests[0].body.options;
const clauses = request.filter.match(/_geoRadius\([^)]*\)/g);
if (clauses.length !== 2 || clauses[0] !== clauses[1]) throw new Error('Basis prüfen');
request.filter = request.filter.replace(` AND ${clauses[0]}`, '');
console.log(JSON.stringify({
  id: 'SRCH-COMP-GEO-001',
  case_id: fixture.case_id,
  basis_digest: entry.sha256,
  reason: 'Identische Radiusklausel entfernen; die Treffermenge bleibt unverändert.',
  evidence: ['web/src/lib/stores/trail_store.ts', 'scripts/srch0/expectations.test.mjs'],
  observed
}, null, 2));
JS
```

Den überprüften Eintrag in das Array `changes` von `changes.json` aufnehmen.
Weitere von derselben Korrektur betroffene Fälle erhalten eigene Einträge.
Pro Fall gilt genau eine aktive Ersatzbeobachtung. Weitere Korrekturen
bearbeiten diesen Eintrag; der Digest bleibt an die ursprüngliche Basis
gebunden. Unbekannte Fälle, doppelte Einträge und veraltete Basisdigests werden
abgewiesen. Der vollständige Ersatz macht die erwartete Antwort direkt
lesbar; seine fachliche Begründung und Evidenz bleiben Reviewaufgabe.

Für geänderte Startup-Settings kann die Ersatzbeobachtung `settings_overrides`
enthalten. Der Startupvergleich berücksichtigt diese gezielten Profiländerungen;
die Engine-Bestandssuite qualifiziert weiterhin ihre historischen Profile.
Ein neues Produktprofil benötigt eigene versionierte Qualifikationsfälle.

Anschliessend nur Metadaten neu erzeugen und die betroffenen Schichten prüfen:

```sh
node scripts/srch0/manifest.mjs --write
node scripts/srch0/manifest.mjs --base-ref origin/dev
make srch0-check
```

`--write` verändert keine Beobachtung. Der Vergleich mit der PR-Basis schützt
die veröffentlichte Identität und `observed`, auch wenn jemand den
Evidenztext ändert. Ein neuer historischer Erfassungsfall braucht eine neue
Case-ID. Bereits veröffentlichte IDs bleiben bei Rückzug mit Begründung und
Nachfolger im Manifest reserviert. Vor der Erstveröffentlichung können
Duplikate ohne Reservierung entfallen. Änderungen an Dataset/Profil brauchen
eine höhere Artefaktrevision;
neue/entfernte Fälle und geänderte Artefakte eine höhere Manifestrevision.

## Eine neue Baseline qualifizieren

Vor einem stabilen Release mit aktiven Solländerungen, spätestens bei
20 betroffenen Basisfällen, wird eine neue Korpusgeneration vorbereitet:

1. Die Beobachtungen am festgelegten Releasecommit frisch erfassen; vorhandene
   Solländerungen nicht automatisch als neue Basisbeobachtungen übernehmen.
2. Die bisherige Generation samt `changes.json` unverändert archivieren und
   alte Case-IDs plus Basisdigests auf neue Case-IDs plus Digests abbilden.
   Entfallene Fälle begründen; die neue Generation startet mit leeren Änderungen.
3. Alle sechs Familien einschliesslich API und beider Engineprofile erneut
   qualifizieren und die Zuordnung im Review prüfen, bevor die aktive
   Generation wechselt.

Eine Generation bezeichnet eine neue historische Bezugsbasis, keine
inkompatible Schemaversion. Bei gleicher Feldbedeutung kann die Formatversion
gleich bleiben. Die heutigen Schema-Konstanten für Commit und Datum sowie die
Runner sind jedoch an `srch0/v1` gebunden. Diese Bindungen und die Auswahl einer
weiteren Generation müssen vor dem Wechsel angepasst und geprüft werden.
Diese Regel beschreibt den Wartungsablauf, keine bereits vorhandene
Unterstützung für einen weiteren Korpuspfad.

## Grenzen der Adapter

Die Go-Testdatenbank verwendet echte Repositorymigrationen. Ausgenommen sind
nur drei historische Migrationen für externe Meilisearch-Settings; aktuelle
Settings werden im Startup- und Engine-Test ausgeführt. Auth, Feldtypen,
Relationen und Validierung stammen aus der Anwendung. Ungültige Difficultywerte
sind defensive Rohdatentests, keine angeblich gespeicherten Produktionsdaten.
Lesbare Fixture-IDs werden an der Datenbankgrenze auf gültige IDs abgebildet.
Fotos sind echte, synthetisch erzeugte JPEG-Uploads.

Die Ausschlussliste wird vor der ersten Migration vollständig gegen die
registrierten Dateinamen geprüft. Umbenannte oder fehlende Einträge führen zu
einer gezielten Fehlermeldung. Neue Migrationen mit externen Aufrufen müssen
bewusst in die Ausschlussliste aufgenommen werden.

Die Engine übernimmt die Suchregeln der geprüften Go-Tokenfälle. Unabhängige
Eigenschaftsprüfungen vergleichen ACL, Counts, Einzigartigkeit, Werte und
numerische Sortierung mit dem Quelldataset. Bereichsverengungen dürfen keine
zusätzlichen Treffer schaffen. Gleitkommazahlen erlauben beim Quellenvergleich
nur eine Abweichung im Bereich der Maschinengenauigkeit; die beobachteten
DTO-Digests bleiben davon unabhängig.

Bei Gleichständen erlaubt `hit_groups` einen Tausch innerhalb der Gruppe.
Unsortierte, abgeschnittene Mengen und Empfehlungen prüfen Form, Anzahl und
Eignung, ohne beliebige Trefferidentitäten einzufrieren. `non_contract` ist
kein pauschaler Testausschalter. Beide Engines liefern derzeit dieselben
Beobachtungen; unbenutzte `by_engine`-Sonderlogik wurde entfernt.

Die API-Auswahl in `selection.mjs` nennt die repräsentativen Fälle und ihre
Gründe: Principal-Grenzen, Präferenzen, Anfrageform, Mehrfachsuche, Fehler und
Startup. Die vollständige Suchmatrix läuft direkt gegen beide Engines;
API-Transport und Startup-Zustände werden gezielt zusätzlich geprüft.

Startup wird mit kontrollierten HTTP-Zwischenstufen geprüft. Suchbereitschaft
vor erfolgreich abgeschlossenen Tasks ist ein Testfehler. Die API-Suite prüft
den vollständigen Ready-Endzustand. Fehlerweitergabe und Wiederaufnahme eines
abgebrochenen Erstaufbaus werden zusätzlich getestet.

Die CI-Codepfade bleiben wegen indirekter Suchabhängigkeiten bewusst breit;
ein Cache vorbereiteter Testdatenbank-Vorlagen bleibt als optionale
Laufzeitoptimierung zurückgestellt.

## Befunde weiterverfolgen

[BEFUNDE.md](BEFUNDE.md) ist die fachliche Lesefassung. Die Web-Proben erzeugen
`srch0-plausibility.json`; jede verletzte Eigenschaft ist ein fehlgeschlagener
Test, unabhängig davon, ob der Fehler schon bekannt war. Ein Fix muss diese
Prüfung bestehen; die Rückkehr des alten Fehlers muss sie wieder scheitern
lassen. Go prüft korrekte Projektionen, Freigaben und aktualisierte Metadaten
unabhängig von den historischen Goldens. Vollständigkeits- und Startuptests
dürfen fehlende Treffer oder vorzeitig verfügbare Teilindizes nicht akzeptieren.

Die Kürzel in `successor_refs` sind Planungsreferenzen. Ihre Dokumente liegen
nicht in diesem Branch; sie sind weder tote Links noch eine Voraussetzung,
einen bestätigten Fehler mit dem beschriebenen Änderungsmechanismus zu beheben.
