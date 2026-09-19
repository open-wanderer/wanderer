---
title: SRCH-COMP — Bestandsadapter
description: Serverseitiger Normalisierer, Compiler, URL-Codec und Legacyadapter auf dem heutigen Backend und Index.
editUrl: false
sidebar:
  order: 3
  badge: Blockiert
spec:
  id: SRCH-COMP
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-foundation
  exposure: cutover
  implementationDependsOn: [SRCH-V1, IDX0]
  releaseGates: [SEC-VIS-0]
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED, FEDERATION-SECURITY-V1]
  lastReviewed: '2026-09-19'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch SRCH-V1 und IDX0 |
| Implementierungsabhängigkeiten | SRCH-V1, IDX0 |
| Releasegate | SEC-VIS-0 für den produktiven gemischten Suchpfad |
| Ergebnis | V1-Vertrag auf heutigem Backend; der Legacycompiler erzeugt den Bestands-Startpunktradius genau einmal |

## Nutzerergebnis und Releaseeinheit

Nach dem Cutover erzeugt der Browser keine freien Meilisearch-Ausdrücke mehr.
Alle heutigen Suchwege durchlaufen einen serverseitigen, allowlisteten V1-
Normalisierer und Compiler. Beim Übersetzen des vorhandenen
Startpunktzustands erhält SRCH-COMP die bereits in SRCH0 geprüfte Semantik:
genau eine `_geoRadius`-Klausel und gültige Nullkoordinaten. Die Migration
erhält Treffer, `total`, Reihenfolge, Seite, Radius und Punktquelle der
korrigierten Ausgangsbasis. Spätere Panel- und Geo-Slices erhalten eine
erprobte API.

## Scope

- `SearchRequestV1` validieren und kanonisieren.
- Normalisierte V1-Filter auf den heutigen Index und seine realen Felder
  kompilieren.
- Bestehende URL-/Store-Zustände über einen expliziten Legacyadapter übersetzen.
- Den Bestands-Startpunktradius pro normalisiertem Auftrag höchstens einmal in
  `_geoRadius` kompilieren und seine unveränderte Trefferwirkung gegen das
  SRCH0-Korpus beweisen.
- Responses auf die erlaubten V1-DTOs begrenzen.
- Alte und neue Eingabe für alle SRCH0-Fixtures gegeneinander testen.
- Alle heutigen serverseitigen Engine-Consumer einschliesslich Einzel-/Multi-
  Search, Actor-, Profil-, Empfehlungs-, Cluster-, Bounding-Box-, Upload-
  Duplikat- und sonstigen Legacyhilfsrouten über einen gemeinsamen Readiness-/
  Tenant-Token-Adapter führen. Nur dieser Adapter darf nach einem Engine-`403`
  den intern gecachten Token invalidieren, einmal neu holen und denselben
  normalisierten Auftrag genau einmal wiederholen. Bei jedem
  `SearchReadinessV1.status = not_ready` muss derselbe Adapter vor Tokenausgabe
  und Enginezugriff mit `search_unavailable` abbrechen.
- Tenant-Token ausschliesslich lazy für einen tatsächlich auszuführenden
  Engineauftrag beziehen. Ein normaler Webrequest ohne Token-Cookie sowie ein
  Upload mit `ignoreDuplicates = true` sind keine Search-Consumer und bleiben
  bei roter Readiness frei von Token- und Enginezugriff.
- Den von IDX0 produzierten Legacy-Readinesszustand konsumieren; SRCH-COMP
  definiert weder einen zweiten Bootstrap noch eine eigene Readiness-Wireform.
- Typisierte Fehler für ungültige Filter, Sortierungen, Radien, Cursor und nicht
  verfügbare Capabilities liefern.

## Nichtziele

- Kein neuer Shadow-Index, kein Federation-Gateway und keine neuen Felder.
- Keine neue Difficulty-Projektion oder Listenaggregatsemantik; diese
  Erweiterungen besitzen mit SRCH2 beziehungsweise SEC-VIS-0 eigene
  Delivery-Owner. Bereits für SRCH0 behobene Fehler bleiben korrigiert.
- Keine Counts oder Histogramme, die der heutige Pfad nicht vertragsgemäß
  berechnen kann.
- Kein stilles Durchreichen generischer Enginefilter.

## Migration und Rollback

Der neue Adapter läuft zunächst gegen Golden Fixtures und optional im
Response-Shadow ohne ausstellbare Antwort. Der Cutover schaltet alle Browser-
und API-Aufrufe gemeinsam auf den V1-Pfad. Rollback darf auf den alten Adapter
zurückschalten, solange kein neuer URL-State veröffentlicht wurde, den dieser
nicht versteht. Nach öffentlicher Ausgabe neuer V1-Zustände bleibt der
Legacyparser als Eingangsadapter erhalten.

## Korrigierte SRCH0-Basis und Adapter-Deltas

SRCH-COMP konsumiert historische Basisfälle zusammen mit den abgenommenen
aktiven Solländerungen und unabhängigen Properties. Radius-, Datums- und
Kartenfehler müssen bereits vor SRCH0-Abnahme behoben sein. Feldauswahl und
Sortier-Robustheit sind gemäss [SRCH0](/develop/specs/trail-search/work-items/search/srch0/)
keine Abnahmeblocker; ihre unten beschriebene Zielsemantik bleibt für den
Adapter vorgesehen:

