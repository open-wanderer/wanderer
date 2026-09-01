---
title: "ADR 0001: Cursorzustand und Snapshot-Lifecycle"
description: Angenommene Entscheidung zu HMAC-Cursorn, Inhaltsständen, Generationenbindung und Retention.
editUrl: false
sidebar:
  order: 1
  badge: Angenommen
spec:
  id: ADR-0001
  kind: decision
  status: accepted
  capability: FOUNDATION
  lastReviewed: '2026-08-29'
---

- Status: angenommen
- Datum: 29. August 2026
- Umsetzungsbezug: `STATE1` spezifiziert persistente Collections und
  Transitionen; `IDX3` bezeichnet Epoch-Publikation und Retention; `SRCH1b`
  bezeichnet Cursor, Keyring und Gateway
- Normative Umsetzungsgrundlage:
  [Federation-, Index- und Gateway-Zustandsvertrag](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
- Normatives produktives Cutover-Gate:
  [Federation-Sicherheits- und Publikationsvoraussetzungen](/develop/specs/trail-search/contracts/federation-security/)
- Priorisierung, konkrete Umsetzungsschnitte und Beiträge:
  [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
- Betrifft: `trail-search.v1`

## Kontext

Treffer, Total, Facetten, Histogramme, Highlights und Pagination einer
Suchantwort müssen denselben logischen Inhaltsstand verwenden. Gleichzeitig
werden Suchprojektionen in Meilisearch asynchron aktualisiert und
Sichtbarkeitsverengungen müssen ohne Verzögerung fail-closed wirken.

Der Cursor muss deshalb die Anfrage, den Principal, den Inhaltsstand und die
nächste Position binden, darf aber selbst nichts autorisieren. Er muss kurze
Deployments, Schlüsselrotation und einen Generationstausch überstehen, ohne
unnötigen serverseitigen Sitzungszustand einzuführen.

Dieses ADR entscheidet die zuvor offenen Alternativen für `IDX3` und `SRCH1b`:

1. selbstenthaltender Cursor oder serverseitiger Cursorzustand;
2. `immutable` oder `revision_fenced` als Snapshotmodus der V1-Suche.

## Entscheidung

### 1. Selbstenthaltender, MAC-geschützter Cursor

V1 verwendet einen selbstenthaltenden, versionierten und mit HMAC-SHA-256
authentisierten Cursor. Derselbe Search-Gateway stellt ihn aus und prüft ihn;
eine asymmetrische Signatur ist deshalb nicht erforderlich. Der Algorithmus ist
serverseitig fest vorgegeben und wird nicht aus einem frei wählbaren Tokenfeld
übernommen.

Der Cursor ist für Clients API-opak, aber nicht vertraulich. Sein Payload enthält
nur die zur Fortsetzung und Validierung notwendigen, nicht vertraulichen Werte:

- Formatversion und `kid`;
- opake Control-Plane-Inkarnation als Restore-Fence;
- `as_of` und exklusives `valid_until`;
- `context_id`, Generation, Snapshot-ID und `content_epoch`;
- nächste Position beziehungsweise den Offset;
- kontextgesalzene, keyed Bindings für normalisierte Spec, Seitengrösse,
  Includes und Principal;
- notwendige opake Profil-, Policy-, Security-, Preference- und
  Federationrevisionen.

MAC-, Request- und Principalbindung werden aus demselben durch `kid` gewählten
und gemeinsam retenierten Cursor-Root-Key mittels HKDF-SHA-256 und festen,
versionierten Domainlabels abgeleitet. Damit existiert kein unversionierter
zweiter Binding-Key-Lifecycle. Die Principalbindung umfasst Tenant, effektiven
Actor beziehungsweise `anonymous_public` und Autorisierungsklasse, enthält diese
Werte aber nicht im Klartext.

Nicht im Cursor zulässig sind insbesondere:

- Suchtext, private Filterwerte oder private Geoangaben;
- rohe Actor-, Tenant-, Share- oder Overlay-IDs;
- interne Indexnamen oder Enginefilter;
- verborgene Ranking-, Match- oder ACL-Werte.

Benötigt eine spätere Pagingstrategie vertrauliche Positionswerte, darf sie
nicht still in den lesbaren Payload aufgenommen werden. Sie benötigt dann ein
neues ADR für AEAD-Verschlüsselung, serverseitigen Zustand oder einen anderen
Execution-Pfad.

Der Cursor ist replaybar und nicht single-use. Regulärer Ablauf erfolgt über
`valid_until`; eine Inhalts- oder Sicherheitsänderung invalidiert ihn über die
unten beschriebenen Epoch- und Fence-Prüfungen. Jede Cursorseite authentisiert
den Request erneut und prüft die aktuellen Restrictions. Der Cursor erweitert
niemals eine Berechtigung.

### 2. `revision_fenced` als einziger V1-Snapshotmodus

`content_snapshot.mode` ist in V1 immer `revision_fenced`. Die Content-Epoch ist
an eine konkrete Indexgeneration gebunden. Normale Änderungen in der vom Cursor
gepinnten Generation machen den Cursor `search_context_stale`; ein Wechsel des
aktiven Generation-Pointers allein tut dies nicht, solange die gepinnte alte
Generation unverändert und erreichbar bleibt.

Genau ein serialisierter Publisher führt Änderungen an einer abfragbaren
Generation aus:

1. Er wählt einen lückenlosen Katalog-Cutoff und setzt die Generation dauerhaft
   von `published` auf `updating`.
2. Erst danach darf er Kern-, Overlay- oder Geometry-Writes und Settings-Tasks
   an Meilisearch senden.
3. Er persistiert Taskintents und jede physische Submission. Der Intent muss
   terminal `succeeded` sein; jede möglicherweise angenommene Submission ist
   entweder selbst terminal bestätigt oder durch den in STATE1 normierten
   Barrier-/Endmarkerbeleg vollständig abgedeckt.
4. Delivery- und Security-Watermark müssen für alle Bestandteile lückenlos bis
   zum Cutoff bestätigt sein.
5. Erst dann publiziert ein atomarer Compare-and-swap die nächste monotone
   `content_epoch` und setzt den Zustand wieder auf `published`.

Diese Epochpflicht gilt für normale Content-, Geometry- und Settingswrites.
Der in STATE1 normierte `fence_apply` ist die einzige zustandserhaltende
Ausnahme: Bereits vor Ranking und Counts wirksame `exact_target`-Gateway-Fences
dürfen nur als vollständig gebundener kumulativer Visibilitystand aller das
physische Ziel überlappenden Subject-Frontiers beobachtungsäquivalent
realisiert werden. Dieser persistierte und serialisierte Security-Attempt
ändert weder Content-Epoch/-Cutoff noch andere Dokumentwerte oder Settings und
darf niemals einen Grant/Clear oder die Lockerung eines konservativen
Whole-Subject-Guards ausrollen. Ändert sich eine gebundene Frontier, kann der
alte Apply weder Delivery noch Retirement bestätigen; sein Fehler lässt die
Guards und damit die Antwort weiterhin fail-closed.

„Genau ein“ bezeichnet einen logisch exklusiven Writer mit monotonem
Fencing-Token. Ein Failover darf den Writer erst übernehmen, nachdem der alte
Prozess auf Infrastruktur- beziehungsweise Credentialebene vom Meilisearch-
Writepfad getrennt ist. Active/Active-Publisher ohne einen von der Engine
durchgesetzten Fence sind in V1 nicht unterstützt. Ein supersedierter Writer
darf keinen weiteren Engine-Task einreichen.

Ein HTTP `202`, die höchste beobachtete Task-ID oder einzelne erfolgreiche
Teilkomponenten reichen nie zur Publikation. Nach einem Crash wird der
persistierte Publish-Versuch fortgesetzt oder die Generation bleibt fail-closed
unbenutzbar; ein teilweise mutierter Stand wird nie als alte Epoch freigegeben.

Der Gateway liest Generation, Status, Epoch und Watermarks vor und nach dem
vollständigen Hits-/Total-/Facetten-/Histogramm-Batch. Nur zwei identische
`published`-Lesungen dürfen eine Antwort erzeugen. Eine erste Seite führt bei
einem Wechsel einen begrenzten vollständigen Retry aus und liefert danach
`search_unavailable`; eine Cursorseite liefert `search_context_stale`.

Sichtbarkeitsverengungen sind aktueller als jede Content-Epoch. Restriction-Fence
und Security-Watermark werden vor und nach jeder Antwort geprüft. Delete,
Public→Private, Share-Entzug oder Sperre können einen Kontext deshalb sofort
stale machen und dürfen weder Treffer noch Rang, Total, Bucket oder Highlight
leaken.

Die umgekehrte Richtung bleibt ebenfalls epochsicher: Ein Grant/Clear wird nur
mit einer normalen neuen Content-Epoch publiziert. Das gilt auch für die
Lockerung eines temporär konservativen Whole-Subject-Guards auf den exakten,
für einzelne Principals wieder sichtbaren eingeschränkten Zielstand;
epochloses `fence_apply` verlangt dagegen vollständige
Beobachtungsäquivalenz. Ist die neue Epoch bereits sichtbar, der relevante
bisherige Restriction-Fence aber noch nicht nachweislich `retired`, blockiert
der Gateway die Ausstellung jedes neuen Suchkontexts und Cursors. Derselbe
Guard läuft nochmals unmittelbar vor der Response-Sichtbarkeit. Ein Cursor der
vorherigen Epoch wird stale; ein Cursor der neuen Epoch kann deshalb nicht
zuerst unter dem alten Fence und später nach dessen Retirement mit erweitertem
Universum fortgesetzt werden. Wurde die alte Epoch durch einen Head-Swap
versiegelt, erzwingt der retenierte Guard-Release-Beleg die relevante
Stale-Entscheidung weiterhin; ein Pointerwechsel allein ist weder Sicherheits-
noch Stalenzbeleg.

Übernimmt eine spätere, nachweislich engere Restriction einen offenen
Guard-Release oder Clear, bleibt diese Regel erhalten: Vor Retirement ist die
geschlossene normale Successor-Epoch Pflicht. Alt-Cursor vor
`guard_release_revision` beziehungsweise `clear_revision` werden über den
retenierten Vorgänger-Fence stale; nach Retirement ausgestellte Kontexte
beginnen auf der Successor-Epoch.

Bei einer versiegelten Generation muss der aktuelle Fence vor Ranking und
Counts in jede betroffene Enginequery eingehen oder die Generation bleibt bis
zu ihrer sicheren Aktualisierung vollständig unbenutzbar. Ein Post-Filter auf
bereits gerankten Treffern ist unzulässig. `read-only` verbietet normale
Projektionswrites; sicherheitsbedingte Änderungen dürfen nur nach dem
vorherigen Stale-/Fail-closed-Schritt erfolgen.

### 3. Generationen, Retention und Deployment

Vollaufbau, grosser Backfill und Schemawechsel verwenden ein Schattenbundle.
Nach bestätigtem Cutoff-Replay und vollständigen Watermarks wird der aktive
Generation-Pointer atomar umgestellt. Die vorherige Generation wird sofort
read-only und erhält keine normalen Projektionswrites mehr.

Diese stillgelegte Generation ist eine unveränderliche Betriebsressource, aber
kein zweiter öffentlicher V1-Snapshotmodus. Der Gateway kann einen bereits
ausgestellten Cursor anhand seiner `index_generation_id` weiterhin dorthin
routen. Sie darf nach späteren Writes auf der neuen Generation nicht direkt als
veralteter Rollbackstand aktiviert werden. Ein Code-Rollback verwendet die
aktuelle kompatible Generation; ein Index-/Schema-Rollback baut aus Change-Log
und Projektionen einen getrennten Kandidaten, holt ihn bis zum aktuellen Cutoff
auf und aktiviert ihn erst nach denselben Task-, Coverage- und Watermark-Gates.
Die cursorgepinnte Generation bleibt dabei versiegelt.

Die Generation bleibt mindestens bis zum Maximum aus

- dem letzten von ihr ausgestellten `valid_until` zuzüglich In-flight-Grace und
- dem Ende des vereinbarten Rollbackfensters

erhalten. Erst danach darf die Garbage Collection sie entfernen.

Vor Rückgabe eines Cursors erhöht der Gateway dauerhaft und monoton die
`issued_valid_until`-High-Watermarks aller durch ihn benötigten Generationen,
Cursor-Root-Keys und versionierten Vertragsartefakte. Ein Crash zwischen dieser
Reservierung und der HTTP-Antwort darf nur zu Überretention führen; die Antwort
darf nie vor der Reservierung sichtbar werden. Dadurch bleibt die sichere
Retention nach Gateway- oder Publisher-Failover ohne per-Cursor-State
rekonstruierbar.

`context_resource_retention_grace_s` ist eine interne Betriebsgrösse und
mindestens die Summe aus maximaler Search-Request-Deadline und zugesagtem
Clock-Skew. Sie verzögert ausschliesslich die Ressourcen-GC und verlängert weder
`valid_until` noch die fachliche Cursorgültigkeit.

MAC-Prüfschlüssel bleiben mindestens bis zum grössten mit ihnen ausgestellten
`valid_until` plus `cursor_key_retention_grace_s` verfügbar. Rollende Deployments
teilen denselben versionierten Keyring und akzeptieren alte Cursorformate und
Vertragsartefakte bis zu deren Retentionsende. Ein Deployment oder eine
Schlüsselrotation allein invalidiert keinen noch gültigen Cursor.

Key- und Formatrollouts verwenden Expand/Contract: Zuerst erhalten alle
möglichen Leser den neuen Root-Key und die neue Formatversion, danach darf die
Ausstellung beginnen. Vor Entfernung alter Leserunterstützung wird die
Ausstellung des alten Formats beendet und dessen persistierter
`issued_valid_until`-High-Watermark plus Grace abgewartet. Ein Rollback auf einen
Reader, der bereits ausgestellte Cursor nicht versteht, ist verboten.

Publisherzustand, Generation-Mapping, Task-IDs, Watermarks und Keyring-Metadaten
sind dauerhaft und failoverfähig. Eine Ersatzinstanz darf eine Generation nur
bedienen, wenn Generation, Epoch, Delivery-/Security-Watermark sowie Schema- und
Settingshash exakt bestätigt sind. Bei Unsicherheit bleibt die Suche
fail-closed.

Ein PocketBase-Restore ist kein normaler Failover. Der in STATE1 definierte
extern linearisierte, sequenzierte Anti-Rollback-Checkpoint bindet
Securityrevision, Cursor-HWM und das vollständige Ressourcenmanifest per CAS.
Er erkennt zurückgedrehte Zusagen; bis Replay, Ressourcenretention und neuer
Control-Plane-Inkarnation bleibt der Gateway fail-closed. Dadurch werden alte
Cursor nach einem Restore nicht gegen einen scheinbar passenden, tatsächlich
älteren SQLite-Stand weitergeführt.

Der produktive Gateway-/Index-Cutover erfolgt erst nach einem vollständigen,
kontrollierten Rebuild und der terminalen Legacyklassifikation aller Records im
freizugebenden Scope. Schema-, Settings-, Coverage-, Delivery- und passende
Security-Watermarks müssen denselben Cutoff bestätigen. Meilisearch ist dann
netz- und credentialseitig nur über den Gateway erreichbar; alte Parent-Search-
Keys, Search-/Tenant-Tokens und direkte Legacy-Enginepfade sind widerrufen,
abgelaufen oder nachweislich gleichwertig gegated. Diese Voraussetzungen aus dem
[Federation-Sicherheits- und Publikationsvertrag](/develop/specs/trail-search/contracts/federation-security/)
blockieren weder Konzeptarbeit noch Backfills, Shadow-Builds oder andere nicht
ausstellbare Implementierungsarbeit vor dem Cutover.

Ein nicht belegter oder nicht konvergierter Federation-, Publication- oder
ACL-Scope wird fail-closed aus Ranking, Pagination, Total, Facetten,
Histogrammen und Highlights ausgeschlossen. Eine globale Suchsperre ist nur
zulässig, wenn der betroffene Scope nicht sicher bestimmt oder nicht vor allen
Auswertungsschritten abgezogen werden kann. Ein bestimmbarer defekter Origin-,
Publication- oder ACL-Scope darf die davon unabhängige lokale Suche und fremde
gesunde Scopes nicht stilllegen.

## Folgen

Vorteile:

- kein cursorbezogener Cache oder Datenbankzustand und keine Cursor-GC;
- kurze, deterministische Gültigkeit mit klarer Principal- und Requestbindung;
- inkrementelle Live-Updates statt einer vollständigen Snapshotkopie je
  Publikation;
- vorhandene Katalogrevisionen, Change-Log und Watermarks werden direkt genutzt;
- alte Generationen ermöglichen unterbrechungsfreie Deployments und dienen als
  geprüfte Quelle für einen getrennt aufgeholten Rollbackkandidaten.

Nachteile:

- jede publizierte relevante Mutation in einer gepinnten Generation kann
  laufende Cursor mit `search_context_stale` beenden;
- während `updating` kann die betroffene Generation keine vertragstreue Antwort
  liefern;
- jeder Meilisearch-Writer muss den Publisher-Fence respektieren;
- der Snapshotmechanismus löst die Deep-Paging-Kosten der Suchengine nicht.

Normale Änderungen dürfen kurz koalesziert werden. Längere Imports, Rebuilds und
Backfills verwenden eine Schattengeneration, damit die aktive Generation nicht
für die gesamte Laufzeit in `updating` bleibt.

## Verworfene Alternativen

### Vertraulich verschlüsselter Stateless-Cursor

AEAD bringt für den festgelegten Minimal-Payload keinen erforderlichen
Sicherheitsgewinn. Die Entscheidung wird neu geöffnet, sobald vertrauliche
Positions- oder Kontextwerte transportiert werden müssen.

### Serverseitiger Cursorzustand

Serverseitiger Zustand erleichtert gezielten Widerruf und kann beliebig grosse
Positionen speichern, benötigt aber replizierte Retention, GC und
Failoverkoordination. Der Cursor autorisiert nichts, lebt kurz und kann durch
aktuelle Fences vorzeitig beendet werden; diese Mehrkosten sind für V1 nicht
gerechtfertigt.

### `immutable` für jeden veröffentlichten Inhaltsstand

Ein solcher Modus benötigt ohne native Online-MVCC mehrere vollständige
Kern-/Overlay-/Geometry-Bundles oder ein neues temporales Dokumentmodell. Er
verbessert die Cursorstabilität, vervielfacht aber Storage, Page-Cache und
Catch-up-I/O beziehungsweise erhöht die Sichtbarkeitslatenz. V1 führt diese
Komplexität nicht ein. Die Alternative wird neu bewertet, wenn das nachfolgende
Mutation-/Paging-Gate die vereinbarten Budgets mit `revision_fenced` verfehlt.

## Verbindlicher Testplan und Abnahme

Testplan, Lastprofil und Budgets werden vor Beginn von `IDX3` festgeschrieben.
Die implementierten Tests müssen vor Abschluss von `IDX3`, vor Abschluss von
`SRCH1b` und erneut vor dem produktiven Cutover bestehen.

### Cursor und Datenschutz

- Manipulation, unbekanntes `kid` und fremder Principal liefern einheitlich
  `invalid_cursor`; eine abweichende Request-Binding liefert
  `cursor_request_mismatch`.
- Der maximal gültige Request mit 16 Aggregationsgruppen sowie aktiven
  Federation-/Spatialbindungen erzeugt einen Cursor von höchstens 4096 Bytes.
- Ein decodierter Maximalcursor enthält keinen Suchtext, keine privaten Filter-,
  Geo- oder Principalwerte, keine internen Indexnamen und keine verborgenen
  Rankingwerte.
- Derselbe Cursor ist replaybar und liefert bei unverändertem Kontext eine
  bytegleich relevante Folgeseite.

### Deep Paging

- vollständige Traversierung der durch `total` bezeichneten Treffermenge über
  mehr als 1'000 Treffer für leere Suche, Relevanz und jede freigegebene
  Sortierung;
- keine Lücken oder Duplikate, totale Tie-Break-Reihenfolge und bytegleich
  relevante Antwort beim Replay desselben Cursors;
- kein stilles Ende am Engine-`maxTotalHits`; kann der V1-Vertrag nicht erfüllt
  werden, blockiert dies die Freigabe oder verlangt einen eigenen Execution-Pfad;
- paralleler Mutation-Soak mit vorab festgelegtem Budget für Cursor-Stale-Rate,
  `updating`-Duty-Cycle, First-Page-Verfügbarkeit und p95/p99-Latenz.

### Retention

- gültiger Cursor unmittelbar vor und ab `valid_until`;
- Crash vor und nach der dauerhaften `issued_valid_until`-Reservierung sowie vor
  Rückgabe der Cursorantwort; es entsteht höchstens sichere Überretention;
- Schlüssel-, Profil- und Generation-GC erst nach ihrem jeweiligen
  Retentionsende;
- absichtlich zu früh verlorenes zugesagtes Artefakt ergibt
  `search_context_unavailable`, nicht `expired` oder eine gemischte Antwort.

### Deployment

- Expand/Contract-Rollout für Reader, Cursorformat und MAC-Key sowie erlaubte
  alte→neue und neue→alte Gatewaywechsel zwischen zwei Seiten;
- Shadow-Build und Generation-Cutover mit einem Cursor auf der vorherigen
  Generation;
- Code-Rollback auf der aktuellen kompatiblen Generation sowie Aufbau und
  Catch-up eines getrennten Index-/Schema-Rollbackkandidaten; die cursorgepinnte
  alte Generation wird nicht mutiert oder direkt veraltet reaktiviert;
- alte Generation bleibt bis zum letzten gültigen Kontext erreichbar und wird
  danach deterministisch bereinigt.
- produktiver Cutover erst nach vollständigem Rebuild, terminaler
  Legacyklassifikation und übereinstimmenden Coverage-, Delivery- und
  Security-Watermarks für jeden freizugebenden Scope;
- Gateway-only-Erreichbarkeit von Meilisearch sowie Rotation, Widerruf oder
  nachgewiesener Ablauf aller alten Parent-Search-Keys und Search-/Tenant-Tokens;
- negative Prüfung, dass direkte Legacy- und Enginepfade die Gateway-, DTO-,
  ACL-, Publication- und Restriction-Regeln nicht umgehen können.

### Failover und Crash-Recovery

- Crash vor und nach `updating`, Task-Submit, Task-Erfolg und Epoch-Publikation;
- `failed`, `canceled`, Timeout und verlorene Taskantwort;
- konkurrierender oder supersedierter Publisher wird vor einem weiteren
  Engine-Write abgewehrt;
- Gateway-/Engine-Wechsel auf eine bestätigte identische Generation sowie
  fail-closed Ablehnung einer Replica mit älterer Epoch oder Watermark;
- Restriction-Fence während Query, Cursorfolge, Shadow-Build und Rollback leakt
  keine geschützten Daten.
- Ein bestimmbarer fehlerhafter Origin-, Publication- oder ACL-Scope wird vor
  allen Auswertungsschritten ausgeschlossen, während lokale Suche und fremde
  gesunde Scopes verfügbar bleiben; nur unbestimmbare Scopezuordnung sperrt den
  gesamten betroffenen Suchpfad.
- Restore eines Backups vor der letzten Securityrevision beziehungsweise
  Cursor-HWM bleibt bis Anti-Rollback-Replay und neuer Inkarnation vollständig
  fail-closed.

Ein verfehltes Stale-/Availability-Budget öffnet dieses ADR erneut; es darf nicht
durch stilles Paging-Clamping oder schwächere Konsistenz umgangen werden.

## Referenzen

- [Vertrag: Trail-Suche v1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [Vertrag: Federation-Sicherheits- und Publikationsvoraussetzungen v1](/develop/specs/trail-search/contracts/federation-security/)
- [Vertrag: Federation-, Index- und Gateway-Zustandsautomaten v1](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
- [Konzept: Erweiterte Trail-Suche](/develop/specs/trail-search/)
- [Meilisearch: Tasks überwachen](https://www.meilisearch.com/docs/capabilities/indexing/tasks_and_batches/monitor_tasks)
- [Meilisearch: Snapshots](https://www.meilisearch.com/docs/resources/self_hosting/data_backup/snapshots)
- [Meilisearch: Pagination](https://www.meilisearch.com/docs/capabilities/full_text_search/how_to/paginate_search_results)
