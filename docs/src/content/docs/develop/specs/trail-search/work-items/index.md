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
7. September 2026 in Umsetzung. `fix/srch0-findings` enthält getrennte Produktkorrekturen;
`fix/search-index-startup` besitzt noch kein Implementierungspaket. Diese
Arbeitsteilung erlaubt parallele Implementierung, ersetzt aber keine Abnahme
auf einer gemeinsamen Zielrevision. Bekannte Fehler im vereinbarten Umfang
bleiben Merge- und Abnahmeblocker; `knownViolation`, `xfail` oder historische
Fehlerausgaben dürfen die aktive Suite nicht grün machen. SRCH-COMP, SRCH2 und
SEC-VIS-0 übernehmen die korrigierte Basis und ihre jeweiligen Erweiterungen;
die notwendigen Bestandskorrekturen warten nicht auf diese Nachfolger.

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
