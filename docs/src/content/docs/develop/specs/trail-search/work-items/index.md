---
title: Work-Item-Register
description: Umsetzungsnahe Spezifikationen für die aktuell priorisierten Such-, Panel-, Engine-, Geo- und Aggregationsbausteine.
editUrl: false
sidebar:
  order: 1
  label: Übersicht
  badge: Arbeitsstand
spec:
  id: WORK-ITEMS
  kind: delivery
  status: draft
  lastReviewed: '2026-09-19'
---

Ein Work Item ist der stabile fachliche und technische Scope eines geplanten
Bausteins. Es ist nicht zwingend mit genau einem GitHub-Pull-Request identisch:
Ein kleiner Baustein kann 1:1 umgesetzt werden, ein umfangreicher Baustein wie
G3b darf in mehrere reviewbare PRs zerfallen. Die zugehörigen PRs werden in der
Work-Item-Spezifikation referenziert, ohne deren Akzeptanzvertrag aufzuteilen.

Der [Delivery-Katalog](/develop/specs/trail-search/delivery/) bleibt die einzige Quelle für den
vollständigen Kandidaten- und Abhängigkeitsgraphen. Diese Seiten konkretisieren
nur aktiv betrachtete Ausschnitte. Ein fehlendes Work-Item-Dokument bedeutet
nicht, dass eine im Katalog genannte Abhängigkeit entfällt.

## Statusmodell

| Feld | Bedeutung |
| --- | --- |
| Spec-Status | Reife des Dokuments: `stub`, `draft`, `reviewable`, `accepted` oder `superseded` |
| Delivery-Status | Umsetzungsstand: `candidate`, `blocked`, `ready`, `in-progress`, `validating`, `released` oder `withdrawn` |
| Implementierungsabhängigkeit | Muss für die Implementierung oder Integration vorhanden sein |
| Implementierungsbedingung | Erfüllbare OR-/Providerbedingung, die kein eigener Graphknoten ist |
| Releasegate | Darf parallel bearbeitet werden, muss aber vor der sichtbaren Freigabe erfüllt sein |
| Exposure | Art der Wirkung: intern, Shadow, betrieblich, sichtbar oder Cutover |

## Aktuelle Such- und Panel-Linie

| Work Item | Ziel | Stand |
| --- | --- | --- |
| [SRCH0](/develop/specs/trail-search/work-items/search/srch0/) | fachlich korrekte Regressionsbasis, stabile Case-IDs und unveränderte historische Evidenz | reviewable; Delivery `blocked`: Tests begonnen, Bestandskorrekturen und Startup-Paket noch zu integrieren und gemeinsam grün nachzuweisen |
| [SRCH-V1](/develop/specs/trail-search/work-items/search/srch-v1/) | normativen Suchvertrag implementierbar verankern | wartet auf die abgenommene SRCH0-Regressionsbasis |
| [SRCH-COMP](/develop/specs/trail-search/work-items/search/srch-comp/) | Compiler und Legacyadapter unter Erhalt der korrigierten SRCH0-Semantik | wartet auf SRCH-V1 und IDX0; sichtbare Aktivierung zusätzlich SEC-VIS-0 |
| [SRCH2](/develop/specs/trail-search/work-items/search/srch2/) | Bestandsfelder und Subkategorien sowie explizite Presence-/Unknown-Modellierung auf der korrigierten Basis | wartet auf SRCH-COMP und IDX0 |
| [SRCH4a](/develop/specs/trail-search/work-items/search/srch4a/) | vollständiges Panel ohne Fake-Counts | wartet auf SRCH2; sichtbare Freigabe zusätzlich SEC-VIS-0 |
| [SRCH-SAVED](/develop/specs/trail-search/work-items/search/srch-saved/) | benannte Suchen und Standardsuche für die Trail-Liste | Implementierung nach SRCH-COMP; sichtbare Freigabe nach SRCH4a und SEC-VIS-0; kein Geo-/Count-Gate |

