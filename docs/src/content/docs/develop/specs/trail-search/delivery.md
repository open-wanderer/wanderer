---
title: Liefer- und Beitragskatalog
description: Bau- und Freigabekarte für Prioritäten, Abhängigkeiten und eigenständig veröffentlichbare Schnitte der Trail-Suche.
editUrl: false
sidebar:
  order: 3
  badge: Arbeitsstand
spec:
  id: DELIVERY
  kind: delivery
  status: draft
  lastReviewed: '2026-09-19'
---

## Zweck und Leseanleitung

Dieses Dokument ist die aktuelle **Bau- und Freigabekarte** für das
[erweiterte Trail-Suchkonzept](/develop/specs/trail-search/). Es soll vier
praktische Fragen beantworten:

1. Welcher Nutzwert hat derzeit Priorität?
2. Welcher Baustein kann als Nächstes begonnen werden und was benötigt er?
3. Welche Arbeiten können parallel laufen?
4. Welche Bausteine und Security-Gates müssen erfüllt sein, bevor ein Ergebnis
   sichtbar veröffentlicht werden darf?

Die fachliche oder technische Detailumsetzung eines Bausteins gehört nicht in
diesen Katalog. Dafür gelten die verlinkten Verträge, ADRs und
[Work-Item-Spezifikationen](/develop/specs/trail-search/work-items/). Dieser
Katalog ordnet sie lediglich in eine Arbeits- und Freigabereihenfolge ein.

