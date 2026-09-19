---
title: Gemeinsame Invarianten der Trail-Suche
description: Capability-übergreifende Regeln für Bestandsparität, Datenqualität, Versionierung, Projektionen, Provider und Freigaben.
editUrl: false
sidebar:
  order: 2
  badge: Entwurf
spec:
  id: TRAIL-SEARCH-SHARED
  kind: shared
  status: draft
  lastReviewed: '2026-09-19'
---

> **Rolle:** Dieses Dokument ist der capability-übergreifende
> Produktvertrag der erweiterten Trail-Suche.
>
> **Normativität:** Die hier formulierten Produktinvarianten sind für jede
> betroffene Produktscheibe verbindlich. Request-/Responsefelder und
> Fehlercodes der Suchausführung gehören ausschliesslich in den
> [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/),
> diejenigen für Persistenz, Default und Auflösung gespeicherter Suchen in den
> [Saved-Search-Vertrag](/develop/specs/trail-search/contracts/saved-trail-search-v1/),
> Federation-Sicherheit in den
> [Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/),
> persistente Zustandsautomaten in
> [STATE1](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
> und angenommene Architekturentscheide in die zugehörigen ADRs. Dieses
> Dokument ist keine zweite Wire- oder State-Spezifikation.

Die Capability-Dokumente beschreiben, **was** ein Produkt fachlich leistet:

- [FOUNDATION](/develop/specs/trail-search/capabilities/foundation/) für
  Bestands- und Suchfundament;
- [DERIVED](/develop/specs/trail-search/capabilities/derived/) für lokale
  Routenfakten und Personalisierung;
- [CONTEXT](/develop/specs/trail-search/capabilities/context/) für räumliche,
  segmentbasierte und externe Kontexte;
- [DIALOG](/develop/specs/trail-search/capabilities/dialog/) für die
  dialogische Nutzungsschicht.

Bausteincodes, Abhängigkeiten und die aktuelle Arbeitsfolge stehen nur im
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/). Sie
sind keine zusätzliche Produktsemantik.

## 1. Unveränderliche Produktgrenzen

Die Suche hält vier Arten von Aussagen getrennt:

1. **Objektive Trail-Fakten** beschreiben Route und Metadaten, etwa Distanz,
   Höhenmeter, Routenform, Anstiege oder Oberfläche.
2. **Persönliche Auswertung** interpretiert diese Fakten gegen das bewusst
   gewählte Ausflugsprofil. Leistungsgrenzen des Erstellers werden weder am
   Trail gespeichert noch als Schwierigkeit anderer Personen ausgegeben.
3. **Zeitabhängiger Kontext** wie Wetter, Saison oder eine konkrete
   Verbindung besitzt Quelle, räumlich-zeitliche Gültigkeit und eigenen
   Fehlerzustand. Er überschreibt keine objektiven Trail-Fakten.
4. **Bedienung** über Filterpanel, API, App, Chat oder einen optionalen
   MCP-Adapter erzeugt denselben typisierten Suchauftrag. Keine Oberfläche
   erhält eine private Ersatzsemantik.

Fachanalysen, Map-Matching und Providerzugriffe finden weder während eines
Meilisearch-Rebuilds noch pro Trail im interaktiven Suchrequest statt.
Reproduzierbare Ableitungen werden vorab versioniert persistiert. Dynamische
Kontexte werden über begrenzte, versionierte Caches und einen kontrollierten
Orchestrator ausgewertet.

## 2. Bestandsparität und additive Evolution

Vor jeder Migration wird der tatsächlich produktive Bestand als
Regressionstest-Matrix erfasst und fachlich geprüft. Die freigegebene
SRCH0-Baseline sichert korrektes Bestandsverhalten ab; historische
Fehlresultate bleiben als Evidenz erhalten, werden aber nicht zum
Abnahmeziel. Dazu gehören mindestens:

