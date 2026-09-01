---
title: SRCH-SAVED — Gespeicherte Suchen und Trail-Listen-Default
description: Owner-private gespeicherte TrailSearchSpecV1-Aufträge, ein oberflächenspezifischer Default und deterministische Listenauflösung.
editUrl: false
sidebar:
  order: 6
  badge: Blockiert
spec:
  id: SRCH-SAVED
  kind: work-item
  status: draft
  deliveryStatus: blocked
  capability: FOUNDATION
  productSlice: saved-searches
  exposure: user-visible
  implementationDependsOn: [SRCH-COMP]
  releaseGates: [SRCH4a, SEC-VIS-0]
  normativeSources: [SAVED-TRAIL-SEARCH-V1, SRCH-V1-CONTRACT, TRAIL-SEARCH-SHARED]
  lastReviewed: '2026-08-30'
---

## Metadaten

| Feld | Wert |
| --- | --- |
| Delivery-Status | Implementierung wartet auf SRCH-COMP; sichtbarer Slice zusätzlich auf SRCH4a und SEC-VIS-0 |
| Implementierungsabhängigkeiten | SRCH-COMP |
| Sichtbare Releasegates | SRCH4a, SEC-VIS-0 für den verwendeten Suchpfad |
| Ergebnis | benannte Suchen und eine verlässliche Standardsuche für die Trail-Liste |

## Nutzerergebnis und Releaseeinheit

Angemeldete Nutzer können einen aktuellen Trail-Suchauftrag benennen,
wiederverwenden und als Standardsuche für die Trail-Liste festlegen. Ein
frischer Einstieg in `/trails` beginnt damit; Deep Links, Browsernavigation und
bewusst abweichende Suchen bleiben unverändert reproduzierbar.

Der Slice ist erst vollständig, wenn Persistenz, owner-sichere API,
Defaultauflösung, kanonische URL-Materialisierung, sichtbare Bedienaktionen und
Fehlerreparatur gemeinsam funktionieren. Ein blosses Browser-LocalStorage-Preset
ist kein veröffentlichbares Ergebnis.

## Scope

- Owner-private Persistenz für mehrere `SavedTrailSearchV1`-Datensätze.
- Getrennte eindeutige Benutzer-/Surface-Präferenz
  `TrailSearchDefaultV1`, in V1 für `trail_list`.
- Capability-, CRUD-, Default- und Resolve-Endpunkte einschliesslich
  Revision/`If-Match` und vollständig begrenzter Metadatenliste.
- Serverseitige Normalisierung jeder gespeicherten Spec über SRCH-COMP.
- Einmalige Auflösung vor dem ersten Listenrequest und Materialisierung der
  vollständigen Spec per `history.replaceState`.
- UI-Aktionen für Speichern, Umbenennen, Aktualisieren, Löschen, Anwenden, als
  Default setzen, Default entfernen, Default wiederherstellen und bewusstes
  Fortfahren ohne einen reparaturbedürftigen Default.
- Getrennte, eindeutig benannte Aktion „Alle Filter entfernen“.
- Reparaturzustand für inzwischen ungültige IDs, Berechtigungen, Locales oder
  Capabilities ohne stilles Entfernen von Bedingungen.
- E2E-Abdeckung für URL-Präzedenz, History, Reload, Pagination, mehrere Geräte
  und konkurrierende Revisionen.

## Nichtziele

- Keine öffentlichen oder gemeinsam bearbeitbaren Saved-Search-Datensätze;
  geteilt wird weiterhin die kanonische URL.
- Keine gespeicherten Treffer, Counts, Cursor, Snapshots oder Search-Tokens.
- Kein Default für Karte, Homepage oder Dashboard ohne eigenen registrierten
  Surface-Vertrag.
- Keine zeitgesteuerten Suchen, Benachrichtigungen oder Ergebnisabonnements.
- Keine Chat-Schreibaktion; Q6 darf dieselben APIs später nach Bestätigung
  konsumieren.
- Keine neue Meilisearch-Projektion und kein besonderes Ranking.

## Heutiges Verhalten und Evidenz