| Referenz | Erhaltene Semantik beziehungsweise explizite V1-Erweiterung |
| --- | --- |
| `SRCH0-GAP-GEO-001` | genau eine Radiusklausel mit unveränderter fachlicher Wirkung |
| `SRCH0-GAP-DATE-001` | vollständiges inklusives lokales Enddatum; die V1-Übersetzung ergänzt ihre vertraglichen Warnungen |
| `SRCH0-GAP-GEO-002` | Latitude oder Longitude `0` bleibt ein gültiger Radiusanker |
| `SRCH0-GAP-MAP-001` | Karten-Descent-Default verwendet den Descent- statt des Gain-Grenzwerts |
| `SRCH0-GAP-DTO-001` | Retrievalfelder liegen in der vom Engineadapter tatsächlich ausgewerteten Optionsstruktur |
| `SRCH0-GAP-SORT-001` | der geprüfte Legacy-Storagefallback bleibt erhalten; ungültige explizite V1-Aufträge enden typisiert vor dem Enginezugriff |

Neue Transport-, Validierungs- und URL-Semantik wird als eigenes Overlay
gegen die korrigierte SRCH0-Basis beschrieben. Jedes Overlay bindet Case-ID
und Basisdigest und beweist alle nicht betroffenen Treffer-, ACL-, Paging-
und URL-Dimensionen unverändert. Historische Fehlresultate werden dadurch
nicht wieder zu zulässigen Erwartungen.

## Abnahme

- Alle aktiven SRCH0-Erwartungen und unabhängigen Properties einschließlich
  ACL-Kontexten bleiben grün. Historische `known_gap`-Ergebnisse sind kein
  Rückfallziel.
- Der Adapter erzeugt genau eine `_geoRadius`-Klausel und erhält Treffer,
  `total`, Reihenfolge und Seite der korrigierten Legacyausführung.
- Die erhaltenen Korrekturen und zusätzlichen V1-Adapter-Deltas werden
  separat geprüft; kein Delta öffnet freie Engineparameter oder eine
  unzulässige Treffermenge.
- Keine freie Engine-Syntax erreicht den Compiler.
- Normalisierung ist deterministisch und idempotent.
- Kanonische URL, API-Auftrag und ausgeführter Suchauftrag bleiben nach
  Roundtrip gleich.
- Negativtests beweisen, dass alte Direktzugänge den Adapter nicht umgehen.
- Jede oben genannte Consumerfamilie verwendet nach der `SEC-VIS-0`-Parent-Key-
  Rotation den gemeinsamen Adapter. Ein alter 24-Stunden-Cookie wird wegen der
  erhöhten Ziel-/Rollback-Epoche vor Verwendung ersetzt; ein erzwungener erster
  `403` führt zu genau einem erfolgreichen Refresh/Retry, ein zweiter endet ohne
  Schleife.
- Jede Consumerfamilie läuft zusätzlich bei abstraktem Readinesszustand
  `not_ready`: Search-, Token-, Proxy- und Hilfsantworten bleiben fail-closed
  und der Engine-Mock beobachtet exakt null Requests. Ob Fachwrites oder
  Federation-Worker in diesem Zustand laufen, bestimmt ihr jeweiliger
  Ownervertrag und nicht SRCH-COMP.
- Ein Nicht-Suchrequest ohne Token-Cookie und ein Upload mit
  `ignoreDuplicates = true` bleiben bei rotem IDX0-Zustand erfolgreich und
  rufen weder `/search/token` noch Meilisearch auf. Der Empfehlungsendpunkt und
  ein Upload mit aktiver Duplikatprüfung bestehen dagegen denselben
  No-Engine-Guardtest wie die übrigen Search-Consumer.
- Derselbe No-Engine-Test läuft als Integration gegen einen real roten und
  danach grünen IDX0-DB-/Web-Readinessproduzenten; ein nur im Mock bekanntes
  `not_ready` erfüllt die Abnahme nicht allein.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Der erste reale Vertragsverbraucher läuft auf dem heutigen Backend, nicht erst auf G3b. |
| 2026-08-31 | Alle First-Party-Engine-Consumer teilen Readiness, Tokeninvalidierung und höchstens einen 403-Refresh; Einzelrouten dürfen keinen eigenen Retrypfad besitzen. |
| 2026-08-31 | Kein `not_ready`-Zustand öffnet eine versteckte Engine-Nebenroute; jeder First-Party-Consumer scheitert im gemeinsamen Guard vor Token und Engine. |
| 2026-09-01 | SRCH-COMP besitzt die Entfernung der doppelten `_geoRadius`-Klausel als eigenes SRCH0-Delta-Overlay; Difficulty- und Listenprojektion bleiben ausserhalb seines Scopes. |
| 2026-09-01 | IDX0 produziert Bootstrapzustand und Readiness; SRCH-COMP konsumiert den bestehenden Wirevertrag für alle Legacyconsumer. |
| 2026-09-19 | SRCH-COMP erhält die bereits für SRCH0 korrigierte Produktsemantik; seine eigenen Deltas betreffen den V1-Adapter. Die frühere Verschiebung der Bestandsfehler zu SRCH-COMP ist ersetzt. |
