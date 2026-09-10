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
  lastReviewed: '2026-09-08'
---

## Zweck und Aussagekraft

SRCH0 beschreibt das Suchverhalten am Commit
`e9b7a8cade980002acbcf2e2f5b2a083934f29d2` vom 7. September 2026. Der Branch
`feat/srch0` setzt direkt auf diesem aktualisierten `origin/dev` auf. Der
UI-Prototyp und die übrigen Spezifikationen des früheren Spec-Branches sind
nicht Bestandteil dieser Implementierung.

SRCH0 darf bei fachlich falschen geprüften Ergebnissen nicht grün sein. Die
Tests werden auf `feat/srch0` verschärft. Produktkorrekturen entstehen in zwei
separaten PRs: `fix/srch0-findings` für die übrigen Suchfehler und
`fix/search-index-startup` für das gesamte Startup-Paket. Dieses umfasst den
Erhalt bestehender Indizes, synchrone Initialisierung vor Suchbereitschaft,
Fehlerweitergabe, Wiederaufnahme und den Reparaturbefehl.

Beide Produkt-PRs blockieren den Merge von SRCH0. Bis der geprüfte Branchstand
beide Pakete enthält, müssen seine betroffenen Tests rot bleiben. Ein
kombinierter Testlauf dient der Fixprüfung und macht den ungefixten
SRCH0-Stand nicht mergebar. Der bisherige grüne kombinierte Lauf enthielt auch
das Startup-Paket und belegt keinen grünen SRCH0-Lauf ohne diese Änderungen.

Der Korpus unterscheidet drei Dinge:

1. **Historische Beobachtung:** Was erzeugte die Baseline bei einer konkreten
   Eingabe? Dieser Wert steht im unveränderten Basisfall unter `observed`.
2. **Aktive Solländerung:** Welche vollständige Beobachtung verlangt eine spätere
   Produktkorrektur? Sie steht in `changes.json` und ersetzt im Test die
   historische Erwartung, gebunden an deren Basisdigest.
3. **Plausibilitätsprüfung:** Erfüllt das Ergebnis eine unabhängig formulierte
   Eigenschaft, etwa Sichtbarkeit, gültige Koordinaten oder eine konsistente
   Sortierung? Diese Prüfung liest keine erwartete Treffermenge aus `observed`.

Jede Verletzung einer geprüften Eigenschaft lässt den Test scheitern. Es gibt
keine Ausnahme für bekannte Verstösse. Historische Beobachtungen sind weder
eine fachliche Freigabe noch ein Ersatz für korrekte aktive Erwartungen.

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
Karte und die nicht weitergereichte Feldauswahl. Ein negatives Thumbnail mit
vorhandenem Foto ist im Produktionsschema speicherbar und bringt den
Projektor zum Absturz.

Die Clusterabfrage erreicht beim grossen Dataset die Indexgrenze von 1'000
Treffern; die Duplikatprüfung untersucht nur die erste Engineantwort mit
höchstens 20 Trails. Strikte Produktprüfungen verlangen eine vollständige
Clustergrundlage und das Finden eines passenden Duplikats auch nach dieser
ersten Antwort. Eine dokumentierte Begrenzung darf diese Tests nicht bestehen
lassen.

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
