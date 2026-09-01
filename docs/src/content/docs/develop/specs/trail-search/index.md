---
title: Erweiterte Trail-Suche
description: Produktüberblick, Capability-Landkarte und aktuelle Baureihenfolge für Suche, Geo-Discovery und Personalisierung.
editUrl: false
sidebar:
  order: 1
  label: Überblick
  badge: Einstieg
spec:
  id: TRAIL-SEARCH-OVERVIEW
  kind: overview
  status: draft
  lastReviewed: '2026-08-30'
---

Diese Spezifikation beschreibt, wie Wanderer von einer überwiegend statischen
Filtersuche zu einer typisierten, räumlichen und auf Wunsch personalisierten
Trail-Suche ausgebaut werden kann. Dieses Dokument ist der Einstieg in den
privaten Spezifikationskorpus. Es erklärt Produktziel, Prioritäten und
Dokumentgrenzen, ist aber bewusst weder vollständiger API-Vertrag noch
Implementierungsanleitung.

Der Korpus ist kein Zeitplan und kein Vollständigkeitsversprechen. Bausteine
können von Maintainerinnen, Maintainern oder der Community übernommen, geteilt
oder zusammengelegt werden. Sichtbare Releases müssen ein eigenständiges
Benutzerziel vollständig erfüllen; eine vorläufige Oberfläche ohne echte
Funktion ist kein Produktschnitt.

## Warum die Suche erweitert wird

Die heutige Suche besitzt bereits wertvolle Filter, bildet aber mehrere typische
Planungsfragen nur unzureichend ab:

- Ein Ortsradius berücksichtigt bislang den Startpunkt statt die gesamte Route.
- Kategorie, Subkategorie, Ort und Sortierung sind in der Oberfläche nicht als
  klare, zusammenhängende Suchwege erkennbar.
- Globale Maximalwerte machen Distanz- und Höhenregler für normale Touren kaum
  bedienbar; Beispielzahlen reagieren nicht auf die tatsächliche Ergebnismenge.
- Gespeicherte Schwierigkeit und Dauer spiegeln häufig den Ersteller oder eine
  neutrale Referenz wider, nicht die gewählte Person oder Gruppe.
- Wegbeschaffenheit, Anstiege, Anreise, Saison und Wetter sind segment- oder
  zeitabhängige Fakten und lassen sich nicht ehrlich durch einzelne statische
  Trailfelder ersetzen.
- Federation vergrößert Suchraum und Sicherheitswirkung. Counts, Pagination,
  ACL und Widerrufe müssen deshalb denselben klar gebundenen Datenstand nutzen.

Das Ziel ist nicht eine möglichst lange Filterliste. Wanderer soll Fragen wie
„Welche familiengeeignete Rundtour berührt diesen Ort, ist heute realistisch und
hat überwiegend passende Wege?“ über dieselbe fachliche Suchsprache in Panel,
Karte, API und später einem Dialog beantworten können.

## Aktuelle Produktprioritäten

Nach den jeweils betroffenen Security-Freigabegates besitzen derzeit diese
Vorhaben den höchsten sichtbaren Nutzwert:

1. **Geo-Discovery:** typisierte Ortssuche und Suche entlang der gesamten Route.
2. **Filterpanel-Redesign:** bestehende Filter erhalten, Kategorie und
   Subkategorie, Sortierung, verständliche Bereiche und belastbare Zustände
   sichtbar machen.
3. **Wegbeschaffenheit und Wegtyp:** versionierte, persistierte Segmentfakten mit
   dokumentiertem Kartenstand und begrenztem Refresh-Lifecycle.
4. **Persönliche Trailschwierigkeit:** Eignung für Walk, Hike und Cycle aus
   objektiven Routenmerkmalen und dem gewählten Einzel- oder Gruppenprofil
   ableiten statt die Leistungsgrenzen des Erstellers zu übernehmen.

Diese Liste ist eine Wertpriorisierung. Die technische Baureihenfolge beginnt
mit dem gemeinsamen Suchvertrag und einem einfacheren realen Verbraucher, damit
die räumliche Integration nicht gleichzeitig Vertrag, UI, ACL, Federation,
Cursor und Pagination erstmals erproben muss.

## Planungssituationen

Das Zielbild unterscheidet bewusst mehrere Situationen:

- **Allein unterwegs:** Ein persönliches Profil kann Kondition, Aktivitätsart
  und bevorzugte technische Anforderungen abbilden.
- **Mit Kind oder Gruppe:** Ein Ausflugsprofil enthält mehrere Teilnehmende,
  Pausen, Ausrüstung und die jeweils begrenzende Anforderung. Ein Kind benötigt
  dafür kein eigenes Konto.
- **Saisonale Planung:** Statische Routenfakten bleiben von Wetterprognose,
  Wind, Temperatur, Schnee oder Tageslicht getrennt. Ein zeitgebundener Modus
  darf Dauer und Eignung für ein konkretes Fenster neu bewerten.
- **Anreise und Logistik:** Rundweg, Start/Ziel, Bahn, Bus, weitere
  Verkehrsmittel und Parkmöglichkeiten sind getrennte, erklärbare Merkmale.
- **Föderierte Suche:** Ein Ergebnis darf nur erscheinen, wenn Herkunft,
  Publikationsstatus und Sichtbarkeit für den gebundenen Suchkontext belegt sind.

## Vier Capability-Klassen

| Klasse | Leitfrage | Detaildokument |
| --- | --- | --- |
| `FOUNDATION` | Welche vorhandenen oder direkt indexierbaren Fakten werden gesucht, gefiltert und sortiert? | [Suchfundament](/develop/specs/trail-search/capabilities/foundation/) |
| `DERIVED` | Was lässt sich reproduzierbar aus Route, Metadaten und gewähltem Profil ableiten? | [Lokale Ableitungen und Personalisierung](/develop/specs/trail-search/capabilities/derived/) |
| `CONTEXT` | Welche räumlichen, segmentbasierten oder externen Kontexte verändern die Planung? | [Geo, Anstiege, Zugang, Wetter und Surface](/develop/specs/trail-search/capabilities/context/) |
| `DIALOG` | Wie wird dieselbe typisierte Suche dialogisch bedient, ohne eine zweite Semantik zu schaffen? | [Dialogbasierte Suche](/develop/specs/trail-search/capabilities/dialog/) |

Die Klassen sind keine Releasephasen. Ein vollständiger Produktschnitt darf
Capability-Grenzen überschreiten, und unabhängige Beiträge dürfen parallel
entstehen. Gemeinsame Regeln stehen einmalig in den
[querschnittlichen Invarianten](/develop/specs/trail-search/shared-invariants/).

Der [Vertrag für gespeicherte Trail-Suchen](/develop/specs/trail-search/contracts/saved-trail-search-v1/)
macht typisierte Suchaufträge benennbar und geräteübergreifend wiederverwendbar.
Als erster Produktschnitt kann ein Benutzer genau eine davon als sichtbaren
Startzustand der Trail-Liste wählen. Explizite URLs und Browser-History gewinnen
weiterhin; der Default ist kein versteckter Filteroverlay. Dieselbe Grundlage
ist später der Anschlusspunkt für konfigurierbare Startseiten oder Dashboards,
ohne eigene Filtersemantik oder zweiten Suchpfad.

## Globale Produktinvarianten

Über alle Capabilities gelten insbesondere diese Regeln:

1. **Kein Bestandsfilter verschwindet.** Migrationen müssen heutige Filter,
   Sortierungen, URL-Zustände und ACL-Kontexte inventarisieren, nachweisen und
   zurückrollen können.
2. **Keine vorgetäuschte Funktion.** Filter verwenden echte Daten; noch nicht
   unterstützte Counts, Histogramme oder Felder werden nicht durch feste
   Beispielwerte ersetzt.
3. **Eine Suchsprache.** Panel, Karte, API, App und Dialog verwenden denselben
   typisierten Suchauftrag. Browser oder Modell erzeugen keine freien
   Meilisearch-Ausdrücke. Web und Mobile dürfen unterschiedliche Teilmengen
   bereits freigegebener Controls anbieten; fehlende Mobile-UI blockiert weder
   Backend noch Web, solange gemeinsame Felder dieselbe Semantik besitzen.
4. **Fakten bleiben von Kontext getrennt.** Objektive Routen- und Segmentfakten
   werden nicht mit Benutzerprofil, Wetterfenster oder Erklärung vermischt.
5. **Externe Anreicherung geschieht nicht im Suchrequest.** Benötigte Daten
   werden vorab persistiert, versioniert, mit Quelle und Datenstand versehen und
   über begrenzte Jobs aktualisiert.
