---
title: SRCH4a — Filterpanel-Redesign
description: Nutzbares responsives Filterpanel mit Bestandsparität, Hierarchie, Sortierung, Chips und stabilem URL-State.
editUrl: false
sidebar:
  order: 5
  badge: Blockiert
spec:
  id: SRCH4a
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-panel
  exposure: user-visible
  implementationDependsOn: [SRCH-COMP, SRCH2]
  releaseGates: [SEC-VIS-0]
  normativeSources: [SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-09-19'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch SRCH-COMP und SRCH2 |
| Implementierungsabhängigkeiten | SRCH-COMP, SRCH2 |
| Sichtbares Releasegate | SEC-VIS-0 |
| Nicht enthalten | Optionscounts und Histogramme aus SRCH4b |

## Nutzerergebnis und Releaseeinheit

Das Filterpanel macht die bereits funktionierenden Suchwege verständlich und
bedienbar: Freitext, Kategorie/Subkategorie, Ort, Sortierung, aktive Filter und
weitere Filter besitzen eine klare Hierarchie auf Desktop und Mobilgeräten.
Jede sichtbare Auswahl verändert echte Requests und Treffer.

## Scope

- Prominente Suchwege für Freitext, Kategoriehierarchie, Ort und Sortierung.
- Drei bis vier häufige Filter direkt, übrige unter „Weitere Filter“.
- Aktive Chips mit einzelnem Entfernen und verständlichem Gesamt-Reset.
- Kategorieicons immer mit sichtbarer Bezeichnung oder zugänglichem Namen.
- Stabile URL-Synchronisierung einschließlich Zurück/Vorwärts und Reload.
- Lade-, Leer-, Fehler- und nicht verfügbare Capability-Zustände.
- Responsive Tastatur-, Fokus-, Screenreader- und Touchbedienung.
- Entfernen aller fest codierten Beispielzahlen und funktionslosen
  Prototypfelder; dieses UI-Delta bindet `SRCH0-GAP-UI-001`, ohne andere
  aktive SRCH0-Erwartungen oder deren historische Evidenz umzuschreiben.
  Der Prototyp gehört nicht zur Produktbaseline `e9b7a8cad` und wird nicht
  als funktionierende Suchfähigkeit in deren Korpus aufgenommen.

## Nichtziele

- Noch keine Optionscounts, Histogramme oder scheinbar dynamischen Ersatzwerte.
- Keine neuen Fachfilter jenseits von SRCH2.
- Keine typisierte Provider-Ortssuche L3 und kein Routenradius L4; die
  vorhandene Ortssuche und der Startpunktmodus bleiben jedoch erreichbar.
- Noch keine benannten oder benutzergebundenen Standardsuchen; diese bilden in
  SRCH-SAVED einen eigenen vollständigen Produktschnitt.

## URL- und Zustandsvertrag

Die UI ist eine Ansicht des normalisierten V1-Auftrags. Sie besitzt keinen
zweiten lokalen Filterzustand mit abweichenden Defaults. Ungültige oder nicht
mehr verfügbare URL-Werte werden nach dem Vertrag erklärt beziehungsweise
kanonisiert; sie verschwinden nicht still.

Der Gesamt-Reset erzeugt die fachlich uneingeschränkte V1-Suche für die
aktuelle Locale und ihre explizite kanonische V1-URL. SRCH-SAVED ergänzt später getrennt „Standardsuche
wiederherstellen“; es deutet den bestehenden Reset nicht still um.

## Abnahme

- Sämtliche SRCH0-Filter bleiben im neuen Panel erreichbar.
- Jede Steuerung erzeugt einen typisierten SRCH-COMP-Auftrag und echte Treffer.
- Das Overlay zu `SRCH0-GAP-UI-001` entfernt alle funktionslosen
  `TrailFilterPreview`-Beispielwerte; keine Beispieldaten werden als Count,
  Range oder Verfügbarkeit dargestellt.
- Tastatur- und mobile Bedienpfade decken Öffnen, Ändern, Anwenden und Entfernen
  vollständig ab.
- URL-Roundtrip, Browsernavigation, Abbruch überholter Requests und Fehlerzustand
  sind automatisiert getestet.

## Mögliche Review-Schnitte

Visuelle Komponenten dürfen isoliert in Story-/Fixture-Zuständen reviewed
werden. Das sichtbare Release erfolgt erst mit vollständiger Bestandsparität,
echter API-Anbindung und bestandenen Accessibility-/URL-Tests.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Das Panel ist der erste einfache reale Verbraucher des V1-Vertrags; Counts bleiben ein additiver Slice. |
| 2026-09-01 | SRCH4a besitzt das UI-Overlay zu `SRCH0-GAP-UI-001`; der neutrale SRCH0-Basisfall bleibt unverändert. |
| 2026-09-19 | SRCH4a erhält die korrigierte aktive SRCH0-Semantik. Das UI-Delta ersetzt ausschliesslich den separat dokumentierten Spec-Prototyp. |
