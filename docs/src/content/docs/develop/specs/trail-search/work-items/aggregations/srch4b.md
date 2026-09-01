---
title: SRCH4b — Count- und Range-UI
description: Dynamische Optionszahlen, Histogramme und robuste Lade-/Fehlerzustände als additive Panelverbesserung.
editUrl: false
sidebar:
  order: 2
  badge: Blockiert
spec:
  id: SRCH4b
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: search-aggregations
  exposure: user-visible
  implementationDependsOn: [SRCH3, SRCH4a]
  releaseGates: [SEC-VIS-1, SEC-AUTH-1, SEC-DUR-1, SEC-SCOPE-1]
  normativeSources: [SRCH-V1-CONTRACT]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | blockiert durch SRCH3 und SRCH4a |
| Ergebnis | echte dynamische Zahlen und Verteilungen im bestehenden Panel |
| Degradierung | einzelne fehlerhafte Gruppen sichtbar nicht verfügbar; Treffer bleiben nutzbar |

## Nutzerergebnis und Releaseeinheit

Das Filterpanel zeigt, wie viele Trails eine Option unter den übrigen aktiven
Filtern ergeben würde. Numerische Filter erhalten reale Histogramme und
verständliche offene Endbereiche. Zahlen ändern sich mit der Auswahl und werden
nicht aus einem alten oder unvollständigen Request weiterverwendet.

## Scope

- Optionscounts neben Kategorie-, Subkategorie- und weiteren diskreten Filtern.
- Histogramme hinter Distanz-, Aufstiegs- und Abstiegsbereichen.
- Stabile Bucketlabels, Einheiten und „und mehr“-Endbereiche.
- Getrennte Lade-, Erfolg-, Leer-, Gruppenfehler- und nicht verfügbare Zustände.
- Abbruch überholter Lookup-Requests und Schutz vor vertauschten Antworten.
- Ausgewählte Optionen auch mit Count 0 anzeigen und entfernbar halten.
- Treffer und angeforderte Aggregationen in derselben V1-SearchResponse und
  demselben `search_context` auswerten. Der interne Plan darf dafür mehrere
  gebundene Enginequeries verwenden; die UI führt keine zweite fachliche
  Aggregationsanfrage zusammen.

## Nichtziele

- Keine selbst berechneten Clientcounts.
- Keine Beispielzahlen als Lade- oder Fehlerersatz.
- Keine freie automatische Bucketverschiebung, die geteilte URLs oder
  Vergleiche unverständlich macht.
- Keine räumlichen Optionszahlen, bevor G3b-AGG verfügbar ist.

## Abnahme

- Jede sichtbare Zahl stammt aus der aktuellen erfolgreichen SRCH3-Gruppe.
- Schnelles Ändern mehrerer Filter kann keine ältere Antwort einblenden.
- Gruppenfehler entfernen nicht die Trefferliste und werden zugänglich erklärt.
- Tastatur, Screenreader und Touch vermitteln Bucket, Bereich, Einheit und
  offenen Endpunkt eindeutig.
- Beim Routenradius bleiben räumliche Zahlen bis G3b-AGG explizit nicht
  verfügbar, statt aus einem nicht räumlichen Universum angezeigt zu werden.

## Entscheidungsprotokoll

| Datum | Entscheidung |
| --- | --- |
| 2026-08-30 | Das Panel wird zuerst ohne Fake-Counts veröffentlicht; SRCH4b ergänzt echte Zahlen additiv. |
