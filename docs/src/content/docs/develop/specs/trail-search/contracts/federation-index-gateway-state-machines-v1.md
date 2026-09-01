---
title: Federation-, Index- und Gateway-Zustandsautomaten v1
description: Normativer interner Vertrag für Control Plane, Revisionen, Queues, Generationen, Fences, Gateway und Recovery.
editUrl: false
sidebar:
  order: 4
  badge: Entwurf
spec:
  id: STATE1-CONTRACT
  kind: contract
  status: draft
  capability: FOUNDATION
  lastReviewed: '2026-09-01'
---

Status: Normativer Entwurf, 1. September 2026

Dieser Vertrag definiert die persistente Control Plane für den lokal
materialisierten föderierten Trail-Katalog, seine Suchprojektionen,
Indexgenerationen und den Search-Gateway. Er konkretisiert den
[Vertrag der Trail-Suche v1](/develop/specs/trail-search/contracts/trail-search-v1/), den
[Cursor- und Snapshot-Lifecycle aus ADR 0001](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/),
für den Routenradius die angenommene
[GeoJSON-Direct-Entscheidung aus ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
sowie das Zielbild der
[erweiterten Trail-Suche](/develop/specs/trail-search/).
Die Sicherheits- und Publikationsvoraussetzungen aus dem
[Federation-Sicherheits- und Publikationsvertrag v1](/develop/specs/trail-search/contracts/federation-security/)
sind eine normative Voraussetzung dieses Zustandsvertrags. Bei einer
scheinbaren Abweichung gilt die strengere fail-closed Regel; insbesondere darf
ein technischer Gateway-, Index- oder Publisherzustand kein dort noch offenes
Produktionsgate ersetzen.

Die Schlüsselwörter **MUSS**, **DARF NICHT**, **SOLL** und **KANN** sind
normativ gemeint. Collection- und Feldnamen sind Teil dieses internen V1-
Vertrags. Eine Implementierung darf sie nur ändern, wenn Migration, Diagnose,
Recovery und alle hier beschriebenen Invarianten äquivalent erhalten bleiben.

## 1. Geltungsbereich

Der Vertrag gilt für jede Capability, die den materialisierten föderierten
Suchbestand, generationengebundene Indexprojektionen oder den Search-Gateway
verwendet. Er legt dafür fest:

- welche internen PocketBase-Collections die Control Plane bilden;
- wie Fachmutation, Katalogrevision, Dirty-Queue, Change-Log, Tombstone und
  Restriction-Fence atomar zusammenhängen;
- welche Federation-, Projektions-, Generationen-, Publisher-, Fence- und
  Gatewayzustände existieren und welche Transitionen erlaubt sind;
- welche Zeitquelle und welche Regeln für TTL, Refresh, Grace und Clock-Skew
  gelten;
- wie Shadow-Build, Cutoff-Replay, Epoch-Publikation, Generationstausch,
  Rollback und Retention funktionieren;
- wie die append-orientierte Route-Shard-Zuordnung, ihre persistente
  Manifesthistorie und die generationengebundene physische Shardfamilie
  funktionieren;
- wie Crash-Recovery und öffentliche Fehlerantworten aus internem Zustand
  folgen;
- wie PocketBase-Restore, Anti-Rollback-Checkpoint und Ressourcenretention
  fail-closed zusammenwirken.

Nicht Gegenstand dieses Vertrags sind die fachlichen Felder des
`trail_hit_v1`, die konkrete Meilisearch-Querysyntax, die endgültige
Overlay-Ausführung und die fachliche ActivityPub-Darstellung. Diese werden von
ihren jeweiligen Domainverträgen festgelegt, dürfen die hier gesetzten
Konsistenz- und Sicherheitsinvarianten aber nicht abschwächen.

Davon ausgenommen ist die für Zustandskonsistenz notwendige Topologie des
`route_radius_ux_v1`: ADR 0002 ist für GeoJSON Direct, das S50-/r100-Profil,
höchstens 1.000 Trails pro Route-Index und eine logisch federierte Query über
die vollständige Shardfamilie massgeblich. Dieser Vertrag spezifiziert die
persistente Zuweisung, Generationenbindung und atomare Auswahl dieser Familie
und darf die Entscheidung nicht durch Repacking, einen zweiten Datenpfad oder eine
anwendungsseitig zusammengefügte Teilabfrage umdeuten. Die bytegenaue
Meilisearch-Requestsyntax bleibt Aufgabe des Engineadapters.

Die Authority-, Public-Adressierungs-, Replay-, Origin-, Tombstone-/
Neupublikations- und
Objektordnungsregeln sind Eingaben aus `FED0`; dieser Vertrag erfindet sie nicht
neu.
FED0 muss insbesondere für zwei verschiedene autorisierte Effekte desselben
Objekts einen maschinenprüfbaren Vergleich `equal`, `after`, `before`,
`concurrent` oder `unknown` sowie einen Currentness-Beleg definieren. Lokale
Empfangsreihenfolge und ungebundene Remotezeit sind dafür unzulässig. Bis FED0
als eigenes Artefakt extrahiert ist, ist für die übrigen Regeln der Federation-
Abschnitt im
[Konzept der erweiterten Trail-Suche](/develop/specs/trail-search/shared-invariants/#6-search-context-counts-und-federation)
die benannte vorläufige Quelle. Die betroffenen Teile dieses Vertrags dürfen
nicht als abgeschlossen angenommen werden, solange dort eine von ihren Guards
benötigte FED0-Entscheidung offen ist.

Diese offene FED0-Abhängigkeit blockiert nur Capabilities, deren korrekte oder
sichere Ausführung den jeweils noch undefinierten Guard benötigt. Rein lokale
UI-Slices und bestehende lokale Filter, die weder föderierte Treffer noch
föderierte Counts, ACL-/Publication-Effekte oder einen verbreiterten Suchpfad
ausstellen, hängen nicht pauschal von der Annahme dieses Vertrags ab. Konkrete
Reihenfolge, Umsetzungsstatus und Beitragszuschnitt gehören in
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/),
nicht in diesen Zustandsvertrag.

Produktive föderierte Treffer, Counts oder verbreiterte Suchpfade bleiben bis
zu den jeweils festgelegten Gates gesperrt. Die Gate-Namen entsprechen exakt
den Capability-Gates `SEC-VIS`, `SEC-AUTH`, `SEC-DUR`, `SEC-SCOPED` und
`SEC-TARGETED` des
[Federation-Sicherheits- und Publikationsvertrags](/develop/specs/trail-search/contracts/federation-security/);
nummerierte Lieferpakete oder PR-Bezeichnungen sind keine zusätzlichen Gates.
Für die hier modellierte Control Plane unterscheidet V1 verbindlich:

- `SEC-GLOBAL` ist nur ein dauerhafter Circuit-Breaker. Er darf eine ganze
  Suchfamilie konservativ sperren, wenn schon ihre Scopebestimmung nicht
  vertrauenswürdig ist, beweist aber niemals produktionsfähige Federation.
- `SEC-SCOPED` ist das Produktionsminimum. Es schützt mindestens `local`,
  `all_federated`, Origin-, Actor-, ACL- und Publication-Scopes vor Ranking,
  Pagination, Total, Facetten, Histogrammen und Highlights.
- `SEC-TARGETED` ergänzt additiv exakte Trail-, Publication- und ACL-Guards zur
  besseren Verfügbarkeit. Es ersetzt weder `SEC-SCOPED` noch dessen Actor- und
  Origin-Massenwiderrufe.

Eine Restriction erhöht im linearisierenden DB-Commit die
`desired_revision = max(old, R)` jedes betroffenen Scope-Ziels. Ein laufender
Worker baut seinen Claim zu Ende und höchstens eine anschliessende Runde bis
zur neueren Sollrevision; Ereignisbursts werden nicht als serielle
Indexvollrunden ausgeführt. Kann ein bestimmter Scope nicht sicher abgezogen
werden, bleibt genau dieser Scope gesperrt. Nur eine unzuverlässige
Scopebestimmung eskaliert zu `SEC-GLOBAL`.

### 1.1 Grenze zu Legacy-, Reindex- und Migrationsowner

Ein vor STATE1 aktiver lokaler Suchindex gehört einem eigenständigen Legacy-
beziehungsweise Migrationsowner. Dessen Reindex, Vergleich, Task-Wait,
Startupmodus, manuelle Recovery, Swap- und Rollbackverfahren sind ausdrücklich
nicht Teil dieses Vertrags. STATE1 liest oder mutiert weder dessen interne
Epochen und Attestierungen noch dessen Audit- oder Run-Dateien.

Die folgende Grenze ist ausdrücklich **kein** Lieferartefakt von IDX0 und
keine Vorbedingung für dessen normalen Legacybetrieb. Erst der Baustein IDX3,
der den Generationsbuild und Pointer-Cutover tatsächlich benötigt, liefert
einen temporären Legacy-Handoff-Adapter. Dieser Adapter wird vor dem
Handoff-Preflight in das Legacydeployment aufgenommen, initialisiert seine
Snapshots aus PocketBase und umfasst danach jeden projektionsrelevanten
Source-Commit. Ein partieller Authority-CAS beendet ihn nur für die dabei
übergebenen Rollen. Für jede extern verbleibende Rolle und deren Anteil an
gemischten Fachcommits bleibt er aktiv, bis auch diese Rolle irreversibel an
STATE1 übergeben oder ausdrücklich aus der Handoffmenge retired wurde. Erst
danach darf der Adapter entfernt werden. Ohne installierten Adapter beginnt
STATE1 keinen Handoff; SRCH0, IDX0 und SRCH-COMP bleiben davon unabhängig.

Die Aktivierung des Adapters nimmt die kanonischen Source-/Authority-Locks,
schreibt initiale Snapshots, Authorityrecords und den aktiven Dispatchmarker
in einem PocketBase-Commit und gibt die Locks erst danach frei. Jeder spätere
projektionsrelevante Fachcommit prüft diesen Marker und aktualisiert seinen
Boundary-Anteil in derselben Transaktion. Damit existiert zwischen
Initialsnapshot und laufender Erfassung kein ungeschütztes Writefenster.

Der IDX3-Adapter MUSS für einen möglichen Handoff genau die geschlossene,
transaktional lesbare Grenze `LegacyAuthorityBoundaryV1` bereitstellen. Die
hier verwendeten `SafeRevisionV1`- und `DigestV1`-Formen sind in Abschnitt 3.1
beziehungsweise beim Swaprecord definiert; jedes Objekt auf jeder Tiefe hat
`additionalProperties: false`:

```text
LegacyAuthorityBoundaryV1 = {
  "schema": "legacy_authority_boundary_v1",
  "records": [{
    "logical_uid": non-empty text,
    "authority": "legacy" | "state1",
    "authority_version": SafeRevisionV1,
    "source_revision": SafeRevisionV1,
    "source_snapshot_digest": DigestV1,
    "legacy_resource_set": [{
      "engine_scope": non-empty text,
      "physical_index_id": non-empty text
    }],
    "legacy_resource_set_digest": DigestV1,
    "operation_state": "ready" | "running" | "recovery_required",
    "authority_head_key": null | non-empty text,
    "authority_pointer_version": null | SafeRevisionV1
  }]
}

LegacySourceSnapshotPreimageV1 = {
  "schema": "legacy_source_snapshot_preimage_v1",
  "logical_uid": non-empty text,
  "source_revision": SafeRevisionV1,
  "projection_contract_revision": non-empty text,
  "entries": [{
    "source_key": non-empty text,
    "projection_input_b64u": canonical unpadded base64url text
  }]
}
```

Alle aufgeführten Properties sind verpflichtend; `null` ist nur an den beiden
ausdrücklich so typisierten Authoritybindings zulässig. `records` ist eindeutig
und nach UTF-8-Bytes von `logical_uid` sortiert. Zu
jedem Record stellt der IDX3-Übergangsadapter das durch `source_snapshot_digest`
inhaltsadressierte unveränderliche `LegacySourceSnapshotPreimageV1` bereit;
sein `logical_uid` und seine `source_revision` stimmen mit dem Record überein.
Seine `entries` sind eindeutig und nach UTF-8-Bytes von `source_key` sortiert.
`projection_input_b64u` enthält die exakten Bytes des durch
`projection_contract_revision` bezeichneten, vollständigen
Projektionsinputs. Dadurch umfasst der Snapshot auch alle aufgelösten
denormalisierten Abhängigkeiten und die Dokumentmitgliedschaft, ohne deren
owner-spezifisches Innenschema in STATE1 nachzubauen.

`source_snapshot_digest` ist der `DigestV1` über das zu derselben Rolle
gehörende vollständige `LegacySourceSnapshotPreimageV1`. Der Digest des
Ressourcensets hat exakt folgende Preimage, wobei `resources` bytegleich
`legacy_resource_set` ist und nach `(engine_scope, physical_index_id)` in
UTF-8-Byteordnung sortiert sowie als Tupel eindeutig ist:

```text
LegacyResourceSetDigestPreimageV1 = {
  "schema": "legacy_resource_set_digest_preimage_v1",
  "logical_uid": non-empty text,
  "resources": [{
    "engine_scope": non-empty text,
    "physical_index_id": non-empty text
  }]
}
```

Der IDX3-Übergangsadapter hält je Rolle genau einen atomar mit PocketBase
aktualisierbaren Authorityrecord in dieser Form. Solange `authority = legacy`
gilt, sind `authority_head_key` und `authority_pointer_version` beide `null`;
bei `authority = state1` sind beide gesetzt und binden den kanonischen
STATE1-Head. Jede Recordänderung erhöht `authority_version` um genau eins.
Jeder projektionsrelevante Source-Commit erhöht zusätzlich `source_revision`
um genau eins und ersetzt im selben PocketBase-Commit Snapshot und Digest;
ein Ressourcenset- oder Operationswechsel ersetzt seine vollständigen Felder
im selben Authorityversions-CAS. Ein Überlauf ist fail-closed.

Source-Mutationen, `operation_state`-Kanten, Ressourcensetänderungen und der
finale Authority-CAS verwenden dieselben je Rolle kanonisch geordneten
Source-/Authority-Locks. Nur `operation_state = ready` ist terminal ruhig und
handofffähig. Der Handoff liest die kleinen Authorityrecords im konsistenten
PocketBase-Schnitt, löst danach ausschliesslich die digestgleich dazu gehörenden
unveränderlichen Source-Snapshots auf und validiert ihre vollständige Preimage;
ein bloss behaupteter Digest ohne rekonstruierbaren Inhalt ist ungültig.

Wie der IDX3-Übergangsadapter diese Aussagen beweist, bleibt sein
Implementierungsvertrag.
Insbesondere macht STATE1 weder einen Legacy-Taskhead noch einen
Dokumentvollvergleich oder einen korrekturspezifischen Aktivierungsbeleg zu
seinem dauerhaften Readinessinput.

STATE1 übernimmt eine unterstützte Rollenmenge nur mit folgendem eigenen
`legacy_authority_handoff_v1`-Protokoll:

1. Alle Rollen der deklarierten Handoffmenge sind vom Ziel-Entity-/Querymodell
   und vom neuen Membermanifest vollständig abgedeckt. V1 deckt als
   Suchdokumentrolle `trails` ab; nicht abgedeckte Rollen enden vor jeder
   Wirkung mit `state1_handoff_uid_unsupported`.
2. STATE1 liest die externe Authority- und Source-Bindung in einem kurzen
   konsistenten Preflight. Der folgende Build verwendet ausschliesslich die
   über deren Digest aufgelösten unveränderlichen Source-Snapshots. Vor
   `prepared` entsteht keine persistente Buildplanbindung; die Locks werden
   nicht über den Generationsbuild gehalten.
3. STATE1 baut eine vollständige neue Generation ausschliesslich auf
   physischen IDs auf, die von allen Legacyressourcen disjunkt sind. Alle
   eigenen Buildtasks müssen nach den Attempt- und Taskautomaten dieses
   Vertrags terminal erfolgreich und owner-bestätigt sein.
4. Unmittelbar vor dem Pointer-CAS nimmt STATE1 Source- und Authority-Locks in
   kanonischer Reihenfolge, liest Source-, Authority- und CAS-Version erneut
   und hält diese Locks nur bis zum Commit. Jede Abweichung zur prozesslokalen
   Preflight-Bindung quarantänisiert die staged Generation und beginnt mit
   neuem Snapshot einen vollständigen Neubau. Für eine noch extern autoritative
   Rolle besitzt V1 keinen Cutoff-/Replayvertrag.
5. Genau eine PocketBase-Transaktion publiziert den ersten
   `search_generation_head`, bindet den neuen Pointer und setzt die Authority
   aller deklarierten Rollen auf `state1`. Alle Rollen gewinnen oder keine.
6. Nach Commit routet ausschliesslich STATE1 auf die neue Generation. Die
   Legacyressourcen sind keine Fallback- oder Readinessquelle und bleiben bis
   zum vom ehemaligen Owner verantworteten Drain beziehungsweise Cleanup
   physisch isoliert.

Ein Handoff adoptiert nie einen aktiven Legacyindex als STATE1-Generation und
es existiert keine Authority-Rückgabekante. Recovery und Rollback verwenden
danach ausschliesslich neue, durch STATE1 registrierte Generationen. Ein
innerhalb der deklarierten Menge halb gebundener Zustand ist
`search_recovery_required` und darf weder durch einen zweiten Handoff noch
durch Legacy-Routing kompensiert werden.

Das Handoffprotokoll ist absichtlich unabhängig von fachlichen
Bestandskorrekturen, Listenprofilen oder einem bestimmten vorangegangenen
Reindexlauf. Falls ein Release solche Voraussetzungen besitzt, bindet dessen
Releaseowner sie außerhalb von STATE1; der Zustandsautomat benötigt nur die
oben geschlossene Source-/Authoritygrenze und seine eigene vollständige
Zielgeneration.

Normale STATE1-Readiness beginnt nach dem ersten atomaren Head-/Pointercommit.
Falls dieser Commit zugleich einen Authority-Handoff ausführt, sind von da an
`search_generation_head`, Generation, Snapshot, Publisher-, Delivery-,
Security- und Restorezustand dieses Vertrags allein autoritativ. Ein externer
Auditverlust oder eine spätere Legacy-Task verändert diesen Zustand nicht;
letztere kann wegen der physischen ID-Isolation ausschliesslich eine
abgelöste Ressource treffen. Ein Greenfield-Bootstrap ohne Legacyowner erreicht
dieselbe Readinessgrenze durch seinen normalen Bootstrap-Swap.

## 2. Architekturentscheidung: PocketBase-native Control Plane

Die einzige persistente Control-Plane-Wahrheit ist die bestehende
PocketBase-SQLite-Datenbank. Nach einem Authority-Handoff sind
`search_generation_head` und die nachfolgenden STATE1-Records exklusiv
autoritativ. Ein externer Legacy- oder Migrationsrecord bleibt höchstens
historischer Übergangsbeleg und ist weder Routing- noch Readinessquelle.
Dieser Vertrag führt weder eine zweite SQLite-Datei noch einen eigenen
Datenbankprozess oder eine parallele Persistenzschicht ein. Meilisearch und
spätere räumliche Suchengines bleiben vollständig abgeleitete Systeme.

```text
PocketBase / dieselbe SQLite-Datenbank
  Fachcollections
  + Federation-Snapshots und Objektköpfe
  + Katalogrevision, Change-Log und Dirty-Queue
  + Fences, Generationen, Engine-Attempts und Retention
                         |
                         | bestätigte, wiederholbare Nebenwirkungen
                         v
              generationenspezifische Suchindizes
                         |
                         v
                  strukturierter Gateway
```

Alle nachfolgend genannten Datensätze werden standardmässig als interne
PocketBase-Base-Collections per normaler Wanderer-Migration angelegt. `Intern`
bedeutet dabei nicht `collection.system = true`; die Collections bleiben normal
migrier- und rückbaubar. Für `listRule`, `viewRule`, `createRule`, `updateRule`
und `deleteRule` gilt jeweils PocketBase-`null`, niemals die leere und damit
öffentliche Regel: Weder Browser noch normale PocketBase-API-Clients dürfen sie
direkt lesen oder schreiben. Ein begrenzter Admin-Diagnoseendpunkt KANN daraus
allowlistete DTOs bilden.

Gezielte SQL-Anweisungen gegen diese Collection-Tabellen sind innerhalb einer
von PocketBase verwalteten Transaktion erlaubt und für atomare Inkremente,
Compare-and-swap, Claims und High-Watermark-Maxima vorgesehen. Sie laufen nur
über einen gemeinsamen typisierten Transition-Service. Dieser erzwingt
Safe-Integer-, Zustands- und Maximalguards und aktualisiert eigene
`transitioned_at`-/`version`-Felder; ein SQL-CAS darf sich nicht auf von
PocketBase-Hooks automatisch gepflegtes `updated` verlassen. Daraus folgt keine
zweite Speicherabstraktion. Schema, Migration, Backup und Restore bleiben
PocketBase-zentriert.

Change-Log, Dirty-Zeilen, Fences und Tombstones referenzieren möglicherweise
gelöschte Fachrecords ausschliesslich über stabile Textschlüssel. Eine
PocketBase-Relation mit Cascade darf den Korrektheits- oder Retentionsbeleg
nicht zusammen mit dem Fachrecord entfernen. Relationsfelder sind nur zwischen
gemeinsam retenierten Control-Plane-Records zulässig.

Rohe, nicht als Collection registrierte SQLite-Tabellen benötigen ein eigenes
ADR und einen durch Messung oder nicht anders erfüllbaren Constraint belegten
Grund. Sie sind kein Bestandteil von V1. Ebenso DARF ein normaler
`OnRecordAfter*Success`-Hook nicht die einzige Quelle einer Revision, eines
Tombstones, Dirty-Jobs oder Fence sein, weil dieser Hook erst nach dem
Fachcommit läuft.

Externe Netzwerkaufrufe und Engine-Writes finden nie innerhalb der SQLite-
Transaktion statt. Sie konsumieren ausschliesslich bereits committed Control-
Plane-Zustand.

Ein normaler Wanderer-Start liest und validiert Route-Shard-Allocator,
Assignment- und Manifestzustand und übernimmt die generationengebundenen
physischen Indizes unverändert. Er darf weder live Trails neu verteilen noch
die Route-Indexfamilie pauschal löschen oder vollständig neu aufbauen. Mit
Eintritt in STATE1 läuft jeder Backfill, Schema-/Projektionswechsel oder
Vollrebuild als registrierte Shadowgeneration mit neuen physischen IDs und den
in Abschnitt 8 festgelegten Cutoff-, Watermark- und Swap-Gates. Legacy- und
Migrationsowner vor diesem Eintritt bleiben von Abschnitt 1.1 abgegrenzt.

Die beiden folgenden zusätzlichen Infrastruktur-Ausnahmen sind keine parallele
Datenbank: Root-Keymaterial liegt wie üblich in der Secret-Ablage, und ein monotoner
Anti-Rollback-Checkpoint liegt ausserhalb des zu sichernden PocketBase-Backups.
Dieser Checkpoint enthält keine Fachpayload, sondern nur eine global monotone
`checkpoint_sequence`, Restore-Epoch, Control-Plane-Inkarnation, höchste
bestätigte Securityrevision, höchsten Cursor-HWM und das kanonische
Ressourcen-/HWM-Manifest. Das Manifest liegt entweder direkt im Checkpoint oder
als unveränderliches inhaltsadressiertes Objekt, dessen Referenz und Digest der
Checkpoint atomar bindet; ein Digest ohne rekonstruierbaren Inhalt genügt
nicht. Der externe Speicher muss einen Compare-and-swap auf die vollständige
Checkpointidentität unterstützen. Dieser Anker ist nötig, weil ein Restore
derselben SQLite-Datei sonst Fences und bereits zugesagte HWM unbemerkt
zurückdrehen könnte. Ein zweiter SQLite-Dienst oder eine zweite fachliche
Wahrheit entsteht dadurch nicht.

## 3. Gemeinsame Typen, Zeit und Ordnung

### 3.1 Sichere Revisionen

`checkpoint_sequence`, `catalog_revision`, `content_epoch`,
`security_revision`, `snapshot_set_revision`, Route-Shard-`manifest_revision`, `publisher_term`,
`pointer_version`, `lease_token` und Watermarks verwenden den internen Typ
`SafeRevisionV1`:

```text
ganze Zahl, 0 <= Wert <= 9_007_199_254_740_991  # 2^53 - 1
```

Damit bleiben Werte in PocketBase-`number`, Go und JavaScript exakt. Das
PocketBase-Feld verwendet `onlyInt = true`, `min = 0` und
`max = 9007199254740991`. Wenn `0` ein gültiger Initialwert ist, wird dieser
nicht durch eine `required`-Semantik ausgeschlossen. An einer
JSON-API werden Katalogrevisionen, Epochs und Watermarks weiterhin als
kanonische Dezimalstrings ohne führende Nullen übertragen. Interne
PocketBase-Records speichern sie als Zahl und validieren die Safe-Integer-
Grenze. Ein Überlauf DARF NICHT umbrechen; er versetzt katalogverändernde
Schreibpfade in einen administrativen, fail-closed Wartungszustand.

Ein nackter Zahlenwert ist nicht bei jedem Typ eine globale Identität. V1 bindet
die Ordnungsräume ausdrücklich:

- `checkpoint_sequence`, `restore_epoch` und `security_revision` steigen über
  Restores hinweg global und werden extern verankert;
- Katalog- und Federation-Snapshotset-Revisionen gehören zu
  `(control_plane_incarnation, revision)`. Nach einem Restore darf die rohe
  Katalogzahl unter einem früheren, nicht sicherheitsrelevanten Stand liegen,
  aber nie unter der externen Securityrevision; alte und neue Inkarnationen
  werden weder verglichen noch zusammengeführt;
- eine Route-Shard-Manifestrevision gehört zu
  `(control_plane_incarnation, scope_key, manifest_revision)` und darf weder
  mit einer Katalogrevision noch mit dem Manifest einer anderen Inkarnation
  verglichen werden;
- eine Content-Epoch gehört zu `(generation_key, content_epoch)`, eine
  Pointerversion zu `(control_plane_incarnation, contract, entity_kind,
  pointer_version)`;
- der Writer-Fence ist das lexikographische Tupel
  `(restore_epoch, credential_epoch, publisher_term)`. Der rohe Term steigt
  monoton innerhalb dieses Tupelraums, ist allein aber keine
  Restore-übergreifende Identität;
- Lease-Token gehören zu Record-ID plus `queue_incarnation` beziehungsweise
  dem entsprechenden Inboxrecord.

Cursor, Attempt-Guards, Diagnose- und externe DTOs dürfen einen gescopten Wert
nie ohne seine Identität vergleichen oder als Idempotency-Key verwenden. Ein
Restore invalidiert alte Cursor über die neue Control-Plane-Inkarnation und
verwendet neue Generation-/Engine-Identitäten; er darf gleiche rohe Zahlen
nicht als denselben historischen Stand ausgeben.

Innerhalb genau einer PocketBase-Datei ist der Singleton-
`search_control_plane_restore_state` der Namensraumowner aller nicht eigens
qualifizierten Katalog-, Change- und Scope-Revisionszeilen. V1 hält nie zwei
abfragbare Control-Plane-Inkarnationen in derselben Dateilinie; deshalb muss
die Inkarnation nicht in jeder solchen Collectionzeile redundant gespeichert
werden. Ein Restore validiert und übernimmt den beweisbaren Backup-Prefix
unter der neu geminteten Dateilinie, bevor irgendein neuer Commit erlaubt ist,
und quarantänisiert alle alten Generation-/Engine-Identitäten. Jede
dateiübergreifende Referenz – insbesondere Cursor, Checkpoint, Diagnoseexport
und Restorebeleg – trägt die Inkarnation dagegen ausdrücklich. Ohne den
Singleton-Guard darf kein roher Revisionswert gelesen, verglichen oder als
Unique-/Idempotency-Identität exportiert werden.

Eine `catalog_revision` bezeichnet genau einen committed fachlichen
Transaktionsschnitt innerhalb einer Control-Plane-Inkarnation. Alle
Change-Einträge dieses Schnitts teilen die Revision
und unterscheiden sich durch einen bei `0` beginnenden lückenlosen `ordinal`.
Die nächste committed Revision ist exakt die vorige plus eins. Ein Rollback der
SQLite-Transaktion verbraucht keine sichtbare Revision.

PocketBase-`number` ist intern `NOT NULL` und kann einen gültigen Nullwert nicht
von „nicht gesetzt“ unterscheiden. Die in den Schematabellen verwendete
Notation `OptionalSafeRevisionV1` expandiert deshalb immer in genau zwei
PocketBase-Felder: `<name>` als `SafeRevisionV1` und `<name>_set` als
immer vorhandenes Boolean mit SQL-Default `false`. In PocketBase v0.38 bleibt
für dieses Bool `Required = false`, weil `BoolField.Required = true` dort
„muss wahr sein“ und den benötigten Zustand `_set = false` verbieten würde. Der
Transition-Service validiert das Feldpaar auf jeder Schreibkante. Nur wenn
`<name>_set = true`, besitzt die Zahl Semantik. Ein nackter Zahlen-Sentinel ist
verboten.

### 3.2 Zeitquelle und Intervalle

Normative Zeitvergleiche verwenden die einmal pro Transaktion beziehungsweise
Request gelesene UTC-Zeit der PocketBase-Datenbankinstanz. Von Clients,
ActivityPub-Objekten oder HTTP-Headern behauptete Zeiten sind Belegdaten und
niemals die lokale Ablaufuhr.

An öffentlichen und internen JSON-Grenzen werden Zeitpunkte als UTC-RFC-3339
mit `T` und Millisekunden übertragen. PocketBase-`date`-Felder verwenden intern
sein kanonisches UTC-Layout `2006-01-02 15:04:05.000Z`; der typisierte
Transition-Service konvertiert an der Grenze. Gemischte Stringformate dürfen
nie lexikographisch verglichen werden. Ein optionales `date?` ist intern das
leere PocketBase-Datum und erhält ausserhalb der Collection keinen
Zeitpunktwert. Gültigkeitsintervalle sind halboffen:

```text
Federation-Snapshot auffindbar  <=> federated_object.state = discoverable
                                  AND federated_object.current_snapshot gesetzt
                                  AND current_snapshot.federated_object = federated_object
                                  AND current_snapshot.verification_state = verified
                                  AND as_of < federated_object.discoverable_until

Suchkontext gültig              <=> now < valid_until
```

Bei Gleichheit ist der Zustand abgelaufen. Clock-Skew erweitert weder
`discoverable_until` noch `valid_until`.

Eine Gateway- oder Workerinstanz, deren Uhrgesundheit ausserhalb des
zugesagten Skew-Bounds liegt, darf keine neuen Kontexte ausstellen, Cursor
fortsetzen, Federation-Nachweise annehmen oder zeitbasierte GC freigeben. Sie
bleibt bis zu einer bestätigten Uhrkorrektur nicht ready.

V1 unterscheidet ausdrücklich:

- **TTL:** fachliche Obergrenze der Discoverability beziehungsweise des
  Suchkontexts;
- **Refresh-Vorlauf:** Zeitpunkt für den Versuch einer Erneuerung vor der TTL;
- **In-flight-/Resource-Grace:** ausschliesslich zusätzliche Retention bereits
  zugesagter Ressourcen;
- **Inbound-Clock-Skew:** Toleranz bei Signatur- und Replayprüfung entfernter
  Nachrichten.

Es gilt:

```text
context_resource_retention_grace_s
  >= max_search_request_deadline + promised_clock_skew
```

Grace macht einen abgelaufenen fachlichen Zustand nie wieder gültig.

### 3.3 IDs, Hashes und Zustände

Fachliche Zielschlüssel sind kanonische Tupel aus `target_kind`, `target_key`
und `component`. Sie dürfen keine übersetzten Labels oder veränderlichen
Engine-UIDs enthalten. Für föderierte Objekte ist `target_key` die kanonische
Objekt-IRI beziehungsweise ein kollisionssicherer, auf sie zurückführbarer
interner Schlüssel.

Payload-, Input-, Settings-, Schema- und Manifesthashes verwenden einen
festgelegten versionierten Algorithmus und kanonische Eingaben. Ein Hash ist
kein Ersatz für den zugrunde liegenden Typ oder dessen Version.

Transitionkritische Zustände sind geschlossene `select`-Enums. Freie JSON-
Objekte dürfen Payload und sichere Diagnosen, aber weder Leaseeigentum,
Watermarks, Guards noch erlaubte Transitionen repräsentieren.

Die wenigen mehrfach verwendeten Selecttypen besitzen in V1 exakt diese
migrationsgebundenen Werte:

| Typ | PocketBase-`SelectField.Values` |
| --- | --- |
| `SearchTargetKindV1` | `trail`, `federated_object`, `federation_scope`, `federated_publication`, `taxonomy`, `search_contract`, `acl_scope` |
| `SearchProjectionComponentV1` | `core`, `overlay`, `geometry` |
| `SearchChangeComponentV1` | `core`, `overlay`, `geometry`, `federation`, `settings` |
| `GeoShardAllocatorStateV1` | `uninitialized`, `backfilling`, `active`, `quarantined` |
| `GeoShardLifecycleV1` | `open`, `sealed` |
| `SearchIndexPartitionKindV1` | `singleton`, `geo_route_shard` |
| `SearchIndexQueryFamilyV1` | `trail_default`, `route_radius_ux_v1` |
| `SearchChangeReasonV1` | `trail_created`, `trail_updated`, `trail_deleted`, `visibility_restricted`, `visibility_expanded`, `federation_created`, `federation_updated`, `federation_deleted`, `federation_expired`, `federation_refreshed`, `taxonomy_changed`, `projection_contract_changed`, `reconcile_repair`, `admin_retry` |
| `SearchIrrelevanceProofKindV1` | `component_input_unchanged`, `target_not_materializable`, `contract_rule_not_applicable` |
| `FederationDeleteReasonV1` | `activitypub_delete`, `remote_gone`, `authority_revoked`, `admin_delete` |
| `FederationObjectOrderRelationV1` | `unassessed`, `equal`, `after`, `before`, `concurrent`, `unknown` |
| `FenceSubjectKindV1` | `trail`, `federated_object`, `federated_publication`, `acl_scope`, `share`, `actor`, `origin` |
| `SubjectVisibilityFrontierStateV1` | `unrestricted`, `restricted`, `clear_pending`, `deleted` |
| `FenceRetirementProofKindV1` | `restriction_converged`, `guard_release_converged`, `guard_release_superseded_by_restriction`, `clear_converged`, `clear_superseded_by_restriction` |
| `FenceSuccessorKindV1` | `clear_before_retirement`, `clear_advance_before_retirement`, `grant_after_retirement`, `guard_release_superseded_by_restriction`, `clear_superseded_by_restriction` |
| `FenceGatewayGuardKindV1` | `exact_target`, `conservative_subject_exclusion` |
| `FenceRetirementArtifactKindV1` | `catalog_commit`, `catalog_change`, `projection_artifact`, `search_contract_artifact`, `federation_scope_version`, `federation_membership_manifest`, `geo_shard_manifest_version`, `content_snapshot`, `content_snapshot_scope`, `content_snapshot_query_family`, `content_snapshot_member`, `generation`, `generation_query_family`, `generation_member`, `visibility_fence`, `catalog_delivery`, `fence_delivery`, `fence_successor`, `visibility_rule_artifact` |
| `EngineNonSubmissionProofKindV1` | `guard_canceled_before_network` |
| `EngineNonAcceptanceProofKindV1` | `synchronous_engine_rejection` |
| `EngineTerminalNoEffectProofKindV1` | `terminal_failed_without_effect`, `terminal_canceled_without_effect` |
| `SearchEntityKindV1` | `trail` |
| `FederationScopeKindV1` | `local`, `all_federated`, `origin`, `actor`, `acl_scope`, `publication` |
| `FederationSecurityModeV1` | `sec_global`, `sec_scoped`, `sec_targeted` |
| `FederatedVerificationStateV1` | `pending`, `verified`, `gone`, `invalid`, `unreachable` |
| `FederatedVisibilityStateV1` | `unknown`, `public`, `restricted`, `none` |
| `FederatedPublicationStateV1` | `staged`, `publishing`, `published`, `tombstoned`, `quarantined` |
| `FederationMigrationOutcomeV1` | `public_verified`, `scoped_verified`, `restricted_unproven`, `gone`, `invalid`, `unreachable` |

Eine Erweiterung dieser Wertemengen ist eine Migration und Vertragsänderung;
„beliebiger registrierter Wert“ ist in einem V1-Select nicht zulässig.

`geo_shard_id` ist ein opaker stabiler Textschlüssel. Seine fachliche Ordnung
kommt ausschliesslich aus `shard_ordinal`; weder die ID noch eine physische
Engine-ID darf dafür lexikographisch interpretiert werden. Der kanonische
Allocation-Key eines Trails ist das Tupel `(local_created_at, trail_key)`, wobei
`local_created_at` das beim ersten lokalen Speichern festgeschriebene DB-Datum
und `trail_key` die stabile lokale Trail-ID ist. Ein veröffentlichtes oder
föderiertes Quelldatum ist kein zulässiger Bestandteil dieses Keys.

## 4. Normative Collections

Alle Collections besitzen zusätzlich die normalen PocketBase-Felder `id`,
`created` und `updated`. Relationsfelder sind intern und nicht expandierbar über
eine öffentliche API. Jeder in der Tabelle genannte Unique-Key MUSS als
eindeutiger Index umgesetzt werden.

Jeder veränderliche Zustandsrecord besitzt ausserdem ein vom gemeinsamen
Transition-Service gepflegtes `transitioned_at` und eine monotone
Safe-Integer-`version` für CAS, auch wenn die folgende Tabelle sie nicht erneut
aufführt. Append-only-Records benötigen keine nachträgliche Zustandsversion;
bei ihnen sind Identität, Digest und Unique-Keys der Idempotenzbeleg.

### 4.1 Revisions- und Projektionscollections

#### `search_catalog_state`

Genau ein Datensatz pro `scope_key`; V1 verwendet `trail-search.v1`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | Unique, unveränderlich |
| `committed_revision` | SafeRevisionV1 | letzte committed Katalogrevision |
| `security_revision` | SafeRevisionV1 | letzte sichtbarkeitsverengende Revision |
| `gateway_mode` | select | `legacy`, `shadow`, `gateway_authoritative`, `gateway_only` |
| `version` | SafeRevisionV1 | CAS-Version des Datensatzes |

#### `federation_security_state`

Genau ein Datensatz pro Search-Scope. Er beschreibt Freigabegates, nicht den
temporären Zustand einzelner Fences.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | Unique, in V1 `trail-search.v1` |
| `sec_global_enabled` | bool | Circuit-Breaker; `true` sperrt föderiertes Serving konservativ und beweist keine andere Stufe |
| `sec_scoped_state` | select | `disabled`, `shadow`, `migrating`, `validating`, `ready`, `quarantined` |
| `sec_targeted_state` | select | `disabled`, `shadow`, `ready`, `quarantined` |
| `migration` | relation? | für `migrating/validating/ready` gebundener Legacy-Migrationslauf |
| `validated_catalog_revision` / `validated_security_revision` | SafeRevisionV1 | höchste durch das Gate bestätigte Revision |
| `proof_hash` | text? | bei `ready` verpflichteter Test-/Coverage-/Migrationbeleg |
| `version` | SafeRevisionV1 | CAS-Version |

`SEC-SCOPED ready` ist Voraussetzung für produktive föderierte Treffer und
Counts. `SEC-TARGETED ready` ist nur additiv. `sec_global_enabled=false` ist
kein positives Gate. Ein Invariantenbruch setzt den Circuit-Breaker und führt
den betroffenen nichtterminalen Scoped-/Targeted-Zustand nach `quarantined`;
er darf nie durch automatisches Zurückschalten auf Legacy Serving kompensiert
werden.

#### `search_control_plane_restore_state`

Genau ein Datensatz pro `scope_key`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | Unique, in V1 `trail-search.v1` |
| `control_plane_incarnation` | text | opak, nach jedem Restore neu und nie wiederverwendet |
| `restore_epoch` | SafeRevisionV1 | muss dem externen Anti-Rollback-Checkpoint entsprechen |
| `state` | select | `healthy`, `recovery_required`, `replaying`, `validating`, `quarantined` |
| `checkpoint_sequence` | SafeRevisionV1 | global monotone Sequenz des lokal committed Checkpointkandidaten; nie zurücksetzen |
| `confirmed_checkpoint_sequence` | SafeRevisionV1 | nur lokaler Cache der extern bestätigten Sequenz; niemals Restore-Wahrheit |
| `checkpoint_security_revision` | SafeRevisionV1 | kumulativer Securitystand des Kandidaten |
| `checkpoint_cursor_hwm` | date? | kumulatives Maximum aller Cursorzusagen des Kandidaten |
| `resource_manifest_hash` | text | Digest der vollständigen kanonischen Ressourcen-/HWM-Liste des Kandidaten |
| `resource_manifest_ref` | text | restorefeste inhaltsadressierte Referenz oder Kennung des inline gespeicherten Manifests |
| `response_guard_sequence` | SafeRevisionV1 | lokaler, pro zur Auslieferung freigegebenem Responsekandidaten atomar erhöhter Linearisierungszähler |
| `restore_detected_at` / `conservative_retain_until` | date? | Recovery- und GC-Grenzen |
| `version` | SafeRevisionV1 | CAS-Version |

Nur `healthy` ist für Gateway, Publisher und GC ready. Die Inkarnation ist Teil
jeder neuen Cursorbindung; ein nach Restore ausgestellter Kontext kann dadurch
keinen vor dem Restore zugesagten Zustand vortäuschen.

Jede Transaktion, die eine extern zu schützende Securityrevision, eine
Cursor-HWM-Zusage oder den Zustand ihres Ressourcenmanifests ändert, erhöht
`checkpoint_sequence` im selben SQLite-Commit und materialisiert dazu den
vollständigen kumulativen Checkpointkandidaten. Die Transaktion gibt dem
Aufrufer ihre Sequenz und die von ihm benötigte Teilmenge als
Bestätigungsanforderung zurück. `confirmed_checkpoint_sequence` darf erst nach
dem Protokoll aus Abschnitt 10.6 per CAS erhöht werden; sein Wert allein
autorisiert keine Response.

#### `search_control_plane_checkpoint`

Durable, unveränderlicher Kandidat für die externe Bestätigung. Unique-Key:
`checkpoint_sequence`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | in V1 `trail-search.v1` |
| `checkpoint_sequence` | SafeRevisionV1 | identisch zum im selben Commit fortgeschriebenen Restore-State |
| `restore_epoch` | SafeRevisionV1 | gebundene externe Restore-Epoche |
| `control_plane_incarnation` | text | gebundene Inkarnation |
| `security_revision` | SafeRevisionV1 | kumulativer bestätigungspflichtiger Securitystand |
| `cursor_hwm` | date? | kumulatives globales Maximum der Cursorzusagen |
| `resource_manifest` | `CheckpointResourceManifestV1` (json) | vollständige kanonische Liste aller noch zugesagten Ressourcen mit Kind, Key, Zustand, HWM, Retentions-Grace und `retain_until`; keine Fachpayload |
| `resource_manifest_hash` | text | Digest der bytegenau kanonisierten Manifestbytes |
| `resource_manifest_ref` | text | deterministische inhaltsadressierte Off-host-Referenz oder Inline-Kennung |
| `state` | select | `prepared`, `confirmed`, `superseded`, `quarantined` |
| `confirmed_at` | date? | lokale Diagnose; externe Wahrheit bleibt der Anchor |
| `version` | SafeRevisionV1 | CAS-Version für Bestätigungsstatus |

Die Manifestzeilen sind nach `(resource_kind, resource_key)` sortiert. Der
Transition-Service validiert die geschlossene, migrationsgebundene
`CheckpointResourceManifestV1`-Struktur; sie ist kein freies Guard-JSON. Sie
enthält jede nicht abgelaufene Zusage sowie einen etwaigen `lost`-Zustand.
Ein Kandidat wird im selben SQLite-Commit wie die von ihm geschützte
Security-/HWM-Änderung angelegt. Felder und Manifestbytes ändern sich danach
nie; nur `state`, `confirmed_at` und `version` dürfen fortgeschrieben werden.
Ein neuerer Kandidat subsumiert alle zu diesem Zeitpunkt noch gültigen Zusagen
älterer Kandidaten.

```text
prepared -> confirmed -> superseded
        \--------------> superseded  # nur durch bestätigten höheren Einschluss
prepared/confirmed ----> quarantined # Digest-/Subsumptionsbruch
```

`superseded` und `quarantined` sind terminal. Ein bloss höherer Sequenzwert
ohne den in Abschnitt 10.6 verlangten Einschlussbeweis erlaubt die direkte
Kante nach `superseded` nicht.

Ein lokaler `superseded`-Kandidat darf erst nach bestätigtem höherem Anchor,
nach dem maximalen Requestdeadline-/Skewfenster und nur dann gelöscht werden,
wenn kein Restore-, Audit- oder Diagnosepin ihn mehr benötigt. Sein externes
Manifestobjekt darf zusätzlich erst entfallen, wenn kein aktueller Anchor und
keine unterstützte Restorekette mehr darauf verweist.

#### `search_catalog_commit`

Append-only-Header eines fachlichen Revisionsschnitts.

| Feld | Typ | Regel |
| --- | --- | --- |
| `revision` | SafeRevisionV1 | Unique |
| `cause_key` | text | Unique, namespaced Idempotency-Key |
| `kind` | select | `local_mutation`, `federation`, `projection_contract`, `expiry`, `security`, `reconcile` |
| `security_relevant` | bool | wahr, wenn der Schnitt Discoverability verengt |
| `change_count` | SafeRevisionV1 | exakt Anzahl zugehöriger Change-Zeilen |
| `cause_digest` | text | unveränderlicher Hash der vollständigen autorisierten kanonischen Command-/Effektabsicht einschliesslich Fachpayload/-fingerprints |
| `change_manifest_digest` | text | Hash über kanonische Change-Header und deren im Commit bekannte unveränderliche Werte |
| `committed_at` | date | einmal gelesene DB-Zeit der Transaktion |

Ein vorhandener `cause_key` mit identischem `cause_digest` ist ein idempotenter
Replay und erzeugt keine neue Revision. Derselbe Key mit anderem
`cause_digest` ist ein
Invarianzfehler und wird quarantänisiert beziehungsweise mit einem sicheren
Konflikt beendet. Der später vervollständigte Upsert-`payload_hash` eines
Projektionsartefakts ist nicht der Idempotenzbeleg des ursprünglichen Commands
und darf `cause_digest` nie ersetzen.

#### `search_catalog_change`

Lückenloses Replay-Log. Unique-Key: `(revision, ordinal)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `catalog_commit` / `revision` | relation/SafeRevisionV1 | gemeinsam retenierter Commit und dessen für geordnete Scans duplizierter exakter Revisionswert |
| `ordinal` | SafeRevisionV1 | lückenlos `0..change_count-1` |
| `target_kind` / `target_key` | `SearchTargetKindV1`/text | kanonisches Ziel |
| `component` | `SearchChangeComponentV1` | geschlossener Change-Komponententyp |
| `operation` | select | `upsert`, `delete`, `invalidate`, `fence` |
| `reason` | `SearchChangeReasonV1` | versionierte Ursache; nur Diagnose/Invalidationsbreite |
| `resolution_state` | select | `pending`, `ready`, `subsumed`, `irrelevant` |
| `projection_version` | relation? | bei `ready/subsumed` optionaler Lieferstand |
| `superseded_by_revision` | OptionalSafeRevisionV1 | nur bei bewiesener Subsumption |
| `irrelevance_proof_kind` | `SearchIrrelevanceProofKindV1`? | genau bei `irrelevant` gesetzt |
| `irrelevance_rule_version` / `irrelevance_proof_hash` | text? | genau bei `irrelevant` gesetzter versionierter Entscheidungs- und Eingabebeleg |
| `tombstone_authority_hash` | text? | bei Federation-/Security-Delete verpflichtend |
| `payload_hash` | text? | bei Tombstone im Commit, bei Upsert einmalig mit der Resolution gesetzt |

Identität, Operation, Zielwerte und ein bereits gesetzter Hash sind nach Commit
unveränderlich. Nur `resolution_state`, `projection_version`,
`superseded_by_revision`, die zuvor leeren Irrelevance-Belegfelder und ein
zuvor leerer Upsert-`payload_hash` dürfen per CAS fortgeschrieben werden.
`irrelevant` verlangt alle drei Irrelevance-Felder im selben terminalen CAS;
in jedem anderen Zustand bleiben sie leer. Der `irrelevance_proof_hash` bindet
Change-/Commit-Digest, Ziel, Komponente, kanonische Vorher-/Nachher-
Inputfingerprints, `irrelevance_proof_kind` und das unveränderliche Artefakt
`irrelevance_rule_version`. Dieses Regelartefakt bleibt mindestens bis zum GC
des Change erhalten. So ist maschinenprüfbar, dass die konkrete Komponente
durch den Fachschnitt nicht beeinflusst wird; `irrelevant` ist kein
allgemeiner Skipmechanismus.

Der Resolution-Automat ist append-orientiert:

```text
— -> pending -> ready | subsumed | irrelevant
```

Die drei Zielzustände sind terminal. `ready` bindet den exakten Tombstone oder
die unveränderliche Projektionsversion, die diesen Change auflöst;
`subsumed` setzt zusätzlich `superseded_by_revision` auf einen Stand nicht
kleiner als die Change-Revision und verlangt dessen vollständigen
Coveragebeleg. `irrelevant` bindet den vollständig rekonstruierbaren
versionierten Komponenten-Entscheidungsbeleg. Jede Kante prüft den
unveränderten Commit-/Change-Digest
und setzt einen zuvor leeren Upsert-`payload_hash` höchstens einmal. Neue
Fachinformation erzeugt einen neuen Change statt einen terminal aufgelösten
Record zurück nach `pending` zu setzen.

#### `search_projection_dirty`

Koaleszierende Arbeitsqueue. Unique-Key:
`(target_kind, target_key, component)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `target_kind` / `target_key` / `component` | `SearchTargetKindV1`/text/`SearchProjectionComponentV1` | Arbeitsziel |
| `queue_incarnation` | text | opak, Unique und bei Neuerzeugung niemals wiederverwendet |
| `desired_revision` | SafeRevisionV1 | Maximum aller noch nicht gebauten Invalidierungen |
| `state` | select | `queued`, `leased`, `retry_wait`, `dead` |
| `claimed_revision` | OptionalSafeRevisionV1 | beim Claim eingefrorenes Ziel |
| `lease_owner` | text? | opake Workerinstanz |
| `lease_token` | SafeRevisionV1 | monoton pro Claim |
| `lease_until` | date? | exklusive Leasegrenze |
| `attempts` | SafeRevisionV1 | Retrydiagnose |
| `retry_not_before` | date? | Backoff mit Jitter |
| `reason_set` | json | kanonische Enum-Menge; Builder baut trotzdem vollständig |
| `last_error_code` / `last_error_digest` | text? | sichere Diagnose ohne Payload/Stacktrace |

#### `search_projection_sweep_state`

Persistenter Keyset-Cursor der Dirty-Sweeper. Unique-Key:
`(scope_key, partition)`; V1 hält je Scope genau die Partitionen `queued`,
`retry_wait` und `expired_lease`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | in V1 `trail-search.v1` |
| `partition` | select | `queued`, `retry_wait`, `expired_lease` |
| `cursor_set` | bool | `false` bezeichnet den Anfang eines Sweepzyklus |
| `cursor_time` | date? | `retry_not_before` beziehungsweise `lease_until`; bei `queued` leer |
| `cursor_desired_revision` | SafeRevisionV1 | letzter inspizierter Keysetwert |
| `cursor_target_key` / `cursor_record_id` | text | restlicher stabiler Tie-Breaker |
| `cycle` | SafeRevisionV1 | steigt bei jedem atomaren Wrap ans Ende→Anfang |
| `version` | SafeRevisionV1 | CAS-Version |

Der Cursor ist nur ein Fairness- und Fortschrittsbeleg, kein Queue-Ack. Er wird
nach einem hart begrenzten Batch per CAS auf das letzte inspizierte Tupel
gesetzt. Am Partitionsende erhöht derselbe CAS `cycle` und setzt `cursor_set`
auf `false`. Ein Crash darf dadurch höchstens doppelt scannen; ein verlorener
Claim bleibt durch Queue-Lease und Record-CAS sicher. Ein Eintrag, der durch
Insert oder Zustandswechsel vor den aktuellen Cursor fällt, wird spätestens im
nächsten vollständigen Zyklus gesehen.

#### `search_projection_version`

Unveränderliches Lieferartefakt eines Projektionsbuilds. Unique-Key:
`(target_kind, target_key, component, built_through_revision)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `target_kind` / `target_key` / `component` | `SearchTargetKindV1`/text/`SearchProjectionComponentV1` | Projektionsziel |
| `built_through_revision` | SafeRevisionV1 | konsistenter DB-Schnitt des Builders |
| `projection_schema` / `input_contract` | text | unveränderliche Versionen |
| `input_fingerprint` | text | Hash aller relevanten Inputs |
| `payload_hash` | text | Hash der kanonischen Enginepayload |
| `payload` | json | typisierte, kanonische Enginepayload |
| `document_key` | text | stabiler Engine-Dokumentschlüssel |
| `retain_until` | date? | früheste lokale GC-Grenze |

Für `component = geometry` enthält die geschlossene Payload mindestens
`search_geometry_status`, `search_geometry_version`,
`search_geometry_coverage`, `search_geometry_simplification_version`,
`search_geometry_max_segment_length_m` und `spatial_contract_version`.
`search_geometry_max_error_m` ist in `route_radius_ux_v1` verboten: Das
S50-/r100-Profil besitzt keine zertifizierte reproduzierbare geometrische
Fehlerobergrenze. Ein solches Feld darf erst ein späterer, separat angenommener
Geometrievertrag mit nachgewiesenem Bound einführen.

`trail_search_projection` und `trail_search_acl_overlay` bleiben die je Ziel
aktuelle fachliche Read-Model- beziehungsweise Pointerebene aus `IDX1`. Eine
erfolgreiche Buildertransaktion aktualisiert deren aktuelle Version und erzeugt
das unveränderliche Lieferartefakt gemeinsam. Das Lieferartefakt macht
Subsumption und einen beweisbaren Shadow-Cutoff möglich, ohne die fachliche
Collection in ein unbeschränktes Temporalmodell umzudeuten.

#### `search_contract_artifact`

Unveränderliches, inhaltsadressiertes Ausführungsartefakt eines Suchvertrags.
Unique-Keys: `artifact_key` und
`(query_family, contract_version, engine_profile_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `artifact_key` | text | opake, aus Contracttyp und `payload_hash` abgeleitete Unique-ID; niemals wiederverwendet |
| `query_family` | `SearchIndexQueryFamilyV1` | durch dieses Artefakt festgelegte Queryfamilie |
| `contract_version` | text | unveränderliche fachliche Vertragsversion, für ADR 0002 exakt `route_radius_ux_v1` |
| `engine_profile_key` | text | unveränderlicher qualifizierter Engineprofil-Key |
| `payload_schema` | text | exakt `SearchContractArtifactPayloadV1` |
| `payload` | `SearchContractArtifactPayloadV1` (json) | vollständige geschlossene kanonische Ausführungs- und Qualitätsdefinition |
| `payload_hash` | text | Digest aus Payloadschema und bytegenau kanonisierten Payloadbytes |
| `recorded_at` | date | einmalige DB-Zeit der Artefaktanlage; kein Vertragsinput |
| `retention_resource` | relation | unveränderlicher 1:1-Owner der Ressource `contract_artifact/artifact_key` |

`payload`, Identitätsfelder, Hash und Ownerrelation sind append-only. Der
Payload ist die rekonstruierbare Wahrheit; ein Quellpfad, nackter Hash oder die
aktuell laufende Binary darf ihn nicht ersetzen. `SearchContractArtifactPayloadV1`
ist eine geschlossene, nach `query_family` diskriminierte Struktur. Für
`route_radius_ux_v1` besitzt sie in V1 exakt die folgenden Felder. Die
fachlichen Profilwerte sind fest; die drei Engine-Identitätswerte `version`,
`qualified_profile_key` und `image_reference` stammen aus dem jeweils
qualifizierten offiziellen Engineprofil. Weitere Felder sind verboten:

```text
execution_path = geojson_direct_final
post_filter_path = none
geometry_types = [LineString, MultiLineString]
simplification = {
  profile: rdp_s50_v1,
  algorithm: ramer_douglas_peucker,
  tolerance_m: 50
}
densification = { kind: geodetic, max_segment_length_m: 5000 }
antimeridian_split = true
predicate = {
  profile: r100_no_padding_v1,
  function: _geoRadius,
  resolution: 100,
  padding_m: 0
}
radius_m = {
  exclusive_min: 0,
  inclusive_max: 100000,
  error: unsupported_radius_for_spatial_contract
}
capability_projection = {
  predicate: route_geometry_distance_lte,
  default_radius_m: 2000,
  max_radius_m: 100000,
  predicate_accuracy: bounded_approximate,
  display_qualifier: none,
  contract: route_radius_ux_v1,
  geometry_profile: route_geometry_search_v1,
  simplification_profile: rdp_s50_v1,
  max_segment_length_m: 5000,
  resolution: 100
}
public_spatial_contract = {
  kind: route_geometry_radius,
  id: route_radius_ux_v1,
  predicate_accuracy: bounded_approximate,
  max_radius_m: 100000,
  product_boundary_tolerance_m: 50,
  resolution: 100,
  geometry_profile: route_geometry_search_v1,
  simplification_profile: rdp_s50_v1,
  max_segment_length_m: 5000
}
accuracy_contract_projection = {
  kind: spatial_boundary,
  id: route_radius_ux_v1,
  predicate: route_geometry_distance_lte,
  plan: direct,
  max_radius_m: 100000,
  product_boundary_tolerance_m: 50,
  resolution: 100,
  geometry_profile: route_geometry_search_v1,
  simplification_profile: rdp_s50_v1,
  max_segment_length_m: 5000
}
max_trails_per_route_index = 1000
query_orchestration = federated_meilisearch_multi_search
application_side_result_merge = false
unsupported_sort = {
  field: route_proximity_m,
  error: unsupported_sort_for_spatial_contract
}
engine = {
  product: meilisearch,
  distribution: official,
  version: 1.53.1,
  qualified_profile_key: meilisearch-1.53.1-geojson-direct-s50-r100-v1,
  image_reference: getmeili/meilisearch@sha256:8d6643d86d71fad6ad3cba92cde7ccfce9e4d6c384bda67598eb553571c32431
}
result_set_computation = exhaustive
count_computation = exact_for_result_set
predicate_accuracy = bounded_approximate
display_qualifier = none
accuracy_profile = {
  qualification_kind: statistical_corpus,
  per_query_error_bound: none,
  geometric_error_bound_m: none,
  neutral_band_m: 50,
  aggregate_min: 0.99,
  aggregate_metrics: [
    raw_recall,
    clear_interior_recall,
    material_precision_outside_neutral_band,
    top_10_coverage,
    nearest_coverage,
    non_empty_success,
    product_page_recall,
    product_page_precision
  ],
  hard_zero_error_classes: [
    acl_semantics,
    text_semantics,
    filter_semantics,
    semantic_false_positive,
    response_truncation,
    warm_result_instability
  ]
}
count_profile = {
  direct_result_tolerance: 0,
  absolute_error_formula: abs(got - oracle),
  relative_error_formula: abs(got - oracle) / max(1, oracle),
  g1_qualified_series: [total],
  p95_relative_error_lte_per_series: 0.01,
  percentile: {
    order: ascending,
    zero_based_position: 0.95 * (n - 1),
    interpolation: linear
  },
  pool_series: false,
  estimated_or_sampled_counts: false,
  diagnostics: [absolute_error, exact_case_match, false_empty],
  false_empty_formula: oracle > 0 && got = 0
}
provenance_refs = [
  decisions/0002-geojson-direct-route-radius.md#1-produktvertrag,
  contracts/trail-search-v1.md#10-total-facetten-histogramme-und-genauigkeit
]
```

Accuracy- und Count-Projektion sind damit selbst Bestandteil der gehashten
Payloadbytes und ohne Markdown rekonstruierbar. `provenance_refs` erklären nur
ihre Herkunft und liefern keine zur Ausführung oder Validierung erforderliche
Semantik. `qualification_kind = statistical_corpus` samt den beiden leeren
Fehlergrenzen stellt klar, dass `bounded_approximate` weder eine Zusage pro
Anfrage noch eine geometrische Schranke bezeichnet. `direct_result_tolerance =
0` verlangt für Total und jeden Ready-Bucket weiterhin exakte Kardinalität der
Direct-Menge; das G1-p95-Budget qualifiziert ausschliesslich die räumliche
Abweichung der Total-Reihe von der Oracle-Menge. Bei Einführung von Facetten
oder Histogrammen bindet deren eigenes Aggregationsprofil die durch SRCH3 mit
einem für spärliche und häufige Buckets eigens festgelegten Budget separat
qualifizierten Options- und Bucketreihen. G1 gibt ihnen weder die
Ein-Prozent-Schwelle noch ein anderes Budget vor. Diese spätere Evidenz ändert
weder das G1-Artefakt noch den öffentlichen Spatial-Vertrag rückwirkend.
`capability_projection`, `public_spatial_contract` und
`accuracy_contract_projection` müssen für alle gemeinsamen Felder bytegleich
sein. Ihre Grenz-, Accuracy-, Resolution-, Geometrie-, Simplification- und
Segmentwerte müssen wiederum exakt den vorstehenden Executionfeldern
entsprechen; insbesondere sind `default_radius_m` und `geometry_profile` keine
aus Code-Defaults ergänzbaren Werte. Capability, Response, Search-Context und
Cursor verwenden ausschliesslich diese gespeicherten Projektionen.
Eine neue offizielle Meilisearch-Version wird nach GeoJSON-Contract,
99-Prozent-UX-Regression und den operativen Upgradeprüfungen als neues
unveränderliches Engineprofil aufgenommen. Sie erzeugt ein neues immutable
Contract-Artefakt und eine neue Indexgeneration, schreibt das bestehende aber
nie um. Bleiben r100, S50, das 99-Prozent-Produktprofil und die übrige
öffentliche Semantik unverändert, erfordert die neue Engineidentität allein
keine Payloadschema- oder öffentliche Vertragsmigration.

Harness-, Dataset- und Executable-Digests sind Qualifikationsprovenienz und
keine fachlichen Runtimefelder. Der Release-Audit archiviert beziehungsweise
identifiziert sie getrennt zusammen mit Rohreport, Image-Digest und dem
qualifizierten `artifact_key`. Diese Provenienz wird weder in `payload_hash`,
Cursor noch Search-Context aufgenommen und kann ein fehlendes oder nicht
qualifiziertes Engineprofil niemals autorisieren.

Das Artefakt besitzt seine eigene `search_context_resource`. Kandidaten dürfen
eine `staged`-Ressource zum Build binden; querybares Serving verlangt
`issuing`. Jeder ausgestellte Route-Kontext erhöht ihren
`issued_valid_until_hwm` gemeinsam mit Generation, Snapshot und Cursor-Key.
`issuing -> retained` ist erst zulässig, wenn keine aktive, querybare,
ready-/swap-fähige oder aufholende Member-/Snapshotbindung mehr neue Kontexte
mit diesem Artefakt ausstellen kann. Löschung verlangt HWM plus Grace sowie
das Ende sämtlicher Build-, Snapshot-, Cursor-, Audit- und Fencepins.

#### `search_geo_shard_allocator_state`

Persistenter Singleton des append-orientierten Route-Shard-Allocators.
Unique-Key: `scope_key`; V1 verwendet `route-radius.v1`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | Unique und unveränderlich |
| `state` | `GeoShardAllocatorStateV1` | Lifecycle des einmaligen Backfills und des normalen Allocators |
| `current_open_shard` | relation | exakt der einzige offene Shard mit der grössten `shard_ordinal`; ab Migration auf den initialen Shard Ordinal `0` gesetzt |
| `manifest_revision` | SafeRevisionV1 | monoton genau einmal je Revisionscommit mit mindestens einer Assignment-/Shardmanifeständerung |
| `manifest_hash` | text | Hash der aktuellen kanonischen Shardmanifestversion |
| `last_allocation_at` / `last_allocation_trail_key` | date?/text? | gemeinsam gesetzter letzter Allocation-Key; leer genau vor der ersten Zuweisung |
| `backfill_cutoff` | OptionalSafeRevisionV1 | nur in `backfilling` gesetzter konsistenter Katalogschnitt |
| `backfill_cursor_set` | bool | `false` bezeichnet den Anfang des initialen Keyset-Scans |
| `backfill_cursor_local_created_at` / `backfill_cursor_trail_key` | date?/text? | gemeinsam gesetzter letzter Backfill-Key |
| `version` | SafeRevisionV1 | CAS-Version |

Der Automat lautet:

```text
uninitialized -> backfilling -> active
uninitialized/backfilling/active -> quarantined
```

Ein normaler Prozessstart darf `active` weder zurücksetzen noch erneut nach
`backfilling` überführen. `quarantined` besitzt in V1 keine automatische
Reparaturkante; ein neuer Backfill oder eine Neuzuordnung verlangt eine eigene
Migration beziehungsweise Vertragsänderung.

Die Migration erzeugt in einem SQLite-CAS den Singleton mit
`state = uninitialized`, `manifest_revision = 0`, leerem Allocation-Key und
`current_open_shard` auf einen echten persistenten Shard Ordinal `0`. Dieser
Shard ist `open`, besitzt `capacity = 1000`, `assigned_slots = 0`, den
kanonischen Empty-Assignment-Digest sowie leere erste/letzte Allocation-
Grenzen. Seine einmal erzeugte opake `geo_shard_id` wird wie jede spätere
Shard-ID persistiert und bei Restore gelesen, niemals aus Ordinal oder Runtime
neu abgeleitet. Dazu existiert genau eine immutable
`search_geo_shard_manifest_version` für Manifestrevision und Katalogrevision
`0`, deren `shard_manifest` genau diesen Shard enthält und deren
`assignment_count = 0` ist. Eine leere Neuinstallation darf nach vollständiger
Prüfung über diesen physischen Initialzustand nach `active` gehen; ein Bestand
mit Trails muss denselben Shard ab Slot `0` füllen und den lückenlosen
Backfill-Cursor abschliessen. Weder ein leeres Shardmanifest noch ein
synthetischer queryseitiger Sentinel ist in V1 zulässig.

#### `search_geo_route_shard`

Persistenter logischer Shardheader. Unique-Keys: `geo_shard_id` und
`shard_ordinal`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `geo_shard_id` | text | opak, unveränderlich und nie wiederverwendet |
| `shard_ordinal` | SafeRevisionV1 | lückenlos ab `0`, alleinige Shardordnung |
| `capacity` | SafeRevisionV1 | in V1 unveränderlich exakt `1000` |
| `assigned_slots` | SafeRevisionV1 | monoton `0..1000`; `0` ist nur für den initialen noch unbelegten offenen Shard zulässig und zählt wie alle Werte jemals zugewiesene Trails, nicht nur live Dokumente |
| `state` | `GeoShardLifecycleV1` | `open` oder terminal `sealed` |
| `first_local_created_at` / `first_trail_key` | date?/text? | gemeinsam leer genau bei `assigned_slots = 0`, sonst unveränderlicher erster Allocation-Key |
| `last_local_created_at` / `last_trail_key` | date?/text? | gemeinsam leer genau bei `assigned_slots = 0`, sonst monotoner letzter Allocation-Key; bei `sealed` unveränderlich |
| `assignment_digest` | text | Digest der lückenlosen geordneten Assignmentzeilen dieses Shards; bei `assigned_slots = 0` exakt der kanonische Empty-Assignment-Digest |
| `opened_revision` | SafeRevisionV1 | Katalogrevision der Shardanlage; beim initialen Shard exakt `0` |
| `sealed_revision` | OptionalSafeRevisionV1 | genau bei `sealed` gesetzt |
| `version` | SafeRevisionV1 | CAS-Version |

Nur der Shard mit der grössten Ordinalzahl darf `open` sein. Er nimmt neue
Assignments ausschliesslich am Ende auf. Die erste Zuweisung überhaupt füllt
Slot `0` des bereits vorhandenen initialen Shards und setzt dessen zuvor leere
Grenzen. Ist der offene Shard beim nächsten Allocation-CAS bereits voll, setzt
derselbe Commit ihn terminal auf `sealed`, erzeugt den nächsten lückenlosen
Shard und weist diesem den neuen Trail als Slot `0` zu.
Ein Delete, eine Expiry oder ein Federation-Sync reduziert `assigned_slots`
nie und öffnet keinen versiegelten Shard erneut.

#### `search_geo_route_assignment`

Append-only-Zuordnung eines lokal materialisierten Trails zu genau einem
logischen Route-Shard. Unique-Keys: `trail_key` und
`(geo_shard_id, slot_ordinal)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `trail_key` | text | stabile lokale Trail-ID, unveränderlich und unabhängig von Remote-IRIs |
| `local_created_at` | date | beim ersten lokalen Speichern aus DB-Zeit festgeschrieben; unveränderlich |
| `geo_shard` / `geo_shard_id` | relation/text | gemeinsam retenierter Shard und dessen duplizierter stabiler Schlüssel |
| `slot_ordinal` | SafeRevisionV1 | lückenlos `0..999` innerhalb des Shards |
| `assigned_revision` | SafeRevisionV1 | Katalogrevision der ersten Zuweisung |
| `assignment_digest` | text | Hash aus Contractversion, Allocation-Key, Shard-ID, Slot und Revision |

Die Zeile und alle ihre Felder bleiben auch nach Trail-Delete oder
Federation-Tombstone unverändert erhalten. Eine Cascade-Relation zum
Fachrecord ist verboten. Ein wieder auftauchender zulässiger lokaler Trail mit
demselben `trail_key` verwendet ausschliesslich diese bestehende Zuweisung;
eine zweite Slotbelegung oder ein Repacking ist eine Invariantverletzung.

#### `search_geo_shard_manifest_version`

Append-only-Version des vollständigen logischen Shardmanifests. Unique-Keys:
`(scope_key, manifest_revision)` und `(scope_key, catalog_revision)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | in V1 `route-radius.v1` |
| `manifest_revision` | SafeRevisionV1 | exakt der im selben Commit am Allocator fortgeschriebene Wert |
| `catalog_revision` | SafeRevisionV1 | Katalogschnitt dieser Topologie-/Assignmentänderung |
| `shard_manifest` | `GeoRouteShardManifestV1` (json) | vollständige kanonisch nach `shard_ordinal` sortierte Shardliste |
| `assignment_count` | SafeRevisionV1 | Summe aller im Manifest gebundenen `assigned_slots` |
| `manifest_hash` | text | Digest der bytegenauen kanonischen Manifestbytes |
| `recorded_at` | date | DB-Zeit der Revisionstransaktion |

`GeoRouteShardManifestV1` ist eine geschlossene migrationsgebundene Struktur.
Sie enthält immer mindestens den initialen Shard Ordinal `0`. Jeder Eintrag
bindet `geo_shard_id`, Ordinal, Capacity, Zustand, Slotzahl, die gemeinsam
optionalen ersten/letzten Allocation-Grenzen und den aus den retenierten
Assignmentzeilen rekonstruierbaren `assignment_digest`; für Slotzahl `0` sind
die Grenzen leer und der Digest kanonisch leer. Der Empty-Assignment-Digest
ist dabei der versionierte Digest über Typ `GeoRouteShardAssignmentsV1` und
die kanonische geordnete leere Assignmentliste `[]`; er ist weder leerer Text
noch ein frei gewählter Sentinel. Das Manifest enthält keine physischen
Engine-IDs. Erzeugt ein Revisionscommit mehrere Erstzuweisungen, bindet genau
eine Manifestversion deren gemeinsamen vollständigen Endstand in der
kanonischen Allocation-Reihenfolge. Eine Content-Epoch verwendet die neueste
Version mit `catalog_revision <= delivered_catalog_revision`; spätere
Assignments dürfen ihre Shardfamilie nicht rückwirkend erweitern.
Manifestversion und benötigte
Assignmentzeilen bleiben mindestens mit jedem sie bindenden Build,
Content-Snapshot und Cursorbeleg reteniert.

#### `search_catalog_delivery`

Persistenter Beleg Change→Generationenmember. Unique-Key:
`(catalog_change, generation_member)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `catalog_change` / `generation_member` | relation/relation | gemeinsam retenierte Belegenden |
| `change_revision` / `change_ordinal` | SafeRevisionV1/SafeRevisionV1 | für Prefixscans dupliziert |
| `state` | select | `pending`, `applied`, `deleted`, `subsumed`, `fenced`, `failed` |
| `projection_version` | relation? | Lieferartefakt bei `applied/subsumed` |
| `engine_task` | relation? | bestätigter Task bei physischer Zustellung |
| `fence_delivery` | relation? | Beleg für `fenced` |
| `proof_digest` | text? | Marker-/Coverage-/Subsumptionsbeleg |
| `acknowledged_at` | date? | lokale DB-Zeit |
| `error_code` / `error_digest` | text? | sichere Diagnose |
| `version` | SafeRevisionV1 | CAS-Version; steigt bei jeder Zustandskante |

Bei Registrierung eines Generationenmembers entstehen beziehungsweise werden
diese Belege für alle Changes nach dessen Baseline deterministisch aufgebaut.
Bei `route_radius_ux_v1` umfasst „alle“ ausschliesslich Changes, deren
persistentes Trail-Assignment genau die `geo_shard_id` des Members trägt,
sowie generationenkompatible globale Settingschanges und die durch ihren vollständigen
Subject-Closure nachweislich auf diesen Shard wirkenden Security-Changes. Ein
Change eines anderen Route-Shards ist für dieses Member weder ein pending
Delivery noch eine Watermarklücke; diese Nichtrelevanz folgt aus dem
retenierten Assignment-/Manifestbeleg und nicht aus freiem Workerermessen.
Ein Wechsel des gebundenen Contract-Artefakts ist dagegen kein auf bestehende
Member zustellbarer Change, sondern verlangt die neue Generation aus Abschnitt
8; er darf deren immutable Family-/Memberbindung nie fortschreiben.
Der vollständige Basisbuild darf Changes bis einschliesslich
`baseline_revision` gesammelt subsumieren, aber nur mit unveränderlichem
Baseline-, Projection-, Coverage- und Markerhash des vollständigen
Generationsstands. `failed` blockiert den Prefix; es ist kein Ack.

Der Delivery-Automat lautet:

```text
— -> pending -> applied | deleted | subsumed | fenced
               |
               +-> failed -> pending
```

`applied`, `deleted`, `subsumed` und `fenced` sind terminale, unveränderliche
Acks. Die jeweilige Kante prüft per State-/Versions-CAS den unveränderten
Change und das Member samt Engine-Inkarnation. `applied` verlangt einen
erfolgreichen physischen Upsert-Task und passenden Payload-/Markerbeleg,
`deleted` einen erfolgreichen Delete- beziehungsweise gebundenen
Nichtvorhandenseinsbeleg, `subsumed` einen vollständigen Baseline- oder
späteren Coveragebeleg und `fenced` einen Security-Change samt referenzierter
`search_visibility_fence_delivery` in `applied` oder wirksamem
`query_fenced` **zum Zeitpunkt dieses Acks**. Wird der zugehörige Fence später
`retired`, trägt dessen Retirement-Beleg aus Abschnitt 9 die fortdauernde
Restriction beziehungsweise den vollständig projizierten Clear-Nachfolger.
Ein historisches terminales `search_catalog_delivery.state = fenced` bleibt
für den geschlossenen Prefix deshalb nur gültig, wenn entweder der Fence noch
wirksam ist beziehungsweise physisch `applied` wurde oder sein rekonstruierter
`retirement_proof_hash` genau diese Delivery und einen memberabdeckenden,
zweigkorrekten Abschluss umfasst. Dieser Abschluss ist entweder die terminale
`applied/subsumed`-Restriction beziehungsweise der Clear desselben Members,
die strikt spätere mindestens ebenso enge Successor-Fence-Kette oder bei einer
nur noch cursorquerybaren versiegelten Altgeneration ein
`guard_release_converged`-/Supersession-Beleg, der für genau dieses Member die
normale Ziel-/Successor-Epoch sowie alternativ dessen physische
Beobachtungsäquivalenz, irreversible Queryentfernung oder den retenierten
`guard_release_revision`-/`clear_revision`-Stale-Anker bindet. Ein blosses
Retirement ohne diesen membergenauen Abschluss trägt den Prefix nicht weiter.
Kein anderer Übergang darf einen Watermark erhöhen.

`pending -> failed` hält Fehlerklasse und -digest fest. Nur ein auditierter
Scheduler-/Recovery-CAS darf bei weiterhin identischem Change, Member und
Zielintent `failed -> pending` setzen; er erhöht `version`, bindet den neuen
Attempt/Task und leert alte Ackfelder. Dadurch kann ein verspäteter Worker
keinen Retry überschreiben. Ein nachträglich gefundener Erfolgsbeleg wird erst
nach dieser Retrykante durch eine der normalen terminalen Kanten übernommen.

### 4.2 Federation-Collections

#### `federation_ingest_event`

Durable Inbox mit getrennter Transport- und Fachidempotenz. `ingest_key` ist ein
namespaced Digest der vollständigen bytegenauen signierten Eingabe und kann
daher von einer nur behaupteten Activity-ID nicht reserviert werden. Erst nach
erfolgreicher Authorityprüfung setzt der Verifier den fachlichen
`verified_effect_key`; für nichtleere Werte gilt ein partieller Unique-Index.

| Feld | Typ | Regel |
| --- | --- | --- |
| `ingest_key` | text | Unique, Digest der vollständigen signierten Transporteingabe |
| `claimed_activity_iri` | text? | syntaktisch kanonische, noch unbestätigte Behauptung |
| `payload_hash` | text | unveränderlich |
| `signed_body_b64` | text | grössenbegrenzte, bytegenaue unveränderliche Request-Bodybytes; keine JSON-Reserialisierung |
| `verification_envelope` | `FederationVerificationEnvelopeV1` (json) | geschlossene, grössenbegrenzte und verlustfreie Eingabe für spätere Signatur-/Replayprüfung |
| `claimed_actor_iri` / `claimed_object_iri` / `claimed_origin_id` | text? | nur Eingangsbehauptungen |
| `verified_actor_iri` / `verified_object_iri` / `verified_origin_id` | text? | nur nach Authorityprüfung gesetzt |
| `verified_effect_key` | text? | nur beim gewinnenden Effekt gesetzt und partiell Unique; Authority + Activity-ID oder erlaubter semantischer Digest |
| `verified_effect_digest` | text? | kanonischer Digest aus verifiziertem Typ, Authority, Objekt, Payload, Public-Adressierung und Kausaldaten; bei jedem fachlich geprüften Effekt gesetzt |
| `object_order_scheme` / `object_order_token` | text? | versioniertes FED0-Verfahren und dessen kanonischer autoritativer Kausaltoken |
| `predecessor_token` | text? | vom Verfahren gebundener Vorgänger, falls vorhanden |
| `object_order_relation` | `FederationObjectOrderRelationV1` | Ergebnis des FED0-Vergleichs gegen den Objektkopf |
| `currentness_proof_hash` | text? | Digest des autoritativen, Token und Payload bindenden Currentness-Belegs |
| `duplicate_of` | relation? | bei `duplicate` verpflichtender Gewinnerrecord |
| `proof_hash` | text? | nach terminalem Verifikationsabschluss gesetzter Signatur-/Authority- oder Ablehnungsbeleg ohne Geheimnisse; bei `received/verifying/retry_wait` noch leer erlaubt |
| `received_at` | date | lokale DB-Zeit |
| `remote_time` | date? | nur Diagnose und Skewprüfung |
| `state` | select | `received`, `verifying`, `retry_wait`, `accepted`, `stale`, `rejected`, `duplicate`, `quarantined` |
| `lease_token` / `lease_until` | SafeRevisionV1/date? | CAS-Claim |
| `attempts` / `retry_not_before` | SafeRevisionV1/date? | Retry |
| `result_revision` | OptionalSafeRevisionV1 | bei `accepted` sowie bei einer autorisierten Ordnungsquarantäne mit eigener Securityrevision; sonst nicht gesetzt |
| `error_code` / `error_digest` | text? | sichere Diagnose |

`FederationVerificationEnvelopeV1` ist migrationsgebunden und kein freies
Guard-JSON. Es enthält HTTP-Methode/-Signaturschema, Request-Target-Bytes und
die erlaubten signaturrelevanten Header als geordnete typisierte Paare aus
normalisiertem ASCII-Namen und `value_b64`; Duplikate und Reihenfolge bleiben
erhalten. Bodybytes liegen ausschliesslich in `signed_body_b64`. Anzahl,
Einzel-/Gesamtgrösse und zulässige Headernamen haben harte V1-Grenzen. Der
Transition-Service validiert die Struktur vor dem durablen `received`-Commit;
eine später aus Strings rekonstruierte HTTP-Anfrage ist kein Beleg.

#### `federated_object`

Autoritativer lokaler Objektkopf. Unique-Key: `object_iri`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `object_iri` / `origin_id` | text | kanonische Identität; nach Genesis unveränderlich |
| `object_kind` | select | `legacy_trail`, `trail_publication`, `list_publication`; nach Genesis unveränderlich |
| `authority_actor_iri` / `authority_origin_id` | text | beim ersten akzeptierten Effekt atomar gesetzte und danach unveränderliche Object-to-Authority-Bindung; niemals aus `trails.author` oder `lists.author` abgeleitet |
| `authority_rule` | text | in V1 exakt `same_origin_dereferenced_v1`; nach Genesis unveränderlich |
| `first_verified_at` / `first_evidence_digest` | date/text | unveränderlicher Genesisbeleg |
| `authority_hash` | text | Digest der immutable Genesis-Bindung und des aktuell gültigen append-only Keyproofs; Keyrotation ändert weder Actor noch Origins |
| `state` | select | `discoverable`, `expired`, `restricted`, `deleted`, `quarantined` |
| `verification_state` | `FederatedVerificationStateV1` | Prüfzustand, getrennt von Sichtbarkeit |
| `visibility_state` | `FederatedVisibilityStateV1` | `public` nur mit Public-Beleg, `restricted` nur mit gebundenem Actor-/ACL-Scope; niemals aus einem Legacy-`public=true` geraten |
| `current_snapshot` | relation? | nur bei bestätigtem Inhalt |
| `last_effect_key` | text? | stabiler Key des letzten den Kopf kausal fortschreibenden autorisierten Effekts |
| `object_order_scheme` / `object_order_token` | text? | letzter akzeptierter FED0-Kausalstand |
| `currentness_proof_hash` | text? | letzter den Kopfzustand und Token bindender Beleg |
| `verified_at` | date? | lokale Verifikationszeit |
| `discoverable_until` | date? | exklusive lokale TTL |
| `refresh_due_at` | date? | vor Ablauf, mit deterministischem Jitter |
| `last_catalog_revision` | SafeRevisionV1 | letzter inhaltlicher/Visibility-Effekt |
| `delete_revision` / `delete_reason` | OptionalSafeRevisionV1/`FederationDeleteReasonV1`? | genau bei `state = deleted` beide gesetzt, sonst beide semantisch leer |
| `version` | SafeRevisionV1 | CAS-Version |

`deleted` ist in V1 für dieselbe Objekt-IRI absorbierend. Eine verspätete oder
wiederholte Create-/Update-Aktivität darf den Eintrag nicht reaktivieren. Eine
spätere Produktentscheidung für Resurrection benötigt eine ausdrückliche neue
Transition mit stärkerem Authority-Nachweis; sie ist kein normaler Refresh.

Delivery-Signer, `activity.actor` und Object-Authority sind getrennte
Identitäten. Genesis verlangt genau eine `attributedTo`-Authority, eine
kanonisch dereferenzierte HTTPS-Objekt-IRI und einen Beleg, dass Actor-,
Authority- und Object-Origin gemäss `same_origin_dereferenced_v1`
zusammengehören. Eingebettete Attribution allein genügt bei `Announce` nicht.
Update, Refresh und Delete müssen die bestehende Bindung treffen und dürfen sie
nie neu anlegen oder umhängen. Keyrotation ergänzt nur einen append-only Proof
für dieselbe Actor-/Originidentität. Actortransfer, Cross-Origin-Delegation und
Forwarding sind in V1 verboten und benötigen später eine neue Authority-Regel.

#### `federated_trail_snapshot`

Inhaltlich unveränderlicher Snapshot. Unique-Key mindestens
`(federated_object, payload_hash, proof_hash)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `federated_object` | relation | Objektkopf |
| `payload_hash` / `proof_hash` | text | unveränderlich |
| `normalized_payload` | json | kanonische verifizierte Fachpayload |
| `authority_hash` | text? | bei `verified` verpflichteter gebundener Autoritätsbeleg |
| `object_order_scheme` / `object_order_token` | text? | bei `verified` verpflichteter, mit dem Payload gebundener FED0-Kausalstand |
| `currentness_proof_hash` | text? | bei `verified` verpflichteter Payload-, Authority- und Kausaltoken bindender Beleg |
| `source_published_at` / `source_updated_at` | date? | diagnostische Remotezeit |
| `received_at` / `verified_at` | date/date? | lokale Zeiten |
| `verification_state` | select | `candidate`, `verified`, `rejected`, `quarantined` |
| `superseded_at` / `revoked_at` | date? | Lifecycle-Metadaten, Payload bleibt unverändert |
| `retain_until` | date? | Audit-/Kontextretention |

Ein `candidate` darf Authority-/Order-/Currentness-Felder bis zum
Verifikationsabschluss leer lassen. `verified` verlangt dagegen
`authority_hash`, Order-Scheme/-Token, `currentness_proof_hash` und
`verified_at`; diese Werte werden mit dem Payload unveränderlich.

Nur ein `verified` Snapshot, der vom Objektkopf als aktuell referenziert wird,
kann Suchinput sein. Verifikations- und Objektzustand werden nicht aus dem
Vorhandensein eines Trailrecords geraten.

#### `federated_visibility_provenance`

Gemeinsamer, unveränderlicher Evidence-Record aller Federation-Ingresspfade.
Unique-Key: `(federated_object, evidence_digest)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `federated_object` | relation | gebundener Objektkopf |
| `source` | select | `activity_create`, `activity_update`, `direct_announce`, `remote_resolve`, `list_expansion`, `parent_fetch`, `admin_resync`, `legacy_migration` |
| `object_iri` / `authority_actor_iri` / `origin_id` | text | exakt gegen die immutable Genesis-Authority geprüft |
| `audience_manifest` | json | geschlossene kanonische Public-/Actor-/ACL-Adressierung; keine freie Remote-Behauptung |
| `evidence_id` / `evidence_digest` | text | unveränderlicher Identitäts- und Inhaltsbeleg |
| `verification_rule` | text | versionierte Authority-/Adressierungsregel |
| `verification_state` / `visibility_state` | `FederatedVerificationStateV1`/`FederatedVisibilityStateV1` | getrennte Prüf- und Sichtbarkeitsentscheidung |
| `observed_at` | date | lokale DB-Zeit |

Inbox Create/Update, Direct-Announce, Remote-Resolve, Listenexpansion,
synthetischer Parent-Fetch, Admin-Resync und Legacy-Migration speichern Trails
und Listen ausschliesslich über denselben typisierten Federation-
Persistenzdienst. Parser dürfen Records vorbereiten, aber weder `public` noch
Scopes ausserhalb dieses Dienstes aktivieren. Ein Before-Create/Before-Update-
Guard lehnt einen Remote-Author oder eine Remote-IRI ohne passende persistierte
Provenienz fail-closed ab; er errät keine Audience aus Fachfeldern. Lokale
Pluginimporte behalten ihre getrennte Provider-/Owner-Provenienz.

Diese Inbound-Eindämmung hängt nicht von einer reparierten lokalen Outbox oder
der Erreichbarkeit eines Peers ab. Eine fehlende oder fehlerhafte Outbox darf
globale Neu-Publikation blockieren, aber niemals direct/private Inbound-
Containment verzögern.

#### `federated_publication`

Persistente öffentliche Federation-Identität eines lokalen Trails. Ein Trail
und seine Publication sind verschiedene Identitäten. Unique-Keys:
`publication_iri` und `(trail_key, generation)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `trail_key` | text | stabile lokale Fachidentität; keine Cascade auf Audit-/Tombstonebelege |
| `publication_iri` | text | pro Generation neue, unveränderliche kanonische IRI |
| `generation` | SafeRevisionV1 | je Trail lückenlos steigend; P1 und P2 werden nie dieselbe IRI |
| `authority_actor_iri` / `authority_origin_id` | text | immutable lokale Publikationsauthority |
| `audience_manifest` | json | geschlossene kanonische öffentliche Audience |
| `payload_revision` / `payload_hash` | SafeRevisionV1/text | vollständig gebundener öffentlicher Stand |
| `state` | `FederatedPublicationStateV1` | `staged`, `publishing`, `published`, `tombstoned` oder `quarantined` |
| `create_outbox` / `delete_outbox` | relation? | persistierte Create-/Delete-Intents der Publication |
| `tombstoned_at` / `tombstone_proof_hash` | date?/text? | genau bei `tombstoned` gesetzt |
| `version` | SafeRevisionV1 | CAS-Version |

```text
staged -> publishing -> published
staged/publishing/published -> tombstoned
staged/publishing/published -> quarantined
```

`tombstoned` ist für dieselbe Publication-IRI absorbierend und wird beim
lokalen Unpublish im selben Commit wie Fachrestriction und Fence gesetzt; sein
Abschluss wartet nicht auf einen Peer. Konnte `Create(P1)` nachweislich noch
keine Netzwerkwirkung haben, wird es mit einem dauerhaften Non-Delivery-Beleg
abgebrochen. Andernfalls legt derselbe Commit `Delete(P1)` für jeden potenziell
erreichten Empfänger in der Outbox an. Private→Public erzeugt nach lokaler
Projektion und durablem Outbox-Commit `Create(P1)`. Eine spätere erneute
Publikation verwendet zwingend eine neue IRI `P2` und `Create(P2)`; sie wartet
nicht auf die Remote-Zustellung von P1s Delete. Reaktionen oder Links auf P1
werden nicht P2 zugeordnet. Direct-Shares sind keine Publication und erzeugen
niemals Global-Membership.

#### `federation_outbox_event` und `federation_outbox_delivery`

Die Outbox ist die einzige Quelle ausgehender zugesagter Federationwirkungen.
Activity, Authority, Audience und Idempotenz werden vor jeder Netzwerkwirkung
durable gespeichert.

| Feld | Typ | Regel |
| --- | --- | --- |
| `effect_key` / `payload_hash` | text | zusammen fachlich idempotent und unveränderlich |
| `activity_type` | select | typisierte Create-/Update-/Delete-/Announce-/Undo-Wirkung |
| `authority_actor_iri` / `object_iri` | text | immutable gebundene Identitäten |
| `audience_manifest` / `payload` | json | geschlossene kanonische Empfänger-/Activitydaten |
| `state` | select | `queued`, `delivering`, `retry_wait`, `delivered`, `failed`, `canceled`, `quarantined` |
| `attempts` / `retry_not_before` | SafeRevisionV1/date? | begrenzter Retry mit Jitter und `Retry-After` |
| `lease_token` / `lease_until` | SafeRevisionV1/date? | CAS-Claim |
| `terminal_proof_hash` | text? | bei terminalem Ergebnis verpflichteter Beleg |

`federation_outbox_delivery` besitzt pro `(outbox_event, recipient_key)` genau
einen wiederaufnehmbaren Zustellautomaten mit HTTP-Status, sicherem
Response-Digest, Backoff und terminalem Ergebnis. Lokale Privacywirkungen
werden im Fach-/Fence-Commit wirksam; Remote-Zustellung ist nur eine
nachgelagerte Nebenwirkung. Kein API-Erfolg behauptet weltweite Löschung
bereits empfangener Bytes.

#### `federation_legacy_migration`

Ein persistenter Keyset-Sweep klassifiziert jeden vor SEC-SCOPED vorhandenen
Remote-Record fail-closed. Der Singleton bindet Cutoff, Cursor,
Catch-up-Revision, Versuchsbudget, Counts und Zustand
`pending | scanning | catching_up | validating | complete | quarantined`.
`complete` und `quarantined` sind terminal; aus Quarantäne beginnt nur ein
auditierter neuer Migrationslauf mit neuer ID.

Jeder geprüfte Objektkopf erhält genau ein terminales
`FederationMigrationOutcomeV1`: `public_verified`, `scoped_verified`,
`restricted_unproven`, `gone`, `invalid` oder `unreachable`. Ein historisches
`public=true` ist kein Beleg. Nicht erreichbare oder nicht beweisbare Objekte
bleiben aus öffentlichen Scopes ausgeschlossen; ein späterer Admin-Resync ist
ein neuer normaler Federationeffekt und schreibt das terminale
Migrationsergebnis nicht um.

Globales Public-Serving verlangt einen vollständigen Sweep bis zum Cutoff,
Catch-up bis zum aktuellen Federationstand, null unklassifizierte Records im
freizugebenden Scope, übereinstimmende DB-/Projektionscounts und eine
erfolgreiche Validierung der neuen Search-Regeln. Crash, Restore und
Concurrent-Ingest dürfen Cursor oder terminale Ergebnisse nicht verlieren.

#### `federation_scope_head`

Konservativer Federation-Horizont. Unique-Key: `(scope_kind, scope_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_kind` | `FederationScopeKindV1` | `local`, `all_federated`, `origin`, `actor`, `acl_scope` oder `publication` |
| `scope_key` | text | `*` beziehungsweise kanonische Local-, Origin-, Actor-, ACL- oder Publication-ID |
| `snapshot_set_revision` | SafeRevisionV1 | ändert sich bei relevanter Mitgliedschaft/Semantik |
| `last_catalog_revision` | SafeRevisionV1 | Revision der letzten Headänderung |
| `earliest_discoverable_until` | date? | exakt das Minimum der Expiries aller aktuell `discoverable` Scope-Mitglieder; leer genau bei leerer Mitgliedschaft |
| `scope_digest` | text | Hash aus vollständiger Membership-, Authority- und Expirybindung einschliesslich des Minimums |
| `version` | SafeRevisionV1 | CAS-Version |

Jede autorisierte Änderung von `discoverable_until`, auch die reine Verlängerung
bei identischer Payload, verändert die künftige Suchsemantik. Sie erhält deshalb
eine Katalogrevision, erneuert die Suchprojektion und erhöht
`snapshot_set_revision`. Ein bytegleiches Replay, das auch die bereits
bestätigte Zeitgrenze nicht ändert, bleibt dagegen ein No-op. Diese konservative
Regel hält Enginefilter, exakte Counts und `revision_fenced`-Cursor auf demselben
Stand.

Der Head-CAS leitet `earliest_discoverable_until` aus derselben kanonisch
sortierten Mitgliedermenge ab, die `scope_digest` bildet. Bei mindestens einem
`discoverable`-Mitglied ist das Feld gesetzt und exakt gleich
`min(member.discoverable_until)`; bei leerer Menge ist es genau dann leer.
Ein fehlendes, zu spätes oder nicht vom Digest gebundenes Minimum ist kein
konservativer Ersatz, sondern eine harte Invariantverletzung.

#### `federation_scope_version`

Append-only-Historie eines Scope-Heads. Unique-Keys:
`(scope_kind, scope_key, catalog_revision)` und
`(scope_kind, scope_key, snapshot_set_revision)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_kind` / `scope_key` | select/text | exakt wie am Head |
| `catalog_revision` | SafeRevisionV1 | Fachschnitt dieser Version |
| `snapshot_set_revision` | SafeRevisionV1 | fachliche Federationrevision |
| `membership_manifest` | `FederationScopeMembershipManifestV1` (json) | vollständige kanonisch sortierte immutable Mitgliedermenge mit Snapshot-/Authoritybindung und Einzel-Expiry |
| `earliest_discoverable_until` | date? | unveränderliche exakte Minimum-Expiry dieser Version; leer genau bei leerer Mitgliedschaft |
| `scope_digest` | text | exakt der bestätigte Head-Digest |
| `recorded_at` | date | DB-Zeit des Revisionscommits |

Jede Federation-Revision aktualisiert alle betroffenen Local-, Origin-, Actor-,
ACL-, Publication- und Global-Heads
und fügt deren Scope-Versionen in derselben Revisionstransaktion an. Revision 0
besitzt für `all_federated/*` einen leeren initialen Versionsrecord. Die
Historie wird mindestens solange reteniert, wie ein Content-Build-Cutoff sie
noch referenzieren kann.

Die Scope-Version kopiert Mitgliedermenge, jede gebundene
`discoverable_until`, das daraus berechnete exakte Minimum und den Digest aus
dem im selben Commit bestätigten Head. Leere Mitgliedschaft, leeres Minimum
und der dafür kanonisierte leere Digest treten nur gemeinsam auf; eine
nichtleere Mitgliedschaft mit leerem oder nach dem wirklichen Minimum
liegendem Horizont darf weder versioniert noch publiziert werden.

`FederationScopeMembershipManifestV1` ist eine geschlossene, nach kanonischer
Objekt-IRI sortierte Struktur. Jeder Eintrag bindet Objekt-, aktuellen
Snapshot- und Origin-Schlüssel, Authority-/Currentness-Proofdigest sowie die
exakte `discoverable_until`; `scope_digest` hasht Scopeidentität,
Versionsrevisionen und die bytegenauen Manifestbytes. Das leere Manifest ist
kanonisch `[]`. Version, Manifest und die darin referenzierten Snapshot-/
Proofartefakte bleiben gemeinsam reteniert, solange ein Build,
Content-Snapshot oder Cursorbeleg die Version benötigt; ein Digest ohne
rekonstruierbares Manifest beweist das historische Minimum nicht.

Jeder Manifesteintrag bindet zusätzlich den typisierten Visibility-/
Publication-Proof, seinen Scope und seine Revision. Ein public Snapshot ohne
Publication-Proof darf nicht Mitglied von `all_federated` sein; ein direct
Snapshot darf ausschliesslich in seinen belegten Actor-/ACL-Scope gelangen.

### 4.3 Fence-, Generationen- und Publishercollections

#### `search_subject_visibility_frontier`

Genau ein aktueller, transaktional fortgeschriebener Sichtbarkeitskopf je
Subject. Unique-Key: `(subject_kind, subject_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `subject_kind` / `subject_key` | `FenceSubjectKindV1`/text | stabile fachliche Identität |
| `frontier_revision` | SafeRevisionV1 | letzte katalogrelevante Visibility-Revision dieses Subjects |
| `security_watermark` | SafeRevisionV1 | höchste in diesem Kopf enthaltene verengende Securityrevision |
| `state` | `SubjectVisibilityFrontierStateV1` | aktueller autorisierter Sichtbarkeitszustand |
| `restriction_present` | bool | wahr genau bei einem physisch verengenden Zielstand |
| `authoritative_visibility_digest` | text | Hash des vollständigen aktuellen Fach-/Policyzustands |
| `effective_restriction_digest` | text | bei `restriction_present` Hash des kumulativen physischen Zielstands, sonst leer |
| `source_manifest` | `RestrictionFrontierSourceSetV1` (json) | geschlossene kanonisch sortierte Menge aller aktuellen Restriction- und benötigten historischen Retirement-/Successor-Belege |
| `source_manifest_hash` | text | Digest der bytegenau kanonisierten `source_manifest`-Bytes |
| `gateway_guard_set_digest` | text | Digest aller für dieses Subject aktuell wirksamen nicht-retirten Gateway-Guards und ihrer semantischen Anker |
| `version` | SafeRevisionV1 | CAS-Version |

Jede Restriction, jeder Grant/Clear und jeder sie supersedierende
Sichtbarkeitseffekt aktualisiert diesen Kopf im selben Revisionstransaktions-
Commit. `RestrictionFrontierSourceSetV1` ist kein freies JSON: Jeder Eintrag
bindet Fence-/Successor-ID, Set-/Clear-/Release-Revision, Proofkind und
-digest sowie den versionierten Scope-/Subsumptionsbeleg. Bei
`unrestricted` ist `restriction_present = false`; `clear_pending` trägt den
autorisierten Nachfolger, während alte Gateway-Guards noch wirken. Ein
retirter Fence kann Teil des Manifests bleiben, solange seine autoritative
Restriction fortbesteht. Jede autorisierte Visibility-Revision sowie jede
semantisch relevante Successor-, Guard-Release- oder Retirementänderung
schreibt Manifest, beide Digests und `version` unter demselben
SQLite-Schreiblock fort; reine Delivery-/Leasefortschritte verändern den Kopf
nicht. Kein Worker darf diesen Kopf aus Enginezustand zurückrechnen.

`restricted` und `deleted` verlangen `restriction_present = true` und
`frontier_revision = security_watermark`: Ihr jüngster autoritativer
Visibilitystand ist selbst eine Verengung. Bei `clear_pending` kann die spätere
`frontier_revision` dagegen oberhalb des letzten verengenden
`security_watermark` liegen; dieser Zustand ist für einen epochlosen
Securitywrite grundsätzlich unzulässig.

Mehrere Köpfe können dasselbe physische Suchziel beeinflussen, etwa
`actor`/`origin` und `federated_object` oder `acl_scope` und `trail`. Der Kopf
ist deshalb kein Alleinanspruch auf ein Dokument. Jeder physische
Securitywrite muss die nach dem versionierten Visibility-Regelartefakt
vollständige Menge aller für **jedes** mutierte Ziel wirksamen Köpfe binden und
als kumulativen Zielstand materialisieren; ein einzelner Fence- oder
Subjectdigest genügt nicht.

#### `search_visibility_fence`

| Feld | Typ | Regel |
| --- | --- | --- |
| `cause_key` | text | namespaced Idempotency-Key |
| `subject_kind` / `subject_key` | `FenceSubjectKindV1`/text | betroffene Ressource oder Audience |
| `restriction_kind` | select | `delete`, `public_to_private`, `share_revoke`, `actor_block`, `authority_revoke`, `ttl_shorten`, `content_redaction` |
| `set_revision` | SafeRevisionV1 | Securityrevision des linearisierenden Commits |
| `gateway_guard_kind` | `FenceGatewayGuardKindV1` | ob der sofortige Queryguard exakt dem fachlichen Ziel entspricht oder das Subject vorübergehend vollständig ausschliesst |
| `exact_target_digest` | text | unveränderlicher Digest des vollständig autorisierten Restriction-Zielstands bei `set_revision`, auch bei konservativem Gateway-Guard |
| `visibility_rule_version` | text | unveränderliches versioniertes Regelartefakt für Beobachtungsäquivalenz und Subsumption |
| `state` | select | `active`, `propagating`, `satisfied`, `retired` |
| `effective_at` | date | lokale DB-Zeit |
| `satisfied_at` / `retired_at` | date? | Diagnose/GC |
| `clear_revision` | OptionalSafeRevisionV1 | nur vor `retired` einmalig gesetzte Revision des ersten autorisierten Clear-Ankers strikt nach `set_revision`; spätere Clear-Advances und Retirement verändern sie nicht |
| `guard_release_revision` | OptionalSafeRevisionV1 | bei `conservative_subject_exclusion` die im Restriction-Commit gesetzte früheste normale Ziel-/Publishrevision `>= set_revision`; ihre spätere Epoch muss den exakten Zielstand oder einen typisiert nachweislich engeren koaleszierten Successor tragen; sonst nicht gesetzt |
| `retirement_proof_kind` / `retirement_proof_hash` | `FenceRetirementProofKindV1`?/text? | genau bei `retired` gesetzter unveränderlicher Konvergenzbeleg |
| `retirement_artifact_manifest` | `FenceRetirementArtifactManifestV1`? (json) | genau bei `retired` gesetzte, kanonisch sortierte vollständige Identitäts-/Digestliste aller für die Rekonstruktion benötigten Epoch-, Snapshot-, Scope-, Member-, Delivery-, Successor- und Regelartefakte; ihre Einträge sind Audit-Pins |
| `history_retain_until` | date? | mindestens aktueller Security-History-HWM plus Grace |
| `version` | SafeRevisionV1 | CAS-Version |

Unique-Key: `(cause_key, subject_kind, subject_key, restriction_kind)`.

Mehrere aktive Fences desselben Subjects wirken als Vereinigung ihrer
Verengungen. Ein späteres Grant löscht keinen historischen Fence, sondern
erzeugt zuerst eine normale `visibility_expanded`-Revision und einen
append-only Successor-Link. Ist der Fence noch nicht `retired`, setzt derselbe
Revisionstransaktions-CAS einmalig `clear_revision` und der alte Gateway-Guard
bleibt wirksam, bis der exakte erlaubte Nachfolgerstand für alle aktuellen
Generationen physisch bestätigt ist. Ist der Fence bereits mit
`restriction_converged` oder `guard_release_converged` retired, bleibt sein
Record einschliesslich leerer `clear_revision` vollständig unverändert; der
erste Grant wird als späterer Successor und normale neue Content-Epoch
publiziert. Ein `query_fenced`-Beleg
kann eine Verengung sicher überbrücken, aber niemals eine
Sichtbarkeitserweiterung freigeben.

Der Retirement-CAS setzt Kind und Hash gemeinsam mit `retired_at`. Der Hash
bindet Fence-/Subjectidentität, Set-/Clear-/Guard-Release-Revision und
Gateway-Guard-Kind, bei einem Clear die
vollständige lückenlose Successor-Kette samt ihrer beim CAS erneut gelesenen
Tail-ID/-Revision, bei einem Restriction-Supersession-Zweig den eindeutigen
Successor-Link, dessen Retirement-Abhängigkeit, `exact_target_digest`,
`visibility_rule_version` und den linkgebundenen Multi-Frontier-Hash. Ist der
Successorstand inzwischen fortgeschritten, bindet er zusätzlich den vollständig
rekonstruierbaren `SuccessorProgressProofV1` samt aktuellem Closure-/Tail-Hash,
Epoch und weiterhin wirksamen Guards. Ausserdem bindet der Retirementhash den
autoritativen Projektionsdigest, etwaige Subsumptionsbelege und die kanonisch
nach Member-ID geordnete Liste aller verwendeten Fence- beziehungsweise
Clear-Delivery-IDs samt Proofdigests. Diese
Records und das Regelartefakt bleiben mindestens bis `history_retain_until`
reteniert; ein Hash ohne rekonstruierbare Belegkette genügt nicht.

`FenceRetirementArtifactManifestV1` ist kein freies Beweis-JSON, sondern ein
geschlossenes, migrationsgebundenes Manifest. Seine Einträge binden jeweils
Recordart, namespaced Record- beziehungsweise Inhaltsidentität,
Restore-/Control-Plane-Inkarnation, Schema-/Vertragsversion, die gegebenenfalls
beobachtete numerische Recordversion und den Digest der typisierten
Proofprojektion. Der Retirement-CAS erzeugt es unter demselben
SQLite-Schreiblock, prüft, dass jedes Artefakt vorhanden und unverändert ist,
und materialisiert dazu die unten definierten normalisierten Reverse-Pin-
Zeilen. Manifest und Pinmenge müssen bijektiv denselben kanonischen Digest
ergeben; das JSON allein blockiert keinen GC. Die Pinzeilen verhindern die
physische Löschung des benötigten PocketBase-Metadatensatzes und seiner
unveränderlichen Proofartefakte. Das gilt insbesondere für Content-Snapshot,
Epoch-Scope-Bindings und Member-/Fence-Deliveries eines
`SuccessorProgressProofV1`, auch wenn deren normaler Cursor-HWM bereits
abgelaufen ist. Die externe Engine-Ressource darf nach ihren eigenen Cursor-,
Rollback- und Build-Gates gelöscht werden; der persistierte Snapshot-/Marker-
und Deliverybeleg bleibt davon getrennt rekonstruierbar. Ein Pin wird nur im
serialisierten Fence-History-GC-CAS gemeinsam mit dem nicht mehr benötigten
Fence-/Proofzweig freigegeben, niemals einzeln oder allein wegen eines Datums.

#### `search_visibility_fence_successor`

Append-only-Verknüpfung eines autorisierten Sichtbarkeitsnachfolgers mit jedem
von ihm fachlich abgelösten Fence. Unique-Keys sind
`(fence, successor_key)` und `(fence, predecessor_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `fence` / `successor_key` | relation/text | historische Restriction und namespaced unveränderliche Nachfolgeridentität |
| `predecessor_key` | text | beim ersten Link exakt `root:<fence-id>`, danach exakt `successor_key` des bisherigen Tails desselben Fence |
| `successor_change` / `successor_fence` | relation?/relation? | genau eine Relation gemäss `kind` gesetzt |
| `successor_revision` | SafeRevisionV1 | exakt die Change-Revision oder `successor_fence.set_revision`, strikt nach allen gebundenen Vorgängern |
| `kind` | `FenceSuccessorKindV1` | aus dem unter demselben DB-Lock gelesenen Fencezustand |
| `subject_digest` | text | bindet Subject-Kind/-Key, Restriction und autorisierten Nachfolger |
| `authorization_proof_hash` | text | unveränderlicher Authority-/Policy-/Commandbeleg des Nachfolgers |
| `recorded_at` | date | DB-Zeit der Revisionstransaktion |

Jeder Append liest unter dem gemeinsamen SQLite-Schreiblock den eindeutigen
Tail. Fehlt er, muss `predecessor_key = root:<fence-id>` gelten; andernfalls
muss das Feld exakt dessen `successor_key` tragen. Der zweite Unique-Key ist
damit auch für die nichtleere Root-Sentinel ein atomarer No-Fork-CAS. Revision
und autorisierter Subjectzustand müssen strikt nach dem Tail liegen. Ein Link
mit unbekanntem, bereits belegtem oder nicht aktuellem Predecessor scheitert;
die Kette besitzt zu jedem Zeitpunkt genau einen Root und einen Tail.

Ein Grant-Link entsteht im selben SQLite-Commit wie der Successor-Change. Für
den ersten Grant eines noch nicht `retired`en Fence gilt
`kind = clear_before_retirement`; derselbe CAS setzt dessen zuvor leere
`clear_revision` exakt auf `successor_revision`. Jeder weitere autorisierte
Grant vor Retirement verwendet `kind = clear_advance_before_retirement`, hängt
am bisherigen Grant-Tail und lässt `clear_revision` sowie alle älteren Links
unverändert. Ein Restriction-Successor schliesst einen solchen offenen
Grant-Tail; danach darf auf diesem alten Fence kein weiterer Clear-Link
entstehen. Für einen bereits `restriction_converged` oder
`guard_release_converged` retirten Fence gilt beim ersten späteren Grant
`kind = grant_after_retirement`; Fence,
Retirement-Proof und `clear_revision` bleiben bytegleich. Ein bereits
`clear_converged` retirter Fence ist durch seine bestehende Clear-Kette
abgeschlossen und wird von späteren Erweiterungen nicht erneut mutiert; diese
sind normale neue Content-Revisionen. Mehrere abgelöste Fences erhalten je
eine eigene Kette. Link, Successor-Change, dessen Deliveries und Proofartefakte
bleiben mindestens mit der Security-History reteniert.

Wird nach Setzen des `guard_release_revision`-Ankers eines
`conservative_subject_exclusion`-Fence, aber vor dessen Retirement, eine
strikt spätere autoritative
Restriction desselben Subjects mindestens ebenso eng wie dieser exakte
Zielstand wirksam, legt deren Revisionstransaktion am noch leeren Root-Slot
`kind = guard_release_superseded_by_restriction` mit Relation zum neuen Fence
an. Zulässig sind dabei nur gesetzte `guard_release_revision`, leere
`clear_revision`, noch kein bestehender Successor und
`predecessor_key = root:<fence-id>`. Der typisierte
`authorization_proof_hash` bindet `guard_release_revision`, den immutable
`exact_target_digest` samt `visibility_rule_version`, den vollständigen
aktuellen Multi-Frontier-Closure und den Scope-/Restriktionsbeleg, nach dem der
neue bereits gatewaywirksame Fence diesen Zielstand vollständig subsumiert.
Die exakte alte Ziel-Epoch muss zu diesem Zeitpunkt noch nicht publiziert sein;
der Link entsteht gerade auch dann atomar mit F2, wenn Dirty-Koaleszierung den
Zwischenstand F1 später überspringt. Der
alte Fence, sein Guard-Release-Anker und seine Deliveries bleiben unverändert;
der Link schliesst nur den sonst festhängenden Retirementpfad.

Wird vor der vollständigen Clear-Konvergenz eine strikt spätere autoritative
Restriction desselben Subjects mindestens ebenso eng wirksam, erzeugt deren
Revisionstransaktion je betroffenem Vorgängerfence genau einen am aktuellen
Grant-Tail hängenden Link `kind = clear_superseded_by_restriction` auf den
neuen Fence. Sein Proof subsumiert die vollständige Clear-Kette dieses Fence
bis zu genau diesem Tail; mehrere Vorgängerfences besitzen je ihre eigene
Kette. Der `authorization_proof_hash` bindet hierbei zusätzlich den
versionierten Scope-/Restriktions-Subsumptionsbeleg. Alter Fence und dessen
unveränderte `clear_revision` werden nicht umgeschrieben. Die strikt steigende
`successor_revision` macht diese Belegkette azyklisch. Jeder eingehende
`clear_superseded_by_restriction`- oder
`guard_release_superseded_by_restriction`-Link ist zugleich eine
Retirement-Abhängigkeit: Der Successor-Fence darf
unabhängig vom eigenen Proof-Zweig nicht `retired` werden, solange noch einer
seiner dadurch neutralisierten Vorgängerfences nicht `retired` ist. Der
gemeinsame SQLite-Schreiblock serialisiert Linkanlage und beide
Retirement-CAS. Damit wird die Kette vom ältesten Vorgänger zum neuesten
Successor abgebaut, ohne einen benötigten Gateway-Guard vorzeitig zu entfernen.

Ein nach der Linkanlage autorisierter Grant oder weiterer Restriction-Change
auf dem Successor bleibt dabei fortschrittsfähig: Er committet normal auf
dessen eigener Frontier-/Successor-Kette und öffnet die bereits geschlossene
Kette des Vorgängerfence nicht erneut. Die eingehende Retirement-Abhängigkeit
hält jedoch den unveränderlichen, im Link gebundenen Restriction-Guard des
Successors querywirksam. Ist dessen autoritativer Stand beim
Vorgänger-Retirement bereits weitergelaufen, ersetzt ein geschlossener
`SuccessorProgressProofV1` die Annahme, die Linkrevision sei noch der aktuelle
Kopf. Er bindet den unveränderlichen Link-/Restrictionmarker, die vollständige
transitive Menge noch nicht retirter Successor-Abhängigkeiten, deren aktuelle
Frontier-/Tail-Versionen, die höchste darin autorisierte Visibilityrevision,
eine normale geschlossene Content-Epoch mit Cutoff mindestens auf dieser
Revision sowie jeden bis zum geordneten Retirement weiterhin wirksamen
Successor-Guard. Der Epochinhalt darf dabei bereits den späteren Grant oder
eine weitere Restriction tragen; bis der älteste Vorgänger retiert, kann dieser
weitere Stand wegen der statischen Successor-Guards keine Sichtbarkeit
vorzeitig öffnen. Danach retiren die Fences unter demselben Schreiblock streng
von alt nach neu jeweils mit ihrem eigenen Proof-Zweig. Ändert sich Frontier,
Tail, Epoch oder eine Fence-Version während des CAS, verliert dieser und löst
den vollständigen Closure erneut auf. Ein bloss `query_fenced`-ter Successor
ohne diese normale geschlossene Fortschrittsepoch genügt nie.

`history_retain_until` ist nur eine zeitliche Untergrenze. Ein mit
`restriction_converged` oder `guard_release_converged` retirter Fence ohne
`grant_after_retirement`-Link bleibt in V1 unabhängig von diesem Datum
erhalten: Solange kein späterer Grant den Übergang append-only geschlossen hat,
könnte genau dieser historische Fence noch für einen autorisierten Clear
benötigt werden. Grant-Commit und Fence-GC verwenden
denselben SQLite-Schreiblock; der Grant legt zuerst den Link an und verlängert
Fence, Retirement-Proof und Link mindestens bis zum dann aktuellen
Security-History-HWM plus Grace. V1 akzeptiert hier sichere Überretention;
eine spätere automatische Verdichtung ohne Grant benötigt einen eigenen
typisierten, dauerhaft retenierten Beweis, dass nie mehr ein Clear zulässig
sein kann.

#### `search_fence_retirement_artifact_pin`

Normalisierter append-only Reverse-Pin für jedes Element eines
`FenceRetirementArtifactManifestV1`. Unique-Key:
`(fence, artifact_kind, artifact_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `fence` | relation | Owner des terminalen Retirement-Proofs; gemeinsam mit ihm reteniert |
| `artifact_kind` | `FenceRetirementArtifactKindV1` | geschlossene Art des Proofartefakts |
| `artifact_key` | text | vollständige stabile, Restore-/Control-Plane-gescopte Recordidentität; bei Inline-/Content-Artefakten die unten definierte inhaltsadressierte Identität |
| `artifact_contract_version` | text | exakte migrationsgebundene Collection-, Serializer- oder Regelversion dieses Kinds |
| `artifact_record_version` | OptionalSafeRevisionV1 | `present = true` genau für Quellen mit numerischem CAS-`version`-Feld und dann dessen im Retirement-CAS beobachteter Wert; für immutable, append-only, inline oder nur textversionierte Quellen `present = false` |
| `artifact_proof_projection` | `FenceRetirementArtifactProofProjectionV1` (json) | geschlossene, nach `artifact_kind` diskriminierte unveränderliche As-of-Projektion aller aus dieser Quelle im Retirement-Proof verwendeten Werte |
| `artifact_digest` | text | typ-, Contract- und versionsgebundener Digest exakt dieser Proofprojektion |
| `pinned_at` | date | DB-Zeit des Retirement-CAS |

`FenceRetirementArtifactProofProjectionV1` ist eine geschlossene tagged union,
keine freie Map. Für `generation`, `generation_member` und
`visibility_fence` enthält sie die beim CAS verwendeten Identitäten,
Engine-Inkarnation, Epoch-/Cutoff-/Markerbindung, Fenceanker,
Frontier-/Successor-Tail und damaligen Zustand; für `catalog_change`,
`catalog_delivery` und `fence_delivery` deren Revisions-/Ordinalidentität,
terminalen Ack-/Resolutionzustand, Relations-IDs und Proofdigests. Bei
`geo_shard_manifest_version`, `generation_query_family`,
`content_snapshot_query_family` und `content_snapshot_member` enthält sie die
unveränderliche Manifest-/Memberidentität, logischen Shard, Generation,
Queryfamily-/Contractbindung, physischen Index, Cutoff, Watermarks, Marker und
deren Digests. Für
`search_contract_artifact` enthält sie Artifact-Key, Queryfamilie,
Payloadschema, Payloadhash und die vollständigen kanonischen Payloadbytes. Bei
anderen immutable Collections enthält sie die vollständige unveränderliche
Belegprojektion und deren Record-ID. Ein inline gespeichertes
`federation_membership_manifest` verwendet als `artifact_key` exakt
`<scope-version-id>#membership:<digest>` und bettet den vollständigen
kanonischen Manifestinhalt ein; ein content-addressed Regelartefakt bindet
Registry-Key, Contractversion und die separat durch denselben Pin retenierten
kanonischen Bytes. Eine Katalogrevision oder andere fachliche Revision darf
nicht als erfundene Recordversion in `artifact_record_version` eingesetzt
werden.

Die numerische Recordversion ist eine As-of-CAS-Beobachtung und kein
Lifecycle-Freeze. Nach Retirement dürfen mutable Quellen regulär in höhere
Versionen wechseln, etwa F2 `satisfied -> retired`, Generation
`active -> sealed -> retired -> deleting -> deleted` oder Member nach
`deleted`. Die Pinzeile bleibt bytegleich und rekonstruiert den damals
bewiesenen Stand aus `artifact_proof_projection`; der aktuelle Record muss nur
dieselbe stabile Identität behalten und darf keine dort nicht vollständig
gesicherte historische Evidenz zerstören. Benötigt ein Proof einen später
veränderlichen Wert, MUSS er deshalb in dieser Projektion oder einem zusätzlich
gepinnten append-only Artefakt stehen. Ein aktueller Whole-Record-Digest oder
die Forderung `current.version = artifact_record_version` ist ausdrücklich
unzulässig.

Alle Pinzeilen entstehen atomar mit `fence.state = retired`, Proofhash und
Manifest; ein partieller Satz oder ein Digestunterschied lässt den CAS
verlieren. Danach sind sie unveränderlich. Ein Artefakt-GC prüft ihre
Abwesenheit über den Reverse-Index unter demselben SQLite-Schreiblock; er scannt
weder Fence-JSONs noch ungebatchte History. Erst wenn Fence, Proofzweig,
Security-HWM, Grace und alle Successorabhängigkeiten GC-fähig sind, darf ein
einziger Fence-History-GC-CAS Fence-/Proofrecords und sämtliche zugehörigen
Pinzeilen gemeinsam entfernen. Dadurch kann zwischen Pinprüfung und
Artefaktlöschung kein neues Retirement denselben Beleg unbemerkt pinnen.

#### `search_visibility_fence_delivery`

Unique-Key: `(fence, generation_member)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `fence` / `generation_member` | relation/relation | Lieferziel |
| `state` | select | `pending`, `submitted`, `applied`, `query_fenced`, `failed` |
| `engine_task` | relation? | physischer Apply-Versuch |
| `proof_digest` | text? | Gatewayregel- beziehungsweise physischer Apply-Beleg |
| `frontier_binding_hash` | text? | bei epochlosem `fence_apply` verpflichteter Hash des vollständigen Multi-Frontier-Bindings |
| `confirmed_write_marker_hash` | text? | bei physischem `applied` verpflichteter Endmarker des geordneten Memberwriteprefixes |
| `acknowledged_at` | date? | lokale DB-Zeit |
| `error_code` / `error_digest` | text? | sichere Diagnose |
| `version` | SafeRevisionV1 | CAS-Version; steigt bei jeder Zustandskante |

`query_fenced` erlaubt Serving nur, wenn der Gateway den Fence vollständig vor
Ranking, Total und Counts anwendet. Es genügt nicht zum physischen Entfernen des
Fence, solange die Generation noch querybar **oder später aktivierbar** ist.

Erlaubt sind ausschliesslich:

```text
— -> pending -> submitted -> applied
       |           |
       |           +-> query_fenced -> applied
       |                    ^
       +--------------------+
       |
       +-> failed -> pending

submitted -> failed
failed -> query_fenced
pending/submitted/query_fenced -> applied  # nur mit physischem Beleg
```

`submitted` verlangt einen persistenten Taskintent; eine Transportantwort ist
noch kein Ack. `query_fenced` verlangt in demselben SQLite-CAS einen
Gateway-verifizierbaren, Fence, Subject, Member, `set_revision` und
Engine-Inkarnation bindenden `proof_digest`. Der memberbezogene
`query_fenced`-Deliverybeleg darf erst durch physisches `applied`, irreversible
Entfernung des Members aus Query-/Swapmengen oder einen dieses Member
abdeckenden Retirement-Beleg der Art `guard_release_converged`,
`guard_release_superseded_by_restriction`, `clear_converged` oder
`clear_superseded_by_restriction` abgelöst werden. Der
fenceweite Gateway-Guard selbst bleibt unabhängig davon bis zum einzigen
Retirement-CAS aktiv. Physische Retries dürfen während
`query_fenced` über neue persistente Tasks laufen, ohne den sicheren Zustand
auf `submitted` zurückzusetzen. `applied` ist terminal und verlangt
erfolgreichen Task, Endmarker und physischen Proof. Stammt der Apply aus dem
epochlosen `fence_apply`, bindet dieser Proof zusätzlich Attempt, vollständige
Multi-Frontier-Menge, alle mutierten physischen Ziele, den nach dem Apply
bestätigten geordneten Writeprefix und die vollständige
Beobachtungsäquivalenz zur zusammengesetzten aktuellen Querysemantik für alle
relevanten Principals und Query-/Responsebestandteile. Ein späterer Write darf
diesen Marker nur durch einen neuen nachweislich aktuellen kumulativen Marker
ablösen; andernfalls kann der Beleg nicht zum Retirement zählen. `failed` ist retrybar,
blockiert aber `satisfied` und jeden Watermark. Jede Kante ist ein
State-/Versions-CAS gegen unveränderten Fence und unverändertes Member.

#### `search_index_generation`

| Feld | Typ | Regel |
| --- | --- | --- |
| `generation_key` | text | Unique, opak und unveränderlich |
| `intent` | select | `normal`, `shadow`, `rollback`, `recovery` |
| `lifecycle_state` | select | `new`, `provisioning`, `backfilling`, `catching_up`, `validating`, `ready`, `active`, `sealed`, `quarantined`, `retired`, `deleting`, `deleted`, `abandoned` |
| `epoch_state` | select | `unpublished`, `published`, `updating`, `recovery`, `quarantined` |
| `content_epoch` | SafeRevisionV1 | monoton nur innerhalb dieser Generation |
| `catalog_cutoff` | SafeRevisionV1 | letzter publizierter geschlossener Prefix; innerhalb der Generation monoton und nie absenkbar |
| `capture_from_revision` | SafeRevisionV1 | pinnt Replay-/Artefaktretention ab Buildregistrierung |
| `schema_hash` / `settings_hash` / `bundle_hash` | text | bestätigte Manifeste |
| `query_family_manifest_hash` | text | unveränderlicher Digest der vollständigen kanonisch geordneten `search_index_generation_query_family`-Menge; enthält `route_radius_ux_v1` unabhängig vom Belegungsstand des initialen Route-Members |
| `engine_incarnation` | text | bei Provisionierung gebunden; tatsächlicher Wechsel quarantänisiert die Generation |
| `writer_restore_epoch` / `writer_credential_epoch` / `writer_term` | SafeRevisionV1/SafeRevisionV1/SafeRevisionV1 | letztes zugelassenes Writer-Fence-Tupel |
| `retention_resource` | relation | kanonische Cursor-/GC-Zusage dieser Generation |
| `rollback_keep_until` | date? | zusätzliche operative GC-Grenze |
| `version` | SafeRevisionV1 | CAS-Version |

Lifecycle und Epoch sind orthogonal. Insbesondere bedeutet `sealed` einen für
normale Contentwrites unveränderlichen Stand, während aktuelle Security-Fences
weiterhin vor jeder Query gelten. Alle angebotenen Queryfamilien werden mit der
Generation registriert; ihre Zeilen und `query_family_manifest_hash` entstehen
in demselben SQLite-CAS und sind danach unveränderlich. Eine später benötigte
neue Queryfamilie verlangt eine neue Generation.

#### `search_index_generation_query_family`

Unveränderliche Queryfamily-Bindung einer Generation. Unique-Key:
`(generation, query_family)`. Jede Generation besitzt für jede von ihr
angebotene Queryfamilie genau eine solche Zeile, für `route_radius_ux_v1` auch
bei leerem Katalog und vor der Provisionierung ihres verpflichteten initialen
Generationenmembers.

| Feld | Typ | Regel |
| --- | --- | --- |
| `generation` | relation | gemeinsam retenierte Bundlegeneration |
| `query_family` | `SearchIndexQueryFamilyV1` | unveränderliche logische Queryfamilie |
| `contract_artifact` | relation? | für `route_radius_ux_v1` verpflichtetes immutable `search_contract_artifact`, für `trail_default` in V1 leer |
| `contract_artifact_key` / `contract_artifact_hash` | text?/text? | gemeinsam mit der Relation gesetzt und exakt deren `artifact_key`/`payload_hash` |
| `registered_at_revision` | SafeRevisionV1 | Katalogstand der Family-Registrierung; bei Generationsanlage exakt `capture_from_revision` |
| `binding_hash` | text | Digest aus Generation, Queryfamilie, Contractrelation/-Key/-Hash und Payloadschema |

Die Zeile wird vor jedem Memberbuild angelegt und danach nie geändert. Für
`route_radius_ux_v1` ist sie die alleinige Quelle der Contractbindung; weder
ein bereits provisioniertes Route-Member noch dessen Dokumentbelegung ist dafür
erforderlich. Jedes Route-Member derselben Generation, einschliesslich des
initialen leeren Members, kopiert
Relation, Key und Hash exakt aus dieser Zeile. Ein Contractwechsel erzeugt
eine neue Generation samt neuer Family-Bindung.

#### `search_index_generation_member`

Unique-Keys: `(generation, query_family, component, partition_key)`, partiell
für gesetzte Shard-IDs `(generation, query_family, geo_shard_id)` und
`(engine_scope, physical_index_id)`. Eine physische ID darf innerhalb eines
logischen Engine-Scopes nie einem zweiten Generationenmember zugeordnet oder
wiederverwendet werden.

| Feld | Typ | Regel |
| --- | --- | --- |
| `generation` / `generation_query_family` | relation/relation | Bundlegeneration und ihre unveränderliche passende Queryfamily-Bindung |
| `query_family` | `SearchIndexQueryFamilyV1` | logischer Querypfad dieses Members |
| `component` | `SearchProjectionComponentV1` | primäre Projektionskomponente |
| `partition_kind` | `SearchIndexPartitionKindV1` | `singleton` oder logischer Route-Shard |
| `partition_key` | text | bei `singleton` exakt `singleton`, bei Route-Shards exakt `geo_shard_id` |
| `geo_shard` / `geo_shard_id` | relation?/text? | gemeinsam genau bei `partition_kind = geo_route_shard` gesetzt |
| `input_components` | `SearchProjectionComponentSetV1` (json) | geschlossene kanonische Menge aller in den physischen Dokumenten materialisierten Komponenten |
| `introduced_at_revision` | SafeRevisionV1 | frühester Katalogstand, ab dem das Member für einen Snapshot benötigt werden kann |
| `engine_scope` | text | stabile logische Engine-/Cluster-ID |
| `engine_incarnation` | text | muss bei Serving mit Generation und Engine übereinstimmen |
| `physical_index_id` | text | generationenspezifisch, nie öffentlicher API-Wert |
| `schema_hash` / `settings_hash` | text | komponentenspezifisch |
| `contract_artifact` | relation? | bei `query_family = route_radius_ux_v1` verpflichtete immutable `search_contract_artifact`-Relation, sonst in V1 leer |
| `contract_artifact_key` / `contract_artifact_hash` | text?/text? | gemeinsam mit der Relation gesetzt und exakt aus `generation_query_family` kopiert |
| `baseline_revision` | SafeRevisionV1 | konsistenter vollständiger Basisstand |
| `baseline_manifest_hash` | text? | Coverage-/Payloadbeleg aller relevanten Ziele bis Baseline |
| `delivered_revision` | SafeRevisionV1 | höchster lückenlos bestätigter Katalog-/Deliveryprefix einschliesslich `fenced` Security-Changes; monoton und nie absenkbar |
| `security_watermark` | SafeRevisionV1 | höchster lückenlos bestätigter Securityprefix; monoton und nie absenkbar |
| `coverage_hash` / `marker_hash` | text? | Validierungsbeleg |
| `state` | select | `new`, `building`, `ready`, `blocked`, `quarantined`, `deleted` |
| `error_code` / `error_digest` | text? | bei `blocked/quarantined` sichere Diagnose |
| `version` | SafeRevisionV1 | CAS-Version; steigt bei jeder Zustandskante |

`SearchProjectionComponentSetV1` ist kein freies JSON. Für
`route_radius_ux_v1` gilt `partition_kind = geo_route_shard`,
`component = geometry` und `input_components = [core, overlay, geometry]`:
Jeder physische Route-Index enthält die vollständige, für ACL, Text,
Metadatenfilter und das GeoJSON-Direct-Prädikat benötigte Suchprojektion. Er
ist kein isolierter Geometrieindex, der im Requestpfad mit einem anderen Index
gejoint werden dürfte. Pro `(generation, geo_shard_id)` existiert genau ein
solches Member; seine live Dokumentmenge ist eine Teilmenge der höchstens
1.000 dauerhaft diesem logischen Shard zugewiesenen Trails. Seine Family-
Relation und drei Contractfelder sind ab Memberanlage unveränderlich,
untereinander konsistent und stimmen exakt mit
`search_index_generation_query_family` überein. Schema-, Settings-, Bundle-
oder Markerhash allein ist kein Beleg für ADR 0002; Buildpayload, Coverage und
Endmarker müssen zusätzlich `contract_artifact_key` und
`contract_artifact_hash` enthalten.

Der Member-Automat ist mit dem Generationenautomaten gekoppelt:

| Von | Guard | Nach |
| --- | --- | --- |
| — | Member, physische ID und unveränderliche Engine-Inkarnation registriert | `new` |
| `new` | Provisionierungstask persistent; Parent in `provisioning/backfilling` oder das Member ist gemäss Abschnitt 8.1.1 in keinem publizierten Snapshot gebunden | `building` |
| `building` | Schema, Settings, Baseline/Cutoff, Coverage, Marker, Delivery- und Security-Watermark vollständig bestätigt | `ready` |
| `ready` | Member ist in keinem publizierten Snapshot gebunden und benötigt weiteren Vorab-Catch-up; Parent bleibt auf seiner alten Snapshotmenge `published` | `building` |
| `ready` | andernfalls wird im selben CAS ein Kandidat `ready/published -> catching_up/updating` oder die aktive Generation `published -> updating`; Serving ist dadurch fail-closed | `building` |
| `new/building/ready` | ausschliesslich transienter, identitätswahrender Betriebsfehler; Parent kann nicht auf eine dieses Member verlangende Epoch swappen/publizieren; ist es bereits im aktuellen Snapshot gebunden, kann er auch nicht antworten | `blocked` |
| `blocked` | derselbe physische Index und dieselbe Engine-Inkarnation; weitere Writes nötig | `building` |
| `blocked` | keine Writes nötig und sämtliche Ready-Gates neu bewiesen | `ready` |
| jeder nicht gelöschte Zustand | Identitäts-, Inkarnations-, Coverage-, Payload- oder Markerwiderspruch | `quarantined` |
| jeder nicht gelöschte Zustand | Parent ist GC-berechtigt, keine Resource-Zusage hält ihn offen und physische Löschung oder Nichtanlage ist bestätigt | `deleted` |

`blocked` ist retrybar, aber nie query- oder swapfähig. `quarantined` ist für
Serving und Wiederverwendung terminal; von dort ist nur die GC-Kante nach
`deleted` erlaubt. `deleted` ist vollständig terminal. Jede Kante prüft
Member-ID, `version`, Parentversion, Engine-Inkarnation und das aktuelle
Writer-Fence-Tupel in einem CAS. Bei `ready -> building`, `-> blocked` oder
`-> quarantined` muss derselbe Commit für ein bereits snapshotgebundenes
Member den Parent- und gegebenenfalls Headzustand so verengen, dass kein
Gateway den Member weiter liest. Ist das Member nachweislich in keinem
publizierten Snapshot gebunden, bleibt die alte Snapshotmenge querybar; der
Commit blockiert stattdessen jede Ziel-Epoch und jeden Swap, die dieses Member
benötigen.

Die einzige zustandserhaltende physische Mutation eines `ready`-Members ist der
unten definierte, bereits gatewayseitig wirksame `fence_apply`. Sein CAS erhöht
die Memberversion und gegebenenfalls `security_watermark` sowie
`delivered_revision` ausschliesslich über einen lückenlos bestätigten Prefix
aus terminalen `fenced`-Security- und vollständig `irrelevant`-Changes. Er
verändert weder Generation-Cutoff, Content-Epoch, Schema, Settings noch
irgendeinen weiterhin sichtbaren normalen Contentwert; Content-/Settings-
Deliveries bleiben verboten.

#### `search_generation_head`

Genau ein Datensatz pro `(contract, entity_kind)`, in V1 für
`(trail-search.v1, trail)`. Er ist die einzige atomare Serving-Wahrheit.

| Feld | Typ | Regel |
| --- | --- | --- |
| `contract` / `entity_kind` | text/`SearchEntityKindV1` | Unique-Tupel |
| `state` | select | `empty`, `active` |
| `active_generation` | relation? | bei `active` Lifecycle `active`; nur mit Epochzustand `published` querybar |
| `pointer_version` | SafeRevisionV1 | CAS-Version |
| `activated_at` | date? | lokale DB-Zeit der letzten Aktivierung |

Die Migration legt den Head als `empty`, ohne Relation und mit
`pointer_version = 0` an. Dieser Zustand liefert keine Suche und kann nur durch
den Bootstrap-Swap verlassen werden.

Der Gateway routet auf die generationenspezifischen physischen IDs aus dem
unveränderlichen Membermanifest des gebundenen Content-Snapshots. Ein
Meilisearch-Index-Swap ist für diese Member-IDs in V1 verboten, weil er Inhalte
hinter den generationengebundenen Identitäten vertauschen würde. Der einzige
Servingwechsel ist der PocketBase-Head-CAS; innerhalb einer Generation kann
nur ein neuer Epoch-Snapshot eine appendierte Shardfamilie sichtbar machen.

#### `search_content_snapshot`

Unveränderlicher Beleg einer publizierten generationengebundenen Content-Epoch.
Unique-Keys: `snapshot_key` und `(generation, content_epoch)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `snapshot_key` | text | opak, Unique und niemals wiederverwendet |
| `generation` | relation | gemeinsam retenierte Indexgeneration |
| `content_epoch` | SafeRevisionV1 | publizierte Epoch dieser Generation |
| `delivered_catalog_revision` | SafeRevisionV1 | bestätigter Contentprefix |
| `bundle_hash` / `schema_hash` / `settings_hash` | text | exakt bestätigte Generationmanifeste |
| `federation_manifest_hash` | text | Hash der zugehörigen Scope-Bindings |
| `geo_shard_manifest_version` | relation? | neueste Route-Shard-Manifestversion am Contentprefix; genau bei gebundener Route-Familie gesetzt |
| `geo_shard_manifest_hash` | text? | exakt deren unveränderlicher Manifesthash |
| `query_family_manifest_hash` | text | Hash der vollständigen geordneten `search_content_snapshot_query_family`-Menge; enthält die Route-Family auch bei katalogweit null Assignments |
| `member_manifest_hash` | text | Hash der vollständigen geordneten `search_content_snapshot_member`-Menge |
| `published_at` | date | lokale DB-Zeit des Epoch-CAS |
| `retention_resource` | relation | kanonische Cursor-/GC-Zusage dieses Snapshots |

Security ist absichtlich kein unveränderlich gepinnter Payloadbestandteil dieses
Records. Der öffentliche Suchkontext ergänzt den bei Ausstellung aktuellen
`security_watermark`; aktuelle Fences werden bei jeder Seite erneut geprüft.

#### `search_content_snapshot_query_family`

Unveränderliche Queryfamily-Bindung einer publizierten Epoch. Unique-Keys:
`(content_snapshot, query_family)` und
`(content_snapshot, generation_query_family)`. Der Epoch-CAS erzeugt für jede
Generation-Queryfamily genau eine Zeile, insbesondere für
`route_radius_ux_v1` auch beim unbelegten initialen Shard.

| Feld | Typ | Regel |
| --- | --- | --- |
| `content_snapshot` / `generation_query_family` | relation/relation | gemeinsam retenierte Epoch und exakt passende Generation-Family-Bindung |
| `query_family` | `SearchIndexQueryFamilyV1` | aus der Generation-Family kopiert |
| `contract_artifact` | relation? | für `route_radius_ux_v1` verpflichtetes gemeinsam reteniertes Artefakt |
| `contract_artifact_key` / `contract_artifact_hash` | text?/text? | unveränderliche Kopie der Generation-Family-Bindung |
| `member_count` | SafeRevisionV1 | Anzahl der Snapshot-Member exakt dieser Queryfamilie; für `route_radius_ux_v1` gleich der stets positiven Shardanzahl des Geo-Manifests |
| `query_family_member_manifest_hash` | text | Hash der kanonisch geordneten Snapshot-Memberteilmenge; für die Route-Family enthält sie auch das Member des belegungsleeren initialen Shards |
| `geo_shard_manifest_version` / `geo_shard_manifest_hash` | relation?/text? | für `route_radius_ux_v1` immer gesetzt und bindet mindestens den initialen Shard Ordinal `0` |
| `binding_hash` | text | Digest aller vorstehenden unveränderlichen Family-, Contract- und Manifestwerte |

Für `route_radius_ux_v1` ist die durch `query_family_member_manifest_hash`
gebundene Teilmenge bijektiv zu **allen** Shards der gebundenen
Geo-Manifestversion, unabhängig von deren `assigned_slots`. Beim leeren
Katalog enthält sie deshalb genau das leere physische Route-Member für Shard
Ordinal `0`; `member_count = 1`, Membermanifesthash, Contract-Relation, Key und
Hash bleiben verpflichtend. `trail_default` besitzt eine eigene
Family-Zeile und bleibt zugleich Teil des vollständigen
`search_content_snapshot.member_manifest_hash`. Die kanonisch geordnete
Gesamtmenge aller Family-Zeilen ist exakt durch
`search_content_snapshot.query_family_manifest_hash` gebunden und bijektiv zur
durch `search_index_generation.query_family_manifest_hash` gebundenen
Generation-Family-Menge; Family-IDs und snapshotabhängige Bindingbytes werden
dabei gemäss ihrem jeweiligen Serializer getrennt gehasht.

#### `search_content_snapshot_member`

Unveränderliche Memberbindung einer publizierten Content-Epoch. Unique-Keys:
`(content_snapshot, generation_member)` und
`(content_snapshot, query_family, component, partition_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `content_snapshot` / `snapshot_query_family` / `generation_member` | relation/relation/relation | gemeinsam retenierte Epoch, passende Snapshot-Family und physisches Member |
| `query_family` / `component` | `SearchIndexQueryFamilyV1`/`SearchProjectionComponentV1` | aus dem Member kopiert |
| `partition_kind` / `partition_key` | `SearchIndexPartitionKindV1`/text | unveränderliche logische Partition |
| `geo_shard_id` | text? | genau bei `geo_route_shard`, identisch zu Member und Manifest |
| `physical_index_id` / `engine_incarnation` | text/text | generationengebundene physische Identität am Publish-CAS |
| `schema_hash` / `settings_hash` | text/text | bestätigte memberspezifische Verträge |
| `contract_artifact` | relation? | bei `query_family = route_radius_ux_v1` verpflichtetes gemeinsam reteniertes Artefakt, sonst in V1 leer |
| `contract_artifact_key` / `contract_artifact_hash` | text?/text? | unveränderliche Kopie aus `snapshot_query_family`; beim Route-Member identisch zur Generation-Family |
| `delivered_revision` / `security_watermark` | SafeRevisionV1/SafeRevisionV1 | im Epoch-CAS bestätigte monotone Memberstände |
| `marker_hash` | text | bestätigter Endmarker dieses physischen Indexstands |

Der Epoch-CAS legt Snapshot, sämtliche Snapshot-Queryfamily-Bindungen, seine
vollständige Membermenge aus allen Queryfamilien und `member_manifest_hash`
atomar an. Nur die daraus nach
`query_family = route_radius_ux_v1` gefilterte Teilmenge entspricht bijektiv
allen Shards unabhängig von `assigned_slots` in der gebundenen
`GeoRouteShardManifestV1`: kein Route-Member fehlt, keines kommt doppelt oder
aus einer späteren Manifestversion vor. `trail_default` bleibt Teil der
vollständigen Membermenge und ihres Hashes, wird aber nie gegen das
Geo-Shardmanifest gezählt. Jedes gebundene Member ist `ready`, gehört
derselben Generation und Engine-Inkarnation an und bestätigt Delivery,
Security, Schema, Settings und Marker für den Snapshot-Cutoff. Jedes gebundene
Route-Member kopiert zusätzlich Relation, Key und Hash aus der zugehörigen
Snapshot-Queryfamily; diese stimmt mit der Generation-Family überein. Bei
leerem Katalog existiert genau das leere Member des initialen Shards; seine
Snapshot-Family bindet Contract und Geo-Manifest vollständig. Ein später zur Generation
hinzugefügtes Member erweitert einen bestehenden Snapshot nie.

Gateway, Cursor, Retention und Fence-Proofs lösen zuerst die unveränderliche
Snapshot-Queryfamily und danach deren durch den Teilmengenhash gebundene
Memberrecords auf, niemals durch einen Scan aller aktuell vorhandenen
Generationenmember. Snapshot-Family, Snapshot-Member, gebundene
Shardmanifestversion und deren Assignmentbelege werden mindestens gemeinsam
mit der Snapshotressource reteniert.

#### `search_content_snapshot_scope`

Unveränderliches Federation-Binding einer publizierten Content-Epoch.
Unique-Key: `(content_snapshot, scope_kind, scope_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `content_snapshot` | relation | gemeinsam retenierter Epochbeleg |
| `scope_kind` / `scope_key` | `FederationScopeKindV1`/text | vollständiger gebundener Local-, Global-, Origin-, Actor-, ACL- oder Publication-Scope |
| `source_scope_version` | relation | neueste Scope-Version mit `catalog_revision <= delivered_catalog_revision` |
| `snapshot_set_revision` | SafeRevisionV1 | aus der referenzierten Scope-Version dupliziert |
| `earliest_discoverable_until` | date? | aus `source_scope_version` kopiertes exaktes Minimum; leer genau bei epochgebunden leerer Mitgliedschaft |
| `scope_digest` | text | muss Scope-Version und Federationmanifest entsprechen |

Der Publisher erzeugt diese Records im finalen Epoch-CAS für Local- und
Global-Scope, jede in diesem Snapshot bekannte Origin und alle darin
referenzierten Actor-, ACL- und Publication-Scopes. Er bestätigt den Digest
gegen genau die gelieferten Projektionsartefakte; der aktuelle mutable
Scope-Head darf dabei keinen Stand nach dem Content-Cutoff einschleusen. Fehlt
für einen angeforderten oder durch Actor/ACL/Publication implizit betroffenen
Scope das Bindungsrecord oder ist dessen Horizont bereits erreicht, kann diese
Epoch keinen neuen Kontext für diesen Scope ausstellen.

Der finale Epoch-CAS prüft zusätzlich die Iff-Bedingung des Horizonts: Eine
epochgebunden nichtleere Membership besitzt ein gesetztes
`earliest_discoverable_until`, das exakt dem Minimum aller im Scope-Digest
gebundenen Expiries und dem Wert der referenzierten Scope-Version entspricht;
bei leerer Membership sind alle drei Darstellungen kanonisch leer. Eine
fehlende, zu späte oder abweichende Minimum-Bindung blockiert die Publikation
fail-closed.

#### `search_publisher_state`

Genau ein Datensatz pro Writepfad. Unique-Key: `scope_key`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `scope_key` | text | in V1 `trail-search.v1` |
| `restore_epoch` | SafeRevisionV1 | muss dem gesunden Control-Plane-Restore-State entsprechen |
| `term` | SafeRevisionV1 | monotoner logischer Writer-Term innerhalb Restore-/Credential-Epoche |
| `holder` | text? | opake Prozess-/Instanz-ID |
| `lease_until` | date? | nur Koordination, kein Engine-Fence |
| `credential_epoch` | SafeRevisionV1 | bestätigte Infrastruktur-/Credentialgeneration |
| `fence_evidence_hash` | text? | externer Revocation-/Proxy-/Netzpolicybeleg beim Failover |
| `state` | select | `vacant`, `held`, `draining`, `fenced`, `quarantined` |
| `version` | SafeRevisionV1 | CAS-Version |
| `transitioned_at` | date | vom Transition-Service gepflegt |

Generation, Engine-Attempt und dieser Record müssen beim Claim dasselbe
aktuelle `(restore_epoch, credential_epoch, term)`-Tupel bestätigen.
`search_publisher_state` ist
die persistente Term-/Holder-Wahrheit. Er dokumentiert Koordination und
Infrastruktur-Fencing, ersetzt dieses aber nicht.

#### `search_engine_attempt`

Unique-Keys: `attempt_key` und `(generation, attempt_kind, sequence)`. Für
`attempt_kind = epoch_publish` erzwingt ein zusätzlicher partieller Unique-Index
`(generation, target_epoch)`. Ein Retry nimmt denselben Record wieder auf, statt
einen zweiten Versuch für denselben physischen Übergang anzulegen.

| Feld | Typ | Regel |
| --- | --- | --- |
| `attempt_key` | text | Unique, deterministisch, namespaced und über Recovery hinweg stabil |
| `generation` | relation | Zielgeneration |
| `attempt_kind` | select | `build_batch`, `catch_up`, `epoch_publish`, `fence_apply`, `validation` |
| `visibility_fence` | relation? | genau bei `fence_apply` gesetzt; unveränderlicher primärer Auslöser dieses Security-Attempts, nicht dessen alleiniger physischer Zielstand |
| `fence_apply_frontier_binding` | `FenceApplyFrontierBindingSetV1`? (json) | genau bei `fence_apply` gesetzt; vollständiger unveränderlicher Frontier-Abschluss aller mutierten physischen Ziele |
| `sequence` | SafeRevisionV1 | monoton je Generation und Attempt-Art |
| `created_writer_restore_epoch` / `created_writer_credential_epoch` / `created_writer_term` | SafeRevisionV1/SafeRevisionV1/SafeRevisionV1 | unveränderliches erstes Owner-Fence-Tupel |
| `writer_restore_epoch` / `writer_credential_epoch` / `writer_term` | SafeRevisionV1/SafeRevisionV1/SafeRevisionV1 | aktueller Recovery-Owner; muss Generation und Publishertupel entsprechen |
| `from_epoch` / `target_epoch` | OptionalSafeRevisionV1 | bei `epoch_publish` beide gesetzt und exakt `target = from + 1` |
| `from_catalog_cutoff` | OptionalSafeRevisionV1 | bei `epoch_publish` exakt der im Anlage-CAS gelesene bisherige Generation-Cutoff |
| `member_watermark_baseline` | `AttemptMemberWatermarkBaselineSetV1` (json) | bei jedem mutierenden Attempt vollständige unveränderliche Baseline aller Zielmember |
| `target_member_manifest_hash` | text | Digest der vollständigen geordneten Zielmembermenge des Attempts |
| `target_query_family_manifest_hash` | text | Digest der vollständigen geordneten Generation-Queryfamily-Bindungen des Attempts, unabhängig von der Memberzahl |
| `target_geo_shard_manifest_version` | relation? | bei Attempts mit Route-Familie cutoffgenaue persistente Shardmanifestversion |
| `target_geo_shard_manifest_hash` | text? | genau dann deren unveränderlicher Hash |
| `target_route_query_family` | relation? | bei Attempts mit `route_radius_ux_v1` deren immutable Generation-Family-Bindung, unabhängig von der Dokumentbelegung ihrer Member |
| `target_route_contract_artifact` | relation? | genau bei gebundener `route_radius_ux_v1`-Family deren immutable Contract-Artefakt, auch beim belegungsleeren initialen Member |
| `target_route_contract_artifact_key` / `target_route_contract_artifact_hash` | text?/text? | gemeinsam gesetzte unveränderliche Kopie von `artifact_key`/`payload_hash` |
| `target_cutoff` | SafeRevisionV1 | bei `fence_apply` exakt der unveränderte Generation-Cutoff als No-Content-Change-Guard; sonst bei Attemptanlage inputbereiter Zielprefix, mindestens bisheriger Cutoff und eingefrorene Delivery-/Security-Watermarks, vor Content-Commit ausgelieferter geschlossener Memberprefix |
| `target_security_revision` | OptionalSafeRevisionV1 | genau bei `fence_apply` gesetzt; Maximum der gebundenen Restriction-Frontier-Revisionen und mindestens `visibility_fence.set_revision`; bei allen Content-Attempts nicht gesetzt |
| `manifest_max_input_revision` | SafeRevisionV1 | Maximum aller in Taskpayloads/Settings gebundenen Katalogstände; bei Content-Attempts höchstens `target_cutoff`, bei `fence_apply` exakt `target_security_revision` |
| `manifest_hash` | text | vollständige geordnete Taskintents |
| `post_baseline_security_proof` | `AttemptPostBaselineSecurityProofSetV1` (json) | vor Abschluss `[]`; beim Content-Commit exakt die Delivery-/Security-Watermark-Vorläufe über `target_cutoff`, ausschliesslich durch später entstandene aktive Gateway-Fences und irrelevante Zwischenchanges belegt |
| `state` | select | `prepared`, `submitting`, `waiting`, `verifying`, `committed`, `superseded`, `recovery`, `failed`, `abandoned`, `quarantined` |
| `started_at` / `finished_at` | date/date? | lokale Zeiten |
| `error_code` / `error_digest` | text? | sichere Diagnose |

Alle Attempt-Arten verwenden den Lifecycle aus Abschnitt 8.2. Nur
`epoch_publish` darf den Epoch-CAS ausführen. Build-, Catch-up-, Fence- und
Validation-Attempts bestätigen dagegen ausschliesslich ihre Tasks, Marker und
Watermarks; sie publizieren keinen querybaren Zwischenstand.

`fence_apply` ist ein enger Security-Sonderpfad. Sein primärer
`visibility_fence` muss noch nicht `retired`, im Zustand `active`,
`propagating` oder `satisfied` und mit `gateway_guard_kind = exact_target` vor
Ranking, Total und Counts wirksam sein. `clear_revision`,
`guard_release_revision` und die Successor-Kette dieses Fence müssen leer
sein. Insbesondere darf ein nur durch `query_fenced` zufriedengestelltes Ziel
bis zum physischen Apply weiter retried werden.

Der primäre Fence ist nur Auslöser, nicht isolierter Schreibpayload. Der
Attempt leitet für jedes mutierte physische Dokument, ACL-Overlay oder
Indexziel den vollständigen aktuellen Subject-/ACL-/Origin-Abhängigkeits-
Closure und darin die vollständige Menge aller wirksamen
`restriction_present`-Köpfe ab. Deren kumulative Zielstände müssen
`restricted/deleted` sein; kein im Closure relevanter Kopf darf
`clear_pending` sein und kein für das Ziel noch wirksamer Guard darf
`conservative_subject_exclusion` verwenden. Die Tasks materialisieren den
vollständigen autoritativen Restrictionstand dieser Menge, niemals bloss den
älteren Einzelstand des primären Fence. Der Zielstand muss zur gesamten
aktuellen Querysemantik aus physischem Baselinezustand und allen wirksamen
`exact_target`-Guards vollständig beobachtungsäquivalent sein. Der typisierte
Proof deckt für alle relevanten Principals Treffer, Ranking, Total, Facetten,
Histogramme, Highlights, Matchgründe und Response-DTO ab; „mindestens ebenso
restriktiv“ oder ein einzelner Dokumentcount genügt nicht.

Die Tasks dürfen weder Content/Settings noch Sichtbarkeitsfelder ausserhalb
der im Binding vollständig aufgezählten physischen Ziele ändern. Der Attempt
läuft unter demselben serialisierten Writer-Fence-, Submission-, Endmarker-
und Barriervertrag, darf aber ein `ready`-Member einer `published`en aktiven
oder versiegelten Generation zustandserhaltend aktualisieren. Sein gültiger
Abschluss setzt nur die explizit vom kumulativen Proof abgedeckten Fence- und
terminalen Security-Deliveries sowie monotone Security-Watermarks;
Content-Epoch, `catalog_cutoff` und der contentbezogene Stand bleiben
unverändert. `delivered_revision` darf dabei ausschliesslich über den
lückenlos belegten Securityprefix aus terminalen `fenced`-Changes und
vollständig irrelevanten Zwischenchanges vorrücken; ein normaler Content- oder
Settings-Deliveryeffekt ist verboten. Ein `conservative_subject_exclusion`-Fence oder eine physische
Projektion, unter der ein temporär ganz ausgeschlossenes Subject wieder
sichtbar wird, ist niemals `fence_apply`: Sie läuft über einen normalen
`epoch_publish`, macht ältere Cursor stale und behält bis
`guard_release_converged` den Issuance-Fence. Ein Grant/Clear ist ebenfalls
niemals `fence_apply`.

Bei der Attemptanlage gilt abweichend vom Contentpfad: `target_cutoff` ist
exakt der unveränderte aktuelle Generation-Cutoff. Der CAS bindet den
primären Fence samt aktueller Version und leerer Clear-/Release-/Successorlage
sowie den vollständigen `FenceApplyFrontierBindingSetV1`.
`target_security_revision` ist das Maximum der darin gebundenen
`frontier_revision`-/`security_watermark`-Werte, mindestens
`visibility_fence.set_revision`; `manifest_max_input_revision` ist exakt
dieser Wert. Der Abschluss beweist einen geschlossenen Securityprefix bis
dorthin und bestätigt erneut, dass Content-Cutoff, Content-Epoch und normale
Contentinputs bytegleich geblieben sind. Weder ein höherer Contentprefix noch
eine versteckte normale Projektionsmutation darf aus diesem Sonderziel
abgeleitet werden.

`FenceApplyFrontierBindingSetV1` ist eine geschlossene, migrationsgebundene
Struktur. Sie enthält den primären Fence mit ID, Version, semantischem Digest
und explizit leerem Successor-Tail sowie je physischem Ziel dessen stabile
Identität, Visibility-Regelartefakt, Baseline-Visibilitymarker und die
kanonisch sortierte **vollständige** Menge aller anwendbaren Restriction-
Frontiers. Jeder Frontier-Eintrag bindet Record-ID, Subject-Kind/-Key,
`version`, `frontier_revision`, `security_watermark`, Zustand,
Authority-/Restriction-/Source-Manifest- und Gateway-Guard-Set-Digest. Ein
versionierter `dependency_closure_hash` bindet ausserdem die vollständige
Subject-/ACL-/Origin-Abhängigkeitsauflösung; dadurch ist auch das Hinzukommen
eines zuvor nicht vorhandenen überlappenden Fence eine Mengenänderung.
Unrestricted oder noch gar nicht materialisierte Subjects erscheinen nicht als
Enginepayload, ihre aufgelöste Abwesenheit einer Restriction ist aber im
Closure-Hash gebunden. Daher bleibt `target_security_revision` das Maximum der
tatsächlich payloadtragenden Restriction-Frontiers; der Closure ist ein
Submit-/CAS-Guard und kein vorgetäuschter ausgelieferter Katalogprefix. Das
Attemptmanifest hasht die bytegenaue Struktur, jedes Taskintent bindet die
Teilmenge seiner physischen Ziele, und kein freies JSON kann einen fehlenden
Frontier ersetzen.

Attemptanlage, die kurze Transaktion unmittelbar vor **jedem** Engine-Submit
und der Abschluss-CAS lösen diese Abhängigkeitsmenge aus dem jeweils aktuellen
PocketBase-Schnitt erneut auf. Sie verlangen exakte Gleichheit aller
Bindingfelder, den weiterhin nicht-retirten primären Fence mit unveränderter
leerer Clear-/Release-/Successorlage und weiterhin ausschliesslich
`exact_target`-Guards. Eine spätere stärkere Restriction, ein überlappender
Actor-/Origin-/ACL-Fence oder ein Grant/Clear lässt den CAS verlieren. Vor dem
ersten möglichen Engineeffekt darf der Attempt dann `abandoned` werden;
bereits angelegte, noch `prepared`e Submissionrecords wechseln zuvor mit
`guard_canceled_before_network`-Proof terminal nach `canceled`. Nach einem
möglichen Engineeffekt darf er weder `applied`-Deliveries noch einen Watermark
aus dem veralteten Binding committen: Zuerst müssen alle möglichen Submissions
und der geordnete Endmarker terminal oder durch eine bestätigte Barriere
abgedeckt sein. Ist der tatsächlich geschriebene alte Effekt vollständig
bekannt, endet nur `fence_apply` als `superseded`; ist er unbeweisbar, folgt
`recovery` beziehungsweise `quarantined`.

`superseded` lässt sämtliche betroffenen Gateway-Guards wirksam und plant
unter dem serialisierten Writepfad die aktuelle Reparatur: einen neuen
`fence_apply` für einen weiterhin rein verengenden exakten Frontier oder einen
normalen `epoch_publish` bei Clear, konservativem Guard oder sonstiger
Erweiterung. Bis deren aktueller Marker bestätigt ist, darf kein betroffener
Fence retiren. So kann weder ein verspäteter schwächerer F1-Write eine bereits
physisch ausgelieferte stärkere F2-Restriction überschreiben noch ein alter
Restrictionwrite hinter einem Clear dauerhaft liegenbleiben.

`AttemptMemberWatermarkBaselineSetV1` ist eine geschlossene,
migrationsgebundene und nach Member-ID sortierte Liste aus exakt
`generation_member`, dessen `version`, Engine-Inkarnation,
`delivered_revision`, `security_watermark` und bei Route-Membern dessen
Contract-Artefakt-Relation, -Key und -Hash. Sie ist kein freies Guard-JSON.
Der Attempt-Anlage-CAS validiert und friert die vollständige durch
`target_member_manifest_hash` bezeichnete Zielmenge ein; Recovery verwendet
exakt dieselbe Baseline. Der Hash enthält weiterhin jedes `trail_default`-
Zielmember. `target_query_family_manifest_hash` bindet davon unabhängig die
vollständige, nach Queryfamilie sortierte Menge der immutable
`search_index_generation_query_family`-Zeilen. Enthält sie
`route_radius_ux_v1`, sind `target_route_query_family`, Contractrelation,
Key/Hash und Geo-Manifest unabhängig von der Dokumentbelegung verpflichtend.
Die nach Queryfamilie gefilterte Zielteilmenge enthält mindestens das Member
des initialen Shards und ist gegen **alle** Einträge von
`target_geo_shard_manifest_version/hash` bijektiv. Ein nach
Attemptanlage registriertes späteres Shardmember, ein Contract-Artefaktwechsel
oder ein zwischenzeitlicher Manifestfortschritt lässt den Abschluss-CAS
verlieren.

`AttemptPostBaselineSecurityProofSetV1` ist ebenfalls geschlossen,
migrationsgebunden und nach Member, Change-Revision/-Ordinal, Fence-ID und
`set_revision` sortiert. Es enthält genau die Schritte, durch die beim
Content-Commit `delivered_revision` und/oder `security_watermark` über
`target_cutoff` vorausliegen. Je Member bindet es Baseline und aktuelle beide
Watermarks sowie zwei getrennte, geschlossene Achsen:

- `delivery_prefix_entries` enthält für jede relevante Katalogrevision im
  offenen Intervall `(target_cutoff, delivered_revision]` Commit-/Change-ID,
  Resolution und terminalen Deliverybeleg. Zulässig sind dort ausschliesslich
  vollständig belegtes `irrelevant` oder ein Security-Change mit
  `search_catalog_delivery.state = fenced`; Content-/Settings-Apply, Delete
  oder Subsumption oberhalb des Zielcutoffs sind verboten.
- `security_frontier_entries` enthält unabhängig davon jeden Security-Change
  im offenen Intervall `(target_cutoff, security_watermark]` samt Fence- und
  Fence-Deliverybeleg **oder** dessen vollständigem typisiertem
  `irrelevant`-Beleg für genau dieses Member. Diese Liste bleibt auch dann
  vollständig, wenn ein dazwischen offener normaler Katalogchange den
  allgemeinen `delivered_revision` noch bei oder unter `target_cutoff` hält.

Jeder Fence-Eintrag beider Achsen bindet den nach Attemptanlage hinzugekommenen Fence mit
ID, Version, Revision, Fence-Delivery-ID/-Proofdigest und Gateway-Regeldigest.
Der Fence muss beim finalen CAS weiterhin nicht retirte `active`,
`propagating` oder `satisfied` sein, seine Memberdelivery weiterhin
`query_fenced`, und sein Guard muss das Subject vor jeder Query vollständig
schützen. Der serialisierte Writepfad garantiert, dass sein physischer Apply
erst nach dem abschliessenden Contentwrite erfolgen kann. Ein bereits bei
Attemptanlage bekannter Vorlauf, ein retirter Fence, eine Lücke im
Deliveryprefix oder ein frei formuliertes JSON erfüllt diesen Proof nie. Das
Feld wird ausschliesslich im finalen Content-Commit gesetzt und ist danach
unveränderlich.

Ein `irrelevant`-Eintrag beider Achsen bindet stattdessen Change-/Commitdigest,
Member/Komponente, `SearchIrrelevanceProofKindV1`, Regelversion, Vorher-/
Nachher-Inputfingerprints und den unveränderten Irrelevanz-Proofhash aus
Abschnitt 4.1. Er darf keinen Fence vortäuschen und ist nur zulässig, wenn der
Security-Change die physische und gatewayseitige Sichtbarkeit dieses Members
nachweislich nicht beeinflusst. Damit ist auch ein reiner irrelevanter
Security-HWM-Vorlauf geschlossen rekonstruierbar.

#### `search_engine_task`

Unique-Keys: `(engine_attempt, ordinal)` und `intent_key`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `engine_attempt` / `generation_member` | relation/relation | Intentziel |
| `ordinal` | SafeRevisionV1 | Einreichungsreihenfolge |
| `operation` | select | `settings`, `upsert`, `delete`, `verify`, `barrier` |
| `intent_key` | text | Hash aus Attempt-Key, Engine-Inkarnation, Member, Ordinal und `intent_digest` |
| `intent_digest` | text | deterministische idempotente Absicht |
| `input_revision_ceiling` | SafeRevisionV1 | Maximum aller durch diesen Intent gelesenen Projektions-, Tombstone-, Settings- und sonstigen Katalogrevisionen |
| `request_payload_hash` | text | Hash der vollständigen kanonischen Enginerequest-Payload |
| `request_payload` | json | grössenbegrenzte vollständige Payload oder typisierte Liste unveränderlicher Artefakt-IDs |
| `expected_end_marker_hash` | text? | bei `barrier/verify` verpflichteter kanonischer Sollmarker des gesamten relevanten Intentprefixes |
| `confirmed_end_marker_hash` | text? | nur nach Engine-Readback; muss für Erfolg dem Sollmarker entsprechen |
| `engine_incarnation` | text | Restore-/Reset-Fence |
| `state` | select | `prepared`, `submitted`, `succeeded`, `failed`, `canceled`, `unknown` |
| `submission_count` | SafeRevisionV1 | Anzahl persistierter physischer Submits |
| `cancellation_proof_hash` | text? | bei Taskzustand `canceled` verpflichteter Digest des effektfreien Abandon- beziehungsweise terminalen Child-Submission-Belegs |
| `submitted_at` / `finished_at` | date? | lokale Zeiten |
| `error_code` / `error_digest` | text? | sichere Diagnose |

`input_revision_ceiling` ist kein frei gewähltes Limit: Er ist der exakte
Maximalwert der im unveränderlichen Request referenzierten
`built_through_revision`, Change-/Tombstone- und kataloggebundenen
Settingsrevisionen; ein Intent ohne Kataloginput verwendet `0`. Der
Attemptwert ist exakt das Maximum seiner Tasks. Für jeden normalen Content-
oder Settings-Task muss
`input_revision_ceiling <= engine_attempt.target_cutoff` gelten. Ein
`fence_apply`-Task darf stattdessen genau die maximale Revision seiner im
`FenceApplyFrontierBindingSetV1` gebundenen Restriction-Frontiers als Ceiling
tragen; das Maximum aller dieser Tasks ist exakt
`target_security_revision`. Er darf keine ungebundenen Kataloginputs aufnehmen
und verändert den Content-Cutoff nicht.

#### `search_engine_submission`

Jeder physische HTTP-Submit besitzt einen eigenen Record. Unique-Keys:
`(engine_task, submit_ordinal)` sowie partiell für nichtleere IDs
`(engine_incarnation, engine_task_id)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `engine_task` | relation | unveränderlicher Intent |
| `submit_ordinal` | SafeRevisionV1 | beginnt bei 0, lückenlos pro Intent |
| `restore_epoch` / `credential_epoch` / `publisher_term` | SafeRevisionV1/SafeRevisionV1/SafeRevisionV1 | tatsächlich verwendetes Writer-Fence-Tupel |
| `engine_incarnation` | text | muss Intent, Member und Generation entsprechen |
| `engine_task_id` | text? | physische Engine-ID, sobald bekannt |
| `state` | select | `prepared`, `submitting`, `accepted`, `succeeded`, `failed`, `canceled`, `unknown`, `barriered` |
| `barrier_task` | relation? | bei `barriered` verpflichtender späterer `barrier`-Task desselben serialisierten Memberwritepfads |
| `barrier_proof_hash` | text? | bei `barriered` verpflichteter Adapterbeleg, der Submission, Barrier und Endmarker bindet |
| `non_submission_proof_kind` / `non_submission_proof_hash` | `EngineNonSubmissionProofKindV1`?/text? | gemeinsam nur für `prepared -> canceled`; bindet den verlorenen Submit-Guard und die beweisbare Nichtausführung des Netzwerkaufrufs |
| `non_acceptance_proof_kind` / `non_acceptance_proof_hash` | `EngineNonAcceptanceProofKindV1`?/text? | gemeinsam nur für `submitting -> failed/canceled`; bindet den versionierten Adaptervertrag und die beweisbare synchrone Nichtannahme |
| `terminal_no_effect_proof_kind` / `terminal_no_effect_proof_hash` | `EngineTerminalNoEffectProofKindV1`?/text? | optional gemeinsam bei `accepted -> failed/canceled`; für jeden Resubmit verpflichteter versionierter Beleg, dass die terminale Engine-Task keinen Teileffekt hinterliess |
| `prepared_at` / `submitted_at` / `finished_at` | date/date?/date? | lokale Zeiten |
| `response_digest` / `error_code` / `error_digest` | text? | sichere Diagnose |

Vor dem Netzwerkaufruf entsteht `prepared`; unmittelbar davor bestätigt der
Submit-Guard den Term und setzt `submitting`. Die Antwort ergänzt Task-ID und
`accepted`. Der Intent gilt erst als `succeeded`, wenn jede möglicherweise
angenommene Submission entweder selbst terminal bestätigt oder durch einen vom
Engineadapter beweisbaren späteren Barrier- und Endzustandsmarker abgedeckt ist.
Eine einzelne „neueste“ Task-ID genügt nie.

```text
Submission: prepared -> canceled  # nur mit Non-Submission-Proof vor Netzwerk
             prepared -> submitting -> accepted -> succeeded | failed | canceled
                            |            \
                            |             +-> unknown
                            +-> failed | canceled  # nur mit Non-Acceptance-Proof
                            +-> unknown
             unknown -> barriered | Generation-Quarantäne
Intent:     prepared -> canceled  # nur effektfreier Non-Epoch-Abschluss ohne mögliche Submission
            prepared -> submitted -> succeeded | failed | canceled | unknown
                                      unknown -> succeeded nur mit Barrierbeweis
```

Auf Submissionebene sind `succeeded`, `failed`, `canceled` und `barriered`
terminal; `unknown` kann ausschliesslich mit dem folgenden vollständigen Beleg
nach `barriered` wechseln, andernfalls bleibt der Record `unknown`. Ein
direktes `prepared -> canceled` ist nur im selben lokalen CAS wie ein
fehlgeschlagener Submit-Guard zulässig: `submitted_at` und `engine_task_id`
bleiben leer, kein Netzwerkaufruf wurde begonnen, und der typisierte
`guard_canceled_before_network`-Hash bindet Submission-/Attempt-ID,
Requestpayload, Writer-Tupel sowie den verlorenen Frontier-/Term-/State-Guard.
Damit ist der Record terminal beweisbar effektfrei. Non-Submission-,
Non-Acceptance- und Terminal-No-Effect-Prooffelder sind gegenseitig exklusiv;
ein `canceled`-Status ohne die zu seiner Herkunftskante passende Feldkombination
ist eine Invariantverletzung. Ein
direktes `submitting -> failed/canceled` ist nur zulässig, wenn der versionierte
Engineadapter aus der synchronen Antwort beweist, dass kein Task angenommen
und kein Teileffekt möglich war. Der Non-Acceptance-Proof bindet Adaptervertrag,
Engine-Inkarnation, Requestpayloadhash, HTTP-Status und Response-Digest;
Timeout, Verbindungsabbruch oder eine nur vermutete Ablehnung genügen nie und
führen nach `unknown`. `accepted -> unknown` bildet einen zunächst bekannten,
später wegen Task-GC, Engine-Reset oder widersprüchlichem Marker nicht mehr
beweisbaren Effekt ab. Auch diese Kante persistiert Fehlerklasse und Digest und
öffnet keinen direkten Retrypfad.

Nach Failover darf der neue Holder dieselbe `prepared -> canceled`-Kante
ausführen, wenn der Record unverändert `prepared` ist und das technische Fence
dem alten Writer jeden Übergang nach `submitting` unmöglich macht. Der
Non-Submission-Proof bindet dann zusätzlich `fence_evidence_hash` und das neue
Writer-Tupel. Ohne diesen Infrastrukturbeleg darf Recovery aus einem bloss
alten `prepared`-Record keine Nichtausführung raten.

Ein `accepted -> failed/canceled` darf den beobachteten terminalen Enginestatus
auch ohne Retrybeleg festhalten. Eine neue Submission desselben Intents ist
danach jedoch nur mit gemeinsam gesetztem
`terminal_no_effect_proof_kind/hash` zulässig. Der Hash bindet den
versionierten Adaptervertrag, Engine-Inkarnation und -Task-ID, Intent- und
Payloadhash, terminalen Status sowie den erforderlichen Marker-/Readbackbeleg,
der einen Teileffekt ausschliesst. Fehlt dieser persistierte Beleg, bleibt ein
Resubmit nach Crash, Failover oder Adapterwechsel verboten und der Attempt geht
in Recovery beziehungsweise Quarantäne.

Ein terminaler `failed/canceled`-Submissionrecord setzt den übergeordneten
Taskintent noch nicht automatisch terminal: Solange die Submission den zustandsrichtigen
persistierten Non-Submission-, Non-Acceptance- beziehungsweise
Terminal-No-Effect-Proof trägt, das unveränderte Intent unter allen aktuellen
Attemptguards noch gültig ist und der Retryguard unten gilt, bleibt der Task
`submitted` und erhält einen neuen lückenlosen Submissionrecord. So kann etwa
ein nach Termwechsel lokal gecancelter, nie gesendeter Record nach erfolgreichem
`adopt_recovery_attempt` sicher durch denselben Intent fortgesetzt werden.

Ist dagegen der Payload selbst durch einen verlorenen Frontier-/Manifestguard
überholt, ist ein Resubmit desselben Intents verboten. Sind **alle** Tasks und
Submissions dieses nicht epochgebundenen Attempts beweisbar ohne möglichen
Engineeffekt, setzt ein gemeinsamer Abschluss-CAS alle noch offenen
Taskintents terminal auf `canceled` und den Attempt auf `abandoned`; er plant
gegebenenfalls den aktuellen Ersatz. Andernfalls setzt erst die bewusste
Aufgabe des Intents den Task per CAS auf das terminale `failed` oder
`canceled` und den Attempt in seinen Recoverypfad. Auf Taskebene sind
`succeeded`, `failed` und `canceled` terminal; `unknown` besitzt nur die
gezeigte proof-gebundene Kante nach `succeeded`. Es gibt keine Rückkante zu
`prepared/submitted`.

Die Taskkante `prepared -> canceled` ist ausschliesslich Teil dieses
effektfreien Non-Epoch-Abschluss-CAS und verlangt `submission_count = 0` sowie
einen `cancellation_proof_hash`, der Attempt, Intent und verlorenen Guard
bindet. Ein bereits `submitted`er Task darf nur nach terminaler Klassifikation
sämtlicher Child-Submissions und vollständigem Non-Submission-/Non-Acceptance-
beziehungsweise Terminal-No-Effect-Beleg nach `canceled`; sein Hash bindet die
ganze Child-Liste. Keine dieser Kanten ist für `epoch_publish` verfügbar.

`barriered` ist nur zulässig, wenn `barrier_task` auf einen Task mit
`operation = barrier`, demselben `engine_attempt`, höherem `ordinal`,
identischem Member und identischer Engine-Inkarnation zeigt, dieser Task samt physischer Submission
`succeeded` ist und sein `confirmed_end_marker_hash` exakt dem
`expected_end_marker_hash` entspricht. `barrier_proof_hash` bindet diese Werte
und alle von der Barriere abgedeckten Submission-IDs. `response_digest` ist nur
Diagnose und kann keinen dieser Belege ersetzen. Ohne den vollständigen Beleg
bleibt die Submission `unknown` und der Intent darf nicht `succeeded` werden.

Verweist `request_payload` auf Artefakte, müssen IDs, Reihenfolge und Versionen
die kanonische Requestpayload vollständig rekonstruieren; der Attempt pinnt
diese Artefakte bis zum terminalen Abschluss. Hashabweichung ist eine
Invarianzverletzung, kein Anlass zum Neubauen aus aktuellen Fachrecords.

#### `search_generation_swap`

Auditdatensatz für den atomaren DB-Pointerwechsel.

| Feld | Typ | Regel |
| --- | --- | --- |
| `swap_key` | text | Unique, namespaced Idempotency-Key |
| `from_generation` / `to_generation` | relation?/relation | alt/neu; `from` nur bei Bootstrap leer |
| `expected_pointer_version` | SafeRevisionV1 | CAS-Guard |
| `cutoff` | SafeRevisionV1 | von neuer Generation bestätigt |
| `reason` | select | `bootstrap`, `deploy`, `schema`, `rebuild`, `rollback`, `recovery` |
| `handoff_logical_uids` | json? | nur beim ersten Bootstrap aus einem externen Legacyowner; kanonische, nichtleere Liste der vom V1-Entity-/Querymodell vollständig abgedeckten Rollen |
| `handoff_authority_snapshot` | json? | bei Handoff verpflichtetes geschlossenes `State1LegacyAuthoritySnapshotV1`; sein kanonischer Digest ist Pointer-CAS-Guard |
| `state` | select | `prepared`, `committed`, `aborted` |
| `committed_at` | date? | DB-Commitzeit |

Die gespeicherte JSON-Innenform ist keine freie Map. `SafeRevisionV1` ist der
oben definierte Integerbereich `0..M`; `DigestV1` ist `sha256:` gefolgt von
exakt 64 kleingeschriebenen Hexzeichen des SHA-256 über die
RFC-8785-kanonisierten Bytes des bezeichneten Objekts. Alle Properties sind
verpflichtend, jedes Objekt auf jeder Tiefe hat
`additionalProperties: false`, und die Arrayreihenfolge ist Teil der
kanonischen Form:

```text
State1LegacyAuthoritySnapshotV1 = {
  "schema": "state1_legacy_authority_snapshot_v1",
  "uids": [{
    "logical_uid": non-empty text,
    "expected_authority": "legacy",
    "authority_version": SafeRevisionV1,
    "source_revision": SafeRevisionV1,
    "source_snapshot_digest": DigestV1,
    "legacy_resource_set": [{
      "engine_scope": non-empty text,
      "physical_index_id": non-empty text
    }],
    "legacy_resource_set_digest": DigestV1,
    "operation_state": "ready",
    "authority_head_key": null,
    "authority_pointer_version": null
  }]
}
```

`uids` ist eindeutig und nach UTF-8-Bytes von `logical_uid` aufsteigend
sortiert und stimmt bijektiv mit `handoff_logical_uids` überein. Vor dem
Pointer-CAS werden alle Felder des zugehörigen `LegacyAuthorityBoundaryV1`-
Records und das vollständige Legacyressourcenset unter denselben Locks erneut
gelesen und byte- beziehungsweise digestgleich verlangt; die durch den
unveränderten `source_snapshot_digest` bezeichnete Preimage bleibt vollständig
auflösbar und digestgültig. Die eingebetteten Ressourcensets haben
dieselbe kanonische Ordnung und Digest-Preimage wie in Abschnitt 1.1. Die neue
Generation muss sämtliche Rollen vollständig abdecken und ihre physischen IDs
müssen von der Vereinigung aller gebundenen Legacyressourcenmengen disjunkt
sein. Ohne Handoff sind beide `handoff_*`-Felder leer.

Der Snapshot bindet keine internen Epochen, Taskheads, Auditdateien oder
korrekturspezifischen Aktivierungsartefakte des Legacyowners. STATE1 beweist
die Zielbytes ausschliesslich über seine Generation-, Member-, Attempt-,
Task-, Cutoff- und Publisherrecords.

#### `search_context_resource`

Generalisierte Retentionszusage. Unique-Key: `(resource_kind, resource_key)`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `resource_kind` | select | `generation`, `content_snapshot`, `cursor_key`, `cursor_format`, `contract_artifact`, `security_history` |
| `resource_key` | text | opake stabile ID |
| `state` | select | `staged`, `issuing`, `retained`, `gc_eligible`, `deleting`, `deleted`, `lost` |
| `issued_valid_until_hwm` | date? | dauerhaft monoton |
| `retention_grace_s` | SafeRevisionV1 | versionierte Betriebsgrösse |
| `retain_until` | date? | mindestens HWM plus Grace und weitere Gates |
| `version` | SafeRevisionV1 | CAS-Version |

`staged` bezeichnet eine noch nie öffentlich bindbare Kandidatenressource und
erlaubt keine Cursorzusage. `staged -> issuing` ist ausschliesslich im
typisierten Aktivierungs-CAS ihres Owners erlaubt: für Generation und
Content-Snapshot im Epoch-/Head-CAS, für Cursor-Key im Key-Rollout-CAS sowie für
Cursorformat und Vertragsartefakt im jeweiligen Expand-CAS. Ein aufgegebener
Kandidat darf bei leerem HWM und ohne Pins direkt `staged -> gc_eligible`
wechseln. `issuing` erlaubt
neue Kontextbindungen. `retained` verbietet neue Bindungen, bedient aber bereits
zugesagte Kontexte bis zum High-Watermark weiter. Erst danach und nach allen
zusätzlichen Pins ist `retained -> gc_eligible` erlaubt; es folgen
`deleting -> deleted`. `staged/issuing/retained -> lost` erfasst einen
irreversiblen Verlust und ist terminal. Eine einmal `issuing` oder `retained`
gewesene Ressource darf nie wieder nach `staged` oder `issuing` geöffnet werden.
Key- und Formatressourcen referenzieren
die folgende Keyring-Metadaten-Collection; das Secret bleibt ausserhalb der
Datenbank. Ein als `lost` markiertes, noch zugesagtes Artefakt darf nicht als
abgelaufen ausgegeben werden.

#### `search_cursor_key`

Keyring-Metadaten; das Root-Keymaterial selbst liegt ausschliesslich in der
Secret-Ablage. Unique-Key: `kid`.

| Feld | Typ | Regel |
| --- | --- | --- |
| `kid` | text | opake Unique-ID, niemals wiederverwenden |
| `format_version` | text | unterstütztes Cursorformat |
| `state` | select | `staged`, `issuing`, `verify_only`, `retired`, `lost` |
| `secret_ref` | text | Referenz, niemals Root-Keymaterial |
| `retention_resource` | relation | gemeinsam retenierter Control-Plane-Record |
| `issuance_started_at` / `issuance_stopped_at` | date? | Expand-/Contract-Diagnose |
| `version` | SafeRevisionV1 | CAS-Version |

`staged -> issuing` ist erst erlaubt, wenn alle möglichen Reader Key und Format
kennen. `issuing -> verify_only` beendet neue Ausstellung; `verify_only ->
retired` wartet den Resource-HWM plus Key-Grace ab. `lost` ist terminal und
unterscheidet einen vertragswidrig verlorenen bekannten Key von einem beliebigen
unbekannten `kid`. Ein vorzeitiger Secretverlust setzt Key und
`retention_resource` im selben Control-Plane-CAS auf `lost`; ein späterer Hook
darf diese Klassifikation nicht nachziehen. Diese Verlustkante ist exakt aus
`staged`, `issuing` oder `verify_only` erlaubt; `retired` und `lost` sind
terminal.

Für jede Ownerrelation gilt eine unveränderliche 1:1-Bindung:

| Owner | verpflichtete Ressource |
| --- | --- |
| `search_index_generation(generation_key)` | exakt `resource_kind = generation`, `resource_key = generation_key` |
| `search_content_snapshot(snapshot_key)` | exakt `resource_kind = content_snapshot`, `resource_key = snapshot_key` |
| `search_cursor_key(kid)` | exakt `resource_kind = cursor_key`, `resource_key = kid` |
| `search_contract_artifact(artifact_key)` | exakt `resource_kind = contract_artifact`, `resource_key = artifact_key` |

Auf jedem der vier Ownerfelder `retention_resource` liegt zusätzlich ein
Unique-Index. Erzeugungs-, Aktivierungs-, HWM- und GC-CAS prüfen Relation,
Resource-Kind und -Key gemeinsam; weder Owner noch Ressource dürfen später auf
ein anderes Gegenstück umgebogen oder von einem zweiten Owner geteilt werden.
Ein Typ-/Key-Mismatch ist `invariant_violation` und blockiert Issuance sowie GC.

### 4.4 Verbindliche Sekundärindizes

Zusätzlich zu Primär- und oben genannten Unique-Indizes müssen mindestens die
folgenden Zugriffspfade als PocketBase-Collection-Indizes existieren. Die
konkreten Indexnamen sind nicht normativ; Feldreihenfolge und Präfixe sind es.

| Collection | Indexfelder in Reihenfolge | Zweck |
| --- | --- | --- |
| `search_control_plane_checkpoint` | `(state, checkpoint_sequence)` | externe Bestätigung und Crash-Recovery |
| `search_catalog_change` | `(component, resolution_state, revision, ordinal)` | ältesten offenen Komponentenprefix finden |
| `search_catalog_change` | `(target_kind, target_key, component, revision)` | Subsumption und Zielreplay |
| `search_catalog_delivery` | `(generation_member, state, change_revision, change_ordinal)` | geschlossenen Memberprefix beweisen |
| `search_projection_dirty` | `(state, desired_revision, target_key, id)` | `queued`-Partition keyset-basiert scannen |
| `search_projection_dirty` | `(state, retry_not_before, desired_revision, target_key, id)` | fällige Retrypartition scannen |
| `search_projection_dirty` | `(state, lease_until, desired_revision, target_key, id)` | abgelaufene Leases übernehmen |
| `search_projection_sweep_state` | `(scope_key, partition)` | genau ein persistenter Cursor je Sweep-Partition |
| `search_projection_version` | `(document_key, built_through_revision)` | versioniertes Lieferartefakt auflösen |
| `search_projection_version` | `(retain_until)` | sichere Artefakt-GC |
| `search_contract_artifact` | UNIQUE `(query_family, contract_version, engine_profile_key)` | qualifiziertes Ausführungsartefakt eindeutig auflösen |
| `search_contract_artifact` | UNIQUE `(query_family, payload_hash)` | inhaltsadressierte Contractidentität erzwingen |
| `search_contract_artifact` | UNIQUE `(retention_resource)` | unveränderlichen Resource-Owner erzwingen |
| `search_geo_shard_allocator_state` | UNIQUE `(scope_key)` | serialisierter Allocator und Backfill-Cursor |
| `search_geo_route_shard` | UNIQUE `(shard_ordinal)` | lückenlose append-orientierte Shardordnung |
| `search_geo_route_shard` | `(state, shard_ordinal)` | aktuellen offenen Shard und Manifestbrüche prüfen |
| `search_geo_route_assignment` | UNIQUE `(trail_key)` | unveränderliche Trailzuweisung |
| `search_geo_route_assignment` | UNIQUE `(geo_shard_id, slot_ordinal)` | höchstens 1.000 nie wiederverwendete Slots je Shard |
| `search_geo_route_assignment` | `(local_created_at, trail_key)` | initialen und diagnostischen Allocation-Keyset-Scan ausführen |
| `search_geo_shard_manifest_version` | UNIQUE `(scope_key, catalog_revision)` | Cutoff-genaue Shardfamilie auflösen |
| `search_geo_shard_manifest_version` | UNIQUE `(scope_key, manifest_revision)` | Manifestidentität und Retention auflösen |
| `federation_ingest_event` | `(state, retry_not_before, received_at, id)` | fällige Inboxarbeit |
| `federation_outbox_event` | `(state, retry_not_before, created, id)` | fällige durable Outboxarbeit |
| `federation_outbox_delivery` | UNIQUE `(outbox_event, recipient_key)` | Empfängerzustellung idempotent wiederaufnehmen |
| `federated_visibility_provenance` | UNIQUE `(federated_object, evidence_digest)` | gemeinsamen Ingressbeleg erzwingen |
| `federated_publication` | UNIQUE `publication_iri`; UNIQUE `(trail_key, generation)` | P1/P2-Identität und absorbierenden Tombstone erzwingen |
| `federation_legacy_migration` | `(state, created, id)` | offenen beziehungsweise terminalen Bestandslauf diagnostizieren |
| `federation_ingest_event` | `(state, lease_until, id)` | abgelaufene Claims |
| `federation_ingest_event` | `(claimed_object_iri, received_at, id)` | IRI-Audit und Reihenfolge |
| `federated_object` | `(state, discoverable_until, id)` | Expiry und globaler Scope-Horizont |
| `federated_object` | `(origin_id, state, discoverable_until, id)` | Origin-Scope und dessen Horizont |
| `federated_object` | `(state, refresh_due_at, id)` | Refreshplanung |
| `federated_trail_snapshot` | `(federated_object, verified_at)` | Snapshothistorie |
| `federated_trail_snapshot` | `(retain_until, id)` | sichere Snapshot-GC |
| `federation_scope_version` | `(scope_kind, scope_key, catalog_revision)` | Cutoff-genauen Scope-Stand auflösen |
| `search_subject_visibility_frontier` | `(state, frontier_revision, subject_key, id)` | Restriction-/Clear-Frontiers prüfen und reparieren |
| `search_visibility_fence` | `(state, set_revision)` | aktive Propagation und Securityprefix |
| `search_visibility_fence` | `(subject_kind, subject_key, state)` | queryrelevante Fences |
| `search_visibility_fence` | `(history_retain_until, id)` | History-GC |
| `search_visibility_fence_successor` | UNIQUE `(fence, predecessor_key)` | genau ein Root und kein Fork je Successor-Slot |
| `search_visibility_fence_successor` | `(fence, successor_revision, successor_key)` | azyklische Clear-/Grant-/Restriction-Nachfolgerkette |
| `search_visibility_fence_successor` | `(successor_fence, successor_revision)` | neutralisierte Clears vom neuen Fence aus auditieren |
| `search_fence_retirement_artifact_pin` | UNIQUE `(fence, artifact_kind, artifact_key)` | Manifest-/Pin-Bijektion und fenceweiser History-GC |
| `search_fence_retirement_artifact_pin` | `(artifact_kind, artifact_key, fence)` | linearisierten Reverse-Pin vor Artefakt-GC prüfen |
| `search_visibility_fence_delivery` | `(state, generation_member)` | offene Fencezustellungen |
| `search_index_generation` | `(lifecycle_state, epoch_state)` | Serving-/Recoverykandidaten |
| `search_index_generation` | `(lifecycle_state, capture_from_revision)` | Change-Log-Pins |
| `search_index_generation_query_family` | UNIQUE `(generation, query_family)` | Family- und belegungsunabhängige Contractbindung auflösen |
| `search_index_generation_query_family` | `(contract_artifact, generation)` | Contractpins und Generation-GC prüfen |
| `search_index_generation_member` | `(generation, state)` | Bundlegates |
| `search_index_generation_member` | UNIQUE `(generation, query_family, component, partition_key)` | logische Singleton-/Shardpartition generationengenau binden |
| `search_index_generation_member` | UNIQUE `(generation, query_family, geo_shard_id)` für gesetzte Shard-IDs | physischen Route-Shard einer Generation eindeutig auflösen |
| `search_index_generation_member` | `(contract_artifact, generation, query_family)` | Contractbindung, Retention und Rebuild prüfen |
| `search_engine_attempt` | `(generation, state, started_at)` | Build-/Publisher-Recovery |
| `search_engine_attempt` | `(visibility_fence, state, started_at)` | Fence-Apply und dessen Retirement-/Recoverybezug |
| `search_engine_attempt` | `(target_route_contract_artifact, state, started_at)` | eingefrorene Route-Contractbindung recovern |
| `search_engine_attempt` | `(target_route_query_family, state, started_at)` | Route-Family-Attempts einschliesslich initialem Shard recovern |
| `search_engine_task` | `(state, submitted_at)` | unbekannte/laufende Tasks prüfen |
| `search_engine_submission` | `(state, submitted_at, id)` | physische Submits und Unknown-Recovery |
| `search_content_snapshot` | `(generation, content_epoch)` | gepinnten Epochstand auflösen |
| `search_content_snapshot` | `(geo_shard_manifest_version, generation, content_epoch)` | Route-Manifestpins eines Snapshots prüfen |
| `search_content_snapshot_query_family` | UNIQUE `(content_snapshot, query_family)` | Family-Authority unabhängig von Memberdokumentbelegung auflösen |
| `search_content_snapshot_query_family` | UNIQUE `(content_snapshot, generation_query_family)` | Generation-/Snapshot-Family-Bijektion prüfen |
| `search_content_snapshot_query_family` | `(contract_artifact, content_snapshot)` | belegungsleere und belegte Route-Contractpins sowie GC prüfen |
| `search_content_snapshot_member` | UNIQUE `(content_snapshot, query_family, component, partition_key)` | immutable Queryfamilie einer Epoch auflösen |
| `search_content_snapshot_member` | `(generation_member, content_snapshot)` | Memberpins und GC prüfen |
| `search_content_snapshot_member` | `(contract_artifact, content_snapshot, query_family)` | Route-Contractpins und Cursorretention prüfen |
| `search_content_snapshot_scope` | `(scope_kind, scope_key, snapshot_set_revision)` | epochgebundenes Federation-Binding |
| `search_generation_swap` | `(state, created, id)` | vorbereitete Swaps recovern |
| `search_context_resource` | `(state, retain_until, id)` | Eligibility und Recovery der GC |

Jeder Claim- und GC-Scan ist zusätzlich hart gebatcht und setzt einen stabilen
Keyset-Tie-Breaker (`id`) ein. Ein fehlender Index darf nicht durch einen
unbegrenzten periodischen Full-Scan im Requestpfad kompensiert werden.

## 5. Revisionstransaktion

### 5.1 Normale Fachmutation

Auch während eines Authoritywechsels existiert genau **eine**
Fachtransaktion. Der Command berechnet mit der versionierten
Projektions-Abhängigkeitsregistry aus demselben konsistenten Vorher-/
Nachherschnitt die vollständige logische Zielhülle `A`. Er liest die Authority
aller Ziele und bildet `T` als die kanonische Teilmenge, deren aktueller Owner
`state1` ist. Zusätzlich bezeichnet `C` die geschlossene Menge der unabhängig
von einer Suchrollen-Authority verpflichteten STATE1-Federation-, Visibility-,
Security-, Fence- und Tombstone-Changes.

Registryrevision, `A`, `T`, `C` und alle erwarteten Authority-CAS-Versionen
werden im `cause_digest` gebunden. Es gilt
`needs_catalog_revision = (T != []) || (C != [])`. Die Fachmutation wird nie
doppelt ausgeführt. Ziele ausserhalb `T` werden ausschliesslich durch ihren
aktuellen externen Owner verarbeitet; dessen Transitionservice muss seinen
Anteil in derselben PocketBase-Transaktion persistieren. STATE1 liest oder
schreibt dabei keine ownerfremden Epochen, Attestierungen oder Taskguards.

Jeder STATE1-relevante Command führt in genau einer PocketBase-Transaktion
aus:

1. `cause_key`, vollständigen `cause_digest`, Registryrevision, erwartete
   Authoritypartition und deren CAS-Versionen vor jeder Wirkung prüfen; eine
   geänderte Partition rollt den gesamten Command zurück und wird beim Retry
   neu berechnet;
2. die einmalige DB-Zeit lesen, die genau einmalige Fachmutation und bei einer
   STATE1-Erstaufnahme deren lokalen Allocation-Key vorbereiten;
3. falls `needs_catalog_revision`,
   `search_catalog_state.committed_revision + 1` per CAS als Revision `R`
   reservieren; nur bei `T = []` und `C = []` existiert für STATE1 kein `R`;
4. die Fachmutation genau einmal anwenden und nur für ein auf `T` gemapptes,
   erstmals lokal materialisiertes Trailziel im Allocatorzustand `active`
   unter demselben CAS dessen unveränderliches
   `search_geo_route_assignment` samt Shardheader und neuer Manifestversion
   schreiben; während des nicht ausstellbaren initialen `backfilling`
   übernimmt stattdessen der durable Keyset-Sweep diese Zuweisung;
5. bei `needs_catalog_revision` genau einen `search_catalog_commit(R)` und
   seine lückenlosen `search_catalog_change(R, ordinal)` für die Vereinigung
   aus den auf `T` gemappten Projektionschanges und `C` anlegen;
6. je auf `T` gemapptem Projektionsziel und je durch `C` verpflichteten
   Dirty-Ziel `search_projection_dirty` mit
   `desired_revision = max(alt, R)` upserten; bei Visibilityänderungen umfasst
   dies jedes betroffene `federation_scope`-Ziel;
7. für jedes relevante registrierte Generationenmember mit
   `R > baseline_revision` einen `pending`-Beleg in
   `search_catalog_delivery` anlegen;
8. die zu diesen Änderungen gehörenden Delete-Tombstones und bei
   Visibilityänderungen `search_subject_visibility_frontier`, Restriction-
   Fence beziehungsweise Successor-Link persistieren; und
9. `change_count`, `change_manifest_digest`, Katalogzustand, Fachrecord und
   sämtliche STATE1-Control-Records gemeinsam committen.

Authority-unabhängige `C`- und zusätzlich verpflichtete Outbox-/Anti-
Rollback-Effekte dürfen nicht mit `T = []` entfallen. Kein Enginezugriff findet
in der Transaktion statt. Nach Commit nehmen STATE1-Worker und Publisher
ausschliesslich den durablen `T`-/`C`-Anteil auf. Ein Authority-CAS-Verlust
rollt die gesamte Fachtransaktion zurück; der Retry bestimmt die aktuelle
Partition neu. Damit bedient STATE1 kein Ziel, das noch einem externen Owner
gehört, und kein Handoff kann denselben Fachcommit auf beiden Seiten
duplizieren oder verlieren.
Bei der Erstaufnahme stammt `local_created_at` ausschliesslich aus der in
Schritt 2 gelesenen lokalen DB-Zeit. Ein Import- oder Federation-Payload darf
dieses Feld weder vorbelegen noch durch `published`, `created`,
`origin_created` oder eine andere Remotezeit überschreiben. Mehrere in
demselben Command erstmals materialisierte Trails werden vor dem Allocator-CAS
nach `(local_created_at, trail_key)` sortiert. Der einmalige Legacy-Backfill
verwendet denselben Key über die vor seinem konsistenten Cutoff lokal
festgeschriebenen Erstellungswerte. Sobald der Allocator `active` ist, darf
kein katalogfähiger Trail ohne Assignment committen.

Der Allocator liest unter dem gemeinsamen SQLite-Schreiblock Singleton,
aktuellen Shard, Slotzahl und letzten Manifesthash. Ein noch nicht voller
offener Shard erhält den nächsten lückenlosen Slot. Ist er voll, werden dessen
`sealed`-Kante, der neue Shard mit nächster Ordinalzahl, Slot `0`, Assignment,
die auf den neuen Shard wechselnde `current_open_shard`-Relation,
Allocatorstand und immutable Manifestversion im selben Revisionscommit
geschrieben. Ein verlorener CAS beginnt mit dem nun aktuellen Manifest neu; er
darf den bereits gewählten Trail nie mit einer geratenen Shard-ID committen.

Im Zustand `backfilling` ist die Route-Familie noch nicht ausstellbar. Der
persistente Keyset-Cursor scannt alle bis `backfill_cutoff` vorhandenen und
danach committed unzugewiesenen Trails nach `(local_created_at, trail_key)` und
wendet denselben Slotallocator in hart begrenzten Revisionsbatches an. Der
finale `backfilling -> active`-CAS läuft unter dem gemeinsamen Schreiblock,
beweist einen vollständigen Sweep bis zum aktuellen Katalogstand, null
verbleibende katalogfähige Trails ohne Assignment sowie einen
rekonstruierbaren Manifesthash. Eine zwischen zwei Batches neu aufgenommene
Zeile wird dadurch vor Aktivierung entweder später im Keyset gesehen oder lässt
den finalen CAS verlieren.

Bei `route_radius_ux_v1` ist ein Generationenmember für einen Change genau
dann relevant, wenn sein `geo_shard_id` dem persistenten Assignment des
betroffenen Trails entspricht oder eine generationenkompatible globale
Settingsänderung seine gesamte Queryfamilie betrifft. Ein Contractwechsel
erzeugt stattdessen eine neue Generation. `component = geometry` allein ist kein
Fan-out auf alle Shards. Ein Security-Change verwendet dagegen den
vollständigen betroffenen Trail-/Subject-Closure und erzeugt alle daraus
folgenden membergenauen Deliveries; eine nur teilweise Shardmenge darf keinen
Securityprefix schliessen.

Ein Command darf mehrere Fachrecords ändern. Entweder ist jeder committed
Zwischenstand ein gültiger katalogfähiger Zustand, oder der Import verwendet
eigene Stagingrecords und genau einen fachlichen Aktivierungscommit. Getrennte
PocketBase-Saves, deren Zwischenstände fachlich ungültig sind, dürfen nicht
durch nachträgliche Dirty-Koaleszierung kaschiert werden.

Alle katalogrelevanten CRUD-, Federation-, Plugin-, Import-, Merge-, Admin- und
Reconcilepfade verwenden denselben transaktionalen Mutation-Service mit dem im
Callback erhaltenen PocketBase-`txApp`. `UnsafeWithoutHooks`, direkte Saves und
After-Success-Hooks dürfen diese Outboxgrenze nicht umgehen. Ein
After-Success-Hook darf höchstens einen Worker aufwecken; Reconciliation findet
normale verpasste Arbeit, ersetzt aber insbesondere bei Privacy-Mutationen nie
den atomaren Nachweis.

### 5.2 Sichtbarkeitsverengung

Bei Delete, Public→Private, Share-Entzug, Actor-/Authority-Sperre, einer
vorzeitigen TTL-Verkürzung und einer nicht beweisbar harmlosen Inhaltsredaktion
gilt zusätzlich:

- Die autoritative Restriction und ein `active` Fence werden spätestens im
  selben DB-Commit sichtbar. Kann eine Fachsaga dies nicht, setzt ein erster
  Commit Fence und dauerhafte Restriction-Absicht, bevor ein späterer Commit den
  Fachzustand abschliesst.
- Derselbe Commit schreibt für jedes betroffene Subject den vollständigen
  `search_subject_visibility_frontier` fort. Restriction, Fence,
  Source-Manifest, Gateway-Guard-Set-Digest und Frontier-Version sind ein
  atomarer Autorisierungsstand; eine spätere Workerrekonstruktion ist
  unzulässig. Grant/Clear und Restriction-Successor verwenden dieselbe Grenze.
- Derselbe Commit klassifiziert den sofortigen Gateway-Guard unveränderlich als
  `exact_target` oder `conservative_subject_exclusion` und bindet unabhängig
  davon `exact_target_digest` samt `visibility_rule_version`. Im zweiten Fall setzt
  er `guard_release_revision` auf die früheste normale Ziel-/Publishrevision,
  ab der eine Content-Epoch den exakten eingeschränkten Zielstand oder einen
  typisiert nachweislich engeren koaleszierten Restriction-Successor tragen
  darf. Der Anker behauptet noch keine Publikation; eine spätere Lockerung des
  temporären Whole-Subject-Ausschlusses ist kein epochloses Security-Apply.
- Dieser Commit erhöht `security_revision` auf `R` und erzeugt Tombstone/
  Change-Log. Im selben Commit erhöht er `checkpoint_sequence` und legt den
  kumulativen `search_control_plane_checkpoint` an. Der Gateway schliesst das
  Subject ab diesem Commit aus und bleibt bis zur externen Bestätigung
  fail-closed.
- Enginekonvergenz ist eine nachgelagerte, bestätigte Nebenwirkung. Ein Fehler
  lässt den Fence aktiv und darf höchstens unterbelichten.

Eine reine Sichtbarkeitserweiterung darf nach dem Fachcommit asynchron folgen.
Ihre Revision legt für jeden abgelösten Fence den typisierten append-only
Successor-Link an. Sie entfernt keinen noch aktiven Fence, bis der neue
erlaubte Stand vollständig projiziert und sein Clear-Guard erfüllt ist; einen
bereits retirten Fence mutiert sie nie. Wird die Erweiterung vor Konvergenz
durch eine spätere mindestens ebenso enge Restriction neutralisiert, erzeugt
diese stattdessen den typisierten Restriction-Successor und hält ihren eigenen
Fence ab demselben Commit querywirksam.

Eine Federation-Update-Payload, die zuvor indizierte Inhalte entfernt oder
deren weitere öffentliche Gültigkeit nicht beweisbar erhält, gilt ebenfalls als
Verengung. Kann der versionierte Difftyp sie nicht nachweislich als rein
nichtverengend klassifizieren, fenced Wanderer das ganze Objekt bis zur
bestätigten neuen Projektion.

### 5.3 Lokale Sichtbarkeits- und Publikationstransitionen

Die vollständige fachliche Matrix aus
[Federation-Sicherheits- und Publikationsvertrag v1, Abschnitt 7](/develop/specs/trail-search/contracts/federation-security/#7-verbindliche-transitionmatrix)
ist normativ. Jede Implementierung konkretisiert pro Zeile Authority,
Fachcommit, betroffene SEC-SCOPED-Subjects, Dirty-Ziele, Fence, Outboxintent,
API-Erfolgspunkt, Idempotenz und Recovery. Mindestens gilt:

| Übergang | Atomare lokale Wirkung | Projektion und Federation |
| --- | --- | --- |
| Trail create private | nur Owner-/ACL-Sichtbarkeit; keine Publication | lokale und betroffene ACL-Scopes dirty; keine Public-Outbox |
| Trail create public | lokaler Public-Stand erst nach normalem Publish sichtbar | neue immutable Publication Pn stagen und `Create(Pn)` durable einplanen |
| private → public | keine vorzeitige Erweiterung | neue Epoch, neue Publication-IRI und Create-Outbox; niemals Tombstone reaktivieren |
| public → private | Fachzustand, Publication-Tombstone und Fence im selben Commit | `Delete(Pn)` bei möglicher voriger Netzwerkwirkung durable; Trail bleibt bestehen und Remote-Zustellung ist nachgelagert |
| Trail delete | sofortiger Trail-/Publication-Fence und lokaler Tombstone | aktive Publication löschen; Outboxzustellung bleibt nachgelagert |
| Share add/update | genau bezeichneten Actor-/ACL-Grant atomar setzen | Trail-/Actor-/ACL-Ziele per `max(desired_revision,R)` dirty; Direct-Distribution optional durable, öffentliche Publication unverändert |
| Share revoke | genau diesen Grant entfernen; übrige Shares neu aus autoritativem DB-Stand bilden | Share-/Actor-/ACL-Fence, optional `Undo(Announce)`; nie pauschal alle Shares löschen |
| Account-/Actor-Delete oder Block | alle abgeleiteten Actor-Scopes synchron restricten | Actor- und betroffene Publication-/ACL-Ziele koaleszieren |
| Origin block | nur betroffene Origin und deren abhängige Scopes restricten | Origin-Ziel koaleszieren; `SEC-GLOBAL` nur bei unklarer Scopebestimmung |
| Block-/Share-Clear | keine epochlose Erweiterung | normale neue Revision, Successor-Link und bestätigte Content-Epoch vor Fence-Retirement |

Lokale Privacywirkung wartet nie auf Remote-Zustellung. Der synchrone API-
Erfolgspunkt liegt nach durablem Fach-, Fence-, Dirty- und Outboxcommit sowie
der erforderlichen externen Securitycheckpoint-Bestätigung, nicht nach dem
Netzwerkversand.

## 6. Federation-Zustandsautomaten

### 6.1 Ingest-Automat

| Von | Ereignis und Guard | Nach | Nebenwirkung |
| --- | --- | --- | --- |
| — | signierte Eingabe dauerhaft gespeichert | `received` | noch keine Fachwirkung; danach darf `202` folgen |
| `received/retry_wait` oder `verifying` mit abgelaufenem Lease | gültiger CAS-Claim | `verifying` | Lease mit höherem Token und neuer Deadline |
| `verifying` | nach Authorityprüfung existiert derselbe `verified_effect_key`, aber dessen `verified_effect_digest` weicht ab | `quarantined` | Key-/Digestkonflikt vor Unique-CAS; nur Inboxrecord, keine Objektwirkung |
| `verifying` | derselbe `verified_effect_key` und exakt derselbe `verified_effect_digest` existieren bereits | `duplicate` | vorige sichere Antwort referenzieren; keine Revision |
| `verifying` | Objekt ist nicht `deleted/quarantined`; Authority, Replay, Payload und öffentliche **oder direct/scoped** Adressierung sind valide; Effect-Key-CAS und Objektkopf-CAS gewinnen; FED0 beweist `after` beziehungsweise Genesis-Currentness | `accepted` | Revisionstransaktion und gegebenenfalls Snapshot/Objektkopf; Public-Beleg aktualisiert Public-/Origin-/Actor-/Publication-Scopes, Direct-Beleg ausschliesslich seine Actor-/ACL-Scopes |
| `verifying` | autoritative konditionale Revalidierung beweist `equal`, bindet Headtoken und unveränderten Payloadhash, wurde nach dessen Annahme angefordert und der effektive Objektzustand ist `discoverable` oder nur TTL-bedingt `expired`; im zweiten Fall begann der Revalidierungsrequest erst nach Expiry | `accepted` | reine Refresh-Revision; TTL darf erneuert werden, Kausaltoken und Payload bleiben gleich |
| `verifying` | Objekt ist `quarantined` und ein ausdrücklich angeforderter autoritativer Current-State-Resync löst Authority und Ordnung gegen den unveränderten Kopf vollständig auf | `accepted` | `quarantined -> restricted` mit aktivem Fence oder bei bewiesenem Delete `-> deleted`; nie direkt discoverable |
| `verifying` | nach Authorityprüfung ist der Objektkopf bereits `deleted` und der Input kein exakter Duplicate | `stale` | absorbierender Tombstone; keine Resurrection, Revision oder TTL |
| `verifying` | gültiger Effekt ist gegen den unveränderten Objektkopf beweisbar `before` oder nur `equal`, ohne den oben definierten Revalidierungsbeleg | `stale` | keine Revision, keine TTL- oder Sichtbarkeitserweiterung |
| `verifying` | transienter Fetch-/Keyfehler vor Annahme | `retry_wait` | begrenzter Backoff mit Jitter |
| `verifying` | permanent ungültig oder unautorisiert | `rejected` | sichere 4xx-Klasse, keine Discoverability |
| jeder nichtterminaler Zustand | Digestkollision, IRI-Kollision oder unklärbare Authority | `quarantined` | nur Inboxrecord quarantänisieren; keinerlei Objektkopf-, TTL-, Restriction- oder Fenceänderung |
| `verifying` | Authority ist bereits bestätigt, Objektordnung aber `concurrent/unknown` und fortbestehende öffentliche Gültigkeit nicht beweisbar | `quarantined` | Objekt per eigener Securityrevision restricten/fencen; Admin-/Resync-Diagnose |

Ein abgelaufener Lease erlaubt einen neuen Claim mit höherem `lease_token`.
Ergebnisse eines supersedierten Workers scheitern am Abschluss-CAS. `accepted`,
`stale`, `rejected`, `duplicate` und `quarantined` sind für denselben
`ingest_key` terminal. Ein erneuter Eingang derselben bytegenauen signierten Eingabe gibt den
bereits persistierten Zustand sicher wieder und verändert den Originalrecord
nicht. Eine andere Signatur oder Payload mit derselben behaupteten Activity-ID
erhält einen anderen Inboxrecord und kann den legitimen Effect-Key erst nach
eigener Authorityprüfung belegen. Beim konkurrierenden partiellen Unique-CAS
gewinnt genau ein autorisierter Effekt; der andere wird nur bei gleichem
`verified_effect_digest` `duplicate`, sonst `quarantined`. Ein
unautorisierter Fremdclaim reserviert keine fachliche Identität. Ein
angenommener Gewinner verwendet
`federation-effect:<verified_effect_key>` als Revisions-`cause_key`; ein
`stale`-Gewinner belegt dagegen keine Revision. Eine konservative
Ordnungsquarantäne verwendet den getrennten idempotenten
`federation-order-quarantine:<verified_effect_key>`-Cause. Der Duplicate-Record
setzt nur `duplicate_of` und belegt keine zweite Revision.

Der Key-/Digestvergleich hat Vorrang vor dem partiellen Unique-CAS. Gleicher
Key bei anderem kanonischem Effektdigest ist niemals ein idempotenter Replay,
auch wenn Activity-ID oder rohe `payload_hash` gleich aussehen; der Konflikt
wird quarantänisiert und verändert das Objekt nicht.

Die fachliche Annahme serialisiert zusätzlich pro Objekt. Der Abschluss-CAS
prüft `federated_object.id`, `version`, Authority-Binding, letzten
`object_order_token` und den dagegen berechneten Vergleich. Nur ein bewiesener
strikter Nachfolger darf Payload ändern oder einen fachlich `restricted`en
Stand wieder öffentlich machen und den Kopf mit neuem Token fortschreiben. Die
einzige TTL-Verlängerung beziehungsweise Wiederaufnahme aus reinem
`expired` ohne neuen Token ist die explizite `equal`-Revalidierung aus der
Transitionstabelle. Für ein bislang unbekanntes Objekt muss FED0
entsprechend beweisen, dass die gelesene Darstellung am autoritativen Endpunkt
aktuell ist. Verliert der CAS gegen einen neueren Kopf, wird der Vergleich gegen
diesen Kopf wiederholt; das alte Ergebnis darf nicht trotzdem committen.

Eine autoritative Current-State-Revalidierung derselben Kausalversion darf eine
TTL nur erneuern, wenn ihr konditionaler Beleg den bereits akzeptierten
Headtoken und unveränderten Payloadhash umfasst und nach dessen Annahme
angefordert wurde. Sie aktualisiert Objektversion, `currentness_proof_hash`,
lokale Verifikationszeit, TTL und Revision, aber weder `last_effect_key` noch
Kausaltoken oder Payload. Ein blosses `equal`, eine Remotezeit oder ein späteres
`received_at` genügt nicht. Ein nicht streng ordenbarer, aber möglicherweise
verengender Effekt darf konservativ eine eigene Securityrevision samt Fence
auslösen; er darf den Kausalhead nicht auf seinen Payloadstand setzen. Das
Objekt bleibt `restricted` oder `quarantined`, bis ein autoritativer
Currentness-Beleg die Ordnung wieder auflöst.

### 6.2 Snapshot- und Objekt-Automat

Snapshotverifikation:

```text
candidate -> verified
          -> rejected
          -> quarantined
```

Die Payload eines Snapshots ändert sich nach Anlage nie. Wird ein neuer
`verified` Snapshot aktiviert, erhält der vorherige lediglich
`superseded_at`; er bleibt für Audit und zugesagte Kontexte bis zu seiner
Retention erhalten.

Objektkopf:

```text
absent -> discoverable                     # verified public
absent -> restricted                       # verified direct/scoped
absent -> deleted                         # autorisierter Tombstone

discoverable -> expired | restricted | deleted | quarantined
expired      -> discoverable | restricted | deleted | quarantined
restricted   -> discoverable | deleted | quarantined
quarantined  -> restricted | deleted
deleted      -> [keine Kante]
```

Erlaubte Regeln:

- `absent -> discoverable` verlangt einen autoritativen, öffentlich
  adressierten Genesis-Currentness-Beleg.
- `absent -> restricted` verlangt denselben Authority-/Currentness-Beleg sowie
  eine verifizierte nichtöffentliche Actor-/ACL-Adressierung. Der Snapshot ist
  fachlich `verified`, bleibt aber ausserhalb von `all_federated` und jedem
  unbelegten Scope.
- `absent -> deleted` verlangt einen autorisierten, die Objekt-IRI eindeutig
  bindenden Delete-/Gone-Beleg und legt sofort den absorbierenden
  Domain-Tombstone an. Ein blosses Fremdclaim erzeugt keinen Objektkopf.
- `restricted -> discoverable` verlangt einen neuen autoritativen öffentlich
  adressierten Effekt strikt nach dem Restriction-Head sowie den Fence-
  Clear-Guard; eine `equal`-Revalidierung reicht dafür nie.
- `expired -> discoverable` darf entweder denselben strikten Nachfolger oder
  eine erst nach dem TTL-Ablauf angeforderte autoritative
  `equal`-Current-State-Revalidierung desselben Headtokens und Payloadhashs
  verwenden. Sie erzeugt neue Revision, TTL, Scope-Version und Projektion.
- Eine identische, rechtzeitige Erneuerung darf `discoverable_until` nur
  mit der oben beschriebenen Current-State-Revalidierung verlängern. Der
  Snapshotpayload bleibt unverändert; Zeitgrenze,
  `snapshot_set_revision`, Katalogrevision und Suchprojektion werden jedoch
  gemeinsam fortgeschrieben. Ein dadurch semantisch überholter Cursor darf
  gemäss `revision_fenced` stale werden.
- Eine Payloadänderung erzeugt einen neuen Snapshot, eine Katalogrevision,
  Projektion und `snapshot_set_revision`.
- `discoverable -> expired` ist fachlich bereits bei Erreichen der exklusiven
  Zeitgrenze wirksam. Der Scheduler materialisiert Zustand, Revision und
  Projektion nur zur Bereinigung; Korrektheit hängt nicht von seiner Pünktlichkeit
  ab.
- `403`, ein autoritativ interpretierter `404`, nicht öffentliche Adressierung
  und Public→Private führen aus `discoverable` oder `expired` unmittelbar zu
  `restricted` samt Fence. Aus `restricted` sind sie ein idempotenter No-op
  oder eine neue streng geordnete Restrictionrevision, nie eine Erweiterung.
- `410` oder ein autorisiertes ActivityPub-Delete führen aus
  `absent`, `discoverable`, `expired`, `restricted` oder `quarantined` zu
  `deleted` samt dauerhaftem Domain-Tombstone.
- Ein unbekannter beziehungsweise unautorisierter Actor darf TTL, Payload,
  Restriction oder Tombstone nicht verändern.
- Ein `before`, `concurrent` oder `unknown` eingestufter Effekt darf weder
  `restricted/expired -> discoverable` auslösen noch eine TTL verlängern.
  Empfangs- und Verarbeitungsreihenfolge sind niemals ein Ersatz für diesen
  Guard.
- Ein harter Authority-, Kausal-, Digest- oder Currentness-Widerspruch darf
  `discoverable`, `expired` oder `restricted` nach `quarantined` verengen; bei
  zuvor möglicher Sichtbarkeit entstehen Securityrevision und Fence im selben
  Commit. Quarantäne darf nie die Discoverability erweitern.
- `quarantined -> restricted` verlangt einen neuen autoritativen
  Current-State-Resync, der Authority, Kausalordnung und aktuellen Payloadstand
  gegen den unveränderten Objektkopf beweist. Der Fence bleibt aktiv, bis die
  bestätigte Projektion anschliessend den normalen
  `restricted -> discoverable`-Clear-Guard erfüllt. Beweist der Resync einen
  autoritativen Delete-/Gone-Stand, ist stattdessen `quarantined -> deleted`
  erlaubt. Keine andere Kante verlässt `quarantined`; `deleted` ist
  absorbierend und vollständig terminal.

### 6.3 TTL und Refresh

```text
discoverable_until =
  verified_at_db
  + clamp(akzeptierter signierter Providerhinweis oder lokaler Default,
          federation_ttl_min,
          federation_ttl_max)
```

Ohne akzeptierten Hinweis gilt der versionierte Instanzdefault. Die Remotezeit
darf den lokalen Beginn nicht ersetzen. `refresh_due_at` liegt um einen
versionierten Vorlauf mit deterministischem Jitter vor der Expiry.

Timeout, `429` und `5xx` planen einen Retry, verlängern aber die TTL nicht und
widerrufen den bis dahin noch gültigen Snapshot nicht. `Retry-After` wird
begrenzt respektiert. Nach Ablauf gilt der Zeitguard auch bei ausgefallenem
Scheduler.

Signatur-/HTTP-Zeit muss innerhalb des zugesagten
`inbound_federation_clock_skew` liegen. Eine ausserhalb liegende Nachricht wird
nicht durch Vergrösserung der Discoverability kompensiert.

Bei durablem Inbox-Processing wird das zulässige Eingangszeitfenster gegen das
lokale `received_at`, nicht gegen den möglicherweise viel späteren Workerstart
geprüft. Ein interner Retry macht dieselbe Nachricht dadurch weder nachträglich
gültig noch ungültig.

Für `source = federated | all` verwendet der Gateway den konservativen Horizont
von `all_federated`; für `origin_instance` den der konkreten Origin. Er darf
einen früheren, nie einen späteren Horizont wählen. Ein späterer Vertrag kann
ein querygenaueres Minimum ergänzen, wenn es Treffer, Total und jede
disjunktiv verbreiterte Aggregationsgrundmenge vollständig berücksichtigt.

„Konservativ“ erlaubt hierbei nur eine zusätzlich **frühere** Vertragsgrenze,
nicht einen unbestimmten Scope-Wert: Head und immutable Scope-Version tragen
für ihre gebundene Membership immer das exakte Minimum. Bei nichtleerer Menge
ist ein leeres Feld ebenso ungültig wie ein Wert nach der kleinsten Expiry;
bei leerer Menge ist ein gesetzter Wert ungültig. Scope-Digest und
Snapshot-Federationmanifest binden Mitgliederschlüssel, jede Einzel-Expiry und
dieses Minimum gemeinsam.

`federation_scope_head` ist eine transaktional gepflegte Beschleunigung, kein
Korrektheitsersatz für den Zeitguard. Fehlt der Head, ist er nach Restore
unbestätigt oder liegt sein frühester Horizont bereits bei beziehungsweise vor
`as_of`, darf der Gateway ihn im Requestpfad nicht still nach hinten schieben.
Er kann in einem konsistenten DB-Schnitt das konservative Minimum aller
Objektköpfe mit `state = discoverable` und `discoverable_until > as_of`
diagnostizieren, muss die Änderung aber als normale `expiry`-
Revisionstransaktion samt Scope-Version und Dirty-Projektion einreihen. Bis zu
deren Publikation stellt die betroffene alte Epoch keinen neuen föderierten
Kontext aus. Der Zeitguard selbst schliesst abgelaufene Dokumente unabhängig
von einem pünktlichen Sweeper aus.

Der aktuelle Federation-Head und das epochgebundene Scope-Binding erfüllen
verschiedene Aufgaben. Für `snapshot_set_revision` und `valid_until` einer
Antwort ist ausschliesslich `search_content_snapshot_scope` massgeblich; der
aktuelle Head darf diese Werte nie nach hinten verlängern. Er dient zum Erkennen
neuer Restriction-/Stale-Ereignisse und zur Planung der nächsten Projektion.
Eine nachweislich nicht verengende Änderung nach dem Content-Cutoff darf dem
Head vorauslaufen, ohne den älteren vollständig publizierten Bestand
umzudeuten. Jede andere Änderung wirkt sofort über den Fence. Treffer, Total
und Counts bleiben exakt für den ausgewiesenen gelieferten Bestand und werden
nie mit noch nicht publizierter Payload ergänzt.

## 7. Dirty-Queue, Projektionsversion und Change-Log

### 7.1 Queue-Automat

```text
queued -> leased -> [Erfolg: entfernt oder queued]
                  -> retry_wait -> leased
                  -> dead

dead -- neue höhere desired_revision oder expliziter Admin-Retry --> queued
```

Claim:

- nur `queued`, fälliges `retry_wait` oder `leased` mit
  `lease_until <= now`;
- setzt `claimed_revision = desired_revision`, neuen monotonen `lease_token`
  und `lease_until` per CAS;
- faire Auswahl scannt die drei persistenten Partitionen aus
  `search_projection_sweep_state`: `queued` nach
  `(desired_revision, target_key, id)`, `retry_wait` nach
  `(retry_not_before, desired_revision, target_key, id)` und abgelaufene
  `leased`-Einträge nach `(lease_until, desired_revision, target_key, id)`.
  Der Cursor wird zyklisch gewrappt; kein Worker bedient ausschliesslich den
  heissen Kopf.

Build und Abschluss:

1. Der Worker liest einen konsistenten Fachstand und dessen
   `built_through_revision`.
2. Er erzeugt deterministisch Payload, Inputfingerprint und Hash ausserhalb
   einer langen Schreibtransaktion.
3. Eine kurze Abschluss-Transaktion prüft PocketBase-Record-ID,
   `queue_incarnation`, Lease-Token, Zielidentität und unveränderte relevante
   Inputrevisionen.
4. Sie schreibt `search_projection_version`, aktualisiert das aktuelle Read
   Model und markiert alle dadurch bewiesenen Changes bis zum Buildstand
   `ready` beziehungsweise `subsumed`.
5. Ist `desired_revision <= built_through_revision`, wird die Dirty-Zeile
   entfernt. Andernfalls geht sie ohne Verlust zurück auf `queued`.

Ein verlorener Lease oder geänderter Input verwirft den Abschluss; der Worker
darf sein Resultat nicht erzwingen. Ein transienter Fehler geht mit begrenztem
Backoff nach `retry_wait`. `dead` ist sichtbar und blockiert den betroffenen
geschlossenen Watermarkprefix; es ist kein Skip. Nur eine neuere fachliche
Sollrevision oder ein auditierter Admin-Retry darf ihn erneut einreihen.
Sicherheitsänderungen dürfen nie auf einen Dead-Letter-Worker warten, weil ihr
Fence bereits synchron wirkt.

Ein Dirty-Upsert während `leased` erhöht nur `desired_revision` und lässt Claim,
Token und Inkarnation unberührt. Nach erfolgreichem Löschen erzeugt ein späterer
Upsert einen neuen PocketBase-Record mit neuer `queue_incarnation`. Dadurch kann
ein verspäteter Worker auch bei erneutem Ziel und zurückgesetztem Token keinen
ABA-Abschluss gewinnen.

Worker schreiben nie direkt in Meilisearch. Damit können Lease-Supersession und
CAS vollständig in der Control Plane erzwungen werden.

### 7.2 Subsumption und Tombstones

Da die Dirty-Queue koalesziert, darf eine spätere vollständige
Projektionsversion mehrere frühere Invalidierungen desselben Ziels subsumieren.
Subsumption ist nur zulässig, wenn die Projektion auf einem konsistenten Schnitt
mindestens bis zur höchsten subsumierten Revision beruht. Sie wird im Change-
Log ausdrücklich referenziert.

Delete wird niemals aus dem Fehlen einer Projektionszeile abgeleitet. Der
Change-Eintrag trägt `operation = delete`, stabilen Dokument-/Objektschlüssel,
Revision, Grund, Payloadhash und bei Federation/Security den letzten
Authority-Binding-Hash. Ein Tombstone dominiert jeden älteren Upsert desselben
Ziels. Ein später eintreffender alter Build oder ActivityPub-Create scheitert
am Revision-/Authority-Guard.

Für die Route-Familie löst Builder und Publisher jeden Upsert, Delete,
Federation-Sync und Tombstone über `search_geo_route_assignment` auf. Weder ein
geänderter Fachpayload noch ein Remote-Erstellungsdatum darf das Zielmember
wechseln. Ein Delete entfernt das physische Dokument aus genau diesem
generationengebundenen Shard, erhält Assignment, Slot und logischen
Shardheader aber dauerhaft. Die dadurch frei gewordene live Dokumentposition
ist kein wiederverwendbarer Allocation-Slot; ein Rebuild liest dieselbe
persistente Zuordnung statt die verbleibenden Trails neu zu sortieren.

### 7.3 Geschlossener Prefix und GC

Ein **inputbereiter Zielprefix** `R` liegt vor, wenn jeder relevante Change
`<= R` fachlich aufgelöst ist: `resolution_state` ist `ready`, `subsumed` oder
mit vollständig rekonstruierbarem Kind-/Regelversions-/Digestbeleg
`irrelevant`, seine exakten unveränderlichen Projektions-/Tombstone-/
Settingsinputs sind auswählbar, und deren jeweilige Revisionsobergrenze liegt
höchstens bei `R`. Die zugehörigen `search_catalog_delivery`-Records dürfen in
diesem Zustand ausdrücklich noch `pending` sein. Nur dieser inputbereite
Zielprefix ist Guard für das Anlegen eines Engine-Attempts.

Ein **ausgelieferter geschlossener Memberprefix** und damit ein
Komponenten-Watermark `R` liegt dagegen erst vor, wenn der bestätigte
Baseline-Manifesthash alle relevanten Changes bis `baseline_revision` abdeckt
und für jede danach liegende Katalogrevision `<= R` sowie jeden relevanten
Change entweder `resolution_state = irrelevant` gilt oder genau ein zu diesem
Member gehörender `search_catalog_delivery`-Record terminal `applied`,
`deleted`, `subsumed` oder bei Security `fenced` bestätigt. Change-Revision und
-Ordinal müssen mit dem referenzierten Change übereinstimmen; Task, Projection,
Marker und Fencebeleg erfüllen die zustandsspezifischen Pflichtfelder. Das
Maximum einzelner Acks oder die höchste Engine-Task-ID genügt nie.

Beide Prefixbegriffe sind ausserdem nach oben geschlossen: Kein für Build oder
Publish verwendeter Task darf ein Projektionsartefakt, Tombstone, Settings- oder
sonstiges Kataloginput mit Revision `> R` enthalten. Insbesondere darf ein
Change `<= R`, der durch eine koaleszierte
`search_projection_version.built_through_revision = R2 > R` aufgelöst wurde,
nicht unter Cutoff `R` als zugestellt gelten. Der Publisher verwendet entweder
ein reteniertes passendes Artefakt mit Buildstand `<= R` oder erhöht den Cutoff
vor dem ersten Submit mindestens auf `R2` und schliesst dann ausnahmslos den
gesamten Prefix bis zum neuen Cutoff.

Vor Beginn eines Shadow-Builds registriert die Generation
`capture_from_revision = B`; jedes zu diesem Schnitt bereits benötigte Member
setzt `baseline_revision = B`. Ein erst nach `B` durch eine neue Route-Shard-
Manifestversion eingeführtes Member verwendet gemäss Abschnitt 8.1.1 deren
Einführungsrevision als eigene Baseline, ohne den generationenweiten
`capture_from_revision` anzuheben.
Dadurch bleiben Change-Log, Tombstones und benötigte Projektionsversionen
gepinnt. `B` muss bereits ein inputbereiter Projektionsprefix sein. Der Build
streamt unveränderliche, revisionsmarkierte Payloadartefakte, die diesen Prefix
mindestens abdecken, und bestätigt IDs, Versionen und Coverage im Baseline-
Manifest. Enthält ein gescanntes Artefakt schon eine Revision `> B`, bleibt der
zugehörige Change trotzdem im Deliverybeweis und der spätere Publikationscutoff
muss mindestens diese Revision einschliessen. Danach wird jeder Change mit
`revision > B` bis zu wiederholt gewählten Cutoffs idempotent verarbeitet.
Deletes und Subsumption sind Teil desselben geschlossenen Prefixbeweises.

Ein Route-Shadow-Build bindet zusätzlich die neueste
`search_geo_shard_manifest_version` mit `catalog_revision <= B` und erzeugt
zuerst genau eine immutable
`search_index_generation_query_family(route_radius_ux_v1)` samt validierter
`search_contract_artifact`-Relation, Key und Payloadhash. Diese Zeile ist auch
beim leeren Katalog verpflichtend. Für jeden Manifestshard, einschliesslich des
initialen Shards mit `assigned_slots = 0`, erzeugt und provisioniert er danach
genau ein generationenspezifisches Member, das die Contractwerte aus der
Family-Bindung kopiert. Er liest die
zugehörigen persistenten Assignmentzeilen; ein erneutes Sortieren oder
Verteilen der aktuellen live Fachrecords ist verboten. Entsteht während des
Builds eine spätere Shardmanifestversion, registriert der Catch-up das neue
Member mit exakt derselben Contractbindung und schliesst dessen Changes wie
jeden anderen offenen Prefix. Der Kandidat kann erst `ready` werden, wenn die
`query_family = route_radius_ux_v1`-Teilmenge seiner Zielmember exakt zur
Manifestversion seines Zielcutoffs passt; beim leeren Katalog enthält sie
genau das leere initiale Member. Seine vollständige Queryfamily- und Zielmembermenge samt
beiden Hashes enthalten unabhängig davon weiterhin die Route-Family
beziehungsweise `trail_default`.

Ein Change/Tombstone darf erst gelöscht werden, wenn:

- jede aktive, ready, swap-fähige oder noch aufholende relevante Generation ihn
  bestätigt oder mit einem späteren Vollstand beweisbar subsumiert hat;
- keine registrierte Buildgeneration mit älterem `capture_from_revision` ihn
  mehr benötigt;
- kein retenierter Content-Snapshot beziehungsweise Scope-Binding seine
  Change-, Projection- oder Federation-Scope-Version mehr referenziert;
- keine Buildgeneration und kein retenierter Content-Snapshot die zugehörige
  Route-Shard-Manifestversion oder Assignmentzeile mehr als Coverage-,
  Member- oder Routingbeleg benötigt;
- bei Security jede noch querybare versiegelte Generation physisch sicher ist
  oder ein weiterhin retenierter aktueller Fence sie vollständig ausschliesst;
- die administrative Mindestretention abgelaufen ist.

Federation-Domain-Tombstones für `deleted` werden in V1 nicht automatisch
gelöscht.

## 8. Generationen, Publikation, Swap und Rollback

### 8.1 Generationen-Lifecycle

```text
new -> provisioning -> backfilling -> catching_up -> validating -> ready
                                                              |
                                  active <---------------------+
                                     |
                                  sealed -> retired -> deleting -> deleted

abandoned  -> retired
quarantined -> retired

ready -- committed_revision > catalog_cutoff --> catching_up
```

Vor `active` kann ein irreparabler Kandidat nach `abandoned` oder
`quarantined` wechseln. Eine aktive Generation mit unbewiesener Engine-
Inkarnation, Coverage, Settings oder Payload behält ihren Head, wechselt aber
im Epoch-Automaten fail-closed nach `quarantined`; eine versiegelte Generation
wechselt im Lifecycle dorthin. Keine davon geht zurück nach `ready`.
`abandoned -> retired` ist erst zulässig, wenn alle Attempts terminal sind,
die Generation nie Head war und ihre ausschliesslich `staged`en Ressourcen
ohne HWM oder Buildpin `gc_eligible` sind. `quarantined -> retired` verlangt
zusätzlich, dass kein Head, Cursorpfad oder Swapset mehr auf die Generation
zeigen kann, ihre Ressourcen terminal `lost` oder nach HWM plus Grace
`gc_eligible` sind und alle unbekannten Enginewrites durch den jeweiligen
Fence-/Restorevertrag unschädlich gemacht wurden. Von `retired` geht es erst
nach `deleting`, wenn sämtliche Member-/Artefaktpins freigegeben sind;
`deleted` folgt ausschliesslich nach bestätigter physischer Löschung aller
Member und der Generationressource. Keine dieser GC-Kanten reaktiviert einen
Attempt oder eine Ressource.
`ready` verlangt:

- die vollständige deklarierte Generation-Queryfamily-Menge einschliesslich
  `route_radius_ux_v1` vorhanden und ihr Manifesthash bestätigt;
- die vollständige Zielmembermenge aller Queryfamilien vorhanden und ihre
  Schema-/Settingshashes bestätigt;
- vollständigen Basisbuild und geschlossenen Cutoff-Replay;
- Delivery- und Security-Watermarks jedes Mitglieds mindestens am Zielcutoff;
- alle aktuell aktiven Fences physisch `applied` oder für den Querypfad
  nachweislich `query_fenced`;
- Coverage-, Marker- und Beispielquery-Gates;
- vollständiges Federationmanifest mit Scope-Versionen am Zielcutoff;
- bei `route_radius_ux_v1` die cutoffgenaue persistente
  Shardmanifestversion und ausschliesslich für die auf diese Queryfamilie
  gefilterte Zielmemberteilmenge eine bijektive Bindung **aller** darin
  enthaltenen Shards, auch des initialen Shards mit null Assignments;
  `trail_default` bleibt ausserhalb dieser Bijektion;
- für die Route-Generation-Family unabhängig von der Dokumentbelegung eine gültige
  Contract-Artefakt-Relation samt Key und Payloadhash, einen vollständig
  validierten kanonischen Payload und eine für den Kandidaten mindestens
  `staged`, beim Serving `issuing`e Contract-Ressource; alle Route-Member
  kopieren exakt daraus;
- kompatible Contract-, Engine- und Gatewayversion.

Nicht querybare Generationen dürfen während `backfilling/catching_up` über
persistierte Engine-Attempts der Art `build_batch/catch_up` Writes ausführen,
ohne dabei einen alten öffentlichen Snapshot zu schützen. Jeder Intent und
Task bleibt trotzdem durable und idempotent recoverbar. Nach allen Gates
publiziert ein CAS den ersten `search_content_snapshot` und wechselt
`unpublished(content_epoch = 0) -> published(content_epoch = 1)`; erst danach
ist `ready` zulässig. Derselbe CAS legt sämtliche
`search_content_snapshot_query_family`-, `search_content_snapshot_member`- und
`search_content_snapshot_scope`-Records samt ihren gebundenen Manifesthashes
an. Generation und erster Snapshot erhalten dabei
`search_context_resource.state = staged`, weil vor dem Head-Swap noch kein
öffentlicher Kontext auf sie zeigen darf.

`ready` ist vor dem Head-Swap kein eingefrorener Sackgassenzustand. Liegt die
committed Katalogrevision über seinem Cutoff, setzt ein CAS den nicht querybaren
Kandidaten `ready/published -> catching_up/updating` und startet einen
persistierten `epoch_publish`-Attempt mit den nötigen Catch-up-Taskintents und
denselben Prefix-, Task- und Recoveryguards wie unten. Ein vorgelagerter
`catch_up`-Attempt darf Inputs vorbereiten, aber keinen Snapshot publizieren.
Der Abschluss des `epoch_publish` erzeugt einen neuen
Content-Snapshot mit `staged`-Ressource, setzt
`catching_up/updating -> validating/published`, verwirft den vorigen noch nie
ausgestellten Snapshot nach dessen Buildpins über `staged -> gc_eligible` und
führt nach erneuter Validierung wieder zu `ready`. Die Generationsressource
bleibt während der ganzen Schleife `staged`.

Der Swap-CAS liest `search_catalog_state.committed_revision` unter demselben
SQLite-Schreiblock wie den Head. Ist der Kandidat inzwischen erneut zurück,
scheitert nur der CAS und die obige Catch-up-Schleife läuft weiter. Ist er
aktuell, kann keine Mutation zwischen Guard und Pointercommit treten. Damit ist
eine bereits einmal erreichte `ready`-Generation nach späteren Revisionen
weiterhin aufhol- und swapfähig.

#### 8.1.1 Append-orientierte Route-Shard-Familie

Die logische Shardzuordnung ist generationsübergreifend stabil, die physische
Umsetzung dagegen immer generationengebunden. Für jede Generation und jeden
Manifestshard, einschliesslich des initialen Shards mit null Assignments,
existiert deshalb eine andere, global nie
wiederverwendete `physical_index_id`. `geo_shard_id` wird weder aus ihr
abgeleitet noch beim Rebuild geändert. Der von ADR 0002 festgelegte
Route-Search-Vertrag wird ausschliesslich durch die explizite Relation, den
Key und den Payloadhash des immutable `search_contract_artifact` in
`search_index_generation_query_family` gebunden; das gilt bereits bei null
Assignments und vor der Provisionierung des verpflichteten initialen Members.
Route-Member kopieren diese Werte nur;
Schema-, Settings- und Bundlehash reichen dafür nicht aus. Die je Epoch
veränderliche vollständige Membermenge aller Queryfamilien wird getrennt durch
`member_manifest_hash` gebunden.

Jede Generation registriert für den initialen Shard ein Member mit dessen
persistenter `geo_shard_id`, `introduced_at_revision = 0` und einer eigenen
global nie wiederverwendeten `physical_index_id`. Sein Basisbuild erzeugt und
validiert einen realen leeren Meilisearch-Index samt Schema, Settings,
Contract-Key/-Hash, Coverage und Endmarker; ein nur in PocketBase behauptetes
Member genügt keinem Ready-, Publish- oder Swap-Guard.

Der erste katalogfähige Trail insgesamt erzeugt keinen neuen logischen Shard
und kein neues Generationenmember. Sein Allocation-CAS füllt Slot `0` des
persistenten initialen Shards Ordinal `0`, dessen bereits vorhandenes leeres
physisches Member danach über normale Delivery, Catch-up und eine neue Epoch
befüllt wird. Ein zuvor ausgestellter Empty-Kontext wird nach den normalen
Epoch-/Relevanzregeln stale und niemals gegen den neuen befüllten Stand als
weiterhin leer oder teilweise neu interpretiert.

Wird im aktiven Betrieb der erste Trail eines neuen logischen Shards
zugewiesen, registriert derselbe Control-Plane-Schnitt für die aktive
Generation und jede noch build-, catch-up- oder swap-fähige Generation das
deterministische Generationenmember im Zustand `new`. Eine versiegelte, nur
noch durch alte Cursor querybare Generation erhält dieses spätere Member
nicht: Ihre unveränderliche Content-Epoch liegt vor der Assignmentrevision.
Das neue Member übernimmt exakt Family-Relation, Contractrelation, Key und
Hash aus `search_index_generation_query_family(route_radius_ux_v1)`; ein
Artefakt- oder Profilwechsel innerhalb der Generation ist verboten.
Bei einem erstmals angelegten Shard ist `introduced_at_revision` und die
anfängliche `baseline_revision` exakt die Katalogrevision seiner ersten
Assignment-/Manifestversion; der Baseline-Coveragebeleg muss alle diesem Shard
bis dahin zugewiesenen Trails einschliessen. Spätere Änderungen laufen normal
über Deliveries nach dieser Baseline.

Für ein neues Member einer aktiven Generation gilt:

```text
nicht im publizierten Snapshot
  -> new -> building -> ready
  -> nächster epoch_publish bindet es im neuen Content-Snapshot
```

Solange das Member in keinem publizierten
`search_content_snapshot_member` vorkommt, darf es über persistierte
`build_batch`-/`catch_up`-Attempts provisioniert und beschrieben werden,
während die bisherige Generation `active/published` bleibt. Diese enge
Ausnahme mutiert weder ein bereits snapshotgebundenes Member noch dessen
Marker, Cutoff oder Content-Epoch; der Gateway kann den neuen physischen Index
nicht adressieren. Alle Taskinputs müssen ausschliesslich Trails mit dem
passenden persistenten Assignment und Revisionen nach dem bisherigen
Snapshot-Cutoff betreffen.

Erst wenn das neue Member `ready` ist, seine vollständige Baseline, alle
relevanten Deliveries, Security-Fences und Marker bestätigt sind und der
gesamte Zielprefix inputbereit ist, darf der normale `epoch_publish` beginnen.
Sein finaler CAS bindet die neueste Shardmanifestversion am Zielcutoff und die
dazu bijektive, nach `query_family = route_radius_ux_v1` gefilterte
Zielmemberteilmenge einschliesslich des initialen Shards. Die vollständige Snapshot-
Membermenge enthält daneben weiterhin `trail_default`; die unabhängig von der
Memberzahl erzeugte Snapshot-Queryfamily kopiert dieselbe Contract-Artefakt-
Relation samt Key und Hash, und vorhandene Route-Member kopieren daraus.
Misslingt Provisionierung oder
Catch-up, bleibt der alte Snapshot unverändert querybar und der neue
Katalogstand unpubliziert; eine Teilfamilie oder ein Query-Fallback auf einen
anderen Geo-Pfad ist verboten.

`AttemptMemberWatermarkBaselineSetV1` bindet bei jedem Attempt die
vollständige für sein Ziel vorgesehene Membermenge. Der davon unabhängige
`target_query_family_manifest_hash` und `target_route_query_family` binden die
Route-Contract- und Shardmanifestwerte auch bei katalogweit null Assignments;
die Route-Zielmenge enthält dann genau das dokumentleere initiale Member. Ein
nach Attemptanlage hinzukommender Shard kann
nicht still in denselben Attempt aufgenommen werden: Der CAS verliert
beziehungsweise der nächste vollständige Catch-up-/Epochversuch bindet die
höhere Manifestversion. Globale Settingsänderungen müssen jedes betroffene
Member der vollständigen Zielmenge erreichen. Eine Contractänderung erzeugt
ein neues immutable Artefakt und eine Shadowgeneration; sie schreibt nie die
Contractbindung bestehender Member um. Trailbezogene Änderungen werden nur
zum Member ihres unveränderlichen Assignments geroutet.

### 8.2 Epoch-Publisher

Publisher-Ownership folgt diesem Automaten:

| Von | Guard | Nach |
| --- | --- | --- |
| `vacant` | CAS setzt neuen Holder, bindet aktuelle Restore-/Credential-Epoche und erhöht `term` | `held` |
| `held` | CAS sperrt neue Attempts; bereits durabel angelegte, nichtterminale Attempts bleiben dem aktuellen Holder/Term zugeordnet | `draining` |
| `draining` | kein `updating` und kein unresolved Attempt | `vacant` |
| `held/draining` | alter Prozess ist nachweislich durch Netzpolicy, Write-Proxy oder Credentialrotation vom Writepfad getrennt | `fenced` |
| `quarantined` | auditierter Recovery-Command klassifiziert jeden unresolved Attempt; alle alten Writer technisch gefenced, Widerspruch behoben oder auf neue IDs isoliert | `fenced` |
| `fenced` | CAS setzt neuen Holder und ein lexikographisch höheres bestätigtes Restore-/Credential-/Term-Tupel | `held` |
| jeder Zustand | Term-, Credential- oder Engine-Inkarnation widersprüchlich | `quarantined` |

Leaseablauf allein erlaubt weder `held -> vacant` noch eine Übernahme. Er löst
nur den Failoverprozess aus, der zuerst das technische Writer-Fence bestätigen
muss. Nur `held` erlaubt den Claim eines neuen Attempts. Der aktuelle Holder
darf in `draining` ausschliesslich bereits vor dem Drain-CAS durabel angelegte,
nichtterminale Attempts desselben Terms beenden; neue Attempts sind gesperrt.
`fenced`, `vacant` und
`quarantined` erlauben keinen Submit.

`quarantined` ist keine still reparierbare Sackgasse, besitzt aber genau die
oben gezeigte administrative Recoverykante. Vor `quarantined -> fenced`
bleiben Gateway und Publisher fail-closed. Der CAS bindet einen auditierten
Recovery-Command, einen neuen vollständigen Infrastruktur-Fence-Beleg und eine
höhere Credential-Epoch; er leert den Holder und klassifiziert jeden
nichtterminalen Attempt entweder als unverändert recoverbar oder gemeinsam mit
seiner Generation terminal `quarantined`. Attempt-, Task- und
Submissionrecords werden nie gelöscht oder umgedeutet. Bei Engine-
Inkarnations-/Markerwiderspruch darf nur eine Recovery-Generation auf neuen
physischen IDs fortfahren. Erst der normale nachfolgende `fenced -> held`-CAS
vergibt das höhere Writer-Fence-Tupel; ein direkter Übergang aus
`quarantined` nach `held` oder `vacant` ist verboten. Liegt der Widerspruch an
einem PocketBase-Restore, ersetzt dieser Ablauf nicht Abschnitt 10.6.

Existiert für eine Generation kein nichtterminaler Attempt, übernimmt ein neuer
Holder ihren Idle-Term mit `adopt_idle_generation_term`. Derselbe
PocketBase-CAS prüft Publisher `held`, dessen
Restore-Epoch/Holder/Term/Credential-Epoch,
unveränderte Generationversion und Engine-Inkarnation sowie das Fehlen jedes
Nichtterminal-Attempts für Generation und serialisierten Writepfad; er setzt
das Writer-Fence-Tupel der Generation auf das globale Publishertupel und legt
unmittelbar den neuen Attempt mit genau diesem Tupel an. Das gilt
für aktive und idle Kandidatengenerationen sowie für eine noch
cursorquerybare versiegelte Generation ausschliesslich zum Claim eines
`fence_apply`; alle normalen Content-/Settings-Attempts bleiben dort verboten.
Nach einem sauberen
`draining -> vacant -> held` ist kein zusätzliches externes Fence nötig, weil
der Drain-Guard bereits alle alten Attempts abgeschlossen hat. Stammt der neue
Term aus `fenced -> held`, muss der CAS dagegen weiterhin den gespeicherten
`fence_evidence_hash` und die neue Credential-Epoch binden. Eine getrennte
Best-effort-Aktualisierung des Generationterms ist verboten.

Nach einem Failover übernimmt der neue Holder einen nichtterminalen Attempt nur
mit `adopt_recovery_attempt`: Ein PocketBase-CAS prüft den externen
`fence_evidence_hash`, höhere Restore-/Credential-/Term-Identität,
unverändertes vollständiges Membermanifest/Ziel, bei Route-Attempts die
unveränderte Generation-Queryfamily samt Bindinghash, die auch beim leeren
Katalog mindestens das initiale Member enthaltende Geo-Teilmenge und Contract-
Artefakt-Relation samt Key/Hash sowie die bisherige
Engine-Inkarnation. Er setzt
Writer-Fence-Tupel von Generation und Attempt sowie Attemptzustand gemeinsam
auf das bereits am Publisherrecord gehaltene neue Tupel beziehungsweise
`recovery`.
Das `created_writer_*`-Tupel, Intent-Keys und vorhandene Submissions bleiben
unverändert. Erst danach darf der neue Holder bekannte Tasks prüfen oder einen
zulässigen weiteren Submit ausführen. Ohne diesen CAS bleibt der Attempt
blockiert.

Für eine abfragbare aktive Generation gilt:

```text
published(e)
  -> updating(attempt, cutoff R)
  -> published(e + 1)
     | recovery
     | quarantined
recovery(attempt, cutoff R)
  -> published(e + 1)
     | quarantined
```

Normative Reihenfolge:

1. Der logisch einzige Publisher wählt einen vorläufigen inputbereiten Zielcutoff
   `generation.catalog_cutoff <= R <= search_catalog_state.committed_revision`
   und löst die exakten unveränderlichen
   Projektions-, Tombstone- und Settingsartefakte auf. Ein reteniertes älteres
   Artefakt ist nur ein Input für die anschliessende Vorwärtsprojektion, nie
   die Erlaubnis, den publizierten Cutoff zurückzusetzen. Für eine Route-
   Familie löst er zusätzlich die neueste persistente
   Shardmanifestversion mit `catalog_revision <= R`, die immutable
   Generation-Queryfamily samt Contractbindung und die daraus folgende,
   mindestens das initiale Shardmember enthaltende
   `route_radius_ux_v1`-Zielmemberteilmenge auf. Die vollständige
   generationenspezifische
   Zielmembermenge enthält daneben auch `trail_default`.
2. Vor jedem Submit berechnet er deren maximales
   `input_revision_ceiling = M`. Ist `M > R`, setzt er `R = M`, schliesst alle
   fachlich noch nicht aufgelösten Changes bis dorthin und löst die Artefakte
   erneut auf. Diese Fixpunktbildung endet erst, wenn der gesamte Zielprefix
   inputbereit ist und
   jedes gewählte Input `<= R` gilt; alternativ verwendet er ein reteniertes
   älteres Artefakt `<= R`. Bei jeder Erhöhung löst er auch
   Shardmanifestversion, vollständige Queryfamily-/Zielmembermenge,
   Route-Teilmenge und Contract-Artefakt erneut auf. Währenddessen gibt es noch keinen
   Engineeffekt.
3. Ein DB-CAS prüft `active/published`, `from_epoch`, Generationversion und
   Publisherzustand `held`. Er bestätigt ein bereits aktuelles Writer-Fence-Tupel
   oder führt im selben Schnitt `adopt_idle_generation_term` aus, legt Attempt plus alle
   deterministischen Taskintents `prepared` an, bindet
   `from_catalog_cutoff = generation.catalog_cutoff`, die vollständige
   `member_watermark_baseline`, `target_member_manifest_hash`,
   `target_query_family_manifest_hash` sowie bei einer Route-Familie
   `target_geo_shard_manifest_version/hash`, `target_route_query_family` und
   `target_route_contract_artifact` samt Key/Hash; diese Route-Bindungen sind
   auch beim belegungsleeren initialen Route-Zielmember gesetzt,
   `from_catalog_cutoff <= target_cutoff = R` sowie
   `manifest_max_input_revision <= R` und
   `R <= search_catalog_state.committed_revision` gegen den im selben CAS
   gelesenen Wert; jede eingefrorene
   `delivered_revision` und jeder eingefrorene `security_watermark` liegen
   ebenfalls höchstens bei `R`, `target_security_revision` ist nicht gesetzt.
   Dann setzt er die
   Generation dauerhaft auf `updating`. Cutoff, Baseline, Manifest und Taskinputs sind danach
   unveränderlich; ein später benötigter höherer Stand gehört in die nächste
   Epoch.
4. Unmittelbar vor jedem einzelnen Engine-Submit bestätigt der Publisher erneut
   Attemptzustand, Generationversion, aktuelle Restore-/Credential-/Term-
   Identität und Holderzustand. Für Route-Tasks prüft er ausserdem unveränderte
   Contract-Artefaktfelder, rekonstruierbaren Payloadhash und einen nicht
   verlorenen `staged/issuing/retained`-Resourcezustand. `draining` ist nur für
   einen nachweislich vor dem Drain-CAS durabel angelegten, nichtterminalen
   Attempt desselben Holders/Terms zulässig. Erst nach diesem Guard darf der
   Write erfolgen.
5. Jede physische Submission, ihre Task-ID und ihr terminaler Zustand werden
   persistiert. Nur vollständig bewiesenes `succeeded` gilt als Erfolg; `202`,
   Transporterfolg oder die höchste Task-ID nicht.
6. Nach Taskabschluss werden Payload-/Settingsmarker, die exakten Task-
   Revisionsceilings, Coverage, Delivery- und Security-Watermark jedes
   manifestgebundenen Zielmembers geprüft. Bei Route-Membern müssen Payload,
   Coverage und Endmarker zusätzlich exakt den eingefrorenen
   Contract-Artefakt-Key/-Hash bestätigen. Erst die terminalen Delivery-Acks
   machen aus dem Zielprefix den ausgelieferten geschlossenen Memberprefix.
7. Ein letzter CAS beweist erneut
   `generation.catalog_cutoff = from_catalog_cutoff <= target_cutoff`,
   `manifest_max_input_revision <= target_cutoff` sowie
   `target_cutoff <= search_catalog_state.committed_revision` gegen den im
   finalen CAS gelesenen Wert. Bei einer Route-Familie müssen die eingefrorene
   Shardmanifestversion weiterhin die neueste Version `<= target_cutoff`, ihr
   Hash unverändert und ausschliesslich die nach
   `query_family = route_radius_ux_v1` gefilterte Zielmemberteilmenge dazu
   über alle Manifestshards bijektiv sein. Die eingefrorene Generation-
   Queryfamily muss weiterhin exakt das rekonstruierbare Contract-Artefakt
   binden, und alle vorhandenen Route-Member müssen daraus kopiert sein;
   `trail_default` bleibt Teil des
   vollständigen `target_member_manifest_hash`. Ausserdem
   bestätigt der CAS für jedes unverändert inkarnierte
   manifestgebundene Zielmember aktuelle Watermarks mindestens an der
   eingefrorenen Baseline. Weder eingefrorene `delivered_revision` noch eingefrorener
   `security_watermark` dürfen oberhalb des Zielcutoffs liegen. Aktuelle
   `delivered_revision` und `security_watermark` werden nie abgesenkt. Liegt
   eine von ihnen oberhalb des Zielcutoffs, muss der vollständige
   `AttemptPostBaselineSecurityProofSetV1` den lückenlosen Vorlauf exakt aus
   nach Attemptanlage entstandenen, weiterhin nicht-retirten Gateway-Fences
   und etwaigen vollständig irrelevanten Zwischenchanges schliessen;
   andernfalls verliert der CAS. Insbesondere darf kein normaler physischer
   Content-/Settings-Effekt oberhalb des Zielcutoffs vorausgelaufen sein. Der
   Generation-Memberprefix kann durch diese Securityausnahme bereits vor dem
   Snapshot liegen; dessen unveränderliche `delivered_catalog_revision` und
   der Generation-`catalog_cutoff` bleiben dennoch exakt
   `target_cutoff`. Dann publiziert derselbe Commit Snapshot-ID, die aus
   `federation_scope_version <= target_cutoff` abgeleiteten Scope-Bindings, die
   vollständigen immutable Snapshot-Queryfamily- und Snapshot-Memberzeilen
   aller Queryfamilien samt beiden Manifesthashes, die cutoffgenaue Route-
   Shard-Manifestbindung, die Route-Snapshot-Family samt Contractrelation,
   Key/Hash auch beim belegungsleeren initialen Member und deren Kopie in allen Route-Membern,
   `content_epoch + 1`, den monotonen Cutoff und `published` gemeinsam. Ein
   Baseline-, Membership- oder Vorlaufkonflikt bricht den CAS ab und darf den
   physischen Stand nicht als kleinere Epoch deklarieren.

Der `epoch_publish`-Attempt selbst folgt; die übrigen Attempt-Arten verwenden
dieselben Recoverykanten, aber keinen Epochübergang:

```text
prepared -> submitting -> waiting -> verifying -> committed
prepared/submitting/waiting/verifying -> recovery
prepared/recovery -- nur Nicht-epoch_publish vor möglichem Engineeffekt --> abandoned
submitting/waiting/verifying -> failed -> recovery
recovery -> submitting | waiting | verifying
recovery/verifying -> committed  # nur über vollständigen Abschlussbeweis
submitting/waiting/verifying/failed/recovery -> superseded
  # nur fence_apply: Binding überholt, alle möglichen Effekte terminal/barriered
failed/recovery -- Stand nicht beweisbar ----------> quarantined
```

`committed`, `superseded`, `abandoned` und `quarantined` sind terminal.
`superseded` ist ausschliesslich für `fence_apply` erlaubt, wenn das gebundene
Frontier nach einem möglichen Submit überholt wurde, jede mögliche Submission
terminal beziehungsweise barrier-covered und der tatsächlich letzte
Visibilitymarker bekannt ist. Die Kante committet keine Delivery und keinen
Watermark aus dem alten Binding; sie lässt alle Guards stehen und legt die in
Abschnitt 4.3 verlangte aktuelle Reparatur durable an. Fehlt einer dieser
Belege, bleibt nur `recovery` oder `quarantined`.

Für `attempt_kind = epoch_publish` ist `abandoned` vollständig verboten, weil
Attemptanlage und `generation.epoch_state = updating` atomar zusammenfallen;
der Record muss mit demselben Manifest beendet, über Recovery übernommen oder
quarantänisiert werden. Für die übrigen Attempt-Arten ist `abandoned` nur vor
jedem möglichen Engineeffekt, also solange keine Submission je nach
`submitting` gewechselt hat, und zusammen mit dem zuständigen
Generationen-Lifecycle-CAS erlaubt. Bereits persistierte
`prepared`-Submissions müssen derselbe Abschluss-CAS oder ein unmittelbar
vorheriger CAS mit typisiertem Non-Submission-Proof terminal `canceled`
haben, und alle Taskintents wechseln im Abschluss-CAS terminal nach
`canceled`; ein `submitting`, `accepted`, `unknown` oder sonst möglicherweise
wirksamer Record verbietet `abandoned`. `failed -> recovery`
verwendet denselben Attempt, Manifest und dieselben Intent-Keys; ein neuer
Attempt für dieselbe Ziel-Epoch ist unzulässig. `quarantined` schliesst den
Attempt dauerhaft, bevor ein Recovery-Kandidat auf neuen physischen IDs gebaut
werden darf.

Die effektfreie Kante `recovery -> abandoned` ist für einen nicht
epochgebundenen Attempt ausdrücklich zulässig, wenn Adoption beziehungsweise
frühere Ausführung keinen möglichen Engineeffekt hinterlassen hat und sein
unveränderlicher Frontier-/Manifestguard beim Abschluss nicht mehr aktuell ist;
ob der Guard vor oder nach der Adoption verloren ging, ist unerheblich. Der
`adopt_recovery_attempt`-CAS darf einen solchen bereits überholten
`prepared`-Attempt entweder zunächst unverändert nach `recovery` übernehmen
oder Adoption und effektfreies `prepared -> abandoned` atomar verbinden. Er
prüft dafür die historische Attemptidentität und den technischen Writer-Fence,
nicht fälschlich die fachliche Aktualität des bereits als überholt erkannten
Payloads; zwischen Adoption und Abschluss bleibt jeder Submit verboten. Der
gemeinsame Abschluss-CAS verlangt dieselben vollständigen
Non-Submission-/Cancellation-Belege für alle Submissionrecords und
Taskintents wie `prepared -> abandoned`, bindet zusätzlich das bei der Adoption
bestätigte technische Writer-Fence und ändert weder Content-Epoch noch
Member-Watermark. Sobald irgendeine Submission `submitting` erreicht hat, ist
diese Kante verboten; dann gelten ausschliesslich Effektklärung,
`superseded` für `fence_apply`, Recovery oder Quarantäne.

Die Kante `prepared -> recovery` ist insbesondere für den Failover nach dem
Attemptcommit und vor dem ersten Engine-Submit erforderlich. Wie jede
Übernahmekante ist sie nur im selben `adopt_recovery_attempt`-CAS mit
bestätigtem technischem Fence des alten Writers und höherem Writer-Tupel
erlaubt; Manifest, Taskintents, Baseline und Ziel bleiben bytegleich. Ohne
Holderwechsel kann derselbe noch berechtigte Holder aus `prepared` normal nach
`submitting` fortsetzen.

Nur bei `attempt_kind = epoch_publish` bleibt die Generation nach
`adopt_recovery_attempt` im Epochzustand `recovery`, während der Attempt je
nach durablem Befund nach `submitting`, `waiting` oder `verifying` weiterläuft.
Sein einziger Erfolgsabschluss ist derselbe finale Epoch-CAS wie im
Normalpfad: Er prüft sämtliche Tasks, Submissions, Marker,
Revisionsceilings, den unveränderten `from_catalog_cutoff`, die persistente
Memberbaseline sowie monotone Delivery-/Security-Watermarks und setzt Attempt
`committed`, neuen Content-Snapshot sowie Generation
`recovery -> published(e + 1)` in einem SQLite-Commit. Ein direkter
Recovery-Erfolg ohne diesen Beleg ist verboten.

Bei `build_batch`, `catch_up`, `validation` und `fence_apply` verändert die
Adoption dagegen weder Epochzustand noch Content-Epoch/-Snapshot. Sie übernimmt
nur Writer-Tupel und denselben unveränderlichen Attempt samt Tasks. Danach
gelten seine artbezogenen Task-/Marker-/Watermark- und Lifecycle-Gates;
`recovery -> submitting|waiting|verifying -> committed` beendet ihn ohne
Epoch-CAS. Ein überholter `fence_apply` ohne möglichen Engineeffekt darf über
die soeben definierte Kante `recovery -> abandoned` enden; nach einem möglichen
Submit bleibt ausschliesslich der vollständig belegte `superseded`-Pfad aus
Abschnitt 4.3. Besonders
eine aktive oder versiegelte `published`e Generation bleibt dabei in genau
diesem Epochzustand; ein zustandserhaltender Security-Recovery darf keine neue
Content-Epoch erfinden.

Nach einem möglicherweise angenommenen Engine-Write existiert kein Übergang
zurück zu `published(e)`. Unklare Taskannahme führt nach `recovery` und bleibt
als eigene `search_engine_submission` sichtbar. Eine weitere Submission ist nur
zulässig, wenn der Engineadapter sowohl die Effektidempotenz als auch eine
geordnete Barriere beweist und danach sämtliche alten und neuen Submits sowie
den erwarteten Endmarker bestätigt. Für den V1-Meilisearch-Pfad gilt ohne einen
solchen Beleg: `unknown` wird in derselben Generation nicht erneut eingereicht;
eine aktive Generation wird quarantänisiert, ein Kandidat verworfen und auf
neuen physischen IDs neu gebaut. Ein bestätigtes Submission-
`failed/canceled` darf nur dann eine neue Submission desselben noch nicht terminalen
Intents erhalten, wenn der passende persistierte
`terminal_no_effect_proof_kind/hash` den vom versionierten Enginevertrag
garantierten fehlenden Teileffekt bindet und noch kein späterer anderer Intent
ausgeführt wurde. Ist wegen
Engine-Reset, anderer `engine_incarnation` oder fehlendem Marker der Stand nicht
beweisbar, gilt ebenfalls Quarantäne statt geratenem Erfolg.

Ein PocketBase-Lease oder Writer-Fence-Tupel allein ist kein durch Meilisearch
erzwungener Writer-Fence. V1 erlaubt deshalb genau einen Credential-/Netzpfad
für Engine-Writes. Vor Failover muss der alte Prozess durch Prozessaufsicht,
Write-Proxy, Credentialrotation oder Netzpolicy technisch vom Writepfad
getrennt sein. Erst danach darf ein höherer Term übernommen werden.

### 8.3 Atomarer Generationstausch

Physische Index-IDs enthalten die unveränderliche Generation und werden beim
Cutover nicht umbenannt. Der atomare Bundlewechsel ist eine einzige
PocketBase-Transaktion. Sobald sämtliche statischen und dynamischen Bindings
feststehen, wird der deterministische Auditrecord einmal als `prepared`
angelegt; das verändert den Head nicht. Beim Legacy-Authority-Handoff geschieht
dies ausdrücklich erst nach dem letzten Buildtask und der unten definierten
Precommit-Beobachtung. Die Pointertransaktion setzt ihn gemeinsam mit allen
übrigen Änderungen auf `committed`. Ein
`prepared -> aborted` ist nur erlaubt, wenn noch kein Pointercommit stattfand
und ein dauerhaft verlorener Guard beziehungsweise expliziter Abbruch belegt
ist; `committed` und `aborted` sind terminal.

Der Handoff-spezifische Teil eines Bootstrap-Kommandos ist genau dieses
geschlossene Objekt; alle Properties sind verpflichtend:

```text
State1LegacyHandoffRequestV1 = {
  "schema": "state1_legacy_handoff_request_v1",
  "reason": "bootstrap",
  "requested_handoff_logical_uids": [non-empty text]
}
```

`requested_handoff_logical_uids` ist die unveränderliche Commandabsicht, nicht
bereits der gleichnamige Inhalt des späteren Swaprecords. Der Wert ist eine
nichtleere, duplikatfreie, nach UTF-8-Bytes sortierte Liste nichtleerer
Strings. Erst die Precommit-Stufe
materialisiert daraus die geschlossenen `handoff_*`-Felder. Vorher existiert
bewusst weder ein persistenter Handoff-Buildplan noch ein adoptierbarer
Snapshotbeleg. Ein Crash oder Prozesswechsel vor Precommit quarantänisiert
deshalb jede angelegte staged Generation; der nächste Lauf baut nach neuem
Preflight vollständig auf neuen physischen IDs.

Ein bei einem neuen Kommando gefundener `search_generation_swap(state =
prepared)` mit Handofffeldern wird vor jeder weiteren Wirkung vollständig
schemavalidiert. Bei gültigem Record darf ein Request dessen Rollenmenge weder
ersetzen noch erweitern. Weil sein früherer Source-/Authority-Lockkontext nicht
adoptiert werden kann, wird er nach belegtem Nichtcommit `aborted`, seine
Zielgeneration quarantänisiert und die Absicht nur für einen vollständigen
Neubau rekonstruiert.

```text
Handoff-Dispatch, vor prepared-Record, Aufbau von new und jedem anderen Effekt:
  falls requested_handoff_logical_uids nicht leer ist oder ein bestehender
    search_generation_swap(state=prepared) irgendein handoff_* Feld trägt:
    head.state = empty AND head.active_generation leer AND V = 0;
      andernfalls state1_incremental_authority_handoff_unsupported, null Wirkung
    reason = bootstrap;
      andernfalls state1_handoff_binding_invalid, null Wirkung
    ohne bestehenden prepared-Swap:
      Request ist geschlossen; requested_handoff_logical_uids ist kanonisch,
        nichtleer und duplikatfrei; andernfalls
        state1_handoff_binding_invalid, null Wirkung
    mit bestehendem prepared-Swap:
      handoff_logical_uids und handoff_authority_snapshot sind beide gesetzt,
        vollständig schema-valide und bijektiv; ein Request ist entweder leer
        oder nennt exakt dieselbe kanonische Menge; andernfalls
        state1_handoff_binding_invalid, null Wirkung
      requested_handoff_logical_uids exakt aus der validierten Menge rekonstruieren
    jede Rolle dieser Menge ist vom V1-Entity-/Querymodell unterstützt;
      andernfalls state1_handoff_uid_unsupported, null Wirkung
    bei bestehendem prepared-Swap erst jetzt nach Nichtcommitbeleg Swap ->
      aborted und seine Zielgeneration samt Membern quarantänisieren; keine
      Generation oder Buildbindung übernehmen
    zum Legacy-Authority-Handoff-Preflight verzweigen
  andernfalls:
    kein Authority-Handoff; der spätere Swaprecord lässt sämtliche
      handoff_* Felder leer

Legacy-Authority-Handoff-Preflight:
  vor prepared-Record, Aufbau von new sowie jedem Head-, Authority- oder
    Engineeffekt
  vollständiges LegacyAuthorityBoundaryV1 für exakt die angeforderte
    Rollenmenge in einem kurzen konsistenten Read lesen; Schema, kanonische
    Ordnungen, Bijektion, Source-Snapshot- und Ressourcenset-Preimages sowie
    beide Digests vollständig validieren; bei Struktur-, Ordnungs-, Coverage-,
    Auflösungs- oder Digestfehler state1_handoff_binding_invalid, null Wirkung
  je Rolle verlangen:
    current.authority = legacy
    current.authority_version < M;
      andernfalls state1_authority_revision_exhausted, null Wirkung
    current.operation_state = ready
    current.authority_head_key = null AND
      current.authority_pointer_version = null
  Boundary samt vollständigen unveränderlichen Source-Snapshots nur
    prozesslokal an diesen Build binden; keinen Source-/Authority-Lock über
    den Build halten
  bei jeder Erwartungsabweichung: Command ohne Wirkung verwerfen und neu planen

Aufbauphase bei gewähltem Legacy-Authority-Handoff:
  new samt Generation-/Snapshot-/Membermanifest ausschliesslich auf von der
    gebundenen Legacyressourcenmenge disjunkten physischen IDs bauen
  Projektionsbytes ausschliesslich aus den im Preflight gebundenen
    LegacySourceSnapshotPreimageV1-Objekten erzeugen
  jeden STATE1-Engine-Task nach den eigenen Attempt-/Submission-/Taskautomaten
    terminal succeeded und owner-bestätigt abwarten
  new vollständig ready/published machen
  bei Crash oder Verlust des Prozesskontexts vor Precommit new und seine
    Member quarantänisieren; niemals fortsetzen oder adoptieren

Legacy-Authority-Handoff-Precommit-Binding:
  Source-Lock und Authority-Locks der vollständigen Handoffmenge in
    kanonischer Reihenfolge nehmen und bis zum Pointercommit halten
  unter diesen Locks kann keine neue Legacy-Source-Mutation committen; ein
    bereits begonnener Writer gewinnt entweder vor dem finalen Read oder
    verliert nach dem Authority-CAS vollständig
  vollständiges LegacyAuthorityBoundaryV1 erneut lesen; alle Authorityrecords,
    Authorityversionen < M, Source-Revisionen, Source-Snapshot-Digests,
    Ressourcensets und Ressourcenset-Digests bytegleich zum prozesslokalen
    Preflight verlangen; jede gebundene immutable Source-Preimage bleibt
    vollständig auflösbar und digestgültig
  vollständige Abdeckung jeder Handoffrolle durch Generation-, Queryfamily-,
    Member- und Snapshotmanifest beweisen
  Disjunktheit jeder neuen physischen ID von der gebundenen
    Legacyressourcenmenge erneut beweisen
  bei jeder Abweichung Locks freigeben, noch keinen prepared-Record anlegen,
    new samt Membern quarantänisieren und mit neuem Preflight eine vollständige
    Generation auf neuen physischen IDs bauen; kein externer Source-Cutoff/-
    Replay und keine Wiederverwendung
  erst bei vollständiger Gleichheit daraus
    `State1LegacyAuthoritySnapshotV1` erzeugen und gemeinsam mit der
    kanonischen Handoffmenge in einem neuen
    `search_generation_swap(state=prepared)` persistieren
  bis zum Pointercommit die Locks halten; nach `prepared` sind keine
    Source-, Authority- oder Handofffeldänderungen erlaubt
  geht Prozesskontext oder ein Lock verloren:
    prepared nach belegtem Nichtcommit -> aborted und zwingend zum statischen
    Preflight zurückkehren; new samt Membern immer quarantänisieren und eine
    vollständige Generation auf neuen physischen IDs bauen

Gemeinsamer Guard:
  head.pointer_version = V
  new = ready/published und alle Gates bestätigt
  Ressourcen von new Generation und aktuellem new Snapshot = staged
  new.catalog_cutoff = search_catalog_state.committed_revision
  new Snapshot bindet die neueste Geo-Shard-Manifestversion <= diesem Cutoff
  new Snapshot-Queryfamily-Manifest enthält route_radius_ux_v1 auch beim
    dokumentleeren initialen Route-Member und stimmt mit der Generation-Family überein
  new Snapshot-Membermanifest enthält die vollständige Menge aller
    Queryfamilien einschliesslich trail_default
  ausschliesslich seine query_family=route_radius_ux_v1-Teilmenge entspricht
    dem Geo-Shardmanifest bijektiv und enthält jedes Route-Shard-Member mit
    generationenspezifischer physischer ID genau einmal, einschliesslich des
    initialen Shards mit assigned_slots=0
  Generation- und Snapshot-Queryfamily binden dieselbe unveränderte
    search_contract_artifact-Relation samt Key/Hash und validieren deren
    kanonischen ADR-0002-Payload; vorhandene Route-Member kopieren daraus
  die Contract-Artefakt-Ressource ist staged oder issuing, niemals retained,
    gc_eligible, deleting, deleted oder lost
  jedes new Member security_watermark >= search_catalog_state.security_revision
  für jeden relevanten Security-Fence bis zu dieser Securityrevision gilt:
    retired => Retirement-Proof und azyklische Successor-Kette sind vollständig;
               jeder visibility_expanded-Successor bis new.catalog_cutoff ist
               im new Member physisch applied/subsumed;
               ohne späteren Successor gilt je nach Proof die physische
               Restriction, der Clear-Nachfolger oder der neuere Fence
    nicht retired => applied oder query_fenced; bei query_fenced bleibt der
                     Fence im Gateway-Guard des neuen Heads aktiv

Normaler Guard:
  head.state = active
  old = head.active_generation = active/published
  Ressourcen von old Generation und aktuellem old Snapshot = issuing
  handoff_logical_uids und alle handoff_* Felder sind leer;
    andernfalls state1_incremental_authority_handoff_unsupported vor Wirkung

Bootstrap-Guard:
  head.state = empty AND head.active_generation leer AND V = 0

Legacy-Authority-Handoff-Commit-Guard, falls handoff_logical_uids nicht leer:
  reason = bootstrap
  Handoffmenge ist kanonisch, nichtleer und vollständig vom V1-Entity-/
    Querymodell sowie new Generation-/Membermanifest abgedeckt
  `handoff_authority_snapshot` ist ein schema-valides
    `State1LegacyAuthoritySnapshotV1` und deckt bijektiv dieselbe Rollenmenge
    ab
  unter weiterhin gehaltenen Locks stimmen je Rolle current.authority = legacy,
    authority_version < M, authority_version und source_revision exakt mit dem
    Snapshot überein, operation_state = ready und beide Authoritybindings = null
  Source-Snapshot-Digest, Ressourcenset und Ressourcenset-Digest stimmen
    bytegleich mit den jeweiligen Snapshotfeldern überein; die bezeichnete
    Source-Preimage bleibt vollständig auflösbar und digestgültig
  kein nichtterminaler Legacy-Operationszustand liegt vor; die gehaltenen
    Source-/Authority-Locks schliessen bis zum Commit jede neue
    Legacy-Mutation aus
  jede neue Member-Physical-ID ist global eindeutig und von sämtlichen
    gebundenen Legacy-Physical-IDs disjunkt
  kein Legacyindex ist from_generation oder nach dem Commit querybar
  keine nicht genannte Rolle wird in diesem Commit übergeben

Recovery-Guard:
  head.state = active
  old = head.active_generation = active/{updating,recovery,quarantined}
  alter Writer technisch gefenced; new verwendet andere physische IDs
  jede alte Ressource ist vorher als issuing, retained oder terminal lost klassifiziert
  Contract-Artefaktbytes, Relation, Key, Hash und Resource-Owner sind aus dem
    bestätigten Ressourcenmanifest rekonstruierbar und vollständig validiert

Gemeinsamer Commit:
  new.lifecycle_state = active
  Ressourcen von new Generation und aktuellem new Content-Snapshot:
    staged -> issuing
  von new gebundene Contract-Artefakt-Ressource: staged -> issuing;
    bereits issuing bleibt issuing
  normal: old.lifecycle_state = sealed
  recovery: old.lifecycle_state = quarantined
  falls old existiert: seine noch issuing Generation-/Snapshotressourcen:
    issuing -> retained
  bereits retained bleibt retained; terminal lost bleibt unverändert lost
  head.state = active
  head.active_generation = new
  head.pointer_version = V + 1
  falls Legacy-Authority-Handoff:
    jede gebundene externe Authorityzeile per erwartetem authority_version-CAS:
      authority = state1
      authority_head_key = kanonischer (contract, entity_kind)-Key dieses Heads
      authority_pointer_version = V + 1
      authority_version = erwartete Version + 1
  search_generation_swap = committed
```

Der Pointer-/Authority-CAS linearisiert gegen die Fachtransaktion aus Abschnitt
5.1. Gewinnt ein Fachcommand vor der Precommit-Lockaufnahme, ändern
Authorityversion oder Source-Revision die prozesslokale Bindung; Precommit legt
keinen `prepared`-Swap an, quarantänisiert `new` und baut vollständig neu. Nach
erfolgreichem finalem Read kann unter den gehaltenen Locks kein Fachcommand vor
dem Handoff-CAS gewinnen. Gewinnt dieser CAS, verliert ein mit alter Partition
begonnener Fachcommand seinen Authority-CAS, rollt Fachrecord und sämtliche
Controlrecords gemeinsam zurück und partitioniert beim Retry neu. Ein Crash
kann daher weder eine übergebene Rolle nachträglich beim Legacyowner zustellen
noch ihren STATE1-Change auslassen.

Der Swap schliesst eine alte Contract-Artefakt-Ressource nicht pauschal mit
der alten Generation. Bindet `new` dasselbe Artefakt, bleibt dessen Ressource
`issuing`. Bindet `new` ein anderes qualifiziertes Artefakt, darf das alte nur
dann atomar oder später nach `retained` wechseln, wenn keine andere aktive,
querybare, ready-/swap-fähige oder aufholende Bindung daraus neue Kontexte
ausstellen kann; bereits zugesagte Cursor behalten ihren HWM-Pin.

Ein Crash vor dem Commit lässt den bisherigen Headzustand unverändert. Ein
Crash danach sieht den neuen Pointer und den vollständig committed
Auditdatensatz. Es gibt keinen
Zwischenzustand zwischen mehreren Bundlemitgliedern, weil der Gateway alle
physischen IDs ausschliesslich aus dem unveränderlichen Membermanifest des
gepinnten Content-Snapshots liest. Der Pointercommit schaltet damit auch eine
Route-Shard-Familie immer vollständig um; ein shardweiser Head- oder
Engine-Swap ist verboten.

Beim Bootstrap nach dem vollständigen STATE1-Modell ist `from_generation` leer und
`reason = bootstrap`; der bis dahin leere Head ist nicht querybar. Wird dabei
eine bisher vom Legacyowner bediente Rolle übernommen, ist ihr alter physischer Index
keine `from_generation`: Erst der gemeinsame Head-/Authority-Commit beendet die
Legacyautorität, und ihr historischer Übergangsrecord kann sie danach nicht
reaktivieren. Ein Recovery-Swap verwendet
`reason = recovery`. Die beschädigte alte Generation bleibt für alle alten
Cursor stale/fail-closed und wird weder versiegelt weiterbedient noch jemals
reaktiviert. Ihr möglicherweise verspäteter Enginewrite kann die neue
Generation wegen der bereits im Handoff-Guard gebundenen global eindeutigen,
disjunkten physischen IDs nicht verändern.

### 8.4 Versiegelung und Rollback

Eine versiegelte Generation:

- erhält keine normalen Content-, Settings- oder Catch-up-Writes;
- bleibt für bereits ausgestellte Cursor nur mit identischer Epoch und
  aktuellen Query-Fences erreichbar;
- wird sofort blockiert, wenn ein aktueller Fence nicht vor Ranking und Counts
  erzwungen werden kann;
- bleibt bis zum Maximum aus dem HWM ihres `retention_resource` plus Grace,
  Rollbackfenster und sonstiger Artefaktretention erhalten.

Index-/Schema-Rollback ist immer eine neue Generation mit neuer ID und
`intent = rollback`. Sie wird aus autoritativer Projektion und Change-Log bis
zum aktuellen Cutoff aufgebaut und durchläuft dieselben Ready-/Swap-Gates. Ein
Übergang `sealed -> active` sowie das Aufholen der cursorgepinnten
Altgeneration sind in V1 verboten.

Für `route_radius_ux_v1` verwendet auch der Rollback die aktuelle persistente
Shardmanifestversion und jedes bestehende `search_geo_route_assignment`
unverändert. Er löst zusätzlich das für den Rollback ausdrücklich ausgewählte,
qualifizierte `search_contract_artifact` aus dessen kanonischen Payloadbytes
auf und legt damit zuerst die immutable Generation-Queryfamily an, auch beim
leeren Katalog. Jedes Route-Member einschliesslich des initialen leeren Members
kopiert Relation, Key und Hash daraus. Er
erzeugt pro logischem Shard neue physische IDs, darf weder den damaligen Shardstand einer
alten Generation reaktivieren noch live Trails neu sortieren, gelöschte Slots
auffüllen, `geo_shard_id` ändern oder einen Contract nur aus alten Schema-/
Settings-/Bundlehashes erraten.

Ein Code-Rollback darf die aktuelle Generation nur verwenden, wenn Schema,
Settings, Cursorformat, Contract-Artefakt-Key/-Hash und Gatewayreader
kompatibel sind.

## 9. Restriction-Fence-Automat

```text
active -> propagating -> satisfied -> retired
```

- `active` ist ab dem linearisierenden SQLite-Commit querywirksam.
- `propagating` bedeutet, dass physische Updates/Deletes zu allen aktiven,
  ready oder swap-fähigen Generationen geplant sind.
- `satisfied` verlangt für jede relevante Generation entweder `applied` oder
  einen vollständig wirksamen `query_fenced`-Pfad. Neue Kandidaten starten
  blockiert und übernehmen alle aktuellen Fences, bevor sie `ready` werden.
- Vor jeder Kante nach `retired` prüft derselbe SQLite-Schreiblock, dass kein
  nicht-retirter Vorgänger über einen eingehenden
  `clear_superseded_by_restriction`- oder
  `guard_release_superseded_by_restriction`-Link auf diesen Fence angewiesen
  ist. Ein jüngerer Successor-Fence bleibt also querywirksam, bis alle von ihm
  neutralisierten älteren Clear-/Guard-Release-Pfade von alt nach neu retiren
  konnten.
- `retired` unterscheidet danach fünf ausschliessliche Guards.
  `restriction_converged` verlangt bei leerer `clear_revision` und leerer
  `guard_release_revision` einen `exact_target`-Gateway-Guard, den passenden
  autoritativen Restrictionstand und eine unter demselben Schreiblock frisch
  aufgelöste vollständige Multi-Frontier-Menge jedes betroffenen physischen
  Ziels. Kein relevanter Kopf darf `clear_pending`, kein wirksamer Guard
  konservativ sein. In jeder noch querybaren, `ready`, swap-fähigen oder
  aufholenden Generation muss ein physisches `applied` exakt diese aktuelle
  Menge, ihren kumulativen Restrictiondigest, den vollständigen
  Beobachtungsäquivalenzbeleg und den weiterhin letzten geordneten
  Visibilitymarker binden. Fence-Apply-Submissions nach diesem Marker oder ein
  abweichender aktueller Dependency-Closure-/Frontier-Hash lassen den CAS
  verlieren. Nur diese Deliverykontinuität erlaubt, den Queryguard ohne
  Content-Epoch-Wechsel zu entfernen.
  `guard_release_converged` verlangt bei leerer `clear_revision`, gesetzter
  `guard_release_revision` und `conservative_subject_exclusion`, dass eine
  normale neue Content-Epoch mindestens diese Revision publiziert, aktive,
  `ready`, swap-fähige oder aufholende Member den exakten eingeschränkten
  Zielstand `applied/subsumed` haben und jede noch cursorquerybare
  `sealed`-Generation entweder denselben beobachtungsäquivalenten
  Restrictionstand trägt, irreversibel aus der Querymenge entfernt ist oder
  über den retenierten Fencebeleg alle relevanten Cursor mit
  `content_snapshot.delivered_catalog_revision < guard_release_revision`
  ausdrücklich stale macht. Ein Head-Swap allein genügt dafür nicht. Zwischen
  Epoch-Publikation und diesem Retirement blockiert der Issuance-Fence neue
  Kontexte.
  `guard_release_superseded_by_restriction` verlangt stattdessen bei ebenfalls
  leerer `clear_revision`, gesetzter `guard_release_revision` und
  `conservative_subject_exclusion` den einzigen Root-Successor-Link dieser Art
  auf einen strikt späteren Fence. `guard_release_revision`, immutable
  `exact_target_digest`, `visibility_rule_version` und der im Link gebundene
  Multi-Frontier-Closure müssen rekonstruierbar sein; der Linkproof beweist,
  dass der damalige neue Restrictionstand dieses exakte alte Ziel mindestens
  subsumiert. Der aktuelle Closure ist im Retirement-CAS entweder identisch
  oder wird getrennt durch `SuccessorProgressProofV1` gebunden. Eine
  eigene publizierte F1-Zwischenepoch ist nicht erforderlich, wenn eine
  normale geschlossene F2-Content-Epoch mit Cutoff mindestens auf
  `successor_revision` sie koalesziert. **Diese Successor-Epoch ist vor
  Retirement zwingend:** Aktive, `ready`, swap-fähige und aufholende Member
  müssen den kumulativen F2-Zielstand terminal `applied/subsumed` tragen; ein
  nur `query_fenced`-ter F2 genügt nicht. Ist F2 oder seine transitive
  Successor-Kette seit der Linkrevision durch einen späteren Grant oder eine
  weitere Restriction fortgeschritten, tritt stattdessen der oben definierte
  `SuccessorProgressProofV1` mit der normalen geschlossenen aktuellen Epoch
  ein; F2s im Link gebundener Restriction-Guard bleibt wegen der eingehenden
  Abhängigkeit bis nach F1-Retirement wirksam. Noch cursorquerybare versiegelte
  Altgenerationen bleiben durch den retenierten Anker-Stale-Beleg oder einen
  beobachtungsäquivalenten physischen Stand sicher. Der Successor-Fence muss
  unter demselben Schreiblock weiterhin nichtretirt und `satisfied` sein. Der
  Proof bindet Epoch/Snapshot, Deliveries, aktuellen Successor-Frontier und die
  fortbestehende Retirement-Abhängigkeit. Dadurch darf der alte konservative
  Whole-Subject-Guard enden, obwohl sein ursprünglich exakter Zielstand durch
  die stärkere Restriction überholt ist; Epochwechsel und Successor-Guard
  verhindern gemeinsam jedes Öffnungsfenster.
  `clear_converged` verlangt bei gesetzter `clear_revision`, dass der Tail der
  unter demselben Schreiblock erneut gelesenen lückenlosen Successor-Kette ein
  `clear_before_retirement` oder `clear_advance_before_retirement` ist, der
  autoritative Stand exakt diesen letzten Nachfolger trägt und jede
  Grant-Change der Kette in aktiven, `ready`, swap-fähigen oder aufholenden
  Membern terminal `applied` oder durch vollständige physische
  Baseline/Coverage `subsumed` ist. Eine nur für alte Cursor
  querybare `sealed`-Generation darf stattdessen ihre physisch bestätigte alte
  Restriction behalten; eine Sichtbarkeitserweiterung verändert keine alte
  Content-Epoch rückwirkend. In allen Zweigen kann ein Member durch
  irreversible Entfernung aus Query- und Swapmengen ersetzt werden.
- Ein gleichzeitig appendender Grant und der Retirement-CAS werden durch den
  SQLite-Schreiblock geordnet: Gewinnt der Grant, muss der Retirement-Proof den
  neuen Tail vollständig einschliessen; gewinnt Retirement, sieht der Grant
  einen terminalen Fence und läuft je nach Proof-Zweig als
  `grant_after_retirement` oder als normale spätere Content-Revision.
- Ist ein begonnener Clear vor seiner Konvergenz durch einen strikt späteren,
  autorisierten und nachweislich mindestens ebenso engen Fence neutralisiert,
  erlaubt dessen noch nicht retirter, bereits wirksamer Gateway-Guard den
  fünften Guard
  `clear_superseded_by_restriction`. Der Proof bindet den append-only
  Successor-Link und den Subsumptionsbeleg. Vor Retirement muss eine normale
  geschlossene Successor-Content-Epoch mit Cutoff mindestens auf dessen
  `successor_revision` publiziert sein; aktive, `ready`, swap-fähige und
  aufholende Member tragen den kumulativen neuen Restrictionstand terminal
  `applied/subsumed`. Ist die Successor-Frontier inzwischen autorisiert
  weitergelaufen, darf derselbe Guard nur über `SuccessorProgressProofV1` und
  eine normale geschlossene Epoch mindestens bis zum aktuellen transitiven
  Visibilitykopf gewinnen; der statische Successor-Restrictionguard bleibt
  dabei bis zum geordneten Vorgänger-Retirement wirksam. Ein bloss
  `query_fenced`-ter Successor ohne solche Epoch genügt nicht.
  Noch cursorquerybare Altgenerationen bleiben über den retenierten
  `clear_revision`-Stale-Beleg, ihre alte physische Restriction oder
  vollständige Beobachtungsäquivalenz sicher. Zusätzlich bindet der Proof den
  weiterhin nichtretirten `satisfied`-Successor-Guard für jedes relevante
  Member; der alte Clear darf weder Sichtbarkeit öffnen noch umgeschrieben
  werden.
- Derselbe Retirement-CAS setzt den passenden `retirement_proof_kind` und den
  vollständigen rekonstruierbaren Proofhash. Erst dieser Commit beendet den
  alten Gateway-Guard und hält zugleich die historische `fenced`-Delivery als
  Teil der unveränderlichen Security-Prefixkette gültig.
- Ein nur `query_fenced`-tes Ziel genügt niemals für Retirement, solange diese
  Generation noch aktivierbar werden kann. Insbesondere darf ein Gatewayfilter
  keinen Grant freigeben: Bis der Clear-Nachfolger für alle aktuellen
  Generationen physisch bestätigt oder durch einen mindestens ebenso engen
  normal epochpublizierten Successor-Restrictionstand samt weiterhin wirksamem
  Successor-Fence neutralisiert ist, bleibt der alte Fence aktiv.

Ein globaler Security-Watermark beweist nur einen lückenlosen Prefix; er ersetzt
nicht die konkrete Fence-Menge und ihren Scope. Der Gateway prüft aktuelle
Fences und Securityrevision vor und nach jedem vollständigen Suchbatch.
`retired` entfernt nur den laufenden Gateway-Guard; fachliche Wahrheit ist
danach die bei diesem Commit beobachtungsäquivalent bewiesene Restriction, der
epochgebunden vollständig projizierte exakte Zielstand eines zuvor
konservativeren Guards, der vollständig projizierte Clear-Nachfolger oder die
verlinkte stärkere Nachfolgerrestriction. Ein erst danach autorisierter Grant
mutiert diesen terminalen Fence nicht. Nach `restriction_converged` oder
`guard_release_converged` schliesst der erste solche Grant den Übergang über
seinen append-only `grant_after_retirement`-Link; nach
`clear_converged` ist die alte Fence-Kette bereits abgeschlossen und die
Erweiterung eine normale neue Content-Revision. Die monotone Securityrevision wird nie
zurückgesetzt. Ein vom Fence betroffener, zuvor
ausgestellter Cursor bleibt deshalb auch nach Fence-GC zuverlässig stale.

Zwischen der Publikation eines gegenüber dem temporären Gateway-Guard weiter
sichtbaren Zielstands und Retirement gilt ein ausdrückliches Issuance-Fence:
Schneidet das vollständige Treffer-/Aggregationsuniversum einen noch nicht
retirten Fence, ist `clear_revision` oder `guard_release_revision` gesetzt und
hat der aktive Content-Snapshot den jeweils gesetzten Anker bereits erreicht,
darf der Gateway keinen neuen Suchkontext und damit keinen Cursor ausstellen;
er liefert nach dem begrenzten Full-Context-Retry `search_unavailable`. Ein
Snapshot vor diesem Anker kann bestehende Cursor nur unter dem alten Fence
bedienen und wird beim Publish der neuen Content-Epoch normal stale. Der finale
Response-Visibility-CAS prüft diesen Guard erneut. Damit existiert kein Cursor,
der dieselbe Content-Epoch zuerst mit altem Gateway-Fence und nach Retirement
mit erweitertem Universum fortsetzt. Nach `guard_release_converged` oder
`clear_converged` beginnt die erste neue Ausstellung bereits unter der
endgültigen Sichtbarkeit; bei `guard_release_superseded_by_restriction` oder
`clear_superseded_by_restriction` übernimmt der mindestens ebenso enge
Successor-Fence den Guard.

Ein Fortschritt der globalen Securityrevision allein macht einen Cursor nicht
zwingend stale. Bei einer Differenz wertet der Gateway die lückenlose
Security-Change-Historie seit dem gebundenen Watermark gegen Principal,
Source-Scope und Suchuniversum aus. Ein relevanter Change ergibt stale; ein
bewiesen irrelevanter Change erlaubt die Fortsetzung unter allen aktuellen
Fences. Kann Relevanz nicht bewiesen werden, gilt fail-closed stale.

Fence- und Security-Change-Historie bleibt mindestens bis zum bei Retirement
aktuellen `security_history`-HWM plus Resource-Grace erhalten. Damit kann jeder
noch gültige stateless Cursor die relevante Änderung auch nach physischer
Konvergenz und Fence-Retirement erkennen.

Fencefehler dürfen keine Privacy-Mutation zurück auf sichtbar kompensieren.
Unterbelichtung und vorübergehendes `search_unavailable` sind zulässig;
Überbelichtung ist es nicht.

## 10. Gateway-, Cursor- und Retentionsautomat

### 10.1 Stabiler Kontrollstempel

Der interne Kontrollstempel einer Query enthält mindestens:

```text
head.pointer_version
control_plane_incarnation + restore_epoch
publisher.scope_key + state + version + restore_epoch + credential_epoch + term
generation_key + lifecycle_state + epoch_state + content_epoch
Generation-Queryfamily-Manifesthash
catalog_cutoff + Member-Delivery-Watermarks
security_revision + Member-Security-Watermarks + relevanter Fence-Digest
schema_hash + settings_hash + bundle_hash + engine_incarnation
Content-Snapshot-Key + dessen Federation-Manifesthash
Content-Snapshot-Queryfamily-Manifesthash + Route-Queryfamily-ID + Bindinghash
  + Queryfamily-Membermanifesthash
Content-Snapshot-Membermanifesthash + geordneter Route-Shard-Memberdigest
Route-Shard-Manifestrevision + Route-Shard-Manifesthash
Route-Contract-Artefakt-ID + Key + Payloadhash + Payloadschema
Route-Contract-Resource-Ownerpaar + für den Request zulässige Stateklasse
epochgebundene Scope-Revision + Scope-Digest + Expiry-Horizont
aktuelle Scope-Head-Version zur Restriction-/Stale-Erkennung
```

Nur zwei identische, vollständig `published` bestätigte Lesungen vor und nach
Hits, Total, Facetten und Histogrammen dürfen eine Antwort erzeugen.

Mutable Resource-`version` und `issued_valid_until_hwm` gehören bei Contract-
Artefakt, Generation, Snapshot, Key, Format und Security-History ausdrücklich
nicht zum semantischen Pre-/Post-Query-Stempel. Dieser bindet das immutable
Ownerpaar und die für erste Seite beziehungsweise Cursor zulässige
Stateklasse. Eine parallele monotone HWM-Erhöhung ändert die Querysemantik
nicht und darf keinen Retry-Sturm auslösen; Statekanten, Ownerwechsel oder
`lost/gc_eligible/deleting/deleted` bleiben dagegen stempel- beziehungsweise
Issuance-relevant.

`search_publisher_state = quarantined` macht den Gateway unabhängig vom
weiterhin möglicherweise `published`en Generation-Head nicht ready. Alle
anderen Publisherzustände sind für Reads zulässig, ihre Version und ihr
Fence-Tupel bleiben jedoch Bestandteil des Stempels, damit ein Übergang in
Quarantäne keinen bereits berechneten Batch sichtbar werden lässt.

Für die erste Seite gehört `head.pointer_version` zwingend zum stabilen Stempel.
Eine Cursorseite routet dagegen ausschliesslich auf ihre gepinnte Generation.
Ein reiner Head-Swap und die gekoppelte erlaubte Transition dieser Generation
von `active/issuing` nach `sealed/retained` ändern deren Dokumente oder Epoch
nicht. Tritt dies während einer Cursorquery ein, verwirft der Gateway den Batch
höchstens einmal und wiederholt ihn auf derselben gepinnten Altgeneration; der
Pointerwechsel allein ist niemals `search_context_stale`. Epoch, Engine-
Inkarnation, Fence oder Ressourcenverlust bleiben weiterhin relevant.

#### 10.1.1 Logische Route-Shard-Query

Für `route_radius_ux_v1` löst der Gateway die Queryfamilie ausschliesslich aus
`search_content_snapshot_query_family(route_radius_ux_v1)` des gepinnten
Snapshots auf und prüft deren Übereinstimmung mit der Generation-Family. Erst
danach liest er die durch `query_family_member_manifest_hash` gebundene, nach
`query_family = route_radius_ux_v1` gefilterte Teilmenge der Snapshot-Member. Die nach
`shard_ordinal` geordnete Liste enthält für jeden gebundenen logischen Shard
genau eine generationenspezifische physische ID. `trail_default` bleibt im
vollständigen Snapshot-Membermanifest, ist aber kein Geo-Shard. Der Gateway
ergänzt weder ein inzwischen neu registriertes Generationenmember noch einen
Shard aus dem aktuellen mutable Allocator-Head.

Die Snapshot-Queryfamily muss die `search_contract_artifact`-Relation, Key und
Payloadhash binden; alle gebundenen Route-Snapshot-Member müssen diese Werte kopieren
und exakt mit ihren Generationenmembern übereinstimmen. Der Gateway
lädt die kanonischen Payloadbytes, prüft Schema und Hash und leitet daraus
Direct-, Geometrie-, Radius-, Engine-, Count- und Accuracy-Profil ab; nackte
Schema-/Settings-/Bundlehashes oder kompilierte Defaults genügen nicht. Für
eine erste Seite muss die Contract-Ressource `issuing`, für eine Cursorseite
darf sie bei bereits reserviertem HWM auch `retained` sein. Jede fehlende,
widersprüchliche oder `gc_eligible/deleting/deleted/lost`e Bindung lässt den
gesamten Batch fail-closed scheitern.

Jede logische primäre Route-Query wird als eine federierte
Meilisearch-Multi-Search ausgeführt: genau eine Subquery pro gebundenem Shard,
in jeder Subquery identische normalisierte ACL-, Text-, Federation-, Metadaten-
und `_geoRadius`-Prädikate sowie dieselbe Sortierung mit stabilem
`trail_id ASC` als abschliessendem Tie-Breaker. Die Top-Level-Federation
berechnet globale Reihenfolge, zurückgegebene Seite, `totalHits` und die
Grundlage für `next_cursor`. Sie exponiert keine Seiten- oder Gesamtseitenzahl. Ein Abruf
einzelner Shardseiten mit
nachträglichem Gateway-Merge ist verboten, weil er Pagination, Total und
Counts nicht als eine Menge beweist.

Die gebundene Geo-Manifestliste darf nie leer sein. Beim katalogweit leeren
Stand enthält sie genau den persistenten initialen Shard Ordinal `0` mit
`assigned_slots = 0`; Snapshot-Family und Membermanifest binden dazu genau ein
leeres physisches Route-Member. Auch dieser Fall läuft für jede logische
Anfrage durch exakt **eine** federierte Meilisearch-Multi-Search mit genau
**einer** Subquery und allen
normalisierten ACL-, Fence-, Text-, Filter- und Spatialprädikaten. Nur deren
exhaustive Antwort darf `hits = []` und `totalHits = 0` liefern; die
V1-Response setzt dazu `page.returned = 0` und `page.next_cursor = null`. Ein
Gateway-konstruiertes Leerresultat, das Auslassen des Members oder
eine zusätzliche Sentinel-Subquery ist verboten; Context-, Pre-/Post-Stempel-
und HWM-Gates gelten unverändert.

Jede zusätzliche logische Facetten- oder Histogrammquery verwendet denselben
Snapshot, dieselbe Shardliste und denselben Spatial-/Filterkontext; die
disjunktive Gruppenverbreiterung ändert nur den im Suchvertrag erlaubten
Gruppenfilter. Fehlt in der primären Multi-Search eine Shardantwort, ist sie
abgeschnitten, nicht exhaustiv oder gehört sie nicht zum gebundenen
Membermanifest, wird der vollständige kritische Batch verworfen. Bei einer
optionalen Aggregationsgruppe folgt stattdessen ausschliesslich deren
typisierte Gruppenfehlersemantik aus dem Trail-Suchvertrag; sie darf nie aus
den übrigen Shards extrapolieren.

Beim leeren Katalog folgen Facetten und Histogramme demselben Enginepfad. Eine
Ready-Aggregation muss aus der exhaustiven Ein-Subquery-Ausführung und dem
gebundenen Options-/Bucketvertrag alle sichtbaren Optionen beziehungsweise
Histogrammbuckets vollständig mit `0` ausgeben. Kann sie das nicht beweisen,
folgt der typisierte Gruppenfehler ohne Teil-, Schätz-, synthetisch ergänzte
oder extrapolierte Zahlen.

Pre- und Post-Stempel binden Snapshot-Family, Manifestrevision/-hash,
Membermanifesthash, jede physische Memberidentität sowie Contract-Artefakt und
zulässige Resource-Stateklasse. Ein
Unterschied verwirft den ganzen Batch. So
bleiben Treffer, globales Paging und exhaustive Counts exakt für die
tatsächlich berechnete Direct-Menge; nur die räumliche Prädikatsqualität trägt
die in ADR 0002 festgelegte Approximation.

### 10.2 Erste Seite

```text
Schema validieren
-> authentisieren und normalisieren
-> Uhrgesundheit bestätigen
-> Publisher-Gateway-Readiness bestätigen
-> DB-Zeit as_of setzen
-> active/published Generation, Content-Snapshot, dessen immutable
   Queryfamily-/Membermanifeste und Scope-Binding auflösen
-> gebundene Contract-Artefakte aus Snapshot-Family, Relation, Key, Hash und
   Payload validieren, einschliesslich der Kopie im initialen Route-Member
-> relevanten Guard-Release-/Clear-Issuance-Fence ausschliessen
-> Kontrollstempel vor Query lesen
-> vollständigen kritischen Batch ausführen
-> identischen Kontrollstempel nach Query bestätigen
-> alle Resource-HWM reservieren und externen Anti-Rollback-Checkpoint bestätigen
-> finalen lokalen Response-Visibility-CAS ausführen
-> HTTP-Antwort sichtbar machen
```

Bei Stempelwechsel wird der ganze Kontextkandidat verworfen. Ein begrenzter
Full-Context-Retry verwendet neues `as_of`, neue Bindings und keine Teilergebnisse
des ersten Versuchs. Danach folgt `search_unavailable`.

### 10.3 Cursorseite

Die Präzedenz aus dem Trail-Suchvertrag gilt:

```text
strukturelles Schema
-> kid-Ressourcenstatus, MAC und Principalbindung
-> Request-Binding
-> Uhrgesundheit; bei Fehler search_unavailable statt Zeitentscheidung
-> now >= valid_until
-> relevante Epoch-/Fence-/Policy-/Federation-Staleness
-> zugesagte Ressourcenverfügbarkeit
-> Publisher-Gateway-Readiness derselben Control-Plane-Inkarnation
-> Pre-/Query-/Post-Stempel
-> HWM-Reservierung und externe Checkpointbestätigung
-> finaler lokaler Response-Visibility-CAS
-> Antwort
```

Eine Cursorseite wechselt nie auf die neue aktive Generation oder Epoch. Ein
relevanter Stempelwechsel ergibt `search_context_stale`, nicht einen stillen
Retry unter anderer Bedeutung.

Bei einer Route-Query bindet der signierte Cursor zusätzlich
den vollständigen `query_family_manifest_hash` des Content-Snapshots, ID und
`binding_hash` seiner `search_content_snapshot_query_family`, deren
`query_family_member_manifest_hash`, `geo_shard_manifest_hash`, den
vollständigen `member_manifest_hash` seines Content-Snapshots sowie
`contract_artifact_key` und `contract_artifact_hash`. Diese Felder und das
initiale Route-Member sind auch bei null Assignments gesetzt. Seine Fortsetzung liest exakt diese immutable
Shardfamilie und denselben Contractpayload. Eine inzwischen
publizierte Epoch mit neuem Shard oder anderem qualifizierten Engineprofil
erweitert beziehungsweise reinterpretiert den Cursor nicht; je nach normaler
Epoch-/Relevanzregel bleibt er auf dem alten Snapshot ausführbar oder wird
stale, aber er wechselt nie teilweise auf die neue Familie oder das neue
Artefakt.

Bei einem retenierten Fence gelten zusätzlich unabhängig vom aktuellen Head
zwei Ankerregeln. Schneidet das gebundene Treffer-/Aggregationsuniversum den
Fence, ist der Cursor `search_context_stale`, wenn

- `retirement_proof_kind = guard_release_converged` oder
  `guard_release_superseded_by_restriction` und seine
  `delivered_catalog_revision < guard_release_revision` ist; oder
- `retirement_proof_kind = clear_superseded_by_restriction` und seine
  `delivered_catalog_revision < clear_revision` ist.

Die zweite Regel ist nötig, weil der stärkere Successor zwar den Clearstand
subsumiert, aber gegenüber der ursprünglichen temporär weiterwirkenden
Restriction dennoch eine Erweiterung sein kann. Beide Regeln gelten
ausdrücklich auch für eine weiterhin physisch erreichbare
`sealed`-Altgeneration und bei bereits im Cursor gebundenem
Successor-Securitystand; weder Pointerwechsel noch unveränderte Securityrevision
dürfen sie umgehen. Nur ein typisierter Irrelevanz- oder vollständiger
Beobachtungsäquivalenzbeleg erlaubt die Fortsetzung.

Innerhalb der Integritätsstufe gilt ein enger Sonderfall: Ist `kid` syntaktisch
valide und als von dieser Control-Plane-Inkarnation ausgestellt bekannt, seine
Ressource aber entgegen einem noch laufenden HWM `lost`, folgt
`search_context_unavailable`, obwohl die MAC ohne Secret nicht mehr geprüft
werden kann. Ein unbekanntes, nie ausgestelltes oder manipuliertes `kid` sowie
jede MAC-Abweichung bleibt `invalid_cursor`. So wird ein gebrochenes
Retentionsversprechen nicht als Clientfehler fehlklassifiziert.

### 10.4 `valid_until` und Issuance

V1 legt keinen Datensatz pro Cursor an. Der Cursor bleibt gemäss ADR 0001
selbstenthaltend; persistent sind nur Keyring-Metadaten und monotone
High-Watermarks der gemeinsam verwendeten Ressourcen.

```text
valid_until = min(
  as_of + context_max_ttl,
  search_content_snapshot_scope.earliest_discoverable_until,  # nur Federation
  explizite frühere Vertragsgrenze
)
```

Bei Federation muss das Scope-Binding zum gepinnten Content-Snapshot gehören,
sein Digest im Federationmanifest enthalten sein und sein Horizont entweder
leer oder strikt nach `as_of` liegen. Der aktuelle mutable Head darf die Grenze
nur durch einen relevanten Fence beziehungsweise Stale-Entscheid verengen, nie
verlängern. Ist die epochgebundene früheste Expiry bereits erreicht, bleibt die
Epoch für neue föderierte Kontexte unbenutzbar, bis ein neuer vollständig
publizierter Content-Snapshot den abgelaufenen Bestand verarbeitet hat.

Vor der Berechnung prüft der Gateway den typisierten Minimum-Beleg des
Scope-Bindings erneut: Das Feld ist leer genau bei leerer epochgebundener
Membership, andernfalls gesetzt, identisch zur referenzierten Scope-Version
und exakt das kleinste im Scope-Digest gebundene `discoverable_until`. Ein
fehlendes oder zu spätes Minimum ist `invariant_violation` und liefert
fail-closed `search_unavailable`; es darf weder als „keine Grenze“ noch durch
den aktuellen Head repariert werden. Wegen der halboffenen Zeitintervalle ist
`response_now = earliest_discoverable_until` bereits abgelaufen.

Vor Ausgabe eines Cursors erhöht eine einzige PocketBase-Transaktion
`issued_valid_until_hwm` jeder benötigten Generation, jedes Content-Snapshots,
Cursor-Keys, Cursorformats, Vertragsartefakts und der Security-History auf
mindestens `valid_until`. Ändert sich dadurch ein HWM oder Manifestzustand,
erhöht sie im selben Commit `checkpoint_sequence` und erzeugt den vollständigen
`search_control_plane_checkpoint`; andernfalls bindet sie per CAS-Prüfung den
bereits vorhandenen, diese Zusagen enthaltenden Kandidaten. Der Request merkt
sich Sequenz, Securityrevision und seine Ressourcen-/HWM-Teilmenge. Erst nach
Commit darf die Response vorbereitet werden. Vor ihrer Sichtbarkeit muss die
in Abschnitt 10.6 definierte monotone externe Bestätigung genau diese
Anforderung nachweislich abdecken.

Für `route_radius_ux_v1` ist „jedes Vertragsartefakt“ exakt das von
`search_content_snapshot_query_family(route_radius_ux_v1)` gebundene Artefakt,
auch wenn ihr verpflichtetes initiales Member dokumentleer ist. Relation,
`artifact_key`, `payload_hash`, Resource-Kind/-Key und der auf mindestens
`valid_until` erhöhte `issued_valid_until_hwm` gehen gemeinsam in die
Requestteilmenge und das externe Checkpointmanifest ein. Ein HWM nur auf
Generation oder Snapshot pinnt den Contractpayload nicht implizit.

Die externe Bestätigung allein gibt die Response noch nicht frei. Danach führt
der Request eine letzte kurze PocketBase-Transaktion als
Response-Visibility-CAS aus. Sie prüft erneut:

- eine in dieser finalen Transaktion frisch bestätigte Uhrgesundheit sowie die
  einmal daraus gelesene DB-Zeit `response_now < valid_until`; Zeit aus dem
  Querybeginn oder von vor der externen Bestätigung genügt nicht;
- Restorezustand `healthy`, unveränderte Restore-/Control-Plane-Identität und
  einen externen Anchor, dessen Securityrevision mindestens der jetzt lokalen
  entspricht;
- Publisherzustand nicht `quarantined` sowie unveränderte
  Publisher-Version und unverändertes Restore-/Credential-/Term-Tupel aus dem
  Kontrollstempel;
- den vollständigen semantischen Kontrollstempel mit den in Abschnitt 10.1
  ausdrücklich erlaubten Pointer-only-Ausnahmen für Cursorseiten und ohne
  mutable Resource-Version/HWM;
- keinen relevanten, noch nicht retirten Fence mit gesetzter `clear_revision`
  oder `guard_release_revision`, dessen gesetzter Anker im gebundenen
  Content-Snapshot bereits enthalten ist;
- für jede gebundene Ressource dieselbe immutable Ownerrelation und dasselbe
  `(resource_kind, resource_key)`, einen HWM mindestens bei `valid_until` und
  einen für diesen Request zulässigen Zustand, niemals
  `gc_eligible/deleting/deleted/lost`;
- falls die lokale `checkpoint_sequence` inzwischen über der extern für den
  Request bestätigten Sequenz liegt, zusätzlich das neueste lokale
  Kandidatenmanifest: Es muss alle Requestressourcen weiterhin mit mindestens
  ihrem HWM und ohne Verlustzustand enthalten.

Gewinnen alle Guards, erhöht dieselbe Transaktion
`response_guard_sequence` per CAS und bildet den lokalen Linearisierungspunkt
der Response. Verliert ein Guard, werden die berechneten Bytes verworfen; je
nach Ursache folgt der erlaubte Full-Context-Retry, `search_context_stale` oder
`search_context_unavailable`. Ist `response_now >= valid_until`, beginnt eine
erste Seite einen erlaubten vollständigen Neukontext-Retry und liefert nach
dessen Grenze `search_unavailable`; eine Cursorseite folgt ihrer Präzedenz und
liefert `search_context_expired`. Insbesondere kann ein vor dem finalen CAS bereits
committeter Resource-`lost`-Übergang nicht durch einen zuvor extern bestätigten
älteren Manifeststand überholt werden. Der Visibility-CAS ändert selbst keine
HWM-Zusage und benötigt deshalb keinen weiteren externen Checkpoint.

Für einen neuen Kontext müssen alle Ressourcen `issuing` sein. HWM-Reservation
und finaler CAS verlangen dasselbe immutable Ownerpaar und
`current_hwm >= valid_until`; eine parallele weitergehende HWM-Erhöhung ist
zulässig. Der finale CAS vergleicht keine alte Resource-Version und verlangt
keine exakte HWM-Gleichheit. Verliert er gegen Pointer-, Epoch-, Fence-,
`issuing -> retained`-, Loss- oder GC-Fortschritt, wird die gesamte Antwort
verworfen. Committet die Reservation zuerst, muss ein konkurrierender GC-CAS
den erhöhten HWM sehen und abbrechen. Ein Crash nach der Reservierung
verursacht deshalb nur Überretention. Dieselbe Self-HWM-Regel gilt für
Generation, Snapshot, Contract-Artefakt, Cursor-Key/-Format und
Security-History.

Eine Cursorfortsetzung darf eine inzwischen im Zustand `retained` befindliche
Ressource weiter verwenden, wenn ihr gebundenes `valid_until` den bereits
reservierten HWM nicht überschreitet. Sie darf den HWM einer geschlossenen
Ressource weder erhöhen noch einen neuen Kontext daran binden.

Die GC-Grenze einer Ressource ist mindestens:

```text
issued_valid_until_hwm + retention_grace_s
```

und zusätzlich das Maximum aller Rollback-, Audit- und Build-Pins. Ist eine
noch zugesagte Ressource irreversibel verloren, trägt sie `state = lost` und
führt zu `search_context_unavailable`; sie wird nicht als abgelaufen
fehlklassifiziert.

Die Retention einer Content-Snapshot-Ressource umfasst stets ihre vollständigen
`search_content_snapshot_query_family`-, Member- und Scope-Records, die gebundene
`search_geo_shard_manifest_version` sowie alle zur Rekonstruktion ihrer
Assignment-Digests benötigten `search_geo_route_assignment`-Zeilen. Die
Generationressource hält zusätzlich alle Generation-Queryfamilies und jedes
von einem solchen Snapshot adressierte physische Member. Bei der Route-Family
hält die Snapshotressource auch beim dokumentleeren initialen Member die
Family-Authority sowie deren Relation, Key und Hashbindung zum
`search_contract_artifact`; dessen
kanonische Payloadbytes und Ownerrecord werden separat durch die eigene, vom
Cursor-HWM erhöhte Contract-Ressource reteniert. Ein abgelaufener neuerer
Snapshot darf daher keinen älteren, noch zugesagten Cursor seiner Shardfamilie
oder seines Contractprofils berauben.

Ein `retirement_artifact_manifest` pinnt dabei nur die für den terminalen
Fencebeleg erforderlichen PocketBase-Metadaten und Proofartefakte. Es hält
einen alten physischen Engine-Index nicht über dessen eigene Cursor-,
Rollback- oder Buildzusage hinaus am Leben. Der Ressourcen-GC darf seinen
externen Payload entfernen und `state = deleted` setzen, aber weder die
gepinnten Owner-/Snapshot-/Deliveryrecords noch ihre
`artifact_proof_projection` löschen oder umschreiben. Normale mutable
Lifecycle-/Versionsübergänge ausserhalb dieser unveränderlichen As-of-
Projektion bleiben erlaubt. Fence-History-GC und Metadaten-GC lesen ausschliesslich die
normalisierten `search_fence_retirement_artifact_pin`-Zeilen über ihren
Reverse-Index und unter demselben SQLite-Schreiblock; ein Scan des JSON-
Manifests ist kein zulässiger Abwesenheitsbeweis.

Wenn keine neuen Kontexte mehr gebunden werden dürfen, wechselt die Ressource
per CAS von `issuing` auf `retained`. Erst nach HWM plus Grace und allen
zusätzlichen Gates folgt `gc_eligible`. GC wechselt sie nach erneutem CAS aller
Eligibility-Gates auf `deleting`. Erst danach darf eine externe Index- oder
Secretlöschung beginnen. Ein Crash in `deleting` ist idempotent wiederaufnehmbar;
`deleted` wird erst nach bestätigtem physischem Abschluss gesetzt.

Der Epoch-CAS einer aktiven Generation legt die Ressource des neuen Content-
Snapshots zunächst `staged` an, öffnet sie im selben Commit nach `issuing` und
schliesst die des vorigen Snapshots nach `retained`. Bei einem noch nicht
aktiven Kandidaten bleiben Generation und erster Snapshot `staged`; erst der
Generation-Swap öffnet beide und schliesst die alten Ressourcen im selben
Pointercommit. Der Cursor-Key-CAS koppelt `search_cursor_key staged -> issuing`
mit `retention_resource staged -> issuing`; `issuing -> verify_only` koppelt er
mit `retention_resource issuing -> retained`. Cursorformat und
Vertragsartefakte folgen demselben Expand-/Contract-Muster. Beim Route-
Contract validiert der Öffnungs-CAS zusätzlich den vollständigen kanonischen
Payload, alle bestehenden Memberbindungen des Kandidaten und die
Readerkompatibilität; erst Head-Swap beziehungsweise Aktivierung öffnet
`staged -> issuing`. Diese gekoppelten Transitionen dürfen nicht durch einen
späteren Best-effort-Hook nachgezogen werden.

### 10.5 Gateway-Cutover

| Zustand | Bedeutung | Erlaubte nächste Transition |
| --- | --- | --- |
| `legacy` | heutige direkte Search-/Tenant-Pfade; neue Garantien nicht behauptet | `shadow` |
| `shadow` | neuer Gateway baut und vergleicht, ist aber nicht autoritativ | `gateway_authoritative` oder zurück zu `legacy` |
| `gateway_authoritative` | V1-Endpunkt nutzt Gateway; verbleibende Legacyleser sind bereits mindestens durch `SEC-VIS` und aktuelle Restrictions gleichwertig eingeengt, erhalten keine verbreiterte Capability und werden noch migriert | `gateway_only` oder zurück zu `shadow` |
| `gateway_only` | alle Search-V1-Leser und Writer gehen durch Control Plane/Gateway | kein stiller Rückfall |

`gateway_only` verlangt als Guard:

- `SEC-SCOPED = ready` mit gebundenem Proofhash; `SEC-GLOBAL` ist nicht aktiv
  und sein blosses Deaktivieren zählt nicht als positives Gate;
- der gebundene `federation_legacy_migration`-Lauf ist terminal `complete`, bis
  zum aktuellen Federationstand aufgeholt und besitzt null unklassifizierte
  Records in jedem freizugebenden Scope;
- Inbound-Containment und immutable Object-Authority sind für alle Ingresspfade
  aktiv; globales Federation-Serving nimmt nur `public_verified`-Objekte mit
  aktiver, belegter Publication-Membership Pn auf;
- durable Inbox und Outbox sind autoritativ; kein Fire-and-forget-Pfad kann
  eine zugesagte Publication oder Rücknahme umgehen;
- alle direkten Meilisearch-Writer durch Publisher ersetzt;
- alle V1-Leser und internen Suchconsumer auf generationengebundene Gateway-
  Dienste migriert;
- alte Search-/Tenant-Keys und noch gültige alte Tokens widerrufen;
- Meilisearch nicht aus Browsernetz beziehungsweise öffentlichem Netz
  erreichbar;
- Negativtest auf verbotene Attribute, direkte Query und Fence-Umgehung
  bestanden.

Nur `gateway_only` **zusammen mit** den obigen Security-, Migrations- und
Publication-Gates darf die vollständigen V1-Fence- und DTO-Garantien als
produktiv freigegeben behaupten. Shadowarbeit, interne Analysen und nicht
ausstellbare Capabilities bleiben vor diesem Gate ausdrücklich zulässig.

### 10.6 Restore- und Anti-Rollback-Automat

```text
healthy -> recovery_required -> replaying -> validating -> healthy
recovery_required -> quarantined
replaying         -> quarantined
validating        -> quarantined
quarantined -- auditierter neuer Recoverybeleg --> recovery_required
```

Es gibt keine direkte Kante `quarantined -> healthy`. Ein neuer Versuch aus
Quarantäne verlangt einen auditierten Admin-Command, weiterhin bestätigtes
technisches Writer-Fencing und neue beziehungsweise reparierte, unveränderlich
identifizierte Restore-/Replayinputs; er beginnt wieder vollständig bei
`recovery_required`. Jeder Digest-, Security-History-, Ressourcenmanifest-,
Engine-Marker- oder Validierungsbruch in einem der drei nichtgesunden
Arbeitszustände führt per CAS nach `quarantined`.

Der externe Anchor ist ein einzelner lineariserbarer Record aus
`checkpoint_sequence`, Restore-Epoch, Inkarnation, Securityrevision,
Cursor-HWM, Manifestreferenz und Manifestdigest. Seine Sequenz steigt global
über normale Laufzeit und Restores hinweg. Eine Bestätigung verwendet folgendes
Protokoll:

1. Das vollständige Manifest eines durablen lokalen Kandidaten `C` wird zuerst
   inline vorbereitet oder unter seiner inhaltsadressierten Referenz
   unveränderlich off-host gespeichert und zurückgelesen verifiziert.
2. Der Bestätiger liest Anchor `A` samt dessen Speicher-Version. Gilt
   `A.checkpoint_sequence < C.checkpoint_sequence`, ersetzt er `A` nur mit
   einem Compare-and-swap auf exakt gelesene Sequenz, Speicher-Version und
   Restore-Identität. Bei verlorenem CAS liest er neu; ein kleinerer oder
   gleich alter Writer darf den Anchor niemals überschreiben.
3. Liegt der gelesene Anchor bereits bei einer Sequenz mindestens der vom
   Request benötigten, gilt das nicht automatisch als Erfolg. Restore-Epoch und
   Inkarnation müssen passen, Securityrevision und globaler HWM müssen
   mindestens die Requestwerte erreichen, und das verifizierte kanonische
   Manifest muss jede vom Request gebundene Ressource mit mindestens ihrem
   zugesagten HWM enthalten. Eine noch auszuliefernde Response darf insbesondere
   keine `lost/deleted`-Ressource referenzieren.
4. Erst nach diesem Einschlussbeweis darf der lokale
   `confirmed_checkpoint_sequence` per Max-CAS nachgezogen, der passende
   Kandidat `confirmed` und ältere Kandidaten `superseded` markiert werden.
   Verletzt ein neuerer Anchor die erwartete Subsumption, wird nicht geraten:
   Readiness und betroffener Request gehen fail-closed in Recovery/Quarantäne.

Dieses Protokoll bestätigt die externe Retentionszusage, ersetzt aber nie den
anschliessenden lokalen Response-Visibility-CAS aus Abschnitt 10.4.

Damit können zwei SQLite-Transaktionen denselben Zeit-HWM, aber verschiedene
Ressourcenmengen reservieren: Die höhere Sequenz enthält wegen der seriellen
Commits beide noch gültigen Zusagen. Gewinnt sie den externen CAS zuerst, kann
der ältere Request sie nur nach explizitem Manifest-Einschluss als Bestätigung
verwenden. Ein verspäteter Writer der kleineren Sequenz kann den Anchor nicht
zurücksetzen. Ist die Zusage inzwischen abgelaufen, wird die wartende Response
verworfen statt nachträglich ausgeliefert.

Das unterstützte Restorewerkzeug setzt zuerst einen extern bestätigten
Maintenance-/Writer-Fence, blockiert neue Responses und widerruft den alten
Engine-Writecredentialpfad. Erst nach diesem technischen Beleg erhöht es
`restore_epoch` und die globale `checkpoint_sequence` per externem CAS und
ersetzt danach die PocketBase-Datei. Jeder Prozess vergleicht beim Start Sequenz,
Restore-Epoch, Securityrevision, Cursor-HWM und Ressourcenmanifest mit dem
externen Checkpoint. Jede nicht als sicher monoton erklärbare Abweichung oder
unbekannte Inkarnation setzt `recovery_required`, bevor ein Search-/Publisher-/
GC-Pfad ready wird. Liegt nur der lokale Bestätigungscache wegen eines Crashes
hinter einem ansonsten identischen externen Anchor, darf er nach vollständigem
Einschluss- und Digestbeweis nachgezogen werden. Liegt die fachliche DB hinter
der externen Marke, gilt der vollständige Restorepfad.

Recovery verfährt fail-closed:

1. Search-Gateway, Publisher und GC bleiben blockiert; der bereits gesetzte
   technische Credential-/Netzfence wird erneut bestätigt. Neue
   Enginecredentials entstehen erst unter der neuen Restore-/Credential-Epoche.
   Recovery mintet jetzt eine nie verwendete `control_plane_incarnation`; alle
   folgenden Replaycommits gehören bereits zu ihr, bleiben aber noch unready.
2. Das extern gebundene kanonische Ressourcenmanifest wird vollständig geladen
   und gegen Referenz sowie Digest geprüft. Alle vorhandenen Ressourcen werden
   mindestens bis zum Maximum aus ihrem Manifest-HWM, dem externen globalen
   Cursor-HWM und `restore_time + context_max_ttl`, jeweils plus Grace,
   konservativ reteniert. Im Manifest zugesagte, aber nicht mehr auffindbare
   Ressourcen werden als minimale `search_context_resource`-Platzhalter mit
   demselben `(resource_kind, resource_key)`, HWM, Grace und terminalem `lost`
   rekonstruiert. Ist der Manifestinhalt nicht verfügbar oder nicht
   verifizierbar, bleibt die gesamte Suche
   `quarantined`.
   Für jede zugesagte Contract-Ressource müssen zusätzlich der eindeutige
   `search_contract_artifact`-Owner, seine kanonischen Payloadbytes und deren
   Key-/Hashbindung rekonstruierbar sein. Ein Ressourcenplatzhalter ohne
   Artefaktpayload beweist nur den Verlust und darf keinen Cursor oder Rebuild
   autorisieren. Jede zugesagte Route-Snapshotressource muss ausserdem ihre
   `search_content_snapshot_query_family` samt Memberteilmengenhash und das
   initiale Route-Member rekonstruieren; null Assignments erlauben weder das
   Weglassen der Family-Zeile noch des Members.
3. Securityereignisse, Fences und Control-Plane-Commits werden aus
   Point-in-time-/Off-host-Backup bis mindestens zur extern bestätigten
   Securityrevision replayt. Ist dieser Beleg nicht möglich, bleibt der Zustand
   `quarantined`; ein älterer sichtbarer Stand darf nicht freigegeben werden.
   Der neue inkarnationslokale `committed_revision` liegt danach mindestens bei
   der globalen `security_revision`.
4. Ein PocketBase-Restore verwendet in V1 **immer** eine neue logische
   Engine-Inkarnation beziehungsweise einen isolierten neuen Engine-Namespace,
   eine neue Generation und global noch nie verwendete physische IDs. Der
   Wiederaufbau liest nur die autoritative, replayte PocketBase-Projektion. Bei
   `route_radius_ux_v1` validiert er vorher Allocator, lückenlose Shardordinals,
   den stets vorhandenen initialen Shard Ordinal `0`, die auf den exakt
   grössten offenen Shard zeigende `current_open_shard`-Relation, jeden nie
   wiederverwendeten Slot, Assignment-Digests und alle niemals leeren immutable
   Manifestversionen. Ausserdem validiert er das ausgewählte immutable
   Contract-Artefakt aus Relation, Key, Payloadhash, geschlossenem ADR-0002-
   Payload und 1:1-Resource-Owner. Recovery erzeugt zuerst die immutable
   Generation-Queryfamily und später für jede Epoch die Snapshot-Queryfamily,
   auch wenn der Katalog leer ist. Sie erzeugt für den im niemals leeren
   Manifest gebundenen initialen Shard genau ein dokumentleeres physisches
   Route-Member; Route-Member und Snapshot-Member der
   Recovery-Generation kopieren genau daraus; die Bindung darf weder aus der
   laufenden Binary noch aus Schema-/Settings-/Bundlehashes rekonstruiert
   werden. Ein vorhandener Trail ohne Assignment, eine doppelte Slotbelegung,
   ein fehlendes Contract-Artefakt oder ein Digest-/Boundarywiderspruch führt nach
   `quarantined`; Restore oder normaler Start dürfen die live Trails nicht neu
   sortieren, um eine Zuordnung zu erraten. Die Recovery-Generation übernimmt
   dieselben logischen `geo_shard_id` und erzeugt dafür neue physische Member.
   Danach durchläuft sie alle normalen Task-, Coverage-, Marker- und
   Swap-Gates. Keine
   alte physische Generation wird aufgrund eines momentanen Marker-Reads
   reaktiviert oder als Buildquelle behauptet: Ein vor dem Restore angenommener,
   im zurückgespielten Submissionlog aber fehlender Task könnte sonst erst nach
   der Validierung wirken. Der Credential-/Netzfence verhindert neue alte
   Submits; die getrennten IDs verhindern zusätzlich, dass bereits angenommene
   verzögerte Tasks die Recovery-Generation erreichen. Kann die Engine diese
   Adressraumisolation nicht garantieren, bleibt Recovery `quarantined`.
   Alte IDs bleiben ebenfalls quarantänisiert und dürfen erst nach einem vom
   Adapter bewiesenen, **alle möglicherweise angenommenen Pre-Restore-Submits**
   umfassenden geordneten Drain/Barrier oder nach bestätigtem vollständigem
   Engine-Namespace-Reset gelöscht werden. Ohne diesen Beleg entsteht sichere
   Überretention statt geratenem GC.
5. Nach vollständiger Invariantenprüfung werden die neue
   `control_plane_incarnation`, das neue Publisher-Fence-Tupel und ein neuer
   lokaler Kandidat mit einer Sequenz oberhalb des externen Anchors gemeinsam
   als Recoveryabschluss gebunden. Dessen vollständiges
   Ressourcenmanifest wird mit dem obigen CAS bestätigt; erst dann folgt
   `healthy`.

Nach jedem securityrelevanten DB-Commit muss dessen im selben Commit erzeugter
Checkpointkandidat extern mindestens dessen Sequenz und Securityrevision
bestätigen, bevor der auslösende synchrone API-Command Erfolg meldet oder die
Instanz weiter ready bleibt. Bei einer bereits mit `202`
angenommenen Inbox ist der interne Commit nicht rücknehmbar; bis zur externen
Bestätigung bleibt der Gateway nicht ready. Nach jeder Cursor-HWM-Reservierung
muss der Checkpoint analog Sequenz, HWM und Ressourcenmanifest bestätigen,
bevor die Search-Response sichtbar wird. Ein Crash oder Checkpointfehler zwischen DB-
Commit und externer Bestätigung führt nur zu einem nicht bestätigten Clientretry
beziehungsweise fail-closed Readiness und sicherer Überretention. Weder
Global verankerte Sequenz, Securityrevision und HWM dürfen durch Restore nicht
kleiner werden; für die übrigen Revisionen gelten die expliziten
Identitätsscopes aus Abschnitt 3.1.

## 11. Fehlerabbildung

Der öffentliche Problem-Details-Vertrag und seine Präzedenz werden nicht hier
dupliziert, sondern wie folgt aus dem Automatenzustand abgeleitet:

| Interner Zustand | Erste Seite | Cursorseite |
| --- | --- | --- |
| Head `empty` oder keine querybare aktive Generation | nach begrenztem Retry `503 search_unavailable` | aktueller Head ist unerheblich: ausschliesslich gepinnte Generation/Epoch/Resource prüfen; bei intaktem `sealed/published` normal antworten, sonst deren spezifischen Stale-/Unavailable-Zustand verwenden |
| Generation-/Snapshot-Queryfamily, das niemals leere Route-Shard-Manifest, die dazu bijektive Route-Teilmenge, Contract-Artefaktbindung/-payload, Snapshot-Membermenge oder deren Digest ist unvollständig/widersprüchlich | `503 search_unavailable` und Control Plane fail-closed; bei null Assignments bleiben initialer Shard und genau ein leeres Route-Member verpflichtend | bei irreversibel verlorener zugesagter Family, Manifest, Member- oder Contract-Artefaktbindung `503 search_context_unavailable`, sonst `503 search_unavailable`, jeweils nach höherrangiger Integritäts-, Binding-, Expiry- und Stale-Prüfung |
| Restorezustand nicht `healthy` | `503 search_unavailable` | bei Control-Plane-Inkarnationsbruch `503 search_context_unavailable`, bei derselben Inkarnation `503 search_unavailable`, jeweils erst nach höherrangiger Integritäts-, Binding-, Expiry- und Stale-Prüfung |
| Publisherzustand `quarantined` | `503 search_unavailable` | bei derselben Control-Plane-/Engine-Inkarnation `503 search_unavailable`, jeweils nach höherrangiger Integritäts-, Binding-, Expiry-, Stale- und bewiesener Ressourcenverlust-Prüfung; ein zugleich bewiesener Inkarnations- oder Retentionsverlust verwendet dessen spezifischen Fehler |
| lokale Uhrgesundheit ausserhalb des zugesagten Bounds | `503 search_unavailable` | nach Struktur-, Integritäts- und Request-Binding-Prüfung `503 search_unavailable`; unzuverlässige Zeit wird weder als Expiry noch zur Fortsetzung verwendet |
| externer Checkpoint transient nicht lesbar/bestätigbar oder Response-Visibility-CAS verliert ohne Stale-/Loss-Beleg wiederholt | nach begrenztem Full-Context-Retry `503 search_unavailable` | `503 search_unavailable`; ein bewiesener Ressourcenverlust bleibt `503 search_context_unavailable` |
| Engine transient nicht erreichbar, gepinnte Generation aber unverändert `published` | `503 search_unavailable` | `503 search_unavailable` |
| relevanter Clear oder gegenüber dem temporären Gateway-Guard weiter sichtbarer Zielstand ist in der Content-Epoch publiziert, sein Fence aber noch nicht `retired` | nach begrenztem Full-Context-Retry `503 search_unavailable`; kein Kontext und kein Cursor werden ausgestellt | ein vor dieser Epoch ausgestellter Cursor ist `409 search_context_stale`; ein dennoch nachgewiesener Same-Epoch-Cursor verletzt die Issuance-Invariante und ergibt `503 search_context_unavailable` |
| relevanter Fence ist `guard_release_converged` oder `guard_release_superseded_by_restriction`, Cursor-Snapshot liegt vor dessen `guard_release_revision` und besitzt keinen Beobachtungsäquivalenz-/Irrelevanzbeleg | nicht anwendbar; erste Seiten verwenden nur den aktiven Stand | `409 search_context_stale`, auch wenn die gepinnte versiegelte Generation physisch noch erreichbar ist |
| relevanter Fence ist `clear_superseded_by_restriction`, Cursor-Snapshot liegt vor dessen `clear_revision` und besitzt keinen Beobachtungsäquivalenz-/Irrelevanzbeleg | nicht anwendbar; erste Seiten verwenden nur den aktiven Stand | `409 search_context_stale`, auch bei bereits gebundenem Successor-Securitystand und unveränderter Content-Epoch |
| gepinnte Generation `updating/recovery/quarantined`, Epoch abweichend oder Fence/Policy/Federation relevant verändert | neuer Full-Context-Retry, danach gegebenenfalls `503 search_unavailable` | `409 search_context_stale` |
| nur aktiver Head-Pointer gewechselt, gepinnte Altgeneration weiterhin `sealed/published` | neuer Full-Context-Retry | einmaliger Retry derselben Altgeneration, danach normale Antwort |
| `now >= valid_until` | nicht anwendbar | `410 search_context_expired` |
| zugesagte, noch gültige Generation, bekannter Key oder Artefakt `lost` | `503 search_unavailable` bei neuem Kontext | `503 search_context_unavailable`, einschliesslich bekanntem `kid` vor MAC-Prüfung |
| unbekanntes `kid`, MAC oder Principalbindung ungültig | nicht anwendbar | `400 invalid_cursor` |
| Request-Binding weicht ab | nicht anwendbar | `409 cursor_request_mismatch` |
| kritisches Batchbudget abgelaufen | `504 search_timeout` | `504 search_timeout`, sofern nicht vorher stale/expired |

Interne Worker-, Publisher- und Adminfehler gehören zu stabilen Klassen:

- `transition_conflict`: CAS-/Lease-/Pointerguard verloren; idempotent neu lesen;
- `retryable_dependency`: Netzwerk, `429`, `5xx`, noch laufender Enginetask;
- `terminal_input`: ungültige Signatur, Authority, Payload oder Contract;
- `invariant_violation`: Digestkonflikt, Revisionslücke, unmöglicher Zustand;
- `engine_state_unknown`: Task-/Markerstand nicht beweisbar;
- `resource_lost`: zugesagtes Artefakt irreversibel verloren.

Nur sichere Codes und Digests werden persistiert oder ausgeliefert. Engine-
Filter, Taskpayload, Stacktrace, Cursorinhalt, Actor-/Tenantgeheimnisse und
interne Indexnamen erscheinen nie in öffentlichen Fehlern.

Für die Federation-Inbox gilt:

- harte Body-/Medientyp-/JSON-Grenzen, Instanz-/Origin-Rate-Limits und sichere
  Transportvalidierung dürfen vor durablem Commit ablehnen;
- `202` erst nach durablem, idempotentem `federation_ingest_event`-Commit samt
  bytegenauer signierter Eingabe, Verification-Envelope und Transport-
  Idempotency-Key;
- idempotenter Replay kann `200` oder `202` mit derselben sicheren Semantik
  liefern und erzeugt keine neue Revision;
- permanente Syntax-/Signatur-/Authorityfehler liefern nur dann eine passende
  sichere 4xx-Antwort, wenn sie vor der dauerhaften Annahme synchron feststehen;
  nach einem bereits gelieferten `202` wechseln sie den Inboxrecord intern nach
  `rejected`, ohne eine zweite HTTP-Antwort zu erfinden;
- `429/503` nur, solange die Eingabe nicht bereits dauerhaft zur Verarbeitung
  angenommen wurde; nach Annahme ist Retry interne Verantwortung.

Ein terminal `accepted`er scoped/direct Effekt ist ein normaler Erfolg, ohne
Global-Membership offenzulegen. Ein nach `202` intern `rejected`er oder
`quarantined`er Record erzeugt keine zweite HTTP-Antwort. Der Replay-Ack eines
terminalen Records bleibt statusneutral und verrät weder Payload,
Migrationsergebnis, Authoritydetail noch Sichtbarkeit. Ein vor durablem Commit
erkannter unbekannter Activitytyp ist sicherer 4xx; wird er erst danach
klassifiziert, endet der Inboxrecord intern `rejected`.

## 12. Crash- und Recovery-Matrix

| Crashpunkt | Persistenter Befund | Normative Recovery |
| --- | --- | --- |
| vor Revisionscommit | keine Fachmutation und kein Control-Plane-Effekt | normaler Clientretry |
| nach normalem Revisionscommit, vor Worker | Commit, Change und Dirty/Fence für `T`/`C` vorhanden | Queue beziehungsweise Reconciler nimmt den STATE1-Anteil auf; bei Securitycommit bleibt Serving bis zum erforderlichen Checkpoint-CAS fail-closed |
| nach authority-gemischtem Fachcommit, vor STATE1-Worker | Fachrecord, ownerfremder Anteil sowie Commit/Change/Dirty für `T`/`C` sind in derselben SQLite-Transaktion durable | jeder Owner recoveriert ausschliesslich seinen persistenten Anteil; keine Seite rekonstruiert die Fachmutation oder erzeugt den Anteil der anderen |
| Fachcommand gewinnt vor der Handoff-Precommit-Bindung | Authorityversion oder Source-Revision ist neuer als der prozesslokale Preflight | Precommit legt keinen `prepared`-Swap an, quarantänisiert die Kandidatengeneration und baut nach neuem Preflight vollständig neu |
| Handoff-CAS gewinnt unmittelbar vor Fachcommand-Commit | übergebene Rolle ist bereits `state1` | Fachcommand-CAS rollt vollständig zurück, berechnet die Authoritypartition neu und schreibt beim Retry für diese Rolle ausschliesslich STATE1-Change/Dirty |
| nach erstem Assignment in den bereits vorhandenen initialen Shard, vor Memberwrite | Assignment auf Slot `0`, befüllte Manifestversion und Change sind durable; das bisher leere physische Member und seine Family-Bindung bestehen bereits | Delivery in genau dieses Member idempotent aufnehmen und eine neue Epoch publizieren; den ersten Trail nie durch einen zweiten Shard oder ein zweites Member umleiten |
| nach Allocation eines weiteren Route-Shards, vor physischer Memberprovisionierung | Assignment, Slot, neuer logischer Shard, Manifestversion, Change und Generationenmember sind atomar durable; dessen Contractwerte stammen aus der bereits gebundenen Generation-Queryfamily; alter Snapshot enthält das neue Member nicht | Memberbuild idempotent aus dieser Family-Bindung aufnehmen; alter Snapshot bleibt vollständig querybar, neuer Katalogstand bis zur nächsten vollständigen Epoch unpubliziert |
| nach Contract-Artefakt-/Resourcecommit, vor erster Memberregistrierung | immutable Payloadbytes und 1:1-Resource sind `staged`, keine öffentliche Bindung und kein HWM | denselben Artifact-Key idempotent weiterverwenden oder bei endgültigem Abbruch ohne Pins `staged -> gc_eligible`; nie aus Binarydefaults neu erzeugen |
| nach Generation-Queryfamily-Commit beim leeren Katalog, vor Provisionierung des initialen Members | immutable Family-/Contractbindung, niemals leeres Geo-Manifest mit Shard Ordinal `0` und vollständiger Generation-Queryfamily-Manifesthash sind durable; die Generation ist noch nicht ready | aus derselben Family idempotent genau das dokumentleere generationenspezifische Member für den initialen Shard provisionieren und dessen Marker bestätigen; nie ohne dieses Member publizieren |
| Worker nach Claim | Lease läuft aus | höherer Token übernimmt; altes Ergebnis scheitert an Record-ID/Inkarnation/Token-CAS |
| nach Projektionsbuild, vor Abschlusscommit | kein sichtbares Lieferartefakt | deterministisch neu bauen |
| nach erfolgreichem Build eines neuen Route-Shard-Members, vor Epoch-CAS | physischer Index und Tasks/Marker durable, aber kein Snapshot bindet das Member | Marker erneut prüfen und normalen Epoch-CAS fortsetzen; der Gateway darf das Member vorher nicht adressieren |
| nach Abschlusscommit, vor Publish | Version und resolved Changes vorhanden | Publisher bildet daraus einen inputbereiten Zielprefix; Delivery ist noch offen |
| nach Attemptcommit, vor erstem Enginetask | Attempt und Tasks `prepared` | derselbe Holder darf guarded einreichen; nach Writer-Failover führt `adopt_recovery_attempt` mit technischem Fence `prepared -> recovery` aus und verwendet unverändert dieselben Intents |
| nach `epoch_publish`-Attempt des leeren Katalogs, vor Epoch-CAS | Attempt bindet Generation-/Route-Family, Contract, initialen Shard, dessen genau ein leeres Route-Member sowie vollständige Zielmanifeste und Marker-/Taskbelege | Recovery validiert dieselben realen Member-/Enginebelege und führt den Epoch-CAS fort; kein vakuoser Mengenbeleg darf Member oder spätere Ein-Subquery-Ausführung ersetzen |
| nach Submission-`prepared`, vor HTTP | kein möglicher Engineeffekt | bei weiter gültigem Guard denselben Submissionrecord verwenden; bei verlorenem Frontier-/Term-/State-Guard `prepared -> canceled` mit `guard_canceled_before_network`-Proof |
| synchrone Engine-Ablehnung mit typisiertem Non-Acceptance-Proof | Submission `failed/canceled`, keine Task-ID und kein möglicher Teileffekt | denselben Taskintent nur unter normalem Retryguard mit nächstem lückenlosen Submissionrecord erneut versuchen |
| angenommene Task endet `failed/canceled` | Task-ID und terminaler Status bekannt | nur mit persistiertem typisiertem Terminal-No-Effect-Proof und ohne späteren Intent erneut submitten; sonst Recovery/Quarantäne ohne Resubmit |
| Engine hat Task möglicherweise angenommen, Task-ID fehlt | Submission `submitting/unknown` | nicht blind resubmitten; beweisbare Barriere oder Generation quarantänisieren/neu bauen |
| bekannte Task-ID wird nach Annahme durch Task-GC, Engine-Reset oder Markerwiderspruch unbeweisbar | Submission `accepted -> unknown` | Barrier-/Endmarkerbeweis versuchen; sonst Generation quarantänisieren/neu bauen, nie als `failed` erraten |
| mehrere bekannte Submits desselben Intents | alle Submission-IDs persistent | alle terminal bestätigen und Endmarker prüfen |
| nach `succeeded`, vor Delivery-/Watermark-Ack | Task, Submissions und Intent persistent | Marker erneut prüfen, `search_catalog_delivery` ack und Prefix fortsetzen |
| Visibility-Frontier ändert sich nach `fence_apply`-Anlage, aber vor möglichem Submit | Binding-CAS verliert, kein möglicher Engineeffekt | etwaige nur `prepared`e Submissions beweisbar `canceled`, Attempt aus `prepared` oder nach Writer-Adoption aus `recovery` nach `abandoned`, aktuellen kumulativen Frontier neu planen |
| Visibility-Frontier ändert sich nach möglichem `fence_apply`-Submit | alter Payload kann physisch wirksam sein, Gateway-Guards und Submissions persistent | keinen alten Delivery-/Watermark-Ack setzen; Effekt plus Endmarker/Barriere vollständig klären, dann `superseded` und aktuellen Fence-Apply-/Epoch-Repair durable anlegen oder bei Unklarheit Recovery/Quarantäne |
| nach allen Acks, vor Epoch-CAS | Generation weiter `updating` | Gates erneut prüfen und CAS committen |
| während Query | Pre-/Post-Stempel verschieden | gesamte Response verwerfen |
| Shardmanifest, Contract-Artefakt-/Resourcebindung oder Snapshot-Memberdigest ändert sich während Route-Query | Pre-/Post-Familiendigest verschieden oder Snapshotbeleg widersprüchlich | gesamte Multi-Search-Antwort verwerfen; niemals bereits erhaltene Shards als Teilantwort liefern |
| nach HWM-/Securitycommit, vor externem Checkpoint | Kandidat samt Sequenz und Manifest durable; Response nicht sichtbar | denselben oder einen nachweislich subsumierenden neueren Kandidaten per externem CAS bestätigen; sonst fail-closed |
| konkurrierender höherer Checkpoint gewinnt zuerst | externer Anchor besitzt höhere Sequenz | Manifest-Einschluss des wartenden Requests prüfen; kleinerer Writer darf nie zurückschreiben |
| nach externem Checkpoint, vor lokalem Visibility-CAS | Retention extern zugesagt, Response noch nicht sichtbar | aktuellen lokalen Manifest-/Resource-/Stempelstand prüfen; bei `lost` oder neuer Restriction verwerfen |
| nach lokalem Visibility-CAS, vor HTTP-Antwort | sichere Überretention und lokal linearisierte Response | keine fachliche Recovery nötig |
| Publisher-Failover mit unresolved Attempt | alter Term und Submissions persistent | extern fencen, höheren Term übernehmen, `adopt_recovery_attempt`-CAS |
| vor Generation-Swapcommit | bisheriger Head unverändert | Bootstrap-/Normal-/Recovery-Swap erneut versuchen |
| nach Generation-Swapcommit | neuer Head und beide Lifecycles atomar sichtbar | neuen Head bedienen; keine Enginekompensation |
| während des Legacy-Authority-Generationsbuilds, vor Precommit-Binding | Head und Legacy-Authority unverändert, `new` höchstens staged/teilgebaut, noch kein `search_generation_swap` und keine persistente Handoff-Buildbindung | `new` samt Membern quarantänisieren; unter neuem Prozesskontext Preflight wiederholen und auf neuen physischen IDs vollständig neu bauen, niemals staged Arbeit fortsetzen oder adoptieren |
| Source-, Operations-, Ressourcen- oder Authoritybindung ändert sich zwischen Preflight und Precommit | noch kein `prepared`-Swap; die prozesslokale Buildbindung ist stale | unter den Precommit-Locks Abweichung feststellen, keinen Swap anlegen, `new` samt Membern quarantänisieren und nach neuem Preflight auf neuen physischen IDs vollständig neu bauen; kein externer Source-Cutoff/-Replay |
| nach `prepared`, vor gemeinsamem STATE1-Head-/Legacy-Authority-Handoff-Commit geht Prozesskontext oder ein Lock verloren | Head und Authority bleiben unverändert oder Source kann nach Lockverlust fortschreiten; der alte Swap besitzt keinen adoptierbaren Lockkontext | alten Swap nach Nichtcommitbeleg `prepared -> aborted`, `new` samt Membern quarantänisieren und zwingend ab Preflight vollständig neu bauen; bei intakten Locks kann in diesem Fenster keine Legacyänderung committen |
| nach gemeinsamem STATE1-Head-/Legacy-Authority-Handoff-Commit | Head aktiv und jede deklarierte Authorityzeile atomar `state1` mit derselben Pointerversion | nur STATE1 bedienen/recovern; der Legacyowner darf alte physische Indizes weder routen noch reaktivieren |
| Engine-Restore/-Reset | `engine_incarnation` abweichend | betroffene Generation blockieren/quarantänisieren und validieren oder neu bauen |
| PocketBase-Restore/-Rollback | externer Restore-/Security-/HWM-Checkpoint liegt vor DB | Gateway, Publisher und GC blockieren; Abschnitt 10.6 vollständig ausführen |
| vor Restore angenommener Enginetask fehlt im zurückgespielten Submissionlog und läuft verzögert weiter | alter Task kann nur alte physische IDs adressieren | alte Generation nie reaktivieren; neue Inkarnation/Generation/IDs vollständig bauen, alte IDs bis umfassendem Drain/Barrier oder Namespace-Reset quarantänisiert retenieren |
| Fencepropagation fällt aus | Fence bleibt `active/propagating` | Gateway verengt oder blockiert; Retry, niemals Freigabe |

Recovery darf `checkpoint_sequence`, `restore_epoch` oder `security_revision`
nie verkleinern. Katalog-/Snapshotset-/Pointerrevision, Content-Epoch,
Publisherterm und Leasetoken dürfen innerhalb ihrer in Abschnitt 3.1
festgelegten Identität ebenfalls nie sinken oder wiederverwendet werden. Ein
Restore eröffnet stattdessen ausdrücklich eine neue Control-Plane-,
Generationen- beziehungsweise Writer-Fence-Identität; gleiche rohe Zahlen aus
verschiedenen Identitäten behaupten keine historische Kontinuität.

## 13. Verbindliche Invarianten und Abnahme

### 13.1 Datenbankinvarianten

- Genau ein `search_catalog_state`, ein Restore-State und ein Generation-Head
  pro Scope beziehungsweise Contract/Entity-Kind; der Head darf vor Bootstrap
  ausdrücklich `empty` sein.
- Pro logischer Suchrolle existiert genau eine Mutations- und Routingautorität.
  Ein `legacy -> state1`-Wechsel ist nur gemeinsam mit dem ersten sie
  vollständig abdeckenden Head-CAS, mit identisch gebundenem Head-Key und dessen
  neuer Pointerversion als dauerhaftem Handoff-Floor sowie ohne Rückgabekante
  zulässig. Vorher müssen die externe Authorityversion, Source-Revision, der
  Source-Snapshot und die Legacyressourcenmenge exakt dem Handoff-Snapshot
  entsprechen; nachher sind ausschliesslich Head, Generation und Publisher von
  STATE1 autoritativ.
- `search_catalog_commit.revision` ist lückenlos monoton; Zahl und
  `change_manifest_digest` seiner Change-Zeilen stimmen, und `cause_digest`
  bindet die vollständige autorisierte Mutationsabsicht.
- Jede projektionsrelevante committed Fachmutation bindet ihre vollständige
  Registryrevision und Authoritypartition. Bei
  `needs_catalog_revision = (T != []) || (C != [])` besitzt sie genau einen
  Commit und Changes für die Vereinigung aus den von STATE1 verantworteten
  Projektionseffekten `T` und den authority-unabhängig verpflichteten Control-
  Changes `C`; nur bei `T = [] && C = []` erzeugt sie keine leere STATE1-
  Katalogrevision. Fachrecord, ownerfremde Transitionseffekte, STATE1-Anteil
  und Authority-CAS sind atomar ganz oder gar nicht sichtbar.
- Der Route-Shard-Allocator besitzt genau einen Singleton. Nach `active` hat
  jeder katalogfähige Trail genau ein append-only Assignment; `local_created_at`,
  `geo_shard_id`, Slot und Assignmentrevision ändern sich nie. Remotezeiten,
  Update, Federation-Sync, Expiry und Delete können keine Neuzuordnung
  auslösen.
- Shardordinals und Slots sind lückenlos, pro Shard existieren höchstens 1.000
  jemals zugewiesene Trails, `assigned_slots` sinkt nie und ein versiegelter
  Shard wird nie wieder geöffnet. Das Shardmanifest ist niemals leer: Vor der
  ersten Zuweisung enthält es genau den offenen initialen Shard Ordinal `0`
  mit `assigned_slots = 0`, leeren Allocation-Grenzen und kanonischem Empty-
  Assignment-Digest; `current_open_shard` zeigt auf ihn. Jeder aktuelle Shard-/Allocatorhash und jede
  immutable Manifestversion ist aus den retenierten Assignmentzeilen exakt
  rekonstruierbar.
- Ein Head ist nur im Zustand `active/published` mit bestätigten Bundlehashes
  und allen erforderlichen expliziten Contract-Artefaktbindungen querybar;
  `updating`, `recovery` oder `quarantined` bleiben am Head sichtbar, liefern
  aber fail-closed keine Antwort.
- `published` bedeutet für jedes im aktuellen Content-Snapshot gebundene
  Member `delivered_revision` mindestens am Generation-Cutoff und einen
  `security_watermark`, der jeden bis zu diesem Cutoff existierenden
  Security-Change lückenlos schliesst, dazu epochgenaue
  Federation-Scope-Bindings und ausschliesslich terminal
  erfolgreiche **epochgebundene Content-/Settings-Taskintents** samt aller
  physischen Submissions; kein solches Taskinput und kein
  Projektionsbuildstand liegt oberhalb dieses Cutoffs. Ein nichtterminaler
  `fence_apply` ist nur unter dem bereits wirksamen, vollständig
  beobachtungsäquivalenten `exact_target`-Security-Sondervertrag mit
  vollständigem Multi-Frontier-Binding aus Abschnitt 4.3 zulässig und
  verändert diese Epoch nicht. Ein erst für eine spätere Route-
  Manifestversion registriertes, noch ungebundenes Member kann gemäss
  Abschnitt 8.1.1 parallel `new/building/ready/blocked` sein; es ist weder Teil
  dieser Aussage noch querybar und blockiert jede Epoch, die es benötigt.
- Innerhalb einer Generation sinken `content_epoch`, `catalog_cutoff`,
  `delivered_revision` und `security_watermark` nie. Jeder
  `epoch_publish`-Attempt bindet den bisherigen Cutoff und sämtliche
  Watermarks seiner manifestgebundenen Zielmember vor dem ersten Submit; sein
  Zielcutoff liegt mindestens auf allen eingefrorenen Delivery- **und**
  Security-Watermarks. Normal- und
  Recovery-Abschluss verwenden dieselbe Baseline und dürfen keinen bereits
  weiterliegenden physischen Content- oder Securitystand als kleineren Cutoff
  deklarieren. Nur ein nach der Baseline entstandener
  Delivery-/Security-Vorlauf, dessen Security-Einträge entweder durch
  weiterhin `query_fenced`-te Gateway-Fences oder als vollständig irrelevante
  Changes typisiert belegt sind, darf mit vollständigem Post-Baseline-Proof
  bestehen bleiben; normale Content-/Settings-Effekte über dem Cutoff bleiben
  verboten.
- Kein Generation-Cutoff liegt oberhalb der committed Katalogrevision. Der
  Head-Swap verlangt unter demselben SQLite-Schreiblock exakte Gleichheit;
  weder eine zukünftige Revision noch ein nur behauptetes Prefix kann dadurch
  spätere reale Changes überspringen.
- Kein Watermark überspringt einen unresolved relevanten Change; jeder
  gezählte Beleg ist durch `search_catalog_delivery` auf Change und Member
  zurückführbar.
- Jeder als `irrelevant` übersprungene Change besitzt den typisierten,
  versionierten und aus retenierten Regel-/Inputartefakten rekonstruierbaren
  Komponentenbeleg; ein leerer oder nur frei formulierter Grund schliesst
  keinen Prefix.
- Innerhalb eines Engine-Scopes gehört jede physische Index-ID genau einem
  Generationenmember und wird nie wiederverwendet.
- Jedes `search_contract_artifact` ist aus seinen gespeicherten kanonischen
  Payloadbytes und dem Payloadschema vollständig rekonstruierbar, sein Hash
  stimmt und seine unveränderliche Ownerrelation zeigt 1:1 auf
  `contract_artifact/artifact_key`. Provenienzpfade, laufende Binary sowie
  Schema-, Settings- oder Bundlehashes ersetzen diesen Beleg nie.
- Jede Generation und jeder Content-Snapshot mit
  `route_radius_ux_v1` besitzt genau eine immutable Queryfamily-Bindung mit
  Contractrelation, Key und Hash. Die Snapshot-Family kopiert aus der
  Generation-Family; beide existieren auch beim belegungsleeren initialen
  Route-Member. Die vollständigen Generation-/Snapshot-Family-Mengen stimmen
  bijektiv überein und ihre jeweiligen kanonischen Manifesthashes sind gesetzt;
  Member sind ausschliesslich konsistente Ausführungskopien dieser Authority.
- Für `route_radius_ux_v1` gehört jedes physische Member zusätzlich genau
  einer `(generation, geo_shard_id)`-Identität. Sein Input enthält nur Trails
  mit diesem persistenten Assignment; ein Rebuild oder Rollback erzeugt neue
  physische IDs über derselben logischen Zuordnung und repackt nie. Relation,
  Key und Hash seines Contract-Artefakts stimmen mit Payload, Taskinput,
  Coverage und Endmarker überein; alle Route-Member derselben Generation
  kopieren dasselbe Artefakt aus ihrer Generation-Queryfamily.
- Jeder Content-Snapshot besitzt eine unveränderliche vollständige
  Membermenge aller Queryfamilien, und `member_manifest_hash` schliesst
  insbesondere `trail_default` ein. Ausschliesslich seine nach
  `query_family = route_radius_ux_v1` gefilterte Teilmenge ist bijektiv zu
  **allen** Shards der gebundenen cutoffgenauen Manifestversion. Beim leeren
  Katalog enthält sie genau das dokumentleere Member des initialen Shards und
  `member_count = 1`. Die Snapshot-Queryfamily bindet den Contract unabhängig
  davon und alle Member kopieren daraus. Ein später
  registriertes Member verändert weder diesen Snapshot noch einen darauf
  gebundenen Cursor.
- Der Gateway leitet jede Route-Subquery aus genau diesem Snapshotmanifest ab.
  Treffer, globale Sortierung, Seite, Total und Counts dürfen nie aus einer
  Teilfamilie oder aus anwendungsseitig zusammengeführten Shardseiten stammen.
  Der belegte Empty-Catalog-Fall führt je logischer Anfrage genau eine
  federierte Multi-Search mit genau einer Subquery an das initiale leere Member
  aus; nur deren exhaustive
  Antwort liefert leer/0. Ready-Aggregationen liefern nur aus diesem Enginebeleg
  und mit vollständig gebundenem Options-/Bucketvertrag vollständige Nullcounts.
- Höchstens ein mutierender Engine-Attempt des serialisierten Writepfads ist
  gleichzeitig nichtterminal; Failover ändert seinen Ownerterm nur über
  `adopt_recovery_attempt`, ein idle Termwechsel nur atomar mit dem nächsten
  Attempt über `adopt_idle_generation_term`.
- Ein aktiver Fence ist in jeder querybaren oder swap-fähigen Generation
  `applied` oder vor Ranking/Counts `query_fenced`.
- Jeder epochlose `fence_apply` bindet für jedes mutierte physische Ziel die
  vollständige aktuelle Menge aller überlappenden Subject-Frontiers. Anlage,
  jeder Pre-Submit-Guard, Abschluss und späterer Retirement-CAS prüfen dieselbe
  Dependency-Closure samt Frontier-/Guard-Digests erneut. Ein überholter
  Apply kann nur vor jedem möglichen Engineeffekt `abandoned` werden; bereits
  angelegte `prepared`-Submissions sind dann mit typisiertem
  Non-Submission-Proof terminal `canceled`. Nach einem möglichen Effekt ist
  nur der vollständig geklärte Zustand `superseded`; der alte Beleg erhöht
  keinen Watermark und entfernt keinen Guard.
- Ein `retired` Fence besitzt in keiner querybaren, `ready`, swap-fähigen oder
  aufholenden Generation ungeschützt lediglich seine alte
  `query_fenced`-Delivery: Je nach Zweig ist dort die Restriction physisch
  wirksam, in aktuellen Generationen der Clear `applied/subsumed`, in einer
  versiegelten Alt-Epoch weiterhin die alte Restriction physisch wirksam, ein
  mindestens ebenso enger Successor-Fence aktiv, die Generation irreversibel
  aus Query- und Swapmengen entfernt oder jeder betroffene Alt-Cursor durch den
  membergebundenen retenierten `guard_release_revision`-/`clear_revision`-
  Anker nachweislich stale. Derselbe zweigkorrekte Retirement-Proof hält dann
  den historischen `fenced`-Catalog-Ack für diesen Member gültig.
- Ein epochloser `restriction_converged`-Retirement-CAS ist nur für
  `gateway_guard_kind = exact_target` und mit typisiertem vollständigem
  Beobachtungsäquivalenzbeleg erlaubt. Jede Lockerung eines konservativen
  Whole-Subject-Guards läuft über normale Content-Epoch,
  `guard_release_converged` und den Issuance-Fence.
- Nach `guard_release_converged` oder
  `guard_release_superseded_by_restriction` kann keine relevante Cursorseite
  aus einer vor `guard_release_revision` liegenden Alt-Epoch den temporären
  Whole-Subject-Guard später gelockert beobachten: Ihr Snapshot ist anhand des
  retenierten Fencebelegs stale, die Generation ist irreversibel unquerybar
  oder ihr physischer Stand ist zum danach gültigen Ziel
  beobachtungsäquivalent.
- Nach `clear_superseded_by_restriction` kann ebenso kein relevanter Cursor mit
  Snapshot vor `clear_revision` von der alten F1-Restriction auf den
  möglicherweise weiteren F2-Stand wechseln. Vor Retirement ist die normale
  F2-Successor-Epoch publiziert; alte aktive Cursor sind durch Epochwechsel
  beziehungsweise retenierten Clear-Anker stale, neue Kontexte beginnen auf
  dieser Successor-Epoch.
- Jeder `retired` Fence besitzt den zweigkorrekten, aus retenierten
  Member-/Delivery-/Projektionsbelegen rekonstruierbaren Retirement-Proof; er
  hält jede zuvor terminale `fenced`-Catalog-Delivery auch nach Ende ihres
  Gateway-Guards auditierbar gültig. Spätere Successor-Links sind append-only,
  streng revisionssteigend und werden bis zum Ende der Security-History
  vollständig verfolgt; der terminale Fence selbst wird nie umgeschrieben.
- Ein Fence mit eingehendem `clear_superseded_by_restriction`- oder
  `guard_release_superseded_by_restriction`-Link bleibt bis zum Retirement
  sämtlicher dadurch neutralisierter Vorgänger selbst nicht-retired.
  Linkanlage und Retirementprüfungen laufen unter demselben SQLite-Schreiblock;
  die strikt steigenden Successor-Revisionen erzwingen eine azyklische
  Retirement-Reihenfolge von alt nach neu.
- Jede Fence-Successor-Kette besitzt dank nichtleerer Root-Sentinel und Unique
  `(fence, predecessor_key)` genau einen Root und keinen Fork. Ein
  `clear_converged`-Proof bindet die ganze Kette bis zum unter demselben
  Schreiblock erneut gelesenen Tail; `clear_revision` bleibt ihr erster
  unveränderlicher Clear-Anker.
- Kein Suchkontext und kein Cursor wird ausgestellt, wenn sein Content-Snapshot
  eine für sein vollständiges Treffer-/Aggregationsuniversum relevante
  `clear_revision` oder `guard_release_revision` bereits enthält, der
  zugehörige Fence aber noch nicht `retired` ist. Der finale
  Response-Visibility-CAS prüft diese Bedingung erneut.
- Eine `deleted` Federation-IRI wird nicht durch alten oder doppelten Input
  reaktiviert.
- Jeder angenommene Federation-Payloadstand ist durch einen Objektkopf-CAS und
  einen FED0-Currentness-/Kausalbeleg strikt nach seinem Vorgänger geordnet;
  ungeordneter Input kann weder Sichtbarkeit noch TTL erweitern.
- Jede nichtleere Federation-Scope-Version und jedes daraus publizierte
  Snapshot-Binding besitzt ein rekonstruierbares Membershipmanifest und als
  `earliest_discoverable_until` exakt dessen kleinste Einzel-Expiry. Das Feld
  ist genau bei leerem Manifest leer; Scope-Digest und Federationmanifest
  binden Membership, Expiries und Minimum gemeinsam.
- Keine zugesagte Generation, kein Content-Snapshot, kein Cursor-Key, kein
  Vertragsartefakt und keine benötigte Security-History wird vor HWM plus Grace
  gelöscht.
- Ein mit `restriction_converged` oder `guard_release_converged` retirter Fence
  ohne `grant_after_retirement`-Link ist unabhängig von
  `history_retain_until` nicht GC-fähig. Ein später Grant und ein GC-Versuch
  sind durch denselben SQLite-Schreiblock geordnet; entweder der Link samt
  verlängerter Retention committet oder GC verliert fail-closed.
- Jeder terminale Fence besitzt ein vollständiges
  `retirement_artifact_manifest` und eine dazu bijektive normalisierte Menge
  aus `search_fence_retirement_artifact_pin`; solange dessen Fence-/Proofzweig
  reteniert ist, bleiben alle darin gepinnten PocketBase-Snapshot-, Epoch-,
  Scope-, Member-, Delivery-, Successor- und Regelartefakte rekonstruierbar.
  Das Löschen eines externen Engine-Index darf diese Metadaten-Pins nicht lösen.
- Jeder terminal `canceled`-Submissionrecord besitzt genau den zu seiner
  Herkunftskante passenden Non-Submission-, Non-Acceptance- oder
  Terminal-No-Effect-Beleg. Ein Task `prepared -> canceled` ist nur im
  effektfreien Abschluss eines Nicht-Epoch-Attempts bei
  `submission_count = 0` erlaubt; ein bereits eingereichter Task bindet im
  `cancellation_proof_hash` sämtliche terminalen Child-Belege. Erst dann darf
  der Attempt `abandoned` werden.
- Der reguläre Öffnungs-/Schliesspfad einer `search_context_resource` ist
  `staged -> issuing -> retained`; ein unbenutzter Kandidat darf stattdessen
  GC-fähig und jeder physische Verlust terminal `lost` werden. Eine bereits
  geschlossene Ressource wird nie wieder geöffnet. `barriered` besitzt immer
  typisierte Taskrelation, passenden bestätigten Endmarker und Proofhash.
- Mutable Resource-Version und HWM sind kein semantischer Pre-/Post-
  Query-Stempelbestandteil. Die eigene Reservation und der finale CAS prüfen
  immutable Owneridentität, zulässigen Zustand und
  `current_hwm >= valid_until`; eine parallele grössere HWM-Erhöhung bleibt
  gültig, während Retention, Loss und GC fail-closed gewinnen.
- Gateway, Publisher und GC sind nur bei Restorezustand `healthy` ready; der
  externe Security-/HWM-Checkpoint liegt nie hinter einer bestätigten Response,
  seine Sequenz sinkt nie und sein Manifest enthält jede noch gültige Zusage.
- Jedes `OptionalSafeRevisionV1` besitzt ein konsistentes Presence-Boolean;
  `false` verleiht dem gespeicherten Zahlenwert keinerlei Semantik.
- Alle internen Collections besitzen gesperrte normale API-Regeln.

Diese Invarianten erhalten ausführbare Diagnosequeries. Readiness bleibt
fail-closed, sobald eine harte Invariante nicht beweisbar ist.

### 13.2 Modell- und Integrationstests

Vor produktiver Aktivierung einer abhängigen Capability bestehen mindestens:

- Transition-Tabellentests für jede erlaubte und verbotene Kante;
- parallele Revisions-, Dirty-Coalescing-, Lease-Ablauf-, Sweep-Wrap- und
  Sweep-Cursor-CAS-Tests;
- Table-driven Ingressmatrix über Create/Update, Direct-Announce,
  Remote-Resolve, Listenexpansion, Parent-Fetch, Admin-Resync und
  Legacy-Migration: Public/direct/fehlender Beleg ergeben auf jedem Pfad
  dieselbe Authority-, Provenienz-, Visibility- und Scopewirkung;
- anonyme und authentifizierte Suche über local/federated ×
  public/private/shared/revoked/deleted/expired/unverified prüft Treffer,
  Total, Facetten, Histogramme, Highlights und DTOs gegen denselben
  SEC-SCOPED-Stand;
- Direct→Public, Public→Direct und Public→Private→Public erzeugen niemals eine
  unbelegte Global-Membership; erneute Publikation verwendet P2 mit neuer IRI,
  während P1 absorbierend tombstoned bleibt;
- richtige Authority, fremder Actor, falsche Origin, Keyrotation, unbekannte
  IRI und Replay werden für Update und Delete geprüft; `trails.author`,
  `lists.author` oder der Autor eines Parent-Comments/-Logs autorisieren nie
  eine Objektmutation;
- mehrere Shares, Widerruf des letzten und Widerruf eines von mehreren Shares
  beweisen, dass nur der bezeichnete ACL-Grant entfällt;
- hundert Restriktionen desselben Origin-/Actor-/ACL-/Publication-Scopes
  während eines Worker-Lease erzeugen höchstens eine laufende und eine
  anschliessende Projektion; Local- und fremde Origin-Scopes bleiben bereit;
- Crashs vor und nach Inboxcommit, Restriction-/Outboxcommit, Remote-Send,
  Scope-Dirty-Upsert, Engine-Submit und Scope-Watermark sind idempotent
  recoverbar; nach durablem Inbox-`202` entsteht nie eine zweite HTTP-Antwort;
- Legacy-Migration mit Offline-Origin, `404/410`, ungültiger Authority,
  Neustart, Restore und Concurrent-Ingest endet je Record genau in einem
  terminalen Outcome und kann vor `complete` kein P2-Gate öffnen;
- alte Tenant-Tokens und direkte Legacy-Enginepfade können nach Cutover weder
  private noch unverified/quarantänisierte Remote-Dokumente lesen;
- gleicher `cause_key` mit identischer Absicht ist Replay, mit abweichender
  Fachpayload trotz gleicher Change-Header dagegen Konflikt/Quarantäne;
- Out-of-order-, Duplicate-, Authority-Wechsel-, TTL-, `403/404/410`-, Delete-
  und Resurrection-Negativtests der Federation; insbesondere darf ein spät
  verarbeitetes älteres Public-Update einen neueren Private-/Delete-Stand oder
  dessen TTL nicht überschreiben, und gleicher Effect-Key mit anderem
  Effektdigest muss statt `duplicate` in Quarantäne enden;
- Federation-TTL `T1 -> T2` zwischen zwei Epochs: die alte Epoch bindet weiter
  `T1`, der aktuelle Head kann sie nicht verlängern;
- Scope-Minimum-Negativtests: Nichtleere Membership mit leerem Minimum,
  Minimum nach der kleinsten Einzel-Expiry, leere Membership mit gesetztem
  Minimum sowie Manifest-/Digestabweichung blockieren Scope-Version, Publish
  und Issuance. Bei `response_now = earliest_discoverable_until` ist der
  halboffene Kontext bereits abgelaufen;
- verpasster Federation-Refresh: eine erst nach `T1` gestartete autoritative
  `equal`-Revalidierung darf `expired -> discoverable` mit neuem Scope-/
  Projektionsstand ausführen, aber nie `restricted/quarantined/deleted`
  wieder öffnen;
- Tombstone-/Subsumption-/GC-Tests mit spät registrierter Shadowgeneration;
- Route-Shard-Grenztests mit 0, 1, 999, 1.000 und 1.001 Erstzuweisungen: Bei
  `0` existiert genau der offene initiale Shard Ordinal `0` mit leeren Grenzen
  und Empty-Assignment-Digest; Zuweisung `1` füllt dort Slot `0`. Slots und
  Ordinals bleiben lückenlos, der zweite Shard entsteht erst bei Zuweisung
  `1.001` über den serialisierten Allocator-CAS und kein physischer Index
  enthält mehr als 1.000 jemals zugewiesene Trails;
- parallele Erstaufnahmen sowie ein Backfill mit gleichen
  `local_created_at`-Werten werden innerhalb jedes Allocation-Batches stabil
  nach `trail_key` geordnet. Verlorene CAS werden vollständig neu aufgelöst;
  doppelte Trails, Slots oder Manifestrevisionen sind unmöglich;
- Update, Delete, Federation-Refresh/-Sync und ein Remote-Payload mit früherem
  oder späterem Quell-Erstellungsdatum erhalten `local_created_at`,
  `geo_shard_id` und Slot bytegleich. Delete senkt weder Slotzahl noch
  Assignment-Digest; ein anschliessender Rebuild füllt die Lücke nicht auf;
- Restart und PocketBase-Restore lesen die persistente Assignment-/
  Manifesthistorie. Bei `assignment_count = 0` verlangen sie Initialshard,
  Empty-Assignment-Digest und dessen physisches Member, aber keine erfundene
  Assignmentzeile. Bei positiver Slotzahl führen fehlende Assignmentzeile,
  doppelte Slotbelegung, fehlende/falsch gerichtete `current_open_shard`-Relation,
  Boundary-/Digestkonflikt und der Versuch, aktuelle live Trails neu zu
  sortieren, führen fail-closed nach Quarantäne;
- `irrelevant` schliesst den Prefix nur mit passendem Proof-Kind,
  Regelartefakt, Vorher-/Nachher-Inputdigest und unverändertem Commitdigest;
  ein fehlender, manipulierter oder nach GC nicht mehr rekonstruierbarer Beleg
  blockiert den Watermark fail-closed;
- Dirty-Koaleszierung `R1 -> Projektionsartefakt R2`: Ein Publish mit Cutoff
  `R1` ist verboten; der Publisher muss ein älteres Artefakt wählen oder den
  vollständigen Prefix und alle Scope-/Delivery-Belege bis `R2` schliessen;
- Ein `epoch_publish` mit `target_cutoff` unter dem bisherigen
  Generation-Cutoff, einer eingefrorenen Member-Deliveryrevision oder einem
  eingefrorenen Member-Security-Watermark sowie ein finaler CAS nach
  widersprüchlichem Watermark-Vorlauf müssen scheitern; Recovery darf dieselbe
  Baseline ebenfalls nicht absenken;
- Epochloser `fence_apply` der Restriction F@100 und anschliessendes Retirement,
  danach ein normaler Publishversuch mit R=95 und altem Upsert-Artefakt: Der
  Attempt muss wegen seines eingefrorenen `security_watermark = 100` vor dem
  ersten Submit scheitern. Entsteht F@100 erst nach einer gültigen
  R=95-Baseline, darf der finale CAS nur mit vollständigem
  Post-Baseline-Proof für den weiterhin aktiven `query_fenced`-Guard gewinnen.
  Dabei darf auch `delivered_revision > 95` nur aus der exakt belegten
  lückenlosen Folge von `fenced`-Security- und `irrelevant`-Changes stammen;
  ein normaler Content-/Settings-Deliverybeleg in diesem Intervall lässt den
  CAS verlieren. Physischer Fence-Apply und Retirement folgen erst nach dem
  Contentwrite;
- Reiner Security-Vorlauf: Nach R=95 bleibt ein normaler Change C@96 offen,
  danach wird F@100 `query_fenced`. `delivered_revision` bleibt deshalb
  höchstens 95, während `security_watermark = 100` zulässig sein kann. Der
  Post-Baseline-Proof muss F@100 trotzdem in der getrennten vollständigen
  `security_frontier_entries`-Achse binden; er darf weder C@96 als geliefert
  ausgeben noch die Fencezeile wegen des kleineren Deliveryprefixes auslassen;
- Rein irrelevanter Security-Vorlauf: Nach R=95 entsteht S@96, dessen
  Securitywirkung für genau dieses Member vollständig typisiert `irrelevant`
  ist. `security_watermark = 96` darf ohne erfundenen Fence-Eintrag nur
  gewinnen, wenn `security_frontier_entries` den vollständigen
  Irrelevanzbeleg bindet; ein leerer Eintrag, ein freier Grundtext oder ein
  Fence-Platzhalter muss den finalen CAS verlieren;
- Attemptanlage und Abschluss mit einem Cutoff oberhalb der committed
  Katalogrevision sowie ein Swap mit ungleichem Cutoff scheitern; eine später
  tatsächlich angelegte Revision darf nie unter einem zuvor behaupteten
  Generation-Prefix verschwinden;
- Crash-Injection an jeder Zeile aus Abschnitt 12;
- konkurrierender beziehungsweise supersedierter Publisher, der keinen
  weiteren Engine-Write ausführen kann;
- Publisher-`quarantined` bleibt ohne auditierten vollständigen
  Infrastruktur-Fence und Attemptklassifikation submit-unfähig; nur
  `quarantined -> fenced -> held` mit höherem Writer-Tupel kann Recovery
  fortsetzen, nie ein direkter Reset nach `vacant/held`;
- Ein Publisherwechsel nach `quarantined` zwischen Query und finalem
  Response-Visibility-CAS verwirft den gesamten Batch auch bei weiterhin
  `published`em Head. Erste Seite und Cursor derselben Inkarnation liefern
  `503 search_unavailable`; ein bereits höher priorisiert bewiesener
  Inkarnations-/Ressourcenverlust behält seinen spezifischen Fehler;
- sauberer `draining -> vacant -> held`-Handoff bei idle aktiver und
  Kandidatengeneration: Der nächste Attempt adoptiert den höheren Term atomar
  und kann ohne unnötiges externes Failover-Fence fortfahren;
- verlorener Frontier-/Term-/State-Guard nach durablem Submission-`prepared`,
  aber vor Netzwerk: `prepared -> canceled` nur mit typisiertem
  `guard_canceled_before_network`-Proof, danach darf ein nicht epochgebundener
  Attempt ohne möglichen Effekt `abandoned` werden. Eine zweite Variante
  adoptiert den Attempt zuerst mit `prepared -> recovery` und ändert erst
  danach den Frontier; eine dritte ändert den Frontier bereits während des
  alten Writer-Ausfalls und führt danach den atomaren
  Adopt-and-Abandon-Pfad aus. Beide verlangen denselben vollständigen
  Child-Beleg; `superseded` wäre ohne möglichen Submit verboten;
- synchron bewiesene Nichtannahme `submitting -> failed/canceled`, spätes
  `accepted -> unknown`, angenommene terminale Fehl-/Abbruchtask mit und ohne
  gültigen persistierten Terminal-No-Effect-Proof, Unknown-Submit, mehrere
  bekannte Submission-IDs, Barrierbeweis und `adopt_recovery_attempt` nach
  Credential-Fence;
- laufender/fehlgeschlagener `fence_apply` bei `published` und `sealed`: Das
  bereits query-gefencete Subject beeinflusst weder Treffer noch Ranking/
  Counts, Content-Epoch/-Cutoff bleiben gleich, und kein Task darf andere
  Dokumentwerte oder eine Sichtbarkeitserweiterung schreiben; ein nur
  `query_fenced`-tes `satisfied`-Ziel bleibt retrybar und eine cursorquerybare
  versiegelte Generation kann nach sauberem Publisher-Handoff genau für
  diesen Attempt ihren Writer-Term adoptieren. Der Attempt hält dabei
  `target_cutoff` exakt auf dem alten Content-Cutoff, setzt separat
  `target_security_revision` auf das Maximum der vollständig gebundenen
  Restriction-Frontiers (mindestens `fence.set_revision`) und erhöht
  ausschliesslich den geschlossenen Securityprefix;
- Überlappende Frontier-Races: F1 schwach `query_fenced`, danach stärkere F2
  physisch `applied/retired`, dann ein verzögerter F1-`fence_apply` darf nie
  den alten F1-Einzelpayload schreiben oder damit retiren. Vor Submit verliert
  sein vollständiges Binding; nach möglichem Submit endet er erst nach
  Terminal-/Barrierbeleg `superseded` und repariert auf den aktuellen
  kumulativen F2-Stand. Dasselbe gilt symmetrisch für
  `F1 -> Grant/Clear -> verspäteter F1-Apply`: Der alte Guard bleibt bis zur
  normalen aktuellen Epoch aktiv;
- Derselbe Race-Test kreuzt Subjectgrenzen: Ein Actor-/Origin- oder
  ACL-Scope-Fence und ein Object-/Trail-Fence treffen dasselbe physische
  Dokument. Jeder Apply bindet beide Dependency-Closures; eine Änderung eines
  Kopfes zwischen Anlage, Submit und Abschluss verhindert alten
  Delivery-/Retirement-Beleg und kann die andere Restriction nicht
  überschreiben;
- Cursor zwischen `query_fenced` und physischer Konvergenz bei
  `content_redaction` beziehungsweise einem für einzelne Principals weiter
  sichtbaren Public-to-private-Ziel: Ein konservativer Whole-Subject-Guard darf
  weder per `fence_apply` noch `restriction_converged` gelockert werden. Die
  exakte Zielprojektion publiziert eine normale neue Epoch, der alte Cursor
  wird stale, zwischen Publish und `guard_release_converged` entsteht kein
  neuer Cursor; nur ein vollständiger Beobachtungsäquivalenzbeleg erlaubt den
  epochlosen `exact_target`-Pfad;
- `F(conservative)` auf G1/Epoch e, danach Cursor C unter dem Whole-Subject-
  Guard, Swap auf G2 mit exaktem Zielstand und erst dann
  `guard_release_converged`: C auf der versiegelten G1 muss über
  `guard_release_revision` plus retenierten Fencebeleg
  `search_context_stale` werden, sofern G1 nicht nachweislich
  beobachtungsäquivalent oder irreversibel unquerybar ist; der Head-Swap allein
  darf weder Stale noch Sicherheit vortäuschen. Der historische
  `search_catalog_delivery = fenced`-Ack für G1 bleibt nur gültig, wenn genau
  dieser memberbezogene Anker im Retirement-Proof enthalten ist; fehlt er,
  scheitern Retirement und jeder darauf gestützte Prefix-/Watermarkbeleg;
- `F1(conservative)` publiziert seine exakte Ziel-Epoch und setzt
  `guard_release_revision`, vor F1-Retirement wird jedoch die strikt spätere,
  mindestens ebenso enge F2 wirksam: Der einzige Root-Link
  `guard_release_superseded_by_restriction` bindet F1-Epoch, aktuellen
  Multi-Frontier-Closure und F2-Deliveries. Erst eine geschlossene normale
  F2-Content-Epoch mindestens auf dessen Successor-Revision erlaubt F1 ohne
  Öffnungsfenster zu retiren; F2 übernimmt den Guard. Ein absichtlich zuerst versuchter
  F2-Retirement-CAS muss bis zu F1-Retirement scheitern. Ohne Link oder mit nur
  teilweise wirksamem F2 bleibt der Issuance-Fence fail-closed aktiv;
- Cursor-Race desselben Zweigs: C wird nach F2s Securitycommit, aber vor
  F1-Retirement auf einem Snapshot `< F1.guard_release_revision` noch unter
  F1s Whole-Subject-Guard ausgestellt und bindet bereits F2s Securitystand.
  Die vor Retirement zwingende F2-Content-Epoch macht C stale; liegt C auf
  einer versiegelten Altgeneration, erzwingt spätestens der retenierte
  F1-Beleg dieselbe Entscheidung. Auch wenn beim Retirement weder Content-Epoch
  noch Securityrevision erneut steigen, darf C nie auf den gegenüber dem
  temporären Guard weiteren F2-Stand wechseln. Ein nach Retirement neu
  ausgestellter Kontext stammt dagegen aus der F2-Epoch und ist nicht durch
  den alten Anker selbststale;
- Derselbe Guard-Release-Supersession-Test mit F2 **vor** dem ersten
  F1-Publish: F2s Revisionstransaktion muss den Root-Link bereits gegen
  F1s `guard_release_revision`, `exact_target_digest` und Regelversion
  anlegen. Dirty-/Projektionskoaleszierung darf den nie publizierten
  F1-Zwischenstand überspringen; sobald die geschlossene F2-Content-Epoch den
  Successorstand terminal publiziert und F2 `satisfied` ist, kann F1 über den
  Link retiren. Weder ein nachträglicher zweiter Link-Commit noch ein
  nicht mehr auswählbares R1-Artefakt darf für Fortschritt erforderlich sein;
- `epoch_publish` darf auch vor dem ersten Submit nicht `abandoned` werden und
  bleibt über denselben eindeutigen Attempt recoverbar; nach Writer-Failover
  führt der technisch gefencete Übernahme-CAS ausdrücklich
  `prepared -> recovery` aus;
- Build, Catch-up, Fence während Build, atomarer Head-Swap und getrennt
  aufgeholter Rollbackkandidat;
- Empty-Catalog-Bootstrap für `route_radius_ux_v1`: Generation und
  `epoch_publish`-Attempt binden ihre Generation-Queryfamily, das qualifizierte
  Contract-Artefakt, das niemals leere Geo-Shardmanifest mit initialem Shard
  Ordinal `0`, dessen genau ein dokumentleeres physisches Route-Member sowie
  die vollständigen Queryfamily- und Membermanifest-Hashes. Der atomare
  Publish-/Swap-CAS erzeugt die Snapshot-Queryfamily mit `member_count = 1`;
  Route-Teilmenge und Ein-Shardmanifest sind bijektiv. Fehlende Familyzeile,
  fehlende Contractbindung, fehlendes/zusätzliches Member, nichtkanonischer
  Empty-Assignment-Digest oder ein vakuos bestandener Memberguard blockieren
  Attempt, Publish und Swap;
- die primäre logische Anfrage einer ersten Seite auf diesem katalogleeren
  aktiven Snapshot führt exakt eine federierte Meilisearch-Multi-Search mit
  exakt einer Subquery gegen das leere physische Member aus, durchläuft ACL-, Fence-, Context-, Pre-/Post-Stempel-,
  Resource-HWM- und Checkpointguards vollständig und akzeptiert nur deren
  exhaustive `hits = []`, `totalHits = 0` und daraus `page.returned = 0` sowie
  `page.next_cursor = null`. Angeforderte
  Ready-Facetten-/Histogrammserien stammen jeweils aus ihrer gleichermassen
  vollständigen logischen Ein-Subquery-Ausführung und werden nur bei vollständig
  gebundenen sichtbaren Optionen/Buckets vollständig mit `0` ausgegeben;
  andernfalls entsteht der typisierte Gruppenfehler und weder
  Teil- noch Schätzzahlen. Ein so ausgestellter Cursor bleibt über seine
  Snapshot-Family-/Contract-/Ein-Shard-Member- und HWM-Pins reproduzierbar;
- der erste Trail nach dem Empty-Snapshot belegt Slot `0` desselben initialen
  Shards. Es entsteht weder ein zweiter Shard noch ein zweites Member; das
  vorhandene Member kopiert weiterhin seine Contractwerte aus der Family und
  wird über normale Delivery und neue Epoch befüllt. Der alte Empty-Kontext
  folgt der normalen Stale-Regel; die neue Epoch wird weiterhin durch genau
  eine federierte Multi-Search mit einer Subquery abgefragt;
- während eines Route-Shadow-Builds entsteht ein neuer logischer Shard: Alte
  aktive Epoch und ihr Cursor fragen weiter exakt ihre alte Membermenge ab,
  der Kandidat wird vor `ready` um ein neues generationenspezifisches Member
  erweitert, und der Swap gewinnt nur, wenn seine
  `route_radius_ux_v1`-Teilmenge zum Manifest am aktuellen Cutoff bijektiv ist.
  Ein vorhandenes `trail_default`-Member bleibt im vollständigen
  `member_manifest_hash`, zählt aber weder als zusätzlicher noch als
  widersprüchlicher Geo-Shard. Ein Rollback baut dieselben logischen
  Assignments auf vollständig neuen physischen IDs statt eine alte Familie zu
  reaktivieren;
- Mehrshard-Contracttest für `route_radius_ux_v1`: Eine logische federierte
  Multi-Search liefert globale Sortierung mit `trail_id ASC` als
  abschliessendem Tie-Breaker, zurückgegebene Seite, `totalHits` und korrekte
  `next_cursor`-Grundlage über genau
  die nach `query_family = route_radius_ux_v1` gefilterte
  Snapshot-Memberteilmenge. Fehlendes,
  doppeltes, zusätzliches oder abgeschnittenes Shardergebnis sowie ein
  Manifestwechsel zwischen Pre-/Post-Stempel verwerfen den Batch; eine
  Facetten-/Histogrammgruppe darf nur mit ihrem typisierten Gruppenfehler
  ausfallen und nie aus den übrigen Shards extrapolieren;
- Contract-Artefakt-Tests rekonstruieren den vollständigen
  `route_radius_ux_v1`-Payload ausschliesslich aus der Collection und prüfen
  jeden festgelegten Direct-, S50-, Densification-, Antimeridian-, r100-,
  Radius-, Capability-, Public-Spatial-Contract-, Engineimage-, Accuracy-,
  Count- und Unsupported-Sort-Wert. Sie prüfen insbesondere die bytegleiche
  Projektion gemeinsamer Geometry-/Simplification-/Segment-/Resolutionwerte,
  `default_radius_m = 2000`, die getrennte Spatial-Boundary-Accuracy-
  Contractform sowie die aufsteigend sortierte, nullbasierte lineare
  p95-Interpolation an Position `0.95 * (n - 1)`. Ein
  fehlendes Payloadfeld, unbekanntes Zusatzfeld, veränderte Canonicalization,
  falscher Hash oder ein anderes unqualifiziertes Engineprofil blockiert
  Build, Publish, Swap und Serving;
- Relation/Key/Hash-Negativtests variieren Generation-Queryfamily,
  Snapshot-Queryfamily, Generationenmember, Snapshot-Member und Attempt
  unabhängig: fehlender Owner, fremdes Artefakt, Family-/Membermischung,
  abweichender Queryfamily-Manifesthash, Marker ohne Artefakthash oder nur
  passende Schema-/Settings-/Bundlehashes führen fail-closed. Dieselben
  Negativfälle gelten beim dokumentleeren initialen Route-Member; dessen
  Vorhandensein darf eine fehlende Family-Autorität nicht verdecken. Ein
  `trail_default`-Member benötigt in V1 keine Route-Contractbindung und bleibt
  trotzdem Bestandteil des vollständigen Membermanifesthashes;
- Contract-Retentionstest stellt sowohl auf einer belegungsleeren als auch auf
  einer belegten Route-Family einen Cursor aus und erhöht die eigene
  `contract_artifact/artifact_key`-Ressource bis zu dessen `valid_until`.
  `issuing -> retained`, GC, Restore mit fehlenden Payloadbytes sowie ein
  `lost`-Übergang vor dem finalen Visibility-CAS werden gegen Resource-Owner,
  externes Checkpointmanifest und HWM geprüft; Generation-/Snapshot-HWM allein
  darf den Artefaktpin nicht ersetzen;
- Fence-Retirement gegen einen nur `query_fenced`-ten `ready`-Kandidaten muss
  scheitern; erst dessen physischer Apply-Beleg oder irreversible Entfernung
  aus der Swapmenge erlaubt Retirement, und kein späterer Swap darf die
  Restriktion verlieren;
- Trifft ein autorisierter Grant vor dem physischen Apply der alten
  Restriction ein, bleibt deren Gateway-Fence aktiv. Retirement und neue
  Sichtbarkeit sind erst erlaubt, wenn aktive sowie `ready`/swap-fähige
  Generationen den exakten Clear-Nachfolger physisch `applied` oder per
  vollständigem Baseline-/Coveragebeleg `subsumed` haben; der
  `clear_converged`-Retirement-Proof beendet dann den Queryguard und hält die
  historische terminale `fenced`-Delivery weiterhin rekonstruierbar gültig;
- Trifft derselbe Grant erst nach einem `restriction_converged`- oder
  `guard_release_converged`-Retirement ein,
  bleiben Fence, leere `clear_revision` und Retirement-Proof bytegleich; ein
  `grant_after_retirement`-Link und die normale neue Epoch tragen die
  Erweiterung. Beide Zeitordnungen müssen denselben aktuellen Suchbestand
  ergeben, ohne einen terminalen Fence umzuschreiben;
- F1 ist `restriction_converged`, Security-HWM plus Grace verstreicht und erst
  deutlich später trifft ein autorisierter Grant ein: Der GC-Versuch muss F1
  ohne bestehenden `grant_after_retirement`-Link behalten; Grant, Link und
  verlängerte Retention committen atomar und bleiben vollständig auditierbar;
- Derselbe späte-Grant-/GC-Test gilt nach `guard_release_converged`: Auch dieser
  Proof trägt weiterhin eine autoritative Restriction, erlaubt den ersten
  späteren `grant_after_retirement`-Link und bleibt bis dahin unabhängig von
  der Zeitgrenze reteniert;
- F1 erhält `G1 clear_before_retirement`, G1 wird nur teilweise ausgeliefert
  und vor F1-Retirement folgt Grant G2: G2 muss als
  `clear_advance_before_retirement` exakt am bisherigen Tail hängen, ein
  konkurrierender Fork am selben Predecessor scheitert, und Retirement bindet
  die vollständige Kette bis G2 sowie deren terminale Deliveries. Gewinnt der
  Retirement-CAS zuerst, läuft G2 als normale post-Retirement-Revision;
- Nach einem `SuccessorProgressProofV1` läuft der normale Cursor-HWM des
  verwendeten Content-Snapshots ab: Die physische Altgeneration darf gemäss
  ihren eigenen Gates gelöscht werden, Snapshot-/Epoch-/Member-/Delivery-
  Metadaten bleiben aber durch `retirement_artifact_manifest` gepinnt. Ein
  vorzeitiger Metadaten-GC-CAS muss am normalisierten Reverse-Pin verlieren;
  ein fehlender, zusätzlicher oder digestabweichender Pin blockiert bereits den
  Retirement-CAS. Erst gemeinsamer Fence-History-GC darf die Pins lösen;
- F1 pinnt beim Retirement den noch `satisfied`en F2 sowie aktive
  Generationen/Member; danach wechseln F2 nach `retired` und die Generation
  samt Membern geordnet bis `deleted`. Ihre aktuellen Recordversionen steigen,
  während F1s Pinprojektionen bytegleich und rekonstruierbar bleiben. Ein
  fälschlicher Whole-Record-Versionsvergleich darf weder diese Lifecyclekanten
  blockieren noch F1s historischen Proof invalidieren;
- Clear-Epoch publiziert, Retirement künstlich verzögert: Eine erste Seite
  endet nach begrenztem Retry mit `503 search_unavailable` und stellt weder
  Kontext noch Cursor aus; ein vor der Clear-Epoch ausgestellter Cursor wird
  `search_context_stale`. Retired der Fence zwischen Query und finalem
  Response-Visibility-CAS, muss dessen Kontrollstempel-CAS den gesamten Batch
  verwerfen. Erst eine vollständig nach dem Retirement begonnene oder
  wiederholte erste Seite darf einen neuen Kontext unter der erweiterten
  Sichtbarkeit ausstellen;
- Folge `F1 Restriction -> G1 Clear` nur teilweise ausgeliefert -> strikt
  spätere mindestens ebenso enge `F2 Restriction`: F2 und der
  `clear_superseded_by_restriction`-Link neutralisieren G1. F1 darf erst nach
  einer geschlossenen normalen F2-Content-Epoch mindestens auf der
  Successor-Revision mit typisiertem Proof retiren; ein bloss
  `query_fenced`-tes F2 genügt nicht. Trifft ein noch späterer Grant auf F2
  **vor** F1-Retirement ein, committet er auf F2s eigener Kette und eine
  normale geschlossene Grant-Epoch; F1s Kette und `clear_revision` bleiben
  bytegleich. F1 darf dann nur mit `SuccessorProgressProofV1` gegen F2s
  weiterhin wirksamen statischen Restriction-Guard retiren, bevor F2 unter
  seinem Clear-Proof folgt. Der Test wiederholt diese Reihenfolge auch über
  zwei transitive Successor-Fences und lässt jeden CAS bei zwischenzeitlich
  geänderter Tail-/Frontier-Version verlieren. Ein konkurrierender oder
  absichtlich zuerst versuchter
  Retirement-CAS für F2 muss scheitern, solange F1 nicht retired ist; nach
  `F1 -> retired` darf F2 unter seinem eigenen zweigkorrekten Proof retiren;
- Cursor-Race dieses Clear-Supersession-Zweigs: C wird nach F2s Securitycommit,
  aber vor der F2-Content-Epoch auf Snapshot `< F1.clear_revision` unter F1s
  alter Restriction ausgestellt. Die zwingende F2-Epoch beziehungsweise auf
  versiegelten Altgenerationen der retenierte F1-Clear-Anker macht C stale,
  bevor F1-Retirement die Sicht auf den gegenüber F1 möglicherweise weiteren
  F2-Stand umstellen kann. Nach Retirement ausgestellte Kontexte verwenden die
  F2-Epoch und sind nicht selbststale;
- Bootstrap-Swap sowie Recovery-Swap von `updating/recovery/quarantined` auf
  neue physische IDs; der Bootstrap aus einem Legacyowner injiziert Crashs vor
  und nach dem gemeinsamen Head-/Authority-CAS und beweist atomare,
  irreversible Exklusivität ohne Adoption eines alten physischen Index;
- der Legacy-Authority-Bootstrap akzeptiert nur Rollen, die vom V1-Entity-/
  Querymodell und vom neuen Membermanifest vollständig abgedeckt sind.
  Nicht unterstützte, unbekannte oder nur teilweise abgedeckte Mengen enden
  vor `prepared`, Generation, Head-, Authority- und Enginewirkung mit
  `state1_handoff_uid_unsupported`; ein falscher Reason oder widersprüchliches
  Handofffeldset endet ebenso wirkungslos mit
  `state1_handoff_binding_invalid`. Auf aktivem Head hat
  `state1_incremental_authority_handoff_unsupported` Vorrang;
- fehlende oder zusätzliche Boundaryproperties, unsortierte oder duplizierte
  Rollen/Ressourcen/Source-Keys, nicht bijektive Snapshotrollen, eine nicht
  auflösbare Source-Preimage sowie jeder Source- oder Ressourcenset-
  Digestfehler enden ebenfalls vor Generation-, Head-, Authority-, `prepared`-
  und Enginewirkung mit `state1_handoff_binding_invalid`;
- der STATE1-Generationsbuild erzeugt absichtlich Engine-Tasks nach dem
  statischen Legacy-Snapshot. Alle werden durch STATE1s eigene Attempt-,
  Submission- und Taskautomaten terminal bestätigt; der Precommit bindet
  danach ausschliesslich die erneut gelesene Source-/Authoritygrenze und
  vollständige Manifestabdeckung, keinen Legacy-Taskhead;
- Source-Revision, Source-Snapshot, Operationszustand, Authorityversion oder
  Legacyressourcenmenge ändern sich zwischen Preflight und Precommit. Es
  entsteht kein `prepared`-Record; die gebaute Generation wird samt Membern
  quarantänisiert und nach neuem Preflight vollständig auf neuen physischen
  IDs gebaut. Weder Gleichheitsadoption noch externer Source-Cutoff/-Replay
  sind zulässig;
- Crash oder Prozessverlust zu beliebigem Zeitpunkt vor Precommit behandelt
  auch eine vollständig gebaute staged Generation genauso: kein Resume und
  keine Adoption, sondern Quarantäne und vollständiger Neubau;
- zwischen `prepared` und Pointer-CAS kann unter intakten Source-/Authority-
  Locks keine Legacyänderung committen. Prozess- oder Lockverlust abortet nach
  Nichtcommitbeleg den alten Swap, quarantänisiert seine Zielgeneration und
  beginnt mit vollständigem Neubau beim Preflight; kein Test setzt in diesem
  Fenster eine normale konkurrierende Source-Mutation voraus;
- `authority_version = M-1` der unterstützten Rolle `trails` besteht Preflight
  und Commit-Guard und wird im gemeinsamen Handoff-CAS exakt zu `M`.
  `authority_version = M` derselben Rolle endet bereits vor Generation-, Head-,
  Authority-, `prepared`- und Enginewirkung vollständig mit
  `state1_authority_revision_exhausted`;
- Partial-Ownership mit einer STATE1-Trailrolle und weiterhin extern
  verwalteten Rollen: Actor-, Taxonomie- oder Trailänderungen persistieren im
  selben SQLite-Commit den ownerfremden Anteil sowie STATE1-Change/Dirty für
  alle Ziele in `T` und die unabhängigen Control-Changes `C`. Commitrollback,
  Crash direkt danach und beide Rennordnungen gegen den Handoff-CAS beweisen
  Recovery ohne Doppelung oder Lücke;
- Auch wenn `T = []`, erzeugt eine sichtbarkeitsverengende Fachmutation mit
  `C != []` weiterhin Katalogrevision, Security-Change, Fence und Checkpoint.
  Ein inhaltlicher Write ohne `T` oder `C` erzeugt dagegen keine leere
  STATE1-Katalogrevision;
- Cursor auf intakter `G1 sealed/published` nach Swap zu `G2`: Fällt nur die
  aktive G2 aus oder wird sie quarantänisiert, läuft der G1-Cursor unabhängig
  vom aktuellen Head normal weiter; nur neue erste Seiten sind unavailable;
- `abandoned`-Kandidat und nach Recovery aus dem Head entfernte
  `quarantined`-Generation erreichen nur nach ihren unterschiedlichen
  Attempt-/Resource-/HWM-Guards `retired -> deleting -> deleted`; ein
  unbekannter alter Enginewrite oder offener Cursorpin blockiert GC;
- Cursor unmittelbar vor, bei und nach `valid_until`, HWM-Crash und vorzeitig
  verlorenes Artefakt;
- zwei parallele HWM-Reservierungen mit gleichem oder verschiedenem
  Zeitmaximum, überlappenden Ressourcenmengen und umgekehrter externer
  CAS-Reihenfolge: Mutable Resource-Version und `issued_valid_until_hwm`
  verändern den semantischen Pre-/Post-Stempel nicht. Jeder finale CAS gewinnt
  mit unverändertem Ownerpaar, weiterhin zulässigem State und
  `current_hwm >= eigener valid_until`, auch wenn der andere Request den HWM
  inzwischen weiter erhöht hat; der höhere Checkpoint subsumiert beide und der
  kleinere kann ihn nicht überschreiben. Kontrollvarianten mit `retained` vor
  neuer Issuance beziehungsweise `lost/gc_eligible/deleting/deleted` nach der
  Reservation verwerfen die Response statt durch Self-HWM-Retries zu
  verhungern;
- Resource `issuing -> lost` zwischen HWM-Commit und externer Bestätigung:
  Auch wenn zuerst der ältere Anchor gewinnt, muss der finale lokale
  Visibility-CAS die Response verwerfen;
- eine verzögerte externe Checkpointbestätigung überschreitet `valid_until`:
  Der finale CAS liest die Uhr neu, stellt keine abgelaufene Response aus und
  ergibt bei Cursorfortsetzung `search_context_expired`, bei der ersten Seite
  nur einen vollständigen Neukontext-Retry beziehungsweise danach
  `search_unavailable`;
- Dirty-Queue-ABA nach Löschen/Neuerzeugen sowie Restore vor Securityrevision
  und Cursor-HWM;
- Restorefehler jeweils in `recovery_required`, `replaying` und `validating`
  führen nach `quarantined`; nur ein auditierter kompletter Neustart darf nach
  `recovery_required` zurück, nie direkt nach `healthy`;
- ein vor Restore akzeptierter verzögerter Enginetask, dessen Submissionrecord
  im PocketBase-Backup fehlt und dessen alter Marker zunächst noch passt, darf
  nach der Validierung nur alte quarantänisierte IDs verändern; die neue
  Recovery-Generation bleibt davon physisch unerreichbar und alte IDs werden
  ohne vollständigen Drain-/Barrierbeleg nicht gelöscht;
- ungesunde Uhr, nicht bestätigbarer externer Checkpoint und wiederholt
  verlorener Visibility-CAS ergeben gemäss Cursorpräzedenz die in Abschnitt 11
  festgelegten `search_unavailable`-/`search_context_unavailable`-Antworten;
- Gateway-Cutover mit widerrufenem altem Tenant-Key und negativem direkten
  Meili-Zugriff;
- ADR-0001-Tests für Deep Paging, Mutation-Soak, Retention, Rolling Deployment
  und Failover.

Für Revision/Publisher/Fence/Swap SOLL zusätzlich ein kleines ausführbares
Zustandsmodell oder ein äquivalenter Property-Test die Invarianten über alle
Interleavings prüfen.

### 13.3 Betriebsdiagnose

Mindestens sichtbar sind:

- `SEC-GLOBAL`-Circuit-Breaker, SEC-SCOPED-/SEC-TARGETED-Zustand, gebundener
  Proofhash und validierte Katalog-/Securityrevision;
- Legacy-Migrationslauf, Cutoff, Cursor, Catch-up-Lag, Zahl je terminalem
  Outcome und unklassifizierte Records je Origin/Scope;
- Inbox-/Outbox-Tiefe, ältestes Ereignis, Retry-/Quarantänezahl und terminale
  Deliveryzustände ohne Payload oder Empfänger als Metriklabel;
- Publication P1/P2 nach Zustand, offene Retracts und Tombstones sowie
  Authority-/Provenienzabweichungen als sichere aggregierte Diagnose;
- aktuelle Katalog-/Securityrevision und ältester unresolved Change;
- Dirty-Anzahl, ältester fälliger Job, Leasealter, Retry- und Dead-Zahl;
- Route-Shard-Allocatorzustand, Manifestrevision/-hash, Anzahl und Belegung je
  logischem Shard, aktueller offener Shard, Assignment-/Capacityverletzungen
  sowie fehlende Generation-/Snapshot-Memberbindungen; bei null Assignments
  werden initiale Shard-ID, Ordinal `0`, Empty-Assignment-Digest und die genau
  eine physische Memberbindung ausdrücklich angezeigt;
- Federationzustände, Head-/Epoch-Scope-Revision, nächste gebundene Expiry,
  Refreshlag und Quarantäne;
- aktive Generation, Epochzustand, Cutoff, Delivery-/Securitylag je Member;
- Route-Shard-Fan-out, Snapshot-Membermanifesthash und Mapping
  `geo_shard_id -> generation -> physical_index_id` ohne öffentliche Ausgabe
  der physischen IDs;
- Generation-/Snapshot-Queryfamily-ID, vollständiger Queryfamily-Manifesthash,
  Family-Bindinghash, Queryfamily-Membermanifesthash und `member_count`, auch
  für die belegungsleere `route_radius_ux_v1`-Family mit erwartetem
  `member_count = 1`;
- Route-Contract-Artefakt-Key/-Hash, Payloadschema, qualifiziertes
  Engineprofil, Owner-/Resourcezustand, Cursor-HWM und Bindungsabweichungen je
  Generation/Snapshot, ohne interne Payloadbytes als Metriklabels;
- aktive Fences und ältestes unbestätigtes Generationenziel;
- Engine-Attempts, Tasks und physische Submissions nach Art und sicherer
  Fehlerklasse;
- Resource-HWM, `retain_until` und früheste GC-Freigabe;
- Restore-Epoch/-Inkarnation, lokale/extern bestätigte Checkpointsequenz,
  Manifestdigest/-referenz, Checkpointlag, Gatewaymodus und unerfüllte
  Cutoverguards.

Metriken dürfen keine privaten Suchwerte, IRIs mit vertraulichen Parametern,
Cursorinhalte oder Enginepayloads als Labels verwenden.

## 14. Abhängigkeit und Änderungsregel

Schema- oder Laufzeitcapabilities für Federation-Ingest, Indexprojektion,
Revision/Dirty-Queue, Fences, Generationenaufbau, Publisher oder Gateway müssen
vor produktiver Aktivierung die von ihnen verwendeten Teile dieses Vertrags
implementieren und deren zugehörige Gates nachweisen. Die Annahme dieses
Dokuments behauptet weder, dass eine solche Capability implementiert ist, noch
dass alle beschriebenen Capabilities gemeinsam geliefert werden. Unabhängige
lokale UI- oder Filteränderungen unterliegen ihm nur, soweit sie eine seiner
Zustands-, Sichtbarkeits-, Count-, Pagination- oder Cursorinvarianten berühren.

Reihenfolge, Status, Zuschnitt konkreter Beiträge und Releasekoordination
werden ausschliesslich in
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
geführt. Aus diesem Vertrag folgt keine Personen- oder Teamzuordnung.

Änderungen an Zustandsmenge, Zeitsemantik, Revisionstransaktion, Fence-
Linearisierung, Watermarkbeweis, Swap-Commitpunkt, Cursorretention oder
Fehlerpräzedenz sowie an Route-Shard-Kapazität, Allocation-Key,
Initialshard-/Assignment-Unveränderlichkeit, Generation-/Snapshot-
Queryfamily- und Memberbindung oder dem kanonischen
Contract-Artefaktpayload benötigen eine explizite Vertragsänderung
beziehungsweise ein ADR. Betriebsparameter
innerhalb der hier gesetzten Grenzen dürfen über versionierte Konfiguration
kalibriert werden; die ADR-0002-Grenze von 1.000 Trails pro Route-Index ist
kein frei kalibrierbarer V1-Betriebsparameter.

## Referenzen

- [Federation-Sicherheits- und Publikationsvoraussetzungen v1](/develop/specs/trail-search/contracts/federation-security/)
- [Vertrag: Trail-Suche v1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [ADR 0001: Cursorzustand und Snapshot-Lifecycle](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/)
- [ADR 0002: GeoJSON Direct für den Routenradius](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
- [FED0-Quelle: Federation im materialisierten Suchbestand](/develop/specs/trail-search/shared-invariants/#6-search-context-counts-und-federation)
- [Konzept: Erweiterte Trail-Suche](/develop/specs/trail-search/)
- [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