- Freitext, Autor, Kategorie, Subkategorie und Tags;
- eigene, private, öffentliche und geteilte Trails;
- angegebene beziehungsweise importierte Schwierigkeit;
- der bisherige Startpunkt-Ort samt Punkt-Radius;
- Distanz, Aufstieg, Abstieg und Von-/Bis-Datum;
- „von mir geliked“ und Abschlussstatus;
- alle vorhandenen Sortierungen, einschliesslich der heutigen Quellwerte für
  `duration` und `difficulty`;
- API-, URL-, Browser-History- und ACL-Zustände dieser Funktionen.

Für jeden UI-, Gateway-, Projektion- oder Indexwechsel gelten folgende
Invarianten:

- Alte URLs und Requests werden weiter verstanden oder eindeutig in
  denselben Suchauftrag migriert.
- Der Key der angegebenen Schwierigkeit wird nie für persönliche Kondition
  wiederverwendet.
- Der Startpunkt-Radius bleibt ein eigener Modus. „Route berührt den
  Suchbereich“ erhält ein separates Requestfeld und deutet alte URLs nicht
  um.
- Ein Cutover darf erst erfolgen, wenn Paritäts-E2E alle Bestandsfilter,
  ACL-Kontexte und Sortierungen gegen die fachlich geprüften, aktiven
  SRCH0-Erwartungen bestanden haben. Gleichheit mit dem vorherigen Pfad
  allein genügt nicht, wenn dieser einen nachgewiesenen Fehler enthält.
- Ein Rollback aktiviert nur einen bis zum aktuellen Cutoff aufgeholten und
  bestätigten Kandidaten, der denselben Bestandsvertrag erfüllt.
- SRCH0 hält unveränderte historische Referenzdaten, beobachtete Ergebnisse
  und stabile Case-IDs getrennt von den aktiven, fachlich begründeten
  Erwartungen. Bei einem Fehler gilt die korrigierte Erwartung auf derselben
  Case-ID. Unabhängige fachliche Eigenschaften müssen die Korrektur zusätzlich
  belegen; ein Golden-Update aus der aktuellen Implementierung genügt nicht.
- Jeder nachgewiesene Fehler innerhalb des SRCH0-Bestandsumfangs blockiert
  dessen Abnahme und Merge, bis die Produktkorrektur und ihre Regressionstests
  auf demselben Integrationsstand bestehen. `knownViolation`, Skip oder ein
  erwarteter Fehlschlag dürfen diese Verpflichtung nicht umgehen. Testkorpus
  und Produktkorrekturen dürfen in getrennten PRs entstehen; die Korrekturen
  sind dann Voraussetzungen der SRCH0-Abnahme. Sie werden nicht auf
  SRCH-COMP, SRCH2 oder SEC-VIS-0 verschoben, die die freigegebene Baseline
  erst konsumieren.
- Die am 19. September 2026 ausdrücklich ausgenommenen Korrekturen
  `SRCH0-GAP-SORT-001` und `SRCH0-GAP-DTO-001` gehören nicht zum verbindlichen
  SRCH0-Abnahmeumfang. Die Sortier-Robustheit bleibt zurückgestellt;
  die Feldauswahl wird separat korrigiert. Beide sind kein Merge- oder
  Abnahmeblocker. Die begrenzte Einstufung und ihr Verhältnis zu den noch
  unveränderten Tests stehen in
  [SRCH0](/develop/specs/trail-search/work-items/search/srch0/); gültige
  Bestandssortierungen und alle anderen Blocker bleiben verbindlich.
- Nachfolgende Tasks erhalten die aktive Baseline. Bewusste neue Semantik
  benötigt ein ausdrücklich begründetes, auf stabile Case-IDs bezogenes
  Delta-Overlay samt Nachweis aller unveränderten Resultate. Historische
  Beobachtungen bleiben dabei unverändert; ein Overlay ist keine Freistellung
  für einen bekannten Bestandsfehler.
- Eine neue Oberfläche darf Controls neu ordnen oder verständlicher benennen,
  aber keine produktive Funktion still verbergen oder semantisch ersetzen.
  Ausschliesslich wirkungslose Prototypfelder und Beispielzahlen werden
  entfernt.

