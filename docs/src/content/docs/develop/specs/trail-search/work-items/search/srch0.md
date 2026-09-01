---
title: SRCH0 — Bestandsvertrag
description: Ausführbares Kompatibilitätsledger für produktive Trail-Suche, Legacyzustand, Projektion, ACL und drei Bestandskorrekturen.
editUrl: false
sidebar:
  order: 1
  badge: Reviewbereit
spec:
  id: SRCH0
  kind: work-item
  status: reviewable
  deliveryStatus: candidate
  capability: FOUNDATION
  productSlice: search-foundation
  exposure: cutover
  releaseGates: [SRCH-COMP, SEC-VIS-0]
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED, FEDERATION-SECURITY-V1, STATE1-CONTRACT]
  lastReviewed: '2026-09-01'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Work Item | `SRCH0` |
| Spec-Status | `reviewable` |
| Delivery-Status | Kandidat; vier reale Engine-Evidenzprototypen bleiben vor `accepted` und Livefreigabe offen |
| Capability | `FOUNDATION` |
| Produktschnitt | `search-foundation` |
| Exposure | `cutover`; Korpus und Shadowarbeit bleiben intern, die drei Korrekturen werden gemeinsam aktiviert |
| Implementierungsabhängigkeiten | keine für Korpus, Harness, Code und Shadowindex |
| Aktivierungsvoraussetzung | SRCH-COMP und SEC-VIS-0 haben ihre SRCH0-Livefälle erfüllt; der Reindex selbst setzt keine dort erst zu bauende Control Plane voraus |
| Releasegate | `SRCH-COMP`, `SEC-VIS-0` und alle als `srch0_live` markierten Zielerwartungen |
| Ausgangsrevision | Git-Commit `e6db729cb73a490a939420e256b176acf9fd8fc3` |
| Engine-Ausgangsprofile | Meilisearch 1.11.3 und 1.36.0 mit den Settings der Ausgangsrevision |
| Ergebnis | ausführbare Paritäts-, Korrektur- und Regressionsmatrix für die produktive Trail-Suche |
| Letzte Prüfung | 1. September 2026 |

## Nutzer- und Betreiberergebnis

Nach SRCH0 kann jede spätere Änderung an Suchvertrag, UI, API, Projektion,
Engine oder ACL gegen denselben reproduzierbaren Bestand geprüft werden. Ein
Review sieht pro Fall nicht nur einen erzeugten Meilisearch-String, sondern den
ausgewerteten Browserzustand, den Principal, die erwartete Treffermenge,
Reihenfolge, `total`, Seite und die bewussten Abweichungen vom Ausgangsstand.

SRCH0 bereitet ausserdem drei vollständige Bestandskorrekturen vor:

- Der Legacycompiler erzeugt den Startpunkt-`_geoRadius` genau einmal.
- Eine fehlende oder unbekannte lokale Quellschwierigkeit wird in Projektion,
  Defaultsuche, Trefferkonvertierung und Anzeige als `unknown` behandelt und
  niemals als `easy` oder `difficult` ausgegeben.
- Föderierte Listenstatistiken werden ohne Livezugriff auf die Origininstanz
  deterministisch aus den lokal materialisierten `lists.trails` berechnet. Bis
  zum Full-Sync ist eine leere Relation sichtbar als Trailzahl und Summenwert
  `0`; `needs_full_sync` blockiert deshalb weder Startup noch Search-Readiness.

SRCH0 besitzt zwei getrennte Abschlussmarken. **Korpus/Tooling fertig** bedeutet,
dass Fixturekorpus, Harness, Code und ein nicht exponierter Kandidat
reproduzierbar grün sind; diese Marke entsperrt SRCH-V1 und SEC-VIS-0.
**Produktiv aktiviert** bedeutet zusätzlich, dass das Releasegate aus diesem
Dokument erfüllt und SRCH-COMP gemeinsam mit dem geprüften Reindex umgeschaltet
ist. Ein Fixtureverzeichnis ohne ausführende Tests, ein Codefix ohne Bestandsnachweis
oder ein Shadow-Erfolg ohne Livegate erfüllt die jeweilige Marke nicht.

## Scope

- Die tatsächlich produktiven Trail-Suchpfade in Liste, Karte, Profil,
  Suchproxy, Multi-Search, Cluster- und Hilfsabfragen inventarisieren.
- Browserdefaults, URL-, LocalStorage-, Snapshot-/History- und
  Paginationzustand vor jeder V1-Migration reproduzierbar auswerten.
- Alle produktiven Filter, Filterkombinationen und neun sichtbaren
  Sortierungen mit positiven, negativen und Randfällen erfassen.
- Projektion aus dem PocketBase-Trailrecord, Meilisearch-Einstellungen,
  Tenant-Token-ACL und serverseitige Kategoriepräferenzen in getrennten
  Testschichten abdecken.
- Einen sprachneutralen, schema-validierten Fixturekorpus bereitstellen, den
  Go-, Vitest-, Meilisearch-Integrations- und Playwright-Tests gemeinsam
  konsumieren.
- Den doppelten `_geoRadius` entfernen und seine einmalige Kompilierung sowie
  unveränderte Trefferwirkung beweisen.
- Unknown-Difficulty Ende-zu-Ende korrigieren und den bestehenden Trailindex
  über einen geprüften Shadow-Rebuild atomar umschalten.
- Föderierte Listenaggregate auf die lokal materialisierte Trailrelation
  umstellen, den bisherigen Live-Origin-Read entfernen und die sichtbare
  numerische `0`-Semantik bis zum Full-Sync als SRCH0-Korrektur belegen.
- Das neue interne Sortierfeld vor jeder Aktivierung an Request-, Engine- und
  Responsegrenze gegen freie Nutzung und Ausgabe abschotten.
- Liveness, Search-Readiness und den eng begrenzten Offline-Reindex als
  ausführbare Betreiberpfade festlegen.
- Den von `SEC-VIS-0` verantworteten produktiven Compose-, Quickstart- und
  Installationspfad ohne veröffentlichten Engineport als Livegate binden.
- Die Search-Parent-Key-Rotation mit einer neuen Web-Token-/Cookie-Epoche und
  dem Verhalten bereits gecachter 24-Stunden-Tokens als selben Releasevertrag
  prüfen.
- Einen maschinenlesbaren Abweichungsbericht mit stabilen Case-IDs erzeugen.
- Den authentifizierten Actor-Suchpfad und die Duplikatprüfung beim Trail-
  Upload als eigenständige Engine-Consumer mit ihren tatsächlichen Defaults
  und Projektionsabhängigkeiten erfassen.

Der rohe Proxy `POST /api/v1/search/{index}` macht heute technisch beliebige
Meilisearch-Optionen erreichbar. SRCH0 friert daraus ausschliesslich die im
Repository vorhandenen First-Party-Requestformen und ausdrücklich
dokumentierten Felder ein. Eine zufällig von der Engine akzeptierte freie
Filter-, Ranking- oder Retrievaloption wird dadurch nicht zum dauerhaften
Wanderer-Vertrag. Bevor der SRCH0-Zielindex öffentlichen Traffic erhalten darf,
muss SRCH-COMP den Browser und alle First-Party-Consumer am Kandidaten im
geschlossenen Aktivierungsprofil `srch0_compat` qualifiziert haben.
Adapterrouting und Indexswap werden danach im selben geschlossenen
Wartungsfenster aktiviert. Dieses Profil kompiliert ausschliesslich die
inventarisierten bestehenden Browserzustände serverseitig und verwendet die
geschlossene V1-Trefferprojektion als Responsegrenze; es veröffentlicht aber
noch **keine** neue öffentliche V1-Suchcapability.

Insbesondere bleiben `POST /api/v1/trails/search`, Capability Discovery,
Cursor-/Snapshotausstellung, Facetten, Histogramme, neue Filter und Sorts,
Route-Radius sowie `federated`-, `all`- und Origin-Scopes am öffentlichen
Ingress deaktiviert. Der generische Einzel- und Multi-Search-Pfad ist beim
Öffnen des Ingress deaktiviert oder akzeptiert nur einen geschlossenen,
serverseitig kompilierten Legacyzustand; freie Indexnamen, `filter`, `sort`,
`attributesToRetrieve` und sonstige Engineoptionen werden nicht mehr
angenommen. Alte direkte Search-/Tenant-Token sind rotiert und als Bypass
unbrauchbar. SRCH0 definiert dafür keine zweite öffentliche Requestgrammatik,
sondern liefert Negativfixtures und blockiert seinen Live-Swap bis zum
SRCH-COMP-Nachweis. Jede spätere Freischaltung des vollständigen V1-Adapters
bleibt ein eigener Capability-/Gateway-Cutover mit seinen vollständigen
Security- und STATE1-Gates.

Die Implementierungsgrenze dafür ist geschlossen: Die beim Cutover
ausgelieferte Browseranwendung übergibt nur die inventarisierten bestehenden
Seiten- und Formularzustände an eine Same-Origin-SvelteKit-Load-/Action-
Grenze. Diese wertet den Zustand mit dem gemeinsamen Legacy-Evaluator aus und
ruft danach die nicht per HTTP exportierte typisierte Schnittstelle
`LegacyTrailSearchAdapter.Search(ctx, EvaluatedLegacyTrailSearch)` auf. Erst
dieser serverseitige Adapter erzeugt Indexname, Filter, Sortierung und
Retrievalprojektion für Meilisearch. Der Browser darf keine dieser
Engineoptionen mitsenden. Die rohen Einzel- und Multi-Search-Routen weisen
nicht intern erzeugte Aufrufe vor jedem Enginezugriff ab. Die interne
Schnittstelle erscheint weder in OpenAPI noch in Capability Discovery und ist
keine zweite öffentliche Grammatik.

## Nichtziele

- Keine neue öffentliche Such-API, keine V1-Normalisierung und kein
  kanonischer V1-URL-Codec; diese gehören zu SRCH-V1 und SRCH-COMP.
- Kein neues fachliches Indexmodell, kein Federation-Gateway, keine Counts,
  Histogramme oder neuen Filter.
- Keine Paritätszusage für beliebige freie Meilisearch-Ausdrücke oder
  undokumentierte Responsefelder des generischen Proxys.
- Keine Aufnahme des sichtbaren `TrailFilterPreview` als produktive
  Capability. Seine Beispielwerte verändern die reale Suche nicht und werden
  lediglich als wirkungsloser UI-Prototyp inventarisiert.
- Keine Umdeutung der gespeicherten Quellschwierigkeit in eine persönliche
  Schwierigkeit und keine rückwirkende Reparatur des fachlichen Quellrecords.
- Kein neuer allgemeiner oder online betriebener Index-Control-Plane- und
  Generationenautomat. Zulässig ist ausschliesslich der in STATE1 § 1.1
  definierte kleine persistente Crash-Guard `search_index_state` für
  `offline-srch0-v1`; er erhält weder Generationen noch Publisherzustände und
  übergibt seine Mutationsautorität später explizit an STATE1. Der temporäre
  Shadow-Index ist nur das sichere Transportmittel für diese gestoppte
  Bestandskorrektur und kein zweiter unterstützter Suchpfad.

## Heutiges Verhalten und Evidenz

### Datenfluss

```text
URL / LocalStorage / SvelteKit-Snapshot / UI
  -> oberflächenspezifische Defaults und sanitizeTrailFilter
  -> TrailFilter
  -> buildFilterText + q + sort + page
  -> SvelteKit-Suchproxy
  -> Kategoriepräferenzen + Meilisearch-Tenant-Token
  -> Index trails
  -> TrailSearchResult -> Trail
  -> Liste, Tabelle, Kartenliste oder Cluster

PocketBase-Trail + expandierte Relationen
  -> documentFromTrailRecord
  -> Meilisearch-Dokument im Index trails
```