SRCH0 ist auf `feat/srch0` mit Ausgangsrevision `e9b7a8cad` vom
7. September 2026 in Umsetzung. `fix/srch0-findings` bleibt die Sammelreferenz
für einzeln vorzubereitende Produkt-PRs. Als erster Fix ist
`fix/search-radius-filter` (`398b45682`, direkt ab `dev` bei `c73966d6c`)
mit 24 Radiusregressionen lokal geprüft; weder Push noch PR oder Integration
in `dev` beziehungsweise `feat/srch0` sind erfolgt.
Die [SRCH0-Spezifikation](/develop/specs/trail-search/work-items/search/srch0/#aktueller-umsetzungsbezug)
führt die Prüfungsnachweise. `fix/search-index-startup` besitzt noch kein
Implementierungspaket. Diese Arbeitsteilung erlaubt parallele
Implementierung, ersetzt aber keine Abnahme auf einer gemeinsamen
Zielrevision. Bekannte Fehler im verbindlichen Abnahmeumfang
bleiben Merge- und Abnahmeblocker; `knownViolation`, `xfail` oder historische
Fehlerausgaben dürfen die aktive Suite nicht grün machen. SRCH-COMP, SRCH2 und
SEC-VIS-0 übernehmen die korrigierte Basis und ihre jeweiligen Erweiterungen;
die notwendigen Bestandskorrekturen warten nicht auf diese Nachfolger.

`SRCH0-GAP-SORT-001` ist gemäss Entscheidung vom 19. September 2026
**kein SRCH0-Blocker**. Die Bereinigung ungültiger Storagewerte und die
Fallback-Richtung sind als kleine Robustheitsverbesserung zurückgestellt;
vorerst wird dafür kein eigener PR vorbereitet. Diagnosefälle und historische
Evidenz bleiben erhalten. Die vorhandene Suite muss die nicht blockierende
Einordnung noch übernehmen; die [SRCH0-Spezifikation](/develop/specs/trail-search/work-items/search/srch0/)
hält diese Abgrenzung fest. Die Tests gültiger Sortierungen und die übrigen
Abnahmevoraussetzungen gelten weiterhin.

Die Feldauswahl `SRCH0-GAP-DTO-001` ist ebenfalls **kein SRCH0-Blocker**,
wird aber separat auf `fix/search-retrieved-fields` korrigiert, lokal ohne
Push oder PR. Die [SRCH0-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-feldauswahl)
beschreibt den begrenzten Umfang und die noch ausstehende Trennung der
Diagnoseprüfungen.

Die globale Tag-Umbenennung `SRCH0-MUTATION-008` ist ebenfalls **kein
SRCH0-Blocker**. Sie ist ein administrativer Sonderfall, der in der normalen
UI und für normale API-Benutzer nicht verfügbar ist.
`fix/search-tag-metadata` (`122974380`) bleibt als
separater lokaler Fix ab `dev` (`c73966d6c`) vorbereitet, ohne Push, PR oder
Integration. Die [Tag-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-tag-umbenennung)
hält die noch ausstehende Diagnose-/Abnahmetrennung fest; reguläre Tagfilter,
Tagzuordnungen und die übrigen Metadatenbefunde sind nicht ausgenommen.

Auch negative Thumbnailindizes `SRCH0-PROJECTION-021` sind **kein
SRCH0-Blocker**. Der nachgewiesene Absturz tritt mit ungültigen Werten auf,
die normale Fotoauswahl und JSON-API nicht zulassen, andere Schreibpfade
jedoch erlauben. Der separate lokale Fix auf `fix/search-thumbnail-index`
fällt dann auf das erste Foto zurück. Die [Vorschaubild-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-vorschaubild-absicherung)
beschreibt den begrenzten Umfang und die noch ausstehende Diagnose-/Abnahmetrennung.
Andere Projektions-, Sichtbarkeits- und Startup-Prüfungen bleiben verbindlich;
Push, PR und Integration sind noch nicht erfolgt.

SDK-HTTP-Statusweitergabe und Actor-Suchparameter sind ebenfalls **keine
SRCH0-Blocker**. Die unabhängigen lokalen Fixbranches
`fix/search-api-error-status` und `fix/search-actor-parameters` basieren
jeweils direkt auf `dev` (`c73966d6c`), ohne Push, PR oder Integration.
Die [Status-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-api-fehlerstatusweitergabe)
und [Parameter-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-actor-suchparameter)
nehmen nur Fehlerklassifikation, fehlendes `q` und explizite Limits aus.
Authentifizierung, Berechtigungen und die Ablehnung fehlgeschlagener
Engineanfragen bleiben verbindlich. Tests und Korpus bleiben unverändert;
die technische Diagnose-/Abnahmetrennung ist noch nachzuführen.

Upload-Duplikatprüfung und Clusterbegrenzung sind ebenfalls **keine
SRCH0-Blocker**. `fix/upload-duplicate-check` wird als unabhängiger
Importfix direkt ab `dev` (`c73966d6c`) vorbereitet, lokal ohne Push, PR
oder Integration. SRCH0 erfasst den technischen Meilisearch-Consumer
weiter. Für die aktuelle Karte bleibt das Cap als bekannte Begrenzung
akzeptiert; der Cluster-Nachladefix erhält keinen eigenen Branch und
bleibt ohne Produktänderung zurückgestellt. Ein UI-Signal, eine Änderung
von `maxTotalHits` oder vollständige Cluster oberhalb des Caps werden
damit nicht zugesagt. Die [Duplikat-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-upload-duplikatprüfung)
und [Cluster-Einstufung](/develop/specs/trail-search/work-items/search/srch0/#nicht-blockierende-clusterbegrenzung)
halten die noch offene Diagnose-/Abnahmetrennung bei unveränderten Tests
und Korpus fest. Zugriffsscope, Sichtbarkeit und Berechtigungen bleiben
verbindlich.

## Parallele Index- und Engine-Schnitte

| Work Item | Ziel | Stand |
| --- | --- | --- |
| [IDX0](/develop/specs/trail-search/work-items/engine/idx0/) | sicherer Alltags-Bootstrap, Legacy-Readiness und dokumentierter erster Wrapper-Rollout | ohne Vorgänger sofort umsetzbar; kein Engineupgrade |
| [M1](/develop/specs/trail-search/work-items/engine/m1/) | unterstützter Betreiber-Upgradepfad auf Meilisearch 1.53.1 | fachlich bereit; eigener Betriebsrelease |

SRCH0 prüft schon vor seiner Abnahme Datenerhalt beim Startup, abgeschlossene
Initialisierung vor Suchfreigabe und Fehler-/Retrypfade. Der vollständige
IDX0-Wrapper samt Readiness-Control-Plane bleibt separat; IDX0 kann weiterhin
ohne Vorgänger beginnen.

## Geo-Discovery

| Work Item | Ziel | Stand |
| --- | --- | --- |
| [G2](/develop/specs/trail-search/work-items/geo-discovery/g2/) | versionierte segmenterhaltende Suchgeometrie | wartet auf A1 |
| [G3a](/develop/specs/trail-search/work-items/geo-discovery/g3a/) | generationengebundene räumliche Indexfamilie | wartet auf G2 und IDX3 |
| [G3b](/develop/specs/trail-search/work-items/geo-discovery/g3b/) | Search-Integration Core mit `total` und Pagination | wartet auf G3a und SRCH1b |
| [L4](/develop/specs/trail-search/work-items/geo-discovery/l4/) | sichtbarer Routenradius für einen normalisierten Punktanker | wartet auf G3b und Place-/Suchanker |

## Typisierte Ortssuche

| Work Item | Ziel | Stand |
| --- | --- | --- |
| [L3](/develop/specs/trail-search/work-items/place-search/l3/) | Scope-Chips und gruppierte Startpunkt-Ortssuche | unabhängige Providerlinie; wartet auf GEO2 und Adapter |

L3 ist kein Vorläufer von L4. L4 kann einen Punkt aus der erhaltenen
Submit-Suche verwenden und benötigt weder Photon noch Autocomplete. L3 wird
dagegen erst aktiviert, wenn ein Provider seine typisierten Kategorien und
Autocomplete-Fähigkeit nach GEO2 belegt.

## Additive Counts und Histogramme

| Work Item | Ziel | Stand |
| --- | --- | --- |
| [SRCH3](/develop/specs/trail-search/work-items/aggregations/srch3/) | exhaustive disjunktive Counts und versionierte Histogramme | wartet auf Zielindex/Gateway und SRCH2/A0 |
| [SRCH4b](/develop/specs/trail-search/work-items/aggregations/srch4b/) | Count-/Range-UI mit Lade- und Fehlerzuständen | wartet auf SRCH3 und SRCH4a |
| [G3b-AGG](/develop/specs/trail-search/work-items/aggregations/g3b-agg/) | dieselben Aggregationen im räumlichen Suchkontext | wartet auf G3b und SRCH3; kein L4-Gate |

## Noch nicht geschlossener Baugraph

Die initialen Work Items bilden die aktuelle Fokuslinie ab, aber nicht alle
Voraussetzungen als eigene Seiten. Insbesondere A0/A1, IDX3, SRCH1/SRCH1b,
PLACE1, L1, GEO1/GEO2 und die Security-Bausteine bleiben verbindlich. Solange
für sie keine eigene Work-Item-Spec existiert, gelten ihre Definition im
[Delivery-Katalog](/develop/specs/trail-search/delivery/) und die jeweils verlinkten normativen
Verträge. Ein abhängiges Work Item darf nicht allein wegen einer vorhandenen
Markdown-Datei als `ready` markiert werden.

Neue Seiten verwenden das [Work-Item-Template](/develop/specs/trail-search/work-items/template/).