Neue Funktionen werden additiv und capability-gebunden eingeführt.
Feature-Flags verbergen ausschliesslich neue Funktionen. Eine spätere
Deprecation eines Bestandsfilters wäre ein eigenes Produktvorhaben.

## 3. Kanonische Datenrollen

Persistenz dient nicht nur dem schnellen Indexbuild. Sie macht Herkunft,
Eingangsstand, Modellversion, Abdeckung und Aktualität erklärbar und
reproduzierbar.

| Rolle | Beispiele | Verbindlicher Zweck |
| --- | --- | --- |
| Versionierte Fachfakten | `trail_analysis`, `trail_climb`, `trail_surface_analysis`, `trail_access_summary`, `trail_rating_summary`, `federated_trail_snapshot` | Kanonische, erklärbare Resultate je Eingangs- und Modellversion |
| Verwerfbares Kern-Read-Model | `trail_search_projection` mit Payload, Hash und Revisionen | Genau ein deterministisch erzeugtes Kerndokument je Trail |
| ACL-sensitives Read-Model | `trail_search_acl_overlay` mit Scope-, Payload- und ACL-Hash | Enger sichtbare Suchterme und Presence-Fakten ausserhalb des Kerns |
| Arbeitszustand | koaleszierende Dirty-/Desired-Revision plus persistente Jobs | Wiederaufnahme, Supersession und begrenzte Verarbeitung |
| Lückenloses Änderungslog | Change-Log und Tombstones | Replay, Reconciliation und vollständiger Schattenaufbau |
| Delivery und Sicherheit | Generationen, Watermarks und Restriction-Fences | Nur bestätigte Stände abfragen; Verengungen sofort fail-closed |
| Federation-Control-Plane | Inbox/Outbox, Object-Authority, Publikation und Provenienz | Public, Direct Share, Unpublish und Remote-Lifecycle beweisbar trennen |
| Persönliche und dynamische Daten | Profile, `trail_saved_search`, `trail_search_default` sowie versionierte Zeit-/Providercaches | Owner-gebunden halten; niemals als globale Trail-Eigenschaft materialisieren |

Kompakte Fachfakten und Projektionen liegen in der vorhandenen
PocketBase-/SQLite-Datenbank. Voluminöse Segmente erhalten nur bei belegtem
Bedarf eine normalisierte Repräsentation oder begrenzte Artefakte. Der
Meilisearch-Index ist ein verwerfbares Lieferartefakt, keine fachliche
Wahrheitsquelle. Counts werden nicht als Fachfakten persistiert.

Die Kernprojektion enthält nur Inhalte, die für die gesamte Trail-Audience
auffindbar sein dürfen. Ein Foto- oder Waypoint-Fakt mit engerer Sichtbarkeit
gehört in ein scopegebundenes Overlay. Union, Deduplikation, Ranking, Counts,
Pagination und Widerruf müssen über den vollständigen autorisierten
Kandidatenraum korrekt sein; ein Post-Filter auf der ersten Seite ist
unzulässig.

## 4. Unknown, Presence, Coverage und Einheiten

`unknown` ist ein fachlicher Zustand und kein bequemer Ersatz für `0`, einen
leeren String oder eine fehlende Relation.

| Zustand | Bedeutung |
| --- | --- |
| Filter ausgelassen | Keine Einschränkung; bekannte und unbekannte Werte bleiben im Suchuniversum |
| `absent` | Die Relation oder Eigenschaft fehlt fachlich, etwa keine Subkategorie |
| `unknown` | Der Wert ist nicht belastbar bekannt oder die nötige Abdeckung fehlt |
| `unsupported` | Familie, Disziplin oder Evaluator ist bekannt, aber die aktive Capability besitzt dafür noch kein freigegebenes Modell |
| `not_applicable` | Die Achse gilt für das gewählte Modell fachlich nicht; sie ist weder null noch unbekannt |
| bekanntes `0` | Autoritativ geliefert oder durch einen erfolgreichen versionierten Lauf belegt |
| `unmapped` | Ein fremder Wert ist vorhanden, aber nicht auf die lokale Taxonomie abbildbar |
| `ambiguous` | Ein vorhandener Wert besitzt mehrere nicht eindeutig auflösbare Mappings |
| `partial` | Eine Analyse liefert Detaildaten, darf aber noch nicht alle Filteraussagen autorisieren |
| `opaque` | Quelle oder Datasetrevision ist nicht beweisbar; sichtbar für Diagnose, nicht als exakte Filterbasis |