Der [Bestandsaudit vom 30. August 2026](/develop/specs/trail-search/evidence/evidence-and-calibration/#bestandsaudit-vom-30-august-2026)
belegt diesen Ausgangspunkt am genannten Commit. SRCH0 prüft ihn erneut; die
ausführbaren Fixtures ersetzen den Audit als Paritätsevidenz.

| Grenze | Heutiger Codeanker | Für SRCH0 einzufrierender Anteil |
| --- | --- | --- |
| Listen-Defaults und Legacy-URL | `web/src/routes/trails/+page.ts`, `web/src/routes/trails/+page.svelte` | Defaultfilter, First-Value-URL, Storagepräzedenz, Seite |
| Karten-Defaults und View-State | `web/src/routes/map/+page.ts`, `web/src/routes/map/+page.svelte` | abweichende Defaultsortierung, Bounds und Kartenkoordinaten |
| Sanitizing | `web/src/lib/util/trail_filter_util.ts` | Typkonversion, Clamp, Fallback und Min/Max-Korrektur |
| Legacycompiler | `web/src/lib/stores/trail_store.ts` (`buildFilterText`) | Filterlogik, Rundung, Operatorgruppen und Engineauftrag |
| Suchproxy | `web/src/routes/api/v1/search/[index]/+server.ts`, `multi/+server.ts` | bekannte First-Party-Requestformen, keine freie API-Zusage |
| Präferenzfilter | `web/src/lib/server/category_preference_filter.ts` | actorbezogene Kategorie-/Subkategorieausblendung |
| ACL | `db/routes/search_token.go` | anonymer und authentifizierter Tenant-Token-Kontext |
| Projektion | `db/util/meilisearch.go` (`documentFromTrailRecord`) | Dokumentfelder und Difficulty-Abbildung |
| Engineprofil | `db/main.go` (`initMeilisearchConfig`) | Attribute und Rankingregeln als Fingerprint |
| Trefferkonvertierung | `web/src/lib/stores/trail_store.ts` (`searchResultToTrailList`) | erlaubte Trefferfelder und Unknown-Darstellung |
| Actor-Suche | `web/src/routes/api/v1/search/actor/+server.ts` | Route ist auth-pflichtig; Handle-Lookup vor lokalem `actors`-Search, `includeSelf` und Defaultlimit 3 |
| Upload-Duplikatprüfung | `web/src/routes/api/v1/trail/upload/+server.ts` (`findDuplicate`) | leere Query ohne Optionen, Engine-Defaultlimit 20 und Abhängigkeit von `_geo`, Distanz, Auf-/Abstieg sowie Fehlerfeldern |
| Weitere Consumer | lokales/entferntes Profil, Empfehlung, Bounding-Box, Cluster und ID-Multi-Search | injizierte Filter, Live-Remote-Grenze und Eligibility statt zufälliger Reihenfolge |

### Bestätigte Legacy-Semantik

| Eingabe | Auswertung am Ausgangsstand |
| --- | --- |
| Freitext `q` | unverändert an Meilisearch; Suchattribute in der Reihenfolge `author_name`, `name`, `description`, `location`, `tags` |
| Distanz, Aufstieg, Abstieg | Minima immer mit `floor`; Maxima nur unter dem dynamischen Oberlimit mit `ceil`; Grenzen inklusive |
| Difficulty | Werte `0`, `1`, `2` innerhalb der Gruppe OR; der Default `[0,1,2]` erzeugt heute ebenfalls einen Filter |
| Autor | exakte Actor-ID |
| Public/Private/Shared | Browserzweige werden mit dem aktuellen Actor erzeugt; der Tenant-Token bleibt die autoritative zusätzliche Einschränkung |
| Von mir geliked | `true` filtert auf die Actor-ID; `false` ist wirkungslos |
| Datum | Start und Ende werden als Unixsekunden aus dem JavaScript-Datum kompiliert; das Ende trifft heute nur den Beginn des angegebenen Tages |
| Kategorie/Subkategorie | Kategoriebranches OR; echte Subkategorie oder `__no_subcategory__:<category>` verengt ihren Parentbranch |
| Tags | ausgewählte Tagnamen OR; gegenüber allen anderen Filtergruppen AND |
| Abschlussstatus | `true` und `false` filtern exakt; ausgelassen ist neutral |
| Startpunktnähe | nur in der Listenabfrage, nur bei truthy Latitude und Longitude; dieselbe `_geoRadius`-Klausel wird doppelt erzeugt |
| Kartenbounds | Bounding-Box-Prädikat für Liste und Cluster; View-State ist kein Startpunkt- oder Routenradius |
| Sortierung | `name`, `distance`, `duration`, `difficulty`, `elevation_gain`, `elevation_loss`, `like_count`, `created`, `date`, jeweils auf-/absteigend |
| Oberflächendefault | Trail-Liste `created ASC`, Karte `created DESC` |
| Pagination | Meilisearch-`page` und `hitsPerPage`; ohne gespeicherten Wert startet die Liste mit 25. Ein vorhandenes `paginationItems` wird beim Mount nach Darstellungsmodus gebucketet: Karten `12/24/48/96`, sonst `10/25/50/100`; gespeicherte 25 werden in Kartenansicht zu 24 |

Bei nichtleerem `q` steht die explizite Sortierung in den heutigen
Rankingregeln erst nach Wort-, Tippfehler-, Nähe- und Attributrelevanz. Diese
Mischung wird als `legacy_ranking_v0` erfasst; SRCH0 behauptet nicht
nachträglich, der sichtbare Sortkey sei dabei immer der Primärschlüssel.

Die Karte setzt Liste, Cluster und ID-Details aus mehreren Requests ohne
gemeinsamen Snapshot zusammen. Die Fixtures laufen deshalb auf einem
unveränderlichen Dataset und prüfen die drei Ergebnismengen getrennt. Eine
snapshotgebundene Zielsemantik wird daraus nicht abgeleitet.

Die alte Listen-URL liest nur `author`, je den ersten `category`- und
`subcategory`-Wert sowie `page`. Der komplette Filter liegt ansonsten unter
`trailListFilter`: Sobald `author`, `category` oder `subcategory` in der URL
vorhanden ist, wird dieses Filterobjekt vollständig verworfen. Davon unabhängig
überschreiben die separaten Storagekeys `sort` und `sort_order` den so
ermittelten Filter **ohne** Sanitizing; `paginationItems` wird separat gelesen
und nach Darstellungsmodus gebucketet. Die Präzedenz ist damit feldweise und
nicht pauschal „URL vor Storage“. Ein `q`- oder `sort`-Queryparameter ohne
V1-Kennung hat in dieser URL keine Suchwirkung. Diese Tatsachen sind
Legacyevidenz und keine Empfehlung für den späteren URL-Vertrag.

„Geteilt“ bezeichnet in der Suchmatrix ausschliesslich einen Actor-Share, der
im Indexfeld `shares` und im Tenant-Scope vorkommt. Ein Link-Share-Token erlaubt
PocketBase-Detailzugriff auf genau einen Trail und erweitert das
Meilisearch-Suchuniversum nicht.

### Zusätzliche Engine-Consumer

Die Actor-Suche und die Upload-Duplikatprüfung dürfen nicht in der allgemeinen
Trailmatrix verschwinden:

- `GET /api/v1/search/actor` weist nicht authentifizierte Requests bereits an
  der Route mit `401` ab. Für authentifizierte Requests erlaubt der heutige
  Tenant-Token den vollständigen `actors`-Index; `includeSelf=false` ergänzt
  den Filter `id != <eigener-actor>`. Ein syntaktisch gültiger Remote-Handle
  nimmt zuerst den direkten ActivityPub-Zweig; dort hat `includeSelf` heute
  keine Wirkung. Ein anonymer Aufruf des rohen Actor-Proxys ist mangels
  `actors`-Regel im anonymen Token abzulehnen. Diese vier Fälle erhalten
  getrennte Fixtures.
- `findDuplicate` führt heute eine authentifizierte, tenantgescopte
  Placeholder-Suche mit leerem `q`, Offset 0, ohne Sortierung und mit dem
  Engine-Defaultlimit 20 aus. Der Helper liest exakt `id`, `name`,
  `author_name`, `domain`, `distance`, `elevation_gain`, `elevation_loss` und
  `_geo`. SRCH0 macht Limit und Retrievalliste explizit, ändert die Semantik
  aber nicht. Ein passendes Duplikat erst an Position 21 bleibt daher ein
  benannter `preserve`-/`legacy_only`-Fall und kein stiller Vollscan.

Ein unbekannter Wert aus den separaten Storagekeys `sort` oder `sort_order`
wird heute raw bis zur Engine gereicht. Das Baselinefixture hält den daraus
entstehenden Fehler fest; der Zielzustand ist ein klassifizierter
`correct`-Fall mit `delivery_owner: SRCH-COMP`. Browser-Sanitizing und
serverseitige Ablehnung freier Sortkeys werden als zwei Grenzen getestet.

### Bewusste Lücken statt falscher Parität

Security besitzt Vorrang vor beobachtetem Altverhalten. Ein Treffer, den der
[Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/)
für den Principal nicht erlaubt, wird nie als zu erhaltende Parität
klassifiziert. Das Fixture hält in diesem Fall synthetisch den beobachteten
Gap und die erforderliche Erwartung fest; die Runtimekorrektur gehört zu
`SEC-VIS-0`.

Die Legacy-Matrix in Abschnitt 15 des
[Trail-Suchvertrags v1](/develop/specs/trail-search/contracts/trail-search-v1/)
nennt zusätzlich die lokale inklusive Datumssemantik und gültige Koordinaten
mit Wert `0` als bewusste Korrekturen. SRCH0 muss beide Fälle reproduzieren und
klassifizieren, ändert sie aber nicht vor dem serverseitigen Normalisierer in
SRCH-COMP. Dadurch werden sie weder vergessen noch versehentlich als heutige
Semantik ausgegeben.

Der Kartendefault verwendet am Ausgangscommit für `elevationLossLimit`
versehentlich `max_elevation_gain` statt `max_elevation_loss`. SRCH0 erfasst
diesen Surface-Gap als `correct` mit `delivery_owner: SRCH-COMP`; nur
unabhängige Werte im Referenzdataset verhindern, dass er im Fixture unbemerkt
gleich aussieht.

Der allgemeine First-Party-Helper `search_store.searchTrails` legt
`attributesToRetrieve` ausserhalb von `options` ab; der Proxy reicht dieses
Feld deshalb nicht an Meilisearch weiter. Der Baselinefall hält die tatsächlich
breitere Engineantwort fest, aber nicht als öffentliche DTO-Zusage. Die
allowlistete Responsegrenze ist `correct` mit `delivery_owner: SRCH-COMP`.

## Normative Quellen

- Die [gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/)
  definieren Bestandsparität, Unknown/Presence, Migration und Rollback.
- Der [Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/)
  definiert die erlaubte Treffermenge je Principal und die Owner der bekannten
  Securitygaps.
- Der [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
  konsumiert die SRCH0-Evidenz. Seine Legacy-Tabelle ist ein verpflichtender
  Abgleich, darf aber beobachtetes Altverhalten nicht rückwirkend erfinden.
- Der [STATE1-Vertrag](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
  definiert den späteren allgemeinen Generationen-, Recovery- und
  Rollbackvertrag. SRCH0 verwendet nur den dort abgegrenzten lokalen
  Offline-Einmallauf und führt keine vorgezogene verteilte Control Plane ein.
- Der [Delivery-Katalog](/develop/specs/trail-search/delivery/) bleibt alleiniger
  Owner von Arbeitsreihenfolge und globalem Abhängigkeitsgraph.
- Die Evidenzseite ist datierter Input und ausdrücklich nicht normativ.

Bei einem Konflikt gilt: Security vor Parität, normative Produktinvariante vor
beobachtetem Implementierungsunfall und explizit klassifizierte Korrektur vor
unbegründetem Golden-Update.

## Abhängigkeiten und Freigabegates

SRCH0 besitzt keine Implementierungsabhängigkeit. Die Unit-, Fixture- und
Shadowarbeiten können auf dem heutigen Backend beginnen. Die reale
Engineintegration läuft gegen die beiden belegten Ausgangsprofile 1.11.3 und
1.36.0 und speichert je Lauf einen Hash der relevanten Indexsettings. Das
friert den Bestand ein, ist aber noch keine allgemeine Betreiberfreigabe. Den
Upgradepfad und das spätere Zielprofil entscheidet M1 separat.

SRCH0 führt keine neue Capability ein. Seine produktive Aktivierung ist aber
ein Index- und Verhaltens-Cutover und besitzt deshalb ein eigenes Releasegate.
Der technische Reindex darf unabhängig davon gebaut und auf Fixtures oder
einem nicht exponierten Kandidaten ausgeführt werden. Der Produktionslauf
beginnt erst, wenn SRCH-COMP am auszuliefernden Build grün ist und SEC-VIS-0
Netzcontainment sowie rotationsbereite Search-Credentials bestätigt. Der
Ingress bleibt darüber hinaus bis zur tatsächlichen post-terminalen Rotation
und ihrem Negativtest geschlossen. Beide Bausteine dürfen die SRCH0-Matrix
vorher konsumieren; der Listenreindex ist Teil desselben atomaren SRCH0-Laufs
und kein separates Werkzeug. Keine von SRCH0 erst erfundene Credential-,
Observer- oder Zeitinfrastruktur ist Voraussetzung für das Erzeugen des
Kandidatensatzes.
Die folgenden Belege müssen gleichzeitig vorliegen:

1. Korpus-, Implementierungs- und Reindex-Systemtests sind für beide
   Ausgangsprofile grün.
2. **Alle** ACL-/Security-Zielerwartungen der SRCH0-Matrix sind im gebundenen
   CI-/Staging-`fixture-verification.json` am freizugebenden DB-/Webartefakt und
   Profil über den realen SvelteKit-Pfad grün; für den Live-Swap gibt es kein
   `pending_owner`. Die Produktionsinstanz wird dafür nicht als Testdaten- oder
   ACL-Orakel verwendet.
3. SRCH-COMP ist gegen das synthetische Wegwerfindexset mit exakt denselben
   Zielprofilen im geschlossenen Aktivierungsprofil `srch0_compat` vollständig
   grün und als cutover-bereites Artefakt identifiziert; alle dort deaktivierten
   V1-/Federationcapabilities sind als negative Releasefixtures gebunden;
   der Produktionskandidatensatz ist getrennt dokumentweise verifiziert. Der
   CI-/Staging-Aktivtest bestätigt, dass der gemeinsam freizugebende Build keine
   freie Engine-Requestform annimmt; insbesondere erreicht
   clientgeliefertes `filter`, `sort`, `attributesToRetrieve` oder ein
   Indexname die Engine nicht.
4. `SEC-VIS-0` bestätigt aktuelle Tenant-/Restriction-Grenzen, die private
   Engine-Netzgrenze, bereitstehende Search-Credentials und den ausführbaren
   Rotationsschritt. Die eigentliche Rotation samt Alt-Token-Negativtest folgt
   erst nach der SRCH0-Terminalentscheidung und vor Ingress. SEC-VIS-0 liefert
   zwingend das qualifizierte Listenziel-Superset `sec-vis0-list-v1`; dessen
   Revision muss vor SRCH0-Planpublikation feststehen und wird im selben
   Zwei-Paar-Swap statt in einem separaten Listen-Reindex aktiviert.
5. `difficulty_known` ist kein öffentliches DTO-Feld und kein akzeptierter
   clientseitiger Filter oder Sortkey; nur der serverseitige Adapter darf es
   zum Sortieren einsetzen.
6. `SEC-VIS-0` besitzt als `delivery_owner` die produktiven Varianten von
   Root-Compose, `docs/public/setup.sh`, Quickstart und manueller Docker-
   Installation. Deren gerenderte Konfiguration veröffentlicht keinen
   Meilisearch-Hostport und enthält einen Migrationshinweis für bestehende
   Installationen. Das Root-Entwicklungsprofil ist eindeutig als lokal
   gekennzeichnet und bindet einen optionalen Port höchstens an `127.0.0.1`.

ACL-Fälle, deren erforderliches Ergebnis der heutige Pfad nicht erfüllt,
blockieren nicht die Erstellung des Korpus und des Shadowkandidaten. Sie werden
als `correct`, `delivery_owner: SEC-VIS-0` und
`activation_gate: srch0_live` geführt und blockieren damit zwingend den
Produktionsswap. Das ist keine zyklische Implementierungsabhängigkeit:
SRCH-V1, SRCH-COMP und SEC-VIS-0 konsumieren die intern fertige SRCH0-Matrix;
erst ihre cutover-bereiten Nachweise öffnen die gemeinsame Liveaktivierung.
SRCH-COMP muss vorher nicht auf dem fehlerhaften Altindex öffentlich aktiv
sein: Seine V1-/Proxytests laufen mit dem zu aktivierenden Build auf einem
Wegwerfindex mit identischem Zielprofil. Der Produktionskandidat wird separat
vollständig gegen die PocketBase-Projektion verglichen. Der interne Shadowbeleg
entsteht in CI beziehungsweise Staging auf einem unveränderlichen Testbestand.
Ein Produktionslauf und sein Schreibstopp beginnen erst, wenn das Livegate in
demselben Wartungsfenster erfüllbar ist; ein alter Staging- oder
Produktionskandidat wird nicht über die Implementierungszeit gerettet.

## Vorgesehene Änderung und Schnittstellen

### Artefaktlayout

Die Implementierung legt die gemeinsame Wahrheit sprachneutral ab:

```text
testdata/trail-search/srch0/v1/
  schema.json
  manifest.json
  schemas/reindex-plan-v1.schema.json
  schemas/reindex-verification-v1.schema.json
  profiles/trails-legacy-v0.json
  profiles/trails-legacy-srch0-v1.json
  profiles/lists-legacy-v0.json
  profiles/lists-legacy-srch0-v1.json
  profiles/legacy-actor-v0.json
  README.md
  datasets/*.json
  cases/state/*.json
  cases/projection/*.json
  cases/compiler/*.json
  cases/search/*.json
  cases/mutation/*.json
  cases/browser/*.json
  cases/migration/*.json
```

`schema.json` ist das ausführbare Fixture-JSON-Schema. Die beiden Schemas unter
`schemas/` werden in den CLI-Build eingebettet: Das Planschema beschreibt
`plan.json`; `reindex-verification-v1.schema.json` ist eine geschlossene, über
`contract` diskriminierte `oneOf`-Union für Index-, Fixture- und
Produktionsverifikation, Vorwärts-/Reverse-Intent, Submit-Attempt, Admission,
Decision, Terminalbeleg, den eingebetteten `swap_effect_proof_v1`-Beleg und das
`srch0-engine-restore-scope.v1`-Manifest sowie
`opaque_legacy_baseline_v1`. Jeder gelesene oder publizierte
Kontrollrecord muss genau einen Zweig erfüllen; ein unbekannter Contract oder
mehrere passende Zweige sind ungültig. Die fünf Profile enthalten die belegten
Trail- und Listen-Baselineprofile, beide SRCH0-Zielprofile sowie das heutige
Actorprofil. Die Zielprofile sowie Trail- und Actor-Baseline besitzen
versionierte Vollprojektoren und sind aus PocketBase wiederherstellbar. Das
Listen-Baselineprofil ist ausdrücklich die unten definierte opake Engine-
Rollbackbaseline und keine aus PocketBase ableitbare Projektion.
`trails-legacy-v0` erhält dabei ausdrücklich die heutige Unknown-Abbildung;
`trails-legacy-srch0-v1` enthält die korrigierte Projektion.
`lists-legacy-v0` bindet nur Primärschlüssel, Settings und das bekannte
Dokumentformat der heutigen Enginebytes; es besitzt weder einen PocketBase-
Projektor noch einen Origin-Oracle. `lists-legacy-srch0-v1` enthält die lokale
Aggregatprojektion und selbst keinen Netzclient. `manifest.json` zählt alle
erforderlichen Case-Familien und verhindert, dass ein leer gewordenes
Verzeichnis als grüne Suite gilt. Das geschlossene Profilmanifest enthält
Primärschlüssel, Searchable-, Filterable-, Sortable-, Displayed-Attribute und
Rankingregeln in ihrer wirksamen Reihenfolge. Es bindet ausserdem die nicht
frei konfigurierbaren Laufzeitkonstanten
`post_response_release_deadline = 5 s`,
`engine_submit_deadline = 5 s`,
`task_observer_poll_period = 250 ms`,
`task_observer_request_deadline = 500 ms`,
`task_head_detection_budget = 1 s` und
`offline_task_terminal_timeout = 120 s`. Eine Änderung dieser Werte erhöht
Manifest- und Vertragsrevision; Umgebungsvariablen dürfen die Beweisgrenzen
nicht aufweiten. Go-Runtime, SvelteKit-
Readiness, Testharness und Reindex-CLI konsumieren dasselbe Artefakt; keine
Schicht führt einen handgeschriebenen zweiten Settings-Hash. `README.md`
enthält die aus denselben Fixtures erzeugte menschenlesbare Bestandsmatrix;
CI lehnt Drift zwischen Matrix, Manifest und Fällen ab.

Der Legacy-State-Evaluator und -Compiler werden aus dem Store in reine
TypeScript-Funktionen unter `web/src/lib/search/legacy/` extrahiert. URL,
Storage, Surface, Principal, Katalog und dynamische Limits sind explizite
Parameter; die Funktionen lesen weder Browserglobals noch Svelte-Stores. Die
produktiven Aufrufer verwenden danach dieselben Funktionen wie das Harness.
Diese Extraktion darf für `preserve`-Fälle keine Semantik ändern.

Die Go-Projektion erhält eine isoliert testbare Difficulty-Abbildung. Ein
unbekannter Wert wird als explizites JSON-`null` im Legacy-Indexdokument
geschrieben. Das ist für partielle `UpdateDocuments` wichtig: blosses Weglassen
würde einen früheren numerischen Wert im Dokument stehen lassen. Das interne
Feld `difficulty_known` enthält dafür `1` oder `0`, ist nur sortierbar und wird
weder als Filter noch im Treffer-DTO veröffentlicht. Seine Abschottung ist
Bestandteil des Livegates und nicht bloss eine Annahme über gutartige Clients.

Die Trailprojektion besitzt drei ausdrücklich verschiedene, gemeinsam
getestete Formen:

- `FullTrailDocument` enthält Basisfelder sowie deterministisch sortierte
  `shares`, `likes` und `like_count`. Create, Vollreindex und eine vollständige
  Autor-Neuprojektion verwenden diese Form.
- `TrailCorePatch` enthält nur Basisfelder. Ein partielles Update muss
  vorhandene Relationsfelder im Engine-Dokument erhalten.
- `TrailSharesPatch` und `TrailLikesPatch` ändern nur `shares`
  beziehungsweise `likes` plus `like_count`.

Der Kandidatenvergleich erzeugt immer `FullTrailDocument`. Ein Mutationstest
beweist, dass Core-Update und Relationspatches denselben materialisierten
Endzustand wie ein anschliessender Vollreindex ergeben. Damit wird die heutige
`includeShares=true|false`-Verzweigung beschrieben und getestet, statt zwei
unterschiedliche Payloads fälschlich als „dieselbe Projektion“ zu bezeichnen.

### Fixture-Vertrag `wanderer.srch0/v1`

Jede Fixturedatei ist ein einzelnes JSON-Objekt mit
`additionalProperties: false`. Folgende Felder sind Pflicht:

| Feld | Vertrag |
| --- | --- |
| `schema_version` | exakt `wanderer.srch0/v1` |
| `case_id` | stabile ID `SRCH0-<FAMILIE>-NNN`; wird nie neu verwendet |
| `family` | `state`, `projection`, `compiler`, `search`, `mutation`, `browser` oder `migration` |
| `classification` | `kind`, `migration_disposition`, `delivery_owner`, `activation_gate`, Begründung und gegebenenfalls Release-Hinweis |
| `baseline` | Commit, Surface, getestete Engineprofile und Settings-Fingerprint |
| `dataset_ref` | Referenz auf genau ein synthetisches, versioniertes Dataset |
| `context` | Principal, Locale, Zeitzone, Präferenzen und Surface |
| `input` | rohe Eingabe der Familie, ohne vorherige Normalisierung |
| `baseline_expected` | am Ausgangscommit reproduzierter Zustand |
| `expected` | nach der klassifizierten Änderung erforderlicher Zustand |
| `allowed_delta` | exakte JSON-Pointer, an denen beide Erwartungen abweichen dürfen |
| `evidence` | Codeanker oder begründeter normativer Verweis |

Roh-URL und LocalStorage bleiben Strings. Damit doppelte Querykeys, ungültiges
JSON und die historische Präzedenz testbar sind, speichert ein State-Fixture
keine bereits geparsten Ersatzobjekte. Erwartungsobjekte verwenden je nach
Familie folgende Felder:

- `legacy_state`: vollständig ausgewerteter `TrailFilter`, Surface-Default,
  Seite und Seitengrösse;
- `engine_request`: `q`, bytegenauer Legacyfilter, Sortierung und Pagination;
- `projection`: erlaubte Indexfelder einschliesslich `difficulty: null` und
  internem `difficulty_known`;
- `result`: `hit_groups`, `total`, `page` und `total_pages`;
- `browser_state`: wirksame URL-, Storage- und Historywerte;
- `mutation_state`: erwartete Projektion nach Create/Update/Delete und
  Relationsänderung sowie die abzuwartende Engine-Task;
- `diagnostics`: stabile Codes, keine freien Fehlermeldungen als Oracle;
- `migration_state`: Kandidatenindizes, Taskausgang, Zähler, Settings-Hashes und
  aktiver Index nach Cutover oder Rollback.

`hit_groups` ist eine geordnete Liste von ID-Gruppen. Eine einelementige Gruppe
ist exakt geordnet. Mehrere IDs in derselben Gruppe dürfen am Legacyprofil nur
dann die Reihenfolge tauschen, wenn der heutige Sort-/Rankingvertrag keinen
Tie-Breaker besitzt. SRCH-V1 darf diese Gruppe später mit einem eigenen
klassifizierten Contractfall verengen. Processing-Zeit, Engine-Task-IDs,
zufällige Empfehlungen, signierte Tokens, Dateipfade und lokalisierte Texte
werden nie als Goldens gespeichert.

### Klassifikation und Golden-Regel

Jeder Fall besitzt genau eine Klasse:

| `kind` | Bedeutung | Zulässige Differenz |
| --- | --- | --- |
| `preserve` | autorisiertes produktives Verhalten bleibt erhalten | `baseline_expected` und `expected` sind semantisch identisch; `allowed_delta` leer |
| `correct` | belegter Fehler wird durch den genannten Owner korrigiert | nur deklarierte Pointer; Regression, Migration und gegebenenfalls Release-Hinweis erforderlich |
| `remove` | wirkungsloser Implementierungsunfall verschwindet | Engine-/UI-Zwischenzustand darf sich ändern, Ergebnisorakel nicht |

Davon getrennt beschreibt `migration_disposition`, ob der Fall
`v1_equivalent`, `legacy_only` oder `non_productive` ist. Historisch
widersprüchliche Accesskombinationen bleiben so ausführbar, ohne eine
automatische V1-Abbildung zu behaupten. Der Mock-Preview ist `remove` plus
`non_productive`.

`delivery_owner` ist `SRCH0`, `SRCH-COMP`, `SEC-VIS-0` oder ein anderer bereits
im Delivery-Katalog definierter Baustein. `activation_gate` ist exakt `none`
oder `srch0_live`. Alle ACL-/Bypassfälle und alle Fälle, die
`difficulty_known` oder die lokale Föderierte-Listenaggregation berühren,
tragen `srch0_live`; bewusst erst durch
SRCH-COMP korrigierte Datums-, URL- oder Nullkoordinatenfälle tragen `none`.
Die SRCH0-CI besitzt drei getrennte Gates:

1. Das **Korpusgate** validiert alle Fälle, Owner und erforderlichen
   Erwartungen. Es akzeptiert keinen unklassifizierten Gap.
2. Das **Implementierungsgate** verlangt für alle Fälle mit
   `delivery_owner: SRCH0` das Zielergebnis. Fälle anderer Owner erscheinen mit
   stabiler ID als `pending_owner`, nicht als still übersprungene Parität.
3. Das **Livegate** verlangt für jeden Fall mit
   `activation_gate: srch0_live` das Zielergebnis unabhängig vom Owner. Ein
   solcher Fall darf in einem Cutoverreport nie `pending_owner` sein.

Ein Golden wird nie aus der Ausgabe des gerade getesteten Systems
überschrieben. Jede Änderung an `expected` benötigt Klassifikation,
`allowed_delta`, Begründung und Evidenz im selben Review. Da V1 unbekannte
Felder ablehnt, erzeugt jede Änderung der Fixtureform oder Feldbedeutung
`wanderer.srch0/v2`. Neue Fälle und neue Werte innerhalb bereits ausdrücklich
offener Listen ändern dagegen nicht das Schema.

### Synthetischer Referenzbestand

Der gemeinsame Datasetkorpus enthält keine Produktionsdaten und mindestens:

- anonymer Principal sowie lokale Actor Alice und Bob;
- lokale öffentliche, eigene private, fremde private und mit Alice/Bob
  geteilte Trails;
- verifizierbare synthetische Remote-, unverified-, revoked- und deleted-Fälle
  für die Securitymatrix;
- einen nur als Federation-Stub vorhandenen Listenrecord mit
  `needs_full_sync = true` und leerer Trailrelation sowie eine vollständig
  materialisierte föderierte Liste mit zwei referenzierten Trails;
- zwei Kategorien, mehrere Subkategorien, einen Trail ohne Subkategorie,
  überlappende Tagnamen und actorbezogene ausgeblendete Taxonomien;
- je einen gültigen Difficultywert sowie fehlenden, leeren und unbekannten
  Quellwert;
- eindeutige Werte und Gleichstände für jede Sortierachse;
- numerische Werte direkt auf, unter und über Rundungs- und Slidergrenzen;
- Datumswerte an normalem Tag, DST-Wechsel und Tagesende;
- Startpunkte im normalen Bereich sowie Latitude oder Longitude `0`;
- mindestens 2,5 Seiten bei Seitengrösse 2, damit Seite, `total` und
  Reihenfolge unabhängig geprüft werden.

IDs, Timestamps und Katalogrevisionen sind feste lesbare Testwerte. Kein Test
bezieht Kategorien, Benutzer oder Zeit aus einer laufenden Entwicklerinstanz.
Ein separat versionierter deterministischer Generator erzeugt für Cluster-
und Paginggrenzen 10'001 abgeleitete Trails; diese Masse wird nicht als
handgeschriebenes Golden dupliziert.

### Mindestabdeckung

| Familie | Pflichtfälle und Assertions |
| --- | --- |
| State | Listen-/Karten-Default, feldweise URL-/Storage-Präzedenz, defektes Storage, Snapshot-Restore, raw `sort`/`sort_order`, Darstellungsmodus-Buckets, separate Page-Keys und wirkungslose Legacy-URL-Keys |
| Volltext | leere Query, eindeutiger Treffer je Suchattribut, mehrere Tokens und ein Rankinggleichstand |
| Access/ACL | anonymous/authenticated × local/remote × public/private/owned/Actor-Share/revoked/deleted; Link-Share nur als Detailzugriff; Hits, `total` und DTO gemeinsam |
| Taxonomie | Parent, Child, gleicher/fremder Parent, kein Child, unbekannte ID und actorbezogene Präferenz |
| Tags/Booleans | ein/mehrere Tags, `liked` neutral/aktiv, `completed` ausgelassen/true/false |
| Ranges/Datum | anonyme Fallback- und actorbezogene dynamische Limits, jede Achse einzeln und kombiniert, Floor/Ceil, offenes Maximum, Min grösser Max, Tagesgrenzen |
| Difficulty | drei bekannte Werte, missing/leer/unbekannt, Defaultmenge, echte Teilmenge, Trefferanzeige und beide Sortierrichtungen |
| Geo | kein Radius, normaler Startpunkt, genau eine Klausel im Ziel, Koordinate `0` als klassifizierter SRCH-COMP-Fall, Karte ohne Umdeutung |
| Sort/Paging | alle neun Keys in beide Richtungen, Listen-/Kartendefault, Gleichstandsgruppen, erste/mittlere/leere Seite |
| Surfaces | Liste, Kartenliste, Clustereligibility einschliesslich 10'000er-Grenze, lokales und live weitergereichtes Remote-Profil, Actor-Route mit Handle-/`includeSelf`-Zweigen, bekannte Multi-Search-Form und Empfehlungseligibility |
| Upload-Duplikatprüfung | authentifizierter Tenant-Scope, leere Placeholder-Query, Offset 0, implizites Limit 20, kein Sort sowie exakt die acht gelesenen Felder; ein 21.-Treffer-Fall friert die heutige Begrenzung ein |
| Mutationen | Trail Create/Update/Delete, Public→Private, Actor-Share add/revoke, Like add/revoke sowie Autor-, Tag-, Taxonomie- und föderierte Listenmaterialisierungsänderung; Core-Patch erhält Relationsfelder, Full-Reindex ergibt denselben Endzustand; vor dem Oracle Task-Success abwarten |
| Negative API | unbekannter Index/Feld/Sortkey, freie Retrievalfelder und direkter Bypass als inventarisierte, nicht zugesagte Proxyfälle |
| Migration | Abbruch vor Aufbau, während Batch, vor/nach Swap, erneuter Lauf, Verifikation und Rückswap ausschliesslich vor Aktivbeleg |

### Technische Korrektur: `_geoRadius`

Der Legacycompiler berechnet einmalig, ob der Startpunktradius aktiv ist, und
hängt höchstens eine Klausel an. SRCH0 ändert dabei weder Meter, Punktquelle
noch die Bedeutung zur routeweiten Suche. Für einen normalen gültigen Punkt
gilt:

```text
count(engine_filter, "_geoRadius(") == 1
```

Der Baselinefall enthält zwei identische Klauseln, der Zielfall eine. Treffer,
`total`, Sortierung und Seite müssen identisch bleiben; deshalb ist der Fall
`remove`, nicht `correct`. Die heutige Truthiness-Lücke bei Koordinate `0`
erhält einen separaten `correct`-Fall mit `delivery_owner: SRCH-COMP` und wird
nicht als Nebenwirkung dieses Deduplizierens eingeschmuggelt.

### Technische Korrektur: Unknown-Difficulty

Die Korrektur ist Ende-zu-Ende und verwendet folgende eine Semantik:

1. `easy`, `moderate` und `difficult` projizieren weiterhin auf `0`, `1` und
   `2`.
2. Fehlend, leer oder jeder andere String bedeutet fachlich `unknown`. Im
   heutigen Indexformat werden dafür `difficulty: null` und
   `difficulty_known: 0` geschrieben; bekannte Werte erhalten
   `difficulty_known: 1`.
3. Der Legacy-Default `[0,1,2]` bedeutet „keine Einschränkung“ und erzeugt
   keine Difficulty-Klausel. Dadurch bleiben Unknown-Trails im normalen
   Suchuniversum.
4. Eine echte Teilmenge erzeugt weiterhin `difficulty IN [...]`; Unknown
   erfüllt sie nicht. Das Legacypanel führt noch keine neue Unknown-Auswahl
   ein.
5. Bei Difficulty-Sortierung kompiliert der Legacyadapter zuerst
   `difficulty_known:desc` und danach `difficulty` in der gewählten Richtung.
   Unknown steht damit in beiden Richtungen innerhalb der heutigen
   `legacy_ranking_v0`-Sortierstufe zuletzt. Andere Sortkeys bleiben
   unverändert.
6. Das Such-DTO akzeptiert `difficulty: 0 | 1 | 2 | null`. Nur die drei
   bekannten Zahlen werden auf ein Trail-Enum abgebildet; `null`, fehlend oder
   ein anderer Wert bleibt `undefined`. Alle Trefferansichten verwenden dafür
   denselben lokalisierten Wert `trail-value-unknown`; ein Fallback auf
   `easy`, `difficult` oder einen leeren Wert ist unzulässig.
7. `difficulty_known` wird in regulären Suchhits nicht abgerufen oder nach
   aussen durchgereicht. Das Zielprofil schliesst es aus
   `displayedAttributes` aus, Trailantworten laufen
   durch die feste SRCH-COMP-/V1-DTO-Projektion, und die produktive API nimmt
   keine clientgelieferten Enginefelder an. Der Server darf das Feld
   ausschliesslich intern vor `difficulty` in den Legacy-Engine-Sort einsetzen.
   Der rohe Einzel-/Multi-Search-Pfad und alte Engine-Tokens müssen vor dem
   Zielindex-Cutover bereits deaktiviert oder vollständig gekapselt sein.
8. Der Quellrecord wird nicht auf `easy` oder einen neuen String
   umgeschrieben. Die Korrektur ist eine Projektions- und Lesesemantik. Eine
   spätere explizite Benutzerbearbeitung folgt dem normalen Trail-Write-
   Vertrag.

Das SRCH0-Zielprofil ergänzt `difficulty_known` zu den sortierbaren Attributen
und ersetzt den impliziten Displayed-Default `*` durch die geschlossene Menge
der im Korpus bestätigten First-Party-Retrievalfelder; `difficulty_known` und
`likes` gehören nicht dazu. Die vier Boundsfelder bleiben enthalten, weil die
serverseitige Bounding-Box-Hilfsabfrage sie mit privilegiertem Enginezugang
liest. Searchable- und filterable Attribute sowie die Rankingregeln bleiben
gegenüber dem Legacyprofil unverändert. Die exakte Displayed-Liste ist im
versionierten Profilmanifest hinterlegt, entspricht folgender geschlossener
Menge und wird gegen alle First-Party-Projektionen getestet:

```text
id, author, author_name, author_avatar, name, description, location,
distance, elevation_gain, elevation_loss, duration, difficulty,
category, category_id, category_icon, subcategory_id, is_federated,
federated_category_name, federated_subcategory_name, completed, date,
created, public, thumbnail, gpx, tags, polyline, domain, iri,
min_lat, max_lat, min_lon, max_lon, bounding_box_diagonal, _geo, shares,
like_count
```

Diese Liste ist ausschliesslich die interne Engine-Readback-Grenze für
serverseitige Bestandshelper; sie ist **kein** öffentliches Treffer-DTO. Der
SRCH-COMP-Adapter bildet daraus nur die im
[Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/#91-erlaubte-trailfelder)
erlaubten Felder, wandelt rohe Assetreferenzen in autorisierte lokale URLs um
und gibt insbesondere weder `gpx`, `polyline`, `shares`, Bounds, `_geo` noch
andere interne Projektionsteile ungeprüft aus. Bounding-Box- und Clusterhelper
lesen ihre geschlossene Teilprojektion ausschliesslich serverseitig. Der
generische Proxy ist zu diesem Zeitpunkt gekapselt und die Engine nicht direkt
erreichbar; daher erweitert die Displayed-Liste den V1-Wirevertrag nicht.

Der Produktionsvergleich verwendet nicht die Search API, sondern die mit dem
administrativen Reindex-Key aufgerufene Documents API. Das vom gepinnten
`meilisearch-go`-Client verwendete `POST /documents/fetch` erhält die
vollständige, aus dem Profilmanifest generierte Feldliste ausdrücklich.
`fields: ["*"]` ist verboten: Im POST-Body ist `"*"` ein wörtlicher Feldselektor
und keine Wildcard; auf 1.53.1 wurde deshalb ein leeres Objekt beobachtet. Das
ist weder eine `displayedAttributes`-Filterung noch ein SDK-
Serialisierungsfehler. Ohne `fields` liefert POST alle Felder, während GET
`fields=*` eine eigene Wildcardsemantik besitzt. Request-Capture des Clients
sowie rohe POST-Tests mit expliziter Hidden-Field-Liste auf 1.11.3 und 1.36.0
bleiben vor Freigabe offen und Teil der Engine-Matrix. Ein zusätzlicher
Versionstest belegt getrennt, dass Search-Hits nicht dargestellte Felder selbst
bei `attributesToRetrieve: ["*"]` auslassen. Eine Änderung in einem späteren
Engineprofil blockiert dessen Freigabe, erzwingt aber nicht vorsorglich die
Ausgabe interner Felder.

Die Reihenfolge im Manifest ist die oben angegebene Reihenfolge. Der
Settings-Fingerprint erfasst Primärschlüssel und die geordneten Listen aller
dieser Settings als SHA-256 über nach RFC 8785 kanonisiertes UTF-8-JSON;
Zeitstempel und Engineantwort-Metadaten gehen nicht in den Hash ein.

Die Unknown-Garantie von SRCH0 gilt ausschliesslich für lokal aus PocketBase
projektierte Dokumente. Eine alte Remoteinstanz hat einen unbekannten
Quellwert möglicherweise bereits irreversibel als numerische `0` ausgegeben;
lokal ist dieser Wert nicht von echtem `easy` unterscheidbar. Ein explizites
Remote-Negotiationsfeld wird hier bewusst nicht erfunden, weil Wiresemantik
ausschliesslich dem Trail-Suchvertrag v1 gehört. Unversionierte Profilfälle
tragen `migration_disposition: legacy_only`, die Diagnose
`legacy_unverifiable` und `delivery_owner: SRCH-COMP`. Sie werden nicht als
korrigiert behauptet und blockieren den lokalen SRCH0-Cutover nicht, wohl aber
jede globale Remote-Unknown-Zusage. Erst der V1-/Federationvertrag darf dafür
eine interoperable Presence-/Provenienzangabe und Capability-Aushandlung
festlegen.

### Technische Korrektur: föderierte Listenaggregate

Für `lists` ist `r.ExpandedAll("trails")` der einzige Aggregatinput. Der Adapter
setzt `trails` auf die Zahl dieser lokal persistenten Relation und bildet
`distance`, `duration`, `elevation_gain` sowie `elevation_loss` als exakte Summe
der vier Felder der lokal materialisierten referenzierten Trails. Das gilt
identisch für lokale und föderierte Listen. Eine leere Relation ergibt für alle
fünf Werte `0`, ausdrücklich auch bei `iri != ""`, Remote-Actor und
`needs_full_sync = true`. Dieses Flag ist ein Refreshhinweis und weder fehlender
Projektionsinput noch ein Readinessblocker. Einen Originwert, den PocketBase
nicht persistiert, erfindet der Adapter nicht.

Die heutige Verzweigung `documentFromRemoteRecord` und ihr Live-`POST` auf
`/api/v1/search/lists` entfallen aus Vollprojektion, Fingerprint, Reindex,
Vergleich, Recovery und inkrementeller Projektion vollständig. Ein Full-Sync,
der Trails lokal materialisiert oder die `lists.trails`-Relation ändert, ist ein
normaler projektionsrelevanter Commit und invalidiert über dieselbe
Abhängigkeitshülle genau das Listendokument. Erst eine tatsächlich persistierte,
aber nicht auflösbare erforderliche Trail-/Actorrelation endet deterministisch
mit `source_projection_incomplete`; ein Federation-Stub allein tut das nie.

Diese sichtbare Änderung ist ein eigener Fixturefall mit `kind: correct`,
`migration_disposition: legacy_only`, `delivery_owner: SRCH0` und
`activation_gate: srch0_live`. Seine `allowed_delta` enthält ausschliesslich
Trailzahl und die vier Aggregatfelder samt den unmittelbar daraus gerenderten
UI-Werten. Der Release-Hinweis lautet sinngemäss: „Föderierte
Listenstatistiken basieren ab SRCH0 auf lokal materialisierten Trails und können
bis zum Full-Sync `0` sein.“ Damit ist dies die dritte ausgewiesene
Bestandskorrektur; es gibt weder einen stillen Origin-Fallback noch einen
globalen Startupfehler für normale Federation-Stubs.

Go-Unit-Tests decken Vollindexierung und partielles Update ab. TypeScript-Tests
decken Default-/Teilfilter und Trefferkonvertierung ab. Ein realer
Meilisearch-Test beweist, dass `null` nicht auf `difficulty = 0` matcht, ohne
Difficultyfilter aber im Resultat und bei beiden Sortierrichtungen zuletzt
bleibt. API-Negativtests versuchen zusätzlich, freie Indexnamen, Filter, Sorts,
Retrievalattribute und insbesondere `difficulty_known` über beide alten
Proxyformen einzuschleusen. Nach dem SRCH-COMP-Cutover erreicht kein Versuch
die Engine oder beeinflusst `hits`, `total`, Reihenfolge oder Fehlertiming
anhand privater Dokumentwerte.

### Testschichten

| Schicht | Aufgabe | Darf nicht ersetzen |
| --- | --- | --- |
| JSON-Schema-/Manifesttest | Form, Referenzen, Case-ID-Eindeutigkeit und Mindestabdeckung | fachliche Ausführung |
| Vitest | Statepräzedenz, Sanitizing, Legacycompiler und Hitkonvertierung | reale Enginefilterung |
| Go-Unit | deterministische Trailprojektion und Null-Clearing bei Update | Browser- und ACL-Verhalten |
| Meilisearch-Integration | Settings, Filter, Rankinggruppen, `total`, Paging und Projektion auf 1.11.3 und 1.36.0 | serverseitige ACL |
| PocketBase/Meili-Mutation | Hook-Konvergenz für Create/Update/Delete und Relationsänderungen; jede Task bis Success verfolgen | unmittelbare Revocationgarantie aus SEC-VIS |
| SvelteKit/API-Integration | Tenant-Token, Präferenzfilter, bekannte Proxyformen und DTO-Grenze | Browser-History |
| Playwright | URL, Storage, Snapshot, Navigation, Liste und Karte | vollständige Engineversionsqualifikation |
| Reindex-Systemtest | Tasks, Kandidatenprüfung, Swap, Abbruch, Wiederholung und Rollback | M1-Upgrade-/Soaknachweis |

Tests melden mindestens Case-ID, Familie, Engine-/Settings-Fingerprint und den
kleinsten strukturellen Diff. Private Treffer oder Actor-IDs aus einer realen
Instanz dürfen weder im Report noch in Logs erscheinen.

Schema-, Manifest-, Go- und Vitest-Suite laufen auf jedem Pull Request. Die
kleine reale Engine-/API-Matrix läuft ebenfalls vor Merge. Beide
Ausgangsversionen, das 10'001-Trail-Korpus, Playwright und die vollständige
Crash-/Swap-Suite sind verpflichtende Merge-/Releasejobs; ein schneller
PR-Subset darf ihren letzten grünen Lauf nicht als aktuellen
Freigabenachweis ausgeben.

## Migration, Backfill, Cutover und Rollback

### Betriebsmodell und Vertrauensgrenze

Meilisearch ist für die SRCH0-Zielprofile vollständig aus PocketBase
ableitbar. Die einzige Ausnahme ist die bereits vorhandene
`lists-legacy-v0`-Rollbackbaseline: Ihre heutigen Remoteaggregate sind nicht
lokal rekonstruierbar und werden während des gestoppten Migrationslaufs opak
aus der Engine gebunden, niemals durch einen Origin-Dial nachgebaut. Der
einmalige Produktionsübergang läuft deshalb als lokale Offline-Operation mit genau einer
PocketBase-Datenbank, einer privaten Meilisearch-Instanz und einem
Reindexprozess. Öffentlicher Ingress, DB-Serveprozess, Webprozess und alle
projektionsrelevanten Worker sind dabei gestoppt. Ein Rolling Deployment,
parallele Writes, Remote-Engine ohne lokalen Plattenzugriff, Catch-up-Log oder
mehrere DB-Writer gehören nicht zu diesem Profil; sie benötigen STATE1/IDX3.

SRCH0 verwendet die heute vorhandene Engine-Authentisierung. Im Composepfad
erhält der One-shot `MEILI_MASTER_KEY` aus derselben Secretquelle wie der
heutige DB-Prozess. Der Beleg behauptet weder einen externen Credential-
Controller noch eine lückenlose historische Key-Inventur. Vor einer
öffentlichen Aktivierung muss `SEC-VIS-0` den direkten Enginezugang, die
Tenant-/Search-Token und die produktive Netzgrenze korrigiert und negativ
getestet haben. Diese Securityarbeit ist ein Releasegate, aber keine
technische Voraussetzung für Plan, Kandidatenaufbau oder Recovery.

Die Kandidatenstrategie bleibt trotz Schreibstopp bewusst erhalten. Ein
In-place-`AddOrReplace` würde während des Aufbaus alte und neue Dokumentformen
mischen, entfernte Quelldokumente nicht zuverlässig beseitigen, Settings und
Dokumentbatches nicht gemeinsam aktivieren und die einzige sofortige
Rückfallbaseline überschreiben. Der zusätzliche Plattenbedarf und der eine
atomare Swap sind für SRCH0 der vertretbare Preis für vollständige
Vorabverifikation und einen Rückswap vor Wiederöffnung der Writes.

Fresh Install, vorhandene Installation und verlorenes Meilisearch-Volume sind
keine verschiedenen Provenienzklassen. Der beobachtete Engine-/PB-Zustand
bestimmt ausschliesslich, welche der folgenden Operationen zulässig ist; eine
neue Zielsemantik wird daraus niemals ausgewählt. `auto` darf nur die committed
Revision recovern oder verifizieren, während `migrate` zusätzlich die unten
festgelegte ausdrückliche Operatorwahl verlangt:

Alle drei Operationen gelten ausschliesslich für ihre aktuell in `O`
partitionierten logischen UIDs. Eine explizit adressierte UID in `S` wird mit
`search_control_plane_owned_by_state1` vor State-/Enginewirkung abgewiesen;
gemischtes `auto` überspringt sie read-only. Grüne STATE1-Ownergates bleiben
Voraussetzung der späteren globalen Search-Readiness, sind aber bei
strukturell gültigem S/C-Lag kein Eintrittsgate der O-only-Reindexstufe.

Für die einmalige Listenmigration gilt zusätzlich das monotone historische
Prädikat `srch0_pair_activated_v1`: Die **aktiven** Felder der Records `lists`
und `trails` tragen beide den terminal committed Vertrag `srch0_compat_v1` und
die Zielprofile `sec-vis0-list-v1` beziehungsweise
`trails-legacy-srch0-v1`. Operationsmarker gehören ausdrücklich nicht zu
diesem Prädikat; ein späteres legitimes `ready -> recovering -> ready` einer in
`O` verbliebenen UID kann den bereits erfolgten Paar-Cutover nicht rückgängig
machen. STATE1 V1 kann aus dieser Paarmenge ausschliesslich `trails` übernehmen;
`lists` ist durch sein Trail-only-Generationen-/Membermodell nicht abgedeckt und
endet schon im Handoff-Preflight mutationsfrei mit
`state1_handoff_uid_unsupported`. Solange das Aktivierungsprädikat nicht gilt,
endet auch der einzig zulässige Trail-Handoff vor Head-, State- oder
Enginewirkung mit `search_pair_migration_required`. Der kanonische SRCH0-
Produktionslauf muss also zuerst beide UIDs unter Offline-Authority terminal
aktivieren.

Der anschliessende administrative Handoff verlangt getrennt
`srch0_pair_handoff_ready_v1`: Das monotone Aktivierungsprädikat gilt, ein zur
terminalen Paar-Aktivierungskette passendes `active.json` ist vorhanden, und auf
der Engine existiert keine reservierte physische UID
`<lists|trails>__srch0__<run-id>` mehr. Der Handoff bindet den Artifactdigest
in `handoff_srch0_pair_activation` und die unter seinem exklusiven Lock erneut
gelesene leere Kandidatenmenge samt Scope/Head in
`handoff_srch0_pair_cleanup_observation`. Diese dynamische Cleanup-/Headbindung
entsteht erst nach allen terminal erfolgreichen STATE1-Generationsbuildtasks
in der effektfreien Precommit-Stufe und verwendet denselben finalen Head wie
der Quieszenzbeleg; ein vor dem Build gelesener Head ist kein Handoffinput.
Die im Terminalbeleg enthaltenen
`state_revision`-Werte sind historische Aktivierungsfloors und werden nicht mit
den durch spätere Writes, Recoveries oder Handoffs erhöhten aktuellen CAS-
Werten gleichgesetzt. Dieser Beleg ist ausschliesslich ein administratives
Handoff-Gate, kein Startup- oder Search-Readinessinput; sein Verlust blockiert
keinen laufenden Owner, aber diese neue Authorityübergabe. Fehlt er, endet sie mit
`search_pair_activation_evidence_required` und verlangt das bereits zulässige
artifact-only `resume`; verbleibt eine Kandidaten- oder Rückfallbaseline, endet
er mit `search_pair_cleanup_required` und verlangt zuerst
`resume --cleanup`. Erst danach darf STATE1 im **einzigen ersten**
STATE1-Bootstrap exakt die Einermenge `["trails"]` übernehmen. `lists` bleibt in
diesem V1-Profil dauerhaft Offline-Authority; ein späterer Erweiterungsversuch
endet vor jeder Wirkung mit
`state1_incremental_authority_handoff_unsupported`. Ein dennoch vorgefundener Mix aus nicht erfülltem
`srch0_pair_activated_v1` und mindestens einer Paar-UID in `S` ist kein
zulässiger Reparaturzustand, sondern
`search_pair_migration_authority_conflict`; weder Offline noch STATE1 mutiert
ihn automatisch.

- `migrate`: Alle zu migrierenden Offline-Indizes existieren mit bekannten
  Baselineprofilen. PocketBase-ableitbare Baselines sind vollständig gegen ihre
  committed Legacyprojektion verglichen. Eine aktive `lists-legacy-v0` wird
  stattdessen unter demselben Schreibstopp als opake Enginebaseline erfasst und
  unmittelbar vor dem Swap erneut gegen diesen Snapshot geprüft; ein
  Kandidatensatz wird aufgebaut, geprüft und in einem Request atomar getauscht.
- `recover`: Ein oder mehrere Offline-Indizes fehlen oder ihr Dokumentbestand
  weicht bei bekanntem, PocketBase-ableitbarem aktivem Profil von PocketBase
  ab. Sie werden unter geschlossenem Ingress direkt und vollständig aus
  PocketBase aufgebaut, bevor Search-Readiness, projektionsrelevante Writes und
  Worker freigegeben werden. Eine fehlende oder von ihrem zuvor gebundenen
  opaken Snapshot abweichende `lists-legacy-v0` ist nicht recoverbar; sie endet
  mit `legacy_list_baseline_unrecoverable` und verlangt die Wiederherstellung
  eines exakt passenden Enginebackups. Ohne vorhandene Baseline gibt es keinen
  Rückswap und keinen bestätigungslosen Einweg-Cutover.
- `verify`: Das Offline-Zielprofil existiert bereits. Ein vollständiger Vergleich darf
  den PocketBase-Vertragsstand neu bestätigen, auch wenn das Auditverzeichnis
  verloren ging.

`auto` wertet ausschliesslich `recover` und `verify` aus und darf niemals eine
neue Vertrags- oder Profilrevision aktivieren. `verify` ist nur zulässig, wenn
logische UID, Engineprofil und PocketBase-State bereits dieselbe committed
Revision mit `operation_state = ready` nennen; eine nur runlokal vorhandene
Ziel-UID erfüllt dieses Prädikat nicht. Ein fehlender oder gedrifteter
PocketBase-ableitbarer Offline-Legacyindex wird auf seinem committed Profil
repariert und dieser Lauf endet ohne semantische Migration. Für
`lists-legacy-v0` gilt stattdessen der gerade definierte unrecoverable-Zustand.
Der SRCH0-Kandidatensatz entsteht ausschliesslich durch
einen separaten Operatorlauf mit `--mode migrate`, vollständig genanntem Ziel
und einer an die Zielvertragsrevision gebundenen Bestätigung. Eine ungeklärte
Baseline ist in keinem Modus migrierbar.

Ein dauerhafter Fresh-Marker, `engine_genesis` und eine `installation_id` sind
für diese abgeleitete lokale Datenhaltung nicht erforderlich und kommen in
Plan, Belegen oder Readiness nicht vor. Baselinefreie Initialisierung ist nur
zulässig, wenn für die betroffene UID weder Engineindex noch jemals committed
Profilzustand existiert und ihre PocketBase-Quellmenge leer ist; dann wird
direkt das konfigurierte Zielprofil als leerer Bestand committed. Das ist ein
beweisbar semantikgleicher Empty-Fall, kein aus Installationsalter abgeleiteter
Fresh-Beweis. Bei nichtleerer Quelle bleibt eine fehlende Legacylistenbaseline
unrecoverable.

### Implementierungsschnitt und Bedienpfade

Die Implementierung liegt im bestehenden DB-Binary als Cobra-Unterbefehl und
verwendet denselben PocketBase-Bootstrap, dieselben Projektionen und denselben
Meilisearch-Client wie der Serveprozess. Es gibt fünf Reindexbefehle, zwei
Search-Index-Guardbefehle und den gestoppten STATE1-Reconcilebefehl:

```text
/pocketbase search-reindex run    --dir /pb_data --mode auto --output <dir>
/pocketbase search-reindex run    --dir /pb_data --mode migrate \
  --target-contract srch0_compat_v1 \
  --target-profile trails=trails-legacy-srch0-v1 \
  --target-profile lists=sec-vis0-list-v1 \
  --confirm-cutover srch0_compat_v1 \
  --available-bytes <uint64> --capacity-path /meili_data/data.ms --output <dir>
/pocketbase search-reindex resume --dir /pb_data --run-dir <dir>
/pocketbase search-reindex status --dir /pb_data --run-dir <dir>
/pocketbase search-reindex status --dir /pb_data --orphans --full
/pocketbase search-reindex abort  --dir /pb_data --run-dir <dir>
/pocketbase search-reindex cleanup-orphan --dir /pb_data --uid <uid> \
  --expected-settings-fingerprint <sha256:...> \
  --expected-document-fingerprint <sha256:...>
/pocketbase search-index ensure   --dir /pb_data [--full]
/pocketbase search-index invalidate --dir /pb_data [--uid <logical_uid> ...] \
  [--restore-scope-manifest <file> --restore-scope-digest <sha256:...>] \
  --reason source_restore|engine_restore|engine_admin|operator
/pocketbase search-state1 reconcile --dir /pb_data --durable-only --until-ready
```

`--target-profile` ist die wiederholbare Form `logical_uid=profile_revision`;
der Zielvertrag muss alle betroffenen logischen UIDs eindeutig auflösen.
Ein Produktionsplan muss für `lists` genau das von SEC-VIS-0 gelieferte und
gegen dessen Sichtbarkeitsfelder qualifizierte Superset `sec-vis0-list-v1`
binden. `lists-legacy-srch0-v1` bleibt der vollständige SRCH0-Projektor für
nicht exponierte CI-/Staging-Kandidaten; er ist kein zulässiges produktives
Cutoverziel. Fehlt das qualifizierte Superset oder sein Manifestdigest, endet
der Lauf vor Planpublikation mit `security_list_profile_required`.
`--confirm-cutover` muss bytegleich der Zielvertragsrevision entsprechen; ein
boolesches `--yes`, TTY-Prompt oder `--mode auto` ersetzt diese Bestätigung
nicht. Ohne alle Migrationsparameter endet der Befehl vor Planpublikation mit
`migration_confirmation_required`. `run --mode auto` wählt nur `recover` oder
`verify` anhand der unten geschlossenen Zustandsmatrix. Jeder solche
Offline-Befehl ist owner-scharf, auch bei `S = []` und `C != []`: Er verlangt
widerspruchsfreie Authoritybindungen und gesunde Restore-/
Inkarnationsgrundlagen, mutiert und attestiert aber ausschliesslich `O`. Ein strukturell gültiger Dirty-/
Publisherlag von `S`/`C` blockiert diese Offline-Stufe nicht, sondern erscheint
im strukturierten Ergebnis als `stage = offline_repair_complete`,
`state1_drain_required = true` und gebundener `state1_cutoff`; `S` bleibt
bytegleich und der Befehl erzeugt keine
STATE1-Enginearbeit. Authority-, Restore-, Inkarnations- oder
Publisherstrukturfehler bleiben hart. Der Zwischenerfolg behauptet niemals
globale Search-Readiness, die weiterhin beide Ownergates verlangt. Der Plan
bindet `O/S`, Authority-/State-Revisionen und den beobachteten STATE1-Cutoff;
jeder Operationsmarker referenziert ihn über `operation_run_id`. Ein Auto-
`resume` erbt nach jedem Crash exakt diese O-only-Semantik und terminiert wieder
mit derselben Stage und demselben Drainfeld, bevor der getrennte STATE1-Befehl
beginnt. Für Migrationspläne darf `resume` mit
`--activate --activation-report <file>` den terminalen Aktivierungsschritt
ausführen; nach einem Terminalbeleg löscht nur das gesonderte
`resume --cleanup` die darin exakt gebundene veraltete UID. Bei der
Zwei-Paar-Migration ist dieser Cleanup vor dem einzigen in STATE1 V1
zulässigen Trail-Handoff verpflichtend und deshalb nur zulässig, solange beide Paar-UIDs noch
Offline-Authority besitzen. Der Befehl validiert `active.json`, bindet die dort
genannten physischen UIDs und Fingerprints, sendet Delete-Index-Tasks nur für
noch vorhandene gebundene UIDs, wartet sie terminal und verlangt danach die
Abwesenheit sämtlicher reservierter Paar-UIDs. Nach einem Crash ist eine bereits
fehlende exakt gebundene UID ein idempotent abgeschlossener Teilschritt; eine
fremde oder fingerprintabweichende UID wird nie adoptiert oder gelöscht.
`status` ist strikt read-only; `status --orphans --full` berechnet für
reservierte, ungebundene UIDs
ohne Dokumentpayload den Settings- und vollständigen Dokumentfingerprint.
`cleanup-orphan` ist der einzige Pfad für einen Orphan ohne Runverzeichnis. Er
verlangt das exakte UID-/Fingerprint-Tupel, das exklusive Lock bei gestopptem
DB-Prozess und beweist erneut, dass kein State-, Intent-, Terminal- oder
Routingrecord die UID referenziert; erst dann sendet er genau einen
Delete-Index-Task und wartet ihn terminal ab. Abweichung ist
`orphan_cleanup_guard_mismatch`, und logische UIDs sind immer verboten.
Settings und Felder müssen dabei einem bekannten eingebetteten SRCH0-Profil
entsprechen; ein unbekanntes Profil endet mit `orphan_profile_unknown` und
bleibt für manuelle Diagnose erhalten.
`abort` entfernt keine Daten; nach
einem Vorwärtsswap führt es gegebenenfalls den belegten Rückswap aus und
publiziert danach `abort.json`.

`search-state1 reconcile --durable-only --until-ready` ist kein normaler
Serveworker und kein semantischer Migrationspfad. Er erhält bei gestopptem
Serveprozess dasselbe exklusive Quelllock, friert den zu Beginn committed
STATE1-Katalogcutoff ein und verarbeitet ausschliesslich die bereits durable
`S`-/`C`-Arbeit bis einschliesslich dieses Cutoffs nach dem STATE1-Task-,
Publisher-, Delivery-, Fence-, Checkpoint- und Recoveryvertrag. Er erzeugt
weder Fachwrites noch Offline-Epochen, nimmt keine neue Arbeit an und ändert
kein Profil. Ohne STATE1-Rückstand ist er ein belegter No-op. Er endet erst mit
grünen Ownergates oder fail-closed mit dem konkreten STATE1-Recoverycode; ein
Crash wird aus denselben persistenten Attempts idempotent fortgesetzt. Das ist
der ausdrückliche Operator-Drain für Arbeit, die während
`search_manual_intervention` durable wurde, keine automatische Enginefreigabe
des laufenden `manual`-Serveprozesses.

`search-index invalidate` mutiert keine Engine. Es erhält dasselbe exklusive
Lock. Wiederholte `--uid` werden als kanonisch sortierte eindeutige Menge
gebunden; ohne `--uid` ist die Zielmenge atomar **jede aktuell offline
verwaltete UID**. Eine explizit genannte STATE1-UID beendet den gesamten Befehl
vor jeder Mutation mit `search_control_plane_owned_by_state1`; bei impliziter
Zielwahl wird sie read-only übersprungen. Gibt es dabei keine Offline-UID,
endet der Befehl mit demselben stabilen Code. Nur für die gültige Offline-
Zielmenge setzt der CAS den unten definierten dauerhaften Vollvergleichsgrund.
Er setzt `attested_projection_epoch` nicht vor und löscht keine Epochenhistorie;
der Grund allein verbietet den Epochen-Kurzpfad bis zum nächsten grünen
Vollvergleich.
Die Zielsemantik ist grundabhängig geschlossen:

- `source_restore` verbietet `--uid` und umfasst zwingend alle aktuell
  offline verwalteten UIDs, weil dieselbe zurückgespielte PocketBase-Quelle
  jede Projektion betreffen kann; vorhandene STATE1-UIDs verpflichten zugleich
  das unten genannte gemeinsame Restoregate;
- `engine_restore` ohne `--uid` umfasst alle Offline-UIDs. Eine Teilmenge ist
  nur mit **beiden** Restore-Scope-Argumenten zulässig. Der CLI liest die
  Manifestbytes aus `--restore-scope-manifest`, verlangt
  `additionalProperties: false`, `contract = srch0-engine-restore-scope.v1`,
  den Digest des wiederhergestellten Engine-Snapshot-/Volume-Artefakts und eine
  kanonisch sortierte eindeutige Liste
  `{logical_uid, physical_uid}` aller ersetzten Indizes. Er kanonisiert nach
  RFC 8785, prüft den SHA-256 bytegleich gegen `--restore-scope-digest` und
  verlangt, dass die logische Menge exakt `--uid` entspricht. Das
  versionierte Restorewerkzeug erzeugt diese Datei beim Restore; ein Digest
  ohne lesbare validierbare Manifestbytes genügt nicht. Menge und Digest
  erscheinen im strukturierten Befehlsresultat. Sie sind keine späteren
  Readinessinputs und benötigen kein zusätzliches Statefeld;
- `engine_admin` und `operator` dürfen eine explizite UID-Menge verwenden;
  ohne sie gelten wiederum alle Offline-UIDs.

Bei `engine_restore` und `engine_admin` setzt derselbe SQLite-Commit den
endpointweiten Taskguard zusätzlich auf `invalid` mit
`observation_reason = restore_or_admin`; `source_restore` und `operator`
benötigen diese globale Tasklog-Invalidierung nicht. Weil der Guard global ist,
verlangt seine spätere Reattestierung den read-only Vollvergleich aller UIDs in
`O`, auch wenn das validierte Restore-Scope nur einzelne Indizes reparieren
muss. Das erweitert weder die Engine-Mutationsmenge noch einen expliziten
UID-Selektor.

Ein Verstoss endet vor jeder Mutation mit
`search_invalidation_scope_invalid`. Insbesondere kann ein partielles
`source_restore` nie nicht genannte Offline-UIDs kurzpfadfähig lassen.
Die Zeile muss `operation_state = ready` und leere Operationsfelder besitzen;
andernfalls endet der Befehl mit `search_recovery_required` und zuerst ist der
vorhandene Recovery-/Cutoverzustand zu klären.
Der unterstützte PocketBase- oder Meilisearch-Restore- und Engine-Admin-
Runbookpfad muss diesen Befehl bei gestopptem DB-Prozess **nach** Einspielen
des Restores und **vor** dem nächsten Start ausführen. Betrifft derselbe
PocketBase- oder Engine-Restore einen Authority-Mix, ist dies ein gemeinsames
Maintenancegate: Der Befehl invalidiert alle Offline-UIDs, während die STATE1-
UIDs den Restore-/Inkarnationsvertrag aus Abschnitt 10.6 der State-Machine
durchlaufen; der DB-Prozess startet erst, wenn beide Teilgates durable
abgeschlossen sind, und bleibt bei Fehler vollständig gestoppt.

`search-index ensure --full` überspringt den Epochen-Kurzpfad und erzwingt
Taskquieszenz-Retry plus Vollvergleich. Zuerst validiert der Befehl Authority,
unterstützte Profil-/Vertragsrevision und Operationsstruktur für die
vollständige kanonische Offline-Zielmenge. Vor dem ersten Quieszenzversuch und vor dem
ersten Engine-Dokumentfetch klassifiziert er dann jede Zeile:

- Sind `ready`, Epochen, attestierter Zähler, Primärschlüssel und Settings
  kurzpfadfähig, ist der Grund leer und stimmen globaler Taskguard und aktueller
  Head, ist das flüchtige `--full` der einzige
  Vollvergleichstrigger. Nur für diese Menge `L` setzt ein einziger SQLite-
  Control-Commit dauerhaft `verification_required_reason = operator`. Jede
  Zeile in `L` benötigt die einheitliche Vorbedingung
  `state_revision <= 2^53-4`.
- Besitzt eine Zeile bereits einen dauerhaften Trigger wie Epochenlücke,
  fehlenden/abweichenden attestierten Zähler oder nichtleeren Grund, wird sie
  im selben Snapshot ohne Revisionswrite CAS-geprüft und folgt ihrem bereits
  verpflichteten Vergleichs-/Recoverypfad. Ein `recovering`-/`cutover`-Marker,
  Profil- oder Authorityfehler wird niemals mit `operator` überschrieben,
  sondern nach seiner eigenen Matrix behandelt.

Ein invalider/abweichender globaler Taskguard ist unabhängig von einzelnen
Zeilen ebenfalls bereits ein durabler Vollvergleichstrigger; er erzeugt keinen
zusätzlichen UID-Revisionswrite.

Der Commit bindet die gesamte Zielmenge und setzt `L` für alle Zeilen oder für
keine. Fehlt einer ansonsten nur durch `--full` getriggerten Zeile die
einheitliche `state_revision`-Vorbedingung oder verliert ein CAS, endet der
Befehl vor Quieszenzversuch und
Dokumentfetch mit `search_state_revision_exhausted` beziehungsweise dem
konkreten CAS-Fehler. Das Flag selbst ist kein späterer Readinessinput; sein
durabler Grund ist es aber. Ein Crash nach diesem Vorab-Latch, während oder
nach dem Vergleich erzwingt daher beim nächsten Start auch ohne `--full`
erneut den Vollvergleich. Dessen grüner Attestierungs-CAS leert den Grund;
erkannter Drift behält ihn und setzt nur dann wie unten normiert
`document_drift`, wenn das Feld vor der Diagnose noch leer war. Insbesondere
wird der vom selben `--full`-Lauf gesetzte `operator`-Grund bei Drift nicht zu
`document_drift` umgeschrieben; eine solche rein diagnostische Promotion würde
einen vierten Revisionsschritt verbrauchen. Nur die spätere Manual-
Phasenöffnung verlangt grüne STATE1-Ownergates.

Im Composepfad liefert SRCH0 ein Wrapper-Script, das mindestens folgende
Reihenfolge ausführt:

```bash
docker compose stop web db
capacity_path="/meili_data/data.ms" # im vollständigen Script aus MEILI_DB_PATH aufgelöst
available_bytes="$(docker compose exec -T search \
  df -B1 "${capacity_path}" | awk 'END { print $4 }')"
docker compose run --rm --no-deps --entrypoint /pocketbase db \
  search-reindex run --dir /pb_data --mode migrate \
  --target-contract srch0_compat_v1 \
  --target-profile trails=trails-legacy-srch0-v1 \
  --target-profile lists=sec-vis0-list-v1 \
  --confirm-cutover srch0_compat_v1 \
  --available-bytes "${available_bytes}" \
  --capacity-path "${capacity_path}" \
  --output /pb_data/search-reindex
```

Das vollständige Script prüft bereits vor dem Stopp die gerenderte Compose-
Konfiguration und bricht mit `public_search_engine_port` ab, wenn die produktive
Search-Instanz einen Hostport veröffentlicht. Es löst den tatsächlichen
`MEILI_DB_PATH` auf, standardmässig `/meili_data/data.ms`, und misst mit `df`
genau diesen Pfad. Beim heutigen Bind-File würde `/meili_data` dagegen das
Container-Overlay messen; beim Named Volume ist der aufgelöste Dateipfad
ebenfalls zulässig. Das Script validiert die Ausgabe als nichtleeren
vorzeichenlosen Dezimalwert und übergibt sie zusammen mit dem Messpfad explizit
an die CLI. Der DB-Container ist `FROM scratch` und mountet das Meilisearch-
Volume nicht; die CLI darf daher weder einen eigenen `statfs` vortäuschen noch
einen fehlenden oder ungültigen `--available-bytes`-Wert im Migrationsmodus
ersetzen. Die aktuelle Engine-Datenbankgrösse liest sie separat authentifiziert
aus `GET /stats` und bindet `databaseSize`, freie Bytes, Messpfad und
Messzeitpunkt in `plan.json`.

Root-Compose bleibt ein Entwicklungsprofil und ist erst nach dem oben
zugeordneten `SEC-VIS-0`-Umbau ein zulässiger lokaler Pfad; seine heutige
Veröffentlichung von `7700` ist kein Produktionsnachweis. Das
Ausgabeverzeichnis ist Operator-Audit, kein
Runtimeinput. DB und Web müssen es deshalb weder separat noch read-only
mounten; ein Verlust dieser Dateien macht einen bereits konsistenten Index
nicht unready.

DB-Serveprozess und Reindexbefehl verwenden
`<pb-data>/.wanderer-search-index.lock`. Der einzige unterstützte
Serveprozess hält das exklusive Dateilock für seine Lebenszeit. Nach Lock und
PocketBase-Migrationsbootstrap registriert der `OnServe`-Hook zuerst Guards und
Healthrouten und setzt den lokalen Zustand `search_initializing`. Er ruft
`event.Next()` auf, damit PocketBase Socket und Mux bindet, startet danach genau
eine an denselben Lockbesitz gebundene `ensure`-Goroutine und kehrt unverzüglich
zurück; erst die Rückkehr aus der gesamten `OnServe`-Kette lässt PocketBase
`http.Server.Serve(listener)` ausführen. `ensure` darf deshalb nicht synchron im
Hook die Serverloop blockieren. Ein separater Reindexbefehl erhält das Lock nur bei gestopptem
DB-Prozess. Ein zweiter Writer, ein nicht lokales Dateisystem ohne verlässliche
Locks oder ein Lockverlust endet mit `search_source_lock_unavailable` vor jeder
Engine-Mutation.

`SEARCH_INDEX_STARTUP` besitzt die Werte `auto` und `manual`; Default ist
`auto`, damit das heutige Self-Healing erhalten bleibt. Während
`search_initializing` liefert `GET /health` bereits `200`,
`GET /health/search` aber zunächst `503 search_recovery_required` und nach einer
eindeutigen Diagnose gegebenenfalls den spezifischeren Readiness-Code;
Searchrouten brechen vor dem ersten Enginezugriff ab. Alle
projektionsrelevanten mutierenden HTTP-Routen bleiben
mit `503 search_initializing` gesperrt; Cronjobs, Indexhooks und andere Worker
werden noch nicht gestartet. Der Write-Guard liegt vor der Fachmutation und
umfasst normale PocketBase-Recordendpunkte, Superuser-/Adminwrites,
Pluginimporte und interne Jobs; ein bloss unterdrückter nachgelagerter
Indexhook genügt nicht. Dadurch bleibt die Quelle für Fingerprint und
Recovery stabil, ohne dass ein langer Vergleich den zeitlich begrenzten
Docker-Liveness-Healthcheck reisst. Ein einziger prozesslokaler Phasenwechsel
nach grünem `ensure` öffnet Writes, Worker und Search-Readiness. Andere
read-only Routen dürfen vorher antworten, dürfen aber weder die Quelle noch
Meilisearch verändern.

`manual` besitzt daneben genau einen engen App-only-Ausgang aus dem
Bootstrapfenster. Hat ein gegen die währenddessen stabile Quelle vollständig
beendeter read-only Vollvergleich bei `operation_state = ready`, bekanntem
committed Profil, vollständig gebundener Authoritypartition und ohne offenen
Offline-Engine-Task **ausschliesslich auf den Offline-UIDs** Dokumentdrift
festgestellt, persistiert er unmittelbar den unten definierten Grund-Commit.
Das gilt unabhängig davon, ob die STATE1-Ownergates bereits grün sind. Erst
nach erfolgreichem Commit und zusätzlich grünen STATE1-Ownergates wechselt der
Prozess nach `search_manual_intervention`. Sind sie rot, bleibt er mit
dauerhaftem Driftgrund in `search_initializing`; allgemeine Writes und Worker
bleiben geschlossen. In `search_manual_intervention` öffnen allgemeine
PocketBase-, Admin- und Pluginwrites sowie nicht suchende Worker einschliesslich
Federation-Inbox und -Outbox. Search-Readiness, Search-/Token-/Proxyrouten und
alle engine-mutierenden Indexhooks oder Searchworker bleiben dagegen
fail-closed und führen keinen Engineaufruf aus. Die transaktionale
Epochen-/Fan-out-Buchhaltung jedes danach committed Fachwrites bleibt aktiv;
es wird lediglich kein Engineauftrag eingereicht. Ein abgebrochener Vergleich,
Enginefehler, unbekanntes oder gemischtes Profil, `recovering`, `cutover`, ein
innerhalb seiner deklarierten UID-Menge nur halb gebundener beziehungsweise
inkonsistenter Authority-Handoff oder Lockverlust darf diese Phase nicht
öffnen. Eine vollständig übergebene UID neben weiterhin offline verwalteten
UIDs ist dagegen ein gültiger Authority-Mix und verwendet die unten definierte
Mischtransaktion.
Unmittelbar nach der vollständig und stabil gebundenen Driftdiagnose setzt ein
einziger SQLite-Control-Commit bei weiterhin geschlossenem Ingress auf der
vollständigen kanonischen Menge gedrifteter, noch offline verwalteter UIDs
`verification_required_reason = document_drift`, sofern dort noch kein anderer
nichtleerer Grund steht; einen vorhandenen Wert erhält er bytegleich. Für jede
Zeile CAS-prüft der Commit die beim Vergleich stabilen Source-/Epoch-/
Quieszenzversuchswerte. Nur eine Zeile mit zuvor leerem Grund wird mutiert, erhöht dabei
`state_revision` und muss `state_revision <= 2^53-4` erfüllen; ein bereits
nichtleerer Grund ist selbst der durable Latch und wird ohne Revisionswrite
bytegleich geprüft. Jede Zeile gewinnt oder der ganze Commit rollt zurück;
einzelne vor einem Crash markierte UIDs sind verboten. Bei CAS-, Budget- oder
Commitfehler bleibt der Prozess
`search_initializing`. Die Ownergates werden erst nach diesem erfolgreichen
Commit für die mögliche Phasenöffnung ausgewertet; ein rotes Gate darf seine
Persistierung nicht verhindern oder rückgängig machen. Damit erzwingt auch ein
sofortiger Crash bei noch rotem STATE1-Gate und ein Neustart ohne `--full` oder
zwischenzeitlichen Fachwrite erneut den Vollvergleich für die gesamte
Driftmenge. Der Grund bleibt bis zum grünen Offline-Reindex/Vollvergleich
bestehen. Die Phase selbst ist rein prozesslokal und autorisiert keinen
parallel laufenden Reindex.

„Synchron“ bedeutet in diesem Vertrag: Settings-, Fingerprint-, Vergleichs-
und Recoveryphasen laufen innerhalb dieser einen Guard-Goroutine strikt
serialisiert und jede Engine-Task wird terminal abgewartet. Es bedeutet nicht,
dass der `OnServe`-Hook oder die HTTP-Serverloop bis zu ihrem Abschluss
blockiert. Eine zweite Ensure-Goroutine ist verboten.

`auto` verwendet ausschliesslich für UIDs in `O` den unten definierten
Epochen-Kurzpfad und repariert dort bei bekanntem Profil fehlende oder
abweichende Dokumente im Modus `recover`; UIDs in `S` bleiben dem STATE1-Owner
vorbehalten. Der
persistente Operationsmarker verhindert Readiness nach einem Crash mitten in
dieser Reparatur. Der `manual`-Ensure-/Vergleichslauf selbst mutiert weder
Engine noch Fach-, Vertrags-, Profil- oder Operationszustand; erst der oben
definierte App-only-Phasenwechsel autorisiert danach normale Fachwrites, aber
weiterhin keine Engineprojektion. Ein grüner Vollvergleich darf jedoch als einzige
**nach dem Vergleich** zulässige Control-Metadatenmutation, neben dem
gegebenenfalls vorher gesetzten `--full`-Latch, seine `verified_*`-/Attested-
Werte per CAS persistieren und den dadurch erfüllten Vollvergleichsgrund leeren; das ist eine
Attestierung beobachteter Gleichheit, keine Reparatur. Dadurch startet auch der
erste folgende Fachwrite von einem persistenten Applied-/Attested-Gleichstand.
Bei der oben eng definierten Dokumentdrift attestiert `manual` dagegen nichts;
seine einzigen zulässigen Control-Mutationen sind der gegebenenfalls bereits
vor dem Vergleich ausgeführte `--full`-Vorab-Latch und der unmittelbar nach
stabil gebundener Diagnose, vor der Ownergate-Entscheidung atomar über die
ganze Driftmenge ausgeführte Grund-Commit. Dieser setzt bei leerem Feld
`document_drift` durable oder erhält einen vorhandenen nichtleeren
Vollvergleichsgrund bytegleich. Für den dokumentierten
Composepfad protokolliert Diagnose/Status diese direkt ausführbare
Operatoranweisung; andere Installationsprofile müssen dieselben vier Schritte
mit ihrem bereits dokumentierten Binarypfad rendern:

```bash
docker compose stop web db
docker compose run --rm --no-deps --entrypoint /pocketbase db \
  search-reindex run --dir /pb_data --mode auto \
  --output /pb_data/search-reindex
docker compose run --rm --no-deps --entrypoint /pocketbase db \
  search-state1 reconcile --dir /pb_data --durable-only --until-ready
docker compose up -d db web
```

Der STATE1-Schritt ist nur dann ein geprüfter No-op, wenn an seinem
eingefrorenen Cutoff weder `S`- noch `C`-Rückstand oder sonstige Ownergate-
Arbeit existiert. Auch bei vollständig offline verwalteten Such-UIDs kann ein
authority-unabhängiges Security-/Federation-`C` zu drainieren sein. Im
Authority-Mix verarbeitet der Schritt zusätzlich `S`: Der Offline-Reindex
repariert nur `O`; der anschliessende gestoppte Drain bringt die während der
Manual-Phase durabel angefallenen `S`-/`C`-Changes auf grüne Ownergates. Erst
danach darf der normale Neustart erfolgen. Schlägt einer der beiden Schritte
fehl, bleiben DB und Web gestoppt.

In allen anderen ungeprüften oder unsicheren Zuständen bleibt der Prozess
liveness-only. Ein bekanntes PocketBase-ableitbares Legacyprofil bleibt
startfähig und wird nur durch den expliziten SRCH0-Produktionslauf semantisch
migriert. `lists-legacy-v0` ist für den neuen Serveprozess dagegen nur
wartungs- und rollbackfähig: Solange es aktiv ist, endet Startup mit
`legacy_list_migration_required`, weil der neue inkrementelle Pfad weder den
alten Origin-Dial fortsetzt noch seine Semantik vortäuscht. Nach einem
belegten Rückswap kann weiterhin das alte Release gestartet werden. Ein
unbekanntes oder gemischtes Profil wird in beiden Modi niemals automatisch
überschrieben.

### Profil- und Readinesszustand

Die Namensräume sind absichtlich getrennt:

| ID | Bedeutung |
| --- | --- |
| `offline-srch0-v1` | operatives Single-Writer-Verfahren aus STATE1 |
| `srch0_compat_v1` | stabile Runtimevertragsrevision für Compiler, ACL und DTO |
| `trails-legacy-srch0-v1` | konkretes SRCH0-Zielprofil des Trailindex |
| `lists-legacy-srch0-v1` | nicht exponiertes SRCH0-CI-/Staging-Zielprofil des Listenindex mit lokaler Föderationsaggregation |
| `sec-vis0-list-v1` | verpflichtendes produktives Superset des Listen-Zielprofils mit derselben lokalen Aggregation und den SEC-VIS-0-Sichtbarkeitsfeldern |

Die stabile Laufzeitkompatibilität heisst
`search_contract_revision: srch0_compat_v1`. Sie bezeichnet Requestcompiler,
ACL-/DTO-Grenze und das geschlossene ausgewählte Drei-Index-Profilset, nicht die Bytes eines bestimmten
DB- oder Webbuilds. Ein kompatibler Build deklariert diese Revision in seinem
Releaseartefakt. UI-only-, CVE- oder Rebuild-Releases mit derselben Revision
benötigen weder Reindex noch Reattest; eine inkompatible Änderung erhöht die
Revision.

Eine interne PocketBase-Base-Collection `search_index_state` besitzt pro
logischem Index genau einen Record:

```text
index_uid                    text, required, unique; trails | lists | actors
control_plane_authority      select, required; offline-srch0-v1 | state1
authority_head_key           text, optional, max 128
authority_pointer_version    number, optional, onlyInt, min 1, max 9007199254740991; irreversible Handoff-Untergrenze
search_contract_revision     text, required, max 64
profile_revision             text, required, max 64
settings_fingerprint         text, required, ^sha256:[0-9a-f]{64}$
applied_projection_epoch     number, required, onlyInt, min 0, max 9007199254740991
attested_projection_epoch    number, required, onlyInt, min 0, max 9007199254740991
attested_document_count      number, optional, onlyInt, min 0, max 9007199254740991
verified_projection_epoch    number, optional, onlyInt, min 0, max 9007199254740991
verified_source_fingerprint  text, optional, ^sha256:[0-9a-f]{64}$
verified_document_count      number, optional, onlyInt, min 0, max 9007199254740991
verified_at                  date, optional, UTC; nur Diagnose, Zeitpunkt der letzten tatsächlichen UID-Attestierungs-CAS
verification_required_reason select, optional; source_changed | task_failed | document_drift | source_restore | engine_restore | engine_admin | operator
operation_state              select, required; ready | recovering | cutover
operation_run_id             text, optional, ^[a-z2-7]{16}$
operation_target_contract_revision  text, optional, max 64
operation_target_profile_revision   text, optional, max 64
operation_target_settings_fingerprint text, optional, ^sha256:[0-9a-f]{64}$
operation_baseline_kind      select, optional; pocketbase_projection | opaque_engine_snapshot
operation_baseline_document_fingerprint text, optional, ^sha256:[0-9a-f]{64}$
operation_baseline_document_count number, optional, onlyInt, min 0, max 9007199254740991
state_revision               number, required, onlyInt, min 1, max 9007199254740991
transitioned_at              date, required, UTC
```

Daneben besitzt jeder konfigurierte private Engineendpoint genau einen internen
Base-Record `search_engine_task_guard`:

```text
scope_key                    text, required, unique, ^sha256:[0-9a-f]{64}$; Digest aus normalisiertem Endpoint und nicht geheimer Authoritykennung
attested_global_task_head    number, optional, onlyInt, min 0, max 9007199254740991
attested_task_log_empty      bool, required
observation_state            select, required; ready | invalid
observation_reason           select, optional; bootstrap_unattested | task_head_changed | task_log_rewound | task_history_incomplete | task_submission_unknown | task_failed | task_terminal_timeout | engine_scope_changed | observer_timeout | restore_or_admin
revision_token               text, required, ^[a-z2-7]{26}$
attested_at                  date, optional, UTC; nur Diagnose
transitioned_at              date, required, UTC; nur Diagnose
```

Im Zustand `ready` gilt exakt eine Darstellung: Entweder ist
`attested_task_log_empty = true` und der Head leer oder das Flag ist `false`
und der Head Pflicht. Im Zustand `invalid` dürfen beide Belegfelder leer
beziehungsweise `false` sein. `scope_key` bleibt
über normale DB-Prozessneustarts stabil, ist aber weder Engine-Boot-ID noch
Beweis gegen einen ausserhalb des Runbooks restaurierten Engine-Datenträger.
Endpoint-/Authoritywechsel, unterstützter Engine-Restore und Engine-Adminarbeit
setzen den Record vor erneutem Serve mit dem passenden `observation_reason`
auf `invalid`; `ready` verlangt einen leeren Grund. `revision_token` ist ein
opaker CAS-Token und kein fachlicher Zähler.

Beide Collections sind nicht per Produkt-API erreichbar. Die Migration legt sie für
neue und vorhandene Installationen gleich an und backfillt zunächst die
bekannte Legacyrevision mit `control_plane_authority = offline-srch0-v1`.
`authority_head_key` und `authority_pointer_version` sind in diesem Zustand
leer. Bestehende und neue Offlinezeilen beginnen mit
`applied_projection_epoch = 1`, `attested_projection_epoch = 0` und leerer
`attested_document_count` sowie `state_revision = 1`; damit erzwingt die
Schemamigration für PocketBase-ableitbare Profile genau einen Vollvergleich und
erfindet keine Attestierung. Bei `lists-legacy-v0` autorisiert diese Lücke
keinen solchen Vergleich, sondern ausschliesslich den expliziten opaken
Migrationspfad. Der Taskguard wird zugleich mit
`observation_state = invalid`, `observation_reason = bootstrap_unattested`, leerem Head und
`attested_task_log_empty = false` angelegt. Der dokumentierte
Erstinstallationslauf darf sie vor dem ersten Ingress nur im oben definierten
leeren Baselinefrei-Fall direkt auf das Zielprofil heben. Fehlt ein Record, wird
dies nicht als Fresh-Beweis interpretiert: `ensure` vergleicht PocketBase und
Engine und rekonstruiert höchstens eine bekannte PocketBase-ableitbare
Legacyrevision. Eine erkannte `lists-legacy-v0` bleibt unattestiert und verlangt
den Migrationslauf; die Zielrevision entsteht nur nach erfolgreichem
SRCH0-Zielvergleich beziehungsweise leerer Baselinefrei-Initialisierung.
Der Enginezustand bleibt massgeblich; ein Zustandsrecord kann einen fehlenden
Index oder Settingsdrift nie grün erklären.

`operation_state = ready` verlangt leere Operationsfelder einschliesslich der
drei `operation_baseline_*`-Felder. Bei einer
`lists-legacy-v0`-Migration sind diese drei Felder ab dem Multi-Row-CAS auf
`cutover` vollständig und binden den unten definierten opaken Enginesnapshot;
bei PocketBase-ableitbaren Baselines trägt `operation_baseline_kind` den Wert
`pocketbase_projection` und der committed Quellfingerprint bleibt autoritativ.
Vor dem ersten
Enginewrite eines direkten Recoveryaufbaus setzt eine PocketBase-CAS-Operation den Record
auf `recovering` und bindet Run sowie Zielrevisionen. Vor Veröffentlichung des
Vorwärtsswap-Intents setzt eine CAS ihn auf `cutover`. Erst der erfolgreiche
Vollvergleich des Recoveryaufbaus beziehungsweise die Aktiv-/Abbruchentscheidung
setzt ihn mit den passenden aktiven Revisionen zurück auf `ready`. Damit bleibt
ein Crash auch nach Verlust des Auditverzeichnisses sichtbar; der Marker ist
kein Generationenautomat und autorisiert selbst weder Swap noch Aktivierung.
Alle fünf API-Regeln beider internen Collections sind PocketBase-`null`. Aktivierung und
Recovery schreiben ausschliesslich serverintern und erhöhen bei jeder
Operationstransition `state_revision` per Compare-and-swap genau um eins; der
automatisch gepflegte `updated`-Zeitstempel ist kein CAS-Token.
Dasselbe gilt für jede Applied-, Attested-, Vollvergleichs- oder
Invalidierungsänderung: Jede betroffene Zeile erhöht `state_revision` in
demselben CAS genau um eins. Sei `M = 2^53-1`. Es gibt genau eine
Headroomregel: **Jeder neue Eintritt, der einen mehrschrittigen Offlinepfad
beginnt oder um einen weiteren Applied-Commit verlängert, verlangt vor seinem
ersten Effekt `state_revision <= M-3`.** Das betrifft insbesondere Fachwrites,
einen neuen Invalidierungs-, `document_drift`- oder `operator`-Latch sowie den
direkten Eintritt `ready -> recovering|cutover`. Bei einer Multi-UID-
Transaktion muss jede betroffene Zeile in `O` dieselbe Vorbedingung erfüllen;
andernfalls rollt die ganze Transaktion vor Fach-, State- oder Enginewirkung
mit `search_state_revision_exhausted` zurück.

Reine Fortsetzungs- und Terminal-CAS eines bereits so zugelassenen und durable
erkennbaren Pfads eröffnen keine neue Arbeit. Nach dem **jeweils letzten**
separat mit `state_revision <= M-3` zugelassenen Eintritts- oder Applied-Commit
benötigt die Kette bis zu einem sicheren Terminalzustand höchstens zwei weitere
UID-CAS; diese dürfen die reservierten Schritte bis `M` verbrauchen und prüfen
weiterhin erwartete Revision, Run, Authority und Epoche. Eine lebende
koaleszierte Kette darf beliebig viele Applied-Commits und damit insgesamt mehr
als drei Revisionsinkremente enthalten; jede Verlängerung ist ein neuer
Eintritt und prüft dieselbe Vorbedingung erneut. Eine künftige Erweiterung, die
nach dem jeweils letzten zugelassenen Commit mehr als zwei UID-CAS benötigt,
muss diese Invariante und ihre Schema-/Testrevision erhöhen. Der irreversible,
genau ein CAS umfassende STATE1-Handoff verlangt nur `state_revision < M` und
prüft dies vor Aufbau seiner neuen Generation sowie vor jedem Head-, State-
oder Engineeffekt; danach friert er die Offlinezeile ein. Wrap, Sättigung und ein neuer
mehrschrittiger Eintritt bei `M-2`, `M-1` oder `M` sind verboten.

Nach einem grünen Vollvergleich schreibt eine UID-CAS den erneut bestätigten
Quellfingerprint, die Dokumentzahl und den dabei stabilen
`applied_projection_epoch` in die vier `verified_*`-Felder, **wenn** mindestens
eines dieser Felder, Attested, Zähler oder Grund tatsächlich geändert werden
muss. Zugleich setzt sie `attested_projection_epoch =
applied_projection_epoch`, übernimmt die Zahl in `attested_document_count` und
leert `verification_required_reason`. Der Commit ist nur zulässig, wenn sich
der unmittelbar zuvor erneut berechnete Quellfingerprint nicht geändert hat.
Sind dagegen alle UID-Attestierungsfelder bereits exakt gültig und war nur der
globale Taskguard invalid oder gegenüber dem stabilen finalen Head abweichend, werden die unveränderten UID-Zeilen im selben
SQLite-Schnitt read-only per Revision geprüft; die Transaktion darf allein den
globalen Guard CASen. Sie verbraucht dann auch bei einer UID auf `M` keine
UID-Revision; insbesondere bleibt `verified_at` als Zeitpunkt der letzten
tatsächlichen UID-Attestierungs-CAS unverändert und erzwingt keinen
Diagnosewrite. Muss irgendeine UID tatsächlich mutiert werden und besitzt sie
keinen zulässigen Revisionsschritt, bleibt alles mit
`search_state_revision_exhausted` fail-closed. Die `verified_*`-Felder bleiben danach als
an `verified_projection_epoch` gebundener Beleg des **letzten
Vollvergleichs** erhalten; sie behaupten nach späteren Writes nicht, den
aktuellen Inhalt zu beschreiben, und sind kein Eingang des normalen
Epochen-Kurzpfads. Eine an STATE1 übergebene UID wird ausschliesslich über ihre
Ownergates geprüft; ihr eingefrorener Offline-Record wird weder vollverglichen
noch attestiert oder aufgrund eines abweichenden Enginezählers mutiert.

Nur ein vollständiger grüner Vergleich **aller** aktuell konfigurierten UIDs in
`O` darf im selben SQLite-Commit auch den `search_engine_task_guard` von seinem
erwarteten `revision_token` auf `ready` setzen, `observation_reason` leeren und die unmittelbar vor dem
Commit gelesene globale Taskbeobachtung `H2` als Head beziehungsweise als
explizit leeren Tasklog binden. Dazu müssen alle Offline-Delivery-Owner `idle`
und ihre Source-, Authority-, Applied-, Operations- und Revisionswerte
unverändert sein. Die gemeinsame Transaktion mutiert nur diejenigen UID-Zeilen,
deren Werte sich wie oben beschrieben ändern; ein reiner Guard-Reattest ist
damit ausdrücklich zulässig. Ein Teilvergleich kann die betroffenen UID-
Attestierungen erneuern, lässt einen bereits `invalid`en globalen Taskguard aber
`invalid`.

Die einzige read-only Vollvergleichsausnahme ist die unmittelbar vor dem
STATE1-Trail-Handoff liegende Precommit-Stufe. Sie verwendet denselben
höchstens dreimaligen `H0/H1`-Drain, vollständigen Vergleich aller UIDs in `O`
und `H2`-Fence, behauptet aber bewusst **keine** neue UID-Attestierung: Die an
den letzten Vollvergleich gebundenen `verified_*`-Felder dürfen nach normalen
inkrementellen Writes historisch sein und werden weder als Freshnessgate
verwendet noch fortgeschrieben. Zulässig ist dieser Zweig nur bei bereits
gleichen Applied-/Attested-Epochen, passenden attestierten Zählern und
Settings, leeren Gründen/Operationsfeldern, idle Delivery-Ownern sowie seit dem
statischen Handoff-Preflight gehaltenem Source-Lock und allen O-UID-Locks. Der
grüne tatsächliche Feld-/ID-/Digestvergleich darf dann ausschliesslich den
globalen Guard auf `H2` CASen; sein kanonischer All-O-Ergebnisdigest wird im
typisierten Handoff-Quieszenzobjekt als Digest der geschlossenen
`Srch0HandoffAllOfflineProjectionComparisonV1`-Preimage gebunden und ist nur Eingabe des unmittelbar
folgenden Pointer-CAS. Er wird kein Startup-/Kurzpfadbeleg. Jede tatsächliche
Projektionsabweichung, Epochen-/Zählerlücke oder nötige UID-Mutation verwirft
den Handoff und kehrt zum statischen Preflight beziehungsweise zuständigen
Recoverypfad zurück.

Jede Transaktion, deren Vorher-/Nachherzustand die vollständige
Adapterprojektion mindestens einer logischen UID ändern kann, durchläuft
authority-unabhängig denselben Projektionskoordinator. Aus demselben
konsistenten Vorher-/Nachherschnitt und der unten festgelegten versionierten
Abhängigkeitshülle berechnet er die vollständige kanonische UID-Menge `A`.
Vor dem Fachcommit ermittelt er eine vorläufige Offline-Menge, nimmt nur deren
prozesslokale UID-Locks lexikographisch und liest in der einen SQLite-
Transaktion Authority und erwartete `state_revision` jeder UID aus `A` erneut.
Er partitioniert deterministisch in `O = offline-srch0-v1` und `S = state1`.
Weicht diese Partition von der gelockten Vorabansicht ab, verliert der CAS, die
gesamte Transaktion rollt zurück und der Command partitioniert nach Freigabe
der Locks neu.

In **derselben SQLite-Transaktion wie der genau einmal ausgeführte
Fachcommit** erhöht der Koordinator ausschliesslich für jede UID in `O`
`applied_projection_epoch` und `state_revision` genau um eins. Aus demselben
Vorher-/Nachherschnitt erzeugt er je Offline-UID ein
`PostResponseProjectionEnvelopeV1` mit Applied-Epoche, vollständiger betroffener
Dokument-ID-Menge, Vorher-/Nachherprojektion und
`document_count_delta = |keys_after ∖ keys_before| - |keys_before ∖ keys_after|`.
Das Delta folgt der Projektionsmitgliedschaft, nicht der Zahl von Upsert-
Aufrufen: Create/Delete beziehungsweise ein Membershipwechsel ergeben
`+1/-1`, Update und reiner Fan-out `0`; Create→Delete darf zu `0` und keinem
Task, Delete→Create desselben Keys zu `0` und dem letzten Volldokument
koaleszieren. Für `S` schreibt der Commit stattdessen genau den STATE1-
Revisionsschnitt aus Abschnitt 5.1 des State-Machine-Vertrags: Commit, Change,
Dirty/Delivery und gegebenenfalls Fence/Tombstone für die auf `S` gemappten
Projektionsziele. Die `search_index_state`-Zeilen aus `S` bleiben dabei
bytegleich eingefroren.
Authority-unabhängig verpflichtete STATE1-Security-/Federation-Control-Changes
`C` bleiben zusätzlich nach jenem Vertrag anzulegen und können auch bei
`S = []` eine Katalogrevision benötigen; sie übernehmen dadurch keine Offline-
Suchprojektion.
Registryrevision, `A`, `O`, `S` und erwartete Authority-/State-Revisionen sind
kanonisch im Control-/Cause-Digest gebunden. Fachmutation, Offline-Epochen und
STATE1-Invalidierung committen ganz oder gar nicht; kein Ziel darf beiden oder
keinem Owner zugeordnet sein. Vor diesem Commit darf kein Engineauftrag
entstehen.

Der HTTP-/Commandpfad endet nach diesem Commit. Das committed Envelope wird
prozesslokal zunächst als nicht submitfähiges `awaiting_response` registriert.
Bei HTTP-Kommandos gibt ein zwingender `defer`-Finalizer es genau einmal frei,
nachdem der Responseversuch den Request-Lifecycle vollständig verlassen hat;
ob der Client die Antwort noch angenommen oder die Verbindung nach dem
durablen Commit getrennt hat, ändert die Projektionspflicht nicht. Bei internen
Kommandos geschieht dies entsprechend erst nach Rückkehr des erfolgreichen
Commit-Callbacks. Die atomare prozesslokale Transition
`awaiting_response -> collecting` verhindert eine Doppelfreigabe. Kein
Request wartet auf Engine-Submit, Task-Await, Engine-Stats oder eine
Attestierungs-CAS.

Jedes Envelope besitzt ab Fachcommit ein festes
`post_response_release_deadline = 5 s`. Hat bis dahin weder der Finalizer noch
der Delivery-Owner die Freigabe übernommen, schliesst der Watchdog zuerst
Search und projektionsrelevante Write-Admission, submitten darf er das noch
nicht nachweislich post-response liegende Envelope nicht. Danach führt er für
die gesamte Envelope-UID-Menge die bereits reservierte kombinierte
`task_failed|recovering`-CAS aus; ein verspäteter Finalizer sieht den terminalen
prozesslokalen Zustand und bleibt wirkungslos. CAS-Verlust oder Prozesscrash
lässt mindestens den durablen Applied-Vorsprung zurück. Ein verlorener
Finalizer oder Wake-up kann die laufende Instanz damit nicht unbegrenzt
stale-ready halten; der nächste Start vollvergleicht weiterhin. Alle heutigen
direkten Create-/Update-/Delete- und Fan-out-Aufrufe aus Trail-, Listen-,
Actor-, Share-, Like-, User-, Summit-, Plugin- und Federationhooks müssen
diese Grenze verwenden; ein nachgelagerter Einzelhook ist kein Ersatz.

Ausserhalb `search_manual_intervention` besitzt der Offlinekoordinator pro UID
genau einen prozesslokalen Delivery-Owner mit den Zuständen `idle`,
`awaiting_response`, `collecting`, `running[+dirty_suffix]` und
`attesting|failed`. Er sammelt
nach dem ersten Envelope mindestens 25 ms, höchstens 100 ms beziehungsweise bis
500 Dokumente oder 8 MiB kanonisches JSON erreicht sind, ordnet auch
vertauscht eingetroffene Response-Freigaben strikt nach Applied-Epoche und
koalesziert je Dokument-ID auf früheste Vorher- und letzte Nachherprojektion.
Unter dem kurzen UID-Lock fängt er nur Start-Epoche, Arbeitspräfix und
Zählerbasis ein; während Submit und Await hält er weder diesen Lock noch einen
DB-Write-Lock. So dürfen Folgewrites antworten und genau einen geordneten Dirty-
Suffix bilden. Disjunkte UIDs laufen parallel, pro UID sind höchstens ein
Engine-Batch und sein terminales Await gleichzeitig aktiv.

Eine neue Live-Kette darf nur aus
`applied_projection_epoch = attested_projection_epoch`, vorhandenem
`attested_document_count`, leerem Vollvergleichsgrund und `operation_state =
ready` sowie globalem Taskguard im Zustand `ready` starten. Solange
derselbe lückenlose prozesslokale Owner lebt, dürfen
weitere Applied-Commits die Kette verlängern; fehlt ein Envelope, ist die
Epochenfolge nicht lückenlos, ist der Owner nach Crash unbekannt oder liegt
bereits ein Fehler/Manualzustand vor, bleiben weitere Writes source-only und es
entsteht kein neuer Engineauftrag. Der Owner reicht alle nötigen Add-/Replace-/
Delete-Tasks in Epochenreihenfolge ein, wartet jeden terminal und verlangt
ausdrücklich `succeeded`; ein SDK-Aufruf ohne Transportfehler ist kein Erfolg.
Ein späterer Suffix darf einen offenen oder fehlgeschlagenen Präfix nie
überspringen.

Attested bleibt während der gesamten Live-Kette auf ihrer Start-Epoche `T0`.
Erst wenn alle Batches einschliesslich eines währenddessen entstandenen Dirty-
Suffixes erfolgreich sind, unter dem UID-Lock kein weiteres Envelope vorliegt
und Applied stabil auf `Efinal` steht, erfolgt **genau eine** CAS von `T0` auf
`Efinal`. Sie setzt
`attested_document_count = C0 + sum(document_count_delta)` und erhöht
`state_revision`; Unterlauf, Überlauf, Authority-/Statewechsel, ein nichtleerer
Grund oder ein CAS-Verlust verbieten die Attestierung. Der inkrementelle Pfad
ruft für diese Fortschreibung keine Engine-Stats ab. `numberOfDocuments` wird
nur im Startup-Kurzpfad, Vollvergleich und Recovery gegen den fortgeschriebenen
Zähler geprüft.

Jeder Engine-Socketwrite des Offlinekoordinators, eines STATE1-Workers/
Publishers oder eines im exklusiven Prozess laufenden Recovery-/Reindexbefehls
und jede direkt zurückgegebene Task-UID stehen in demselben prozesslokalen
endpointweiten Submitregister. Es unterscheidet mindestens `socket_pending`,
`expected_pending`, `succeeded`, `failed|canceled` und `unknown`. Nur die
direkte `202` desselben ununterbrochenen Prozesskontexts erzeugt
`expected_pending`; `enqueued|processing` bleibt erwartete, noch nicht
attestierbare Arbeit. Jede Transition auf `socket_pending` und der dadurch
autorisierte mögliche Socketwrite nimmt für Offline-, STATE1- sowie Recovery-/
Reindex-Submitter denselben endpointweiten Admission-Mutex; er bleibt bis zur
direkten Antwort oder zur festen `engine_submit_deadline = 5 s` gehalten. Ohne
direkte Antwort endet die Submission bei Deadline als
`task_submission_unknown`, schliesst beide Guards und attestiert keinen
Präfix, auch wenn noch kein neuer Head sichtbar war. Eine vor Deadline
eintreffende, versionsqualifiziert atomare synchrone Non-Acceptance-Antwort
wechselt sofort auf `failed|canceled` und `task_failed`; jede andere Non-202-
oder mehrdeutige Antwort sofort auf `unknown` und
`task_submission_unknown`. Kein Zweig wartet danach bis zur Deadline. Ein erst
danach sichtbarer Task kann den bereits invaliden Guard nicht wieder öffnen.
Wird ein neuer API-sichtbarer Head beobachtet, während
ein `socket_pending`-Record ihn möglicherweise erklärt, pausiert der Observer
neue Search-Admission sofort provisorisch. Nur eine innerhalb des noch
verbleibenden `task_head_detection_budget` eintreffende direkte `202`, deren
Task-UID die vollständige Pagination exakt zuordnet, darf den Record auf
`expected_pending` setzen und die provisorische Pause wieder öffnen. Bleibt die
Antwort aus, ist sie mehrdeutig oder liegt ein fremder Task dazwischen, wird
spätestens eine Sekunde nach der ersten API-Sichtbarkeit mit
`task_submission_unknown` dauerhaft invalidiert. Das Sockettimeout darf diese
Frist nicht verlängern.

Für jeden `expected_pending`-Record beginnt mit der direkten `202` ein
monotoner 120-s-Timer. `enqueued|processing` darf den lokalen Guard nur bis zum
`offline_task_terminal_timeout` grün halten. Ohne terminales Ergebnis schliesst
der Owner danach Search und projektionsrelevante Write-Admission, invalidiert
den Taskguard mit `task_terminal_timeout` und nimmt für betroffene O-UIDs den
reservierten Failure-/Recoverypfad; ein STATE1-Owner setzt entsprechend sein
Ownergate rot. `failed|canceled` verwendet `task_failed`. Ein erfolgreicher
Terminalrecord wird mindestens bis zu einer committed Guard-CAS auf einen ihn
einschliessenden Head behalten und erst danach GC-fähig.

Bei **jeder** erreichten global ruhigen oder prospektiv ruhigen Grenze darf der
endpointweite Owner den globalen Taskguard aktualisieren; Auslöser kann eine
UID-Attestierung, ein reiner STATE1-Ack oder das Ende des letzten offenen
Submits sein. Eine prospektiv ruhige Grenze besitzt genau einen letzten
Offline-Owner in `attesting`: Alle seine Tasks einschliesslich Dirty-Suffix sind
`succeeded`, Applied ist unter dem UID-Lock stabil und seine bereits validierte
Attestierungs-CAS würde ihn im selben SQLite-Commit epochengleich und grundfrei
machen. Alle anderen Offline-Owner sind bereits `idle` und ihre UIDs
epochengleich. Für die Guard-CAS zählt bei diesem einen Owner ausschliesslich
diese Transaktionspostimage; die lokale Transition `attesting -> idle` erfolgt
genau dann unmittelbar nach erfolgreichem Commit und vor Freigabe des
Admission-Mutex. Bei Commit-/CAS-Fehler werden weder Attestierung noch Guard
fortgeschrieben und `idle` wird nicht behauptet. Unter dem
endpointweiten Admission-Mutex liest er `Hq1`, paginiert vom persistenten oder
bereits prozesslokal validierten Head lückenlos bis `Hq1` und verlangt: Der
Guard war zuvor `ready`, jede seit seinem persistenten Head sichtbare relevante
Task gehört zum Submitregister dieses Prozesses und ist terminal `succeeded`,
alle UIDs in `O` sind in der aktuellen DB-Sicht oder der eben definierten
Transaktionspostimage epochengleich und grundfrei, alle Offline-Delivery-Owner
sind `idle` oder genau der eine post-commit-idle `attesting`-Owner, alle
beteiligten STATE1-Tasks besitzen ihren owner-spezifischen durablen Ack und kein
endpointweiter Submit ist offen. Danach liest er unter
demselben Mutex `Hq2`; nur `Hq2 = Hq1` im selben Scope erlaubt die CAS, und sie
bindet **genau `Hq1`**, nie einen ungeprüft neu gelesenen Head. Der Mutex bleibt
bis nach der CAS gehalten. Ein Task nach `Hq2` ist nicht eingesaugt, sondern
überholt den alten Checkpoint und wird vom Observer normal klassifiziert.

Fällt die prospektiv ruhige Grenze mit der letzten UID-Attestierung zusammen,
liegen beide Updates zwingend in derselben SQLite-Transaktion. Andernfalls darf genau eine
nachgelagerte globale CAS koalesziert werden. Ein reiner STATE1-S/C-Erfolg darf
damit eine eigenständige Guard-CAS auslösen, ohne eine O-UID zu mutieren. Ohne
diese Grenze bleibt der alte Head stehen, sodass der nächste Start
vollvergleicht. Ein bereits `invalid`er Taskguard wird niemals allein aus
inkrementellen Erfolgen wieder `ready`. Während einer lebenden Kette darf der
aktuelle Head den persistenten Checkpoint nur um die vollständig paginierte,
exakt im lokalen Submitregister geführte Taskfolge überholen. Ein
prozesslokaler `validated_runtime_task_head` rückt nach jeder solchen stabilen
Pagination vor, sodass der Observer bei langer, nie global ruhiger Schreiblast
nur den neuen Suffix liest. Nach einmal grünem Startup ist er die zulässige
Live-Readiness-Alternative zur persistenten Headgleichheit, aber nur solange
der aktuelle Head exakt diesem Runtimehead entspricht, der ganze Suffix
`expected_pending|succeeded` klassifiziert ist, jede Pending-Lease innerhalb
ihres Terminalbounds liegt, der persistente Guard weiter `ready` und die letzte
Observerbeobachtung höchstens 1 s alt ist. Er ist kein persistenter Startup-
oder Kurzpfadinput und geht beim Crash verloren. Das ist die bestehende
asynchrone Write-Sichtbarkeit, keine neue Startup-Attestierung. Ein anderer oder
neu gestarteter Prozess besitzt Register und Runtimehead nicht und darf den
alten Checkpoint deshalb nicht als Kurzpfad verwenden.

Ebenfalls ausserhalb
`search_manual_intervention` bedienen STATE1-Worker/Publisher ausschliesslich
die `S`-Projektionen und authority-unabhängigen `C`-Control-Changes aus dem
bereits durablen Revisionsschnitt und verwenden
ihre eigenen Leases; der Offlinekoordinator wartet sie nicht ab. Bei
fehlendem Envelope, Submitfehler, unbekannter Annahme, `failed`, `canceled`,
Countdeltafehler oder Attestierungs-CAS-Fehler bleibt
`attested_projection_epoch < applied_projection_epoch`. Soweit der Prozess die
Fehler-CAS noch erreicht, setzt sie in **demselben** Revisionsschritt den Grund
`task_failed`, `operation_state = recovering`, eine frische Run-ID sowie die
committed Zielrevisionen; dadurch benötigt die spätere Vergleichs-/
Reparaturentscheidung nur noch die reservierte terminale `ready`-CAS. Ein Crash
vor dieser Fehler-CAS lässt den Record `ready`, aber nach dem Applied-Commit
noch zwei reservierte Schritte für `ready -> recovering -> ready`. In beiden
Fällen wird der prozesslokale Search-Guard sofort unready. Ein Crash
an jeder Stelle nach dem Fachcommit und vor der erfolgreichen Attestierungs-CAS
lässt dieselbe Ungleichheit dauerhaft sichtbar; danach ist der Gleichstand
bereits committed. Es gilt immer
`0 <= attested_projection_epoch <= applied_projection_epoch <= 2^53-1`; ein
Überlauf endet vor dem Fachcommit mit `search_projection_epoch_exhausted`.

Der erfolgreiche Abschluss der koaleszierten Taskkette persistiert damit genau
das Vertrauen, das der laufende Prozess ohnehin besitzt. Auf einer normal
benutzten gesunden Instanz holen die Attestierungsepochen nach jeder ruhigen
Kettengrenze wieder auf; der nächste
Neustart ist nicht bloss nach Leerlauf oder Crashloop kurzpfadfähig. In
`search_manual_intervention` erhöht der Fachcommit weiterhin nur für `O` die
angewandten Epochen und setzt dort beim ersten solchen Write nur bei leerem
Grund `source_changed`; einen bestehenden nichtleeren Grund überschreibt er
nicht. Für `S` persistiert derselbe Commit den STATE1-Change-/Dirty-Schnitt,
lässt aber dessen Offline-Record bytegleich. Bis zum gestoppten Offline-
Reindex, dem danach verpflichtenden durable-only-STATE1-Drain und dem folgenden
Neustart bleiben sowohl Offline-Enginetasks als auch STATE1-Searchworker/
Publisher pausiert; mit dem App-only-Phasenübergang pausiert auch der
Taskobserver für die restliche Serveprozess-Lebenszeit, sodass danach kein
Engineaufruf entsteht. Die durable Arbeit geht nicht verloren. Die
Attestierung von `O` bleibt zurück.

Der normale Epochen-Kurzpfad ist genau dann zulässig, wenn
`operation_state = ready`, die aktiven Revisionen unterstützt sind,
`applied_projection_epoch = attested_projection_epoch`, eine
`attested_document_count` vorliegt, `verification_required_reason` leer ist,
der `search_engine_task_guard` `ready` ist, seine attestierte globale Head-/
Leerbeobachtung exakt dem aktuellen ungefilterten Engine-Taskhead entspricht
und die Engine für die logische UID den erwarteten Primärschlüssel,
Settings-Fingerprint sowie exakt diese `numberOfDocuments` meldet. Er berechnet
weder eine PocketBase-Vollprojektion noch lädt er Engine-Dokumente; sein Aufwand
ist in der Anzahl logischer UIDs konstant und umfasst genau einen globalen
Taskhead-Read. Fehlende oder ungleiche Epochen,
fehlender oder abweichender Zähler, ein nichtterminaler Operationsmarker, ein
persistierter Vollvergleichsgrund, ein `invalid`er beziehungsweise abweichender
Taskguard oder ein erfolgreich vorab gelatchtes
`search-index ensure --full` erzwingen den vollständigen Feld-/ID-/
Digestvergleich. Scheitert der atomare `--full`-Vorab-Latch wegen knapper
Revision, beginnt dieser explizite Vergleich nicht und entwertet die vorherige
Attestierung nicht. Vor **jedem** erzwungenen
Vollvergleich, also auch bei gleichem Epochenpaar mit Restore-/Admin-/Operator-
Grund, startet der Guard bei weiterhin geschlossenem Ingress
`task_quiescence_retry_v1`. Das ist ausdrücklich eine begrenzte optimistische
Validierung und keine von der Engine nicht angebotene Admission-
Linearisierung. Jeder Versuch bindet `(engine_scope_id, global_task_head)`;
eine nackte wiederverwendbare Tasknummer oder ein nur nach Index gefilterter
Head genügt nicht, weil insbesondere Swaptasks keine einzelne `indexUid`
tragen müssen.

Unter den Source-/UID-Locks liest der Guard `H0`, wartet alle bis `H0`
sichtbaren relevanten `enqueued|processing`-Tasks terminal und behandelt einen
unbekannten möglicherweise relevanten Tasktyp fail-closed. Danach liest er
`H1`. Weichen Scope oder Head ab, verwirft er den ganzen Versuch. Bei stabilem
`H1` führt er den Vollvergleich aus und liest unmittelbar vor der
Attestierungs-CAS `H2`; nur `H2 = H1` im selben Scope sowie unveränderte Source-
Fingerprint-, Authority-, Applied-, Operations- und Revisionswerte erlauben
die gemeinsame CAS der UID-Attestierungen und des globalen Taskguards. Jede
Abweichung verwirft sämtliche Vergleichsergebnisse und startet
mit neuem Head. Es gibt höchstens drei Versuche mit Backoff 50 ms, dann 100 ms
plus deterministisch auf 25 ms begrenztem Jitter; sie liegen vollständig im
jeweiligen 60-/180-Sekunden-Startupbudget. Danach endet der Lauf mit
`search_task_quiescence_failed`, ohne Engine- oder Attestierungsmutation und
ohne den durablen Latch beziehungsweise die Epochenlücke zu leeren. Der nächste
explizite oder automatische Ensure-Lauf darf denselben begrenzten Ablauf erneut
versuchen. Einen permanenten versionsbezogenen Unsupported-Zustand gibt es
nicht.

Auch erfolgreiche Altaufträge gelten nicht als Attestierung, weil ihre
Epochzuordnung nach einem Crash fehlen kann; erst der anschliessende
Vollvergleich ist autoritativ. Noch vor `H0` startet in der geschlossenen
Startup-Phase ein endpointweiter Runtime-Observer; nach dem
Attestierungscommit liest er vor der ersten Search-Freigabe den ungefilterten
globalen Head erneut. Danach startet er auf festen 250-ms-Slots höchstens einen
Head-Read je Slot mit 500 ms Requestdeadline. Ein noch laufender Vorgänger darf
mit genau einem Nachfolger überlappen. Solange Poll `n` weder abgeschlossen
noch transportseitig geschlossen ist, fällt der Startslot von `n+2` aus; eine
ausgelöste Cancellation allein entfernt den Request nicht aus der In-flight-
Zählung. Pollsequenz und beobachteter Head machen jedes später eintreffende
ältere Ergebnis einschliesslich seines Timeout- oder Transportfehlerausgangs
wirkungslos, sobald ein neuerer Poll bereits gültig angewandt wurde. Damit sind
höchstens vier Head-Requests pro Sekunde und auch am exakten 500-ms-Rand
höchstens zwei gleichzeitig offen. Ein
ausgefallener oder bei 500 ms abgebrochener Poll, Scope-/Endpointwechsel, ein Head-Rücksprung sowie jeder neue Head, der
nicht durch vollständige ungefilterte Pagination seit dem attestierten Head
lückenlos einem `expected_pending`- oder `succeeded`-Record dieses Prozesses
zugeordnet werden kann, ist ein Invalidierungskandidat. Vor einem lokalen Close
aufgrund fehlender Registerdaten liest der Observer den persistenten Guard samt
`revision_token` erneut. Liegt ein neuerer `ready`-Checkpoint `Hc` im selben
Scope vor, verwirft er das fehlende Altpräfix: Deckt `Hc` den beobachteten Head
vollständig ab, ist der Poll stale und wird verworfen; liegt `Hc` echt zwischen
seiner alten Basis und dem beobachteten Head, startet er ohne Zurücksetzen des
ursprünglichen 1-s-Detektionsbudgets eine frische Pagination nur für den Suffix
`(Hc, current_head]`. Erst deren fehlende oder unbekannte Registerzuordnung darf
invalidieren. So kann eine Guard-CAS samt Präfix-Register-GC auch bei einem nach
`Hc` registrierten legitimen Task keinen falschen Alarm auslösen. Andernfalls
schliesst er **zuerst**
den prozesslokalen Searchguard und pausiert neue Engine-Submits sowie
projektionsrelevante Write-Admission. Eine nicht mehr lesbare Taskhistorie ist
dabei ebenso unbekannt wie ein unbekannter Tasktyp. Anschliessend setzt eine
CAS den globalen Taskguard auf `invalid` und bindet genau den diagnostisch
passenden `observation_reason`;
ein CAS-Konflikt wird bis zu einem nachweislich `invalid`en oder jüngeren
gleichwertigen Stand neu gelesen. Search bleibt geschlossen und ein neuer
begrenzter Quieszenz-/Vollvergleichslauf über alle UIDs in `O` ist nötig.
Ein exakt zugeordneter `expected_pending`-Task hält Observerlease und lokalen
Searchguard auch über mehrere Polls, aber nie länger als die gemeinsamen 120 s
grün; derselbe oder ein strengerer owner-spezifischer Bound gilt ausdrücklich
auch für alle im Register geführten STATE1-Worker-/Publishertasks. Er verschiebt
den persistenten Head noch nicht. `failed|canceled`, Submit-/Terminaltimeout
oder `unknown` invalidieren mit dem jeweils oben festgelegten Grund wie ein
fremder Task. Eigene direkt attestierte Tasks verschieben den persistenten Head erst an der
oben definierten global ruhigen Grenze. Nur eine Headbewegung löst die
zusätzliche, ab dem `validated_runtime_task_head` begrenzte Taskpagination aus.
Der Observer erzeugt keinen Dokument- oder Stats-Read.

Jeder Search-/Token-/Proxy-/Hilfsrequest erfasst vor seinem Enginezugriff die
lokale Guardgeneration und verlangt vor Versand seiner Antwort dieselbe noch
offene Generation sowie eine höchstens 1 s alte erfolgreiche
Observerbeobachtung. Eine inzwischen erkannte Invalidierung verwirft deshalb
auch den laufenden Response; diese Doppelprüfung ändert nichts am ausdrücklich
verbleibenden Fenster **vor** der Observererkennung.

`task_quiescence_retry_v1` und dieser Observer sind Detect-and-Retry, keine
Admission-Linearisierung. SRCH0 garantiert: Jede vor `H2` oder vor der
Freigabe beobachtete Änderung verwirft den Versuch; jede beim nächsten Start
sichtbare Änderung verhindert den Kurzpfad; ein im laufenden Prozess sichtbarer
Head, der nicht innerhalb des ursprünglichen
`task_head_detection_budget = 1 s` vollständig als eigener
`expected_pending|succeeded`-Suffix klassifiziert wird, schliesst Search samt
durablem Reverify-Latch. Ein rechtzeitig direkt-`202`-klassifizierter eigener
Task darf dagegen bis zu seinem 120-s-Terminalbound observergrün bleiben. Ohne
per-Response Enginefence garantiert
SRCH0 ausdrücklich **nicht**, dass zwischen einer Engine-Registration und der
Observererkennung null Searchantworten ausgeliefert werden. Ein Anspruch auf
dieses Nullfenster müsste alle Searchantworten über einen eigenen
Head-pre/Engine-query/Head-post-Fence führen und ist nicht Teil des engen
Offlineprofils. Migration, Aktiv-/Abbruchprüfung sowie die CI-/Staging-
Releasematrix verwenden immer den Vollvergleich.

Dieser Kurzpfad vertraut ausdrücklich auf die SRCH0-Grenze: Meilisearch ist
privat, der DB-Prozess ist der einzige normale Enginewriter, alle
projektionsrelevanten Quellpfade verwenden den Koordinator und jeder bekannte
Taskfehler invalidiert den Guard. Eine direkte PocketBase-Rückspielung oder
Engine-Adminmutation, die Epochen, Settings und Dokumentzahl passend erscheinen
lässt, kann er nicht erkennen. Deshalb muss der unterstützte Quellrestore-,
Engine-Restore- oder Adminpfad vor dem nächsten Start mit
`search-index invalidate --reason source_restore|engine_restore|engine_admin`
den dauerhaften Vollvergleichsgrund setzen. Ein Austausch unter Umgehung dieses
Runbooks besitzt keine Integritätszusage. Nach Restore, manueller Enginearbeit
oder Verdacht auf Drift ist `ensure --full` zwingend. Der Vollvergleich bleibt
Reparatur- und Auditautorität; der Epochen-Kurzpfad ist der normale
Neustartpfad einer korrekt fortgeschriebenen Instanz.

`control_plane_authority` verhindert, dass dieser lokale Guard und der spätere
STATE1-Publisher dieselbe logische UID mutieren. Das Offline-Authority-Handoff-
Set ist im Trail-only-Modell von STATE1 V1 entweder leer oder exakt
`["trails"]`; jede andere nichtleere Menge endet im Preflight mit
`state1_handoff_uid_unsupported`. Für den übernommenen Trailrecord setzt dessen erster querybarer
`search_generation_head`-Pointer-CAS in **derselben** PocketBase-Transaktion
deren Authority auf `state1` und bindet die neue `pointer_version` in
`authority_pointer_version` sowie den kanonischen `(contract, entity_kind)`-
Schlüssel des Heads in `authority_head_key`, für den V1-Trailhead beispielsweise
`trail-search.v1/trail`. Beide Authority-Bindungen sind bei `state1` Pflicht und
müssen auf denselben tatsächlichen Head zeigen. Die eingefrorene Pointerversion
belegt den Handoffzeitpunkt; spätere STATE1-Swaps dürfen den aktuellen Head nur
auf eine grössere Version bewegen. Readiness verlangt deshalb denselben Head-
Key und `current.pointer_version >= authority_pointer_version`, nicht dauerhafte
Gleichheit. Jede
übernommene Offlinezeile muss dabei
`operation_state = ready`, leere Operationsfelder,
`verification_required_reason` leer sowie
`applied_projection_epoch = attested_projection_epoch` besitzen; der globale
Taskguard muss `ready` sein und sein Head dem unmittelbar gelesenen Head
entsprechen. Der
prozesslokale Offline-Delivery-Owner muss `idle` sein; unmittelbar vor dem CAS
muss ein `task_quiescence_retry_v1`-Versuch samt erneut stabilem Scope/Head
gelingen. Zusätzlich bindet das neue STATE1-Membermanifest für jede übergebene
UID eine global eindeutige physische ID, die von allen bisherigen Offline-IDs
dieser UID disjunkt ist. Der alte Offlineindex ist keine `from_generation`, wird
nach dem Commit nie durch den neuen Head geroutet und bleibt bis zum sichtbaren
Taskdrain beziehungsweise GC quarantänisiert. Damit kann eine erst verspätet
registrierte Offline-Alttask nur die abgelöste Ressource treffen; diese
physische Isolation ist die Sicherheitsgrenze, nicht eine behauptete
No-late-Admission-Eigenschaft des Quieszenzversuchs. Der Commit gilt für
die deklarierte Einermenge ganz oder gar nicht; ein innerhalb dieser Menge nur halb
gebundener Zustand ist fail-closed. Die Handoffmenge muss nach erfülltem
`srch0_pair_handoff_ready_v1` nicht gegen die Projektions-Abhängigkeitshülle
abgeschlossen sein: `actors` und `lists` bleiben unter dem
Offlineprofil, und die authority-partitionierte Mischtransaktion routet
eine kreuzende Fachmutation atomar an beide Owner, aber dieselbe UID besitzt nie
zwei Writer. Vor terminaler Paaraktivierung ist der Trail-Handoff durch
`search_pair_migration_required` verboten; danach bleiben fehlender
Terminalbeleg beziehungsweise nicht bereinigte Kandidaten durch
`search_pair_activation_evidence_required` beziehungsweise
`search_pair_cleanup_required` gesperrt. Der Handoff nimmt Source-Lock und alle
aktuellen Offline-UID-Locks in kanonischer Reihenfolge bis zum Commit, während
der Authority-CAS ausschliesslich den Trailrecord prüft und erhöht. Gewinnt eine Fachmutation zuerst, verliert dieser CAS und muss
mit neuem Stand geplant werden; gewinnt der Handoff zuerst, verliert die alte
Fachcommand-Partition, rollt vollständig zurück und routet beim Retry die
übergebene UID nach STATE1.

Nach diesem Handoff sind die betreffenden `search_index_state`-Records
eingefrorene Kompatibilitätsbelege und keine Readiness- oder Routingautorität
mehr. `search-reindex`, `resume`, `abort` und `search-index ensure` brechen für
diese UIDs vor jeder expliziten Offline-State- oder Enginemutation stabil mit
`search_control_plane_owned_by_state1` ab. Ein impliziter gemischter `ensure`-/
Auto-Reindexlauf validiert beziehungsweise überspringt STATE1-UIDs read-only
und bearbeitet nur seine Offline-Zielmenge. Nur STATE1-Head, Generation,
Snapshot und Publisherzustand entscheiden dann Serving und Recovery; ein
Rollback ist eine neue STATE1-Generation, niemals eine stille Rückgabe an das
Offlineprofil. Eine spätere Migration darf die eingefrorenen Records entfernen,
aber nicht reaktivieren.

Der Aktivierungscommit ist die PocketBase-Transaktion, welche nach grüner
Produktionsprüfung die betreffenden `search_index_state`-Records auf die neue
Vertrags- und Profilrevision erhöht. Danach publiziert
`resume --activate` den Auditbeleg `active.json`. Ein Crash nach dem DB-Commit,
aber vor der Datei liefert nur diese aus dem committed Zustand nach; er erlaubt
keinen Rückswap mehr. Das Runbook öffnet Ingress trotzdem erst nach Beleg und
grüner Readiness. Normale Dokumentmutationen verändern die Vertragsrevision
nicht.

`GET /health` bleibt Prozess-Liveness und antwortet auch während
`search_initializing`. DB `GET /health/search` und Web
`GET /api/v1/health/search` liefern den schlanken `SearchReadinessV1`-Typ aus
dem Trail-Suchvertrag. `ready` verlangt Engineerreichbarkeit, vorhandene
logische Indizes, passende Primärschlüssel/Settings, die erwarteten
Revisionen und Unterstützung durch den laufenden Build. Für eine UID unter
`offline-srch0-v1` verlangt es zusätzlich `operation_state = ready` und den
grünen Vollvergleich oder Kurzpfad dieses Prozessstarts; für eine bereits
übergebene UID gelten ausschliesslich die STATE1-Pointer-/Publishergates. Das
Ergebnis wird prozesslokal gecacht und ist kein neues Aktivierungsartefakt.
Auditdateien, Run-ID, ein Quellfingerprint aus einer externen Datei und der
Digest eines bestimmten Builds sind keine Readinessinputs. Die internen
Applied-/Attested-Epochen samt `attested_document_count` sind lediglich die
oben begrenzte Offline-Kurzpfadattestierung; die `verified_*`-Felder belegen nur
den letzten Vollvergleich.

### Artefakte und Schreibregel

Jeder Lauf verwendet eine zufällige kleingeschriebene Run-ID aus 16
Base32-Zeichen und genau ein Verzeichnis `<output>/<run-id>/`. Fünf Klassen
dauerhafter Kontrollrecords steuern den Produktionslauf:

```text
plan.json
index-verification.json
swap-intent.json | reverse-swap-intent.json
swap-submit/<forward|reverse>/<attempt>.intent.json
swap-submit/<forward|reverse>/<attempt>.admission.json
swap-submit/<forward|reverse>/<attempt>.decision.json
active.json | abort.json
```

`fixture-verification.json` aus CI/Staging,
`production-verification.json` aus dem Wartungsfenster und das nur bei aktiver
`lists-legacy-v0` verpflichtende `opaque-legacy-baseline.json` sind geschlossene
Validierungseingaben, aber keine zusätzlichen Zustände des Reindexautomaten.
Das Fixture-Artefakt bindet zusätzlich die unveränderlichen OCI-/Release-
Artefakt-IDs beziehungsweise SHA-256-Digests von DB und Web sowie den Digest
jedes getesteten Profilmanifests. Der Produktionsbericht liest dieselben
Deklarationen aus den installierten Artefakten und verlangt Gleichheit. Diese
Buildbindung gilt ausschliesslich für die einmalige Aktivierung; sie ist weder
Readinessinput noch Reattestpflicht für spätere kompatible Builds mit derselben
Vertragsrevision. `active.json` bindet Fixture- und grünen Produktionsbericht.
`abort.json` bindet dagegen mit expliziter Presence-Diskriminierung nur die bis
zur Abbruchentscheidung tatsächlich vorhandenen Validierungsberichte und
erfindet insbesondere keinen Produktionsbericht. Ein optionales `tasks.ndjson` dient
ausschliesslich Diagnose und darf nie eine Entscheidung autorisieren. Jeder
Kontrollrecord und Validierungsbericht hat
`additionalProperties: false`, wird nach RFC 8785 kanonisiert und als
vollständige temporäre Datei geschrieben. Erst Datei-`fsync`, atomarer
No-Replace-Rename und Verzeichnis-`fsync` veröffentlichen es. Es gibt keine
Signaturen, Zertifikatsketten, Clock-Snapshots, Credential-Ledger oder
per-Batch-Intents. Swap-Submit-Attempts sind die ausdrücklich notwendige
Ausnahme für die nicht idempotente Engineoperation.

Alle rungebundenen Kontrollrecords und Validierungsberichte enthalten
`contract`, `run_id`, `mode`,
`search_contract_revision`, `profile_revisions`, `created_at` und die Digests
ihrer direkten normativen Vorgänger. Zusätzlich gilt:

| Artefakt | Pflichtinhalt |
| --- | --- |
| `plan.json` | PocketBase-Zielprojektionsfingerprint je betroffener logischer UID, Engineversion, betroffene logische und Kandidaten-UIDs, kanonische `A/O/S`-Partition samt erwarteten Authority-/`state_revision`-Werten, beobachteter STATE1-Katalogcutoff und dessen Pending/Ready-Status, Baseline-Settings/-Dokumentzähler und bei `lists-legacy-v0` zusätzlich Digest des `opaque_legacy_baseline_v1`-Records statt eines erfundenen PocketBase-Baselinefingerprints, Batchgrenzen sowie im Migrationsmodus `available_bytes`, Engine-`databaseSize`, Messpfad, Messzeit und daraus berechnete lokale Kapazitätsgrenze; ein Auto-Resume darf ausschliesslich die gebundene O-Menge mutieren und meldet S/C getrennt über `state1_drain_required` plus `state1_cutoff` |
| `opaque-legacy-baseline.json` | Profilrevision, logische und aktuelle physische UID, Primärschlüssel, vollständiger Settingsfingerprint, Dokumentzahl sowie Fingerprint der nach ID sortierten vollständigen tatsächlichen Engine-Dokumente einschliesslich jeder gespeicherten Feldbezeichnung; ausser IDs, Feldnamensmengen und Dokumentdigests keine Dokumentwerte, kein PocketBase-Quellfingerprint und kein Originbeleg |
| `fixture-verification.json` | CI-/Staging-Matrixdigest, DB-/Web-Release-Artefaktdigests, Contract- und Profilmanifestdigests sowie alle synthetischen ACL-/Proxy-/DTO-Ergebnisse; keine Produktionsdaten |
| `index-verification.json` | Plan-Digest und eine kanonisch sortierte geschlossene Indexliste mit erneut bestätigtem Quellfingerprint je betroffener logischer UID, geprüfter physischer UID, vollständiger Settings-, ID-, Feld-, Dokumentzahl- und Dokumentdigestprüfung sowie terminalen Buildtasks; im Migrationsmodus sind die physischen UIDs die Kandidaten, im Recoverymodus die logischen UIDs |
| Swap-Intent | Richtung, Verifikationsdigest, kanonisch vollständige UID-Paarmenge `{lists, trails}`, erwartete Vor-/Nachfingerprints und Digest der kanonischen Engine-Requestpayload; genau einmal je Richtung und Lauf |
| Swap-Submit-Attempt | lückenlose Safe-Integer-Attemptnummer ab `1`, Richtung, Swap-Intent-Digest, bei `n > 1` Digest der vorigen terminalen Decision, Requestdigest, erwartete Pre-Abbildung/-fingerprints und unmittelbar vor genau einem autorisierten HTTP-POST beobachtetes `pre_submit_observation = (engine_scope_id, global_task_head)`; der Record ist vor jedem möglichen Socketwrite No-Replace durable und behauptet keine Admission-Barriere |
| Swap-Submit-Admission | Attempt-Digest, exakte `(engine_scope_id, task_uid)`, Tasktyp, kanonisch vollständige UID-Paarmenge und Requestdigest; genau einmal No-Replace ausschliesslich aus der direkten `202`-Antwort in demselben ununterbrochenen exklusiven SRCH0-Prozesskontext, niemals durch spätere Tasklistenadoption |
| Swap-Submit-Decision | Attempt-Digest und genau einer der terminalen Zustände `not_admitted`, `applied_verified`, `failed_no_effect` oder `ambiguous`; `not_admitted` verbietet einen Admission-Digest und verlangt entweder den lokalen Beleg `socket_write_not_started` oder eine typisierte synchrone Non-Acceptance-Antwort, nie Tasklisten-Abwesenheit. `failed_no_effect` verlangt Admission-Digest, eine in demselben ununterbrochenen exklusiven SRCH0-Prozesskontext beobachtete terminale Task und atomaren Nichteffektbeleg. `applied_verified` verlangt exklusiv entweder Admission-Digest plus dort terminal `succeeded` beobachtete exakte Swap-Task **oder** `swap_effect_proof_v1`; nach Restart ist nur der Effect-Proof-Zweig zulässig. In beiden Zweigen bindet die Decision den vollständigen stabilen Post-Abbildungs-, Settings- und Dokumentfingerprintvergleich; bei Reverse ist sie damit zugleich der durable Baseline-Vollvergleich. `ambiguous` bindet den tatsächlich vorhandenen oder fehlenden Admissionzustand. Jede Variante ist genau einmal unveränderlich |
| Terminalbeleg | Ergebnis und kanonische Abbildung jeder UID der vollständigen Migrationsmenge auf ihre neue `search_index_state.state_revision`; ein Pre-Swap-`abort` bindet Kandidatenverifikation, exakte Baseline-Pre-Abbildung/-Settings/-Fingerprints und die lückenlose Write-ahead-Kette, deren Fehlen von Swap-Intent und -Attempt beweist, dass kein Socketwrite autorisiert war. `active` bindet Vorwärtsintent, die vollständige gemäss jeder Decision schema-validierte Forward-Kette mit genau den tatsächlich vorhandenen Admissions, terminal `applied_verified`, Endabbildung und Produktionsprüfbericht. Ein Post-Swap-`abort` bindet Vorwärts- und Reverse-Intent, beide ebenso schema-validierten Ketten mit je terminal `applied_verified`, den als Reverse-Decision durablen Baseline-Vollvergleich, Endabbildung und Baseline-State-CAS; bei `recover` stattdessen der Digest des erneuten Index-Vollvergleichs |

Task-UIDs sind nur innerhalb des ununterbrochenen exklusiven SRCH0-
Prozesskontexts sinnvoll, der die direkte `202`-Antwort erhalten hat; die
Submit-HTTP-Anfrage selbst ist vor dem terminalen Polling bereits beendet.
Produktions- und Fixturelauf erhalten getrennte Verifikationsdateien und je
eine runlokale `engine_scope_id`; eine direkt zurückgegebene Taskidentität ist
das Paar `(engine_scope_id, task_uid)`. Diese Kennung ist weder Engine-Boot-ID
noch Tasklog-Kontinuitätsbeweis. Nach Prozess-, Endpoint- oder Scopeverlust darf
eine Admission deshalb nicht taskbasiert weiterverfolgt und eine bloss ähnlich
sichtbare Task nie adoptiert werden. Der Attempt bleibt
`swap_submission_unknown`, ausser der nachfolgende exakte Post-State erfüllt
`swap_effect_proof_v1`. Tasklisten-Abwesenheit beweist weder `not_admitted` noch
autorisiert sie einen Folgeattempt.

`swap_effect_proof_v1` darf einen bereits eingetretenen Swap
ohne verwertbaren taskbasierten Admission-/Kontinuitätsbeleg bestätigen; ein
bereits durabler Admissionrecord bleibt dabei gebunden, wird aber nicht als
Taskwirkungsbeweis verwendet. Nach exklusiver Writerübernahme führt der Adapter
`task_quiescence_retry_v1` aus; unbekannte relevante Tasktypen oder drei
instabile Versuche bleiben fail-closed. Dass höchstens eine Annahme möglich
war, folgt nicht aus einem lückenlosen Engine-Tasklog, sondern aus genau einem
in der lückenlosen Auditkette autorisierten Socketwrite seit der letzten
terminalen Decision. Mehr als ein möglicherweise gesendeter, nicht durch
`not_admitted|failed_no_effect` abgeschlossener Attempt ergibt
`swap_audit_conflict` und verbietet den Effect-Proof.
Bei während des anschliessenden Vollvergleichs unverändertem Scope/Head müssen UID-Abbildung,
Settings sowie beide vollständigen Dokumentfingerprints exakt dem Post-State
des Intent entsprechen. Nur dann darf eine terminale Decision
`applied_verified` ohne Taskidentitätsbeleg entstehen. Dieser
Effect-Proof autorisiert niemals einen weiteren Submit; Pre- oder gemischte
Abbildung bleibt nach Sessionverlust `swap_submission_unknown`.

`plan.json` wird genau einmal vor dem ersten Enginewrite veröffentlicht. Ein
vorhandener abweichender Plan ist `run_plan_conflict`; ein bytegleicher Plan
darf fortgesetzt werden. Im Modus `migrate` erfolgt die Kapazitätsmessung einmal
unmittelbar vor dem Build auf dem tatsächlichen Meilisearch-Dateisystem. Sei
`D = GET /stats.databaseSize`; dann ist
`required_free_bytes = D + max(ceil(D * 0.20), 1 GiB)`. Alle Rechnungen sind
overflow-geprüfte unsigned Integerrechnungen. Fehlt `--available-bytes`, ist
der Wert kein kanonischer Dezimal-`uint64`, weicht der gebundene Pfad vom
Compose-Preflight ab oder gilt `available_bytes < required_free_bytes`, endet
der Lauf vor Create/Settings mit `insufficient_search_capacity`. Damit ist der
Negativfall deterministisch testbar. `verify` braucht keinen Kopierplatz;
`recover` kann bei fehlendem abgeleitetem Bestand ohne Shadowcopy arbeiten,
bleibt bei einem Engine-Kapazitätsfehler aber `not_ready` und behauptet keine
erfolgreiche Reparatur. Vor dem Swap wird kein zweites Vollkopie-Budget
verlangt, sondern nur bestätigt, dass Engine, Lock, Quellfingerprint und
Kandidat unverändert sind.

### Deterministischer Aufbau und Vollvergleich

`lists-legacy-v0` durchläuft **keinen** PocketBase-Vollprojektor. Vor
Planpublikation erfasst `migrate` unter exklusivem Writerlock und stabilem
`task_quiescence_retry_v1` zunächst Primärschlüssel und vollständige Settings
des aktiven Listenindex. Danach liest der lokale Adapter mit begrenztem
`POST /documents/fetch`-Paging und ausschliesslich für diesen opaken Snapshot
mit **ausgelassenem** `fields`-Parameter alle tatsächlichen Dokumentfelder,
sortiert die Dokumente nach `id` und bildet
`SHA256(RFC8785([doc1,...,docN]))` über alle gespeicherten Felder. Der
`opaque_legacy_baseline_v1`-Record bindet Zahl, Gesamtfingerprint und für jedes
Dokument nur `id`, sortierte Feldnamensmenge sowie Dokumentdigest; Feldwerte
werden nach der Digestbildung verworfen. Dieser eng begrenzte lokale
Maintenance-Read ist die einzige Ausnahme von der sonst verpflichtenden
expliziten Profilfeldliste. `fields: ["*"]` bleibt auch hier verboten, weil es
im POST-Body ein wörtlicher Selektor und keine Wildcard ist. Der Snapshot führt
keinen Origin-Dial aus.

Plan und `cutover`-Multi-Row-CAS binden den Recorddigest sowie dieselben
`operation_baseline_*`-Werte. Unmittelbar vor dem Vorwärtsswap muss ein zweiter
vollständiger Engineread exakt denselben Primärschlüssel-, Settings-, ID-/Feld-,
Zahl- und Dokumentfingerprint liefern. Nach dem Swap bleibt der unveränderte
Altindex unter der getauschten runlokalen UID die physische Rollbackbaseline.
Vor einem Reverse-Swap wird diese physische UID erneut gegen den opaken Record
geprüft; nach dem Swap muss dieselbe Prüfung auf der logischen UID grün sein.
Die Reverse-Decision verwendet diesen Engine-gegen-Snapshot-Beleg statt eines
PocketBase-Vergleichs. Fehlt oder driftet die physische Baseline, ist weder
Reverse noch Rekonstruktion erlaubt: Der Lauf endet
`legacy_list_baseline_unrecoverable` und verlangt ein gegen denselben Record
passendes Enginebackup. Der Snapshot behauptet nie PocketBase-Gleichheit und
ist ausserhalb dieses gestoppten Migrations-/Rückswapfensters kein
Readinessinput.

Jeder PocketBase-ableitbare Logical-Index-Adapter definiert seinen eigenen Quellfingerprint exakt als
`SHA256(RFC8785([doc1,...,docN]))`, wobei das kanonische JSON-Array alle
vollständigen Adapterdokumente aufsteigend nach ihrem Primärschlüssel enthält;
es gibt weder rohe Konkatenation noch dokumentweise Teilhashes. Gespeichert
wird `sha256:` plus 64 kleingeschriebene Hexzeichen. Für `trails`
ist dies die Folge aller
`FullTrailDocument`-Objekte einschliesslich aufgelöster Kategorien, Tags,
Autorfelder, Shares und Likes; das Listen-**Ziel** und `actors` verwenden
entsprechend ihre vollständigen Listen- und Actordokumente. Plan und State speichern diese Werte
pro logischer UID und keine privaten Dokumentbytes. Vor
Kandidatenverifikation und unmittelbar vor dem Swap wird jeder betroffene
Fingerprint erneut berechnet; jede Abweichung ist `source_changed` und beendet
den Lauf ohne Swap.

Der CLI-Kern arbeitet über einen geschlossenen Logical-Index-Adapter. Jeder
PocketBase-Adapter nennt Quelle, Vollprojektor, Dokumentschema, Profilmanifest
und logische UID; der opake Legacylistenadapter nennt stattdessen ausschliesslich
Engine-Snapshotter und Baselineprofil. SRCH0 liefert PocketBase-Adapter für die
belegten Trail-/Actor-Baselines sowie Trail- und Listenziel. In nicht
exponiertem CI/Staging ist `lists-legacy-srch0-v1` das konkrete Listenziel. Für
jeden Produktionsplan muss SEC-VIS-0 über dieselbe Schnittstelle genau ein
`sec-vis0-list-v1`-Zielprofil liefern; dieses muss die hier definierte
lokale Aggregatprojektion und alle SRCH0-Listenfixtures unverändert einschliessen.
Es entstehen weder zwei aktive Listenziele noch ein zweites
Migrationsframework. Neue Adapter sind eine Schemaänderung und keine freie
Laufzeitkonfiguration.

Der Adapter besitzt zugleich eine einzige versionierte
Projektions-Abhängigkeitshülle. Dieselbe Registry steuert Vollprojektion,
Quellfingerprint, inkrementelle Engineprojektion und die Menge der im
Fachcommit zu invalidierenden logischen UIDs. Nach Ermittlung der vollständigen
Menge `A` liefert sie authority-abhängig dieselbe eindeutige Abbildung auf
Offline-Epochen/-Tasks für `O` beziehungsweise STATE1-Change-/Dirty-Ziele für
`S`; die Abbildung darf keinen UID-Effekt verlieren oder doppeln. Eine Mutation
muss alle logischen UIDs erfassen, deren vollständige Vorher-/Nachherprojektion
sich ändern kann; Unterinvalidierung ist verboten. Konservative Überinvalidierung
ist korrekt, wird aber im Performancebericht gezählt. Weil der
Epochen-Kurzpfad keine aktuelle Vollprojektion mehr berechnet, wäre ein
fehlender Fan-out jetzt eine falsch-grüne Readiness und nicht bloss eine
unvollständige Diagnose. Für das SRCH0-Profil gilt mindestens:

| PocketBase-Quelle beziehungsweise Relationsänderung | Betroffene logische UID und Dokumentmenge |
| --- | --- |
| projiziertes Trailfeld oder `author`-/`category`-/`subcategory`-/`tags`-Relation | `trails`, genau das Traildokument |
| Trail-`distance`, `duration`, `elevation_gain`, `elevation_loss` oder Entfernen eines referenzierten Trails | zusätzlich `lists`, jede vorher oder nachher referenzierende Liste |
| projiziertes Listenfeld, `needs_full_sync`-Materialisierungscommit oder Listen-`author`-/`trails`-Relation | `lists`, genau das Listendokument; der Full-Sync-Fan-out folgt dem Vorher-/Nachherstand der lokal materialisierten Relation |
| `activitypub_actors.username`, `preferred_username`, `domain`, `iri`, `icon` oder `is_local` | `actors`; bei `preferred_username`, `icon`, `domain` oder `is_local` zusätzlich alle vorher oder nachher vom Actor verfassten `trails` und `lists`; Delete konservativ alle drei UIDs |
| `categories.name` oder `categories.icon` | `trails`, alle vorher oder nachher referenzierenden Trails |
| `tags.name` | `trails`, alle vorher oder nachher referenzierenden Trails |
| `trail_share` beziehungsweise `trail_like` | `trails`, das referenzierte Traildokument mit `shares` beziehungsweise `likes`/`like_count` |
| `list_share` | `lists`, das referenzierte Listendokument mit `shares` |

Eine Änderung der Trail-`tags`-Membership gehört zur Trailzeile, nicht nur zum
Taghook. `subcategories` besitzt in diesem Profil keine direkte
Recordabhängigkeit, weil nur die Relation-ID des Trails projiziert wird; eine
Änderung dieser ID invalidiert dennoch `trails`. Reverse-Referenzen werden aus
Vorher- **und** Nachherzustand gebildet, damit Delete, Reparenting und
Relationswechsel keine abhängigen Dokumente auslassen. Benutzeränderungen, die
zuerst einen Actor materialisieren, nehmen anschliessend dessen vollständige
Actor-Abhängigkeitshülle.

Der geschlossene SRCH0-Listenadapter projiziert auch föderierte Listen
ausschliesslich aus dem persistenten PocketBase-Listen-, Trail- und
Actorbestand. Für Trailzahl und die vier Summenfelder ist
`r.ExpandedAll("trails")` die einzige Quelle; eine leere Relation ergibt für
Trailzahl und alle vier Summen jeweils den numerischen Wert `0`, auch bei
`needs_full_sync = true`. Die heutige Legacy-Verzweigung
`documentFromRemoteRecord` darf weder im Fingerprint noch in Reindex, Vergleich
oder Recovery noch in der normalen inkrementellen Create-/Update-Projektion
einen Remote-Searchrequest ausführen. `source_projection_incomplete` bezeichnet
nur eine lokal persistierte, aber nicht auflösbare Pflichtrelation, nie einen
nicht persistenten Origin-Aggregatwert oder einen regulären Federation-Stub.

Das produktive Profilset besteht ausschliesslich aus den konfigurierten
logischen UIDs `trails`, `lists` und `actors`. Reservierte physische UIDs der
Form `<logical>__srch0__<run-id>` liegen ausserhalb dieses Profilsets. Sie sind
niemals Routingziel, Readinessautoritat oder alleiniger Beleg dafür, dass ein
Zielprofil committed ist.

Die zwingende Migrationsmenge ist `Migrate = {lists, trails}`. Ihr Preflight
verlangt beide UIDs in `O`; der oben definierte Handoff-Guard verhindert, dass
STATE1 diese Voraussetzung vorher irreversibel auflöst. Ihre Kandidaten heissen
`<logical>__srch0__<run-id>`; für `lists` verwendet ein nicht exponierter
CI-/Staging-Lauf `lists-legacy-srch0-v1`, ein Produktionslauf zwingend das oben
beschriebene `sec-vis0-list-v1`. Der
Builder verarbeitet die Menge in kanonischer UID-Reihenfolge und:

1. legt jeden Kandidaten mit Primärschlüssel `id` an;
2. schreibt das vollständige Zielprofil und wartet die Settings-Task ab;
3. liest PocketBase per ID-Keyset-Pagination;
4. sendet Add-or-Replace-Batches mit höchstens 500 Dokumenten und höchstens
   8 MiB kanonischem JSON, je nachdem welche Grenze zuerst erreicht wird;
5. wartet jede zurückgegebene Task bis `succeeded` oder bricht ab; und
6. wiederholt anschliessend den vollständigen Vergleich jedes Kandidaten.

Existiert beim Preflight irgendeine der zufällig erzeugten runlokalen UIDs,
endet ein neuer Lauf vor dem ersten Create mit `candidate_uid_collision`; er
adoptiert, leert oder überschreibt sie nie. Nur
ein nichtterminaler Operationsmarker oder Kontrollrecord mit exakt derselben
Run-ID darf eine vorhandene Kandidaten-UID an `resume` binden.

Create, Settings-Update und Add-or-Replace auf den rungebundenen Kandidaten-UIDs
sind idempotente Stages. Nach einem Crash darf `resume` die ganze Stage oder
bereits gesendete Batches wiederholen; deshalb gibt es keine hundert
fsync-Intents für 10'001 Korpusrecords. Fehlgeschlagene Tasks, ein fremdes Dokument
oder ein abweichender Plan verhindern den Verifikationsbeleg.

Der Vollvergleich prüft Primärschlüssel, jede geordnete Settingliste,
Dokumentzahl, exakt dieselbe ID-Menge sowie jedes Feld des adaptereigenen
Volldokuments (`FullTrailDocument`, `FullListDocument` beziehungsweise
`FullActorDocument`). Er verwendet `POST /documents/fetch` mit der aus dem
Profilmanifest generierten expliziten Feldliste, festem Offset-/Limit-Paging
und ohne serverseitige Sortier- oder ID-Option. Dass diese konkrete POST-
Feldlisten-/Pagingform auf Meilisearch 1.11.3 wie normiert arbeitet, muss die
oben offen gehaltene reale Engine-Matrix bestätigen. Der Verifier schreibt nur `(id, kanonischer
Dokumentdigest)` in eine begrenzte temporäre On-Disk-Tabelle, weist doppelte
oder fremde IDs sofort ab und vergleicht die nach ID geordnete Gesamtmenge mit
der PocketBase-Projektion; die privaten Dokumentbytes gelangen nicht in den
Auditbeleg. `fields: ["*"]` ist verboten. Im Trailprofil werden
`difficulty_known` und `likes` so privilegiert geprüft, obwohl sie nicht in
Search-Hits dargestellt werden; das Listenprofil prüft entsprechend alle fünf
lokalen Aggregatfelder. Synthetische Semantikfixtures laufen auf einer separaten
Wegwerf-Engine und erzeugen `fixture-verification.json`, nie Dokumente im
Produktionskandidaten.

Für Offline-`recover` eines PocketBase-ableitbaren committed Profils setzt der
CLI vor dem ersten Enginewrite den zugehörigen PocketBase-Operationsmarker auf
`recovering`. Eine fehlende logische UID wird direkt aufgebaut; eine
vorhandene, bei bekanntem ableitbarem Profil inhaltlich abweichende UID wird
nach gesetztem Marker vollständig geleert und mit
demselben Builder neu befüllt. Jede Delete-/Create-/Settings-/Dokumenttask wird
terminal abgewartet. Erst nach Vollvergleich wechselt der Marker auf `ready`.
Ein Crash nach Create, Settings oder einem Dokumentbatch kann deshalb nie
allein aufgrund passender Settings ready werden; `ensure` setzt den Aufbau fort
oder startet ihn unter einer neuen Run-ID vollständig neu. Sind alle
Engineindizes verloren, baut `ensure` nur die Schnittmenge der
PocketBase-ableitbaren committed Profile mit `O` in dieser festen Reihenfolge:
`actors`, `lists`, `trails`. Eine dabei aktive nichtleere
`lists-legacy-v0` wird gerade nicht in diese Menge aufgenommen, sondern endet
`legacy_list_baseline_unrecoverable`. UIDs in `S` bleiben bytegleich
und werden separat durch STATE1s Generationen-/Restore-Recovery aufgebaut; erst
beide Ownergates öffnen Search. Das konfigurierte Offline-Profil kann
für `lists` bereits ein von SEC-VIS-0 beigesteuertes `sec-vis0-list-v1` sein.
Jeder neu aufgebaute Index wird vor Search-Readiness und Writefreigabe
vollständig verglichen.
Ein nur teilweise vorhandenes, aber bekanntes Profilset ergänzt fehlende
PocketBase-ableitbare Offline-Indizes; eine intakte aktive
`lists-legacy-v0` verlangt stattdessen Migration, eine fehlende oder gedriftete
die gebundene Backupwiederherstellung. `search_profile_conflict` bezeichnet nur eine konfigurierte
logische UID mit unbekanntem Primärschlüssel/Settingsprofil oder eine
mehrdeutige aktive Swapabbildung. Eine zusätzliche reservierte runlokale UID ist
für sich allein kein Profilkonflikt und wird von `ensure` nicht geroutet oder
mutiert.

### Swap, Aktivprüfung und Terminalentscheidung

Nach grünem Kandidatensatz und allen Pre-Swap-Voraussetzungen setzt der Lauf
die PocketBase-Operationsmarker der vollständigen Migrationsmenge in einem
Multi-Row-CAS auf `cutover` und veröffentlicht danach
genau ein `swap-intent.json`. Unter dem exklusiven Writerlock schliesst er
danach einen stabilen `task_quiescence_retry_v1`-Versuch ab, prüft Quelle,
Pre-Abbildung, Fingerprints und Gates erneut und publiziert unmittelbar vor
dem Request den No-Replace-Record
`swap-submit/forward/1.intent.json`. Dieser bindet die letzte Beobachtung als
`pre_submit_observation = (engine_scope_id, global_task_head)` und autorisiert exakt einen
HTTP-POST der bereits im logischen Intent digestgebundenen Payload.
Unmittelbar vor dem POST müssen Scope, Head und Pre-State noch gelten; jede
Abweichung verwirft den Attempt vor Socketwrite als `not_admitted`. Erst danach
sendet er genau einen atomaren Meilisearch-Swaprequest mit der kanonisch
geordneten Paarmenge
`lists <-> lists__srch0__<run-id>` und
`trails <-> trails__srch0__<run-id>` und wartet die direkt zurückgegebene Task
terminal ab. Im Produktionslauf trägt das Listenpaar zwingend das qualifizierte
`sec-vis0-list-v1`; nur ein nicht exponierter CI-/Staging-Lauf darf stattdessen
`lists-legacy-srch0-v1` binden. Das physische Paar bleibt identisch. Eine
zurückgegebene Task-ID wird zuerst in
`<attempt>.admission.json` No-Replace/`fsync`-durable; erst das spätere
terminale Ergebnis erhält den getrennten `<attempt>.decision.json`. Ein
Transportfehler nach möglichem Send lässt den Attempt dagegen
`swap_submission_unknown`; er ist niemals ein Non-Acceptance-Beleg und derselbe
Attempt wird nie erneut gesendet. Reverse-Swap und alle Folgeattempts verwenden
denselben Automaten und exakt dieselbe vollständige Paarmenge unter
`swap-submit/reverse/...`. Ein partieller Trail- oder Listen-Rückswap ist
verboten. Der
Maintenance- beziehungsweise heute vorhandene Master-Key bleibt bis zur
Terminalentscheidung nutzbar; SRCH0 widerruft nicht vorzeitig die einzige
Rückswapberechtigung.

Nach erfolgreichem Swap endet der One-shot zunächst ohne Öffnung des Ingress;
DB-Serveprozess, Web, Cronjobs und Indexworker bleiben gestoppt. Die
vollständige synthetische Matrix für Legacycompiler, Unknown-Difficulty, ACL,
Actorroute, Uploadhelper, Einzel- und Multi-Search, Profile, Cluster,
DTO-Allowlist und Proxy-Negativfälle ist zuvor in CI oder Staging am exakt
freizugebenden Build sowie Contract-/Profilmanifest grün. Sie wird nicht mit
Produktionsrecords oder kurzlebigen Canaries wiederholt.

Im Wartungsfenster erzeugt das mit SRCH0 gelieferte read-only
`srch0-production-verify`-Script zusätzlich `production-verification.json`.
Sein geschlossenes Schema bindet Run-ID, die vom installierten DB- und
Webartefakt deklarierten Contract-/Profilrevisionen und Artefaktdigests, die
Profilmanifestdigests, den Fixture-Matrix-Digest, die beobachtete aktive
UID-Abbildung und den erneuten Settings-, ID-, Feld-, Zähler- und
Dokumentdigestvergleich. Es bestätigt
ausserdem durch direkten read-only Vergleich mit `search_index_state`, dass
vor dem Aktivcommit noch die committed Baseline und `operation_state = cutover`
gelten: Die Engine zeigt bereits das Zielprofil, der normale Serve-Guard würde
deshalb erwartungsgemäss `not_ready/search_recovery_required` melden. Dieser fail-closed Zwischenzustand
wird nicht durch einen temporären Runtime-Bypass umgangen. Das Script legt
keine Canaries an, ruft weder anonyme noch authentifizierte SvelteKit-/Search-
Endpunkte auf, erzeugt oder verwendet keine Principals beziehungsweise Tenant-
Tokens und wertet die Produktionsinstanz nicht als ACL-/DTO-Orakel aus. Es
loggt keine Record- oder Actorpayloads; leere Produktionscollections sind
zulässig. `resume --activate` akzeptiert nur einen vollständig grünen Bericht
für denselben Lauf.

Danach gibt es genau zwei Terminalergebnisse mit geschlossenen
Voraussetzungen:

- **Erfolg:** DB und Web bleiben gestoppt. `resume --activate` validiert
  denselben Prüfbericht und Enginezustand. Es verlangt die höchste lückenlose
  Forward-Kette mit terminaler Decision `applied_verified` und das vollständige
  Fehlen eines Reverse-Intents oder Reverse-Records. Danach erhöht es die
  `search_index_state`-Revisionen der vollständigen Migrationsmenge in einer
  PocketBase-Transaktion, setzt aktive Zielrevisionen sowie
  `operation_state = ready` und publiziert danach
  `active.json`.
- **Fehler:** DB und Web bleiben gestoppt. Vor einem Swap-Intent darf `abort`
  den `cutover`-Marker nur bei lückenloser Write-ahead-Kette ohne Swap-Intent/
  Attempt sowie exakter stabiler Baseline-Pre-Abbildung/-Settings/-Fingerprints
  ohne Enginewirkung auf Baseline-`ready` zurücksetzen und den Pre-Swap-
  Terminalbeleg publizieren; jede Post-/Mischabbildung ist
  `swap_audit_lost`. Nach dem Vorwärtsswap darf es ein
  `reverse-swap-intent.json` ausschliesslich auf die terminale Forward-Decision
  `applied_verified` folgen lassen. Damit ist Aktivierung dauerhaft verboten.
  Es tauscht bei unverändertem Quellbestand über denselben Attempt-/Admission-/
  Decision-Automaten zurück. Die terminale Reverse-Decision
  `applied_verified` enthält selbst den vollständigen durablen
  Baselinevergleich; nach erneuter Source-/Quieszenz-/Decision-Prüfung setzt der
  gebundene Baseline-State-CAS `operation_state = ready`. Danach
  publiziert es `abort.json` mit beiden Ketten und dem Vergleichsdigest. Der
  alte kompatible Build bleibt bis zum folgenden Securityschritt gestoppt.

Die von `SEC-VIS-0` verlangte Search-Parent-Key-Rotation erfolgt **nach** der
Terminalentscheidung und vor dem normalen Start von DB und Web. Ziel- und
Rollback-Webartefakt sind bereits vor dem Fenster mit derselben, gegenüber der
zuletzt veröffentlichten Version erhöhten `search_token_cookie_epoch`
qualifiziert; beim ersten SRCH0-Release steigt die heutige
`SEARCH_TOKEN_VERSION` daher mindestens von `1` auf `2`. Ein Cookie mit
abweichender Epoche wird vor seinem ersten Engineeinsatz gelöscht und durch
einen neu vom DB-Endpunkt geholten Tenant-Token ersetzt.

Zusätzlich invalidiert der gemeinsame serverseitige Search-Adapter bei genau
einem Engine-`403` seinen intern gecachten Tenant-Token, holt ihn einmal neu und
wiederholt denselben bereits normalisierten Request genau einmal. Ein zweiter
`403` wird ohne Schleife als Suchfehler ausgegeben; clientgelieferte
Enginecredentials werden nie refreshed. `SEC-VIS-0` besitzt Rotation und
Epochensprung, `SRCH-COMP` den gemeinsamen Refreshpfad. Der CI-/Staging-Test
startet mit einem noch gültigen, bis zu 24 Stunden alten Cookie der Epoche `1`,
rotiert den Parent-Key, beweist Löschen, Neuausstellung und höchstens einen
Retry und erhält anschliessend eine erfolgreiche Suche ohne Retryloop.

Der Produktions-Alt-Token-Negativtest prüft nach der Terminalentscheidung nur
die Credentialrevocation direkt und datenunabhängig; er ist kein
authentifizierter ACL-Smoke und verwendet keine Produktionsrecords als Orakel.
Erst danach starten der Ziel- oder beim Abbruch der neu epochierte kompatible
Rollback-Build bei weiterhin geschlossenem externem Ingress. Ihre grüne
Search-Readiness und das erfüllte SEC-VIS-0-Gate dürfen anschliessend den
Ingress öffnen. Die Token-/Cookie-Epoche ist von
`search_contract_revision` und Indexattestierung getrennt und erzwingt keinen
Reindex oder späteren Build-Reattest. SRCH0 behauptet keine Revocation, solange
Compose denselben Master-Key an mehrere Dienste verteilt.

### Wiederaufnahme- und Abbruchmatrix

`resume` leitet den Zustand aus den normativen Dateien, `search_index_state`
und dem tatsächlichen Enginezustand ab; Dateinamen oder die höchste Task-ID
allein genügen nicht.
Wo die folgende Matrix Baseline-Fingerprints oder -Vollvergleich sagt, bedeutet
dies für `trails` den PocketBase-gegen-Engine-Vollvergleich, für
`lists-legacy-v0` dagegen ausschliesslich den oben definierten exakten
Engine-gegen-`opaque_legacy_baseline_v1`-Vergleich. Ein Origin-Dial oder ein
PocketBase-Quellfingerprint darf den Listenbeleg niemals ersetzen.

Je Richtung wird ausschliesslich die höchste vollständig validierte,
lückenlose und digestverkettete Attemptfolge ausgewertet. Eine Attempt-Zeile
der folgenden Matrix meint deren höchsten Attempt ohne Decision und ohne
Nachfolger; die Retryzeile meint die terminale Decision des höchsten Attempts,
für den noch kein Nachfolger existiert. Eine Lücke, ein Fork, ein falscher
Vorgängerdigest, eine für den Decisionzustand unzulässige Admissionkombination
oder irgendein Record nach Richtungsterminalität endet mit
`swap_audit_conflict` und null Engine-/PocketBase-Mutation.

| Beobachtung | Erlaubte Aktion |
| --- | --- |
| kein Plan, Operationsmarker `ready` | neuer `run`; keine Enginewirkung dieses Laufs anzunehmen |
| kein Plan, Operationsmarker `recovering` | neuen Recoveryplan auf demselben committed Profil anlegen und vollständig neu aufbauen oder idempotent fortsetzen |
| kein Plan, Operationsmarker `cutover` | `swap_audit_lost`: Die normative Intent-/Attemptkette des nichtterminalen Swapfensters ist unbeweisbar und darf weder rekonstruiert noch durch `swap_effect_proof_v1` ersetzt werden; keine Adoption, Aktivierung, Rücksetzung auf `ready`, kein Reverse-Intent und kein Enginewrite. Der Effect-Proof ist nur bei erhaltener Attemptkette anschlussfähig |
| Plan, mindestens ein Kandidat fehlt oder ist partiell | Create/Settings/Batches für den gebundenen Kandidatensatz idempotent wiederholen, danach alle voll vergleichen |
| Kandidatensatz vollständig, Verifikationsbeleg fehlt | Quellen und Kandidaten erneut vollständig vergleichen und Beleg publizieren |
| Verifikationsbeleg, `cutover`-Marker, kein Swap-Intent | vorwärts nur nach erneutem Lock-, Quellen- und Gatecheck; `abort` darf den Marker nur bei lückenloser Write-ahead-Kette ohne Swap-Intent/-Attempt und exakter stabiler Baseline-Pre-Abbildung/-Settings/-Fingerprints auf Baseline-`ready` zurücksetzen. Post-/Mischabbildung ist `swap_audit_lost` und mutiert nichts |
| logisches Vorwärts-/Reverse-Intent, aber noch kein Submit-Attempt dieser Richtung | nach stabilem Quieszenzversuch und vollständigem Gatecheck Attempt `1` No-Replace publizieren und dessen Payload exakt einmal posten |
| höchster Submit-Attempt ohne Decision mit durabler Admission; die exakte Post-Abbildung ist noch nicht vollständig belegt | nur im selben ununterbrochenen exklusiven SRCH0-Prozesskontext die direkt zurückgegebene Task weiterverfolgen; nach Restart darf allein `swap_effect_proof_v1` über den exakten Post-State entscheiden, sonst `swap_submission_unknown` |
| höchster Vorwärts-Attempt ohne Decision; beide Ziele sind aktiv und der Kandidatensatz enthält die beiden Baselines | im selben ununterbrochenen exklusiven SRCH0-Prozesskontext mit direkter Admission/Taskbeleg, sonst ausschliesslich mit `swap_effect_proof_v1`, die Post-Abbildung vollständig verifizieren und `applied_verified` publizieren; nie erneut vorwärts tauschen |
| Ziel aktiv, höchste Vorwärts-Decision `applied_verified`, kein Reverse-Intent und noch kein Terminalbeleg | Offline-Produktionsverifikation fortsetzen und danach aktivieren oder genau einmal das logische Reverse-Intent publizieren; ab dem Reverse-Intent ist Aktivierung verboten |
| höchster Reverse-Attempt ohne Decision; Baseline wieder aktiv | im selben ununterbrochenen exklusiven SRCH0-Prozesskontext mit direkter Admission/Taskbeleg, sonst ausschliesslich mit `swap_effect_proof_v1`, die Post-Abbildung vollständig verifizieren und Reverse-`applied_verified` publizieren; noch weder State-CAS noch `abort.json` behaupten |
| Reverse-Decision `applied_verified`, Operationsmarker noch `cutover` | deren durablen vollständigen Baseline-Postvergleich sowie Source-/Quieszenz-/Decision-Bindings erneut prüfen, dann den gebundenen Baseline-State-CAS auf `ready` committen und erst danach `abort.json` mit beiden Ketten und Reverse-Decision-Digest publizieren |
| vollständige Forward-/Reverse-Ketten, Baseline-State bereits committed `ready`, `abort.json` fehlt | ausschliesslich den bytegleichen Terminalbeleg nachliefern; weder erneut vergleichen noch rücktauschen oder State ändern |
| Attempt ohne Admission nach möglichem Socketwrite | keine Task adoptieren und aus keinem leeren Taskintervall `not_admitted` ableiten; exakter Post-State darf über `swap_effect_proof_v1` `applied_verified` ergeben, sonst terminal `ambiguous`/`swap_submission_unknown` ohne Folgeattempt |
| höchste Decision ist `not_admitted` oder qualifiziert `failed_no_effect`, es gibt noch keinen Nachfolger; Quelle, Pre-Abbildung, Settings und Fingerprints sind unverändert | nächsten lückenlosen, digestverketteten Attempt nach frischem stabilem Quieszenzversuch publizieren und genau einmal posten |
| committed Ziel-DB-Revision, weder Reverse-Intent/-Record noch `active.json` vorhanden | nur den Auditbeleg nachliefern; kein Rückswap. `srch0_pair_handoff_ready_v1` bleibt bis zu dieser Nachlieferung falsch |
| `active.json` plus committed DB-Revision | kein Rückswap; spätere Korrektur ist ein neuer Vorwärtslauf |
| Attempt ohne durable Admission nach möglichem Submit; Prozess/Scope verloren, Task-GC oder Annahme nicht eindeutig | nur eine bereits stabile exakte Post-Abbildung darf durch `swap_effect_proof_v1` zu `applied_verified` werden; sonst `swap_submission_unknown`, eine bloss ähnliche Task nicht adoptieren, keine Decision erfinden und kein neuer Attempt |
| Attempt mit durabler Admission nach Restart | nur `swap_effect_proof_v1` über den exakten Post-State darf `applied_verified` entscheiden; sonst `swap_submission_unknown`, Taskstatus oder -wirkung nicht erraten und kein neuer Attempt |
| unbekannte, gemischte oder mehrdeutige Abbildung | `swap_state_ambiguous`; keine automatische Mutation |

Für die Swapzuordnung prüft der CLI logisches Intent, Attempt, erwartete Vor-/
Nachfingerprints, exakten Requestdigest, Tasktyp und beide UIDs. Eine direkt
zurückgegebene Admission darf nur in demselben ununterbrochenen exklusiven
SRCH0-Prozesskontext taskbasiert bis zur Decision verfolgt werden. Blosse
Abwesenheit in einer vollständig paginierten
Taskliste, eine nackte Task-ID, höchste Task-ID oder stabile Pre-Abbildung
beweist niemals Non-Acceptance. `not_admitted` verlangt deshalb entweder den
lokalen Write-ahead-Beleg, dass kein Socketwrite begonnen hat, oder eine
typisierte synchrone Engineablehnung. `failed_no_effect` verlangt direkte
Admission, eine in demselben Prozesskontext terminal beobachtete
`failed|canceled`-Task, die
versionsqualifizierte atomare Nichteffektgarantie und dieselbe vollständige
Preimage-Prüfung. Nur eine dieser Decisions autorisiert den nächsten
lückenlosen Attempt; er bindet ihren Digest, einen frischen stabilen
Quieszenzversuch und wird ebenfalls genau einmal gesendet.

Nach Prozess-, Scope-, Endpoint- oder Sessionverlust bleibt jede mögliche
Annahme `swap_submission_unknown`. Unabhängig davon, ob bereits eine Admission
durable ist, darf allein `swap_effect_proof_v1` den stabil eingetretenen exakten
Post-State ohne Taskkontinuitätsbehauptung als `applied_verified` abschliessen;
er beweist nie `not_admitted` und autorisiert nie einen Submit. Pre- oder
Mischabbildung sowie mehrere möglicherweise gesendete offene Attempts bleiben
fail-closed ohne New-Attempt-Pfad. Vorwärts- und Rückswap verwenden exakt
denselben Automaten.

Geht das Auditverzeichnis verloren, bleibt ein terminal aktiver, settings- und
inhaltskompatibler PocketBase-ableitbarer Index ready. Für eine solche UID in `O` darf ein neuer `verify`-Lauf
durch vollständigen PocketBase-gegen-Engine-Vergleich einen neuen Auditbeleg
erzeugen. Geht das Enginevolume verloren, erkennt `ensure` fehlende
PocketBase-ableitbare Offline-Indizes und baut ausschliesslich diese Teilmenge
von `O` im Modus `recover` vollständig neu. Eine noch aktive
`lists-legacy-v0` bleibt ohne exakt passendes Enginebackup
`legacy_list_baseline_unrecoverable`;
`S` bleibt read-only und wird von STATE1s Generationen-/Restore-Recovery
rekonstruiert. Search öffnet erst nach beiden Ownergates. Diese Pfade gelten für
Altinstallationen genauso wie für neue Installationen.

Kandidaten werden nie automatisch gelöscht. Nach `active` oder `abort` darf
`resume --cleanup` nur die im Terminalbeleg vollständig gebundene runlokale
UID-Menge entfernen; der Operator muss diese Option gesondert auslösen. Vorher
dient der Kandidatensatz beim Aktivpfad als Rückfallbaseline; nach geöffneten
Writes gilt er als veraltet und darf nicht mehr zurückgetauscht werden. Solange
SRCH0 Offlineowner bleibt, darf der Operator die veraltete Baseline zu
Diagnosezwecken behalten. Der Trail-Handoff ist damit jedoch
unvereinbar: Er verlangt zuerst den vollständigen idempotenten Cleanup und
prüft unter seinem Lock erneut, dass keine reservierte Paar-UID existiert. So
bleibt nach der irreversiblen Authorityübergabe kein Engineobjekt ohne
Löschowner zurück.

Beim nächsten Start bindet nur ein passender nichtterminaler
`operation_run_id` eine solche UID an Recovery. Eine terminal gebundene UID ist
ein expliziter Cleanup-Kandidat; eine reservierte UID ohne noch vorhandenen
Runbeleg ist ein diagnostischer Orphan. Beide werden gezählt und können eine
Plattenwarnung auslösen, verursachen aber weder `search_profile_conflict` noch
eine stille Adoption. Ihre Entfernung verlangt immer einen fingerprint- und
UID-gebundenen Operatorbefehl: terminal gebunden `resume --cleanup`, ohne
Runbeleg ausschliesslich den oben definierten `cleanup-orphan`-Pfad.

### Startup-Zustandsmatrix

Der bisherige `initMeilisearchDocuments`-Goroutinepfad mit
`DeleteAllDocuments` entfällt. Der Listener läuft zunächst im oben definierten
liveness-only Bootstrapzustand; `ensure` läuft synchron vor Search-Readiness,
Writes und Workern und wartet jede von ihm erzeugte Task:

Vor Auswertung partitioniert `ensure` jede konfigurierte UID nach ihrer
durablen Authority in `O` und `S`. Sämtliche Epochen-, Zähler-,
Vollvergleichs-, Recovery-, Profil- und Operationsmarkerzeilen der folgenden
Matrix gelten nur für `O`. Für `S` sind die eingefrorenen Offline-Epochen und
-Zähler auch bei legitimer späterer Abweichung kein Eingang; diese UIDs werden
ausschliesslich read-only über ihre STATE1-Head-/Generationen-/Publishergates
bewertet. Allgemeine Transport- und Authoritykonflikte bleiben
ownerübergreifend fail-closed.
Authoritykonflikte und offene `operation_state = recovering|cutover`-
beziehungsweise nichtleere Operationsfelder besitzen zuerst Präzedenz. Nur bei
`operation_state = ready` und vollständig leeren Operationsfeldern besitzen die
beiden `lists-legacy-v0`-Zeilen danach vor allen Kurzpfad-, Vollvergleichs- und
allgemeinen Recoveryzeilen Präzedenz; ein fehlender PocketBase-Projektor darf
nie durch eine allgemeinere Zeile umgangen werden.

| Engine-/PB-Zustand | `SEARCH_INDEX_STARTUP=auto` |
| --- | --- |
| `srch0_pair_activated_v1` ist falsch, während mindestens eine UID aus `{lists, trails}` bereits Authority `state1` besitzt | `search_pair_migration_authority_conflict`; liveness-only und vollständig mutationsfrei bleiben. Die irreversible Authority wird weder zurückgegeben noch durch einen unvollständigen Offline-Cutover umgangen; benötigt wird ein operatorgeprüfter Restore auf den letzten konsistenten Control-Plane-/Engine-Schnitt |
| `operation_state = recovering` oder ein anderer offener Recoverymarker ist vorhanden und aktives sowie Zielprofil sind bekannt und PocketBase-ableitbar | direkten Recoveryaufbau idempotent fortsetzen oder vollständig neu beginnen; erst nach Vollvergleich `ready` und normalen Serve freigeben |
| Offline-Zielindex ist neuer als DB-State, `operation_state = cutover` oder ein anderer Cutovermarker ist vorhanden | `search_recovery_required`; bei intakter Auditkette ausschliesslich `resume` oder `abort`, bei verlorener Cutover-Kette `swap_audit_lost`; niemals einen neuen `verify`- oder Migrationslauf oder stilles State-Nachziehen zulassen. Ein neuer `verify` ist erst nach einem bereits terminal committed `ready` zulässig |
| alle aktiven Offlineprofile sind PocketBase-ableitbare Ziel-/unterstützte Profile; UIDs, Revisionen und Settings passen; pro UID in `O` sind `applied_projection_epoch = attested_projection_epoch`, Grund leer und Engine-Zähler gleich `attested_document_count`; globaler Taskguard ist `ready` und sein Head entspricht dem aktuellen ungefilterten Head | weder PocketBase-Vollprojektor noch Engine-Dokumentfetch für `O` aufrufen; genau einen globalen Head-Read ausführen, Epochen-Kurzpfad grün und nach zusätzlich grünen STATE1-Ownergates normalen Serve atomar freigeben |
| auf einer UID in `O` fehlen Epochen oder sind ungleich, Zähler weicht ab, `verification_required_reason` ist gesetzt oder der globale Taskguard ist invalid/abweichend | höchstens drei `task_quiescence_retry_v1`-Versuche ausführen; nur ein stabiler Vollvergleich aller UIDs in `O` darf UID-Attestierungen und globalen Taskguard erneuern, anhaltende Unruhe endet retrybar mit `search_task_quiescence_failed`; Grund erst mit neuer grüner Vollvergleichsattestierung leeren |
| Epochen-/Zählerattestierung fehlt oder weicht ab, Vollvergleich aller UIDs in `O` ist aber grün | nur tatsächlich abweichende UID-Attestierungsfelder und den globalen Taskguard in derselben SQLite-Transaktion erneuern; sind die UID-Felder bereits exakt und nur der Guard war invalid, ausschliesslich Guard-CAS; keine Engine-Mutation; normalen Serve freigeben |
| alle UIDs in `O` fehlen und ihre committed Profile sind PocketBase-ableitbar | vollständiger Offline-`recover`-Aufbau aus PocketBase mit der zuletzt committed Profilrevision, dann nach grünen STATE1-Gates normalen Serve freigeben; ohne committed State ist nur die leere Baselinefrei-Initialisierung zulässig |
| einzelne UIDs in `O` fehlen, ihre committed Offline-Profile sind bekannt und PocketBase-ableitbar | nur diese Offline-Indizes mit der committed Profilrevision vollständig recovern, Profilset und STATE1-Gates prüfen, dann normalen Serve freigeben |
| PocketBase-ableitbares Profil und State passen, aber der verpflichtete Vollvergleich findet andere IDs/Felder/Digests | Operationsmarker auf `recovering`; betroffene Indizes vollständig aus PocketBase reparieren und vergleichen, erst dann `ready` |
| bei `operation_state = ready` und leeren Operationsfeldern ist aktive `lists-legacy-v0` mit bekanntem Primärschlüssel/Settings intakt | `legacy_list_migration_required`; liveness-only bleiben und den expliziten Produktionslauf mit der kanonischen `{lists, trails}`-Menge, `lists=sec-vis0-list-v1` und Bestätigung ausgeben; weder Origin-Dial noch Serve-Attestierung |
| bei `operation_state = ready` und leeren Operationsfeldern fehlt aktive `lists-legacy-v0` oder weicht von einem bereits gebundenen opaken Snapshot ab | `legacy_list_baseline_unrecoverable`; keine automatische Mutation oder Einwegmigration, exakt passendes Enginebackup wiederherstellen und danach neu planen |
| ausschliesslich ein PocketBase-ableitbares Legacyprofil ist aktiv und vollständig vergleichbar | keine stille Migration; der Build darf es bei sonst grünen Gates weiter bedienen, semantischer Wechsel nur im expliziten Zwei-Paar-Migrationslauf |
| eine reservierte `*__srch0__<run-id>`-UID liegt zusätzlich herum, aber kein offener Marker bindet sie | logisches Profilset normal prüfen; nur Orphan-/Cleanup-Diagnose, kein Konflikt und keine Mutation |
| UID ist an `control_plane_authority = state1` übergeben | Offline-`ensure` ist read-only; STATE1-Head-/Publishergates entscheiden Readiness und Recovery |
| vollständig gebundener Authority-Mix aus STATE1- und Offline-UIDs | jeden Teil ausschliesslich mit den Gates seines Owners prüfen; kreuzende Fachwrites verwenden die atomare Mischtransaktion, der Mix allein ist kein Recoveryfehler |
| Engine nicht erreichbar | `search_engine_unreachable`; Prozess bleibt liveness-only und meldet sich nicht suchbereit |
| konfigurierte logische UID hat unbekannte Settings oder die aktive Swapabbildung ist mehrdeutig | `search_profile_conflict`; keine automatische Mutation |

Für `SEARCH_INDEX_STARTUP=manual` gilt zusätzlich diese geschlossene
Freigabematrix:

| Ergebnis des read-only Starts | Prozessphase und Wirkung |
| --- | --- |
| Offline-Epochen-Kurzpfad oder Offline-Vollvergleich ist grün und sämtliche STATE1-Ownergates sind grün | bei Vollvergleich ausschliesslich für `O` die Control-Attestierung per CAS persistieren; `S` bleibt bytegleich, danach `search_ready`, Writes, alle Worker und Search gemeinsam öffnen |
| vollständig beendeter Vollvergleich findet auf den Offline-UIDs nur Dokumentdrift bei bekanntem committed Profil, `operation_state = ready`, vollständig gebundener Authoritypartition und ohne offenen Task | unabhängig von den STATE1-Ownergates sofort für die vollständige kanonische Driftmenge in einem SQLite-Commit `document_drift` per gebundenem Multi-Row-CAS dauerhaft setzen beziehungsweise einen vorhandenen nichtleeren Grund erhalten; bei Fehler rollt alles zurück und `search_initializing` bleibt bestehen |
| der vorherige Drift-Commit ist durable und alle STATE1-Ownergates sind grün | danach `search_manual_intervention`, allgemeine Writes und nicht suchende Worker einschliesslich Federation-Inbox/-Outbox öffnen, Search und jede Engineprojektion bleiben gesperrt; exakte gestoppte Offline-Reindex-/STATE1-Drainanweisung protokollieren |
| der vorherige Drift-Commit ist durable, aber mindestens ein STATE1-Ownergate ist nicht grün | in `search_initializing` liveness-only bleiben; allgemeine Writes und Worker geschlossen halten; der persistierte Grund verbietet nach Crash oder Neustart ohne `--full` trotzdem den Epochen-Kurzpfad |
| danach ausgeführter Fachwrite | für `O` Applied/Fan-out source-only erhöhen und Attested nicht nachziehen; für `S` STATE1-Change/Dirty im selben Fachcommit persistieren; keinen der beiden Engine-Lieferpfade starten |
| Vergleich unvollständig, Engine unerreichbar, Profil unbekannt/gemischt, Operation offen, deklarierte Handoffmenge halb gebunden/inkonsistent oder Lock verloren | in `search_initializing` liveness-only bleiben; keine Writes oder Worker öffnen |

Der automatische Pfad rekonstruiert nur die zuletzt committed
Vertragsrevision; er aktiviert keine neue Semantik. Dadurch repariert ein
normales `docker compose pull && up -d` weiterhin eine leere abgeleitete Engine,
wenn alle committed Profile PocketBase-ableitbar sind. Eine aktive
`lists-legacy-v0` ist die ausdrückliche Ausnahme und verlangt vor dem neuen
Serve den gestoppten Migrationslauf beziehungsweise bei Verlust ihr passendes
Enginebackup. Der Pfad überspringt den geplanten SRCH0-Cutover nicht und
benötigt nach kompatiblen Builds keinen externen Signer.

## Akzeptanz- und Negativtests

Die interne Marke **Korpus/Tooling fertig** verlangt:

- Schema, Manifest und menschenlesbare Matrix stimmen überein; jede
  Pflichtfamilie besitzt positive, negative und relevante Randfälle.
- Jeder produktive First-Party-Filter, alle neun sichtbaren Sortierungen und
  alle Engine-Consumer sind inventarisiert. Actorroute und Upload-
  Duplikatprüfung besitzen die oben festgelegten eigenen Fälle.
- Die feldweise URL-/Storage-Präzedenz, Darstellungsmodus-Buckets und raw
  Storage-Sortkeys sind reproduziert und korrekt klassifiziert.
- `_geoRadius` erscheint im Zielauftrag genau einmal; Treffer, `total`,
  Reihenfolge und Seite bleiben gegenüber den zwei identischen
  Baselineklauseln gleich.
- Missing, leere und ungültige lokale Difficulty wird bei Create, Core-Patch,
  Relationspatch und Vollreindex `null`, niemals `easy`; der Default enthält
  Unknown und beide Difficulty-Sortierrichtungen stellen Unknown zuletzt.
- `FullTrailDocument`, Core-, Share- und Like-Patch konvergieren auf denselben
  materialisierten Endzustand. Ein Core-Patch verliert vorhandene Shares oder
  Likes nicht.
- Meilisearch 1.11.3 und 1.36.0 bestehen Settings-, Filter-, Sortier-,
  `POST /documents/fetch`-Feldlisten-, Search-Displayed- und Swaptests.
- Der Reindex-CLI kann `trails`, `lists` und `actors` aus PocketBase aufbauen;
  seine geschlossene Adapter-/Profilgrenze nimmt das von SEC-VIS-0
  beigesteuerte `sec-vis0-list-v1` auf, ohne ein zweites Reindexwerkzeug oder
  eine freie Runtime-Projektion einzuführen.
- Ein Produktionsplan mit `lists=lists-legacy-srch0-v1` oder ohne das
  qualifizierte SEC-VIS-0-Manifest endet vor Planpublikation mit
  `security_list_profile_required`; genau derselbe Projektor bleibt auf einer
  nicht exponierten Wegwerf-Engine testbar.
- Eine intakte `lists-legacy-v0` mit absichtlich von PocketBase abweichenden
  Remoteaggregaten wird ohne Origin-Dial und mit ausgelassenem
  `fields`-Parameter von `POST /documents/fetch` in
  `opaque_legacy_baseline_v1` gebunden.
  Vorwärts- und Rückswap verwenden stets die vollständige `{lists, trails}`-
  Paarmenge; der Rückswap liefert für die Liste exakt denselben Settings-, ID-/
  Feld-, Count- und Dokumentfingerprint. Fehlende oder veränderte physische
  Listenbaseline endet `legacy_list_baseline_unrecoverable`, führt weder
  PocketBase-Rebuild noch Einweg-Cutover aus und wird erst nach passender
  Enginebackup-Rückspielung neu planbar. Der neue Serveprozess antwortet bei
  intakter, aber noch aktiver Legacylistenbaseline
  `legacy_list_migration_required` statt sie zu attestieren.
- Nach dem Aktivierungs-CAS blockiert ein fehlendes `active.json` den Paar-
  Handoff mit `search_pair_activation_evidence_required`, bis artifact-only
  `resume` den Beleg nachliefert. `resume --cleanup` wird nach dem ersten der
  beiden Delete-Index-Tasks gecrasht und schliesst beim Retry die bereits
  fehlende gebundene UID idempotent sowie die verbleibende terminal ab. Vor
  vollständiger Abwesenheit aller reservierten Paar-UIDs endet der Handoff mit
  `search_pair_cleanup_required`; danach bindet er Terminaldigest und leere
  Kandidatenbeobachtung. Der erste Bootstrap akzeptiert als einzige nichtleere
  Handoffmenge `["trails"]`; `["lists"]`, `["actors"]`, eine gemeinsame
  Listen-/Trailmenge und unbekannte UIDs enden im vorgelagerten Dispatch mutationsfrei mit
  `state1_handoff_uid_unsupported`. `lists` bleibt in V1 offline; ein späterer
  Versuch, sie in den bereits aktiven Head aufzunehmen, endet mutationsfrei
  `state1_incremental_authority_handoff_unsupported`.
- `run --mode auto` erzeugt in keinem Engine-/Statezustand einen Kandidaten oder
  Swap; erst `--mode migrate` mit exaktem Ziel und
  `--confirm-cutover` darf den Semantikwechsel planen.
- Der Compose-Systemtest misst freie Bytes am aufgelösten
  `/meili_data/data.ms`, übergibt sie als `--available-bytes`, bindet den
  Engine-`databaseSize`-Wert und löst die Kapazitätsgrenze vor dem ersten Write
  reproduzierbar aus.
- Der Bootstrap-Test hält `GET /health` auch bei einem künstlich länger als das
  Compose-Healthcheckbudget laufenden Vollvergleich grün; Search, Writes und
  Worker bleiben während des Vergleichsfensters gesperrt. Ein gesunder Start
  nach Create, Update und Delete beweist bei gleichen Epochen **null** Aufrufe
  des PocketBase-Vollprojektors und **null** Engine-Dokumentseiten; erzwungener
  `--full`-Vergleich sowie Fallback bei Epochen-/Zählerabweichung sind getrennt
  abgedeckt.
- Die Response-Order-Probe blockiert Engine-Submit, Task-Wait und Stats. Ein
  erfolgreicher Fachwrite antwortet trotzdem; Instrumentierung beweist streng
  `SQLite-Commit < Response-Lifecycle-Ende < erster Engine-Submit` und bis
  einschliesslich Attestierung exakt null Write-seitige Stats-Aufrufe. Die
  Crashmatrix stoppt vor Fachcommit, nach Fachcommit/vor Response-Finalizer,
  vor Submit, bei unbekannter Taskannahme, während `enqueued|processing`, nach
  einem `failed|canceled`-Batch, nach terminalem Gesamterfolg/vor
  Attestierungs-CAS sowie in einem bewusst wegen eines anderen offenen Owners
  oder STATE1-Acks noch nicht global/prospektiv ruhigen Fixture nach UID-
  Attestierung/vor der späteren Guard-CAS und nach deren Erfolg. Beim letzten
  beziehungsweise einzigen Owner existiert dieser Zwischencrashpunkt wegen der
  kombinierten Transaktion nicht. Nur der letzte Zustand besitzt Epochen- **und** Taskhead-
  Gleichstand; jeder Zustand ab durablem Fachcommit bis davor
  vollvergleicht beim Start. Ein SDK-Wait ohne Go-Fehler, aber mit
  `failed|canceled`, ist ein Fehler.
- Trennt der Client nach durablem Commit vor Empfang der Antwort, führt der
  `defer`-Finalizer das Envelope dennoch erst nach Rückkehr des Responseversuchs
  genau einmal dem Owner zu. Bei injiziertem Verlust von Finalizer/Wake-up und
  weiterlaufendem Prozess schliesst der 5-s-Watchdog Search und weitere
  projektionsrelevante Admission, erzeugt keinen verfrühten Engine-Submit und
  setzt die kombinierte Failure-/Recovery-CAS. Ein später Callback ist ein
  No-op; der Test wartet nicht auf einen Neustart, um die Sperre zu sehen.
- Der Quieszenztest injiziert eine verzögerte fremde Registration zwischen
  `H0/H1` sowie während des Vollvergleichs vor `H2`; jeweils wird der gesamte
  Versuch verworfen. Weitere unregistrierte Injektionen liegen nach `H2` vor der
  Attestierungs-CAS, nach der CAS vor der initialen Observerlesung und nach
  Search-Freigabe. Die ersten beiden öffnen Search nie; die dritte schliesst den
  lokalen Guard und persistiert den invaliden Taskguard innerhalb 1 s. Ein
  eigener, binnen desselben Budgets direkt-`202`-klassifizierter Task bleibt
  dagegen bis zu seinem 120-s-Terminalbound observergrün. Ein Crash an jeder dieser
  Stellen verhindert über den gespeicherten alten Head den nächsten Kurzpfad.
  Der Test erlaubt ausdrücklich Antworten innerhalb des dokumentierten
  Detektionsfensters und behauptet ohne Enginefence kein Nullfenster. Drei vor
  der CAS flappende Versuche enden innerhalb des Gesamtzeitbudgets mit
  `search_task_quiescence_failed`, unveränderter Attestierung und retrybarem
  Latch. Unbekannte Tasktypen, Scopewechsel und globale Swaptasks ohne
  `indexUid` sind abgedeckt; die Matrix behauptet ausdrücklich keine
  Admission-Linearisierung.
- Ein eigener Submit wird angehalten, bis sein Task bereits im globalen Head
  sichtbar ist. Trifft seine direkte `202` noch innerhalb des verbleibenden
  1-s-Budgets ein, wird er exakt zugeordnet und die provisorische
  Search-Admission-Pause wieder geöffnet; bleibt sie länger aus oder liegt ein
  fremder Task dazwischen, sind lokaler und persistenter Guard spätestens bei
  Budgetablauf geschlossen. Ein zweiter Fall hält den POST ohne sichtbaren Head
  über die 5-s-Submitdeadline: Er endet `task_submission_unknown`, schliesst
  beide Guards ohne Attestierung, und eine erst danach sichtbare Task öffnet sie
  nicht wieder. Eine frühe qualifizierte synchrone Non-Acceptance geht sofort
  in `task_failed`, eine andere Non-202-Antwort sofort in
  `task_submission_unknown`; beide warten nicht auf die Deadline und
  attestieren nichts. Feste Pollslots mit Headantworten nach 249, 300 und
  499 ms halten das Budget ohne falsche Invalidierung; bei 500-ms-Timeout wird
  fail-closed invalidiert. Ein Transport, der am exakten Deadline-Rand trotz
  Cancellation noch offen ist, lässt Slot `n+2` ausfallen und hält die Zahl
  offener Requests bei zwei. Vertauscht eintreffende überlappende Pollantworten
  oder der späte Timeout eines älteren Polls dürfen keinen neueren Head
  zurücksetzen beziehungsweise dessen grünen Zustand schliessen.
- Während der Task für `E1` länger als 1 s, aber kürzer als die 120-s-
  Terminalfrist, und über mehrere Observerpolls
  künstlich `processing` bleibt, beantworten 500 weitere Writes derselben UID
  ohne Engine-Rundreise und ohne falsche Guardinvalidierung. Es existieren höchstens
  ein laufender Batch und ein koaleszierter Dirty-Suffix; Attested bleibt auf
  `T0` und springt nach allen Erfolgen mit genau einer CAS auf `Efinal`.
  Create→Update liefert das letzte Volldokument und Delta `+1`, Create→Delete
  keinen Task und `0`, Delete→Create desselben Keys das letzte Volldokument und
  `0`. Ein fehlender beziehungsweise in Reihenfolge `E2,E1` zugestellter
  Response-Callback beweist lückenlose Epochordnung; Verlust oder Crash lässt
  die Lücke für den Startup-Vollvergleich stehen. Ein erfolgreicher Präfix mit
  fehlgeschlagenem Suffix wird nie teilweise attestiert. Derselbe Task knapp
  vor 120 s bleibt `expected_pending`; knapp nach 120 s schliessen beide Guards,
  der passende Offline- beziehungsweise STATE1-Owner geht in seinen
  Fehlerzustand und kein Präfix wird attestiert.
- Zwei disjunkte UID-Ketten halten den globalen Taskguard bis zur gemeinsamen
  ruhigen Grenze auf dem alten Head. Ist UID A bereits epochengleich/`idle` und
  UID B der letzte vollständig erfolgreiche `attesting`-Owner, schreiben dessen
  Attestierungs-CAS und die Guard-CAS genau eine SQLite-Transaktion gegen die
  prospektiv ruhige Postimage; erst nach Commit wird B `idle`. Ein injizierter
  CAS-/Commitfehler schreibt weder UID noch Guard und behauptet B nicht idle.
  War auch B schon getrennt attestiert, folgt stattdessen höchstens eine
  nachgelagerte koaleszierte Guard-CAS. Der bereits terminale Registerrecord von UID A bleibt erhalten,
  solange UID B noch `processing` ist, und wird erst nach der beide
  einschliessenden Guard-CAS GC-fähig. Ein eigener direkt zugeordneter Task löst keinen falschen
  Observeralarm aus; unbekannte Annahme, Polltimeout, lückenhafte Taskhistorie
  und Taskhead-Rücksprung tun es jeweils. Die ruhige Guard-CAS liest
  `Hq1`, paginiert, liest `Hq2 = Hq1` und bindet genau `Hq1`: Ein fremder Task
  zwischen Pagination und `Hq2` lässt die CAS für O sowie einen reinen S-Lauf
  verlieren. Ein gleichzeitig gestarteter eigener Submit bleibt dagegen am
  Admission-Mutex **vor** `socket_pending` und Socketwrite blockiert, bis die
  Guard-CAS beendet ist; erst danach registriert ihn der Submitter. Die direkte
  `202` und die Observerpagination ordnen ihn als neuen, nicht in `Hq1`
  eingesaugten Suffix zu. Ein
  alter Observerpoll wird zusätzlich nach seinem Guardread angehalten; eine
  neuere Guard-CAS samt Register-GC läuft durch, danach verwirft der alte Poll
  sich aufgrund des vollständig subsumierenden Checkpoints ohne falsches Close.
  Ein zweiter Racefall registriert nach dieser Guard-CAS einen legitimen neuen
  Task: Der alte Poll sieht dessen höheren Head, rebasiert auf den nur teilweise
  subsumierenden Checkpoint und klassifiziert ausschliesslich den neuen Suffix,
  statt wegen der bereits gelöschten Präfixrecords falsch zu invalidieren. Das
  ursprüngliche 1-s-Budget wird dabei nicht neu gestartet. Eine lange
  nie global ruhige Writefolge paginiert jeweils nur ab dem monotonen
  `validated_runtime_task_head` und hält das 1-s-Budget bei beschränktem
  Paginationsaufwand ein.
- Im erst **nach** erfülltem `srch0_pair_handoff_ready_v1` erzeugten
  Authority-Mix `trails=S, actors/lists=O` bleibt ein registrierter STATE1-
  Trailtask während `processing` und nach owner-spezifischem Ack observergrün.
  Sein Runtimehead ist bei vollständiger Registerklassifikation die zulässige
  Live-Readiness-Alternative zum noch alten persistenten Head. Sein reiner
  S-Erfolg löst an der nächsten global ruhigen Grenze eine eigenständige Guard-
  CAS aus; der folgende Neustart vollprojiziert `O` nicht. Ein zwischen
  bekannten O-/S-Task-UIDs eingeschobener fremder Task invalidiert dagegen den
  Guard. Mit aktiver `lists-legacy-v0` weist derselbe versuchte Trail-Handoff
  vor Head-, State- und Enginewirkung `search_pair_migration_required` zurück.
  Nach terminalem Zwei-Paar-Cutover, aber vor `active.json` beziehungsweise vor
  Baseline-Cleanup, scheitert er stattdessen
  `search_pair_activation_evidence_required` beziehungsweise
  `search_pair_cleanup_required`; erst danach ist der Partial-Handoff zulässig.
  Nach diesem gültigen Trail-Handoff setzt ein normaler Offline-Listenfehler
  `lists` auf `ready -> recovering`; ein Crash und `resume` reparieren die Liste
  trotz `trails=S`, weil das monotone `srch0_pair_activated_v1` wahr bleibt und
  der Zustand nie als Authoritykonflikt fehlklassifiziert wird.
- Zwei konkurrierende Writes derselben UID und ein Multi-UID-Fan-out beweisen
  Epochreihenfolge, Countdelta `+1/0/-1` aus den Vorher-/Nachher-Keymengen und
  dass kein späterer Taskerfolg eine ältere Lücke überdeckt. Ein Rollback der
  SQLite-Fachtransaktion verändert keine Epoche; Epochüberlauf verhindert den
  Fachcommit. Für alle neuen mehrschrittigen Eintritte gilt derselbe Grenztest:
  bei `M-3` zulässig, bei `M-2`, `M-1` und `M` vollständig ohne Fach-, State-
  oder Engineeffekt mit `search_state_revision_exhausted` abgewiesen. Bei
  Multi-UID genügt eine knappe Zeile für den All-or-none-Abbruch. Bereits
  durable ausgelöste Pfade dürfen ihre zwei reservierten Fortsetzungen bis `M`
  beenden; der genau ein CAS umfassende Trail-Handoff ist separat bei
  `trails=M-1` zulässig, auch wenn eine read-only in `O` verbleibende UID bereits
  `M` besitzt, und endet bei `trails=M` bereits im Preflight ohne Head-, State- oder
  Engineeffekt mit `search_state_revision_exhausted`. Ein mit `M-1` geplanter,
  vor Preflight aber auf Current `M` fortgeschrittener Handoff baut ebenfalls
  weder Prepared-State noch Generation auf. Die
  Koaleszierungsgrenze startet zusätzlich bei `M-4`: Zwei Writes
  erreichen `M-2`, ein dritter wird abgewiesen; ohne Zwischenattestierung führt
  Gesamterfolg mit einer CAS nach `M-1`, Fehler dagegen über die kombinierte
  `task_failed|recovering`-CAS nach `M-1` und terminales `ready` nach `M`.
  `manual --full` auf sonst kurzpfadfähigem Stand wird bei `M-2..M` vor Fetch
  abgewiesen; ein bei `M-3` zugelassener Operator-/Driftpfad darf bis `M`
  abschliessen. Bei ausschliesslich invalidem globalem Guard und bereits exakt
  gültigen UID-Belegen gelingt der grüne All-O-Vollvergleich bei UID-Revision
  `M-2`, `M-1` und `M` mit einer reinen Guard-CAS und null UID-Writes; sobald
  auch nur eine UID-Attestierung, ein Grund oder Zähler mutiert werden müsste,
  bleibt die ganze Menge bei fehlendem Headroom fail-closed. Zählerarithmetik
  prüft `0 + (-1)` und `M + 1` als Fehler vor CAS sowie `0`, `M` und zulässige
  `+1/0/-1`-Grenzen ohne Wrap oder Sättigung.
- Actor-/User-, Kategorie-, Tag-, Trail→Listen-, Share-/Like-, Delete-,
  Cascade- und Relationswechsel-Fixtures prüfen dieselbe Vorher-/Nachher-
  Abhängigkeitshülle gegen Vollfingerprint, Epoch-Fan-out und inkrementelle
  Dokumentmenge. Admin-, Plugin-, Federation-, interne und heutige
  `UnsafeWithoutHooks`-Pfade verwenden den Koordinator oder werden vor dem
  Commit abgewiesen. Ein Trap-Transport belegt für alle föderierten
  Listenfixtures bei Fingerprint, Vollvergleich, Reindex, Recovery und
  inkrementellem Update jeweils exakt null externe Requests. Der normale Remote-
  Stub mit leerer Relation und `needs_full_sync = true` liefert Trailzahl und
  vier Summen `0`, selbst wenn der Fake-Origin abweichende Nichtnullwerte
  anbieten würde. Eine mit zwei lokalen Trails vollständig materialisierte
  Liste liefert deren exakte Summen; der Stub→materialisiert-Commit invalidiert
  `lists` und projiziert diese Werte nach. Nur ein eigener Dangling-Relation-
  Fall endet ohne Dial mit `source_projection_incomplete`.
- Die Mixed-Authority-Matrix aktiviert zuerst unter Offline-Authority terminal
  beide SRCH0-Zielprofile, übergibt danach `trails` an STATE1 und lässt
  `actors/lists` offline. Actor-Rename/-Delete sowie Trailmetrikänderung
  beweisen je einen atomaren Fachcommit mit STATE1-Change/Dirty für `trails`
  und Applied-Inkrement nur für die betroffenen Offline-UIDs; Rollback liefert
  auf beiden Seiten null Effekt. Beide Rennordnungen zwischen Fachcommand und
  Handoff-CAS werden injiziert und ergeben weder Doppelzustellung noch Lücke.
  Ein weiterer Crash nach dynamischem `prepared`, aber vor dem Handoff-CAS,
  verliert die Locks; ein danach committed Offline-Trailwrite zwingt Resume
  zurück zum statischen Preflight und zu Gleichheitsbeleg oder Neubau der
  STATE1-Generation, statt nur einen frischen Taskhead an alte Bytes zu binden.
  Ein Crash direkt nach dem gemischten Commit lässt beide durablen
  Arbeitsanteile recoverbar. Ändert STATE1 danach Dokumentzahl und Epoch seines
  übergebenen Trails, ignoriert der gemischte Neustart die legitim veralteten
  eingefrorenen Offline-Zähler/-Epochen und mutiert dessen Record nicht.
- `manual` mit genau einem absichtlich gedrifteten Dokument öffnet nach
  beendetem Vollvergleich allgemeine Writes und Federation-Inbox/-Outbox,
  während Search gesperrt bleibt und ab dem
  `search_manual_intervention`-Phasenwechsel sämtliche Engineaufrufe pausieren.
  Im Authority-Mix erhöhen Folgewrites für Offline-UIDs nur Applied und
  persistieren für STATE1-UIDs Change/Dirty; beide Engine-Lieferpfade bleiben
  pausiert. Sie ziehen die ältere Offline-Lücke nicht nach. Ein eigener
  Crashpunkt unmittelbar nach Phasenöffnung und
  vor jedem Folgewrit beweist, dass ein nichtleerer Vollvergleichsgrund durable
  ist und der Neustart den Epochen-Kurzpfad trotz gleicher Epochen und
  Dokumentzahl nicht nimmt. Im expliziten `--full`-Fall ist dieser Grund
  weiterhin `operator`, nicht `document_drift`. Ein `--full`-Lauf crasht
  zusätzlich nach dem atomaren `operator`-
  Vorab-Latch, während des Dokumentfetches und nach dem Vergleich vor der
  Driftentscheidung; jeder Neustart ohne `--full` vollvergleicht wegen des
  durablen Grunds erneut. Ein zweiter Crashpunkt hält mindestens ein STATE1-Ownergate rot,
  crasht nach vollständig gebundener Driftdiagnose und erfolgreichem Grund-
  Commit, aber vor jeder Phasenöffnung, und beweist beim Neustart ohne `--full`
  denselben verpflichtenden Vollvergleich bei weiterhin geschlossenen Writes.
  Ein Zwei-UID-Fall crasht zwischen hypothetischen Row-Writes des Grund-Commits
  und beweist vollständigen Rollback statt einer teilmarkierten Driftmenge.
  Ein Mixed-Folgewrit erzeugt zusätzlich mindestens eine `S`-Projektion
  und ein `C`; exakt die dokumentierte gestoppte Sequenz aus Offline-Reindex,
  `search-state1 reconcile --durable-only --until-ready` und Neustart bringt
  beide Anteile auf grüne Ownergates und öffnet Search. Crashs nach Auto-Plan,
  O-`recovering`, jedem Offline-Batch, Vollvergleich und terminalem O-`ready`
  vor dem strukturierten Stufenergebnis behalten die plangebundene O-only-
  Semantik; `resume` mutiert niemals `S` und meldet danach weiter
  `state1_drain_required = true`. Ist `O` beim Retry bereits grün, ist die Offline-
  Stufe ein belegter idempotenter No-op, sodass der Wrapper trotzdem den Drain
  erreicht. Crashs während des Drains setzen dieselben persistenten Attempts
  ohne Doppelwirkung fort.
  Unvollständiger Vergleich und die übrigen unsicheren Matrixzustände öffnen
  diese App-only-Phase nicht.
- Zusätzliche ungebundene `*__srch0__<run-id>`-UIDs verändern weder Profilwahl
  noch Readiness; nur ein exakt rungebundener nichtterminaler Marker darf
  Recovery aufnehmen.
- Der unterstützte Restoretest ersetzt das Enginevolume durch einen älteren
  Stand mit identischen Settings und identischer Dokumentzahl, setzt danach
  und vor Start `engine_restore` über `search-index invalidate` und beweist,
  dass der folgende Vollvergleich den Felddrift findet. Nach STATE1-Handoff
  scheitert nur eine explizit auf die übergebene UID zielende Invalidierung;
  die implizite Menge überspringt sie read-only.
- Ein getrennter PocketBase-Restoretest spielt bei unverändertem Enginezähler
  einen älteren Quellstand ein, setzt danach und vor Start `source_restore` und
  beweist den verpflichtenden Vollvergleich aller Offline-UIDs; jeder Versuch
  mit `--uid` scheitert atomar mit `search_invalidation_scope_invalid`. Ein Restore ohne den
  unterstützten Invalidierungsschritt erhält ausdrücklich keine
  Integritätszusage. Ein Mixed-Authority-Restoretest schliesst zusätzlich
  Offline-Invalidierung und STATE1-Restoregate vollständig ab, bevor der
  DB-Prozess wieder startet.
- `status --orphans --full` plus `cleanup-orphan` kann einen ungebundenen
  reservierten Index nur mit bytegleichen UID-/Settings-/Dokumentfingerprints
  entfernen; logische, geroutete, run- oder terminalgebundene UIDs sowie jede
  Fingerprintänderung werden vor dem Delete abgewiesen.

Die Marke **produktiv aktiviert** verlangt zusätzlich:

- Alle `activation_gate: srch0_live`-Fälle sind im gebundenen CI-/Staging-
  `fixture-verification.json` am freizugebenden Artefakt und Profil über den
  realen SvelteKit-Pfad grün; kein verbotener Treffer beeinflusst Hits, `total`,
  Reihenfolge oder DTO. In Produktion findet kein authentifizierter ACL-Smoke
  statt.
- SRCH-COMP akzeptiert keine freien Indexnamen, Filter, Sorts oder
  Retrievalfelder. `difficulty_known` und `likes` werden über die
  privilegierte Vergleichsfeldliste geprüft, erscheinen aber nicht in
  Search-Hits oder öffentlichen DTOs.
- SEC-VIS-0 bestätigt die produktive Engine-Netzgrenze und Tokenrotation; ein
  vor der Rotation ausgestellter Token scheitert im datenunabhängigen
  Credential-Negativtest. Ziel- und Rollback-Webbuild tragen die erhöhte
  Cookie-Epoche, und der alte 24-Stunden-Cookie-Fall besteht Refresh und
  Einmal-Retry in CI/Staging.
- Root-Compose, `setup.sh`, Quickstart und manuelle Docker-Dokumentation besitzen
  einen von `SEC-VIS-0` verantworteten Produktionspfad ohne Engine-Hostport;
  bestehende Installationen erhalten eine konkrete Migrationsanweisung.
- Plan, idempotenter Kandidatenaufbau, Vollvergleich, Swap, Wartungsprüfung,
  Aktivierung und Rückswap bei fehlgeschlagener Prüfung wurden mit realer
  Engine ausgeführt.
- Crashs nach Create, Settings, jedem Dokumentbatch, Kandidatenbeleg, exakt
  zwischen dem Multi-Row-`cutover`-CAS und `swap-intent.json`, nach
  Vorwärtsintent, Swap-Submit, erfolgreichem Swap, Reverse-Intent,
  Rückswap, Ziel-State-CAS vor `active.json` und Baseline-State-CAS vor
  `abort.json` führen exakt zu den Zuständen der
  Wiederaufnahmematrix und nie zu einem zweiten ungeprüften Swap. Der
  Swap-Automat injiziert symmetrisch für beide Richtungen Crashs nach Attempt-
  `fsync`/vor Socketwrite, nach Serverannahme/vor Task-ID, nach Task-ID/vor
  Decision-`fsync` und nach Swapwirkung/vor Beobachtung sowie eine verzögert
  sichtbar werdende angenommene Task. Eine fehlende Admission wird nie aus der
  Taskliste nachgetragen; `not_admitted` entsteht nur vor Socketwrite oder aus
  typisierter synchroner Non-Acceptance. Nach Restart wird auch eine durable
  Admission nicht taskbasiert weiterverfolgt.
  Erst eine durable `not_admitted`-/`failed_no_effect`-Decision erlaubt genau
  einen digestverketteten Folgeattempt. Prozess-/Scope-/Sessionverlust bei
  Pre-Abbildung und leerer Taskliste erzeugt null weitere Swaprequests;
  derselbe Verlust bei exaktem Post-State darf nur nach frischem
  `swap_effect_proof_v1` terminal `applied_verified` werden; dies wird einmal
  ohne Admission und einmal mit durabler Admission, aber ohne taskbasierten
  Kontinuitätsbeleg
  ausgeführt;
  mehrere, unbekannte oder unbeweisbar effektfreie Tasks bleiben fail-closed.
  Lücke, Fork, falscher Vorgängerdigest, unzulässige Admission-/Decision-
  Kombination und ein Record nach Richtungsterminalität ergeben jeweils
  `swap_audit_conflict` mit null Engine- oder State-Mutation. Ein Reverse-Intent
  schliesst `resume --activate` dauerhaft aus. Ein Pre-Swap-Abort bei bereits
  beobachteter Post- oder Mischabbildung ergibt `swap_audit_lost` und null
  State-Mutation.
- Ein passender Build mit derselben `search_contract_revision` bleibt ohne
  Reattest ready; eine höhere oder unbekannte Revision bleibt fail-closed.
- Vollständiger Meilisearch-Volumenverlust bei committed PocketBase-
  ableitbaren Zielprofilen wird vor Search-Readiness und Writes aus PocketBase
  repariert, während `GET /health` erreichbar bleibt. Eine noch aktive
  Legacylistenbaseline belegt stattdessen den geforderten Backupfehlerpfad. Verlust
  der Auditdateien bei intaktem Zielindex verursacht keine 503-Schleife und ein
  neuer Vollvergleich kann Audit neu erzeugen.
- Der Produktionsstart leert keinen Index asynchron, setzt Settings nicht
  still nebenläufig und meldet einen unbekannten Mischzustand statt ihn zu
  überschreiben.

Negativtests enthalten mindestens: Sourceänderung nach Plan, zu wenig lokaler
Platz einschliesslich fehlendem/ungültigem `--available-bytes`, fremdes
Kandidatendokument, fehlendes internes Vergleichsfeld,
Engine-Task `failed`/`canceled`, unbekannte Taskannahme ohne quieszenten Head,
verpasster Actor-/Kategorie-/Tag-/Trail→Listen-Fan-out,
`source_projection_incomplete`, Epochenüberlauf,
`legacy_list_migration_required`, `legacy_list_baseline_unrecoverable`,
`security_list_profile_required`, `search_pair_migration_required`,
`search_pair_activation_evidence_required`, `search_pair_cleanup_required`,
`search_pair_migration_authority_conflict`, `state1_handoff_uid_unsupported`,
`state1_handoff_binding_invalid`,
`state1_incremental_authority_handoff_unsupported`, unbekanntes Baselineprofil,
mehrdeutige oder nach unbekannter Annahme unbeweisbare Swap-Task,
`swap_audit_conflict`,
Rückswap nach Aktivcommit, zweiter DB-Prozess, unzuverlässiges Dateilock,
öffentlicher Engineport in jeder produktiven Compose-/Setup-/Dokuvariante,
`--mode auto` mit migrationsfähiger Baseline, zufällige Kandidaten-UID-
Kollision, zurückgelassener ungebundener Kandidat, Offline-Mutation nach STATE1-
Handoff, halb gebundene deklarierte Handoffmenge, doppelte oder fehlende
Authoritypartition einer kreuzenden Projektion, Restore ohne dauerhaften
Vollvergleichsgrund im unterstützten Runbook,
unsicherer Manual-Zustand, der Appwrites zu öffnen versucht, Orphan-Cleanup mit
falschem Fingerprint oder bestehender Referenz sowie Start mit inkompatibler
Vertragsrevision.

## Betrieb, Diagnose und Performance

Der normale Such-Hotpath erhält keinen zusätzlichen synchronen PocketBase-
oder Enginezugriff. DB cached Search-Readiness und invalidiert sie nach einem
Enginefehler; Web liest den DB-Zustand beziehungsweise seinen lokalen Guard.
Ein kompatibler App-Deploy ändert diesen Zustand nicht.

Reindex, Quellfingerprint und der bei Bedarf ausgeführte Vollvergleich sind
`O(Records + expandierte Relationen)` und führen keine Federation-, Geocoding-
oder sonstigen externen Requests aus. Der normale Epochen-Kurzpfad ist dagegen
`O(logische UIDs)`: Er liest nur State, Primärschlüssel, Settings und
Enginezähler plus genau einmal den globalen Taskhead, aber weder PocketBase-
Vollprojektionen noch Engine-Dokumente. Der
Vollvergleich ersetzt bei fehlender Attestierung oder Abweichung den heutigen
Delete/Rebuild zunächst durch einen read-only Vergleich und schreibt nur bei
belegtem Drift. Beide Pfade laufen hinter dem bereits offenen
Liveness-Listener; ihre Laufzeit kann deshalb den Compose-Healthcheck nicht in
eine Restartschleife treiben.

Die folgenden Maxima sind ein Release-Regression-Gate und kein SLA für
beliebige Betreiberhardware. Die Referenzzelle ist Linux/amd64 mit getrennten
DB-/Meilisearch-Containern, festen Quoten von zwei dedizierten vCPU und 4 GiB
RAM, lokalem ext4-SSD-Volume, ohne Paralleljob oder externen Netzverkehr. Sie
trägt die versionierte Kennung `srch0-startup-ci-v1`; ihr eingechecktes Manifest
bindet Runner-Image, Kernel, feste Runnerklasse/CPU-Modell und -Governor,
Cgroup-Quoten, lokalen SSD-Gerätetyp sowie den vor dem Lauf bestandenen CPU-/
`fio`-Kalibrationsreport. Ein Lauf auf einer abweichenden oder unter der im
Manifest festgelegten Kalibrationsgrenze liegenden Zelle ist keine
Releaseevidenz. Die Zelle
verwendet 10'001 Trails plus die feste SRCH0-Actor-/Listen-/Kategorie-/Tag-/
Share-/Like-Matrix einschliesslich trailaggregierender Listen. Engine,
DB-/Engine-Images, Profile und Fixture werden im Bericht digestgebunden; die
Engine ist vor jedem Sample healthy und nur DB startet kalt. Vor jedem Sample
werden bytegleiche PocketBase- und Engine-Ausgangssnapshots wiederhergestellt
und ihre Digests geprüft; erst danach wird der jeweilige Drift-/Verlustfall
injiziert. Kein Sample übernimmt State oder Tasks seines Vorgängers. Das
Manifest bindet genau ein Cache-Regime pro Engineversion, einschliesslich
Prozessneustarts und einer gegebenenfalls festen Vorwärmsequenz; dieselbe
Sequenz gilt für alle zehn Samples und darf innerhalb einer Releaseevidenz
nicht zwischen warm und kalt wechseln. Für **jeden Startfall und jede
unterstützte Engineversion** laufen zehn Samples, und der Höchstwert statt
Mittel oder p95 entscheidet:

| Startfall | maximales `startup_write_freeze_ms` | zusätzliche strukturelle Bedingung |
| --- | ---: | --- |
| gleicher Applied-/Attested-Epochenstand | 5'000 ms | 0 PocketBase-Vollprojektionsrecords, 0 Engine-Dokumentfetches, genau 1 globaler Taskhead-Read vor Freigabe |
| erzwungener grüner Vollvergleich ohne Drift | 60'000 ms | 0 vom Startup/Ensure bis zum Phasenübergang erzeugte Engine-Mutationen |
| `manual`, genau ein gedriftetes Dokument | 60'000 ms bis `search_manual_intervention` | allgemeiner Sentinelwrite erfolgreich; Search weiter 503; 0 vom Startup/Ensure bis zum Phasenübergang erzeugte Engine-Mutationen |
| `auto`, im Startsnapshot sind alle drei logischen UIDs ausdrücklich in `O` gebunden und ihre Indizes fehlen | 180'000 ms bis grünem Vollvergleich und `search_ready` | alle vom Startup/Ensure bis zum Phasenübergang erzeugten Engine-Tasks terminal erfolgreich |

Unmittelbar nach erfolgreichem `event.Next()` und vor Start der Ensure-
Goroutine erfasst `OnServe` den monotonen Messpunkt `listener_bound_at`. Das
Messfenster beginnt dort und endet beim atomaren `writes_open`- beziehungsweise
`search_manual_intervention`-Übergang. Ein authentifizierter deterministischer
Trail-Sentinelwrite läuft mit `max_in_flight = 1`: Erst nach der vorherigen
Antwort folgt frühestens 25 ms später der nächste Versuch, und nach dem ersten
nicht-503 stoppt der Client. Bis zur Freigabe muss jeder Versuch exakt
`503 search_initializing` liefern; danach muss genau ein Versuch committen und
die erwartete Applied-Epoche genau einmal erhöhen. Dieser Commit, das Ende
seiner erfolgreichen Response und jeder daraus erzeugte Engine-Submit liegen
ausdrücklich in dieser Reihenfolge nach dem Messfenster; nach `search_ready`
wartet der Teardown die koaleszierte Taskkette und deren eine Re-Attestierung
terminal ab und beobachtet dabei null Write-seitige Stats-Aufrufe. Im Manual-
Fall beweist er stattdessen null Engine-Tasks, die
durable Offline-Epochenlücke und gegebenenfalls STATE1-Dirty-Arbeit, beendet
den Prozess und setzt ohne unzulässige Online-Attestierung die Snapshots für
das nächste Sample zurück. DB-Schemamigration und
Engine-Containerstart liegen ausserhalb; alle Startup-/Ensure-Roundtrips,
Taskquieszenzversuche und bis zum Phasenübergang dadurch erzeugten Engine-Tasks liegen
innerhalb des Fensters. Jedes
einzelne überschrittene Budget oder Timeout ist Release-Fail. Der Bericht
enthält zusätzlich Peak-Speicher, geladene Enginebytes, temporären
Plattenbedarf, Projektor-Recordzahl, Engine-Fetches und Taskzeiten. Die
Grenzwerte sind von den SRCH0-Systemtests vor **Korpus/Tooling fertig** zu
belegen; sie behaupten keine bereits gemessene Bestandsperformance und erweitern
nicht die unten separat benannte Vierer-Evidenz vor `accepted`.

Separat misst derselbe Lauf den endpointweiten Taskobserver: Im ruhigen Zustand
sind höchstens vier Head-Requests pro Sekunde zulässig, Dokument- und Stats-
Reads bleiben null. Jede injizierte unbekannte Headbewegung, jeder Rücksprung
und jeder Polltimeout muss den lokalen Guard und den durablen Taskguard
innerhalb des festen `task_head_detection_budget = 1'000 ms` schliessen; eine
notwendige Taskpagination zählt vollständig in dieses Budget. Dieses Messgate
ist eine Detektionsschranke ab API-Sichtbarkeit, keine Schranke bis zur Engine-
Registration und keine Null-Stale-Response-Zusage.

`SEARCH_INDEX_STARTUP=manual` ist der ausdrückliche Betreiber-Opt-out für
automatische Engine-Mutationen. Nur der oben definierte vollständig
diagnostizierte Dokumentdrift öffnet die App in
`search_manual_intervention`; Search bleibt bis zum gestoppten Offline-Reindex,
dem anschliessenden durable-only-STATE1-Drain und dem erfolgreichen Neustart
nicht ready. Ohne `S`-/`C`-Rückstand ist der Drain No-op. Der Reindex meldet
ohne private Payloads mindestens:

- gelesene und projizierte Records je Index;
- bekannte Difficultywerte und Unknown-Anzahl;
- Batchanzahl, kanonische Payloadbytes, terminale Taskausgänge und Durchsatz;
- Quell-, Settings- und Dokumentfingerprint;
- Kandidaten- und aktive UIDs sowie beobachtete Swapabbildung; und
- CLI-Stufe und stabilen Diagnosecode bei Fehlern.

Trailnamen, GPX/Polyline, Share-/Like-Actor-IDs, Engine-Key und private
Dokumentbytes erscheinen weder in Belegen noch Logs. Das Defaultbatch von
höchstens 500 Dokumenten/8 MiB begrenzt Speicher und Taskzahl; die reale
Laufzeit und Plattenreserve werden im Systemtest für beide Ausgangsprofile
gemessen. M1 darf für spätere Engineprofile nur über einen versionierten,
digestgebundenen neuen Evidenzstand andere Grenzen qualifizieren; die hier
definierten SRCH0-Profile werden nicht still umkalibriert.

## Festgelegte Entscheidungen und verbleibende Verifikation

Für SRCH0 sind folgende Architekturentscheidungen geschlossen:

- lokaler Offline-One-shot statt vorgezogener verteilter Control Plane;
- Kandidatensatz plus atomarer Swap für vorhandene Baselines, direkter Vollaufbau
  für fehlende abgeleitete Indizes;
- wiederholbare Buildstages und Intents nur für Vorwärts-/Rückswap;
- Runtimekompatibilität über `search_contract_revision`, nicht Build-Digest;
- Auditdateien sind kein Readiness-System-of-Record;
- Liveness-Listener vor der potenziell langen Startprüfung; während des
  Vergleichs bleibt die App schreibgesperrt, danach kann `manual` bei reinem
  Dokumentdrift Appwrites/Inbox ohne Search oder Engineprojektion öffnen;
- normaler Startup-Kurzpfad über monotone Applied-/Attested-Epochen,
  persistierten Enginezähler, Engineprofil und globalen Taskhead-Guard, mit
  begrenztem Taskquieszenz-Retry, laufendem Detect-and-Retry-Observer und
  verpflichtendem Vollvergleich bei jeder Lücke oder Abweichung;
- föderierte Listenaggregate ausschliesslich aus der lokal materialisierten
  Trailrelation, mit sichtbarer numerischer `0`-Semantik bis zum Full-Sync und ohne
  Origin-Dial;
- `lists-legacy-v0` ausschliesslich als opake Engine-Rückswapbaseline; fehlende
  oder gedriftete Legacybytes verlangen ein passendes Backup, während jeder
  Produktionsplan das SEC-VIS-0-Superset `sec-vis0-list-v1` bindet;
- Post-Response-Delivery mit per UID koaleszierter Epochenkette, einmaliger
  Endattestierung und Countfortschreibung ohne Stats-Read pro Write;
- einheitliche `state_revision <= M-3`-Vorbedingung für jeden neuen
  mehrschrittigen Offlineeintritt;
- STATE1 V1 übernimmt im einzigen initialen Authority-Handoff ausschliesslich
  `trails`; der finale Taskhead und All-O-Vergleichsdigest werden erst nach dem
  Generationsbuild gebunden, `actors`/`lists` bleiben Offline-Authority;
- `auto` repariert oder verifiziert ausschliesslich; semantische Migration
  verlangt explizites Ziel, Bestätigung und Kapazitätswert;
- Für jede PocketBase-ableitbare UID in `O` verwenden Initialisierung,
  Bestand und Engineverlust denselben Offline-Rebuildcode; die opake
  Legacylistenbaseline bleibt die ausdrückliche Backupausnahme und `S` seinem
  STATE1-Owner vorbehalten; und
- der Mutationskey bleibt bis zur Aktiv-/Abbruchentscheidung verfügbar.

SRCH0 ist nach Schliessung der fachlichen und strukturellen Entscheidungen
`reviewable`. Vor `accepted`, Implementierungsfreigabe und Liveaktivierung
bleiben genau vier kleine CI-Evidenzprototypen: (1) der gepinnte Go-Client sendet die explizite Fetch-
Feldliste unverändert im POST-Body, (2) rohe `POST /documents/fetch`-Aufrufe
besitzen die beschriebene Literal- und Hidden-Field-Semantik, (3)
`GET /stats.databaseSize` liefert den gebundenen Kapazitätswert und (4)
atomarer Zwei-Paar-Swap für `lists` und `trails`, terminale Taskdetails,
verzögerte Registration vor und nach `H2`, persistenter Taskhead-Guard,
Runtime-Observer samt 1-s-Detektionsbudget, der begrenzte
`task_quiescence_retry_v1`, `swap_effect_proof_v1` und die getrennte Exactly-
once-Attempt-/direkte-Admission-/Decision-Kette verhalten sich wie beschrieben.
Die
beiden Fetch-Semantiken sind auf 1.53.1 bereits belegt; reale offene
Versionsläufe bleiben 1.11.3 und 1.36.0. Scheitert eine Version, wird deren
konkrete Adapterimplementierung angepasst oder das Profil durch M1 als nicht
unterstützt klassifiziert; Live-Upsert ist kein Fallback.

## Review- und Beitragsschnitte

1. Fixture-Schema, vollständige Consumer-/State-Matrix und generierte
   Bestandsdokumentation.
2. Reiner Legacy-Evaluator/Compiler, Radiusfix und Browserzustandstests.
3. Full-/Patch-Projektionen, Unknown-DTO/UI und Mutationskonvergenz.
4. Gemeinsame Profilmanifeste, reale Engine-/API-Tests und
   `search_contract_revision`-Readiness.
5. Reindex-/Ensure-CLI, Compose-Runbook, Crashmatrix, Cutover und Recovery nach
   vollständigem Engineverlust für PocketBase-ableitbare UIDs in `O`, opaker
   Legacylisten-Backupfehlerpfad plus owner-exklusive STATE1-Recovery für `S`.

Jeder Schnitt ist intern reviewbar. Eine sichtbare Aktivierung erfolgt erst,
wenn alle fünf Schnitte sowie SRCH-COMP- und SEC-VIS-0-Livegates erfüllt sind.

## Entscheidungsprotokoll

| Datum | Entscheidung | Begründung |
| --- | --- | --- |
| 2026-08-30 | Commit `e6db729c` ist die SRCH0-Ausgangsrevision. | Alle Baselinefixtures und Codeanker beziehen sich auf denselben reproduzierbaren Stand. |
| 2026-08-30 | Ein sprachneutraler JSON-Korpus speist alle Testschichten. | Getrennte Go-, TypeScript- und Browsergoldens würden unbemerkt divergieren. |
| 2026-08-30 | Unknown-Difficulty wird explizit als `null` plus internes `difficulty_known` projiziert. | Partielle Updates müssen einen früheren falschen Wert löschen und Unknown bei beiden Sortierrichtungen zuletzt halten. |
| 2026-08-30 | Full Projection und drei Patchformen sind getrennte Verträge. | Die heutige `includeShares`-Verzweigung ist absichtlich partiell; Vollreindex und Mutationsfolge müssen trotzdem konvergieren. |
| 2026-08-30 | Der Bestandsreindex verwendet einen vollständig verglichenen Kandidatensatz und einen atomaren Multi-Pair-Swap. | In-place-Upsert oder getrennte Trail-/Listenaktivierung mischt Profile, lässt Orphans zu und zerstört die unmittelbare Rückfallbaseline. |
| 2026-08-30 | Der Produktionslauf ist eine lokale Offline-Operation mit heutiger Engine-Authentisierung. | Credential-Controller, Observer-PKI, Clock- und Capacitydienste existieren im Repository nicht und sind keine SRCH0-Nebenaufgabe. |
| 2026-08-30 | Nur Vorwärts- und Rückswap erhalten durable logische Intents samt No-Replace-Submit-/Admission-/Decision-Kette. | Create, Settings und Add-or-Replace auf den runlokalen Kandidaten-UIDs sind wiederholbar; ein Swap würde bei blinder Wiederholung seine Richtung umkehren. |
| 2026-08-30 | Fresh Install, Legacybestand und Engineverlust verwenden je PocketBase-ableitbarer Offline-UID denselben Recoverybuilder; STATE1-UIDs bleiben owner-exklusiv, `lists-legacy-v0` ist die opake Rollbackausnahme. | Zielprofile sind abgeleitet und reparierbar. Die historischen Remoteaggregate der Legacyliste sind es nicht; ihr Verlust verlangt ein passendes Enginebackup statt erfundener Rekonstruktion oder Authority-Rückdrehung. |
| 2026-08-30 | Readiness bindet `search_contract_revision` und Profile statt vollständiger Buildbytes. | Kompatible Deploys und App-Rollbacks dürfen keinen externen Reattest und keine vermeidbare Suchdowntime verlangen. |
| 2026-08-30 | Auditbelege liegen ausserhalb der Runtime-Autorisierung. | Ihr Verlust darf bei vollständig vergleichbarem Enginezustand keine 503-Schleife erzeugen. |
| 2026-08-31 | Der PocketBase-Vollvergleich gibt `POST /documents/fetch` eine explizite Profilfeldliste; nur der opake Legacylistensnapshot lässt `fields` bewusst aus. | Im POST-Body ist `fields: ["*"]` ein wörtlicher Feldselektor statt der GET-Wildcard; das leere Objekt ist auf 1.53.1 beobachtet, während ausgelassenes `fields` alle Bytes für den lokalen Snapshot liefert. Die rohen Laufzeittests auf 1.11.3/1.36.0 bleiben offen. |
| 2026-08-30 | Der Mutationskey bleibt bis zur Terminalentscheidung verfügbar. | Eine fehlgeschlagene Aktivprüfung muss physisch zurücktauschen können; Widerruf vor dem Commit erzeugt einen nicht reparierbaren Zwischenzustand. |
| 2026-08-30 | Datum, Nullkoordinate, Karten-Descent-Default und raw Storage-Sortkeys werden erfasst, aber SRCH-COMP zugeordnet. | Sie sind belegte Lücken ausserhalb der drei von SRCH0 ausgelieferten Bestandskorrekturen. |
| 2026-08-31 | Der DB-Listener öffnet vor der Startvalidierung liveness-only; `auto` nutzt eine interne Epochenattestierung. | Der beschränkte Compose-Healthcheck darf keinen O(N)-Vergleich abbrechen; während der stabilitätsbedürftigen Prüfung bleiben Search, Writes und Worker fail-closed. |
| 2026-08-31 | Jeder unterstützte Quell-/Engine-Restore und Engine-Admineingriff setzt nach Restore und vor Start einen dauerhaften Offline-Vollvergleichsgrund. | Ein alter PocketBase- oder Enginebestand mit gleichen Epochen, Dokumentzahlen und Settings wäre sonst vom Kurzpfad nicht vom verifizierten Paar zu unterscheiden. |
| 2026-08-31 | `run --mode auto` migriert nie; ein Cutover verlangt `--mode migrate`, exaktes Ziel, revisionsgebundene Bestätigung und gemessene freie Bytes. | Recovery und der einmalige Semantikwechsel dürfen nicht dieselbe unbeaufsichtigte Befehlszeile sein; der DB-Scratch-Container kann das Engine-Dateisystem nicht selbst messen. |
| 2026-08-31 | Runlokale Kandidaten liegen ausserhalb des logischen Profilsets. | Ein bewusst aufbewahrter Rückfall- oder Cleanup-Kandidat darf den nächsten normalen Start nicht als Profilkonflikt blockieren oder still adoptiert werden. |
| 2026-08-31 | Die Tokenrotation erhöht im selben Release die Cookie-Epoche; Ziel- und Rollback-Artefakt beherrschen Invalidierung und genau einen 403-Refresh. | Sonst verwenden wiederkehrende Browser bis zu 24 Stunden einen vom neuen Parent-Key abgewiesenen Tenant-Token. |
| 2026-08-31 | Die Produktion erhält vor Ingress keinen authentifizierten ACL-Smoke. | Die datenunabhängige ACL-/Proxy-/DTO-Matrix läuft in CI/Staging am gebundenen Releaseartefakt und Profil; Produktion prüft offline nur State, UID, Settings und Dokumente, während SEC-VIS-0 Netzgrenze und Credentialrotation separat gatet. |
| 2026-08-31 | `search_index_state` plus genau ein endpointweiter `search_engine_task_guard` bilden den kleinen persistenten Offline-Crash-Guard; die UID-Zeile besitzt eine irreversible Authority-Übergabe an STATE1. | UID-Epochen belegen die Projektionsfolge, der globale Head-Checkpoint verhindert das Vergessen später sichtbarer Alttasks. Nach dem ersten STATE1-Pointer darf das Offlinewerkzeug weder dieselbe logische UID noch deren State mutieren; kreuzende Abhängigkeiten werden im Fachcommit nach Authority partitioniert. |
| 2026-08-31 | `SEC-VIS-0` besitzt die produktiven Compose-/Setup-/Installationsdokumente als Deliverable. | Root-Compose, Quickstart und manuelle Installation veröffentlichen heute `7700`; ohne benannten Owner würde der korrekte Produktionspreflight Bestandsinstallationen nur blockieren. |
| 2026-08-31 | Applied-/Attested-Projektionsepochen ersetzen den Vollprojektions-Kurzpfad als normalen Neustartpfad. | Der Fachcommit erhöht Applied atomar, nur die terminal erfolgreiche und streng geordnete Engine-Taskmenge zieht Attested nach; gesunde benutzte Instanzen starten dadurch ohne korpusabhängigen Vergleich. |
| 2026-08-31 | `manual` öffnet nach vollständig diagnostiziertem reinem Dokumentdrift die App ohne Search- oder Engineprojektion. | Die Quelle muss nur während des Vergleichs stabil bleiben; ein Betreiber-Opt-out darf Upload, Adminwrites und Federation-Inbox nicht unbegrenzt stilllegen. |
| 2026-08-31 | Fingerprint, Epoch-Fan-out und inkrementelle Projektion verwenden dieselbe versionierte Adapter-Abhängigkeitshülle. | Denormalisierte Actor-/Kategorie-/Tagfelder und Trailaggregate in Listen dürfen nicht durch collection-lokale Hooks unterinvalidiert werden. |
| 2026-08-31 | Ein gültiger Trail-Partial-Handoff partitioniert jede kreuzende Projektionshülle atomar in Offline-Epochen und STATE1-Change/Dirty. | Der übergebene Trailrecord neben Offline-`actors`/`lists` ist erst nach terminalem SRCH0-Zwei-Paar-Cutover zulässig. Sein eingefrorener Record darf weder mitinkrementiert noch seine STATE1-Invalidierung ausgelassen werden. |
| 2026-08-31 | Das 10'001er-Startup-Gate besitzt harte Maximalzeiten von 5/60/60/180 Sekunden für Epochenpfad, Vollvergleich, Manual-Drift und Total-Recovery. | Messwerte ohne Referenzzelle und Pass/Fail-Schranke verhindern keine skalierende Write-Sperre. |
| 2026-08-31 | Der Manual-Repair eines Authority-Mix führt bei gestopptem Serveprozess zuerst den Offline-Reindex und danach einen durable-only-STATE1-Drain aus. | Während der App-only-Phase bleiben beide Lieferpfade absichtlich pausiert; ohne ausdrücklichen Drain könnten bereits committed `S`-/`C`-Changes den folgenden Readiness-Guard dauerhaft blockieren. |
| 2026-08-31 | Jeder neue mehrschrittige Offlinepfad verwendet dieselbe Eintrittsgrenze `state_revision <= 2^53-4`. | Eine einheitliche Drei-Schritt-Reserve hält Abschluss und Fehler-Recovery darstellbar, ohne jeden Übergang an eine nie praxisrelevante Budgettabelle zu koppeln. |
| 2026-08-31 | Swap-Decisions und Terminalbelege besitzen zustandsabhängige Admission-/Proofpflichten und werten nur die höchste lückenlose Digestkette aus. | Die nicht idempotente Swapoperation darf bei Auditfork, verlorener Session oder unklarer Annahme weder eine ähnliche Task adoptieren noch einen zweiten Request autorisieren. |
| 2026-08-31 | Föderierte Listenaggregate stammen ausschliesslich aus der lokal materialisierten `lists.trails`-Relation. | Origin-Aggregate werden nicht persistiert; numerische `0`-Werte bis zum Full-Sync sind deshalb die einzige geschlossene netzfreie Projektion und werden als dritte sichtbare SRCH0-Korrektur ausgeliefert. |
| 2026-08-31 | Offline-Submit, Task-Await und Attestierung laufen nach der Antwort in einer pro UID koaleszierten Delivery-Kette. | Der Requestpfad darf keine Engine-Rundreise erben; ein terminal erfolgreicher lückenloser Epochenbereich benötigt nur eine Attestierungs-CAS und keinen Stats-Read pro Write. |
| 2026-08-31 | Der Vollvergleich verwendet begrenzte Taskhead-Validierungsversuche, persistenten globalen Head-Checkpoint und Runtime-Observer statt einer behaupteten Admission-Linearisierung. | Die öffentliche Engine-API bietet keine solche Barriere; ein nicht binnen 1 s vollständig als eigener Suffix klassifizierter Head verwirft den Vergleich beziehungsweise invalidiert Readiness, während erwartete Tasks bis 120 s laufen dürfen und wiederholte Unruhe mit `search_task_quiescence_failed` retrybar fail-closed endet. Ohne Enginefence besteht keine Nullfenster-Zusage vor Observererkennung. |
| 2026-09-01 | `lists-legacy-v0` ist eine opake Engine-Rückswapbaseline; der produktive Zielkandidat bindet zwingend `sec-vis0-list-v1`. | Die historischen Remoteaggregate sind nicht aus PocketBase rekonstruierbar. Ein Snapshot erhält exakten Rückswap ohne Origin-Dial, während fehlende Bytes ein Backup statt stiller Einwegmigration verlangen. |
| 2026-09-01 | Jeder endpointweite Submit besitzt 5 s Annahme-, jeder erwartete Task 120 s Terminalfrist; ruhige Guardfortschreibung verwendet den stabilen `Hq1/Hq2`-Fence. | Hängende Requests oder Tasks dürfen Search nicht unbegrenzt stale-grün halten, und ein Task zwischen Pagination und CAS darf nicht ungeprüft in den Checkpoint gelangen. |
| 2026-09-01 | Die `M-3`-Reserve gilt je neuem Eintritt; nach dem jeweils letzten zugelassenen Commit folgen höchstens zwei UID-CAS. Ein global-only Reattest mutiert keine UID. | Koaleszierte Ketten dürfen viele Applied-Commits enthalten, ohne einer falschen Drei-Inkrement-Gesamtgrenze zu unterliegen; ein gesättigter Diagnosezähler darf die reine Guardreparatur nicht blockieren. |
| 2026-09-01 | STATE1 V1 darf ausschliesslich `trails` und dies erst nach dem terminalen SRCH0-Zwei-Paar-Cutover übernehmen; `lists` bleibt Offline-Authority. | STATE1 V1 besitzt nur Trail-Entity, Trail-Queryfamilien und Trail-Membercoverage; ein Listen-Handoff wäre unbeweisbar und ein vorzeitiger Trail-Handoff würde die zwingende Offline-Migration der opaken Legacylistenbaseline unautorisierbar machen. |
| 2026-09-01 | Paaraktivierung ist ein monotones historisches Prädikat; Handoff verlangt separat Terminalbeleg und vollständigen Cleanup der runlokalen Paar-Baselines. | Ein späterer Offline-Recovery darf den erfolgten Cutover nicht in einen falschen Authoritykonflikt zurückverwandeln, und nach irreversibler Übergabe darf kein veralteter Index ohne Löschowner verbleiben. |
| 2026-09-01 | Die einzige nichtleere Offline-Handoffmenge des ersten STATE1-Bootstraps ist `["trails"]` und endgültig. | Der V1-Automat besitzt weder Listenmember noch einen inkrementellen Authority-Handoff auf einem bereits aktiven Head; `actors`/`lists` bleiben deshalb offline, statt unerreichbare Coverage oder einen zweiten Bootstrap zu behaupten. |
| 2026-09-01 | Der Trail-Handoff trennt statischen Revisions-/Source-Preflight, Generationsbuild und dynamische Precommit-Bindung des finalen Heads. | Buildtasks dürfen den vorab gelesenen Head verändern; erst ihr terminaler Erfolg erlaubt den digestgebundenen read-only All-O-Vollvergleich und die Guard-CAS. Prozess-/Lockverlust kehrt zum statischen Preflight und nötigenfalls Neubau zurück, statt alte Generationbytes nur neu zu attestieren. |
