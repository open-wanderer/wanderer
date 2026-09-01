---
title: Gespeicherte Trail-Suchen v1
description: Normativer Persistenz-, Auflösungs-, URL- und Lebenszyklusvertrag für gespeicherte Suchen und oberflächenspezifische Standardsuchen.
editUrl: false
sidebar:
  order: 2
  badge: Entwurf
spec:
  id: SAVED-TRAIL-SEARCH-V1
  kind: contract
  status: draft
  capability: FOUNDATION
  lastReviewed: '2026-08-30'
---

Status: Normativer Entwurf, 30. August 2026

Dieser Vertrag ergänzt den
[Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
um benutzergebundene gespeicherte Suchaufträge und eine Standardsuche für den
Einstieg in die Trail-Liste. Er definiert keinen zweiten Suchpfad: Gespeichert
wird ein normalisierter `TrailSearchSpecV1`, ausgeführt wird weiterhin ein
normaler `SearchRequestV1`.

Die Schlüsselwörter **MUSS**, **DARF NICHT**, **SOLL** und **KANN** sind
normativ gemeint.

## 1. Produktsemantik

Eine Standardsuche ist der sichtbare **Startzustand** einer Oberfläche und kein
unsichtbarer Dauerfilter. Beim frischen Einstieg in `/trails` ohne expliziten
Suchzustand darf Wanderer die benutzerdefinierte Standardsuche anwenden. Danach
kann der Nutzer sie wie jeden anderen Suchauftrag ändern, leeren oder teilen.

Insbesondere gilt:

- eine Standardsuche wird nie zusätzlich mit einer expliziten URL verknüpft;
- derselbe kanonische Suchauftrag besitzt unabhängig von seiner Herkunft
  dieselbe Filter-, Sortier-, ACL-, Federation- und Count-Semantik;
- die UI zeigt sämtliche aus der Standardsuche stammenden Bedingungen als
  normale aktive Controls beziehungsweise Chips; und
- eine gespeicherte Suche enthält Absicht, niemals bereits ermittelte Treffer.

V1 unterstützt als Default-Scope genau `trail_list`. Weitere Oberflächen wie
`trail_map` oder ein Dashboard benötigen einen eigenen registrierten Scope und
einen eigenen Default-Pointer. Ein Default wird nicht implizit auf eine andere
Oberfläche übertragen.

## 2. Persistierte Rollen

### 2.1 `SavedTrailSearchV1`

Ein gespeicherter Suchauftrag besitzt mindestens:

| Feld | Typ | Regel |
| --- | --- | --- |
| `contract` | String | exakt `trail-saved-search.v1` |
| `id` | opake lokale ID | stabil; nie aus Name oder Spec abgeleitet |
| `name` | String | nach Trim nicht leer; innerhalb des angebotenen Längenlimits |
| `search` | `TrailSearchSpecV1` | vollständig normalisiert, einschliesslich expliziter Locale und Sortierung |
| `normalized_search_fingerprint` | opaker String | vom Search-Normalisierer erzeugt; nicht clientseitig nachbauen |
| `revision` | opaker String | ändert sich bei Name oder Spec |
| `created_at`, `updated_at` | RFC-3339-Zeit | serverseitig vergeben |

`page`, `page.size`, `cursor`, `include`, `search_context`, `as_of`, Treffer,
Counts, Histogramme, Kartenbounds, Scrollposition, Darstellungsart und
Panelzustand gehören nicht in `SavedTrailSearchV1`. Eine Speicherung aus einem
laufenden Resultat übernimmt ausschliesslich dessen `normalized_search`.

Die Locale ist Teil der gespeicherten fachlichen Absicht. Die UI sendet sie beim
Anlegen und Ändern explizit. Ein späterer Wechsel der Benutzerlocale schreibt
eine gespeicherte Suche nicht still um; die Oberfläche darf eine bewusste
Aktualisierung auf die aktuelle Locale anbieten.

### 2.2 `TrailSearchDefaultV1`

Die Standardsuche ist eine getrennte Benutzerpräferenz:

| Feld | Typ | Regel |
| --- | --- | --- |
| `contract` | String | exakt `trail-search-default.v1` |
| `surface` | Enum | in V1 exakt `trail_list` |
| `saved_search_id` | opake ID | muss eine eigene gespeicherte Suche referenzieren |
| `revision` | opaker String | ändert sich beim Setzen oder Entfernen des Pointers |
| `updated_at` | RFC-3339-Zeit | serverseitig vergeben |

Pro Benutzer und Surface existiert höchstens ein Pointer. Der Pointer folgt der
aktuellen Revision der referenzierten Suche; er enthält keine zweite Spec-Kopie.
Eine bereits geöffnete Seite arbeitet dagegen mit der in ihrer kanonischen URL
materialisierten Spec weiter und ändert sich nicht reaktiv.

Die logische Trennung SOLL auch in der Datenbank erhalten bleiben: eine private
Collection für mehrere gespeicherte Suchen bis zum publizierten Limit und eine
eindeutige Benutzer-/Surface-Präferenz für den Default. Ein Feld am Auth-Record
oder ein Browser-LocalStorage-Blob ist nicht die kanonische Persistenz.

## 3. Eigentum, Datenschutz und Grenzen

Gespeicherte Suchen und Default-Pointer sind ausschliesslich für
authentifizierte Benutzer verfügbar und owner-private. Sie werden weder
föderiert noch über eine öffentliche ID auflösbar. Das Teilen einer aktuell
ausführbaren gespeicherten Suche geschieht über ihre vollständige kanonische
V1-URL, nicht durch Freigabe des persistierten Datensatzes.

Suchtext, Ortsanker, Autoren- und Accessauswahl können private Absichten
offenlegen. Deshalb gelten mindestens:

- Owner-Prüfung bei jedem List-, Read-, Write-, Resolve- und Delete-Pfad;
- keine rohen Specs oder URLs in Analytics, Metriklabels oder strukturierten
  Anwendungslogs;
- dieselben Requestgrössen-, Feld- und Mengenlimits wie im Trail-Suchvertrag;
- capability-publizierte Grenzen für Namenslänge und Anzahl gespeicherter
  Suchen pro Benutzer; und
- optimistische Nebenläufigkeitskontrolle über `revision` beziehungsweise
  `If-Match`, damit parallele Geräte keine Änderungen still überschreiben.

Die persistierte Suche ist keine Autorisierung. Jeder spätere SearchRequest
löst IDs und Capabilities erneut im aktuellen Principal-Kontext auf und wendet
die aktuellen ACL-, Restriction-, Kategoriepräferenz- und Federationregeln an.

Der authentifizierte Capability-Endpunkt liefert
`trail-saved-search-capabilities.v1` mit `capability_revision`, den positiven
Integergrenzen `max_saved_searches`, `max_name_codepoints` und
`max_normalized_spec_bytes` sowie einer `surfaces`-Liste. V1 enthält darin genau
den Eintrag `trail_list` mit `supports_default: true` und einem versionierten
`defaultable_profile` aus `id`, `hash`, `search_capability_revision`,
ausführbaren Filterfeldern und Sortierschlüsseln. Dieses Profil ist die
Schnittmenge aus aktueller Search-Capability und den Bedingungen, die die
Trail-Liste vollständig sichtbar anzeigen, ändern und entfernen kann.

`trail_list` bezeichnet in V1 den gemeinsamen Default aller offiziell
unterstützten Listenclients. Sein Profil enthält deshalb nur Felder, die jeder
Client, der diese Surface annonciert, sichtbar anzeigen, ändern und entfernen
kann; ein Client ohne diese Parität darf die Defaultfunktion nicht anbieten.
Spätere client- oder oberflächenspezifische Defaults benötigen einen eigenen
registrierten Surfacewert und werden nicht aus `trail_list` erraten.

Eine Suche darf jeden aktuell ausführbaren `TrailSearchSpecV1` speichern, den
der aktuelle `trail_search_url_v1`-Codec vollständig kanonisieren kann und
dessen kanonische URL `request_limits.max_url_bytes` der gebundenen
Search-Capability nicht überschreitet. Diese Prüfung erfolgt nach der
Normalisierung; `max_normalized_spec_bytes` allein genügt nicht, weil das
URL-Encoding den Auftrag vergrössern kann. Als Default für eine Surface darf
die Suche nur gesetzt werden, wenn ihre vollständige Spec zusätzlich deren
`defaultable_profile` erfüllt. Eine später nicht mehr erfüllte Capability, ein
gesunkenes URL-Limit oder ein nicht mehr erfülltes Surfaceprofil führt beim
Resolve zu `requires_attention`; der Server entfernt keine Felder.

Die Antwort trägt `Cache-Control: private` und einen aus Contract,
`capability_revision`, Limits und Surfaceprofilen abgeleiteten `ETag`. Der
Hauptendpunkt `GET /api/v1/trails/search/capabilities` annonciert den
Saved-Search-Capability-Endpunkt nur für authentifizierte Actors und nur, wenn
der vollständige SRCH-SAVED-Slice freigegeben ist. Fehlt diese Annonce, darf ein
Client die Funktion nicht erraten oder teilweise anzeigen.

Wird ein Betreiberlimit unter einen vorhandenen Bestand gesenkt, bleiben
bestehende Datensätze les- und löschbar. Neue Datensätze sind erst wieder
zulässig, wenn der Bestand unter `max_saved_searches` liegt. Bei Updates darf
eine unveränderte, inzwischen zu grosse Komponente erhalten bleiben; jeder
geänderte Name beziehungsweise jede geänderte Spec muss die aktuellen
Einzelgrenzen erfüllen. Eine bestehende Spec, deren kanonische URL das aktuelle
`max_url_bytes` überschreitet, bleibt über Liste und Detail editierbar, ergibt
bei Saved-Search- und Default-Resolve jedoch `requires_attention` und kann nicht
neu als Default gesetzt werden. Eine automatische Löschung oder Kürzung findet
nie statt. Der Build besitzt zusätzlich nicht konfigurierbare sichere
Obergrenzen; Startup lehnt Betreiberwerte oberhalb dieser Grenzen ab.

## 4. CRUD- und Default-Schnittstellen

Die Domain-API stellt diese Operationen bereit; die endgültige
OpenAPI-Darstellung wird aus demselben Schema generiert:

```text
GET    /api/v1/trails/search/saved/capabilities
GET    /api/v1/trails/search/saved
POST   /api/v1/trails/search/saved
GET    /api/v1/trails/search/saved/{id}
GET    /api/v1/trails/search/saved/{id}/resolve
PATCH  /api/v1/trails/search/saved/{id}
DELETE /api/v1/trails/search/saved/{id}
POST   /api/v1/trails/search/saved/{id}/delete

GET    /api/v1/trails/search/defaults/{surface}
PUT    /api/v1/trails/search/defaults/{surface}
DELETE /api/v1/trails/search/defaults/{surface}
```

Create und Patch normalisieren `search` serverseitig über exakt denselben
Normalisierer wie die Search-API. Ungültige, unbekannte oder aktuell nicht
autorisierte Werte werden nicht gespeichert. Eine neu angelegte oder geänderte
Spec muss sich ausserdem mit der aktuellen Search-Capability als vollständige
kanonische URL innerhalb von `max_url_bytes` materialisieren lassen. Ein Patch
ersetzt Name und/oder vollständige Spec; JSON-Merge in einzelne verschachtelte
Filter ist kein öffentlicher Vertrag.

Ist die geänderte Suche aktuell ein Default, muss ihre neue Spec das
`defaultable_profile` der betreffenden Surface weiterhin erfüllen. Andernfalls
wird der Patch atomar mit `search_not_defaultable_for_surface` abgelehnt. Der
Nutzer kann zuerst den Default entfernen und die Suche danach unabhängig
ändern; es entsteht kein absichtlich kaputter Default-Zwischenzustand.

Die Listenoperation liefert wegen des harten `max_saved_searches`-Limits die
vollständige Metadatenliste aus `id`, `name`, Fingerprint, Revision und Zeiten,
stabil nach `updated_at DESC, id ASC`; die vollständige Spec wird über den
Detailendpunkt geladen. `PATCH` und `DELETE` einer gespeicherten Suche verlangen
`If-Match` mit der zuletzt gelesenen Saved-Search-Revision. Das erstmalige
Setzen eines Defaults verwendet `If-None-Match: *`; Ändern und Entfernen
verlangen dessen aktuelle Default-Revision.

Die Liste verwendet die Hülle `trail-saved-search-list.v1` mit
`complete: true` und `items`; sie wird weder gekürzt noch paginiert. Detail-,
Create- und Patchantworten liefern `SavedTrailSearchV1`, wobei dessen `revision`
zugleich der `ETag` ist. Saved-Search-, Default- und Resolveantworten tragen
`Cache-Control: private, no-store`; die separat authentifizierte
Capabilityantwort darf dagegen privat per `ETag` revalidiert werden.

Der Detailendpunkt ist die editierbare Persistenzsicht und darf daher auch eine
inzwischen nicht mehr ausführbare Spec zurückgeben. Zum tatsächlichen Anwenden
einer benannten Suche verwendet der Client ausschliesslich
`GET saved/{id}/resolve`. Dieser Endpunkt normalisiert und autorisiert die
aktuelle gespeicherte Revision erneut und liefert HTTP `200` mit
`contract: trail-saved-search-resolution.v1`, Saved-Search-ID, Name,
`saved_search_revision`, `normalized_search_fingerprint` und dem Diskriminator
`state`:

```text
ready               -> normalized_search und canonical_url
requires_attention  -> sichere Feldprobleme; keine Suche und keine canonical_url
```

`ready` bedeutet serverseitige Ausführbarkeit und URL-Materialisierbarkeit. Der
aufrufende Client darf die URL nur anwenden, wenn er jede enthaltene Bedingung
sichtbar darstellen, ändern und entfernen kann; andernfalls zeigt er einen
Reparaturzustand und führt keine Suche aus. Der offizielle Trail-Listen-Client
muss diese Parität für jede von ihm angebotene Aktion „Gespeicherte Suche
anwenden“ besitzen. `requires_attention` entsteht nach denselben aktuellen
Capability-, ID-, ACL-, Locale- und URL-Prüfungen wie bei einem Default. Der
Server sucht weder mit einer älteren Revision noch mit einer bereinigten Spec.

Create antwortet mit `201` und dem neuen Datensatz, Patch mit `200` und der
neuen Revision. Erfolgreiches normales oder Compound-Delete sowie das Entfernen
des Default-Pointers antworten mit `204`. Ein erfolgreiches `PUT` des Defaults
antwortet mit der `ready`-Resolution derselben Transaktion.

`PUT defaults/{surface}` akzeptiert genau eine eigene `saved_search_id` und
prüft die aktuelle Spec gegen das `defaultable_profile` dieser Surface. Das
normale `DELETE saved/{id}` lehnt eine aktuell referenzierte Suche mit
`saved_search_is_default` ab.

Der bestätigte Compound-Befehl `POST saved/{id}/delete` akzeptiert
`saved_search_revision`, `clear_default: true` und `default_revision`. Er prüft
beide Revisionen und entfernt Pointer und Saved Search in genau einer
Datenbanktransaktion. Ist die Suche inzwischen nicht mehr der Default oder hat
sich eine Revision geändert, folgt `saved_search_revision_conflict`; es gibt
keine teilweise Mutation. Datenbankregeln verhindern einen dangling Pointer
auch bei konkurrierenden Requests.

`GET defaults/{surface}` ist zugleich der Resolve-Endpunkt und liefert immer
HTTP `200` mit einer diskriminierten Antwort; ein fehlender Default ist kein
`404`. Die Hülle besitzt `contract: trail-search-default-resolution.v1`,
`surface` und den Diskriminator `state`:

```text
not_configured      -> Surface; keine Revision und keine Suche
ready               -> default_revision, Saved-Search-ID, saved_search_revision,
                       normalized_search und canonical_url
requires_attention  -> default_revision, Saved-Search-ID, optionale
                       saved_search_revision und sichere Feldprobleme; keine Suche
```

Pointer und Ziel werden aus demselben konsistenten Datenbank-Snapshot gelesen.
Bei einem dangling Pointer bleibt `saved_search_revision` aus, während
`default_revision` und `saved_search_id` die gezielte Entfernung oder Reparatur
ermöglichen.

`requires_attention` entsteht, wenn eine früher gültige Spec wegen gelöschter
IDs, entzogener Berechtigung, nicht mehr angebotener Capability, ungültiger
Locale, gesunkenem `max_url_bytes`, nicht mehr erfülltem Surfaceprofil oder
inkonsistenter Bestandsdaten nicht mehr ausführbar beziehungsweise nicht mehr
vollständig materialisierbar ist. Der Resolver entfernt keine Bedingung und
führt keine breitere Ersatzsuche aus. `requires_attention` ist ein erwartbarer
Lebenszykluszustand und kein HTTP-Fehler; fehlerhaft geformte, nicht
authentifizierte oder nebenläufig veraltete Aufrufe verwenden dagegen normale
Problem Details.

## 5. Auflösungs- und Vorrangvertrag

Beim Einstieg in die Trail-Liste gilt genau diese Reihenfolge:

1. Eine gültige explizite `trail_search_url_v1` wird unverändert als fachliche
   Quelle verwendet.
2. Eine Legacy-URL wird durch den Legacyadapter ausgewertet.
3. Falls in diesem Browserprofil noch erforderlich, wird einmalig der
   bestehende Legacy-LocalStorage gemäss Trail-Suchvertrag migriert.
4. Nur wenn keinerlei Suchzustand vorliegt, wird der Default für `trail_list`
   aufgelöst.
5. Bei `not_configured` gelten die dokumentierten statischen V1-Defaults.

Eine vorhandene, aber ungültige oder nicht unterstützte URL blockiert mit ihrem
typisierten Fehler. Sie fällt niemals auf die Standardsuche zurück. Ebenso wird
eine `requires_attention`-Standardsuche nicht durch den Systemdefault ersetzt,
bevor der Nutzer sichtbar „ohne Standardsuche fortfahren“, „reparieren“ oder
„Default entfernen“ gewählt hat.

Ein **frischer Einstieg** ist eine Navigation zur kanonischen Trail-Listenroute
ohne Suchparameter und ohne für diesen History-Eintrag wiederherzustellenden
Suchzustand. Dazu gehört insbesondere der normale Navigationslink „Trails“.
Nicht dazu gehören:

- Browser-Zurück oder -Vorwärts zu einer vorhandenen Such-URL;
- Reload einer bereits materialisierten V1-URL;
- ein Deep Link mit Suchparametern;
- ein bewusster Listen-/Karten-Handoff mit vollständiger Spec; oder
- Pagination beziehungsweise eine Cursorfolge derselben Suche.

Damit wird der Default genau einmal als Eingangszustand gewählt und nicht nach
jeder Komponenteninitialisierung oder Filteränderung erneut angewandt.

## 6. URL- und History-Materialisierung

Eine erfolgreich aufgelöste Standardsuche wird vor dem ersten sichtbaren
SearchRequest in die vollständige kanonische `trail_search_url_v1` überführt.
Der Browser ersetzt den aktuellen History-Eintrag; er erzeugt keinen
zusätzlichen Zurück-Schritt. Die URL enthält die vollständige Spec, aber weder
Saved-Search-ID noch Default-Revision.

Eine bewusst angewandte gewöhnliche gespeicherte Suche verwendet dieselbe
kanonische URL, ist aber eine Nutzeraktion: Innerhalb der bereits geöffneten
Trail-Liste erzeugt sie per `history.pushState` einen neuen History-Eintrag;
von einer anderen Oberfläche navigiert sie normal auf diese URL. Erst nach der
Materialisierung folgt genau ein SearchRequest. Zurück, Vorwärts und Reload
behandeln sie danach wie jede andere explizite V1-URL und lösen den
persistierten Datensatz nicht erneut auf.

`saved_search_id`, `saved_search_revision` und die Herkunft `user_default`
beziehungsweise `saved_search` dürfen als nichtfachliche Provenienz in
`history.state` liegen. Nach Reload kann die UI den Namen auch über den
Fingerprint wiedererkennen. Weder Provenienz noch Name beeinflussen Treffer,
Cache-Key, Cursor oder Teilbarkeit.

Diese Materialisierung stellt sicher:

- aktive Filter und Sortierung bleiben sichtbar und teilbar;
- dieselbe URL bezeichnet für unterschiedliche Benutzer dieselbe fachliche
  Spec, jeweils innerhalb ihres autorisierten Universums;
- Reload und Browsernavigation benötigen keinen versteckten zweiten
  Filterzustand; und
- eine spätere Änderung des Defaults verändert bestehende URLs nicht.

## 7. Bedien- und Lebenszyklusregeln

Die Trail-Liste bietet mindestens diese getrennt benannten Aktionen:

- „Suche speichern“ legt einen benannten Auftrag aus der aktuellen
  `normalized_search` an.
- „Gespeicherte Suche anwenden“ löst deren aktuelle Revision über den
  Resolve-Endpunkt auf und materialisiert nur den Zustand `ready` als
  kanonische URL.
- „Als Standardsuche verwenden“ setzt den Pointer für `trail_list`.
- „Meine Standardsuche wiederherstellen“ ersetzt die aktuelle Spec bewusst
  durch die aktuelle referenzierte Revision. Ist diese nicht ausführbar, öffnet
  die Aktion den Reparaturzustand und sucht keine ältere Revision.
- „Alle Filter entfernen“ erzeugt die fachlich uneingeschränkte V1-Spec für die
  aktuelle Locale, materialisiert deren explizite kanonische V1-URL und setzt
  den Default nicht erneut im selben Seitenbesuch.
- „Ohne Standardsuche fortfahren“ ist ausschliesslich im
  `requires_attention`-Zustand verfügbar. Die Aktion lässt den kaputten Pointer
  zur späteren Reparatur bestehen, materialisiert für diesen Besuch dieselbe
  neutrale V1-URL wie „Alle Filter entfernen“ und führt erst damit eine Suche
  aus. Reload und History bleiben neutral; ein neuer Einstieg über das bare
  `/trails` löst denselben Pointer erneut auf und zeigt wieder
  `requires_attention`.
- „Standardsuche entfernen“ löscht nur den Pointer, nicht automatisch die
  gespeicherte Suche.

Die UI kennzeichnet, wenn der aktuelle Auftrag dem aktiven Default entspricht.
Eine Änderung erzeugt zunächst nur einen normalen abweichenden Suchzustand; sie
überschreibt die gespeicherte Suche erst nach einer ausdrücklichen Aktion.

Wird eine als Default verwendete Suche geändert, gilt die neue Revision beim
nächsten frischen Einstieg. Ihr Löschen verlangt eine Bestätigung, die den
Default-Pointer atomar mit entfernt. Falls eine administrative Migration oder
Inkonsistenz dennoch einen Pointer mit fehlendem Ziel erzeugt, bleibt die
Auflösung fail-closed als `requires_attention` und erzeugt eine Diagnose.

Nach „Alle Filter entfernen“ stellen Reload und Zurück/Vorwärts die explizite
neutrale URL wieder her. Erst eine neue Navigation auf das bare `/trails`
wendet den Default erneut an.

## 8. Search-Context-, ACL- und Federation-Verhalten

Die Auflösung des Defaults erzeugt noch keinen langlebigen Suchkontext. Der
anschliessende normale SearchRequest erhält einen frischen `search_context` mit
aktuellen Content-, ACL-, Preference-, Capability- und Federationrevisionen.
Insbesondere werden nie gespeichert oder wiederverwendet:

- Cursor oder zuvor gelieferte Seiten;
- `as_of`, `content_snapshot` oder Federation-Snapshotset;
- frühere Treffer, Totals, Facetten oder Histogramme; oder
- Tenant-/Search-Tokens und andere Autorisierungsartefakte.

Ändert sich nur die gespeicherte Suche oder der Default-Pointer, invalidiert das
keinen bereits laufenden Cursor: Dessen vollständige Spec steht bereits in URL
und Request. Ändert sich dagegen eine vom Trail-Suchvertrag gebundene ACL-,
Preference-, Capability- oder Federationrevision, gelten unverändert dessen
Stale- und Retryregeln.

## 9. Fehlervertrag

Neben den unverändert verwendeten Search- und URL-Fehlern existieren diese
stabilen Domaincodes:

| HTTP | Code | Bedeutung |
| ---: | --- | --- |
| 401 | `saved_search_authentication_required` | anonymer Zugriff auf gespeicherte Suchen oder Defaults |
| 404 | `saved_search_not_found` | ID existiert für diesen Owner nicht; keine Existenzbestätigung fremder Datensätze |
| 409 | `saved_search_revision_conflict` | Saved-Search- oder Default-Revision ist veraltet |
| 409 | `saved_search_is_default` | Delete ohne atomare Entfernung des aktiven Default-Pointers |
| 409 | `saved_search_limit_exceeded` | capability-publiziertes Benutzerlimit erreicht |
| 422 | `invalid_saved_search` | Name oder enthaltene Spec ist ungültig |
| 422 | `saved_search_url_too_large` | normalisierte Spec lässt sich unter dem aktuellen `max_url_bytes` nicht vollständig als kanonische URL materialisieren |
| 422 | `unsupported_default_surface` | Surface ist nicht registriert |
| 422 | `search_not_defaultable_for_surface` | Spec ist ausführbar, aber auf dieser Surface nicht vollständig darstell- und editierbar |

Fehlerantworten nennen sichere Feldpfade und die zugrunde liegenden stabilen
Searchcodes, enthalten aber keine Information über fremde Datensätze oder
Autorisierungsdetails.

## 10. Migration, Rollback und Performance

Der bestehende `trailListFilter` bleibt ausschliesslich Quelle der einmaligen
Legacy-Migration. Er wird nie automatisch als benannte oder als Standardsuche
persistiert. Die UI darf nach erfolgreicher Migration „Diese Suche speichern“
anbieten. Alte separate Sortierkeys dürfen nach dem V1-Cutover weder eine
explizite URL noch eine gespeicherte Sortierung überschreiben; reine
Darstellungspräferenzen und Seitengrösse bleiben davon getrennt.

Die Defaultauflösung benötigt höchstens einen owner-/surfacegebundenen
Pointer-Lookup und einen Lookup der referenzierten Suche. Sie löst keinen
Meilisearch-Indexaufbau und keine besondere Queryform aus. Der anschliessende
Suchaufwand entspricht exakt derselben manuell gesetzten Spec.

Backendpersistenz und Domainservice dürfen vor der sichtbaren UI intern
integriert werden. Bis zum vollständigen Slice fehlen jedoch die
`saved_searches`-Annonce in den Hauptcapabilities und jeder öffentlich routbare
Saved-Search-Endpunkt. Ein Rollback der Oberfläche entfernt Annonce und Routen,
lässt gespeicherte Datensätze jedoch unangetastet. Ein späteres erneutes
Aktivieren muss dieselben Revisionen lesen können; ein Rollback löscht oder
konvertiert keine Benutzersuchen still.

## 11. Normative Akzeptanzmatrix

| ID | Eingabe oder Ereignis | Erwartung |
| --- | --- | --- |
| SAV-01 | aktuelle normalisierte Suche speichern und über `saved/{id}/resolve` anwenden | `ready`, identische `TrailSearchSpecV1`, identischer Fingerprint und kanonische URL vor genau einem SearchRequest |
| SAV-02 | `/trails` ohne Suchzustand und gültiger `trail_list`-Default | genau einmal auflösen, URL per Replace materialisieren, genau ein SearchRequest |
| SAV-03 | explizite gültige V1-URL plus gesetzter Default | URL gewinnt vollständig; kein Merge |
| SAV-04 | explizite ungültige URL plus gesetzter Default | typisierter URL-Fehler; Default wird nicht ausgeführt |
| SAV-05 | Browser-Zurück, Reload oder Pagination | vorhandene Spec wiederherstellen; Default nicht erneut anwenden |
| SAV-06 | Default enthält inzwischen gelöschte oder unzulässige ID | `requires_attention`; keine breitere Ersatzsuche |
| SAV-07 | aktueller Default wird ohne Pointerentfernung gelöscht | `saved_search_is_default`; keine Änderung |
| SAV-08 | Compound-Delete mit beiden aktuellen Revisionen | atomare Entfernung von Pointer und Suche; nächster frischer Einstieg zeigt den Systemdefault |
| SAV-09 | „Alle Filter entfernen“, danach Reload, Zurück/Vorwärts und neue bare Navigation | explizite neutrale V1-URL bleibt bei Reload/History neutral; erst das neue bare `/trails` wendet den Default erneut an |
| SAV-10 | „Meine Standardsuche wiederherstellen“ | aktuelle referenzierte Revision wird materialisiert; bei `requires_attention` keine Suche nach älterer Revision |
| SAV-11 | Saved Search wird während einer Cursorfolge geändert | laufende URL/Spec und Cursor bleiben unverändert |
| SAV-12 | ACL-, Preference- oder Federationstand ändert sich | frischer SearchContext beziehungsweise normale Stale-Regel; kein gespeichertes Ergebnis |
| SAV-13 | fremde Saved-Search-ID lesen, setzen oder löschen | identisches owner-sicheres Not-found-Verhalten |
| SAV-14 | alte LocalStorage-Suche beim ersten V1-Einstieg | einmalige Migration, aber kein automatisches Anlegen eines Defaults |
| SAV-15 | gespeicherte Sortierung plus alte globale Sortierkeys | gespeicherte V1-Sortierung gewinnt; Darstellungszustand bleibt getrennt |
| SAV-16 | Default-Pointer mit fehlendem Ziel | `200 requires_attention` mit `default_revision`, Saved-Search-ID und ohne `saved_search_revision`; gezieltes Entfernen bleibt möglich |
| SAV-17 | ausführbare Spec enthält ein auf `trail_list` nicht vollständig bedienbares Feld | Speichern zulässig; `PUT default` und entsprechender Patch eines aktiven Defaults ergeben `search_not_defaultable_for_surface` |
| SAV-18 | Saved-Search-Capability nicht annonciert oder Actor anonym | UI bietet Funktion nicht an; direkte benutzergebundene Operation ist nicht verfügbar beziehungsweise `401` |
| SAV-19 | nicht als Default verwendete gespeicherte Suche enthält inzwischen eine gelöschte oder unzulässige ID | Detail bleibt editierbar; Resolve ergibt `requires_attention` ohne URL und ohne SearchRequest |
| SAV-20 | Create oder Spec-Patch erzeugt nach Kanonisierung eine URL oberhalb `max_url_bytes`; danach wird das Limit unter einen Bestandsdatensatz gesenkt | Mutation ergibt `saved_search_url_too_large`; der Bestand bleibt les-/löschbar, Resolve ergibt `requires_attention` und verbreitert nichts |
| SAV-21 | Nutzer wählt bei kaputtem Default „Ohne Standardsuche fortfahren“ | Pointer bleibt bestehen; neutrale V1-URL gilt für Besuch, Reload und History; nächstes bare `/trails` zeigt erneut `requires_attention` |

## 12. Entscheidungsprotokoll

| Datum | Entscheidung | Begründung |
| --- | --- | --- |
| 2026-08-30 | Standardsuchen sind gespeicherte `TrailSearchSpecV1` und kein eigener Suchpfad. | UI, URL, API, App und späterer Dialog behalten eine Semantik. |
| 2026-08-30 | Der Default ist ein Surface-Pointer und kein unsichtbarer Filteroverlay. | Deep Links bleiben reproduzierbar und aktive Einschränkungen sichtbar. |
| 2026-08-30 | Eine aufgelöste Standardsuche wird vollständig per Replace in die URL materialisiert. | Reload, Teilen und Browsernavigation benötigen keinen zweiten Filterzustand. |
| 2026-08-30 | Ungültige gespeicherte Bedingungen verbreitern die Suche nicht still. | Entfernte IDs oder Capabilities dürfen keinen unerwartet grösseren Ergebnisscope erzeugen. |