Ein historischer Datenbankdefault `0`, ein Clientdefault oder
`COALESCE(..., 0)` belegt keinen bekannten Wert. Flache Routen dürfen dagegen
bekannte null Höhenmeter besitzen. Distanz `0` und Referenzdauer `0` sind
keine validen bekannten Trailmetriken. Eine Höhe auf Meereshöhe kann bekannt
`0` sein; fehlende Höhe bleibt unknown. Valide negative Höhen benötigen einen
signierten Typ.

API, Projektion, URL, persistierte Fakten und Bucketgrenzen verwenden
kanonische SI-Einheiten. Kilometer, Meilen, Meter, Fuss oder lokalisierte
Zeitangaben sind Präsentation. Stable IDs und numerische Grenzen werden nicht
aus Übersetzung, Position oder aktuellem Count abgeleitet.

Jede filterbare Analyseachse definiert:

- Presence beziehungsweise Status;
- fachliche Einheit und gültigen Wertebereich;
- Quelle und Eingangs-Fingerprint;
- Analyse-, Normalisierungs- und gegebenenfalls Datasetversion;
- Coverage und, wo fachlich sinnvoll, Confidence;
- Berechnungszeit getrennt vom Quelldatenstand;
- Verhalten bei fehlenden, partiellen, abgelaufenen und ungültigen Werten.

Negative Aussagen wie „keine Stufen“ oder Maximalgrenzen sind nur bei der
dimensionsspezifisch erforderlichen Mindest-Coverage erfüllt. Geringe
Coverage ergibt unknown, nie automatisch „keine“, „leicht“ oder „geeignet“.

## 5. Versionierung, Jobs, Backfill und Freshness

Upload, manuelle Planung, Plugin-Import, Federation und Backfill verwenden
denselben serverseitigen Fachpfad. Ein Algorithmuswechsel darf alte und neue
Werte nicht unmarkiert mischen.

Jede neue oder alternde Ableitung benötigt:

- einen idempotenten Create-/Update-Pfad und einen resumierbaren Backfill;
- einen unveränderlichen Input-Fingerprint, mindestens aus Geometriehash,
  relevanten Metadaten und aktiven Modell-/Datasetversionen;
- einen persistenten Sollzustand beziehungsweise eine `desired_revision`;
- eine durable Queue mit atomaren Claims, Lease-Token, Heartbeat,
  Retry/Backoff/Jitter, Fehlerstatus und Supersession;
- einen begrenzten Reconciler mit persistentem Cursor und Wrap-around;
- Zeit-, Mengen-, Punkt-, Meter-, Speicher- und Providerbudgets passend zur
  Arbeit;
- einen Aktivierungs-Compare-and-swap, der Lease, Sollrevision,
  Fingerprint und aktuelle Eingaben gemeinsam prüft;
- eine dokumentierte Freshness-Policy, wenn die Quelle altert;
- Metriken für Queue-Tiefe, älteste fällige Arbeit, Durchsatz, Lag und
  terminale Fehler.

Ein Datenbank-Cron ist nur ein kurzer idempotenter Trigger. Die Queue ist das
System of Record; ein langer ungeschützter Providercallback im Cron ist kein
zulässiger Lifecycle. Nur tatsächlich neu beanspruchte beziehungsweise
eingereihte Arbeit verbraucht ein Batchbudget. Abgelaufene Leases werden
zurückgeholt, ein älterer Worker kann einen neueren Sollzustand nicht
zurückrollen, und ein permanenter Fehler darf den Cursor nicht für alle
folgenden Objekte blockieren.

