---
title: Federation-Sicherheits- und Publikationsvoraussetzungen v1
description: Normative Security- und Cutover-Gates für Federation, Publikation und föderierte Suche.
editUrl: false
sidebar:
  order: 3
  badge: Entwurf
spec:
  id: FEDERATION-SECURITY-V1
  kind: contract
  status: draft
  capability: FOUNDATION
  lastReviewed: '2026-09-01'
---

Status: Normativer Entwurf, 1. September 2026

Dieser Vertrag hält die beim Review der bestehenden Federation- und Suchpfade
gefundenen Sicherheitsvoraussetzungen fest. Er ergänzt das
[Konzept der erweiterten Trail-Suche](/develop/specs/trail-search/), den
[Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/), den
[Federation-, Index- und Gateway-Zustandsvertrag](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
und [ADR 0001](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/).

Die Schlüsselwörter **MUSS**, **DARF NICHT**, **SOLL** und **KANN** sind
normativ gemeint.

## 1. Wirkung auf Konzept, Implementierung und Releases

Die hier beschriebenen Befunde beenden oder pausieren die Konzeptarbeit nicht.
Suchvertrag, UI, Filter, Analyse, GeoJSON, Plugin-Typen, Provider und Chat dürfen
unabhängig von ihrer späteren Produktionsfreigabe weiter als Lösungsraum
spezifiziert und implementiert werden. Die Sicherheitsstufen sind stattdessen
verbindliche **Produktions- und Cutover-Gates**:

- Eine neue oder verbreiterte Suchfunktion darf erst produktiv auf einen
  betroffenen Datenpfad zugreifen, wenn dessen Gate erfüllt ist.
- Rein interne Analysen, Parser, Backfills, Shadowprojektionen und nicht
  ausstellbare Capability-Arbeit dürfen vorher entstehen.
- Alle beim Beginn dieses Vorhabens vorhandenen Filter, Sortierungen und
  URL-Zustände bleiben erhalten. Ein fail-closed Ausschluss nicht belegter
  föderierter Datensätze korrigiert das autorisierte Suchuniversum; er entfernt
  keinen Filter und deutet keinen bestehenden Filter um.
- Ein Gate gilt erst nach negativen Regressionstests und einer beobachtbaren,
  abbrechbaren Migration als erfüllt. Eine Absichtserklärung oder ein
  ausstehender Reindex genügt nicht.

Dieser Vertrag trennt bewusst Soforteindämmung und Zielarchitektur. Die
Eindämmung bleibt so lange aktiv, bis die Zielarchitektur ihren vollständigen
Bestand nachweislich trägt. Sie ist keine fachlich reduzierte Produktversion.
Priorisierung, konkrete Arbeitsschnitte und Beiträge werden nicht durch diesen
Vertrag festgelegt, sondern im
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
geführt.

## 2. Bestätigte Ausgangsbefunde

Die folgenden Punkte sind Bedingungen für die weitere Freigabe und keine
vollständige Sicherheitsanalyse des gesamten ActivityPub-Stacks:

1. Ein direkt an einen Actor adressierter Trail- oder List-`Announce` wird beim
   Empfang derzeit vor Anlage des Shares als `public=true` gespeichert
   (`db/federation/announce.go:125`, `db/util/activitypub.go:260`,
   `db/util/activitypub.go:571`). Die heutigen Search-Rules
   behandeln `public=true` als allgemein auffindbar
   (`db/routes/search_token.go:13`).
2. `TrailFromActivity` ist kein vollständiger Ingest-Chokepoint. Explizites
   Remote-Resolve und Listenexpansion übernehmen Remote-Felder einschliesslich
   `public` über direkte Record-Mutationen
   (`db/routes/remote_trail.go:239`, `db/routes/remote_list.go:218`). Auch Parent-Fetches
   für Kommentare und Summit-Logs müssen dieselbe Provenienzprüfung verwenden;
   der Autor des Nebenobjekts ist nicht automatisch Authority des Trails
   (`db/federation/create.go:483`).
3. Der eingehende Trail-/List-Deletepfad besitzt keine persistierte
   Object-to-Authority-Bindung als Mutationsgrenze
   (`db/federation/delete.go:266`). Ein gültig
   signierter Actor darf nicht allein deshalb ein beliebiges bekanntes Objekt
   verändern oder löschen.
4. Ein Meilisearch-Fehler in Delete-Hooks darf den Prozess nicht über
   `log.Fatalf` beenden
   (`db/hooks/trails.go:122`, `db/hooks/activitypub_actor.go:34`).
5. Der synchrone Inboxpfad antwortet derzeit auch bei Fachfehlern mit HTTP 200
   (`db/routes/activitypub.go:112`). `202 Accepted` ist erst
   zulässig, nachdem die signierte Eingabe dauerhaft und idempotent gespeichert
   wurde.
6. Ausgehende Aktivitäten laufen derzeit in einer Fire-and-forget-Goroutine
   ohne dauerhaften Zustellstatus
   (`db/federation/activity.go:55`). Eine zugesagte
   Rücknahme darf davon nicht abhängen.

Diese Befunde werden in privaten Security-PRs oder einem koordinierten
Disclosure-Prozess behandelt. Öffentliche Roadmap- und API-Texte beschreiben
die Garantien, nicht ausnutzbare Einzelheiten.

## 3. Sicherheitsstufen

### 3.1 `SEC-VIS`: lokale Sichtbarkeitseindämmung

`SEC-VIS` ist das früheste Produktionsgate. Sein Containment muss für einen
betroffenen Datenpfad wirksam sein, bevor dort eine neue oder verbreiterte
Suchfunktion live geht; es hängt nicht von den späteren Zielarchitektur-Gates
ab.

Bis die persistierte Provenienz vollständig migriert ist, gilt für Trail-Suche:

```text
anonymous_public:
  public = true AND is_federated = false

authenticated actor A:
  (public = true AND is_federated = false)
  OR (author = A AND is_federated = false)
  OR shares = A
```

Dieselbe Semantik MUSS für Listen gelten. Da der heutige Listenindex noch kein
filterbares `is_federated` trägt, ergänzt das Containment dieses abgeleitete
Feld und reindexiert den Listenbestand kontrolliert. Fehlende Eligibilityfelder
wirken fail-closed.

`SEC-VIS-0` ist zugleich der Delivery-Owner der dabei notwendigen
Bestandskorrektur für föderierte Listenaggregate. Der qualifizierte
Listenprojektor verwendet für `trails`, `distance`, `duration`,
`elevation_gain` und `elevation_loss` ausschliesslich die lokal persistierte
`ExpandedAll("trails")`-Relation. Eine leere Relation ergibt für alle fünf
Werte den numerischen Wert `0`, auch bei `needs_full_sync = true`; dieses Flag
ist ein Refreshhinweis und kein fehlender Projektionsinput. Projektion,
Reindex und inkrementelle Aktualisierung dürfen dafür keinen Origin- oder
Remote-Searchrequest ausführen. Nur eine persistierte, aber nicht auflösbare
Pflichtrelation endet mit `source_projection_incomplete`.

Das task-spezifische SEC-VIS-0-Ziel-Overlay referenziert
`SRCH0-GAP-LIST-001` und die zugehörigen stabilen Case-IDs, ohne deren
Baselineerwartungen zu ändern. Es erlaubt als fachliches Delta ausschliesslich
die fünf Aggregatwerte und deren unmittelbar gerenderte Darstellung.
Pflichtfälle sind ein leerer Federation-Stub mit
abweichendem Fake-Origin, eine vollständig lokal materialisierte Liste mit
zwei Trails, der Übergang vom Stub zur Materialisierung und eine tatsächlich
dangling persistierte Relation. Der erste Fall erwartet fünfmal `0` und in
allen vier Fällen exakt null externe Projektionsrequests. Die sichtbare
`0`-Semantik erhält einen Release-Hinweis. SRCH0 liefert nur Korpus und
beobachtete Baseline; SRCH-COMP, SRCH2 und STATE1 besitzen dieses Delta nicht.

Die allgemeine SEC-VIS-0-Accessmatrix bindet daneben
`SRCH0-GAP-SEC-001`. Jede dort beobachtete ACL- oder Bypassabweichung wird
ausschliesslich durch die Sichtbarkeits-, Proxy-, Netzwerk- und Tokenregeln
dieses Vertrags geschlossen; das SRCH0-Golden bleibt historisch unverändert.

Später ersetzt die belegte Federation-Sichtbarkeit den pauschalen Ausschluss:

```text
public = true
AND (
  is_federated = false
  OR federation_visibility = verified_public
)
```

Für authentifizierte Benutzer kommen ein lokal belegtes
`author = A AND is_federated = false` und ein nachweislich aktiver
`shares = A` weiterhin als getrennte ACL-Zweige hinzu. Der Owner-Zweig wird
nicht aus einem föderierten Payloadfeld abgeleitet; ein direkter Remote-Share
gelangt ausschliesslich über seinen lokalen Grant in den Share-Zweig. Eine
öffentliche Remote-Behauptung oder ein Legacy-`public=true` genügt nie als
`verified_public`.

Die Containment-Freigabe umfasst:

- neue Regeln für anonyme und authentifizierte Suche;
- Trail **und** Liste;
- Erhöhung der Token-/Cookie-Vertragsversion;
- Widerruf beziehungsweise Rotation des Parent-Search-Keys, damit bereits
  ausgegebene 24-Stunden-Tenant-Tokens die alte Regel nicht weiter verwenden;
- einen kontrollierten Reindex mit explizitem Erfolgs- und Coverage-Gate;
- negative Tests für lokale und föderierte, öffentliche, private, geteilte,
  widerrufene und gelöschte Einträge.

Der Browser-/Adaptertest beginnt zusätzlich mit einem noch nicht abgelaufenen,
bis zu 24 Stunden alten Cookie der vorherigen Epoche. Nach Parent-Key-Rotation
muss er das Cookie vor Verwendung verwerfen, einen neuen Tenant-Token beziehen
und die Suche erfolgreich ausführen. Der getrennte Fallbackfall erzwingt einen
ersten Engine-`403` und beweist genau einen Refresh/Retry sowie den Abbruch ohne
Schleife bei einem zweiten `403`. Ziel- und Rollbackartefakt bestehen dieselbe
Matrix.

`SEC-VIS` wirkt ausschliesslich lokal. Es wartet weder auf eine Outbox noch auf
die Erreichbarkeit einer Remote-Instanz und vertraut nicht darauf, dass ein
Peer eine Nachricht verarbeitet.

### 3.2 `SEC-AUTH`: unveränderliche Objekt-Authority

`trails.author`, `lists.author`, Actor-Domain und IRI-Pfadheuristiken sind
Anzeige- beziehungsweise Fachattribute, aber keine Mutationsberechtigung. V1
persistiert für jede föderierte Objekt-IRI eine unveränderliche
Object-to-Authority-Bindung mit mindestens:

```text
object_iri
object_kind                 legacy_trail | trail_publication | list_publication
authority_actor_iri
object_origin_id
authority_origin_id
authority_rule              same_origin_dereferenced_v1
first_verified_at
first_evidence_digest
state                       active | deleted | quarantined
version
```

Die Bindung entsteht atomar mit der ersten akzeptierten Objektwirkung. In V1
gelten dabei diese Regeln:

1. Delivery-Signer, `activity.actor` und Object-Authority werden getrennt
   modelliert, selbst wenn sie im ersten Interoperabilitätsprofil identisch sein
   müssen.
2. Angeforderte Actor-IRI, zurückgelieferte `Actor.id`, `publicKey.owner`,
   verwendete Key-ID und erlaubte Redirect-Origin müssen zusammenpassen.
3. `Create` verlangt genau eine `attributedTo`-Authority. Sie muss im Profil
   `same_origin_dereferenced_v1` dem verifizierten Activity-Actor entsprechen;
   Actor-, Authority- und Object-Origin sind gleich.
4. Das Objekt wird über seine kanonische HTTPS-IRI dereferenziert. Antwort-ID
   und Attribution müssen dieselbe Bindung bestätigen. Ein eingebettetes
   `attributedTo` allein genügt insbesondere bei `Announce` nicht.
5. `Update`, Refresh und `Delete` legen niemals implizit eine neue Authority an.
   Sie müssen die bestehende Bindung treffen.
6. `Delete` wechselt die Bindung per CAS nach `deleted`; der Tombstone bleibt
   absorbierend. Wiederholtes Delete derselben Authority ist idempotent.
7. Unbekannte oder widersprüchliche Authority verändert weder Fachrecord noch
   Sichtbarkeit und endet permanent abgelehnt oder quarantänisiert.

Keyrotation erneuert einen append-only Proof unter derselben Actor- und
Originidentität. Actortransfer, Cross-Origin-Delegation und Forwarding erhalten
später eigene versionierte `authority_rule`-Werte. Sie werden nicht durch eine
Lockerung von `same_origin_dereferenced_v1` nachgerüstet.

### 3.3 `SEC-DUR`: dauerhafte Inbox und Outbox

Für den heutigen synchronen Inboxpfad gilt bis zur Umstellung:

- erfolgreicher beziehungsweise idempotenter Abschluss: `200` oder `204`;
- permanent ungültige Syntax, Signatur, Authority oder Policy: passender 4xx;
- transienter interner Fehler vor Abschluss: 5xx;
- niemals `200` mit einem serialisierten Fachfehler.

Nach Einführung der durablen Inbox gilt `202` ausschliesslich nach atomarem
Commit der bytegenauen signierten Eingabe und ihres Transport-Idempotency-Keys.
Danach erfolgen Retry, terminale Ablehnung und Quarantäne intern; ein bereits
bestätigtes `202` wird nicht nachträglich in eine zweite HTTP-Antwort
umgedeutet. Replays liefern einen statusneutralen sicheren Ack und offenbaren
weder Payload noch Sichtbarkeitszustand.

Die durable Outbox persistiert Activity, Authority, Audience, Empfänger,
Idempotency-Key, Versuchszustand, Backoff und terminales Ergebnis. Lokale
Privacy-Mutationen wirken bereits im eigenen DB-Commit; Remote-Zustellung ist
eine nachgelagerte, wiederholbare Nebenwirkung. Die UI verspricht deshalb
„föderierte Publikation zurückgezogen“, niemals beweisbare weltweite Löschung
bereits empfangener Bytes.

### 3.4 `SEC-GLOBAL`, `SEC-SCOPED` und `SEC-TARGETED`

`SEC-GLOBAL` ist ein dauerhafter Circuit-Breaker, aber kein ausreichender
Produktionsnachweis für föderierte Suche. Wenn Scope, Control-Plane oder
Securityzustand nicht sicher bestimmbar sind, darf er die gesamte betroffene
Suchfamilie konservativ sperren.

`SEC-SCOPED` ist die Mindestvoraussetzung für produktiv ausgelieferte
föderierte Treffer und Counts. Es kennt mindestens:

```text
local
all_federated
origin:<origin-id>
actor:<actor-iri>
acl:<acl-scope-id>
publication:<publication-iri>
```

Jede Restriction erhöht im linearisierenden DB-Commit die
`desired_revision` aller betroffenen Scopes über `max(old, revision)`. Ein
Worker projiziert den neuesten autoritativen DB-Zustand. Hundert Ereignisse
desselben Scopes sind damit höchstens eine laufende und eine anschliessende
Indexrunde, nicht hundert serielle Runden. Ein Peer, der fortlaufend
Restriktionen erzeugt, kann seinen Origin-Scope quarantänisieren, aber weder
lokale Suche noch andere Origins dauerhaft abschalten.

Der Gateway wendet einen Scope-Guard vor Ranking, Pagination, Total, Facetten,
Histogrammen und Highlights an. Kann er einen betroffenen Scope nicht sicher
abziehen, sperrt er diesen Scope. Nur wenn die Scopebestimmung selbst
unzuverlässig ist, eskaliert er zu `SEC-GLOBAL`.

`SEC-TARGETED` ergänzt später exakte Trail-/Publikations-/ACL-Guards und
verbessert die Verfügbarkeit. Es ersetzt `SEC-SCOPED` nicht: Actor-, Origin- und
Massenwiderrufe bleiben dauerhaft scopebezogene Operationen.

## 4. Gemeinsamer Federation-Ingest und Sichtbarkeitsprovenienz

Alle Federationpfade speichern Trails und Listen über einen gemeinsamen,
typisierten Persistenzdienst. Dazu gehören mindestens:

- Inbox `Create` und `Update`;
- direktes `Announce`;
- explizites Remote-Resolve und periodischer Refresh;
- Remote-Listenimport und expandierte Trails;
- synthetischer Parent-Fetch für Kommentar und Summit-Log;
- Admin-Resync und Legacy-Migration.

Der Dienst verlangt einen geschlossenen Evidence-Typ mit Source, Objekt-IRI,
Authority, Origin, Audience, Evidence-ID/-Digest, Beobachtungszeit und
verwendeter Verifikationsregel. Parser und Fetcher dürfen Records vorbereiten,
aber nicht ausserhalb dieses Dienstes öffentlich schalten.

Ein Before-Create/Before-Update-Guard bildet die zweite Verteidigungslinie: Ein
Record mit Remote-Author oder Remote-IRI darf ohne passende persistierte
Federation-Provenienz nicht `public=true` werden. Der Guard errät keine
Adressierung aus dem Record; er lehnt den Bypass fail-closed ab. Lokale
Pluginimporte behalten ihre eigene Provider-/Owner-Provenienz und werden nicht
als Federation umklassifiziert.

Sichtbarkeit und Prüfstatus sind getrennte Zustände:

| Prüfstatus | Sichtbarkeit | Suchwirkung |
| --- | --- | --- |
| `pending` | unbekannt | öffentlich ausgeschlossen |
| `verified` | `public` | Mitglied eines belegten Public-Scopes |
| `verified` | `restricted` | nur über belegten Actor-/ACL-Scope |
| `gone` | keine | Tombstone; nicht auffindbar |
| `invalid` | keine | terminal quarantänisiert |
| `unreachable` | keine öffentliche | terminal aus öffentlicher Suche ausgeschlossen |

`verified_public` ist der abgeleitete Search-Eligibility-Zustand, kein weiterer
frei schreibbarer DB-Status: Er gilt genau bei `verification = verified`,
`visibility = public`, aktueller Authority-/Audience-Provenienz und aktiver
Publication-/Scope-Membership. `public_verified` bezeichnet dagegen das
terminale Ergebnis, mit dem ein Legacy-Datensatz diese Voraussetzungen im
Migrationslauf bestanden hat. Die Projektion berechnet Ersteres aus den
persistierten Belegen; sie kopiert Letzteres nicht blind in den Index.

`unreachable` verhindert einen unbegrenzten Hot-Retry. Ein langsamer
Wartungsjob oder ein expliziter Admin-Resync darf später einen neuen Versuch
starten, schaltet den Record aber nie allein deshalb frei. Eine bereits
verifizierte direkte ACL kann den Cache weiterhin nur dem bezeichneten Actor
zugänglich machen; ohne solchen Beleg bleibt er vollständig quarantänisiert.

## 5. Migration des Bestands

Die Migration beginnt fail-closed: Alle bestehenden Remote-Records ohne
rekonstruierbaren Public-/ACL-Beleg erhalten `pending` und sind keine
öffentlichen Suchmitglieder. Ein bisheriges `public=true` ist kein Beleg.

Ein persistenter Keyset-Sweep hält Cutoff, Cursor, Catch-up-Revision,
Versuchsbudget und Counts. Jeder Record endet in genau einem terminalen
Migrationsergebnis:

```text
public_verified
scoped_verified
restricted_unproven
gone
invalid
unreachable
```

Produktive Freigabe von `verified_public` verlangt:

- vollständigen Sweep bis zum Cutoff und Catch-up bis zum aktuellen
  Federationstand;
- null unklassifizierte Records im freizugebenden Scope;
- DB-/Projektionscounts und Stichproben gegen denselben Cutoff;
- nachweislich wirksame Suchregeln und widerrufene alte Tenant-Tokens;
- beobachtbare Counts je Ergebnis, ältesten offenen Eintrag und Fehler je
  Origin;
- Crash-, Restore-, Concurrent-Ingest- und Resume-Tests.

Die Release Notes nennen ausdrücklich, dass föderierte öffentliche Treffer
nach dem Upgrade zunächst zurückgehen können und nicht erreichbare Origins
ausgeschlossen bleiben. Eine nicht abgeschlossene Migration wird nicht als
erfolgreicher Start versteckt.

## 6. Föderierte Publikationsidentität

Ein lokaler Trail und seine öffentliche föderierte Repräsentation sind
verschiedene Identitäten:

```text
Trail T                    dauerhafte lokale Fachidentität
FederatedPublication P1   eine veröffentlichte föderierte Generation
```

Eine interne `federated_publication` bindet mindestens Trail, eigene immutable
Publication-IRI, Generation, Authority, Audience, Payloadrevision, Zustand und
Outboxreferenzen. Der öffentliche Lebenszyklus ist:

| Lokale Wirkung | Föderierte Wirkung |
| --- | --- |
| Private Trail wird public | `Create(P1)` nach lokaler Projektion und durablem Outbox-Commit |
| Public Trail ändert öffentlichen Inhalt | vollständiges `Update(P1)` |
| Public → Private | lokale Sichtbarkeit und P1 im selben Commit restricten/tombstonen; bei möglicher voriger Netzwerkwirkung `Delete(P1)` durable einplanen; Trail T bleibt |
| Erneut public | neue IRI und `Create(P2)`; P1 bleibt tombstoned |
| Trail endgültig löschen | aktive Publikation löschen, danach lokalen Trail nach lokaler Policy tombstonen/löschen |

`Delete(P1)` ist damit ehrlich: Es beendet die Publikation, nicht die lokale
Trailidentität. Der lokale P1-Tombstone wartet nicht auf Remote-Zustellung. Hat
ein persistierter Non-Delivery-Beleg ausgeschlossen, dass `Create(P1)` das
Netzwerk erreichen konnte, darf Wanderer den Create-Intent stattdessen
abbrechen; andernfalls entsteht im Unpublish-Commit der Delete-Intent für alle
potenziell erreichten Empfänger. Reaktionen und Links auf P1 werden nicht
automatisch P2 zugeordnet. Das ist der bewusste Preis für interoperables
Unpublish ohne Resurrection einer gelöschten IRI. Ein Wanderer-spezifisches
`Retract` darf höchstens zusätzliche Information sein und ist nie die
Sicherheitsgrundlage.

Direkte Freigaben sind keine globale Publikation. Ein verifiziertes
Direct-`Announce` erzeugt einen `restricted` Snapshot plus Actor-/ACL-Scope und
niemals Membership in `all_federated`. Wird die bestehende Announce-Semantik
beibehalten, widerruft `Undo(Announce)` genau diese Distribution. Der lokale
Share-Entzug wirkt sofort; erfolgreiche Remote-Zustellung ist keine
Voraussetzung. Mehrere verbleibende Shares werden neu berechnet und durch den
Entzug eines einzelnen Shares nicht versehentlich gelöscht.

## 7. Verbindliche Transitionmatrix

Vor der Implementierung von Publikation, Federation-Suche oder neuen
Federation-Counts wird jede Zeile mit Authority, lokalem Fachcommit,
Security-Scopes, Dirty-Zielen, Fence, Outboxintent, API-Erfolgspunkt,
Idempotenz und Recovery konkretisiert. Mindestens enthalten sind:

| Übergang | Synchrone lokale Wirkung | Federation/Projektion |
| --- | --- | --- |
| Trail create private | nur Owner/ACL | keine Public-Publikation |
| Trail create public | lokal projektierbar | neue Publication erst nach Gate |
| private → public | keine vorzeitige Erweiterung | neue Epoch und `Create(Pn)` |
| public → private | sofortiger Fence | `Delete(Pn)` in durable Outbox |
| public/private delete | sofortiger Fence/Tombstone | aktive Publication gegebenenfalls `Delete` |
| Share add/update | genau betroffener ACL-Scope | Dirty-Koaleszierung, optional Direct-Distribution |
| Share revoke | genau diesen Grant sofort entfernen | `Undo(Announce)`/Distribution-Revoke; übrige Shares erhalten |
| Account/Actor delete oder block | alle abgeleiteten lokalen Scopes restricten | Actor-/Origin-Scope koaleszieren |
| Origin block | nur betroffene Origin restricten | Origin-Scope, niemals normale globale Dauersperre |
| Remote public Create/Update | nur nach Authority + Public-Beleg | Public-/Origin-/Actor-/Publication-Scope |
| Remote direct Create/Announce | `restricted`, nur Empfänger-ACL | keine globale Membership |
| Remote Delete/404/410 | Authority prüfen, dann sofort restricten | Tombstone und Scope-Dirty |
| Remote Timeout/5xx | keine Freigabe raten | Retry, danach gegebenenfalls `unreachable` |

Parallelereignisse werden gegen denselben Authority- und Kausalstand per CAS
geordnet. Dirty-Koaleszierung darf fachlich ungültige Zwischencommits nicht
kaschieren.

## 8. Releaseabhängigkeiten und mögliche Umsetzungsschnitte

| Arbeit | Voraussetzung für Release | Blockiert keine Konzeptarbeit an |
| --- | --- | --- |
| Bestands-/Subkategorie-/Sortier-UI auf heutigem Suchpfad | `SEC-VIS` | UI-Vertrag, Compiler, Tests |
| Föderierte Treffer, Counts und Originfilter | `SEC-VIS`, `SEC-AUTH`, `SEC-DUR`, `SEC-SCOPED` | Schema- und Shadowarbeit |
| Public→Private-Publikationsvertrag | `SEC-AUTH`, Publikationsmodell, `SEC-DUR` | lokale Trailanalyse |
| Gateway-/Index-Cutover | alle vier Gates, Parität aller Bestandsfilter | ADR-, Build- und Lasttests |
| `SEC-TARGETED` | `SEC-SCOPED` | bereits sichere scoped Federation |

Ein lokaler Legacy-Indexwechsel, der keine neue öffentliche Such-, Gateway-
oder Federationfähigkeit einführt, darf als enger Bestandsschnitt nach
`SEC-VIS-0` freigegeben werden. Das ist keine Ausnahme von den für den
tatsächlich exponierten Datenpfad geltenden Sicherheitsregeln: Sobald der
Schnitt föderierte Treffer, neue Scopes, Cursor, Aggregationen oder den neuen
Gateway aktiviert, gelten die vollständigen Gates der Tabelle.

Reindex, Quellvergleich, Task-Wait, Swap, Recovery und Rollback gehören dem
jeweiligen Migrations- beziehungsweise Indexowner. Dieser Securityvertrag
macht weder dessen Auditdateien noch interne Epochen, Taskheads oder
Readinesszustände zu Sicherheitsbelegen. Fehlende Engineindizes oder ein
technisch erfolgreicher Swap beweisen insbesondere keine zulässige
Treffermenge.

Für den engen Bestandsschnitt müssen gleichzeitig folgende Securitybelege
vorliegen:

- PocketBase bleibt die autoritative Fachquelle; der Schnitt führt keine neue
  Visibilityklasse oder öffentliche Requestcapability ein.
- Vor Öffnung des Ingress ist Meilisearch aus nicht autorisierten Netzen nicht
  erreichbar. Das Produktions-Compose veröffentlicht keinen Engine-Hostport,
  Web erhält keinen Master-Key, und DB erzeugt Tenant-Tokens aus einer
  ausdrücklich konfigurierten Search-Parent-Key-ID samt Secret.
- `SEC-VIS-0` besitzt im produktiven Root-Compose-Pfad,
  `docs/public/setup.sh`, Quickstart und der manuellen Docker-Dokumentation die
  Netzwerk-, Port-, Key-/Token- und Containmentanteile. Die einmalige
  Bootstrap-/Upgradereihenfolge für den ersten IDX0-Rollout besitzt IDX0 im
  selben gemeinsamen Dateisatz; diese geteilte Datei erzeugt keine
  Implementierungsabhängigkeit zwischen beiden Tasks. Ein lokales
  Entwicklungsprofil bindet `7700` höchstens an `127.0.0.1`; ein
  automatischer Negativtest deckt alle produktiven Pfade ab.
- Das von `SEC-VIS-0` gelieferte Trail-/Listenprofil enthält sämtliche für die
  lokale Containmentregel benötigten filterbaren Eligibilityfelder. Der
  Migrationsowner baut und verifiziert die daraus erzeugten Zielindizes nach
  seinem Vertrag; Security entscheidet weder seine Batch- noch seine
  Swapmechanik.
- Die vollständige ACL-/Proxy-/DTO-Matrix läuft vor dem Produktionsfenster in
  CI oder Staging am auszuliefernden DB-/Webartefakt und demselben Zielprofil.
  Die Produktionsinstanz dient dabei nicht als Principal-, Token- oder
  Sichtbarkeitsorakel.
- `SEC-VIS-0` rotiert den Search-Parent-Key und erhöht die lokale Token-/
  Cookie-Epoche. Ziel- und Rollback-Webartefakt deklarieren vor dem
  Wartungsfenster dieselbe neue Epoche. Ein Cookie einer anderen Epoche wird
  vor dem ersten Engineaufruf ersetzt; der gemeinsame serverseitige Adapter
  darf nach einem Engine-`403` genau einmal Tokenrefresh und Requestretry
  ausführen. Ein zweiter `403` endet ohne Schleife.
- Die Credentialrotation erfolgt nach der terminalen Aktiv- oder
  Abbruchentscheidung des Migrationsowners und zwingend vor öffentlichem
  Ingress. Ein vor der Rotation ausgestellter Tenant-Token wird danach
  nachweislich abgewiesen. Der für einen möglichen verifizierten Rückweg
  benötigte Engine-Mutationskey folgt ausschliesslich dem Migrationsvertrag.
- Die aktive `search_contract_revision` bindet die gemeinsam freigegebene
  Request-, ACL- und DTO-Grenze. Ein kompatibler App-Rollback kann dieselbe
  Revision verwenden; Cookie-Epoche und Indexvertragsrevision bleiben
  getrennt, und der Epochensprung allein verlangt keinen Reindex.

Der gemeinsam aktivierte Adapter gilt nur dann **nicht** als neue Capability,
wenn er ausschliesslich die inventarisierten First-Party-Legacyzustände
serverseitig kompiliert und die erlaubte Treffermenge sowie beobachtbare
Filter-/Sortiersemantik nicht verbreitert. Öffentliche V1-Suche, Capability
Discovery, Cursor/Snapshots, neue Filter oder Sorts, Aggregationen,
Routenradius und Federation-/Origin-Scopes bleiben in diesem Profil negativ
getestet deaktiviert.

Die folgenden Bezeichnungen beschreiben mögliche technische Schnitte, keine
zugesagte PR-Anzahl oder feste Ausführungsreihenfolge. Sie dürfen kombiniert,
geteilt oder anders angeordnet werden, solange ihre normativen Abhängigkeiten
und Releasegates erhalten bleiben:

- **SEC-VIS-0:** eigenständig wirksames Suchregel-Containment für Trail und
  Liste, einschliesslich Tokenrotation und Regressionstests.
- **SEC-VIS-1 (bisherige Lane `FED1a`):** gemeinsamer Federation-
  Persistenzdienst, DB-Invariante, Provenienz und terminale
  Bestandsmigration; Voraussetzung für belegte föderierte Sichtbarkeit.
- **SEC-AUTH-1:** immutable Object-to-Authority, typisierter Dispatch,
  Update-/Delete-Prüfung und Entfernung aller Prozessabbruchpfade;
  Voraussetzung für autorisierte Federation-Mutationen.
- **FED-PUB-1:** `federated_publication`, P1/P2-Lifecycle und vollständige
  Transitionmatrix; baut auf der Authority-Bindung auf.
- **SEC-DUR-1:** durable Inbox/Outbox, Statussemantik, Retry und Diagnose; wird
  für die entsprechenden produktiven Annahme- und Publikationszusagen benötigt.
- **SEC-SCOPE-1:** Scope-Heads, `desired_revision`-Koaleszierung, Gatewayguards
  und Burst-/DoS-Tests; setzt die dafür benötigte persistente Zustands- und
  Zustellbasis voraus.
- **SEC-TARGETED-1:** exakte Guards als additive Verfügbarkeitsoptimierung nach
  `SEC-SCOPED`.

Welche dieser Schnitte als konkrete Änderung vorbereitet wird und wie Beiträge
koordiniert werden, gehört in den
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/).
Bestehende Filter verschwinden durch keinen Schnitt.