Die Trail-Liste erzeugt ihren Default derzeit in
`web/src/routes/trails/+page.ts`. Der Browser kann einen kompletten
`trailListFilter` aus LocalStorage wiederherstellen, entfernt diesen Zustand
aber beim Verlassen von `/trails` in `web/src/routes/trails/+page.svelte`.
Sortierung und Seitengrösse besitzen zusätzliche globale LocalStorage-Keys.
Dieser Zustand ist besuchsbezogen, nicht benutzergebunden, nur teilweise in der
URL abgebildet und deshalb keine geeignete Saved-Search-Persistenz.

SRCH0 hält die genaue Legacypräzedenz und die getrennten Sort-/Page-Keys bereits
als Fixtures fest. SRCH-SAVED ergänzt diese Matrix, ändert sie aber nicht
rückwirkend.

## Normative Quellen

- [Gespeicherte Trail-Suchen v1](/develop/specs/trail-search/contracts/saved-trail-search-v1/)
- [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [Gemeinsame Invarianten](/develop/specs/trail-search/shared-invariants/)
- [Federation-Sicherheits- und Publikationsvoraussetzungen](/develop/specs/trail-search/contracts/federation-security/)

## Abhängigkeiten und Freigabegates

SRCH-COMP ist Implementierungsabhängigkeit, weil gespeicherte Requests nur über
den gemeinsamen Normalisierer und Compiler angenommen werden. SRCH4a ist
Releasegate der sichtbaren Freigabe, weil jede Defaultbedingung im Panel
auffindbar, veränderbar und löschbar sein muss. Die Backendcollection und
owner-sichere API dürfen vorher intern integriert werden, werden aber erst mit
dem vollständigen UI-Slice als Benutzerfunktion exponiert.

SRCH-SAVED blockiert weder Geo-Discovery, M1, Counts noch Histogramme. Eine
gespeicherte Suche kann später jede bereits freigegebene neue Capability
enthalten, ohne dass ihr Persistenzmodell geändert wird; als Default ist sie nur
bei Parität mit dem Surfaceprofil zulässig. Für die sichtbare Trail-Suche gilt
weiterhin `SEC-VIS-0`.

## Vorgesehene Änderung und Schnittstellen

Die physische Umsetzung verwendet bevorzugt zwei getrennte Collections:

```text
trail_saved_search
  owner, name, contract, normalized_search,
  normalized_search_fingerprint, revision, created, updated

trail_search_default
  owner, surface, saved_search, revision, updated
  UNIQUE(owner, surface)
```

Collectionregeln und Domainservice erzwingen Owner-Scope, Referenzintegrität
und atomare Defaultwechsel. Der Client greift ausschliesslich über die
Domainendpunkte aus dem Saved-Search-Vertrag zu, nicht direkt auf Collections.
Insbesondere verwendet er zum Anwenden auch einer nicht als Default gesetzten
Suche deren Resolve-Endpunkt; die rohe Detailantwort ist nur Persistenz- und
Editiersicht. Das bestätigte Löschen einer aktiven Standardsuche prüft
Saved-Search- und Default-Revision und entfernt beide Datensätze in einer
Transaktion.

Beim frischen `/trails`-Einstieg lädt der Page-Resolver die Präferenz
serverseitig beziehungsweise über den authentifizierten Domainservice. Er gibt
die Quelle getrennt vom statischen Systemdefault zurück. Erst die festgelegte
Präzedenz entscheidet einmalig über die Ausgangs-Spec; Komponenten dürfen den
Default nicht später erneut über URL, Snapshot oder Benutzereingaben legen.
Vor dem Setzen prüft der Service die Spec zusätzlich gegen das versionierte
`defaultable_profile` von `trail_list`; blosse Ausführbarkeit genügt nicht.

Sortierung gehört zur gespeicherten Spec. Seitengrösse, Karten-/Listenmodus und
Panelbreite bleiben Darstellungspräferenzen und dürfen die Spec nicht
überschreiben.

## Migration, Cutover und Rollback

Es gibt keinen Backfill in benutzerdefinierte Defaults. Bestehender
`trailListFilter` wird einmalig über den SRCH-COMP-Legacyadapter in eine V1-URL
überführt; erst eine bewusste Nutzeraktion speichert ihn. Ein Migrationsmarker
im jeweiligen Browserprofil verhindert, dass derselbe LocalStorage-Zustand bei
jedem Einstieg erneut eingemischt wird.

Die Datenbankmigration legt Collections, eindeutigen Index und owner-sichere
Regeln vor Aktivierung der API an. Der UI-Cutover entfernt alte Sortierkeys aus
der Suchpräzedenz, behält jedoch reine Anzeige- und Seitengrössenpräferenzen.

Rollback deaktiviert Resolver und UI, behält Saved Searches und Pointer. Die
vorige Trail-Liste fällt auf ihren geprüften SRCH-COMP-/Legacypfad zurück. Eine
Down-Migration, die Benutzersuchen löscht, ist kein zulässiger normaler
Rollback.

## Akzeptanz- und Negativtests

- Vollständige Matrix SAV-01 bis SAV-21 des normativen Vertrags.
- Owner A kann Datensätze von Owner B weder auflisten, unterscheiden,
  referenzieren, ändern noch löschen.
- Ein Default mit Text, Taxonomie, Access, Geo und Sortierung ergibt nach
  Resolve, URL-Codec und SearchRequest denselben Fingerprint.
- Explizite URL, Legacyzustand, History/Reload, Default und Systemdefault werden
  in genau der festgelegten Reihenfolge ausgewählt.
- Eine ungültig gewordene Bedingung erzeugt einen sichtbaren Reparaturzustand
  und keinen SearchRequest mit breiterer Spec.
- „Alle Filter entfernen“ bleibt für den aktuellen Besuch leer; „Default
  wiederherstellen“ ist eine separate bewusste Aktion.
- Ein Update auf Gerät B mit veralteter Revision überschreibt Gerät A nicht.
- Defaultänderung während einer Cursorfolge verändert weder URL noch Cursor der
  geöffneten Suche.
- Eine ausführbare, aber vom `trail_list`-Surfaceprofil nicht vollständig
  bedienbare Spec kann gespeichert, jedoch nicht als Listen-Default gesetzt
  werden.
- Eine inzwischen ungültige gewöhnliche gespeicherte Suche bleibt editierbar,
  wird beim Anwenden aber fail-closed als `requires_attention` aufgelöst.
- Create und Spec-Update weisen einen Auftrag zurück, dessen kanonische URL das
  aktuelle `max_url_bytes` überschreitet; eine spätere Limitsenkung löscht
  bestehende Datensätze nicht.

## Betrieb, Diagnose und Performance

Metriken zählen CRUD-/Resolve-Erfolg, Revisionkonflikte,
`requires_attention`-Gründe und Latenz ohne Namen, Suchtext, IDs oder
Koordinaten als Labels. Der Resolver benötigt höchstens Pointer- und
Saved-Search-Lookup; ein N+1 pro Treffer oder eine Meilisearch-Abfrage vor dem
eigentlichen SearchRequest ist unzulässig.

Ein Lasttest deckt Listenauflösung und CRUD bei der capability-publizierten
Maximalzahl gespeicherter Suchen ab. Die normale Suchlatenz wird getrennt
gemessen, weil dieselbe Spec ohne Saved Search identische Enginearbeit erzeugt.

## Offene Entscheidungen und Spikes

Vor Implementierung werden die konkreten, capability-publizierten Maxima für
Namenslänge, gespeicherte Suchen pro Benutzer und
`max_normalized_spec_bytes` festgelegt. Das URL-Limit wird nicht dupliziert,
sondern aus `request_limits.max_url_bytes` der gebundenen Search-Capability
übernommen. Diese Limits ändern weder die fachliche Spec noch die
Auflösungspräzedenz.

## Review- und Beitragsschnitte

1. Normative Schemas, Collections, Regeln und owner-sichere CRUD-/Resolve-API
   können intern integriert werden.
2. Resolver, URL-/History-Präzedenz und E2E-Fixtures können ohne sichtbare
   Menüaktionen reviewed werden.
3. Verwaltungs- und Panelaktionen bilden den vollständigen sichtbaren Slice;
   erst dieser Schnitt erhält die Produktionsfreigabe.

## Entscheidungsprotokoll

| Datum | Entscheidung | Begründung |
| --- | --- | --- |
| 2026-08-30 | Eigener Work Item nach SRCH-COMP; sichtbare Freigabe nach SRCH4a. | Persistenz und UX sind ein vollständiger Nutzerslice, aber kein Geo-, Count- oder Engine-Gate. |
| 2026-08-30 | Zwei logische Persistenzrollen statt eines LocalStorage-Defaults. | Mehrere benannte Suchen und ein Surface-Pointer bleiben synchronisierbar und erweiterbar. |