| Gesuchte Antwort | Massgeblicher Abschnitt |
| --- | --- |
| Welcher Nutzwert hat Priorität? | [Produktprioritäten](#produktprioritäten) |
| Was sollte als Nächstes gebaut werden? | [Baureihenfolge und Freigabegates](#baureihenfolge-und-freigabegates) |
| Was darf unabhängig oder parallel entstehen? | [Parallele, nicht blockierende Linien](#parallele-nicht-blockierende-linien) |
| Welche Bausteine gibt es und wovon hängen sie ab? | [Kandidatenkatalog](#kandidatenkatalog) |
| Was bildet zusammen ein nutzbares Release? | [Releases und Produktscheiben](#releases-und-produktscheiben) |
| Wo stehen die verbindlichen Details? | [Verbindliche Dokumente](#verbindliche-dokumente) |

### Bedeutung der Einträge

Die Codes bezeichnen mögliche Bausteinschnitte, keine zugesagten Pull Requests.
Bausteine dürfen bei der Umsetzung geteilt, zusammengelegt oder umbenannt
werden, solange Abhängigkeiten und Freigabegates nachvollziehbar bleiben.

Der Katalog ist weder Zeitplan noch Vollständigkeits- oder
Zuständigkeitsversprechen. Aufwand, Kapazität und konkrete Review-Schnitte
werden erst für einen tatsächlich übernommenen Beitrag bewertet. Die
Reihenfolge richtet sich nach Nutzerwert, Security-Gates, technischer Evidenz
und den verfügbaren Beiträgen.

## Produktprioritäten

Die folgende Liste ordnet den sichtbaren Nutzwert, nicht die technische
Baureihenfolge. Security-Voraussetzungen aus dem
[Security- und Publikationsvertrag](/develop/specs/trail-search/contracts/federation-security/)
bleiben davon unabhängig: Ein betroffener Baustein darf erst live gehen, wenn
seine einschlägigen Gates erfüllt sind. Interne oder noch nicht exponierte
Arbeit kann vorher beginnen.

Der sichtbare Nutzwert ist derzeit so priorisiert:

1. **Geo-Discovery:** Orte als Startpunkt suchen und Trails entlang einer
   gesamten Route finden. Beide Teile dürfen getrennt erscheinen, wenn jeder
   für sich nützlich und fachlich vollständig ist.
2. **Filterpanel-Redesign:** alle bestehenden Filter erhalten und Kategorien,
   Subkategorien, Sortierung, aktive Zustände und URL-State verständlich
   darstellen. Gespeicherte Suchen, Counts und Histogramme können danach als
   eigenständige Erweiterungen folgen.
3. **Wegbeschaffenheit und Wegtyp:** Routenabschnitte reproduzierbar und
   hinreichend aktuell analysieren, statt nur eine grobe Einmalberechnung
   anzubieten.
4. **Persönliche Trailschwierigkeit:** die Schwierigkeit aus Routenmerkmalen
   und wählbaren Leistungs- oder Ausflugsprofilen ableiten und später um
   weitere Achsen ergänzen.

Diese Prioritäten sind eine aktuelle Produktentscheidung, keine dauerhafte Reihenfolge des gesamten Katalogs. Neue Sicherheitsbefunde, Nutzerevidenz, technische Ergebnisse oder passende Community-Beiträge dürfen sie verändern.

## Baureihenfolge und Freigabegates

Entscheidungsstand: 19. September 2026.

Produktpriorität und Baureihenfolge sind bewusst verschieden: Geo-Discovery
bleibt das wertvollste nächste Produktziel. Zuerst entsteht jedoch eine
fachlich korrekte Regressionsbasis für die bestehende Suche. Darauf folgt der
gemeinsame Suchvertrag, den das einfachere Filterpanel als erster realer
Verbraucher erprobt. So muss die spätere Geo-Suche nicht gleichzeitig einen
neuen Suchvertrag, räumliche Suche, ACL, Federation, Cursor und Pagination
erstmals integrieren.

Die Arbeit muss nicht auf den Abschluss sämtlicher Security-Bausteine warten.
Nicht exponierte Implementierung darf gemäss ihren Abhängigkeiten parallel
laufen. Sichtbar veröffentlicht wird sie erst, wenn alle für den jeweiligen
Schnitt genannten Security-Gates nachweislich erfüllt sind.

### Sequenzielle Hauptlinie

1. **[SRCH0 abgenommen](/develop/specs/trail-search/work-items/search/srch0/)
   – eine korrekte Regressionsbasis herstellen.** Diese Abschlussmarke umfasst
   einen gemeinsamen Fixturekorpus und automatisierte Tests für Filter,
   Sortierungen, API-/URL-Zustände, Projektion, ACL-Kontexte und den Suchstartup.
   Historische Beobachtungen und stabile Case-IDs bleiben erhalten; aktive
   Erwartungen und unabhängige Eigenschaften prüfen das fachlich korrekte
   Verhalten. Bekannte Fehler im vereinbarten Umfang blockieren Merge und
   Abnahme. Produktkorrekturen dürfen getrennte PRs besitzen, müssen aber vor
   der SRCH0-Abnahme auf derselben Zielrevision integriert und grün sein.
   SRCH-V1, SRCH-COMP, SRCH2, SEC-VIS-0 und die späteren Indexbausteine
   übernehmen diese korrigierte Basis.
2. **[SRCH-V1](/develop/specs/trail-search/work-items/search/srch-v1/) – eine
   gemeinsame Suchsprache festlegen.** Request, Response, Normalisierung und
   URL-Zustand werden eindeutig beschrieben und gegen die SRCH0-Fälle geprüft.
   Auch dieser Schritt besitzt noch keinen Runtime-Cutover.
3. **[SRCH-COMP](/develop/specs/trail-search/work-items/search/srch-comp/) – den
   freigeschalteten Bestandsumfang auf dem heutigen Backend ausführen.** Ein
   serverseitiger Adapter übersetzt die dafür erlaubten typisierten Aufträge in
   Meilisearch und konsumiert den zuvor unabhängig durch IDX0 bereitgestellten
   Readinesszustand. Nach dem gemeinsamen Cutover erzeugt der Browser keine
   freien Engine-Ausdrücke mehr. Golden Fixtures beweisen, dass bestehende und
   neue Eingaben denselben unterstützten Suchauftrag ergeben. Dabei erhält
   SRCH-COMP die bereits korrigierte Radius- und Legacysemantik aus SRCH0;
   eigene Overlays beschreiben ausschliesslich neue Adaptersemantik.
4. **SRCH2 und
   [SRCH4a](/develop/specs/trail-search/work-items/search/srch4a/) – das
   Filterpanel auf realen Daten veröffentlichen.** Alle Bestandsfilter,
   Kategorien und Subkategorien, Sortierung, aktive Chips, responsive
   Darstellung und stabiler URL-Zustand laufen auf dem heutigen Index.
   Beispielzahlen und funktionslose Prototypfelder verschwinden. Counts und
   Histogramme folgen später als eigener Slice.
5. **Geo-Discovery – den Routenradius auf den erprobten Vertrag setzen.** Nach
   den im Kandidatenkatalog genannten Grundlagen führt diese Linie über G2 →
   G3a → G3b → L4. L4 benötigt einen normalisierten Punktanker, aber weder
   Photon noch typisiertes Autocomplete zwingend als Provider.

Die Verantwortungsgrenzen der frühen Bausteine sind bewusst getrennt:

| Baustein | Verantwortet |
| --- | --- |
| SRCH0 | korrekte Regressionsbasis, unveränderte historische Evidenz, stabile Case-IDs und Abnahme der integrierten Bestandskorrekturen |
| IDX0 | nichtdestruktiver Bootstrap der drei heutigen Legacyindizes, Offline-Rebuildmechanik, konkreter `SearchReadinessV1`-Produzent und einmalige Erstrollout-/Upgradeanweisung |
| SRCH-V1 | gemeinsame fachliche Suchsprache |
| SRCH-COMP | serverseitige Ausführung und Ablösung freier Browser-zu-Engine-Ausdrücke unter Erhalt der korrigierten SRCH0-Semantik |
| SRCH2 | bestehende Suchfelder und deren explizite Presence-/Unknown-Modellierung auf der korrigierten SRCH0-Basis sowie zusätzliche Subkategorien |
| SEC-VIS-0 | weitergehendes Sichtbarkeits- und Netzwerkcontainment sowie Umstellung der Listenaggregate auf lokale Relationen unter Erhalt der bereits korrigierten Suchbasis |

Nachgewiesene Bestandsfehler etwa bei Radius, Quellschwierigkeit oder
Sichtbarkeit werden nicht mehr bis zu diesen Nachfolgern aufgeschoben.
Ihre Korrekturen sind Voraussetzungen der SRCH0-Abnahme, ohne daraus eine
Rückabhängigkeit von SRCH0 auf SRCH-COMP, SRCH2 oder SEC-VIS-0 zu machen.
Neue Architektur, Modellierung und erweiterte Semantik bleiben bei den
jeweiligen Nachfolgern. Der heutige Remoteaufruf für Listenaggregate allein
beweist noch keinen ACL-Fehler; die generelle Umstellung auf ausschliesslich
lokale Aggregate bleibt SEC-VIS-0-Scope.

Aktueller Umsetzungsstand: `feat/srch0` enthält die begonnene Testsuite auf
Basis von `e9b7a8cad` vom 7. September 2026. Die Bestandskorrekturen aus der
Sammelreferenz `fix/srch0-findings` werden als einzelne fachliche Fixes mit
eigenen Regressionstests und PRs vorbereitet. Der erste Fix liegt lokal auf
`fix/search-radius-filter` (`398b45682`, direkt ab `dev` bei `c73966d6c`):
Radiuskorrektur und 24 gezielte Regressionstests sind geprüft, aber noch
nicht gepusht, als PR eingereicht oder in `dev` beziehungsweise `feat/srch0`
integriert. Der [SRCH0-Umsetzungsstand](/develop/specs/trail-search/work-items/search/srch0/#aktueller-umsetzungsbezug)
enthält die Prüfungsnachweise. Für `fix/search-index-startup` fehlt weiterhin
die Implementierung. SRCH0 bleibt `blocked`. Die historische Evidenz darf
unverändert auf einen fehlerhaften Stand verweisen; die aktive Abnahmesuite
darf ihn weder mit `knownViolation` noch mit `xfail` oder angepassten
Fehlererwartungen akzeptieren.

Die Darstellung zeigt nur die für diese Entscheidung wichtigen Kanten; die Tabelle enthält die vollständigen Abhängigkeiten:

```text
Such-/Panel-Linie:
SRCH0 Tests + integrierte Bestandskorrekturen -> SRCH0 Abnahme
SRCH0 Abnahme -> SRCH-V1 -> SRCH-COMP
SRCH0 Abnahme -> SEC-VIS-0
IDX0 -> SRCH-COMP
IDX0 -> SRCH2
SRCH-COMP + SEC-VIS-0 -> kontrollierter Bestandsadapter live
SRCH-COMP -> SRCH2 -> SRCH4a
SEC-VIS-0 + SRCH4a -> sichtbares Filterpanel
SRCH-COMP -> SRCH-SAVED (Implementierung)
SRCH4a + SEC-VIS-0 -> SRCH-SAVED sichtbar

Geo- und Betriebs-Linie:
G0 -> G1 -> M1 -> IDX3
A1 + G1       -> G2
G2 + IDX3     -> G3a
G3a + SRCH1b  -> G3b -> L4

Additive Aggregationen:
IDX3 + SRCH1b + SRCH2 + A0 -> SRCH3
SRCH3 + SRCH4a             -> SRCH4b
SRCH3 + G3b                -> G3b-AGG
```

### Parallele, nicht blockierende Linien

- **[IDX0](/develop/specs/trail-search/work-items/engine/idx0/) startet ohne
  Vorgänger.** Es ersetzt den ungeschützten Alltags-Startup-Rebuild, produziert
  die beiden Readiness-Endpunkte und liefert den Offline-Rebuild für SRCH2.
  Seine Gap-Referenz auf SRCH0 ist keine Implementierungsabhängigkeit. SRCH0
  verlangt bereits Datenerhalt beim Startup, abgeschlossene Initialisierung
  vor Suchfreigabe sowie geprüfte Fehler- und Retrypfade. Diese
  Bestandskorrekturen dürfen separat entstehen; der vollständige IDX0-Wrapper
  und seine Readiness-Control-Plane müssen erst vor SRCH-COMP beziehungsweise
  SRCH2 fertig sein.
- **[M1](/develop/specs/trail-search/work-items/engine/m1/) startet parallel zu
  SRCH0.** Es qualifiziert den unterstützten Meilisearch-Upgradepfad und wird
  als eigener Betriebsrelease veröffentlicht, nicht gemeinsam mit einem neuen
  Geo- oder UI-Feature.
- **[SRCH3](/develop/specs/trail-search/work-items/aggregations/srch3/) und
  [SRCH4b](/develop/specs/trail-search/work-items/aggregations/srch4b/) können
  nach dem erprobten Panelvertrag parallel zur Geo-Linie entstehen.** Counts
  und Histogramme sind ein eigener sichtbarer Slice und kein Gate für L4.
- **[SRCH-SAVED](/develop/specs/trail-search/work-items/search/srch-saved/)
  kann nach SRCH-COMP implementiert werden; sichtbar wird es erst nach SRCH4a
  und SEC-VIS-0.** Gespeicherte Suchen blockieren weder Geo-Discovery noch
  Counts, Histogramme oder das Engine-Upgrade.
- **[G3b](/develop/specs/trail-search/work-items/geo-discovery/g3b/) liefert den
  Kern des Routenradius:** Treffer, `total`, Filter, Sortierung und Pagination.
  [G3b-AGG](/develop/specs/trail-search/work-items/aggregations/g3b-agg/) ergänzt
  Counts und Histogramme später, ohne den Kernrelease zu blockieren.
- **[L3](/develop/specs/trail-search/work-items/place-search/l3/) kann mit jedem
  GEO2-konformen Provider erscheinen.** Diese Providerlinie blockiert L4
  nicht.

Nach Filterpanel und Geo-Discovery bleiben Wegbeschaffenheit/-typ und persönliche Trailschwierigkeit die nächsten priorisierten Produktbereiche. Beiträge dürfen unabhängige Linien früher voranbringen; eine sichtbare Freigabe überspringt jedoch nie ihre eigenen Gates.

## Regeln für einen veröffentlichbaren Schnitt

Eine schmale Veröffentlichung ist ausdrücklich zulässig, wenn sie ein eigenständiges Benutzerziel erfüllt, fachlich korrekt ist und ihre Grenzen ehrlich benennt. Nicht zulässig ist eine „Light-Version“, die den versprochenen Nutzen nur vortäuscht oder später einen widersprüchlichen Vertrag erzwingt.

Bereits vorhandene Filter dürfen bei keinem Rollout verschwinden. Ein interner Architekturwechsel muss Bestandsparität, Migration und Rollback nachweisen. Feature-Flags gehören zu einer fachlich vollständigen Produktscheibe, nicht pauschal zu einer Capability-Klasse.

## Kandidatenkatalog

Die Bausteine sind nach fachlichen Arbeitssträngen gruppiert. Abhängigkeiten
sind wichtiger als eine einzige fortlaufende Nummer; mehrere Stränge können
unabhängig oder parallel bearbeitet werden. Migrationen, Worker und Verträge
dürfen vor einer sichtbaren Produktscheibe integriert werden. Die Codes bleiben
veränderbare Kandidatenschnitte. In der Spalte **Abhängigkeit** sind nur
Verweise auf andere Tabellen verlinkt; Bausteine aus derselben Tabelle stehen
als normaler Text.

Die vier Capability-Klassen gruppieren verwandte Arbeit; sie sind keine
zeitlichen Phasen:

| Klasse | Bedeutung |
| --- | --- |
| **FOUNDATION** | Bestandsdaten, gemeinsame Verträge, Security, Suche, Index und wiederverwendbare Plugin-/Job-/Datasetgrundlagen |
| **DERIVED** | Reproduzierbare lokale Ableitungen aus Route, Metadaten und gewählten Profilen |
| **CONTEXT** | Räumliche und segmentbasierte Analysen sowie externe, zeitlich veränderliche Kontexte und deren Orchestrierung |
| **DIALOG** | Dialogbasierte Bedienung derselben typisierten Such- und Fachdienste |

## FOUNDATION

### Suche, Index und Federation

Die folgenden Tabellen trennen fachlich zusammengehörige Bereiche; sie sind
keine zusätzlichen zeitlichen Phasen. Die verbindliche Reihenfolge ergibt sich
weiterhin ausschliesslich aus der Spalte **Abhängigkeit**.

#### Suchvertrag und Bestandskompatibilität

| Baustein                                          | Inhalt                                                                                                                                                                                                                                                                                                         | Abhängigkeit                  | Ergebnis                                                |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- | ------------------------------------------------------ |
| <span id="baustein-srch0"></span>[SRCH0 – Regressionsbasis](/develop/specs/trail-search/work-items/search/srch0/) | Datierter, ausführbarer Testkorpus mit stabilen Case-IDs, unveränderter historischer Evidenz, korrekten aktiven Erwartungen und unabhängigen Eigenschaften | Implementierung: keine; Abnahme: bekannte Fehler im vereinbarten Umfang korrigiert, integrierte Zielrevision grün | fachlich korrekte Basis für spätere Änderungen; aktuell `blocked` |
| <span id="baustein-idx0"></span>[IDX0 – Suchindex-Bootstrap und Readiness](/develop/specs/trail-search/work-items/engine/idx0/) | entfernt den unbedingten asynchronen Startup-Wipe, stellt fehlende beziehungsweise unerwartet leere Legacyindizes kontrolliert her, bietet einen Offline-Rebuild, produziert beide `SearchReadinessV1`-Endpunkte und dokumentiert den einmaligen gefencten Erstrollout | keine | normale Neustarts lassen grüne Indizes unangetastet; SRCH-COMP erhält einen konkreten Readinessproduzenten |
| <span id="baustein-srch-v1"></span>[SRCH-V1 – Normativer Suchvertrag](/develop/specs/trail-search/work-items/search/srch-v1/) | versionierte gemeinsame Suchsprache mit Request, Response, Normalisierung, URL-Codec, Fehlern und Schemas | abgenommene SRCH0-Regressionsbasis | implementierbare gemeinsame Sprache ohne Runtime-Cutover |
| <span id="baustein-srch-comp"></span>[SRCH-COMP – Bestandsadapter](/develop/specs/trail-search/work-items/search/srch-comp/) | führt den freigeschalteten Bestandsumfang unter SRCH-V1 serverseitig auf dem heutigen Backend aus, konsumiert IDX0-Readiness und erhält die korrigierte SRCH0-Semantik; Overlays gelten für neue Adaptersemantik | Implementierung: SRCH-V1 und [IDX0](#baustein-idx0); Liveaktivierung: SEC-VIS-0 | kontrollierter Suchpfad für bestehende First-Party-Aufträge |
| <span id="baustein-sec-vis-0"></span>SEC-VIS-0 – sofortiges Containment          | begrenzt die erlaubte Sichtbarkeit, schliesst direkte Umgehungswege über Netzwerk und Tokens und besitzt die lokale Listenaggregatprojektion samt kontrollierter Migration; erhält die bereits korrigierte SRCH0-Suchbasis | abgenommene SRCH0-Regressionsbasis | die bestätigte Sichtbarkeitsklasse ist lokal eingedämmt |

#### Federationsvertrag und Zustandsmodell

| Baustein | Inhalt | Abhängigkeit | Ergebnis |
| --- | --- | --- | --- |
| <span id="baustein-sec-auth-0"></span>SEC-AUTH-0 – Authority-Profil               | immutable Object-to-Actor/Origin-Bindung, `same_origin_dereferenced_v1`, Actor-/Attribution-/Key-/IRI-Prüfung, absorbierender Tombstone und explizit deaktiviertes Forwarding                                                                                                                                | keine                         | autorisierbare Federation-Mutationen                   |
| <span id="baustein-fed0"></span>FED0 – Federation-Domainvertrag             | Authority-Profil anwenden; Public- und Direct-Adressierung, maschinenprüfbare Objektordnung/Currentness, Origin-/Actor-/ACL-Scope, Taxonomiemapping, Revocation und Duplikatvertrag als reine normative Domainentscheidung                                                                                     | SEC-AUTH-0                             | fachliche Grundlage der lokalen Materialisierung       |
| <span id="baustein-state1"></span>STATE1 – Federation-/Index-/Gateway-Zustand | [normativer Zustandsvertrag](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/): interne PocketBase-Collections, Safe-Integer-/Zeitmodell, Revisionstransaktion, Federation-, Dirty-/Tombstone-, Generationen-/Publisher-, Fence-, Pointer-/Rollback-, Retention-, Gateway- und Recoveryautomaten                | [SRCH-V1](#baustein-srch-v1), [SRCH-ADR1](#baustein-srch-adr1), FED0                | verbindliche Control-Plane vor Schemaimplementierung   |

#### Sichere Federation und Publikation

| Baustein | Inhalt | Abhängigkeit | Ergebnis |
| --- | --- | --- | --- |
| <span id="baustein-sec-vis-1"></span>SEC-VIS-1 (FED1a) – gemeinsamer Ingest      | ein Persistence-Service für Inbox, Announce, Remote-Resolve, Listen- und Parent-Fetch; Provenienz, vorbereitete Objektköpfe/Snapshots, Direct-/Public-Scope, TTL/Refresh, terminale Legacyklassifikation und fail-closed DB-Invariante                                                                         | [STATE1](#baustein-state1), [SEC-AUTH-0](#baustein-sec-auth-0)                       | einheitliches fail-closed Shadow-Kataloguniversum      |
| <span id="baustein-sec-auth-1"></span>SEC-AUTH-1 – Authority-Enforcement          | immutable Genesis-Bindung persistieren; typisierten Create/Update/Refresh/Delete-Dispatch auf allen Ingestpfaden erzwingen; fremde Authority und IRI-Reservierung abweisen; Prozessabbruchpfade aus Hooks entfernen                                                                                           | SEC-VIS-1, [SEC-AUTH-0](#baustein-sec-auth-0), [STATE1](#baustein-state1)           | autoritatives lokales Federation-Kataloguniversum      |
| <span id="baustein-fed-pub-1"></span>FED-PUB-1 – Publikationsidentität           | lokaler Trail und `federated_publication` trennen; Create/Update/Delete, P1-Tombstone, Republish als neue IRI P2 und Reaktions-/Linkfolge explizit modellieren                                                                                                                                                 | [STATE1](#baustein-state1), SEC-AUTH-1, [FED0](#baustein-fed0)                 | ehrlicher Public/Unpublish/Republish-Lifecycle         |
| <span id="baustein-sec-dur-1"></span>SEC-DUR-1 – durable Inbox und Outbox        | idempotente persistierte Inbox vor `202`, atomarer Effect-Commit, durable Outbox mit Retry/Backoff/Zustellzustand; eingehende lokale Restriction und ausgehende Mitteilung bleiben unabhängig                                                                                                                | SEC-AUTH-1, FED-PUB-1, [STATE1](#baustein-state1)            | crashfeste Annahme und zugesagte Zustellung            |
| <span id="baustein-sec-scope-1"></span>SEC-SCOPE-1 – scoped Visibility-Fence       | atomarer Fach-/Revision-/Fence-/Tombstone-Commit; `desired_revision`-Koaleszierung pro Local-/Origin-/Actor-/ACL-/Publication-Scope; bestätigte Gateway-/Generationen-Konvergenz; globaler Circuit-Breaker nur bei unbestimmbarem Scope                                                                        | [IDX2](#baustein-idx2), SEC-DUR-1, [STATE1](#baustein-state1)                 | skalierbare fail-closed Sichtbarkeitsverengung         |

#### Zielindex und produktiver Suchpfad

| Baustein | Inhalt | Abhängigkeit | Ergebnis |
| --- | --- | --- | --- |
| <span id="baustein-m1"></span>M1 – Verbindlicher Meili-Vertrag            | reale Ausgangsprofile 1.11.3 und 1.36.0 inventarisieren; unterstützten Upgradepfad samt möglichen Zwischenständen auf das durch G1/ADR 0002 qualifizierte 1.53.1-Profil testen; 1.44.0-Benchmarkevidenz einbeziehen; Startup-Preflight, Soak, Dump-/Upgrade-/Rollback-Runbook und Geo-/Facetten-Contracts; eigener Betriebsrelease mit Vorlauf | [G1](#baustein-g1) | ein unterstütztes Engineprofil und eine getrennt betreibbare Self-Hoster-Migration |
| <span id="baustein-srch-adr1"></span>SRCH-ADR1 – Cursor-/Snapshot-Lifecycle      | [ADR 0001](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/): HMAC-Stateless-Cursor, generationengebundene `revision_fenced` Content-Epoch, Retention versiegelter Altgenerationen sowie Deep-Paging-, Retention-, Deployment- und Failover-Gates                                                            | [SRCH-V1](#baustein-srch-v1)                                 | technische Cursor-/Snapshotentscheidung                |
| <span id="baustein-idx1"></span>IDX1 – Typisierte Suchprojektion            | `trail_search_projection`, Schema-/Input-/Projektionsrevision, Dokumenthash, unveränderliches Lieferartefakt und reiner Builder mit Golden-Tests                                                                                                                                                              | M1, [SEC-AUTH-1](#baustein-sec-auth-1), [SRCH0](#baustein-srch0) Bestandskorpus, [STATE1](#baustein-state1)            | persistiertes Read Model mit Bestandsparität           |
| <span id="baustein-idx2"></span>IDX2 – Revision, Dirty-Queue und Change-Log | inkarnationsgebundene monotone transaktionale `catalog_revision`; koaleszierende Claims/Leases; getrenntes lückenloses Change-/Tombstone-Log; Subsumption, vollständige Invalidation und Reconciliation                                                                                                               | IDX1, [SEC-AUTH-1](#baustein-sec-auth-1), [STATE1](#baustein-state1)                | driftfreie, replaybare Updates                         |
| <span id="baustein-idx3"></span>IDX3 – Shadow-Rebuild                       | IDX3-eigener temporärer Legacy-Handoff-Adapter; Keyset-/Byte-Batches, Task-Waits, Generationen, Cutoff-Replay samt Tombstones, fenced Epoch-Publisher, Delivery-/Security-Watermarks, `issued_valid_until`-High-Watermarks, versiegelte Altgenerationen, atomarer Generationen-Pointer-CAS und getrennt aufgeholter Rollbackkandidat | IDX2, [SEC-SCOPE-1](#baustein-sec-scope-1), M1, SRCH-ADR1, [STATE1](#baustein-state1) | sicherer Vollaufbau und konsistente Suchstände        |
| <span id="baustein-srch1"></span>SRCH1 – Ziel-Schema und Compiler            | führt denselben SRCH-V1-Vertrag auf Zielprojektion aus: Federation-Scope, gemeinsamer `search_context`, Capability-Revision und strikt allowlistetes `trail_hit_v1`; keine zweite Semantik                                                                                                                     | [SEC-AUTH-1](#baustein-sec-auth-1), IDX1, [SRCH-COMP](#baustein-srch-comp), [STATE1](#baustein-state1)      | ein fachlicher Vertrag für UI, App und Chat            |
| <span id="baustein-srch1b"></span>SRCH1b – Search-Gateway                     | HMAC-Cursor-Keyring und -Validierung samt Expand/Contract, dauerhafte Issuance-High-Watermarks, Tenant-ACL, scope-isolierte Fence-/Generation-/Epoch-Prüfung, Routing gepinnter Altgenerationen und Response-DTO-Allowlist; Meili-Netz abschotten und alte Search-/Tenant-Tokens widerrufen                       | SRCH1, [SEC-SCOPE-1](#baustein-sec-scope-1), IDX3, SRCH-ADR1, [STATE1](#baustein-state1) | sicherer produktiver Runtimepfad                   |
| <span id="baustein-acl1"></span>ACL1 – Actor-sichere Content-Overlays       | ADR/Lasttest und Umsetzung für scopegebundene `trail_search_acl_overlay`; autorisierte Vollmengen-Union, Deduplikation, Ranking, Counts, Widerruf und Negativ-Leaktests                                                                                                                                        | SRCH1b, IDX3, [SEC-SCOPE-1](#baustein-sec-scope-1)     | korrekte engere Asset-/Waypoint-Sichtbarkeit           |

#### Suchfunktionen und Oberfläche

| Baustein | Inhalt | Abhängigkeit | Ergebnis |
| --- | --- | --- | --- |
| <span id="baustein-srch2"></span>SRCH2 – Bestehende und billige Suchfelder   | auf dem heutigen Indexpfad: explizite Presence-/Unknown-Modellierung auf der korrigierten Quellschwierigkeitssemantik ergänzen; Startpunktmodus, Rangfolge, vererbte Taxonomie-/Waypoint-Terme, lokale Likes, Kategorien/Subkategorien und alle Bestandssortierungen erhalten | [SRCH-COMP](#baustein-srch-comp), [IDX0](#baustein-idx0), abgenommene SRCH0-Regressionsbasis | vollständige vorhandene Daten plus Subkategorien |
| <span id="baustein-a0"></span>A0 – Aktivitätsmodell                       | getrennte versionierte Familie, Disziplin und Unterstützungsclaim; lokales/föderiertes Mapping samt Rohwert, Presence, getrennten Auflösungszuständen und -quellen sowie Modell-Supportstatus; Walk/Hike/Cycle nach ADR 0003 als erster produktiver Modellumfang                                                  | SRCH2                         | stabile Schlüssel für p99-Domains und spätere Modelle |
| <span id="baustein-srch2a"></span>SRCH2A – Asset-Projektion                   | `has_photos`/Thumbnail gegen das kanonische Assetmodell; Link/Unlink und eigene Asset-/Waypoint-Sichtbarkeit invalidieren Kern beziehungsweise Overlay korrekt                                                                                                                                                 | SRCH2, [ACL1](#baustein-acl1), [BASE-A](#baustein-base-a) (Assets #948)      | belastbarer actor-korrekter Fotofilter                 |
| <span id="baustein-srch3"></span>SRCH3 – Counts, Ranges und Histogramme      | exhaustive disjunktive Counts, gruppenweise Fehler, ausgewählte Nulloptionen, versionierte Bucket-IDs, Presence/Unknown, feste Buckets, p99/Caps, 16/32/16-Limits, vollständiger Policy-Cache-Key sowie Federation-/Expiry-/Swap-Tests über 1'000 Treffer; räumliche Counts tragen den aktiven Spatial-Vertrag | [IDX3](#baustein-idx3), [SRCH1b](#baustein-srch1b), SRCH2, A0       | dynamische Zahlen exhaustiv für ihre jeweils berechnete Ergebnismenge; bei `predicate_accuracy: bounded_approximate` sichtbar qualifiziert |
| <span id="baustein-srch4a"></span>SRCH4a – Filterpanel-Redesign               | reale API-Daten, vollständige Bestandsparität, prominente Freitext-/Kategorie-/Ort-/Sortierwege, Kategoriehierarchie, „Weitere Filter“, aktive Chips, responsive und zugängliche Zustände sowie stabiler URL-State; Fake-Prototypfelder entfernen | [SRCH-COMP](#baustein-srch-comp), SRCH2, [SEC-VIS-0](#baustein-sec-vis-0)   | eigenständig nutzwertige Such-UI ohne Index-/Gateway-Cutover |
| <span id="baustein-srch-saved"></span>SRCH-SAVED – Gespeicherte Suchen und Default | owner-private normalisierte `TrailSearchSpecV1`, CRUD und Revisionen, getrennter `trail_list`-Default-Pointer, URL-vor-Default-Auflösung, Replace-Materialisierung, Reset/Restore und Reparaturzustand ohne stilles Verbreitern | [SRCH-COMP](#baustein-srch-comp); sichtbare Freigabe SRCH4a, [SEC-VIS-0](#baustein-sec-vis-0) | geräteübergreifende benannte Suchen und verlässlicher Listenstart ohne neuen Suchpfad |
| <span id="baustein-srch4b"></span>SRCH4b – Count-/Range-UI                    | echte Counts, Histogramme, offene Endbereiche, Lade-/Fehlerzustände und Request-Abbruch                                                                                                                                                                                                                        | SRCH3, SRCH4a                 | dynamische Filterzahlen als unabhängige Verbesserung   |

SRCH-V1 definiert die vollständige fachliche Semantik. SRCH-COMP implementiert
den freigeschalteten Bestandsumfang dieser Sprache auf dem heutigen Backend und
kapselt Legacyunterschiede. SRCH0 führt keinen neuen Suchpfad ein, setzt aber
die Integration seiner erforderlichen Bestandskorrekturen voraus. Spätere
Delivery-Owner erhalten die korrigierte Basis und aktivieren ausschliesslich
ihre neue Architektur oder erweiterte Semantik gegen SRCH0-Basisfall und
eigenes Ziel-Overlay unter ihren jeweiligen Releasegates; eine neue
öffentliche V1-Capability ist dadurch noch nicht geöffnet.
SRCH1 und SRCH1b verantworten später den öffentlichen Gatewaypfad auf dem
Zielindex samt seinen eigenen Security- und STATE1-Gates. Sie ergänzen
capability-gebundene Federation- und Count-Funktionen, ohne bereits
unterstützte V1-Aufträge fachlich zu verändern.

Die Security-Bausteine sind Freigabekanten, keine Planungsbarriere: Verträge,
UI, lokale Analyse, Geo-, Plugin- und Shadowarbeit dürfen parallel laufen. Jede
neue sichtbare Suchscheibe, die den heute gemischten lokalen/föderierten Index berührt,
wartet mindestens auf `SEC-VIS-0`; Public-Federation, föderierte Counts und neue
Federation-Scopes warten zusätzlich auf `SEC-VIS-1`, `SEC-AUTH-1`,
`SEC-DUR-1` und `SEC-SCOPE-1`.

Die vorgezogenen SRCH0-Produktkorrekturen beheben Fehler des bestehenden
Suchpfads und sind keine Aktivierung einer neuen Suchscheibe. Ihre
Integration setzt den eigenen Korrektheits- und Sichtbarkeitsnachweis voraus,
aber nicht den vollständigen späteren SEC-VIS-0-Rollout. Damit blockiert
das Securitygate nicht die Behebung bereits nachgewiesener Fehler.

### Plugin-Host und Datenbasis

| Baustein                           | Inhalt                                                                                                                                                                                                                                                      | Abhängigkeit                        | Ergebnis                                                    |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- | ----------------------------------------------------------- |
| <span id="baustein-base-a"></span>BASE-A – Assets                     | laufenden PR #948 integrieren; `assets` und `asset_library.v1` als zweites Domainmuster; lokale Uploads bleiben unverändert verfügbar                                                                                                                       | aktuelle Pluginbasis                | kanonisches additives Assetmodell                           |
| <span id="baustein-base-r"></span>BASE-R – Routing                    | vorhandene Routing-Plugin-Arbeit gegen den dann aktuellen Plugin-Host integrieren; Valhalla bleibt Default; Registry übernimmt auch `profile_introspect.v1` und `profile_prepare.v1`; ein gestapelter Review bleibt bei offener Host-Abhängigkeit möglich | BASE-A für gemeinsamen Host-Cutover | `routing`-Typ, Valhalla/BRouter und Auswahl-/Profilmechanik |
| <span id="baustein-plg1"></span>PLG1 – Typ-/Capability-Registry     | zentrale Registry, typabhängige Manifestvalidierung und migrationssichere Erweiterung um weitere Domain-Typen                                                                                                                                               | BASE-R                              | kein verteiltes Enum-/Schema-Hardcoding                     |
| <span id="baustein-plg2"></span>PLG2 – System-/Multi-Instanzen      | Scope `system` oder `user`, optionaler Owner, `instance_key`, Auswahl, Priorität/Fallback und UI                                                                                                                                                            | PLG1                                | Geocoder, mehrere regionale Feeds und Wetterinstanzen       |
| <span id="baustein-job1"></span>JOB1 – Durable Job Runtime          | interne DB-Queue, Desired-Fingerprint/Unique-Koaleszierung, atomare Claims/Leases, Retry/Backoff/Jitter, Heartbeat, Status, Abbruch sowie Zeit-/Ressourcenbudgets                                                                                           | keine                               | wiederverwendbare Crash-/Overlap-sichere Jobbasis           |
| <span id="baustein-data1"></span>DATA1 – Dataset-/Artefakt-Lifecycle | unveränderliche Snapshotmanifeste, Quelle/Revision/Datenzeitpunkt/Hash/Coverage, begrenzte Artefaktpläne, Streaming, atomare Aktivierung und referenzsichere Retention                                                                                      | JOB1                                | gemeinsame reproduzierbare Datasetbasis                     |
| <span id="baustein-routeattr"></span>ROUTEATTR – Match-Attribute-ABI     | für `type = routing` die Capabilities `map_match_attributes.v1` und `map_dataset_metadata.v1`: SDK/ABI, Hostvalidierung, Limits, Dataset-Provenienz, Connectorfreigabe für `/trace_attributes`/`/status` und Valhalla-Export auf bestehendem Trace-Matching | BASE-R, PLG1, DATA1                 | eindeutiger Routing-/Kartenstandvertrag für SURF1           |

BASE-A und BASE-R sind keine Blocker für die Kernsuche, Mobile oder Homepage. Ein gestapelter Review ist möglich, wenn gemeinsame Manifest-, Instanz- und Migrationsstellen klar ausgewiesen sind. Konkrete Branch-, Rebase- und Retarget-Schritte gehören in das jeweils aktuelle Arbeitsissue, nicht in diesen langlebigen Katalog.

## DERIVED

### Analyse, Profile und Ratings

Das gemeinsame Aktivitätsmodell A0 wird wegen der aktivitätsspezifischen Range-Domains bereits im Suchfundament geführt. Die Analyse-, Profil- und Rating-Arbeit baut auf denselben Familien-, Disziplin- und Mappingwerten auf und definiert keinen zweiten Modellschlüssel.

| Baustein                        | Inhalt                                                                                                                                              | Abhängigkeit                 | Ergebnis                                    |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------- | -------------------------------------------- |
| <span id="baustein-a1"></span>A1 – Analyse-Lifecycle    | `trail_analysis`, gemeinsame Segment-/Chainage-Basis, Geometry-Hash, Status/Fehler, aktive Version, Jobs, Backfill und Golden-GPX-Fixtures          | [A0](#baustein-a0), [IDX3](#baustein-idx3)                     | gemeinsame Fachbasis                         |
| <span id="baustein-a2"></span>A2 – Routenfakten         | Start/Ziel, Rundweg, min/max Höhe, Coverage, Projektionsfelder und Filter                                                                           | A1, [SRCH3](#baustein-srch3)                    | Form und höchster Punkt                      |
| <span id="baustein-a3"></span>A3 – Referenzdauer        | typisierte Rohzeiten und neutrale Komponentenmodelle für Walk/Hike/Cycle; Cycle-Disziplinen und -Kombinator aus ADR 0003, Qualität und `unknown` statt 0 | A1, A2                       | vergleichbare Basisdauer                     |
| <span id="baustein-a4"></span>A4 – Profil-Templates     | System-/Instanz-/Benutzerscope, Vererbung/Overrides, aktivitäts- und komponentenspezifische Leistungsfaktoren, Konfidenz, Einstellungen und Migration | [A0](#baustein-a0)                           | wiederverwendbare Teilnehmerprofile          |
| <span id="baustein-a5"></span>A5 – Ausflugsprofile      | mehrere Teilnehmer, Kind ohne Konto, tatsächliche Unterstützung, Ausrüstung, Gruppen-/Pausenmodell, URL-/API-State                                  | A4                           | „Allein“, „Mit Kind“, „Familie“              |
| <span id="baustein-a6"></span>A6 – Gruppen-ETA          | komponentenweises Gruppenmaximum und Familienkombinator für Walk/Hike/Cycle, statischer Schwellencompiler, ETA-Effekt-Hook, Sortierung, Counts und Erklärung | A3, A5, [SRCH3](#baustein-srch3)                | Dauer „für uns“                              |
| <span id="baustein-a7"></span>A7 – Demand und Eignung   | erweiterbares Evaluator-Framework; körperliche Walk-/Hike-/Cycle-Basisachsen, Gruppen-Maximum, technische AND-Regeln, Unknown/Coverage und Registrierungen durch C4/SURF6 | A2, A5, A6                   | persönliche Schwierigkeit ohne Erstellerbias |
| <span id="baustein-a8"></span>A8 – Gelernte Overrides   | Datenqualitätsfilter, robuster Summit-Log-Schätzer, Review/Undo und Konfidenz                                                                       | A4, A3                       | optionale benutzerspezifische Kalibrierung   |
| <span id="baustein-a9"></span>A9 – Metrics-Kalibrierung | Herzfrequenz/Leistung/Kadenz nach Einwilligung, Kontext- und Qualitätsfilter; keine stillen harten Profiländerungen                                 | A8, Roadmap Activity Metrics | zusätzliche evidenzbasierte Kalibrierung     |
| <span id="baustein-r1"></span>R1 – Ratings              | lokale Rating-Collection, Unique-Regel, Aggregate, gewichtete Sortierung, Filter und Moderation                                                     | [SRCH3](#baustein-srch3)                        | Bewertung getrennt von Likes                 |

## CONTEXT

### Geocoding und POI

| Baustein                                   | Inhalt                                                                                                                                                                                                                               | Abhängigkeit                                 | Ergebnis                                                        |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------- | --------------------------------------------------------------- |
| <span id="baustein-l1"></span>L1 – Nominatim-Compliance            | heutigen Proxy korrigieren: instanzweiter Limiter, Cache/Singleflight, Retry-After, Attribution, Privacy, Locale/Limit; Bestandsfilter bleibt verfügbar                                                                              | keine                                        | policy-konforme bestehende Ortssuche                             |
| <span id="baustein-place1"></span>PLACE1 – Place-Vertrag und Taxonomie | gemeinsame versionierte Scopes, Anchor-/Snapshot-DTOs, Attribution, Geometrie- und Herkunftsmodell für Geocoding und POIs                                                                                                            | [PLG1](#baustein-plg1)                                         | keine divergierenden Ortsmodelle                                |
| <span id="baustein-geo1"></span>GEO1 – `geocoding`-Typ               | Domain-Typ, Capabilities `search.v1`, `reverse.v1`, `area_geometry.v1`, Usage-Profile und kontrollierte Migration von `NOMINATIM_URL` in eine Systeminstanz                                                                          | L1, [PLG1](#baustein-plg1), [PLG2](#baustein-plg2), PLACE1                       | providerneutrale Geocoding-Basis                                |
| <span id="baustein-geo2"></span>GEO2 – Typisierte Capability-Qualifikation | providerneutrale Konformität für `autocomplete.v1` plus `category_filter`: Scope-Taxonomie, positive/negative Golden-Fixtures, Anchor-Snapshots, Deployment-/Importprofil sowie Ressourcen-, Update- und Readinessziele | GEO1, PLACE1 | harte Scope-Filter nur bei belegter Providerfähigkeit |
| <span id="baustein-geo-photon"></span>GEO-PHOTON – Photon-Referenzadapter  | optionaler First-Party-Adapter samt gepinnter Version und Importprofil; implementiert die von GEO2 qualifizierten Capabilities, ohne Sonderfall im Suchvertrag zu werden | GEO2 | betriebsfertige Referenzimplementierung, aber keine Systemvoraussetzung |
| <span id="baustein-l3"></span>L3 – Typisierte Startpunktsuche      | Scope-Chips, gruppierte Ergebnisse, Abbruch, Kartenbias und URL-State; bindet jeden Anker an den bestehenden `start_point`-Radius; aktiviert sich mit jedem GEO2-konformen Provider | GEO2, qualifizierter Adapter, [SRCH-COMP](#baustein-srch-comp), [SRCH4a](#baustein-srch4a) | „Linde + Unterkunft“ mit direkter Startpunktwirkung |
| <span id="baustein-l4"></span>L4 – Räumliche Routenverknüpfung     | zusätzlicher `route_geometry`-Modus für einen normalisierten Punktanker aus bestehender Submit-Suche oder `search.v1`; Anker serverseitig an G3b übergeben; Direct-Vertrag zentral erklären; Regionen erst nach eigenem Flächengate | PLACE1, L1 oder GEO1/`search.v1`, [G3b](#baustein-g3b), [SRCH1b](#baustein-srch1b) | gesamte Route ohne Photon-/Autocomplete-Abhängigkeit verknüpft |
| <span id="baustein-poi1"></span>POI1 – `pois`-Typ und Vertrag        | Capabilities `search.v1`, `dataset_sync.v1`, `detail.v1`, optional `route_corridor.v1`; Systeminstanzen und Datasetmetadaten auf dem gemeinsamen Place-Vertrag                                                                       | [PLG1](#baustein-plg1), [PLG2](#baustein-plg2), [DATA1](#baustein-data1), PLACE1                    | stabile Roadmap-Schnittstelle für POI-Quellen                   |
| <span id="baustein-poi2"></span>POI2 – First-Party Overpass          | begrenzter Snapshot-/Artefaktimport, Attribution, Aktualität, Gebietsabdeckung und kein Live-Overpass im Browser/SearchRequest                                                                                                       | POI1                                         | vollwertige OSM-POI-Quelle; Community-Plugins anschliessbar     |

L1 ist unabhängig veröffentlichbar. L3 ist bereits mit dem erhaltenen Startpunktfilter ein vollständiger Nutzwert und wartet nicht auf L4; es benötigt die Capability-Kombination aus GEO2, aber nicht den Photon-Adapter. Der Routenmodus L4 ist eine additive und von GEO2/GEO-PHOTON unabhängige Produktscheibe. Ein Punkt aus der bestehenden Submit-Suche genügt als Anker. Der öffentliche OSMF-Nominatim-Modus schaltet weder Autocomplete noch harte Kategorien frei, entfernt aber nie die vorhandene Submit-Suche oder ihre Verwendung für einen punktförmigen Routenradius.

### Geo und Anstiege

| Baustein                                      | Inhalt                                                                                                                                                                                                                                                                                                                                                       | Abhängigkeit       | Ergebnis                                            |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------ | ---------------------------------------------------- |
| <span id="baustein-g0"></span>G0 – Reproduzierbarer Geo-Harness       | `geobench`, exaktes Rohgeometrie-Oracle, GeoJSON-Parametersweep, Product-Audit, Contract-Smoke und dauerhafte Rohreports samt Dataset-/Harness-/Executable-/Image-Digest; getrennte harte Direct-Set-Konsistenz und `route_radius_ux_v1`-Qualifikation                                                                                                             | keine              | reproduzierbare Engine- und Produktevidenz           |
| <span id="baustein-g1"></span>G1 – ADR 0002 Punkt-Radius              | **abgeschlossen:** GeoJSON Direct, S50, maximal 5-km-Segmente, r100 ohne Padding, aggregiertes 99-%-UX-Gate, relatives Total-Count-p95 höchstens 1 %, zentrale Approximationserklärung, maximal 100 km, 1000er-Capacity-Shards, H3/exakt als erster Fallback; qualifiziert das 1.53.1-Zielprofil als Evidenz, operationalisiert aber noch keinen Betreiber-Upgradepfad; 10k-Evidenz angenommen, 50k/NAS optional | G0 | qualifizierter Spatial-Vertrag und Fallbackordnung   |
| <span id="baustein-g2"></span>G2 – Suchgeometrie                      | segmenterhaltende S50-GeoJSON-`MultiLineString`-/`LineString`-Suchprojektion aus A1-Chainage, Status/Coverage, geodätische Verdichtung auf höchstens 5 km, Antimeridian, Geometry-/Simplification-/Segmentierungs-/Vertragsversion, persistente `geo_shard_id`-Zuordnung und Backfill nach ADR 0002                                                                         | [A1](#baustein-a1), G1             | korrekte räumliche Repräsentation                    |
| <span id="baustein-g3a"></span>G3a – Spatial Runtime                   | separate generationengebundene Route-Indexfamilie mit höchstens 1'000 dauerhaft zugeordneten Trails pro append-orientiertem Shard, r100 ohne versteckten Exact-Finalizer, persistentes Manifest, atomare Add/Update/Delete-Projektion sowie Shadow-Build und Familien-Swap                                                                                  | G2, [IDX3](#baustein-idx3)           | produktionsfähige Ausführung des Direct-Vertrags     |
| <span id="baustein-g3b"></span>G3b – Search-Integration Core           | eine föderierte Meilisearch-Multi-Search über die snapshotgebundene Shardfamilie; ACL, Federation, Text-/Bestandsfilter, globale Sortierung, exaktes `total` der Direct-Menge, Pagination, Cursor, Context-/Cache-Felder, zentrale Limitations-Dokumentation und typisierter 4xx für Route-Proximity in UI, Compiler, URL, API und Chat | G3a, [SRCH1b](#baustein-srch1b) | vollständiger räumlicher Punkt-Radius ohne Aggregationspflicht |
| <span id="baustein-g3b-agg"></span>G3b-AGG – Räumliche Aggregationen       | SRCH3-Facetten und -Histogramme auf denselben Spatial-, Snapshot-, ACL-, Federation-, Budget- und Fehlervertrag wie G3b binden; keine zweite Kandidaten- oder Count-Semantik | G3b, [SRCH3](#baustein-srch3) | additive räumliche Optionscounts und Histogramme |
| <span id="baustein-g3f"></span>G3f – Bedingter H3-/Exact-Ersatz        | nur falls GeoJSON Direct später seinen verbindlichen Vertrag oder Regressionstest nicht mehr erfüllt: H3 plus vollständige Geometrie-Hydration und exakte Prüfung als eigener kompletter Vertical Slice; neue Generation und eigener Contractlauf, kein automatischer Fallback pro Anfrage                                                                                          | gescheiterter Direct-Vertrag, G3b | vollständiger, expliziter Ersatzpfad                 |
| <span id="baustein-g3c"></span>G3c – Optionaler Exact-/Proximity-Modus | nur nach belegtem Nutzerbedarf: eigener Safe-/Exact-Orchestrator mit vollständiger Kandidatenprüfung, mathematisch exakten Spatial-Counts und Linien-Proximity-Sortierung sowie eigener UI-/API-Capability                                                                                                                                                          | G3b               | additive Exaktheit ohne Blockade des Direct-Produkts |
| <span id="baustein-c1"></span>C1 – Höhen-/Segmentanalyse              | richtungssensitives Resampling, Glättung, Coverage, Version und Detailpersistenz                                                                                                                                                                                                                                                                                                  | [A1](#baustein-a1)                                  | reproduzierbare Höhenbasis                            |
| <span id="baustein-c2"></span>C2 – Anstiegserkennung                  | Tal/Gipfel, Merge-Regeln, Profile, Rollups und Same-Climb-Compound-Tokens                                                                                                                                                                                                                                                                                                         | C1                                  | indexierbare Anstiege                                 |
| <span id="baustein-c3"></span>C3 – Objektives Anstiegsprodukt         | Filter, Counts, Höhenprofil und Erklärung der isolierten Anstiege                                                                                                                                                                                                                                                                                                                 | C2, [SRCH3](#baustein-srch3)                           | vollständiger Ersatz für „Steilheit“                  |
| <span id="baustein-c4"></span>C4 – Climb-Profilintegration            | registrierte A7-Anstiegsachsen und unabhängig versionierte A3-Referenzmodellkomponente                                                                                                                                                                                                                                                                                            | C3, [A3](#baustein-a3), [A7](#baustein-a7)                          | additiver Einfluss auf ETA und persönliche Eignung    |

G3f ist kein stiller Runtime-Fallback. Er beginnt erst nach einer neuen Architekturentscheidung, ersetzt Projektion und Runtime als vollständigen Vertical Slice und wird mit denselben fachlichen Nullfehlern, Snapshotregeln und einem eigenen Contractlauf qualifiziert. Optionale 50k-/NAS-Profile dürfen diesen Entscheid informieren, werden dadurch aber nicht rückwirkend zum Release-Gate von ADR 0002.

### Wegbeschaffenheit und Wegtyp

| Baustein                                | Inhalt                                                                                                                                                                                         | Abhängigkeit         | Ergebnis                                            |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------- | --------------------------------------------------- |
| <span id="baustein-surf0"></span>SURF0 – ADR und Regressionen      | Taxonomie, Filter-/Coverage-Vertrag und alpine, MTB-, Parallelweg-, Out-and-back- und Multi-Segment-Fixtures; Testweg `39669166`                                                               | [A0](#baustein-a0)                   | gesicherte Semantik aus altem POC                   |
| <span id="baustein-surf1"></span>SURF1 – Match-Attribute-Analyse   | konsumiert bei `type = routing` die Capability `map_match_attributes.v1`; besitzt Analyseprofile, A1-Chunking, Overlap-Deduplikation, Chainage, Coverage und Matchmetriken                     | [ROUTEATTR](#baustein-routeattr), SURF0, [A1](#baustein-a1) | belastbares Map-Matching                            |
| <span id="baustein-surf2"></span>SURF2 – OSM-Attributauflösung     | Datasetmanifeste, Snapshot-/Way-ID-Cache, Rohwertparser, Normalisierung, Attribution und lokaler Produktionsimport beziehungsweise fähiger Routing-Adapter                                     | SURF1, [DATA1](#baustein-data1)         | vollständige statt grober Surface-Daten             |
| <span id="baustein-surf3"></span>SURF3 – Persistierte Analyse      | unveränderliche Runs/Input-Fingerprints, Geometry-Hash, Sollrevision/aktiver Run, getrennte Snapshotreferenzen/Coverage, Zeit-/Freshnessfelder, Rollups, Backfill und atomarer Versionswechsel | [A1](#baustein-a1), SURF2, [IDX3](#baustein-idx3)      | reproduzierbare Wegbeschaffenheit                   |
| <span id="baustein-surf3r"></span>SURF3R – Freshness und Refresh    | DB-Queue, fairer Cron-Sweep, Leases und Aktivierungs-CAS, Snapshot-Pinning, Prioritäten, Batch-/Laufzeitbudget, Retry, quellengebundenes `usable_until`, Admin-Diagnose und Kapazitätsgate     | SURF3, [JOB1](#baustein-job1), [DATA1](#baustein-data1)   | dauerhaft aktuelle statt einmalig berechnete Daten  |
| <span id="baustein-surf3d"></span>SURF3D – Delta-Invalidierung      | optionale OSM-Replikationsdiffs sowie Way-/Korridor-Reverse-Index; identische Semantik, verpflichtend nur wenn der TTL-Sweep das Freshness-SLA nicht schafft                                   | SURF3R, [DATA1](#baustein-data1)        | gezielte schnellere Aktualisierung                  |
| <span id="baustein-surf4"></span>SURF4 – Suchvertrag               | Projektionsfelder, Snapshot-/`as_of`-/Expiry-Bedingung, Anteil-/Skalenfilter, disjunktive Counts sowie Unknown-/Coverage-Semantik                                                              | SURF3R, [SRCH3](#baustein-srch3)        | korrekte Filterbasis                                |
| <span id="baustein-surf5"></span>SURF5 – Objektive Darstellung     | getrennte Dimensionsbalken, Coverage, Höhenprofilabschnitte, Quellen und Konfidenz in Web/App                                                                                                  | SURF4, [C1](#baustein-c1)            | vollständiger objektiver Surface-Slice              |
| <span id="baustein-surf6"></span>SURF6 – Modell-/Profilintegration | registrierte Surface-/Technikachsen für A7, unabhängig versionierte A3-Modellkomponente, Profileignungs-UI und reiner `surface_condition_effect`-Vertrag                                       | SURF4, [A3](#baustein-a3), [A7](#baustein-a7)        | zusätzliche persönliche und kontextuelle Auswertung |

SURF0 bis SURF5 einschliesslich SURF3R bilden gemeinsam den objektiven Surface-Slice; es gibt dabei keinen öffentlichen Zwischenzustand „grobe Valhalla-Oberfläche heute, echte OSM-Semantik später“ oder „einmal berechnet, danach unbemerkt veraltet“. SURF3D optimiert dieselbe zugesagte Aktualität und wird nur zum Release-Gate, wenn der begrenzte Sweep das Kapazitätsziel nicht erfüllt. SURF6 ist ein unabhängiger, späterer Mehrwert und blockiert Filter, Segmentdarstellung und Höhenprofil nicht.

### Zugang, Transit, Wetter und Orchestrierung

| Baustein                                 | Inhalt                                                                                                                                                                                                        | Abhängigkeit                | Ergebnis                                                |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- | ------------------------------------------------------- |
| <span id="baustein-acc1"></span>ACC1 – Zugangsdomain               | gemeinsames Endpunkt-/Modus-/Gehweg-/Coverage-Modell für ÖV, Auto- und Veloparkplätze                                                                                                                         | [A2](#baustein-a2)                          | stabile Basis ohne Behauptung konkreter Verfügbarkeit   |
| <span id="baustein-trn1"></span>TRN1 – `transit`-Typ               | Domain-Typ mit `dataset_sync.v1`, `realtime_sync.v1`, `journey_plan.v1`; regionale Instanzen/Auswahl sowie `dataset_ref`, Feed-/Datasetversion, Agency-/Namespace-Mapping und atomare Realtime-Kompatibilität | [PLG1](#baustein-plg1), [PLG2](#baustein-plg2), [DATA1](#baustein-data1)           | offene Fahrplan-Schnittstelle statt Anbieter-Hardcoding |
| <span id="baustein-trn2"></span>TRN2 – Statischer Transit          | First-Party-GTFS-Schedule-Plugin, atomarer Import, Stops/Modi, planmässige Bedienung, Start/Ziel, Gehwege, Persistenz, Filter, Counts und UI                                                                  | TRN1, ACC1, [SRCH3](#baustein-srch3)           | vollständige statische ÖV-Erreichbarkeit                |
| <span id="baustein-trn-jp"></span>TRN-JP – Journey-Provider          | mindestens ein produktionsgeeigneter First-Party-Adapter für `journey_plan.v1`; regionale Coverage, Stop-/Agency-Mapping, Auth/Quota, Fehlernormalisierung und Contract-Tests                                 | TRN1                        | ausführbare Live-Routing-Capability                     |
| <span id="baustein-poi3"></span>POI3 – Parken und POI-Zugang       | Parkplatz/Veloparkplatz, öffentlicher Zugang, Distanz, Aktualität, Persistenz, Filter, Counts und UI auf Basis des POI-Snapshots                                                                              | [POI2](#baustein-poi2), ACC1, [SRCH3](#baustein-srch3)           | vollständiger statischer Parkplatz-/POI-Mehrwert        |
| <span id="baustein-ctx1"></span>CTX1 – Orchestrator-Kern           | vollständiger Kandidatenstrom, Budget-/Abbruchvertrag, reproduzierbare Revision und Registry unabhängiger Kontext-Evaluatoren                                                                                 | [SRCH1b](#baustein-srch1b), [IDX3](#baustein-idx3), [A3](#baustein-a3)            | gemeinsame dynamische Auswertungsbasis                  |
| <span id="baustein-ctx2"></span>CTX2 – Exakte Kontextsuche         | Zeit-/Routeniteration, globale Filter/Sortierung, Counts, Histogramme, Pagination sowie Unknown-/Konvergenzvertrag für jeweils aktive Evaluatoren                                                             | CTX1, [SRCH3](#baustein-srch3)                 | keine First-page-Näherung oder Evaluator-Kopplung       |
| <span id="baustein-trn3"></span>TRN3 – Live-Verbindungen           | `journey_plan.v1`, Datum/Zeit/Zeitzonen, Servicefehler, Erklärungen und UI als registrierter CTX2-Evaluator                                                                                                   | TRN2, TRN-JP, CTX2, [A6](#baustein-a6)      | konkrete Fahrplanverbindungen                           |
| <span id="baustein-wea1"></span>WEA1 – `weather`-Typ               | Domain-Typ mit `forecast.v1`, `seasonal_conditions.v1`, optional `observations.v1`; räumlich-zeitlicher Cache sowie Quellen-/Gültigkeits-/Unsicherheitsvertrag                                                | [PLG1](#baustein-plg1), [PLG2](#baustein-plg2), [JOB1](#baustein-job1)            | offene Wetter-Schnittstelle                             |
| <span id="baustein-wea-f"></span>WEA-F – Forecast-Provider          | mindestens ein produktionsgeeigneter First-Party-Adapter für `forecast.v1`, inklusive Coverage, Modelllauf/Gültigkeit, Auth/Quota, Fehlernormalisierung und Contract-Tests                                    | WEA1                        | ausführbare Prognose-Capability                         |
| <span id="baustein-wea-s"></span>WEA-S – Saison-Provider            | mindestens ein produktionsgeeigneter First-Party-Adapter beziehungsweise Datasetimport für `seasonal_conditions.v1`, inklusive Coverage, Aktualität, Attribution und Contract-Tests                           | WEA1, [DATA1](#baustein-data1)                 | ausführbare Saison-Capability                           |
| <span id="baustein-sea1"></span>SEA1 – Saisonprodukt               | Klimatologie, Schnee, Exposition, Tageslicht, vollständige Filter-/Count-/Sortierintegration und UI als eigener CTX2-Evaluator                                                                                | WEA-S, [A1](#baustein-a1), [C1](#baustein-c1), [A6](#baustein-a6), [A7](#baustein-a7), CTX2 | typische Bedingungen als vollwertige Produktscheibe     |
| <span id="baustein-wea2"></span>WEA2 – Prognoseprodukt             | Wind, Temperatur, Niederschlag, Route-/Zeititeration, vollständige Filter-/Count-/Sortierintegration und UI als eigener CTX2-Evaluator                                                                        | WEA-F, [A1](#baustein-a1), [C1](#baustein-c1), [A6](#baustein-a6), [A7](#baustein-a7), CTX2 | konkrete Prognose als vollwertige Produktscheibe        |
| <span id="baustein-wea3"></span>WEA3 – Surface-Zustandserweiterung | Schlamm-, Eis- und Nasspflastereffekte über den reinen `surface_condition_effect`-Vertrag; eigene Coverage/Unknown-Erklärung                                                                                  | WEA2, [SURF6](#baustein-surf6)                 | zusätzliche Surface-Wetter-Interaktion                  |

TRN2, POI3, SEA1 und WEA2 sind voneinander unabhängige Releases. „Statisch per Bahn und Bus erreichbar“ ist keine Light-Version einer Live-Verbindung, sondern eine andere, klar benannte Aussage. Ebenso ist WEA2 ohne WEA3 vollständig für die zugesagten Wind-/Temperatur-/Niederschlagsdimensionen und behauptet keinen Untergrundeffekt.

## DIALOG

### Chat und MCP

| Baustein                               | Inhalt                                                                                                                            | Abhängigkeit   | Ergebnis                                          |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | -------------- | ------------------------------------------------- |
| <span id="baustein-q1"></span>Q1 – Toolvertrag/Discovery/Evals | typisierte read-only Tools, JSON-Schemas, Capability Discovery, mehrsprachiges Eval-Corpus und Prompt-Injection-Fixtures          | [SRCH-COMP](#baustein-srch-comp)      | testbarer Agentvertrag für den aktiven Suchumfang |
| <span id="baustein-q2"></span>Q2 – Modellprovider/Privacy      | lokaler/entfernter Modelladapter, Konfiguration, Einwilligung, Retention, Limits und Audit                                        | Q1             | self-hosting-taugliche Laufzeit                   |
| <span id="baustein-q3"></span>Q3 – Read-only Chat              | Dialogorchestrierung, editierbare Chips, verfügbare Profil-/Ortswahl, Suche und Trail-Karten; nicht verfügbare Absichten ablehnen | Q1, Q2, [SRCH4a](#baustein-srch4a) | nützliche Dialogsuche ohne Abhängigkeit von CONTEXT          |
| <span id="baustein-q4"></span>Q4 – Erklärung und Vergleich     | `explain_match`, `compare_trails`, Quellen/Revisionen und Unknown-Erklärung für alle jeweils registrierten Fachdaten              | Q3             | belastbare Antworten statt Textfantasie           |
| <span id="baustein-q5"></span>Q5 – MCP-Serveradapter           | dieselben read-only Wanderer-Tools pro Benutzer für externe Clients; Auth/Scopes, keine duplizierte Suchlogik                     | Q1             | optionale externe Nutzung ohne internen Chat      |
| <span id="baustein-q6"></span>Q6 – Bestätigte Schreibaktionen  | gespeicherte Suche oder Default über SRCH-SAVED ändern, Liste ändern, Profil anpassen; eigener Mutation-Toolvertrag, Preview, Bestätigung und Audit | Q3; `save_search`/`set_default_search` zusätzlich [SRCH-SAVED](#baustein-srch-saved) | sicherer aktiver Assistent                        |

## Releases und Produktscheiben

Die kleinste Releaseeinheit besitzt ein erkennbares Benutzer- oder Betreiberziel
und ist fachlich vollständig. Ein Capability-Bereich muss dafür nicht insgesamt
fertig sein; ein rein internes Teilstück ist umgekehrt noch kein Release. Die
Tabellen zeigen den sichtbaren Nutzen und die Abhängigkeiten in beide
Richtungen: **Benötigt** nennt die Voraussetzungen eines Releases;
**Voraussetzung für** nennt die wichtigsten Releases, die anschliessend darauf
aufbauen und nicht mehr zu diesem Release selbst gehören. Bedingungen für den
Livegang stehen ebenfalls unter **Benötigt**; zusätzlich gelten immer die
[gemeinsamen Freigaberegeln](#gemeinsame-freigabe--und-rollbackregeln). `—`
bedeutet, dass keine hier aufgeführte spätere Produktscheibe davon abhängt.
Jedes Bausteinkürzel in diesem Abschnitt führt direkt zu seiner Beschreibung im
Kandidatenkatalog.
Vollständige transitive Abhängigkeiten bleiben im
[Kandidatenkatalog](#kandidatenkatalog).

### Betrieb

| Release | Ergebnis | Benötigt | Voraussetzung für |
| --- | --- | --- | --- |
| Legacy-Suchbootstrap ([IDX0](#baustein-idx0)) | Vorhandene grüne Indizes werden beim Neustart nicht geleert; fehlende beziehungsweise unerwartet leere Indizes werden vor der Suchfreigabe wiederhergestellt, beide Readiness-Endpunkte besitzen einen konkreten Produzenten, und der erste Bestandsrollout ist mit Wrapper, Fehlerpfad und Retry dokumentiert. | keine | [SRCH-COMP](#baustein-srch-comp) und [SRCH2](#baustein-srch2) |
| Meilisearch-Upgrade ([M1](#baustein-m1)) | Ein geprüfter und dokumentierter Upgrade-, Rollback- und Wiederherstellungspfad auf das unterstützte Engineprofil. | [G1](#baustein-g1) | [IDX1](#baustein-idx1) und [IDX3](#baustein-idx3) sowie damit die Zielindex- und Geo-Linie |

[M1](#baustein-m1) wird mit Vorlauf als eigener Betreiberrelease und nicht gekoppelt an einen
UI-Rollout veröffentlicht.

### Suche auf vorhandenen Daten

| Release | Ergebnis | Benötigt | Voraussetzung für |
| --- | --- | --- | --- |
| [L1](#baustein-l1)-Compliance | Die bestehende Ortssuche erfüllt die Nominatim-Vorgaben. | keine weiteren Bausteine in diesem Abschnitt | [GEO1](#baustein-geo1); alternativ kann [L1](#baustein-l1) den Punktanker für [L4](#baustein-l4) liefern |
| Filterpanel ([SRCH4a](#baustein-srch4a)) | Bestandsfilter, Subkategorien und Sortierung funktionieren mit verständlichem, stabilem URL-State. | [SRCH-COMP](#baustein-srch-comp), [SRCH2](#baustein-srch2) und [SEC-VIS-0](#baustein-sec-vis-0); korrigierte SRCH0-Basisfälle und die jeweiligen Erweiterungs-Overlays sind grün | [SRCH-SAVED](#baustein-srch-saved), [L3](#baustein-l3), [SRCH4b](#baustein-srch4b) und [Q3](#baustein-q3) |
| Gespeicherte Suchen ([SRCH-SAVED](#baustein-srch-saved)) | Benannte Suchen und ein verlässlicher Default für `trail_list`; URL und History gewinnen, ungültige Bedingungen führen in einen Reparaturzustand statt zu einer breiteren Suche. | [SRCH-COMP](#baustein-srch-comp); für den Livegang [SRCH4a](#baustein-srch4a) und [SEC-VIS-0](#baustein-sec-vis-0) | Speicher- und Defaultaktionen in [Q6](#baustein-q6) |
| Fotofilter ([SRCH2A](#baustein-srch2a)) | Trails lassen sich zuverlässig nach sichtbaren Fotos filtern. | [BASE-A](#baustein-base-a) (Assets #948), [SRCH2](#baustein-srch2) und [ACL1](#baustein-acl1) | — |
| Counts und Range-Histogramme ([SRCH3](#baustein-srch3)) | Die Suche liefert belastbare Optionszahlen und numerische Verteilungen. | [IDX3](#baustein-idx3), [SRCH1b](#baustein-srch1b), [SRCH2](#baustein-srch2) und [A0](#baustein-a0); für föderierte Counts zusätzlich [SEC-VIS-1](#baustein-sec-vis-1), [SEC-AUTH-1](#baustein-sec-auth-1), [SEC-DUR-1](#baustein-sec-dur-1) und [SEC-SCOPE-1](#baustein-sec-scope-1) | [SRCH4b](#baustein-srch4b), [A2](#baustein-a2), [A6](#baustein-a6), [R1](#baustein-r1), [C3](#baustein-c3), [G3b-AGG](#baustein-g3b-agg), [SURF4](#baustein-surf4), [TRN2](#baustein-trn2), [POI3](#baustein-poi3) und [CTX2](#baustein-ctx2) |
| Count- und Range-UI ([SRCH4b](#baustein-srch4b)) | Das Panel zeigt die echten Zahlen und Verteilungen mit sichtbaren Lade- und Fehlerzuständen. | [SRCH3](#baustein-srch3), [SRCH4a](#baustein-srch4a), [SEC-VIS-1](#baustein-sec-vis-1), [SEC-AUTH-1](#baustein-sec-auth-1), [SEC-DUR-1](#baustein-sec-dur-1) und [SEC-SCOPE-1](#baustein-sec-scope-1) | — |

Für Counts und Ranges gilt zusätzlich:

- Counts sind für ihre jeweils berechnete Ergebnismenge exhaustiv. Eine
  begrenzte Prädikatsabweichung wird einmal zentral im Funktions- und
  API-Vertrag erklärt, nicht mit einem `≈` an jeder Zahl.
- Eine erfolgreiche Gruppe liefert keine teilberechneten Buckets. Eine
  fehlgeschlagene Gruppe wird vollständig als nicht verfügbar angezeigt.
- Der erste Range-Slice umfasst Distanz, Aufstieg und Abstieg nach der
  Presence-/Quality-Migration. Referenzdauer folgt nach [A3](#baustein-a3) in derselben
  nichtnegativen Rangeform; der höchste Punkt übernimmt nach [A2](#baustein-a2) die gemeinsame
  Missing-/Bucketssemantik mit `KnownSignedIntegerV1` und
  `SignedIntegerRangeV1`.
- Fake-Prototypwerte erscheinen bis dahin nicht als echte Filter.

### Lokale Routenfakten und Personalisierung

| Release | Ergebnis | Benötigt | Voraussetzung für |
| --- | --- | --- | --- |
| Routenform und höchster Punkt ([A2](#baustein-a2)) | Form und maximale Höhe werden als belastbare Routenfakten angezeigt. | [A1](#baustein-a1) und [SRCH3](#baustein-srch3) | [A3](#baustein-a3), [A7](#baustein-a7) und [ACC1](#baustein-acc1) |
| Neutrale Referenzdauer ([A3](#baustein-a3)) | Vergleichbare Dauer für Walk, Hike sowie Cycle Touring, Road, Gravel und MTB. | [A1](#baustein-a1) und [A2](#baustein-a2) | [A6](#baustein-a6), [A8](#baustein-a8), [C4](#baustein-c4), [SURF6](#baustein-surf6) und [CTX1](#baustein-ctx1) |
| Ausflugsprofile und Gruppen-ETA ([A4](#baustein-a4) → [A5](#baustein-a5) → [A6](#baustein-a6)) | Templates, mehrere Teilnehmende und eine komponentenweise Gruppenzeit für dieselben Aktivitätsfamilien. | [A0](#baustein-a0), [A3](#baustein-a3) und [SRCH3](#baustein-srch3); innerhalb der Scheibe [A4](#baustein-a4) → [A5](#baustein-a5) → [A6](#baustein-a6) | [A7](#baustein-a7), [A8](#baustein-a8), [TRN3](#baustein-trn3), [SEA1](#baustein-sea1) und [WEA2](#baustein-wea2) |
| Persönliche Kondition ([A7](#baustein-a7)) | Eignung für Walk, Hike und Cycle aus Dauer und Gesamtaufstieg; technische Eignung bleibt zunächst unbekannt. | [A2](#baustein-a2), [A5](#baustein-a5) und [A6](#baustein-a6) | [C4](#baustein-c4), [SURF6](#baustein-surf6), [SEA1](#baustein-sea1) und [WEA2](#baustein-wea2) |
| Ratings ([R1](#baustein-r1)) | Trails können unabhängig von der persönlichen Kalibrierung bewertet werden. | [SRCH3](#baustein-srch3) | — |
| Gelernte Kalibrierung ([A8](#baustein-a8)) | Optionale persönliche Kalibrierung aus geeigneten Aktivitätsdaten. | [A3](#baustein-a3) und [A4](#baustein-a4) | [A9](#baustein-a9) |

### Räumliche und externe Produkte

| Release | Ergebnis | Benötigt | Voraussetzung für |
| --- | --- | --- | --- |
| Startpunktsuche ([L3](#baustein-l3)) | Gruppierte Ortsergebnisse verbessern den bestehenden Startpunktfilter; jeder [GEO2](#baustein-geo2)-konforme Provider ist zulässig, Photon nur eine Referenz. | [GEO2](#baustein-geo2), qualifizierter Adapter, [SRCH-COMP](#baustein-srch-comp) und [SRCH4a](#baustein-srch4a); für den Livegang [SEC-VIS-0](#baustein-sec-vis-0) | — |
| Routenradius ([L4](#baustein-l4)) | Trails entlang einer gesamten Route mit korrektem `total`, Treffern und Pagination für einen normalisierten Punktanker. | [PLACE1](#baustein-place1), [G3b](#baustein-g3b), [SRCH1b](#baustein-srch1b), ein Punktanker aus [L1](#baustein-l1) oder [GEO1](#baustein-geo1)/`search.v1` sowie [SEC-VIS-1](#baustein-sec-vis-1), [SEC-AUTH-1](#baustein-sec-auth-1), [SEC-DUR-1](#baustein-sec-dur-1) und [SEC-SCOPE-1](#baustein-sec-scope-1) | — |
| Räumliche Counts und Histogramme ([G3b-AGG](#baustein-g3b-agg)) | Zahlen und Verteilungen verwenden denselben Spatial-, Snapshot-, ACL- und Federation-Kontext wie Treffer und `total`; es erscheinen keine Zahlen aus einem nicht räumlichen Trefferuniversum. | [G3b](#baustein-g3b), [SRCH3](#baustein-srch3), [SEC-VIS-1](#baustein-sec-vis-1), [SEC-AUTH-1](#baustein-sec-auth-1), [SEC-DUR-1](#baustein-sec-dur-1) und [SEC-SCOPE-1](#baustein-sec-scope-1) | räumliche Count- und Histogrammdarstellung in [SRCH4b](#baustein-srch4b) |
| Objektive Anstiege ([C1](#baustein-c1) → [C2](#baustein-c2) → [C3](#baustein-c3)) | Erkannte Anstiege können gefiltert und erklärt werden. | [A1](#baustein-a1), [SRCH3](#baustein-srch3) und die Kette [C1](#baustein-c1) → [C2](#baustein-c2) → [C3](#baustein-c3) | [C4](#baustein-c4) |
| Objektive Wegbeschaffenheit ([SURF0](#baustein-surf0) → [SURF1](#baustein-surf1) → [SURF2](#baustein-surf2) → [SURF3](#baustein-surf3) → [SURF3R](#baustein-surf3r) → [SURF4](#baustein-surf4) → [SURF5](#baustein-surf5)) | Reproduzierbare und hinreichend aktuelle Oberflächen- und Wegtypinformationen. | [A0](#baustein-a0), [ROUTEATTR](#baustein-routeattr), [A1](#baustein-a1), [DATA1](#baustein-data1), [IDX3](#baustein-idx3), [JOB1](#baustein-job1), [SRCH3](#baustein-srch3) und [C1](#baustein-c1); innerhalb der Scheibe [SURF0](#baustein-surf0) → [SURF1](#baustein-surf1) → [SURF2](#baustein-surf2) → [SURF3](#baustein-surf3) → [SURF3R](#baustein-surf3r) → [SURF4](#baustein-surf4) → [SURF5](#baustein-surf5); [SURF3D](#baustein-surf3d) nur bei unzureichender Sweep-Kapazität | [SURF6](#baustein-surf6) |
| Statischer Transit ([TRN2](#baustein-trn2)) | Planmässige Erreichbarkeit mit Bahn und Bus. | [TRN1](#baustein-trn1), [ACC1](#baustein-acc1) und [SRCH3](#baustein-srch3) | [TRN3](#baustein-trn3) |
| Parken und POI-Zugang ([POI3](#baustein-poi3)) | Park- und Zugangsmöglichkeiten werden als eigener Kontext nutzbar. | [POI2](#baustein-poi2), [ACC1](#baustein-acc1) und [SRCH3](#baustein-srch3) | — |
| Saisonprodukt ([SEA1](#baustein-sea1)) | Typische saisonale Bedingungen werden filter- und erklärbar. | [WEA-S](#baustein-wea-s), [A1](#baustein-a1), [C1](#baustein-c1), [A6](#baustein-a6), [A7](#baustein-a7) und [CTX2](#baustein-ctx2) | — |
| Prognoseprodukt ([WEA2](#baustein-wea2)) | Konkrete Wetterprognosen entlang der Route werden filter- und erklärbar. | [WEA-F](#baustein-wea-f), [A1](#baustein-a1), [C1](#baustein-c1), [A6](#baustein-a6), [A7](#baustein-a7) und [CTX2](#baustein-ctx2) | [WEA3](#baustein-wea3) |
| Live-Verbindungen ([TRN3](#baustein-trn3)) | Konkrete Verbindungen ergänzen den statischen Transit. | [TRN2](#baustein-trn2), [TRN-JP](#baustein-trn-jp), [CTX2](#baustein-ctx2) und [A6](#baustein-a6) | — |

Für den Routenradius gilt: Die Direct-Approximation wird zentral erklärt.
Zusätzliche 50k-/NAS-Läufe sind optionale Kapazitätsdiagnosen und kein
Release-Gate. [G3c](#baustein-g3c) bleibt eine optionale Exaktheitserweiterung; [G3f](#baustein-g3f) wird nur
nach einem künftig belegten Vertragsversagen als vollständiger H3-/Exact-Ersatz
neu entschieden. [L4](#baustein-l4) benötigt weder [GEO2](#baustein-geo2), [L3](#baustein-l3), Photon, Autocomplete noch
[G3b-AGG](#baustein-g3b-agg). Neue Kontext-Evaluatoren registrieren sich in [CTX2](#baustein-ctx2) und blockieren
bereits freigegebene Kontextprodukte nicht erneut.

### Dialog und externe Clients

| Release | Ergebnis | Benötigt | Voraussetzung für |
| --- | --- | --- | --- |
| Read-only Chat ([Q3](#baustein-q3)) | Dialogbasierte Suche im jeweils aktiven strukturierten Suchumfang. | [Q1](#baustein-q1), [Q2](#baustein-q2) und [SRCH4a](#baustein-srch4a) | [Q4](#baustein-q4) und [Q6](#baustein-q6) |
| Read-only MCP-Adapter ([Q5](#baustein-q5)) | Externe Clients verwenden dieselben typisierten Wanderer-Tools unabhängig vom internen Chat. | [Q1](#baustein-q1) sowie die für externe Clients erforderlichen Auth- und Scopes | — |
| Vergleich und Erklärung ([Q4](#baustein-q4)) | Treffer lassen sich anhand registrierter Fachdaten erklären und vergleichen. | [Q3](#baustein-q3) | — |
| Bestätigte Schreibaktionen ([Q6](#baustein-q6)) | Der Assistent darf nach Vorschau und Bestätigung ausgewählte Änderungen ausführen. | [Q3](#baustein-q3), Vorschau, Bestätigung und Audit; für Speicher- und Defaultaktionen zusätzlich [SRCH-SAVED](#baustein-srch-saved) | — |

Capability Discovery veröffentlicht stets genau den aktiven strukturierten
Umfang. Nicht verfügbare Absichten werden abgelehnt, statt nur teilweise oder
mit erfundenen Daten beantwortet zu werden. Neue Filterdimensionen erweitern
diesen Umfang später additiv.

### Gemeinsame Freigabe- und Rollbackregeln

| Regel | Verbindliche Aussage |
| --- | --- |
| Security-Gates | Jede neue sichtbare Suchscheibe auf dem heutigen gemischten Index wartet mindestens auf [SEC-VIS-0](#baustein-sec-vis-0); vorgezogene SRCH0-Fehlerkorrekturen benötigen ihren eigenen Korrektheits- und Sichtbarkeitsnachweis. Public-Federation, föderierte Counts und neue Federation-Scopes warten zusätzlich auf [SEC-VIS-1](#baustein-sec-vis-1), [SEC-AUTH-1](#baustein-sec-auth-1), [SEC-DUR-1](#baustein-sec-dur-1) und [SEC-SCOPE-1](#baustein-sec-scope-1). |
| Gateway-/Indexwechsel | Der Wechsel erfolgt atomar und erst bei vollständiger Parität sämtlicher Bestandsfilter. |
| Atomare Einheiten | Konsistent zusammen wechseln jeweils Restriction-Fence, Indexgeneration, Fachanalyseversion oder ein einzelner Kontext-Evaluator samt API und UI. |
| Aufbewahrung | Die vorige Projektion, Analyseversion und der vorige Index bleiben bis zum Maximum aus vereinbartem Rollbackfenster und letztem gültigem, daran gebundenem Suchkontext zuzüglich In-flight-Grace erhalten. |
| Code-Rollback | Er verwendet die aktuelle kompatible Generation. |
| Index-/Schema-Rollback | Er aktiviert einen getrennt bis zum aktuellen Cutoff aufgeholten Kandidaten und verändert nie eine für Cursor versiegelte Altgeneration. |
| Bestandsparität | Kein Wechsel darf einen bestehenden Filter verlieren. |

## Querschnittliche Regeln für alle Bausteine

Jedes neue abgeleitete Feld benötigt:

- eine fachliche Definition und Einheit in SI;
- `unknown` statt eines erfundenen Null- oder Defaultwerts;
- Quelle, Analyseversion und bei Bedarf Coverage/Confidence;
- bei externen Datengrundlagen eine unveränderliche Dataset-Referenz mit Quelldatenzeitpunkt, Revision/Sequenz, Artefakt-Hash und getrennter Berechnungszeit; unbekannte Providerrevisionen bleiben `opaque`;
- Create-/Update-Pfad und idempotenten Backfill;
- für alternde Ableitungen eine Freshness-Policy, durable Queue mit Claims/Leases/Retry, begrenzten Reconciler und ein Kapazitätsgate statt eines ungeschützten langen Cron-Callbacks;
- atomaren Rollout über parallelen Backfill, Coverage-Gate, aktiven Versionswechsel und einen getrennt bis zum aktuellen Cutoff aufgeholten Rollbackkandidaten;
- Dokumentmapping, Meilisearch-Einstellung und Integrationstest;
- Verhalten für lokale, importierte und föderierte Trails;
- UI-Text, der die tatsächliche Semantik benennt.

Algorithmusänderungen dürfen alte Analysewerte nicht stillschweigend mit neuen mischen. Ein Feature bleibt hinter einem Flag oder aus der Filter-UI verborgen, bis die Datenabdeckung ausreichend ist.

Tests werden nach Verantwortungsbereich getrennt:

- Unit-Tests für Filter-Compiler und Algorithmen;
- Golden-GPX-/OSM-Fixtures für Geometrie, Höhe, Dauer, Anstiege, Map-Matching, Snapshotzuordnung und Surface-Normalisierung; insbesondere Segmentgrenzen, Batch-Overlap, Parallelwege, unbekannte Coverage und den alpinen Regressionstest;
- PocketBase-/Backfill-/Queue-Tests für Versionierung, atomare Claims, Lease-Ablauf, überlappende Cron-Ticks, Crash-Recovery, Retry, fairen Cursor-Wrap, Queue-Kopf-Blockaden und Supersession alter Worker;
- Meilisearch-Integration sowie Browser-E2E für Treffer, Counts, ausgewählte Nulltreffer, gruppenweise Fehler, stabile Bucket-IDs, die 16/32/16-Grenzen, URL-State und Pagination;
- die STATE1-Suite für atomare Fach-/Revisions-/Dirty-/Fence-Commits, kausale Federation-Objektköpfe, TTL und Tombstones, lückenlose Watermarks, sequenziertes Anti-Rollback-Checkpointing, verbotene Transitionen, Publisher-Fencing, Pointer-CAS und die vollständige Crash-Matrix;
- die Federation-Security-Suite für die `SEC-VIS`-Eligibility-Matrix über jeden Ingest-/Read-Pfad, immutable Authority samt Actor-/Attribution-/Key-/Origin-Negativfällen, P1/P2-Publikationsübergänge, Inbox-/Outbox-Crash und -Retry, terminale Legacyklassifikation sowie `SEC-SCOPED`-Isolation und Burst-Coalescing;
- die ADR-0001-Suite für Deep Paging, Mutation-Soak, Cursor-/Schlüssel-/Generationen-Retention, Rolling Deployment, Crash-Recovery und Failover;
- die ADR-0002-Suite für S50/r100, 100-km-Grenze und Proximity-4xx, unabhängiges Rohgeometrie-Oracle, harte Direct-Set-/Countkonsistenz, relatives Total-Count-p95, 999/1'000/1'001-Shardzuordnung, eine globale Multi-Search sowie atomaren Familien-Swap und Recovery ohne Repacking; SRCH3 ergänzt bei Einführung seine eigenen Facetten- und Histogrammqualifikationen.

## Vor Freigabe empirisch zu kalibrieren

Die Architektur und Semantik sind oben festgelegt; folgende Zahlen werden durch ADRs, reale Korpora und Lasttests bestimmt:

- weitere Aktivitätsfamilien ausserhalb des durch ADR 0003 festgelegten ersten Umfangs, Walk-/Hike-Koeffizienten, Cycle-Distanz-/Steigraten, Systemtemplate-Werte und Grenzen von Aktivitätsprofilen; der Cycle-Kombinator samt Faktor `0.5` und die registrierten Histogramm-Bucketgrenzen v1 bleiben davon unberührt;
- Mindest-Coverage je Analyseachse sowie die Schwellen für `ready`, `partial` und `unavailable`;
- Rundwegschwelle und aktivitätsspezifische Anstiegsparameter;
- Surface-Normalisierung, Mindest-Match-Coverage, Anteil-Buckets, zulässiger Abstand zwischen Matcher-/Tag-Snapshot sowie `refresh_due_at`-/`usable_until`-Intervalle;
- Surface-Cron-Schedule, Claim-Batchlimit, Workerparallelität und Zeit-/Punkt-/Meter-/Providerbudgets aus dem Katalog- und Freshness-SLA;
- Kalibrierung möglicher späterer Radiusprofile und eines gegebenenfalls neu zu zertifizierenden geometrischen Fehlervertrags; S50/r100, der Legacy-Startpunktvertrag und der 100-Kilometer-Cap von `route_radius_ux_v1` bleiben dabei stabil;
- je typisiertem Geocoding-Adapter Datenumfang, RAM/Storage, Updatepfad, maximales Datenalter sowie Readiness-/Verfügbarkeitsziele für Managed und Self-hosted; für GEO-PHOTON werden diese Werte am Referenzprofil konkretisiert, ohne andere konforme Provider auszuschliessen;
- optionale Validierung und gegebenenfalls Nachschärfung der DS923+-Kapazitätsdiagnose, des maximal zugesagten Katalogs sowie der Index- und Orchestrator-Latenzbudgets; dies ändert nicht die Releaseentscheidung aus ADR 0002;
- ein späteres Meilisearch-Upgrade gegenüber dem durch ADR 0002 gepinnten 1.53.1-Evidenzstand, jeweils als neues unveränderliches Engineprofil und neue Indexgeneration erst nach bestandenem Contract- und Diagnoselauf, erneut bestandenem 99-Prozent-UX-Gate sowie den normalen operativen Upgradeprüfungen;
- Eval-Schwellen und unterstützte Modellprovider für `DIALOG`.

Bei einem aktiven Eignungs-, Kontext- oder technischen Filter werden unbekannte Werte standardmässig ausgeschlossen. Eine sichtbare Option kann sie einbeziehen; sie bleiben dann ausdrücklich „unbekannt“ und werden nicht als passend gewertet.

## Verbindliche Dokumente

Dieser Katalog beschreibt mögliche Lieferschnitte. Die fachlichen und technischen Invarianten stehen in den folgenden Dokumenten und gehen bei Widersprüchen vor:

- [Erweitertes Trail-Suchkonzept](/develop/specs/trail-search/)
- [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [Gespeicherte Trail-Suchen v1](/develop/specs/trail-search/contracts/saved-trail-search-v1/)
- [Federation-Sicherheits- und Publikationsvoraussetzungen v1](/develop/specs/trail-search/contracts/federation-security/)
- [Federation-, Index- und Gateway-Zustandsvertrag v1](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
- [ADR 0001: Cursor- und Snapshot-Lifecycle](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/)
- [ADR 0002: GeoJSON Direct für den Routenradius](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
- [ADR 0003: Walk, Hike und Cycle in der persönlichen Bewertung](/develop/specs/trail-search/decisions/0003-walk-hike-cycle-personal-evaluation/)