## 9. Mindesttests

- Table-driven Ingestmatrix über Activity `Create`/`Update`, Direct-`Announce`,
  Remote-Resolve, Listenexpansion, Parent-Fetch und Admin-Resync.
- Anonymous/authenticated × local/federated × public/private/shared/revoked/
  deleted/expired/unverified für Treffer, Counts, Highlights und DTOs.
- Update/Delete durch richtige Authority, fremden Actor, falsche Origin,
  unbekannte IRI, Replay und Zustand nach Tombstone.
- Direct→Public, Public→Direct, Public→Private→Public mit neuer Publication-IRI
  und ohne Resurrection von P1.
- Mehrere Shares, Entzug des letzten und eines von mehreren Shares.
- Hundert Restriktionen desselben Origins während Worker-Lease: höchstens eine
  laufende und eine Folgerunde; lokale und fremde Origin-Suche bleiben bereit.
- Crash vor/nach Inboxcommit, lokalem Restriction-Commit, Outboxcommit,
  Remote-Send, Engine-Submit und Scope-Watermark.
- Migration mit Offline-Origin, 404/410, permanent ungültiger Authority,
  Prozessneustart, Restore und gleichzeitigem Neu-Ingest.
- Das SEC-VIS-0-Listenoverlay prüft leeren Federation-Stub, zwei lokal
  materialisierte Trails, Stub→materialisiert und dangling Relation. Die
  ersten drei projizieren ausschliesslich lokale Zahl-/Summenwerte, der letzte
  endet `source_projection_incomplete`; kein Fall erzeugt einen externen
  Projektionsrequest.
- Alte Tenant-Tokens und direkter Legacy-Enginepfad können nach Cutover weder
  private noch unverified/quarantänisierte Remote-Dokumente abfragen.