Externe Datasets werden als unveränderliche Manifeste mit Quelle,
Providerrevision oder Sequenz, `source_data_as_of`, Artefakt-Hash, Coverage,
Lizenz und Aktivierungsstatus referenziert. Build- oder Downloadzeit ist nie
gleich Quelldatenstand. Eine heutige Neuberechnung auf einer alten Karte
verlängert deren fachliche Nutzbarkeit nicht. Ist die Providerrevision nicht
bestimmbar, bleibt die Qualität `opaque`.

Ein produktiver Versionswechsel besteht aus paralleler Berechnung,
Coverage-/Qualitätsgate, Schattenprojektion, Replay bis zum Cutoff und
atomarer Aktivierung. Der Rollbackkandidat wird getrennt bis zum aktuellen
Cutoff aufgeholt. Von gültigen Suchkontexten, Rollbackkandidaten oder lebenden
Jobs referenzierte Analysen und Snapshots bleiben reteniert.

Die vollständigen Revisionstransaktionen, Fence-Acknowledgements,
Generationenwechsel sowie Recovery- und Retentionautomaten definiert
[STATE1](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/).

## 6. Search Context, Counts und Federation

Der
[Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
trennt den reproduzierbaren fachlichen Auftrag, die transportbezogene Anfrage
und die Antwort. Filterpanel, App und Dialog erzeugen denselben fachlichen
Auftrag; der Browser setzt keine freien Meilisearch-Ausdrücke zusammen.
Gespeicherte Suchen persistieren ausschliesslich diesen normalisierten Auftrag;
sie speichern weder Suchkontext noch Ergebnisse und werden bei jeder Ausführung
neu gegen den aktuellen Principal-, ACL- und Federationstand geprüft.

Treffer, `total`, Facetten, Histogramme, Highlights und Pagination einer
Antwort verwenden denselben Suchkontext. Dieser bindet mindestens den
normalisierten Auftrag, Principal-/ACL-Scope, Quell-Scope, `as_of`, exklusives
`valid_until`, Content-Snapshot, Indexgeneration, ausgelieferte
Katalogrevision, Security-Watermark sowie Policy-, Capability-, Profil- und
gegebenenfalls Spatial-/Kontextrevision.

Ein Cache-Key enthält den effektiven Principal und alle diese fachlich
relevanten Stände; ein Spec-Fingerprint allein reicht nicht. Suchcaches werden
nie benutzerübergreifend geteilt und enden vor der frühesten relevanten
Federation-, Dataset- oder Kontext-Expiry. Cursor setzen ausschliesslich ihre
gebundene, retenierte Generation fort. Die angenommene Cursorentscheidung
steht in
[ADR 0001](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/).

Ein Count bedeutet die Zahl auffindbarer Trail-Einträge im materialisierten
Suchbestand **dieser Wanderer-Instanz**, nicht eine Live-Zahl des Fediverse:

```text
(aktive lokale Projektionen
 ∪ (aktive föderierte Projektionen ∩ gültiger Snapshot zum as_of))
∩ ACL des aktuellen Benutzers
∩ lokale Taxonomie-/Ausblendregeln
∩ Quell-Scope local | federated | all | origin_instance
```

Entfernte Instanzen werden nicht pro Suche abgefragt. Föderierte Trails zählen
nur als lokal materialisierte, autoritativ verifizierte und zum gemeinsamen
`as_of` auffindbare Snapshots. Abgelaufene, widerrufene, gelöschte oder nur
direkt geteilte Publikationen zählen nicht zum öffentlichen Suchuniversum.
`origin_instance` ist eine serverseitig normalisierte Origin-ID, kein frei
injizierter Domainfilter.

Eine aktive Publication-IRI zählt genau einmal. Nach `Delete(P1)` ist eine
spätere Publikation mit neuer IRI `P2` eine neue Publikationsgeneration; nur
die aktive, verifizierte und auffindbare Generation zählt. Getrennt
veröffentlichte Kopien derselben Geometrie bleiben getrennte Trail-Einträge.
Authority, Publication und Restriction definiert der
[Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/).

Counts verwenden dieselbe ACL, dieselben Shares, Taxonomiemappings und
Source-Scopes wie Treffer. Private Inhalte dürfen auch über Count-Differenzen
keine Information preisgeben. Normale Inhaltsänderungen werden erst nach
bestätigter Zustellung als neuer Inhaltsstand sichtbar;
Sichtbarkeitsverengungen wirken ab dem autoritativen Commit über den aktuellen
Restriction-Fence sofort und fail-closed auf Treffer, Ranking, Counts,
Highlights, Cache und Pagination.

Fremde Kategorien werden über ein versioniertes Mapping auf die lokale
Taxonomie abgebildet; `unmapped` und `ambiguous` bleiben explizite Zustände.
Eine Mappingänderung invalidiert und reprojiziert alle betroffenen
föderierten Dokumente, auch wenn sie unter der vorigen Revision bereits
erfolgreich zugeordnet waren.

Zugesagte Counts sind exakt für die tatsächlich berechnete Ergebnismenge.
Ein räumliches Prädikat darf gegenüber der kanonischen Rohgeometrie nur dann
approximativ sein, wenn ein angenommener, versionierter Produktvertrag dies
offenlegt. Für den Routenradius ist das
[ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/).
Auch dann bleiben Treffer, Seite, `total` und jede tatsächlich angebotene
Aggregationsgruppe intern konsistent zu derselben Direct-Menge. Eine
fehlgeschlagene Gruppe liefert weder alte Werte, `0` noch Teilzahlen.

Lokale Likes, Originbehauptungen und Ratings bleiben getrennte Konzepte:
„von mir geliked“ ist Actorzustand, `local_observed_like_count` beschreibt auf
dieser Instanz beobachtete Popularität, ein `origin_like_count` ist nur eine
belegte Herkunftsangabe, und Ratings bleiben ohne eigenen
Föderationsvertrag instanzlokal.

## 7. Plugin-, Provider- und Datasetgrundlagen

Die Providerarchitektur trennt:

- **Plugin-Typ:** stabiles fachliches Host-Subsystem mit Datenmodell,
  Lifecycle, Instanz-Scope, Auswahl-/Fallbacklogik, UI und Sicherheitsregeln;
- **Capability:** optionale, versionierte Operation innerhalb eines Typs;
- **Plugin-ID:** konkrete Implementierung oder Quelle;
- **Plugin-Instanz:** konfigurierte Verwendung mit Scope `system` oder
  `user`, optionalem Owner, `instance_key`, Region, Priorität und Credentials;
- **Dataset-Referenz:** unveränderlicher, aktivierbarer Datenstand, der nicht
  mit der Pluginversion verwechselt wird.

Typen werden nach Fachdomäne und nicht nach Anbieter benannt. Deshalb sind
beispielsweise `geocoding`, `pois`, `weather` und `transit` sinnvoll, nicht
`nominatim`, `overpass`, `open_meteo` oder `sbb` als Top-Level-Typen. Ebenso
ist `transit` der Typ und GTFS ein mögliches Format beziehungsweise eine
Plugin-ID. Ein generischer Typ `provider` würde unterschiedliche Datenmodelle
und Lebenszyklen nur verstecken.

Effektive Capabilities sind die Schnittmenge aus Typvertrag,
Pluginimplementierung, Deployment-/Importprofil, Instanzkonfiguration und
Nutzungsrichtlinie. Kein Produktpfad schaltet Funktionen anhand einer
bestimmten Plugin-ID frei.

Die gemeinsame Hostbasis benötigt:

- eine zentrale Typ-/Capability-Registry und typabhängige
  Manifestvalidierung;
- mehrere eindeutig adressierbare System- oder Benutzerinstanzen;
- die oben definierte durable Job- und Datasetlaufzeit;
- persistente Caches mit TTL, ETag, Singleflight, Refresh-ahead und
  dokumentiertem `stale-if-error`;
- sichere Connectoren, kurze Deadlines und Ressourcenlimits im Such-Hotpath;
- kontrollierte datei- oder streambasierte Artefakte statt ungebundener
  grosser synchroner WASM-RPC-Antworten;
- First-Party-Defaults für den üblichen Fall und dieselben Verträge für
  Community-Plugins.

Provider werden weder pro Trail noch pro Kandidat live abgefragt. Eine
benutzerinitiierte, gecachte Ortsauflösung vor der Trail-Suche ist davon
getrennt und besitzt eigene Datenschutz- und Usage-Profile.

## 8. Freigaben und Testverantwortung

Die kleinste sichtbare Releaseeinheit ist eine fachlich vollständige
Produktscheibe mit erkennbarem Benutzerziel. Eine schmale Scheibe ist erlaubt,
wenn sie eigenständig nützlich, semantisch korrekt und ehrlich begrenzt ist.
Sie darf keine „Light-Version“ vortäuschen und keinen Bestandsfilter verlieren.

Security ist kein globaler Vorblock für Spezifikation, Parser, Backfills oder
nicht exponierte Schattenarbeit. Eine sichtbare Scheibe wird jedoch erst
freigegeben, wenn ihre betroffenen Gates aus dem Security-Vertrag bestanden
sind. Ein interner Enabler darf vorher integriert werden, wird aber nicht als
gelieferter Nutzermehrwert ausgegeben.

Vorgezogene SRCH0-Korrekturen des bestehenden Suchpfads brauchen ihren eigenen
Korrektheits- und Sichtbarkeitsnachweis. Sie aktivieren keine neue Suchscheibe
und warten deshalb nicht auf den vollständigen späteren SEC-VIS-0-Rollout.

Tests werden nach Verantwortung getrennt:

- SRCH0-Tests für den fachlich geprüften Bestandsumfang, aktive korrigierte
  Erwartungen und von Implementierung und Goldens unabhängige Eigenschaften;
  historische Beobachtungen bleiben separat reproduzierbare Evidenz;
- Unit-Tests für Compiler, Normalisierung und Fachalgorithmen;
- Golden-Fixtures für GPX, Geometrie, Höhe, Dauer, Anstiege, Map-Matching,
  Snapshotzuordnung und Surface-Normalisierung;
- PocketBase-/Backfill-/Queue-Tests für Versionen, Claims, Leases,
  Überlappung, Crash-Recovery, Retry, Cursor-Wrap und Supersession;
- Meilisearch-Integration und Browser-E2E für Treffer, `total`, Counts,
  Nulloptionen, Fehlergruppen, Bucket-IDs, URL-State und Pagination;
- IDX0-Tests für nichtdestruktiven Normalstart, fehlende beziehungsweise
  unerwartet leere Legacyindizes, terminale Bootstrap-Tasks und beide
  Readiness-Endpunkte sowie dauerhaft fehlerhafte Polyline-Kandidaten ohne
  globales Readinessgate; der No-Engine-Guard gehört zu SRCH-COMP;
- STATE1-Tests für atomaren Fach-/Revisions-/Dirty-/Fence-Commit,
  Watermarks, Tombstones, Publisher-Fencing, Pointer-CAS und Recovery;
- Federation-Security-Tests für Eligibility auf jedem Ingest-/Read-Pfad,
  Authority-Negativfälle, Publikationsübergänge, durable Inbox/Outbox und
  Scope-Isolation;
- ADR-0001-Tests für Deep Paging, Schlüssel-/Generationen-Retention,
  Rolling Deployment, Crash und Failover;
- ADR-0002-Tests für Spatial-Vertrag, Shardzuordnung, Multi-Search,
  Familien-Swap und getrennt qualifizierte Aggregationen.
- Saved-Search-Tests für Owner-Isolation, URL-vor-Default-Präzedenz,
  Revisionkonflikte, History/Reload und fail-closed Reparaturzustände.

Jedes Capability-Dokument ergänzt nur seine fachlich eigenen Fixtures und
Gates. Die aktuelle Reihenfolge und mögliche Lieferschnitte bleiben im
[Delivery-Katalog](/develop/specs/trail-search/delivery/).
