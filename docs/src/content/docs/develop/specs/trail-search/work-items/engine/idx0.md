---
title: IDX0 — Suchindex-Bootstrap und Readiness
description: Nichtdestruktiver Bootstrap der heutigen Suchindizes und konkreter Produzent von SearchReadinessV1.
editUrl: false
sidebar:
  order: 1
  badge: Bereit
spec:
  id: IDX0
  kind: work-item
  status: reviewable
  deliveryStatus: ready
  capability: FOUNDATION
  productSlice: search-index-bootstrap
  exposure: operational
  implementationDependsOn: []
  releaseGates: []
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-09-19'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | bereit; keine Implementierungsabhängigkeit |
| Implementierungsabhängigkeiten | keine |
| Releasegates | keine |
| Bestandslücke | `SRCH0-GAP-BOOT-001` |
| Runtimeumfang | heutige direkte Indizes `trails`, `lists` und `actors` |
| Normative Quelle | [Trail-Suchvertrag v1, Search-Readiness](/develop/specs/trail-search/contracts/trail-search-v1/#121-search-readiness-und-vertragsrevision) |

## Betreiberergebnis und Releaseeinheit

Ein normaler Neustart leert einen vorhandenen Suchindex nicht mehr. Fehlt ein
heutiger Index oder blieb ein von IDX0 begonnener Fill unvollständig, stellt
der Bootstrap ihn vor einem grünen Suchzustand wieder her. Fehler lassen die
Readiness rot. Ein rein prüfender Serve-Start bleibt dabei liveness-fähig; ein
fehlgeschlagener mutierender One-shot lässt DB und Web bis zur Wiederholung
gestoppt.

IDX0 produziert den Legacyzustand und die beiden Readiness-Endpunkte. Den
Zustand vor Token- und Enginezugriff zu konsumieren, bleibt vollständig bei
[SRCH-COMP](/develop/specs/trail-search/work-items/search/srch-comp/). Damit
kann IDX0 ohne SRCH-COMP, SRCH2, STATE1 oder M1 implementiert und abgenommen
werden.

## Heutiges Verhalten und Codeanker

- `db/main.go:236-239` startet `backfillPolylines` und den Indexneuaufbau in
  einer unbeobachteten Goroutine.
- `db/main.go:393`, `419` und `446` senden bei jedem Serve-Start für `trails`,
  `lists` und `actors` ein `DeleteAllDocuments`; die folgenden Dokumenttasks
  werden nicht terminal abgewartet.
- `db/main.go:178` bietet nur die Prozess-Liveness `GET /health`. Für die im
  Trail-Suchvertrag bereits normativ beschriebenen Readiness-Endpunkte gibt es
  keinen Implementierer.
- `db/util/polyline.go:138` speichert den Startup-Polyline-Backfill ohne Hooks.
  Ohne den heutigen Full-Rebuild würden geänderte Trailgrenzen deshalb nicht
  automatisch in den Index gelangen.
- `db/util/meilisearch.go:260-287` kann beim Bau einer föderierten Liste einen
  Remoteaufruf ohne Context oder Timeout ausführen. Ein synchroner Bootstrap
  darf daran nicht unbegrenzt vor `Serve` hängen.

SRCH0 hält diese historische Diagnose unter `SRCH0-GAP-BOOT-001` fest und
verlangt bereits vor seiner Abnahme Indexerhalt, erfolgreiche Initialisierung
vor Suchbereitschaft, Fehlerweitergabe und Wiederaufnahme. Diese
Bestandskorrektur darf unabhängig als Startup-Paket umgesetzt werden.
IDX0 erhält deren geprüfte Eigenschaften und besitzt darüber hinaus den
vollständigen Readinessproduzenten, Offline-Rebuild und Betriebsrollout.
IDX0 muss nicht auf die SRCH0-Abnahme warten; umgekehrt setzt diese nicht den
gesamten IDX0-Rollout voraus.

## Scope

IDX0 liefert genau drei zusammengehörige Änderungen:

1. Den asynchronen `DeleteAllDocuments`-Neuaufbau aus dem Serve-Start
   entfernen. Ein passender gefüllter Index bleibt bei einem normalen
   Neustart unangetastet.
2. Einen begrenzten Legacy-Ensure als One-shot vor `serve` und einen offline
   ausführbaren Rebuild bereitstellen. Fehlende, unerwartet leere oder nach
   einem abgebrochenen IDX0-Fill unterzählige Indizes werden aus PocketBase
   gefüllt; jeder eigene Engine-Task muss terminal erfolgreich sein.
3. `DB GET /health/search` und `Web GET /api/v1/health/search` als konkrete
   Legacy-Produzenten des bestehenden `SearchReadinessV1`-Vertrags
   implementieren.

Die festen Legacyrollen sind:

| Logische Rolle | PocketBase-Quelle | Index | Primärschlüssel |
| --- | --- | --- | --- |
| `trails` | `trails` | `trails` | `id` |
| `lists` | `lists` | `lists` | `id` |
| `actors` | `activitypub_actors` | `actors` | `id` |

## Lieferartefakte

### Legacyprofil

Ein gemeinsames, versioniertes Artefakt bindet je Rolle Quelle, Index,
Primärschlüssel und die vollständigen wirksamen Settings einschliesslich
expliziter Engine-Defaults. Es liefert die `profile_revisions`, den erwarteten
Settingsfingerprint und die interne
`search_contract_revision = legacy-search-direct.v1`. DB und Web betten
dasselbe Erwartungsmanifest ein und deklarieren unabhängig davon, ob ihr Build
diese Vertragsrevision unterstützt.

Das Profil führt kein neues Suchfeld und keine neue öffentliche API ein.

### Builder- und Verifiergrenze

Der Executor erhält für jede gewählte Rolle ein versioniertes Profil, einen
Full-Builder und einen Zielverifier. Context und Deadline werden durch alle
Quellreads, Projektionen, Remoteaufrufe, Engine-Submits und Task-Waits
durchgereicht. Ein abgelaufener Context bricht vor einem weiteren Submit ab
und ergibt niemals `ready`.

Für das eingebaute Legacyprofil prüft IDX0 nur Existenz, Primärschlüssel,
vollständige Settings und `numberOfDocuments == source_count`. Das ist kein
Dokumentvergleich und kein Driftbeweis. Für ein ownerfremdes Zielprofil ist
ein vom Owner gelieferter Verifier zwingend; dessen Dokument- und
Fachsemantik gehört nicht IDX0.

Der Erweiterungspunkt wird in IDX0 mit dem eingebauten Legacyprofil oder einem
lokalen Testprofil abgenommen. Die konkrete Kombination aus SRCH2-Builder,
SRCH2-Profil und SRCH2-Verifier wird ausschliesslich in SRCH2 abgenommen und
ist keine Voraussetzung für den Abschluss von IDX0.

## Ensure und Serve-Preflight

Der mutierende Ensure läuft im unterstützten Deployment als One-shot vor DB
und Web. Der DB-Prozess verwendet vor `se.Next()` denselben Inspektions- und
Abschlusscode, bleibt bei Reparaturbedarf aber mutationsfrei und rot.

Der gemeinsame Ablauf ist:

1. Den prozesslokalen Ownerzustand auf
   `not_ready/search_recovery_required` setzen. Den rollenbezogenen Engine-
   Task-Head und alle bereits sichtbaren nichtterminalen Alttasks lesen. Das
   Serve-Preflight wartet keine solche Task unter laufendem Web ab, sondern
   verweist rot auf den gefencten One-shot. Nur der One-shot wartet sie
   innerhalb der Ensure-Deadline terminal ab.
2. Quellanzahl, Indexexistenz, Primärschlüssel, vollständige Settings und
   Dokumentanzahl je Rolle lesen. Ein Quellfehler ist niemals die Zahl `0`.
   Die Zahl noch vom Polyline-Backfill betroffener Trails wird getrennt als
   Best-Effort-Diagnose erhoben und ist kein Readinessinput; auch ein Fehler
   dieses Diagnose-Reads ändert den Readinesscode nicht.
3. Benötigt der Befund einen Enginewrite und läuft nur das Serve-Preflight,
   mit `search_recovery_required` und der One-shot-Anweisung abbrechen. Der
   Serve-Pfad repariert nicht selbst.
4. Im One-shot fehlende Indizes mit Primärschlüssel `id` erstellen und nur
   abweichende Settings aktualisieren. Danach den Polyline-Backfill in
   begrenzten Batches ausführen: Geometrie zunächst nur im Record berechnen,
   den daraus erzeugten Trail terminal in die Engine schreiben und erst danach
   die abgeleiteten PocketBase-Felder ohne Hooks speichern. Ein Crash vor dem
   DB-Save lässt den Record für den nächsten idempotenten Versuch auffindbar.
   Scheitern GPX-Read, Parse oder Grössencheck vor dem ersten Engine-Submit,
   erhöht das nur den Diagnosezähler; der Record bleibt Kandidat und der
   strukturelle Ensure läuft weiter. Sobald ein Patch submitted wurde, bilden
   dessen terminaler Erfolg und der anschliessende PocketBase-Save den
   gemeinsamen Abschluss des gefencten Versuchs. Taskfehler, Save-Fehler oder
   Prozessverlust verhindern den grünen Wrapperabschluss und verlangen dessen
   vollständige Wiederholung.
5. Ist die nichtleere Quelle gegenüber einem fehlenden, leeren oder nach einem
   IDX0-Abbruch unterzähligen Index zu füllen, alle Quellrecords in begrenzten
   Batches upserten. Der Ensure sendet kein `DeleteAllDocuments`.
6. Für jede Create-, Settings- und Dokumenttask sowohl den Transporterfolg als
   auch den terminalen Meilisearchstatus `succeeded` verlangen. Der
   Listenbuilder gibt jedem heutigen Remote-Read ein begrenztes Sub-Timeout.
   Dessen Ablauf folgt wie andere Legacy-Fetchfehler der beobachteten
   Nullaggregat-Semantik; der Ablauf des übergeordneten Ensure-Contexts bricht
   den Lauf dagegen vor jedem weiteren Submit rot ab.
7. Nach den eigenen Tasks deren höchsten erwarteten Head festhalten, danach
   Struktur und Counts erneut lesen und den rollenbezogenen Task-Head ein
   zweites Mal prüfen. Ist darüber hinaus ein weiterer Alttask sichtbar
   geworden, beginnt innerhalb derselben Deadline erneut die Inspektion. Erst
   ein stabiler grüner Abschluss setzt
   `ready/search_ready`.

Das ist Detect-and-Retry, keine Zusage einer Task-Admission-Barriere. Ein
Timeout, ein fehlgeschlagener oder nicht sicher klassifizierbarer Task, ein
abweichender Primärschlüssel oder eine unerklärliche Überzahl endet rot mit
einer konkreten Betreiberdiagnose. Eine leere Quelle mit leerem Index ist
gültig. Ein gefüllter, countgleicher und strukturell passender Index löst
weder Dokumentpass noch Enginewrite aus.

Die blosse Existenz eines Polyline-Kandidaten und ein Fehler vor dem ersten
Engine-Submit sind kein Readinessgate. Sobald ein Patch submitted wurde,
gelten die allgemeinen Terminal- und Wrapperregeln; ein erfolgreicher
Enginepatch ohne folgenden PocketBase-Save darf den One-shot nicht grün
abschliessen. IDX0 führt keinen persistenten Fehlversuchsmarker ein und
begrenzt diese Crash-Zusage ausdrücklich auf den normativen Wrapperlauf; nach
dessen Unterbrechung muss der Betreiber den Wrapper erneut starten, nicht
`serve` direkt öffnen.

`initMeilisearchConfig` und `backfillPolylines` bleiben nicht als zweite
ungewartete Startup-Goroutine bestehen. Profilvergleich, nötiges Settings-
Update, Geometriepatch und terminale Waits liegen ausschliesslich im One-shot.
IDX0 persistiert weder Epochen noch einen
fortsetzbaren Laufzustand; nach Prozessverlust liest der nächste Ensure Quelle,
Engine und Task-Head neu.

## Quieszenz und Offline-Rebuild

Serveprozess und Offlinekommandos verwenden denselben betriebssystemseitigen
Instanzlock. Serve hält ihn über seine Laufzeit; der Offline-Rebuild muss ihn
vor dem ersten Enginewrite exklusiv gewinnen und bis zum terminalen Abschluss
halten.

IDX0 liefert dafür das CLI
`pocketbase search-index ensure --offline --profile legacy-search-direct.v1`
für den additiven Bootstrap, daneben
`pocketbase search-index rebuild --offline --profile <revision>` mit
verpflichtetem `--roles <role,...>` für den ausdrücklich gewählten
destruktiven Lauf. Betreiber verwenden beide ausschliesslich über diese
normativen Wrapperformen:

```text
scripts/search-index-bootstrap.sh ensure --profile legacy-search-direct.v1
scripts/search-index-bootstrap.sh rebuild --profile <revision> --roles <role,...>
```

Der Wrapper
stoppt die Compose-Services `web` und `db`, führt den One-shot mit den
produktiven PocketBase- und Meilisearch-Volumes aus und startet `db` und `web`
nur nach dessen grünem Abschluss wieder. Sein Fehlerpfad lässt beide Services
gestoppt und gibt die rollenbezogene Recoveryanweisung aus. Für andere
Orchestratoren ist dieselbe Reihenfolge als Betreiberpräcondition normativ:
`Web stoppen oder Engine-Egress sperren -> DB stoppen -> Ensure/Rebuild -> DB
starten -> DB ready -> Web starten beziehungsweise Egress öffnen`.

Nur dieser Offline-Einstieg darf einen mutationsbedürftigen Ensure oder
Rebuild ausführen. Ein zeitbasiertes Lease oder ein angenommener Request-Drain
ersetzt das Prozess-/Netzwerk-Fence nicht. Ein isolierter DB-Neustart und der
mutationsfreie grüne Normalstart benötigen den Wrapper nicht.

Der erste IDX0-Rollout läuft zwingend über den Wrapper, bevor eine neue DB-
oder Webbinary gestartet wird. Damit endet auch ein vom alten Startup-Pfad
bereits akzeptierter Delete-/Fill-Suffix unter dem Fence. Erst nach diesem
grünen Migrationslauf ist ein isolierter, mutationsfreier DB-Neustart zulässig.

### Betriebs- und Upgrade-Dokumentation

IDX0 aktualisiert im selben Lieferschnitt die unterstützte Docker-
Installations- und Upgradedokumentation in `CHANGELOG.md`, Quickstart und der
manuellen Docker-Anleitung. Für genau den ersten Wechsel von der alten
Startup-Goroutine auf IDX0 ersetzt sie ein nacktes
`docker compose pull && docker compose up -d` durch die vollständige
Wrapperfolge einschliesslich Fehler- und Wiederholungsfall. Diese einmalige
Migrationsanweisung ist ein IDX0-Lieferartefakt und hängt nicht von SEC-VIS-0
ab. SEC-VIS-0 behält die Verantwortung für die dauerhafte Netzwerk-, Token-
und Credentialtopologie; es darf die IDX0-Erstrolloutanweisung weder
voraussetzen noch nachliefern müssen.

Der Offline-Rebuild ist der einzige destruktive Einstieg von IDX0. Er leert
nur ausdrücklich gewählte Rollen und führt danach Builder, terminale
Task-Waits, Struktur-/Countread und Zielverifier aus. Ein ownerfremdes Profil
ohne Verifier wird vor dem ersten Delete abgewiesen. Der Rebuild ändert weder
Aktivierungsrouting noch Fachsemantik; Nachfolger wie SRCH2 besitzen beides
selbst.

Bei einer Überzahl sortiert die Diagnose alle betroffenen logischen Rollen
kanonisch und setzt sie bereits in die gefencte Wrapperzeile ein, zum Beispiel:

```text
scripts/search-index-bootstrap.sh rebuild --profile legacy-search-direct.v1 --roles actors,lists,trails
```

Sie gibt niemals nur den nackten destruktiven Binary-Aufruf oder einen
Platzhalter aus.

## Readiness-Produktion

Wireform, Statuscodes, Präzedenz, Kanonisierung und Geheimnisregeln stammen
ausschliesslich aus `SearchReadinessV1`:

- DB veröffentlicht den IDX0-Ownerzustand auf `GET /health/search`.
- Web validiert den geschlossenen DB-Payload, ergänzt nur seine eigene
  Unterstützung der aktiven Vertragsrevision und veröffentlicht denselben Typ
  auf `GET /api/v1/health/search`.
- Kann Web die DB nicht erreichen oder ihren Payload nicht validieren, darf es
  kein Grün erfinden. Nach Ablauf der vertraglichen Cachegrenze prüft es zuerst
  seine Unterstützung der im eingebetteten Erwartungsmanifest genannten
  Vertragsrevision. Bei fehlender Unterstützung gilt gemäss Präzedenz
  `not_ready/search_contract_unsupported`, sonst
  `not_ready/search_recovery_required`; ein beobachteter Fingerprint fehlt.
- `GET /health` bleibt reine Liveness; die Readiness-Endpunkte führen selbst
  weder Ensure noch Rebuild aus.

IDX0 implementiert keinen Search-, Token- oder Consumer-Guard. Die
Consumerteile von `RDY-03`, `RDY-10` und `RDY-11`, einschliesslich Lazy-
Tokenbezug und der vollständigen First-Party-Routenabdeckung, gehören zu
SRCH-COMP. Bis dieser Consumer ausgeliefert ist, verhindert das obige
Web-/Netzwerk-Fence jede Sichtbarkeit eines IDX0-Enginewrites.

## Nichtziele

- Keine Epochen, Generationen, Intents, persistente Operationsmarker,
  Task-Admission-Barriere, Attestierung oder Belegkette.
- Kein Koordinator für Fachwrites, Federation oder laufende
  Projektionsworker.
- Kein Vollvergleich oder Driftbeweis beim normalen Start.
- Kein Shadowindex, Swap oder atomarer Generationen-Cutover.
- Keine Änderung der Trail-, Listen- oder Actor-Fachprojektion; insbesondere
  bleibt die fachliche Korrektur föderierter Listen bei SEC-VIS-0.
- Kein Engineupgrade, Dump/Restore oder Notfallpfad aus M1.
- Keine Token-, Netzwerk- oder ACL-Härtung aus SEC-VIS-0.
- Kein STATE1-Handoff und keine Abhängigkeit von STATE1.

Das
[SRCH0-Designarchiv](/develop/specs/trail-search/evidence/srch0-design-archive-2026-09-01/)
ist ausschliesslich nicht bindende Vorarbeit und erweitert diesen Scope nicht.

## Abnahme

| ID | Fall | Erwartung |
| --- | --- | --- |
| IDX0-01 | Neustart mit drei gefüllten, passenden Indizes | keine Delete-, Dokument- oder Settings-Task; Readiness wird nach den Strukturreads grün |
| IDX0-02 | Serve-Preflight findet eine fehlende, leere oder nach IDX0-Abbruch unterzählige Rolle | kein Enginewrite; roter Zustand nennt den One-shot-Einstieg |
| IDX0-03 | Quelle und Index sind leer | Rolle ist ohne erfundenes Dokument ready |
| IDX0-04 | Create-, Settings-, Dokument- oder Alttask beziehungsweise der PocketBase-Save nach erfolgreichem Polyline-Patch scheitert | Wrapper endet ungleich null, lässt DB und Web gestoppt und nennt den vollständigen Wiederholungsbefehl; reine pre-submit Polyline-Datenfehler folgen IDX0-14 |
| IDX0-05 | Engine enthält für mindestens eine Rolle mehr Dokumente als PocketBase | kein stilles Delete; Diagnose nennt Ursache und gibt exakt die gefencte Wrapperzeile mit bereits eingesetzten, kanonisch sortierten Rollen aus |
| IDX0-06 | One-shot repariert eine fehlende, leere oder unterzählige Rolle | Compose-Wrapper hält Web und DB gestoppt; vollständiger Upsert ohne Delete; erst terminale `succeeded`-Tasks und Abschlussread starten DB/Web |
| IDX0-07 | gültiger Polyline-Kandidat; Save-Fehler oder Crash nach terminalem Patch, aber vor PocketBase-Save | Wrapper gibt Web nicht frei; der Record bleibt auffindbar, und die dokumentierte Recovery startet denselben gefencten One-shot vollständig neu |
| IDX0-08 | föderierter Listenread überschreitet sein Sub-Timeout | der Aufruf endet begrenzt und verwendet dieselbe Legacy-Nullaggregat-Semantik wie ein heutiger Fetchfehler; läuft stattdessen die gesamte Ensure-Deadline ab, endet der Lauf rot vor weiterem Submit |
| IDX0-09 | Serve-Preflight sieht eine nichtterminale Alttask oder ein Alttask wird erst nach der ersten Inspektion sichtbar | Serve wartet nicht unter laufendem Web und bleibt rot; im gefencten One-shot erzwingt ein veränderter Task-Head begrenzte Neuinspektion |
| IDX0-10 | erster IDX0-Rollout beziehungsweise Rennen von Offline-Rebuild und Serve-Start | Rollout läuft zwingend im Wrapper; genau einer gewinnt den Instanzlock und Web startet erst nach terminalem Erfolg |
| IDX0-11 | Offline-Rebuild mit lokalem Testprofil und Verifier | Erweiterungspunkt läuft vollständig ohne Nachfolgetask; ownerfremdes Profil ohne Verifier wird vor Delete abgewiesen |
| IDX0-12 | DB ist für einen frisch gestarteten Webprozess unerreichbar oder liefert einen ungültigen Payload | Web liefert aus dem gemeinsamen Erwartungsmanifest zuerst `search_contract_unsupported`, falls der eigene Build die Revision nicht unterstützt, sonst `search_recovery_required`; kein beobachteter Fingerprint |
| IDX0-13 | anwendbare Producerfälle aus `RDY-01` bis `RDY-09` sowie `RDY-12` und `RDY-13` | DB und Web bestehen Schema-, Status-, Präzedenz- und Fingerprintfälle; Consumerzusagen werden ausschliesslich in SRCH-COMP getestet |
| IDX0-14 | fehlende/ungültige GPX-Datei oder zu grosse Polyline bleibt über mehrere Starts Kandidat | Fehler- und Kandidatenzähler bleiben sichtbar; Indexstruktur und countgleiche Bestandsprojektion dürfen dennoch `ready` werden |
| IDX0-15 | Upgrade einer Bestandsinstallation auf den ersten IDX0-Release | Installations-/Upgradedokumentation führt zwingend über `scripts/search-index-bootstrap.sh`, beschreibt roten Fehlerpfad und Retry und enthält keinen nackten Pull-/Up-Pfad für diesen Übergang |

## Betrieb und Diagnose

Logs und Metriken nennen logische Rolle, Phase
(`inspect|create|settings|fill|verify`), Batchzähler, Quell- und
Dokumentanzahl, Laufzeit und terminale Fehlerklasse. Sie enthalten weder
Engine-Key, Suchtoken, Engine-URL noch interne Task-UID.

Der Polyline-Backfill meldet davon getrennt mindestens `candidates`,
`attempted`, `succeeded` und `data_failed_before_submit`. `candidates`,
`data_failed_before_submit` und einzelne intern diagnostizierbare Record-IDs
ändern keinen Readinesscode. Fehler nach einem Submit laufen stattdessen als
terminale Phasenfehler und folgen `IDX0-04`.

Der grüne Normalstart besitzt keinen dokumentweisen O(N)-Pfad. Für fehlende,
leere oder unterzählige Indizes zeichnet der Abnahmelauf Dauer und Batchzahl am
SRCH0-Korpus auf; die Ensure-Deadline ist eine deploybare Konfiguration und
wird im Timeouttest verbindlich geprüft.

## Entscheidungsprotokoll

| Datum | Entscheidung | Begründung |
| --- | --- | --- |
| 2026-09-01 | IDX0 besitzt den Legacy-Bootstrap und produziert `SearchReadinessV1`, SRCH-COMP konsumiert ihn. | Startup-Recovery und Zustandserzeugung sind unabhängig lieferbar; der gemeinsame Consumer bleibt beim Adapter. |
| 2026-09-01 | Mutierende Läufe verlangen ein echtes Web-/Netzwerk-Fence. | DB und Web sind getrennte Prozesse; eine frische Probe oder ein Zeitdrain schliesst das TOCTOU-Fenster nicht. |
| 2026-09-01 | Nachfolger liefern ihre fachlichen Builder, Profile und Verifier selbst. | IDX0 stellt nur die getestete Rebuildmechanik bereit und hängt dadurch an keinem Nachfolger. |
| 2026-09-01 | Polyline-Kandidaten und vor Submit erkannte Datenfehler sind Diagnose, kein Readinessgate. | Ein nicht ableitbarer Geometriecache degradiert den betroffenen Trail, nicht die globale Suchfähigkeit. |