6. **Security ist ein Freigabegate, kein globaler Arbeitsstopp.** Interne,
   nicht exponierte Implementierung darf parallel entstehen; jeder sichtbare
   Slice wartet auf die für seinen Datenpfad geltenden Gates.
7. **Schmale Releases sind erlaubt, halbe Produkte nicht.** Ein Slice muss einen
   eigenständigen Nutzen korrekt liefern und seine Grenzen ehrlich benennen.

## Aktuelle Baureihenfolge

Die vollständigen Abhängigkeiten und Freigabegates stehen im
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/). Als aktuelle Arbeitsorientierung
gilt:

```text
Such-/Panel-Linie:
SRCH0 Korpus/Tooling -> SRCH-V1 -> SRCH-COMP -> SRCH0 Liveaktivierung
SRCH-COMP -> SRCH2 -> SRCH4a
SRCH4a -> SRCH-SAVED

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

Der Meilisearch-Betriebsrelease M1 läuft parallel zur Such-/Panel-Linie und wird
nicht mit einem neuen UI- oder Geo-Feature gebündelt. L3 ist eine eigenständige
Providerlinie für typisierte Startpunktsuche und kein Vorläufer von L4. Counts
und Histogramme sind wertvolle additive Produkte, blockieren aber weder das
ehrliche Panel ohne Optionszahlen noch den Routenradius mit korrektem `total`.
SRCH-SAVED ist ein additiver Nutzerslice nach dem Panel und kein Gate für
Geo-Discovery, Counts oder das Engine-Upgrade.

Die konkreten Umsetzungsspezifikationen stehen im
[Work-Item-Register](/develop/specs/trail-search/work-items/). Ein Work Item ist ein stabiler
fachlicher und technischer Scope; es kann später durch einen oder mehrere Pull
Requests umgesetzt werden.

## Dokumentlandkarte und Autorität

| Dokumentart | Autorität |
| --- | --- |
| Dieses Overview | Produktziel, Prioritäten und Navigation |
| [Capability-Dokumente](/develop/specs/trail-search/capabilities/foundation/) | fachliche Details und capability-spezifische Qualitätsgates |
| [Querschnittliche Invarianten](/develop/specs/trail-search/shared-invariants/) | gemeinsame Daten-, Versions-, Job-, Plugin- und Releasegrundsätze |
| [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/) | normative Request-, Response-, Filter-, Count-, URL- und Capability-Semantik |
| [Gespeicherte Trail-Suchen v1](/develop/specs/trail-search/contracts/saved-trail-search-v1/) | normative Persistenz, Defaultauflösung, URL-Präzedenz und Lebenszyklus gespeicherter Suchen |
| [Federation-Sicherheitsvertrag](/develop/specs/trail-search/contracts/federation-security/) | normative Visibility-, Authority-, Durability- und Publication-Gates |
| [Zustandsautomaten](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/) | normative interne Control Plane, Revisionen, Queues, Generationen und Recovery |
| [ADR 0001](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/) | angenommener Cursor- und Snapshot-Lifecycle |
| [ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/) | angenommener GeoJSON-Direct-Vertrag für den Routenradius |
| [ADR 0003](/develop/specs/trail-search/decisions/0003-walk-hike-cycle-personal-evaluation/) | angenommener erster Bewertungsumfang und getrenntes Cycle-Modell |
| [Evidenz und Kalibrierung](/develop/specs/trail-search/evidence/evidence-and-calibration/) | nichtnormative Messungen, Historie und noch zu kalibrierende Werte |
| [Delivery-Katalog](/develop/specs/trail-search/delivery/) | aktuelle Bausteine, Abhängigkeiten, Releaseschnitte und Beiträge |
| [Work-Item-Spezifikationen](/develop/specs/trail-search/work-items/) | Scope, Migration, Tests, Rollback und konkrete Umsetzungsbezüge |

Bei Widersprüchen gelten normative Verträge und angenommene ADRs vor
Capability-Erklärungen und Overview. Messwerte werden erst durch einen expliziten
Entscheid oder Vertrag normativ. Bausteincodes und Abhängigkeiten werden nur im
Delivery-Katalog gepflegt; Work Items verlinken darauf, statt einen zweiten
globalen Plan zu erfinden.
