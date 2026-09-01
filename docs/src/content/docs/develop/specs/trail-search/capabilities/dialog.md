---
title: DIALOG — Dialogbasierte Trail-Suche
description: Produktanforderungen für die Übersetzung natürlicher Sprache in dieselben typisierten und auditierbaren Suchdienste wie im Filter-UI
editUrl: false
sidebar:
  order: 4
  badge: Entwurf
spec:
  id: CAP-DIALOG
  kind: capability
  status: draft
  capability: DIALOG
  lastReviewed: '2026-08-30'
---

## Rolle und Normativität

Diese Seite ist die fachliche Capability-Spezifikation für eine dialogbasierte
Trail-Suche. Sie ist ein **normativer Entwurf für Verhalten und
Sicherheitsgrenzen von DIALOG**, aber weder ein zweites Suchprotokoll noch eine
Festlegung auf ein bestimmtes Sprachmodell oder eine bestimmte
MCP-Implementierung.

Querschnittliche Regeln für Kompatibilität, Datenschutz, Autorisierung,
Versionierung, Beobachtbarkeit und Freigabe sind ausschliesslich in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/) festgelegt. Exakte
normalisierte Suchen, Requests, Responses, Capabilities, Counts, Pagination,
Fehler und URL-Zustände definiert ausschliesslich der
[Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/). Mögliche Bausteinschnitte
und ihre Abhängigkeiten werden ausschliesslich unter
[Delivery und Beiträge](/develop/specs/trail-search/delivery/) geführt.

## Capability-Grenze

DIALOG setzt einen stabilen strukturierten Suchvertrag und mindestens einen
nützlichen, vollständig freigegebenen Suchumfang voraus. Nicht jede Capability
aus DERIVED oder CONTEXT muss zuvor existieren. Capability Discovery begrenzt
den Dialog auf Filter, Fakten, Erklärungen und Aktionen, die auf der Instanz
tatsächlich verfügbar sind. Spätere Produkt-Capabilities erweitern den Dialog
additiv.

Ein Sprachmodell darf Suchabsichten interpretieren, aber nie Treffer,
Filterwerte, Verfügbarkeiten, Counts oder Gründe erfinden. MCP ist für den
internen Chat nicht erforderlich; es ist ein optionaler Adapter über dieselben
typisierten Anwendungsdienste.

## Eine Suchimplementierung

Der Chat implementiert die Suche nicht unabhängig. Er erzeugt dieselbe
versionierte normalisierte Suchspezifikation wie das Filterpanel. Ein
Tooladapter ergänzt lediglich Transportbelange wie Pagination und angeforderte
Responsebestandteile und ruft danach typisierte Wanderer-Anwendungsdienste auf.

Chips, URL-Zustand, klassische Filteroberfläche, API-Clients und Dialog bleiben
dadurch reproduzierbare Ansichten desselben Auftrags. Ihre normative
Gleichwertigkeit definiert der
[Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/).

## Beispielinteraktion

Das folgende Beispiel ist nicht normativ und setzt voraus, dass die
entsprechenden Profil-, Wegbeschaffenheits-, ÖV- und Wetter-Capabilities
freigegeben sind:

```text
„Suche für Samstag eine Wanderung mit Mia, höchstens vier Stunden,
meist Naturweg, ohne Stellen über T1, in Zugnähe und nicht zu windig.“

-> outing_profile = „Mit Mia“
-> activity_family = hike
-> start_time = Samstag …
-> expected_duration.max = 4 h
-> surface.natural_share >= …
-> max_sac_scale = T1
-> access.mode = rail, endpoints = required
-> forecast.wind_difficulty <= …
```

Ein früherer Dialog-Release kann bereits Freitext, bestehende Filter,
Subkategorien und Sortierung sinnvoll anbieten. Nicht unterstützte Bedingungen
und Tools fehlen in Capability Discovery, statt angenähert oder still ignoriert
zu werden.

## Interaktionsdesign

Der Chat zeigt extrahierte Bedingungen als editierbare Chips, markiert Annahmen
und fragt nur nach, wenn eine Mehrdeutigkeit das Ergebnis wesentlich verändern
kann. Treffer verwenden die normalen Trail-Karten und zeigen Quellen,
Datenstand und eine strukturierte Erklärung, warum ein Trail passt. Vergleiche
verwenden strukturierte Fakten statt erfundener Prosa-Scores.

Die interne Anwendungs-Tooloberfläche umfasst mindestens die folgenden
Operationen, sofern ihre zugrunde liegenden Capabilities verfügbar sind:

```text
resolve_place
select_outing_profile
search_trails
explain_match
compare_trails
get_trail_context
plan_transit
get_weather_and_season_context
```

Die Verfügbarkeit eines Tools folgt den Capabilities. Diese Liste autorisiert
kein Tool auf einer Instanz, auf der sein zugrunde liegender Dienst oder sein
Freigabegate nicht verfügbar ist.

Schreibende Tools wie `save_search`, `set_default_search`, `add_to_list` oder
Profiländerungen bleiben von Read-only-Suchtools getrennt. `save_search` und
`set_default_search` konsumieren den
[Vertrag für gespeicherte Trail-Suchen](/develop/specs/trail-search/contracts/saved-trail-search-v1/)
und erfinden weder Persistenz noch Defaultauflösung neu. Schreibende Tools
benennen die konkrete Mutation und verlangen unmittelbar vor der Ausführung
eine explizite Bestätigung. Eine Einwilligung zur Suche oder Erklärung darf
nicht als Einwilligung zu einer Mutation wiederverwendet werden.

Alle Tools laufen als aktueller Wanderer-Benutzer durch dieselben
Autorisierungs-, Federation-, Provider- und Suchgrenzen wie das klassische UI.
Das Modell erhält nie einen Meilisearch-Master-Key oder direkten
Datenbankzugriff.

## Rollen von Sprachmodell und MCP

Der interne Wanderer-Chat sollte die Anwendungsdienste direkt aufrufen. Dadurch
bleiben Authentisierung, Latenz, Betrieb und Tests unabhängig von einem
Protokolladapter.

Ein optionaler MCP-Server darf dieselben Read-only-Tools für externe Clients,
einschliesslich Desktop-Assistenten, bereitstellen. Jeder Aufruf wird als
Wanderer-Benutzer authentisiert, verwendet dieselbe Capability Discovery und
enthält keine duplizierte Suchlogik.

Umgekehrt ist ein MCP-Client nicht der primäre Integrationspfad für Wetter- oder
Fahrplanprovider. Deterministische Datasets, Caches, Scheduler, Gültigkeit und
Dienstzusagen gehören in ihre kontextuellen Provider-Domains.

Die Modellanbindung ist providerneutral und darf lokal betriebene oder
konfigurierte entfernte Modelle unterstützen. Instanzadministratoren steuern
unterstützte Modelle, Tool- und Kontextlimits sowie die Datenübertragung.

## Datenschutz, Aufbewahrung und Audit

Private GPX-Daten, Teilnehmernamen und Suchhistorie verlassen die Instanz nicht
ohne explizite Providerkonfiguration und informierte Einwilligung des Benutzers.
Chats besitzen steuerbare Aufbewahrung und Löschung. Toolaufrufe sind mit
Benutzer, Request-Version und Ergebnisrevision auditierbar, ohne unnötige
Geheimnisse oder rohe private Inhalte zu protokollieren.

Diese Anforderungen spezialisieren die allgemeinen Regeln in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/), duplizieren sie aber nicht.

## Zuverlässigkeit und Evaluation

Ein festes mehrsprachiges Eval-Corpus deckt mindestens ab:

- Kategorien und Subkategorien;
- Negation;
- Einheiten;
- relative Datumsangaben in der Instanzzeitzone;
- Ausflugsprofile;
- unbekannte und nicht verfügbare Daten;
- bösartige Instruktionen in Trailtexten.

Erwartet wird nicht identische Prosa, sondern eine identische normalisierte
typisierte Suchabsicht. Trailbeschreibungen und föderierte Inhalte sind immer
nicht vertrauenswürdige Daten und nie Toolinstruktionen.

Eval-Schwellen für korrekte Extraktion und verbotene Behauptungen werden vor
der Freigabe gewählt und zusammen mit der geprüften Modell-/Providerkonfiguration
festgehalten. Ändern sich Modell, Promptvertrag, Toolschemas oder unterstützte
Sprachen, wird die relevante Evaluation erneut ausgeführt.

## Capability-spezifische Freigabegates

Eine DIALOG-Produktscheibe darf erst ausgestellt werden, wenn ihre Abhängigkeiten
unter [Delivery und Beiträge](/develop/specs/trail-search/delivery/) sowie diese Gates erfüllt sind:

- Filter-UI, Chat und ein optionaler MCP-Aufruf erzeugen für dieselbe Absicht
  dieselbe normalisierte typisierte Suche und dasselbe Suchuniversum;
  transportspezifische Pagination und Response-Includes bleiben getrennt.
- Capability Discovery bietet nur freigegebene Filter und Tools an. Eine nicht
  verfügbare Absicht wird ausdrücklich zurückgewiesen und nie angenähert.
- Jede aktive Bedingung, Annahme und unbekannte Kontextachse ist im Chat sichtbar
  und editierbar.
- Treffer, Counts, Vergleiche und Erklärungen stammen ausschliesslich aus
  typisierten Toolergebnissen mit der einschlägigen Projektions- oder
  Kontextrevision.
- Prompt-Injection-, Autorisierungs-, Bestätigungs- und Datenschutztests bestehen
  für lokale und föderierte Trails.
- Die vorab definierte Eval-Suite erreicht ihre Schwellen für korrekte
  Filterextraktion und unzulässige Behauptungen.
- Nutzung entfernter Modelle, Aufbewahrung, Löschung und Audit entsprechen der
  informierten Einwilligung und der dem Benutzer gezeigten
  Administratorkonfiguration.

Allgemeine Sicherheits- und Cutover-Gates werden hier nicht wiederholt; jedes
ausgestellte Tool erfüllt zusätzlich die anwendbaren Regeln in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/).

## Quellen

- [Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [Gemeinsame Invarianten der Trail-Suche](/develop/specs/trail-search/shared-invariants/)
- [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
