---
title: Trail-Suchvertrag v1
description: Normativer Request-, Response-, Filter-, Ranking-, Count-, URL- und Capability-Vertrag der Trail-Suche.
editUrl: false
sidebar:
  order: 1
  badge: Entwurf
spec:
  id: SRCH-V1-CONTRACT
  kind: contract
  status: draft
  capability: FOUNDATION
  lastReviewed: '2026-09-01'
---

Status: Normativer Entwurf, 1. September 2026

Dieser Vertrag konkretisiert den fachlichen Suchauftrag aus [Erweiterte Trail-Suche und personalisierte Filter](/develop/specs/trail-search/). Der ergänzende [Vertrag für gespeicherte Trail-Suchen](/develop/specs/trail-search/contracts/saved-trail-search-v1/) persistiert genau diese Suchsprache und definiert keinen zweiten Suchpfad. Für `route_radius_ux_v1` ist die angenommene [ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/) die höherrangige Architektur- und Produktentscheidung; abweichende frühere G1-Annahmen gelten nicht fort. Der [Vertrag der Federation-Sicherheits- und Publikationsvoraussetzungen](/develop/specs/trail-search/contracts/federation-security/) ist ein normatives Gate für jeden produktiven Cutover, blockiert aber weder Konzeptarbeit noch interne Parser, Backfills, Shadowprojektionen oder nicht ausstellbare Capability-Arbeit. Der Vertrag ist die gemeinsame fachliche Sprache von Filterpanel, Karten- und Listenansicht, API, Counts, Mobile-App, Chat und einem optionalen MCP-Adapter. Meilisearch-Requests, Datenbankabfragen und URL-Parameter sind Implementierungs- beziehungsweise Transportformate und keine alternative Suchsemantik. Priorisierung, konkrete Umsetzungsschnitte und Beiträge sind nicht Teil dieses API-Vertrags; sie werden im [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/) geführt.

Die Schlüsselwörter **MUSS**, **DARF NICHT**, **SOLL** und **KANN** sind normativ gemeint. Beispiele sind nur dann normativ, wenn eine unmittelbar folgende Regel dies ausdrücklich sagt.

## 1. Geltungsbereich und Schichten

Der Vertrag trennt drei Typen:

```text
TrailSearchSpecV1  = reproduzierbarer fachlicher Suchauftrag
SearchRequestV1    = TrailSearchSpecV1 + Pagination + angeforderte Zusatzdaten
SearchResponseV1   = normalisierter Suchauftrag + Treffer + Counts + Suchkontext
```

Nur `TrailSearchSpecV1` bestimmt Trefferuniversum und Reihenfolge. `page` und `include` verändern die fachliche Treffermenge nicht, werden für einen Cursor aber trotzdem gebunden, damit eine Folgeseite denselben Request fortsetzt.

V1 deckt den heutigen Trail-Katalog, die additive Kategoriehierarchie und alle in SRCH0 als unterstützt bestätigten Filter und Sortierungen ab. Künftige Filter werden als benannte, capability-gebundene Felder ergänzt; V1 veröffentlicht keine generische `{field, operator, value}`-AST und akzeptiert keine freien Meilisearch-Ausdrücke.

Die neuen Domain-Endpunkte lauten:

```text
POST /api/v1/trails/search
GET  /api/v1/trails/search/capabilities
GET  /api/v1/trails/search/options/taxonomy
GET  /api/v1/trails/search/options/tags
GET  /api/v1/trails/search/options/authors
GET  /api/v1/trails/search/options/origins
```

Der bestehende generische Endpunkt `/api/v1/search/{index}` bleibt während der Migration ein ausdrücklich als Legacy markierter Adapter. Er wird nicht still auf die neue Semantik umgedeutet und ist kein Bestandteil von `trail-search.v1`.

## 2. Versionierung

- `SearchRequestV1.contract` MUSS exakt `trail-search.v1` sein.
- Die Bedeutung eines vorhandenen Feldes, Operators oder Enumwerts DARF innerhalb von V1 nicht verändert werden.
- Additiv in V1 definierte optionale Requestfelder, Operatoren und Enumwerte benötigen Capability Discovery. Clients senden sie nur, wenn die aktuellen Capabilities sie aktivieren. Ein im implementierten V1-Schema bekanntes, aber für Actor, Instanzkonfiguration oder aktives Profil deaktiviertes Element ergibt `unsupported_capability`; der Server ignoriert es nie.
- Ein dem implementierten V1-Schema unbekanntes Requestfeld, ein unbekannter Operator oder Enumwert ergibt `invalid_search_request` mit Violation `unknown_field` beziehungsweise `invalid_enum`. V1-Schemas verwenden `additionalProperties: false`.
- Clients MÜSSEN unbekannte Responsefelder ignorieren, damit additive Anzeige- und Diagnosefelder möglich bleiben.
- Ranking-, Projektion-, Histogramm-, Spatial- und Policyverträge besitzen eigene unveränderliche Versionsnamen. Änderungen erzeugen eine neue Version und gelten nur für neue Suchkontexte.
- Ein Cursor pinnt alle aktiven Unterverträge. Ein Serverwechsel darf ihre Bedeutung während eines Suchkontexts nicht ändern.

## 3. Globale Kanonisierung und Validierung

| Eingabe                                                               | Normative Bedeutung                                                               |
| --------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Feld ausgelassen                                                      | dokumentierter Default beziehungsweise keine Einschränkung                        |
| `null` im Request                                                     | ungültig; `null` bedeutet dort weder Unknown noch „Filter entfernen“              |
| leeres `text`                                                         | erlaubt; ergibt nach Normalisierung eine leere Query                              |
| leerer String bei ID, Locale, Datum oder Zeitzone                     | `invalid_search_request`                                                          |
| vorhandener leerer Cursor                                             | `invalid_cursor`; der Cursor-Validator läuft nach der strukturellen Schemaprüfung |
| leeres Selektorarray wie `any_of: []`                                 | ungültig; niemals still „alles“ oder „nichts“                                     |
| leere `filters: {}` oder `include: {}`                                | erlaubt und neutral                                                               |
| leeres Range- oder Datumsobjekt                                       | ungültig                                                                          |
| schema-unbekanntes Objektfeld, Operator oder Enum                     | `invalid_search_request` mit `unknown_field` beziehungsweise `invalid_enum`       |
| schema-bekanntes, aber aktuell deaktiviertes Element                  | `unsupported_capability`                                                          |
| syntaktisch gültige, aber unbekannte ID                               | `unknown_filter_value`; der Filter wird nie entfernt                              |
| doppelter Arraywert                                                   | ungültig; der Legacyadapter dedupliziert vor der V1-Validierung                   |
| Zahl als String, `NaN`, Infinity oder Dezimalwert für ein Integerfeld | ungültig; keine Typkonvertierung                                                  |
| widersprüchliche Grenzen                                              | `invalid_filter_combination`; niemals tauschen oder clampen                       |

Leere Wrapper sind nur dort gültig, wo alle Kindfelder einen dokumentierten Default besitzen: `search: {}`, `filters: {}`, `sort: {}`, `filters.source: {}`, `page: {}` und `include: {}`. Sie normalisieren zu ihren jeweiligen Defaults. Dagegen verlangen `taxonomy`, `subcategory`, `tag_ids`, `author_ids`, `access` und `source_difficulty` ein nichtleeres `any_of`; ihre leeren Objekte sind ebenso ungültig wie ein leeres Range-, Datums- oder Geoobjekt. Ein Taxonomiebranch verlangt immer `category`; ein Startpunktradius verlangt `center.lat`, `center.lon` und `radius_m`.

Alle Mengenarrays mit IDs oder Enumwerten sind semantisch ungeordnet. Nach NFC werden ihre Werte nach Unicode-Codepointfolge aufsteigend sortiert. Taxonomieselectors verwenden zuerst die Kindreihenfolge `id`, `absent`, `unmapped`, `ambiguous` und bei `id` danach die ID; Branches verwenden denselben Comparator zuerst für Kategorie, dann für die normalisierte Kindmenge. `include` wird ebenso kanonisch sortiert. Treffer, Sortierschritte, Fragmente und Histogrammbuckets sind dagegen ausdrücklich geordnete Listen. Zwei Requests mit derselben normalisierten Spec erhalten denselben opaken `normalized_search_fingerprint`.

Alle fachlichen IDs im Request (`category`, `subcategory`, Tag, Author und Origin-Instanz) verwenden `StableIdV1`: 1 bis 128 ASCII-Zeichen, Regex `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`. Taxonomie-IDs dürfen zusätzlich kein `/` enthalten und nicht mit `@` beginnen. Ein Wert ausserhalb dieses Formats ist `invalid_search_request`; erst ein formatgültiger, im relevanten aktuellen beziehungsweise gepinnten Katalog nicht existierender Wert ist `unknown_filter_value`. Cursor, Context-, Snapshot- und Request-IDs sind getrennte opake Typen und nicht als Filter-ID zulässig.

Aggregations- und Taxonomie-Optionsantworten verwenden den davon getrennten opaken Typ `BucketIdV1`: 1 bis 512 ASCII-Zeichen, Regex `^[A-Za-z0-9][A-Za-z0-9._:/-]{0,511}$`. Die längere Grammatik erlaubt namespaced Taxonomie- und Range-IDs, ist aber niemals ein zulässiger Request-Filterwert. Clients vergleichen `bucket_id` höchstens auf Gleichheit und parsen daraus weder Selector noch Grenzen; diese werden immer zusätzlich typisiert ausgeliefert.

JSON-Integer werden als mathematische sichere Ganzzahlen kanonisiert. Geo-Koordinaten werden als endliche IEEE-754-Binary64-Werte interpretiert, `-0` wird `0`, und die normalisierte Spec serialisiert die kürzeste roundtrip-fähige Dezimaldarstellung ohne Exponent. Zwei verschiedene JSON-Lexeme desselben normalisierten Zahlenwerts verändern den Fingerprint nicht.

Text wird in dieser Reihenfolge normalisiert:

1. Unicode NFC;
2. führenden und nachfolgenden Unicode-Whitespace entfernen;
3. zusammenhängenden Unicode-Whitespace durch genau ein Leerzeichen ersetzen.

Der normalisierte Text darf höchstens 256 Unicode-Codepoints und gemäss `trail_text_v1` höchstens zehn Suchtokens enthalten. Eine längere Query ergibt `query_too_long`; sie wird nie still gekürzt.

## 4. `SearchRequestV1`

### 4.1 Vollständiges Beispiel

```json
{
  "contract": "trail-search.v1",
  "search": {
    "entity_kind": "trail",
    "locale": "de-CH",
    "text": "linde aussicht",
    "filters": {
      "taxonomy": {
        "any_of": [
          {
            "category": { "kind": "id", "id": "cat_hiking" },
            "subcategory": {
              "any_of": [
                { "kind": "id", "id": "sub_family" },
                { "kind": "absent" }
              ]
            }
          }
        ]
      },
      "tag_ids": { "any_of": ["tag_panorama"] },
      "source_difficulty": {
        "any_of": ["easy", "moderate", "unknown"]
      },
      "distance_m": {
        "gte": 5000,
        "lte": 20000,
        "missing": "exclude"
      },
      "trail_date": {
        "from": "2026-09-05",
        "through": "2026-09-06",
        "time_zone": "Europe/Zurich",
        "missing": "exclude"
      },
      "start_point_radius": {
        "center": { "lat": 46.8, "lon": 8.2 },
        "radius_m": 10000
      },
      "liked_by_me": true
    },
    "sort": {
      "by": "distance_m",
      "direction": "asc"
    }
  },
  "page": {
    "size": 1
  },
  "include": {
    "facets": ["source_difficulty"],
    "histograms": ["distance_m"],
    "highlights": true
  }
}
```

### 4.2 Top-Level-Felder

| Feld                 | Typ                  | Pflicht | Default und Regeln                                                                               |
| -------------------- | -------------------- | ------- | ------------------------------------------------------------------------------------------------ |
| `contract`           | String               | ja      | exakt `trail-search.v1`                                                                          |
| `search`             | `TrailSearchSpecV1`  | ja      | kein Default; auch bei Cursor-Folgeseiten vollständig vorhanden                                  |
| `page`               | Objekt               | nein    | `{ "size": 24 }`                                                                                 |
| `page.size`          | Integer              | nein    | `24`; zulässig `1..100`                                                                          |
| `page.cursor`        | opaker String        | nein    | ausgelassen bedeutet erste Seite; bei Vorhandensein validiert der Cursor-Validator 1..4096 Bytes |
| `include`            | Objekt               | nein    | `{ "facets": [], "histograms": [], "highlights": false }`                                        |
| `include.facets`     | eindeutige Enumliste | nein    | `[]`; höchstens die in Capabilities angebotenen Gruppen                                          |
| `include.histograms` | eindeutige Enumliste | nein    | `[]`; höchstens die in Capabilities angebotenen Felder                                           |
| `include.highlights` | Boolean              | nein    | `false`                                                                                          |

Total Count ist Bestandteil jeder erfolgreichen Response und kann in V1 nicht abgeschaltet werden. Treffer, Total, Seite, Suchkontext und angeforderte Highlights sind atomar. Angeforderte Facetten und Histogramme dürfen dagegen gruppenweise mit einem typisierten Fehler ausfallen: Die übrige Antwort bleibt `200`, enthält keine unvollständige Zahl und kennzeichnet sich mit `aggregation_status = partial`.

Bei vorhandenem Cursor MÜSSEN die normalisierte Spec, `page.size` und die kanonisch normalisierte `include`-Menge exakt der keyed Request-Binding des Cursors entsprechen. Eine Abweichung ergibt `cursor_request_mismatch`; eine andere Eingabereihenfolge derselben Include-Werte ist keine Abweichung.

Das generierte Requestschema prüft `page.cursor` nur als String. `null` oder ein anderer JSON-Typ ergibt deshalb strukturell `invalid_search_request`; ein leerer oder mehr als 4096 Bytes langer String erreicht dagegen den nachgelagerten Cursor-Validator und ergibt einheitlich `invalid_cursor`.

Defaults bleiben bei einer Folgeseite eindeutig: ausgelassenes `page.size` wird zu `24`, ausgelassenes `include` zum leeren Includeobjekt und alle statischen Specdefaults gelten wie auf Seite eins. Nur eine ausgelassene `search.locale` wird bei vorhandenem Cursor aus dessen gepinnter normalisierter Locale übernommen, niemals erneut aus einer inzwischen geänderten Benutzer- oder Instanzlocale. Clients SOLLEN die `normalized_search` der ersten Response sowie `page.size` und normalisierte Includes ihres ersten Requests vollständig wiederholen; bei einem anfangs nicht standardmässigen Page-/Include-Wert führt späteres Auslassen folglich zum Mismatch.

## 5. `TrailSearchSpecV1`

### 5.1 Felder und Defaults

| Feld          | Typ                    | Default                             | Regeln                                                                                            |
| ------------- | ---------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------- |
| `entity_kind` | Enum                   | `trail`                             | V1 akzeptiert zunächst nur `trail`; spätere Werte sind capability-gebunden                        |
| `locale`      | kanonischer BCP-47-Tag | Benutzerlocale, sonst Instanzlocale | normalisierte Spec enthält ihn immer explizit; URL, Chat und gespeicherte Suche MÜSSEN ihn senden |
| `text`        | String                 | `""`                                | Literaltext gemäss Abschnitt 6; keine Query-Minisprache                                           |
| `filters`     | Objekt                 | `{}`                                | unterschiedliche Gruppen sind AND                                                                 |
| `sort`        | Objekt                 | `{ "by": "relevance" }`             | wird in der normalisierten Spec immer explizit                                                    |

Ein syntaktisch ungültiger oder nicht kanonisch geschriebener BCP-47-Tag ergibt `invalid_search_request`; ein kanonischer, aber nicht angebotener Tag `unsupported_locale`. Die Transport-Defaultauflösung für `locale` ist deterministisch: expliziter Requestwert, danach authentifizierte Benutzerlocale, danach Instanzlocale. Browser- oder Server-OS-Locale spielen keine Rolle.

### 5.2 Filtermatrix

| Feld                 | Typ beziehungsweise Operator | Ausgelassen                     | Besondere Regeln                                               |
| -------------------- | ---------------------------- | ------------------------------- | -------------------------------------------------------------- |
| `taxonomy`           | hierarchisches `any_of`      | alle Taxonomiezustände          | OR zwischen Zweigen                                            |
| `tag_ids`            | `{ "any_of": string[] }`     | alle Tags                       | OR; stabile lokale IDs, keine Namen                            |
| `author_ids`         | `{ "any_of": string[] }`     | alle Autoren                    | OR; opake Actor-IDs                                            |
| `access`             | `{ "any_of": AccessKind[] }` | gesamter ACL-sichtbarer Bestand | actorbezogene Werte benötigen Authentifizierung                |
| `source`             | `SourceSelectorV1`           | `{ "scope": "all" }`            | Origin-ID ist normalisiert und niemals ein freier Domainfilter |
| `source_difficulty`  | `{ "any_of": Difficulty[] }` | bekannte und unbekannte Werte   | OR; echtes `unknown`                                           |
| `distance_m`         | Range                        | keine Einschränkung             | nichtnegative ganze Meter                                      |
| `elevation_gain_m`   | Range                        | keine Einschränkung             | nichtnegative ganze Meter                                      |
| `elevation_loss_m`   | Range                        | keine Einschränkung             | nichtnegative ganze Meter                                      |
| `trail_date`         | lokale Datumsrange           | keine Einschränkung             | inklusive Kalendertage in expliziter IANA-Zone                 |
| `completed`          | Boolean                      | beide Werte                     | `false` bedeutet ausdrücklich „nicht abgeschlossen“            |
| `liked_by_me`        | Boolean                      | beide Werte                     | `true` und `false` benötigen Authentifizierung                 |
| `has_photos`         | Boolean                      | beide Werte                     | nur bei actor-sicherer `has_photos`-Capability                 |
| `start_point_radius` | Punkt-Radius                 | kein Ortsfilter                 | ausschliesslich Startpunkt; niemals gesamte Route              |
| `route_geometry_radius` | Punkt-Radius              | kein Ortsfilter                 | capability-gebunden; gesamte Route gemäss `route_radius_ux_v1` |

Die ACL-, Tenant- und Restriction-Fences gelten zusätzlich zu allen Requestfiltern und können die Ergebnismenge nur verengen. Requestwerte können keine Sichtbarkeit erweitern.

### 5.3 Source-Scope

`SourceSelectorV1` ist exakt eine dieser diskriminierten Formen:

```json
[
  { "scope": "all" },
  { "scope": "local" },
  { "scope": "federated" },
  {
    "scope": "origin_instance",
    "origin_instance_id": "instance_example_org"
  }
]
```

- Bei `origin_instance` ist `origin_instance_id` erforderlich.
- Bei jedem anderen Scope ist `origin_instance_id` verboten.
- `origin_instance_id` ist eine serverseitig normalisierte ID aus Capability-/Suchantworten. Freie Domains, URLs und Meilisearch-Fragmente sind ungültig.
- `all` bezeichnet den lokal materialisierten, für den aktuellen Actor auffindbaren Bestand. Sein öffentlicher Anteil besteht ausschliesslich aus lokalen öffentlichen Inhalten und aktiven föderierten Publikationen mit belegter `verified_public`-Sichtbarkeit. Föderierte Direct-Shares beziehungsweise `restricted`-Publikationen gelangen nur über einen für den aktuellen Actor belegten ACL-Scope in die Ergebnismenge und niemals allein aufgrund eines Remote- oder Legacy-`public=true`. Lokale actorberechtigte private und geteilte Inhalte bleiben gemäss Access-Scope enthalten. `all` löst keine Live-Remote-Abfrage aus.

### 5.4 Access-Scope

Zulässige Werte sind:

```text
public | owned_private | shared_with_me
```

Sie bilden für den aktuellen Actor disjunkte Mengen:

- `public`: öffentlich auffindbarer Trail;
- `owned_private`: nicht öffentlicher lokaler Trail, dessen kanonisch
  persistierter Owner der aktuelle Actor ist; ein Remote-Payloadfeld `author`
  begründet diesen Scope nie;
- `shared_with_me`: nicht öffentlicher Trail eines anderen Actors mit aktiver Freigabe an den aktuellen Actor.

Mehrere Werte sind OR. Anonymous Requests dürfen nur `public` verwenden. Ein ausgelassener Access-Filter umfasst alle nach ACL sichtbaren Mengen. Ein Autorfilter ist zusätzlich AND und ändert diese Klassifikation nicht.

### 5.5 Taxonomie und hierarchisches Multi-Select

Ein Taxonomiezweig besitzt diese Form:

```json
{
  "category": {
    "kind": "id | absent | unmapped | ambiguous",
    "id": "nur bei kind=id"
  },
  "subcategory": {
    "any_of": [
      {
        "kind": "id | absent | unmapped | ambiguous",
        "id": "nur bei kind=id"
      }
    ]
  }
}
```

Die Suchprojektion bildet beide Ebenen ohne Nullwert-Heuristik auf getrennte Presence- und Mappingfakten ab:

```text
category_presence = absent | present
category_mapping_status = mapped | unmapped | ambiguous       (nur bei present)
subcategory_presence = absent | present                        (nur bei gemappter Kategorie)
subcategory_mapping_status = mapped | unmapped | ambiguous     (nur bei present)
```

Ein `id`-Selector verlangt auf derselben Ebene `presence = present`, `mapping_status = mapped` und die entsprechende stabile ID. `unmapped` und `ambiguous` verlangen `presence = present` sowie exakt ihren Mappingstatus; `absent` verlangt `presence = absent`, wobei Mappingstatus und ID nicht ausgewertet werden. Insbesondere darf ein fehlender Kategorie- oder Subkategoriequellwert nie als `unmapped` projiziert werden. Für `category.kind != id` bleiben die Subkategoriefakten fachlich ausserhalb des Zweigs und `subcategory` ist wie unten beschrieben verboten.

Normative Regeln:

- Zweige in `taxonomy.any_of` sind OR.
- Selektoren in `subcategory.any_of` sind OR.
- Taxonomie ist AND zu allen anderen Filtergruppen.
- Bei `category.kind = id` muss jede Subkategorie-ID zu genau dieser Kategorie gehören; andernfalls `invalid_filter_combination` mit Code `orphan_subcategory`.
- Ausgelassenes `subcategory` bedeutet alle Subkategoriezustände dieser Kategorie.
- `absent` bedeutet, dass kein Quellwert vorhanden ist.
- `unmapped` bedeutet, dass ein Quellwert vorhanden, aber nicht zuordenbar ist.
- `ambiguous` bedeutet, dass ein Quellwert vorhanden, aber mehrdeutig zuordenbar ist.
- Bei `category.kind != id` ist `subcategory` verboten.
- Derselbe Kategorieselektor darf nur einmal vorkommen. Ein Client fasst seine Subkategorien in demselben Zweig zusammen.
- Entfernt ein Client eine Kategorie, entfernt er auch die nun verwaisten Subkategorieauswahlen.
- Eine explizit gewählte, in den UI-Präferenzen ausgeblendete Kategorie bleibt wirksam und sichtbar markiert. UI-Präferenzen überschreiben niemals eine explizite Auswahl; ACL-Regeln weiterhin immer.

Alle übrigen Multi-Selects verwenden in V1 ausschliesslich `any_of`. Insbesondere bleiben Tags innerhalb ihrer Gruppe OR.

### 5.6 Range- und Unknown-Vertrag

`distance_m`, `elevation_gain_m` und `elevation_loss_m` verwenden denselben Typ:

```json
{
  "gte": 1000,
  "lte": 5000,
  "missing": "exclude"
}
```

| Element            | Semantik                                                            |
| ------------------ | ------------------------------------------------------------------- |
| `gte`              | inklusive Untergrenze                                               |
| `lte`              | inklusive Obergrenze                                                |
| `missing: exclude` | nur bekannte Werte innerhalb der Grenzen; Default bei aktiver Range |
| `missing: include` | passende bekannte Werte ODER Unknown                                |
| `missing: only`    | ausschliesslich Unknown; `gte` und `lte` sind dann verboten         |

Mindestens eine Grenze ist erforderlich, ausser bei `missing: only`. Es gilt `gte <= lte`. Werte sind nichtnegative JSON-Integer bis zur sicheren JSON-Ganzzahlgrenze und zusätzlich durch die actorbezogenen Capabilities des Feldes begrenzt. Diese nichtnegative Rangeform gilt nur für die drei V1-Felder. Eine künftige Metrik wie `max_elevation_m`, deren valide Werte negativ sein können, verwendet `SignedIntegerRangeV1` und `KnownSignedIntegerV1` mit sicheren vorzeichenbehafteten JSON-Integern; sie darf nicht durch die vorstehende nichtnegative Rangeform oder ein Minimum von `0` abgeschnitten werden. Ein UI-Slidermaximum ist keine implizite API-Grenze: Ein offener letzter Slideranschlag wird durch ausgelassenes `lte` dargestellt.

Der UI-Reset lässt das gesamte Rangefeld aus; JSON-`null` und ein leeres Rangeobjekt bleiben ungültig. Der obere Handle am offenen Domainende entfernt ausschliesslich `lte` und lässt eine vorhandene Untergrenze unverändert. Der offene Bereich `[X, ∞)` entsteht erst, wenn der Request zusätzlich `gte = X` enthält; seine reine Form ist `{ "gte": X, "missing": "exclude" }`. Der Endanschlag allein setzt oder verschiebt `gte` nie.

Allgemein gilt:

- Filter ausgelassen: bekannte und unbekannte Werte bleiben im Suchraum.
- Aktiver Range-Filter: Unknown erfüllt ihn standardmässig nicht.
- `0` ist nur dort ein bekannter Wert, wo die feldspezifische Validitätsregel ihn erlaubt, und niemals Ersatz für Unknown.
- Enumfilter führen `unknown` ausdrücklich als Wert.
- Unknown steht bei jeder Sortierrichtung zuletzt.
- Facetten besitzen nur für fachlich nullable Gruppen einen eigenen Unknown-Zustand; Histogramme besitzen einen getrennten `unknown_bucket` mit stabiler Bucket-ID.
- `absent`, `unknown`, `unmapped` und `ambiguous` werden nie zusammengelegt.

`trail_metrics_v1` besitzt für jede Metrik ein separates Presence-Bit und eine Provenienzrevision. `known` ist ein Wert nur, wenn er entweder in der autoritativen Quelle ausdrücklich vorhanden und mit bekannter Einheit valide ist oder aus einem erfolgreichen, versionierten Analyse-/Geometrielauf mit ausreichender Coverage stammt. Ein historischer Datenbankdefault `0` ohne Presence-Nachweis ist `unknown`. Echtes `0` bleibt bei Aufstieg und Abstieg bekannt, wenn die Quelle es ausdrücklich liefert oder eine valide Berechnung es ergibt; Distanz `0` ist dagegen keine valide bekannte Trailmetrik. Nutzlastwerte werden bei `known = false` nie ausgewertet.

Erst danach quantisiert `trail_metrics_v1` bekannte, endliche Quellwerte: positive Distanz sowie nichtnegative Aufstiegs- und Abstiegswerte werden in SI-Metern, positive Quellzeit in SI-Sekunden jeweils mit `floor(value + 0.5)` auf den nächsten Integer gerundet. Derselbe Integer wird für Filter, Sortierung, Total/Histogramm und DTO verwendet; der Browser rundet nicht erneut. Negative, nicht endliche oder ausserhalb des angebotenen sicheren Bereichs liegende Quellwerte sowie auf `0` quantisierte Distanz oder Zeit werden `unknown` und intern diagnostiziert. Zählwerte wie `local_observed_like_count` sind bereits exakte Integer und werden nicht gerundet. `-0` normalisiert vor der feldspezifischen Validierung zu `0`.

`source_difficulty` verwendet:

```text
easy | moderate | difficult | unknown
```

Eine ungültige oder fehlende Quellschwierigkeit wird in der Projektion `unknown`, nie `easy`.

### 5.7 Datumsvertrag

```json
{
  "from": "2026-09-05",
  "through": "2026-09-06",
  "time_zone": "Europe/Zurich",
  "missing": "exclude"
}
```

- `from` und `through` sind strikte ISO-Kalenderdaten `YYYY-MM-DD`.
- Mindestens eine Datumsgrenze ist erforderlich, ausser bei `missing: only`.
- `time_zone` ist bei einer Grenze erforderlich. Zulässig sind kanonische IANA-Zonen und `UTC`; feste Offsets und Browserabkürzungen sind ungültig.
- Es gilt `from <= through`.
- Beide Grenzen sind aus Produktsicht inklusive.
- Der Server kompiliert sie mit der im Suchkontext gepinnten IANA-TZDB-Revision zu `[start_of_day(from, zone), start_of_day(day_after(through), zone))`. `start_of_day` ist der früheste Instant mit diesem lokalen Kalenderdatum; besitzt eine Zone diesen Kalendertag überhaupt nicht, ist sein Intervall leer.
- DST wird aus der IANA-Zone berechnet; der Server addiert nicht pauschal 86'400 Sekunden.
- Relative Chatangaben werden vor Erstellung der Spec gegen einen expliziten Interpretationszeitpunkt und eine Zeitzone in absolute Daten aufgelöst.
- Bei mindestens einer Grenze ist `missing: exclude` der Default und lässt nur bekannte Daten im Intervall zu. `include` bedeutet passende bekannte Daten ODER Unknown. `only` bedeutet ausschliesslich Unknown; bei `missing: only` sind Datum und Zeitzone verboten.
- `missing: exclude` oder `include` ohne mindestens eine Grenze ist ungültig. Ein ausgelassener gesamter Datumsfilter lässt bekannte und unbekannte Daten zu.

Der projizierte Trailwert ist kein zonenloses Civil Date, sondern ein autoritativer Instant. Er ist nur `known`, wenn die Quelle einen validen RFC-3339-Zeitpunkt oder einen lokalen Zeitpunkt **mit** autoritativer IANA-Zone liefert. Die Projektion normalisiert ihn auf UTC, rundet auf Millisekunden ab und verwendet exakt diesen Instant für Filter, Sortierung und Response. Ein zonenloser Altwert ohne belegbare Zone bleibt in V1 `unknown` beziehungsweise im Legacyprofil, bis eine dokumentierte Migration seine Zone festlegt. `TrailSearchHitV1.trail_date.value` ist deshalb UTC-RFC-3339 mit genau Millisekunden; die UI darf daraus in einer ausdrücklich gewählten Anzeigezone ein Civil Date rendern.

### 5.8 Boolesche Filter

Ein ausgelassener Booleanfilter schränkt nicht ein. Ein vorhandenes `true` oder `false` ist ein exakter Filter. Insbesondere bedeutet in V1 `liked_by_me: false` „nur nicht von mir geliked“; der Legacywert `liked=false` wird dagegen auf „Filter ausgelassen“ migriert.

Actorabhängige Filter ohne Authentifizierung ergeben `authentication_required` und nie eine leere Ergebnismenge.

### 5.9 Startpunkt-Radius

```json
{
  "center": { "lat": 46.8, "lon": 8.2 },
  "radius_m": 10000
}
```

- `center` ist atomar; Latitude und Longitude müssen gemeinsam vorhanden sein.
- Latitude liegt inklusive in `[-90, 90]`, Longitude inklusive in `[-180, 180]`. Nullwerte sind gültig.
- `radius_m` ist eine positive ganze Zahl und darf das in Capabilities angebotene `max_radius_m` nicht überschreiten.
- Koordinaten bezeichnen WGS84 (`EPSG:4326`). `start_point_distance_v1` berechnet die kürzeste Ellipsoid-Geodäte mit dem gepinnten Karney-/GeographicLib-Profil, rundet eine nichtnegative Distanz durch `floor(meters * 1000 + 0.5)` auf Millimeter und vergleicht diesen Integer mit `radius_m * 1000`.
- Das fachliche Prädikat lautet damit `start_point_distance_mm <= radius_m * 1000`; die quantisierte Grenze ist inklusive. Antimeridian und Pole sind normale Geodäsiefälle, keine Bounding-Box-Sonderlogik.
- Ein Trail ohne bekannten Startpunkt erfüllt einen aktiven Radiusfilter nicht.
- Der initiale Webvertrag bietet `default_radius_m = 2000` und `max_radius_m = 10000`; das Requestfeld selbst besitzt keinen Default und verlangt einen Radius.
- `start_point_radius` sagt nichts über den restlichen Routenverlauf. Der durch ADR 0002 angenommene Routenradius verwendet das getrennte, bis zur G2-/G3b-Integration capability-gebundene Feld `route_geometry_radius` und den Spatial-Vertrag `route_radius_ux_v1`. Ein Server fällt niemals zwischen beiden Modi zurück oder deutet einen alten Startpunktradius um.
- Geocoderlabel, Icon und Attribution sind UI-/Anchor-Metadaten und verändern dieses Prädikat nicht.

Ein Backend darf `_geoRadius` zur Kandidatensuche verwenden, muss für die Capability `predicate_accuracy: exact` aber zunächst eine nachweislich vollständige, gegebenenfalls konservativ erweiterte Kandidatenmenge bilden und danach jedes Ergebnis gegen `start_point_distance_v1` finalisieren. Ein reiner Finalizer auf einer Kandidatenmenge mit möglichen False Negatives ist nicht exakt. Ein Legacyradius bleibt `legacy_start_point_radius_v0`, bis Paritätsfixtures seine Treffermenge als identisch beweisen; eine abweichende Grenzsemantik wird nicht automatisch kanonisiert.

### 5.10 Routenradius `route_radius_ux_v1`

Ist die Capability aktiv, hat `route_geometry_radius` dieselbe atomare Transportform aus WGS84-`center` und ganzzahligem `radius_m` wie der Startpunktradius. Es gelten jedoch ein anderes fachliches Prädikat und ein anderer Accuracy-Vertrag:

Die Herkunft des bereits normalisierten `center` ist kein Bestandteil dieses Suchvertrags. Insbesondere setzt `route_geometry_radius` weder `autocomplete.v1`, einen kategorisierten Geocoder noch Photon voraus. Submit-Suche und beliebige providerkonforme `search.v1`-Adapter dürfen denselben Punktanker liefern; Label, Provider und Attribution bleiben Anchor-/UI-Metadaten. `area_geometry.v1` ist nur für einen später separat freizugebenden Regionsvertrag relevant und darf den Punkt-Radius nicht blockieren.

- `0 < radius_m <= 100000`; grössere Werte ergeben `unsupported_radius_for_spatial_contract`.
- Die Capability bietet `default_radius_m = 2000` nur als initialen UI-Wert; der normalisierte Request enthält `radius_m` immer explizit.
- `start_point_radius` und `route_geometry_radius` sind gegenseitig ausgeschlossen. Ihr gemeinsames Auftreten ergibt `invalid_filter_combination`; der Server wählt keinen Modus still aus.
- Fachlich gesucht ist eine Route, sobald mindestens ein reales Segment den Radius berührt. Ausgeführt wird der in ADR 0002 angenommene Direct-Pfad auf der S50-Suchprojektion mit höchstens 5 Kilometer langen Segmenten, Antimeridian-Splitting und `_geoRadius(..., 100)` ohne Padding. Die Meilisearch-Direct-Menge ist das Endergebnis; es gibt keine SQLite-Hydration oder exakte Nachprüfung im Requestpfad.
- Fehlende oder `unavailable` Suchgeometrie erfüllt den aktiven Filter nicht und fällt niemals auf den Startpunkt zurück.
- Die Response liefert `predicate_accuracy: bounded_approximate`, `display_qualifier: none` und den vollständigen `route_radius_ux_v1`-Eintrag gemäss Abschnitt 10. Das 50-Meter-Neutralband ist keine geometrische Worst-Case-Garantie.
- UI und Capability bieten keine Sortierung nach der nächsten Stelle der Route an. Ein entsprechender API-, URL- oder Chat-Versuch ergibt `unsupported_sort_for_spatial_contract` und niemals eine still unsortierte oder nach Startpunktnähe sortierte Erfolgsantwort.
- Der kritische Kern verwendet genau eine föderierte Meilisearch-Multi-Search mit einer semantisch identischen Subquery je Route-Shard des gebundenen Content-Snapshots. Fehlende oder abgeschnittene Shardantworten sind keine partielle Erfolgsmenge.

## 6. Text- und Rankingvertrag `trail_text_v1`

### 6.1 Textmatching

- `text` ist Literaltext. Anführungszeichen, Minuszeichen, Doppelpunkte, Klammern und andere Operatorzeichen aktivieren keine Phrase-, Negations- oder Feldsyntax; sie werden gemäss dem gepinnten Analyzer als Worttrenner behandelt.
- Gross-/Kleinschreibung und kanonisch äquivalente Akzente beeinflussen die Matchberechtigung nicht.
- Alle normalisierten Suchtokens müssen matchen. Der Server entfernt keine Terme, um die Trefferzahl oder Seitengrösse zu erhöhen.
- Nur das letzte Token darf als Präfix matchen.
- Typotoleranz ist für menschliche Textattribute aktiv: 0 Tippfehler bei 1–4 Zeichen, einer bei 5–8 Zeichen, zwei ab 9 Zeichen.
- Eine Transposition benachbarter Zeichen zählt als ein Tippfehler.
- Eine Abweichung am ersten Zeichen verbraucht zwei Tippfehler aus diesem Budget.
- Numerische Tokens, kanonische IDs und interne Schlüssel besitzen keine Typotoleranz.
- V1 verwendet keine freien oder administrativ still veränderbaren Synonyme und keine Stopwordliste. Übersetzte Taxonomiebegriffe und kuratierte Aliase sind versionierte Projektionsdaten, keine Query-Synonyme.
- Phrase-, Negations-, Relaxations- oder Synonymfunktionen benötigen einen neuen Rankingvertrag und eine sichtbare Capability.

Tokenizer-, Normalisierungs- und Typokosten sind Bestandteil des `trail_text_v1`-Profilhashes und werden gegen die gepinnte Engineversion mit Unicode-, Satzzeichen-, Transpositions- und Erstzeichen-Fixtures geprüft. Ein Engineupgrade darf diese Regeln nicht still verändern.

### 6.2 Suchattribute

Die feste Attributpriorität ist:

1. `name`
2. `location`
3. `taxonomy_search_terms`
4. `tags`
5. `waypoint_names`
6. `description`
7. `author_name`

Die Priorität bricht nur Gleichstände nach Wortabdeckung, Tippfehlern und Wortnähe. Ein Match im Namen ist daher keine absolute Ranggarantie.

### 6.3 Relevanzreihenfolge

Für nichtleeren Text gilt:

```text
vollständige Wortabdeckung
Anzahl benötigter Tippfehler ASC
Wortnähe
Attributpriorität
Wortposition
exakter Wortmatch vor Präfixmatch
created_at DESC
trail_id ASC
```

Für leeren Text und `sort.by = relevance` gilt endgültig:

```text
created_at DESC
trail_id ASC
```

Die API liefert keinen einzelnen fachlichen Relevanzscore. Engine-Scores und Score-Details sind weder zwischen Queries vergleichbar noch Teil des öffentlichen DTO.

## 7. Sortiervertrag

Das V1-Requestschema kennt diese Sortkeys:

```text
relevance
name
trail_date
created_at
distance_m
elevation_gain_m
elevation_loss_m
source_duration_s
source_difficulty
local_observed_like_count
route_proximity_m
```

`source_duration_s` und `source_difficulty` sind die heutigen Quellwerte und niemals persönliche ETA oder Eignung. `route_proximity_m` ist im V1-Direct-Vertrag ein absichtlich schema-bekannter **Ablehnungstoken**, kein ausführbarer Sortkey: Er wird nie unter `capabilities.sort.fields` angeboten und erscheint nie in `effective_sort`. Zusammen mit `route_geometry_radius` ergibt er unabhängig von `asc` oder `desc` den typisierten Fehler `unsupported_sort_for_spatial_contract`; ohne diesen Spatialfilter ergibt er `unsupported_capability`. Diese explizite Schemaform erlaubt API, gespeicherter URL und Chat denselben stabilen Fehler, ohne eine unsortierte Erfolgsantwort zu erzeugen.

Regeln:

- Fehlendes `sort` normalisiert zu `{ "by": "relevance" }`.
- Bei `by: relevance` ist `direction` verboten.
- Bei jedem expliziten Feld ist `direction: asc | desc` erforderlich. Es gibt keinen versteckten feldabhängigen Richtungsdefault.
- Ein explizites Feld ist die Primärsortierung. Bei nichtleerem Text bricht `trail_text_v1` nur Gleichstände dieses Feldes.
- Missing und Unknown stehen unabhängig von der Richtung immer zuletzt.
- Schwierigkeit sortiert bekannte Werte `easy < moderate < difficult`; `unknown` folgt zuletzt.
- `name` verwendet den versionierten, locale-unabhängigen `name_sort_key_v1`; Unicode-/Collation-Version sind Teil des Projektionsvertrags.
- `created_at DESC` und abschliessend `trail_id ASC` brechen verbleibende Gleichstände.
- `effective_sort` in der Response legt die tatsächlich angewandte totale Reihenfolge offen.

Jeder `effective_sort`-Schritt ist diskriminiert:

```json
[
  { "kind": "field", "field": "distance_m", "direction": "asc" },
  { "kind": "ranking_profile", "id": "trail_text_v1" }
]
```

Ein Feldschritt darf einen öffentlichen Sortkey ausser `relevance`, den internen Tie-Breaker `trail_id` oder genau eines dieser Presence-Pseudofelder nennen:

```text
trail_date_known
distance_m_known
elevation_gain_m_known
elevation_loss_m_known
source_duration_s_known
source_difficulty_known
```

Ein Presencewert ist intern `1` für bekannt und `0` für Unknown und erscheint immer mit `direction: desc`. `trail_id` erscheint nur `asc`; alle anderen öffentlichen Felder verwenden die effektive Requestrichtung beziehungsweise ihren festgelegten Tie-Breaker. Ein Rankingprofilschritt darf in V1 ausschliesslich `id: trail_text_v1` nennen. `relevance` selbst ist kein Feldschritt.

Bei nichtleerem Text und explizitem Feld folgen auf dessen Presence-/Feldschritte genau der `ranking_profile`-Schritt; dessen Reihenfolge endet bereits mit `created_at DESC, trail_id ASC`. Bei leerem Text folgen stattdessen die noch nicht verwendeten Feldschritte `created_at DESC` und `trail_id ASC`. Für nichtleere Relevanzsortierung besteht `effective_sort` nur aus dem Rankingprofil, für leere Relevanzsortierung aus `created_at DESC, trail_id ASC`.

Ein Legacyrequest behält während der Übergangszeit `legacy_ranking_v0`, falls seine heutige Sortierung Relevanz vor das explizite Feld stellt. Er wird nicht still auf die neue feldprimäre Semantik kanonisiert. Neue V1-Requests verwenden ausschliesslich den oben definierten Vertrag.

## 8. `SearchResponseV1`

Eine erfolgreiche Treffer- und Totalberechnung antwortet mit `200 OK` und `Content-Type: application/json`. V1 verwendet kein `206 Partial Content` und liefert weder partielle Trefferseiten noch nur teilweise berechnete Bucket-Zahlen. Nur ausdrücklich angeforderte Facetten- und Histogrammgruppen dürfen unabhängig `status = error` tragen; in einer Ready-Gruppe ist jeder ausgelieferte Count vollständig berechnet und exakt für den gemeinsamen Suchkontext. Das davon unabhängige Facettenfeld `complete` beschreibt ausschliesslich, ob der Optionskatalog vollständig ausgeliefert wurde.

### 8.1 Vollständiges Beispiel

```json
{
  "contract": "trail-search.v1",
  "normalized_search": {
    "entity_kind": "trail",
    "locale": "de-CH",
    "text": "linde aussicht",
    "filters": {
      "source": { "scope": "all" },
      "taxonomy": {
        "any_of": [
          {
            "category": { "kind": "id", "id": "cat_hiking" },
            "subcategory": {
              "any_of": [
                { "kind": "id", "id": "sub_family" },
                { "kind": "absent" }
              ]
            }
          }
        ]
      },
      "tag_ids": { "any_of": ["tag_panorama"] },
      "source_difficulty": {
        "any_of": ["easy", "moderate", "unknown"]
      },
      "distance_m": {
        "gte": 5000,
        "lte": 20000,
        "missing": "exclude"
      },
      "trail_date": {
        "from": "2026-09-05",
        "through": "2026-09-06",
        "time_zone": "Europe/Zurich",
        "missing": "exclude"
      },
      "start_point_radius": {
        "center": { "lat": 46.8, "lon": 8.2 },
        "radius_m": 10000
      },
      "liked_by_me": true
    },
    "sort": { "by": "distance_m", "direction": "asc" }
  },
  "normalized_search_fingerprint": "sf_opaque",
  "effective_sort": [
    { "kind": "field", "field": "distance_m_known", "direction": "desc" },
    { "kind": "field", "field": "distance_m", "direction": "asc" },
    { "kind": "ranking_profile", "id": "trail_text_v1" }
  ],
  "hits": [
    {
      "trail": {
        "id": "trail_01J6E7Q8",
        "entity_kind": "trail",
        "url": "/trails/trail_01J6E7Q8",
        "name": "Zur Linde am Aussichtspunkt",
        "description_excerpt": "Kurze Rundwanderung oberhalb des Sees.",
        "location": "Zürich",
        "distance_m": { "state": "known", "value": 8340 },
        "elevation_gain_m": { "state": "known", "value": 410 },
        "elevation_loss_m": { "state": "known", "value": 410 },
        "source_duration_s": { "state": "unknown" },
        "source_difficulty": "moderate",
        "trail_date": {
          "state": "known",
          "value": "2026-09-05T10:30:00.000Z"
        },
        "created_at": "2026-08-20T16:05:14.123Z",
        "completed": false,
        "taxonomy": {
          "category": {
            "state": "mapped",
            "id": "cat_hiking",
            "name": "Wandern",
            "icon": "hiking"
          },
          "subcategory": {
            "state": "mapped",
            "id": "sub_family",
            "name": "Familienwanderung",
            "icon": "family"
          }
        },
        "tags": [{ "id": "tag_panorama", "name": "Panorama" }],
        "author": {
          "id": "actor_01J6",
          "display_name": "Mara",
          "handle": "mara@example.org",
          "avatar_url": "/api/v1/avatars/actor_01J6"
        },
        "access": "public",
        "source": {
          "kind": "federated",
          "origin_instance_id": "instance_example_org",
          "origin_instance_label": "example.org"
        },
        "local_observed_like_count": 17,
        "liked_by_me": true,
        "has_photos": true,
        "thumbnail": {
          "url": "/api/v1/trails/trail_01J6E7Q8/thumbnail",
          "alt": "Blick vom Weg auf den See"
        }
      },
      "match": {
        "profile": "trail_highlight_v1",
        "fragments": [
          {
            "field": "name",
            "prefix_omitted": false,
            "suffix_omitted": false,
            "segments": [
              { "text": "Zur ", "matched": false },
              { "text": "Linde", "matched": true },
              { "text": " am ", "matched": false },
              { "text": "Aussicht", "matched": true },
              { "text": "spunkt", "matched": false }
            ]
          }
        ],
        "reasons": []
      }
    }
  ],
  "total": 180,
  "page": {
    "size": 1,
    "returned": 1,
    "next_cursor": "eyJvcGFxdWUiOiJjdXJzb3IifQ"
  },
  "aggregation_status": "complete",
  "facets": [
    {
      "field": "source_difficulty",
      "status": "ready",
      "complete": true,
      "buckets": [
        {
          "bucket_id": "facet.source_difficulty.v1/easy",
          "value": "easy",
          "label": "Leicht",
          "selected": true,
          "visible": true,
          "count": 60
        },
        {
          "bucket_id": "facet.source_difficulty.v1/moderate",
          "value": "moderate",
          "label": "Mittel",
          "selected": true,
          "visible": true,
          "count": 103
        },
        {
          "bucket_id": "facet.source_difficulty.v1/difficult",
          "value": "difficult",
          "label": "Schwer",
          "selected": false,
          "visible": true,
          "count": 51
        },
        {
          "bucket_id": "facet.source_difficulty.v1/unknown",
          "value": "unknown",
          "label": "Unbekannt",
          "selected": true,
          "visible": true,
          "count": 17
        }
      ]
    }
  ],
  "histograms": [
    {
      "field": "distance_m",
      "status": "ready",
      "complete": true,
      "bucket_profile": {
        "id": "distance_m_web_v1",
        "hash": "sha256:opaque-base64url"
      },
      "buckets": [
        {
          "bucket_id": "distance_m.v1/range/0/2000",
          "kind": "range",
          "from_inclusive": 0,
          "to_exclusive": 2000,
          "count": 4
        },
        {
          "bucket_id": "distance_m.v1/range/2000/5000",
          "kind": "range",
          "from_inclusive": 2000,
          "to_exclusive": 5000,
          "count": 27
        },
        {
          "bucket_id": "distance_m.v1/range/5000/10000",
          "kind": "range",
          "from_inclusive": 5000,
          "to_exclusive": 10000,
          "count": 84
        },
        {
          "bucket_id": "distance_m.v1/range/10000/15000",
          "kind": "range",
          "from_inclusive": 10000,
          "to_exclusive": 15000,
          "count": 55
        },
        {
          "bucket_id": "distance_m.v1/range/15000/20000",
          "kind": "range",
          "from_inclusive": 15000,
          "to_exclusive": 20000,
          "count": 42
        },
        {
          "bucket_id": "distance_m.v1/range/20000/30000",
          "kind": "range",
          "from_inclusive": 20000,
          "to_exclusive": 30000,
          "count": 15
        },
        {
          "bucket_id": "distance_m.v1/range/30000/50000",
          "kind": "range",
          "from_inclusive": 30000,
          "to_exclusive": 50000,
          "count": 9
        },
        {
          "bucket_id": "distance_m.v1/overflow/50000",
          "kind": "overflow",
          "from_inclusive": 50000,
          "to_exclusive": null,
          "count": 0
        }
      ],
      "unknown_bucket": {
        "bucket_id": "distance_m.v1/unknown",
        "kind": "unknown",
        "count": 7
      }
    }
  ],
  "search_context": {
    "contract": "trail-search.v1",
    "context_id": "sc_opaque",
    "as_of": "2026-08-29T12:34:56.123Z",
    "valid_until": "2026-08-29T12:39:56.123Z",
    "capability_revision": "search-capabilities-42",
    "time_zone_database_revision": "tzdb-opaque",
    "content_snapshot": {
      "mode": "revision_fenced",
      "id": "cs_opaque",
      "index_generation_id": "trails-g17",
      "delivered_catalog_revision": "184467",
      "security_watermark": "184467",
      "content_epoch": "2074"
    },
    "ranking_profile": {
      "id": "trail_text_v1",
      "hash": "sha256:opaque-base64url"
    },
    "sort_profile": {
      "id": "trail_sort_v1",
      "hash": "sha256:opaque-base64url"
    },
    "metric_profile": {
      "id": "trail_metrics_v1",
      "hash": "sha256:opaque-base64url"
    },
    "aggregation_profiles": [
      {
        "kind": "facet",
        "field": "source_difficulty",
        "id": "source_difficulty_facet_v1",
        "hash": "sha256:opaque-base64url",
        "options_revision": "source-difficulty-options-v1",
        "sort_profile": {
          "id": "source_difficulty_options_sort_v1",
          "hash": "sha256:opaque-base64url"
        }
      },
      {
        "kind": "histogram",
        "field": "distance_m",
        "id": "distance_m_web_v1",
        "hash": "sha256:opaque-base64url",
        "unit": "m",
        "open_from": 50000,
        "statistics_domains": [
          {
            "activity_family": "all",
            "source_scope": { "scope": "all" },
            "catalog_visibility": "public_discoverable",
            "domain_source": "shipped_default",
            "open_from": 50000,
            "revision": "distance-domain-v1-default"
          }
        ]
      }
    ],
    "projection_profile": {
      "id": "trail_hit_v1",
      "hash": "sha256:opaque-base64url"
    },
    "acl_policy_revision": "acl-v1",
    "category_preference_revision": "cat-pref-9",
    "source_scope": { "scope": "all" },
    "spatial_contracts": [
      {
        "kind": "start_point_radius",
        "id": "start_point_radius_v1",
        "distance_profile": "start_point_distance_v1",
        "predicate_accuracy": "exact",
        "max_radius_m": 10000
      }
    ],
    "federation": {
      "universe": "locally_materialized_verified_snapshots",
      "snapshot_set_revision": "fed-733",
      "live_remote_queries": false
    },
    "accuracy": {
      "result_set_computation": "exhaustive",
      "count_computation": "exact_for_result_set",
      "predicate_accuracy": "exact",
      "display_qualifier": "none",
      "contracts": []
    }
  },
  "warnings": []
}
```

### 8.2 Top-Level-Felder

| Feld                            | Typ                                                    | Garantie                                                                                   |
| ------------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `contract`                      | String                                                 | exakt `trail-search.v1`                                                                    |
| `normalized_search`             | `TrailSearchSpecV1`                                    | kanonische, tatsächlich ausgeführte Spec                                                   |
| `normalized_search_fingerprint` | opaker String                                          | stabil für dieselbe kanonische Spec und Vertragsrevision; nicht clientseitig nachbauen     |
| `effective_sort`                | nichtleere Liste aus Feld- oder Rankingprofilschritten | vollständige totale Sortierung einschliesslich Presence- und Tie-Breakern                  |
| `hits`                          | Liste von `TrailSearchHitV1`                           | höchstens `page.size`, in `effective_sort`-Reihenfolge                                     |
| `total`                         | nichtnegative Ganzzahl                                 | exakte Kardinalität der durch den angegebenen Spatial-Vertrag berechneten Treffermenge     |
| `page`                          | Objekt                                                 | Seitengrösse, tatsächlich gelieferte Anzahl und Folgeseiten-Cursor                         |
| `aggregation_status`            | Enum                                                   | `complete`, wenn alle angeforderten Gruppen `ready` sind, sonst `partial`                  |
| `facets`                        | Liste                                                  | genau die angeforderten Facetten als Ready-/Error-Union, in kanonischer Feldreihenfolge    |
| `histograms`                    | Liste                                                  | genau die angeforderten Histogramme als Ready-/Error-Union, in kanonischer Feldreihenfolge |
| `search_context`                | Objekt                                                 | gemeinsamer fachlicher Stand dieser gesamten Response                                      |
| `warnings`                      | Liste                                                  | maschinenlesbare, nicht fatale Hinweise; niemals Ersatz für einen Fehler                   |

`normalized_search` materialisiert `entity_kind`, `locale`, `text`, `sort` und `filters.source`; neutrale übrige Filter bleiben ausgelassen. Bei aktiven Range- und Datumsfiltern wird `missing`, bei Source wird `scope` immer materialisiert. Mengen sind sortiert und redundante Taxonomiezweige entfernt. Diese Darstellung, nicht das Eingabe-JSON, ist Grundlage des Fingerprints.

`page.returned` ist exakt `hits.length` und, ausser auf der letzten Seite, exakt `page.size`. `page.next_cursor` ist ein String, wenn weitere Treffer existieren, andernfalls ausdrücklich `null`. V1 besitzt keinen davon getrennten Traversierungs-Cap: Für die vollständige durch `total` bezeichnete Treffermenge muss jede nicht letzte Seite einen Cursor liefern. Ein Engine-`maxTotalHits`, Offsetlimit oder Ressourcenbudget darf weder `next_cursor = null` vortäuschen noch `total` verkleinern; kann der aktive Execution-Pfad die vollständige Traversierung nicht liefern, ist `trail-search.v1` für diesen Pfad nicht releasefähig. V1 liefert keine Seitenzahl und keine aus `total / size` abgeleitete Gesamtseitenzahl.

Ein Warning besitzt den Pflichtcode `code`, einen lokalisierten Pflichttext `message` und optional `path` als JSON Pointer. Clients steuern nur über `code`, ignorieren unbekannte Warningcodes und dürfen einen Warningtext generisch anzeigen. Normale V1-Suchen liefern derzeit keine semantischen Warnings; die in Abschnitt 15 definierten nicht blockierenden Migrationhinweise stammen vom Legacyadapter. Accuracy, Unknown und eine leere Treffermenge sind keine Warnings.

## 9. Treffer- und Highlight-DTO

### 9.1 Erlaubte Trailfelder

`TrailSearchHitV1` besteht ausschliesslich aus `trail` und `match`. `trail_hit_v1` ist eine serverseitige Allowlist und kein Durchreichen des Suchindexdokuments. Zulässige Felder sind:

| Feld                                                                      | Typ und Semantik                                                                                                |
| ------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `id`, `entity_kind`, `url`, `name`                                        | nichtleere Strings; `url` ist eine lokale, autorisierte Detailroute                                             |
| `description_excerpt`, `location`                                         | sichtbare Anzeigetexte; fehlender Text ist `""`                                                                 |
| `distance_m`, `elevation_gain_m`, `elevation_loss_m`, `source_duration_s` | `KnownIntegerV1`                                                                                                |
| `source_difficulty`                                                       | `easy \| moderate \| difficult \| unknown`                                                                      |
| `trail_date`                                                              | `KnownInstantV1`                                                                                                |
| `created_at`                                                              | UTC-RFC-3339 mit genau Millisekunden; immer bekannt                                                             |
| `completed`                                                               | sichtbarer, nicht actorbezogener Abschlusswert des Trails                                                       |
| `liked_by_me`                                                             | für den aktuellen Actor beobachteter lokaler Like; anonym immer `false`                                         |
| `has_photos`                                                              | optionales Capability-Feld: mindestens ein für den aktuellen Actor sichtbares Foto; nie rohe Assetpräsenz       |
| `taxonomy`                                                                | strukturierte Kategorie und Subkategorie mit getrenntem Mappingzustand                                          |
| `tags`                                                                    | nach Anzeigelabel sortierte Liste aus stabiler ID und sichtbarem Namen                                          |
| `author`                                                                  | sichtbare opake ID, Anzeigename, Handle und lokale Avatar-URL; fehlende optionale Strings sind `""`             |
| `access`                                                                  | `public \| owned_private \| shared_with_me`                                                                     |
| `source`                                                                  | `local` oder `federated`; bei Federation zusätzlich normalisierte Origin-ID und sichtbares Label                |
| `local_observed_like_count`                                               | nichtnegative Ganzzahl; kein behauptetes globales Popularitätsmass                                              |
| `thumbnail`                                                               | optionales Capability-Feld: sichtbares `{url, alt}` oder `null`, wenn kein freigegebenes Vorschaubild existiert |

Für anonyme Suchen ist `liked_by_me` immer `false`; ein entsprechender Filter ist ohne Authentifizierung trotzdem ungültig. `completed` bleibt der sichtbare Trailwert und ist nicht actorbezogen. Ein späterer persönlicher Abschlussstatus erhält ein eigenes Feld und überschreibt `completed` nicht.

`created_at` wird vor Sortierung und Ausgabe auf UTC normalisiert und auf die Millisekunde abgerundet; derselbe Wert dient als Tie-Breaker. Submillisekunden dürfen nicht nur im Index verbleiben und dadurch eine von der Response unsichtbare Reihenfolge erzeugen.

Bei `source.kind = local` sind Origin-Felder verboten; bei `federated` sind `origin_instance_id` und `origin_instance_label` erforderlich. Tags werden nach lokalisiertem Anzeigenamen und bei Gleichstand nach ID sortiert. `has_photos` und `thumbnail` werden erst nach der Actor-sicheren Assetprojektion gemeinsam freigegeben; vorher fehlen die Filter-/Response-Capability und beide Felder. Ihre Abwesenheit bedeutet „nicht unterstützt“, nicht `false` und nicht „kein Asset“.

Semantische Unknown-Werte verwenden diskriminierte Objekte und nie erfundene Nullwerte:

```json
{
  "known_integer": { "state": "known", "value": 410 },
  "unknown_integer": { "state": "unknown" },
  "known_instant": {
    "state": "known",
    "value": "2026-09-05T10:30:00.000Z"
  },
  "unknown_instant": { "state": "unknown" }
}
```

`KnownIntegerV1` ist exakt die Union `{state: known, value: <nichtnegative sichere JSON-Ganzzahl>}` oder `{state: unknown}`; `value` ist im Unknownzweig verboten. `KnownInstantV1` ist exakt `{state: known, value: <UTC-RFC-3339 mit genau Millisekunden>}` oder `{state: unknown}` mit derselben Feldregel. Innerhalb von V1 verändern zusätzliche Felder diese Diskriminierung nie; Clients ignorieren sie gemäss Response-Forward-Compatibility.

Der künftige `KnownSignedIntegerV1` ist davon getrennt exakt `{state: known, value: <sichere vorzeichenbehaftete JSON-Ganzzahl>}` oder `{state: unknown}`. `SignedIntegerRangeV1` übernimmt dieselbe diskriminierte `gte`-/`lte`-/`missing`-Semantik wie Abschnitt 5.6, erlaubt aber capability-gebundene negative Grenzen. `max_elevation_m` verwendet beide Typen erst mit einem eigenen Metrik-/Histogrammprofil; die nichtnegative Validierungsregel von `trail_metrics_v1` darf darauf nicht angewandt werden.

`null` ist in der Response nur dort zulässig, wo das Schema eine reine Anzeigeabwesenheit (`thumbnail`), eine offene Histogrammgrenze oder den Kontrollsentinel `page.next_cursor` bezeichnet. Es bedeutet nie fachliches Unknown.

Taxonomieknoten verwenden exakt einen der folgenden Zustände:

```json
[
  {
    "state": "mapped",
    "id": "cat_hiking",
    "name": "Wandern",
    "icon": "hiking"
  },
  { "state": "absent" },
  { "state": "unmapped", "source_label": "Randonnée" },
  { "state": "ambiguous", "source_label": "Tour" }
]
```

`id`, `name` und `icon` sind nur bei `mapped` zulässig; `source_label` ist nur bei `unmapped` und `ambiguous` optional und wird nur ausgeliefert, wenn der vorhandene Quellwert für den Actor sichtbar ist. Bei `absent` ist `source_label` verboten. Ein Mappingzustand darf keine interne Kandidatenliste offenlegen.

`state: absent` stammt immer aus dem jeweiligen `presence = absent`. `state: unmapped | ambiguous` stammt nur aus `presence = present` und dem entsprechenden Mappingstatus. Die internen Presence- und Mappingfelder müssen im Treffer-DTO nicht zusätzlich ausgeliefert werden; ihre getrennte Projektion bleibt aber für Filter und Counts normativ.

Insbesondere verboten sind rohe Indexfelder, freie ACL-Ausdrücke, Share- oder Like-Actorlisten, interne Tenant-/Overlay-IDs, GPX-/Polyline-Daten, Engine-Rankingwerte und nicht allowlistete denormalisierte Suchtexte. Ein neues Feld benötigt einen neuen oder additiv erweiterten Projektionsvertrag und einen ACL-Test.

### 9.2 Highlights und Matchgründe

Bei `include.highlights = false` ist `match`:

```json
{ "profile": "trail_highlight_v1", "fragments": [], "reasons": [] }
```

Dasselbe gilt bei leerem Suchtext. Bei aktivierten Highlights gelten:

- Die API liefert strukturierte Textsegmente, niemals HTML, Markup oder unverändertes Engine-`_formatted`.
- `segments[].text` ist gültiges UTF-8 und wird vom Client ausschliesslich als Text gerendert.
- Es gibt keine leeren Segmente; benachbarte Segmente mit demselben `matched`-Wert werden zusammengeführt.
- Das Zusammenfügen aller Segmente ergibt exakt den ausgelieferten Ausschnitt. Schnitte liegen an Graphemgrenzen.
- `matched: true` umfasst gemäss Textprofil auch Präfix- und Typotreffer und behauptet keine exakte Zeichenübereinstimmung.
- Höchstens drei Fragmente und 480 Grapheme werden pro Hit geliefert. `name`, `location`, `tags` und `author_name` erhalten höchstens 160, `description` höchstens 240 Grapheme pro Fragment.
- Die Fragmentreihenfolge folgt der Suchattributpriorität. Positionen werden gegen den exakt indexierten sichtbaren Quellwert validiert; ein inkonsistentes Fragment wird ausgelassen.
- Ein Match auf einer sichtbaren Taxonomieübersetzung kann einen strukturierten Grund mit `kind: taxonomy`, sichtbaren IDs und aktuellem Anzeigelabel erzeugen. Ein unsichtbarer Alias oder enger sichtbarer Waypoint erzeugt weder Text, Grund noch Rangbeitrag.

Zulässige Reasons sind exakt diese Varianten:

```json
[
  {
    "kind": "taxonomy",
    "level": "category",
    "id": "cat_hiking",
    "display_label": "Wandern"
  },
  {
    "kind": "taxonomy",
    "level": "subcategory",
    "id": "sub_family",
    "parent_category_id": "cat_hiking",
    "display_label": "Familienwanderung"
  },
  {
    "kind": "waypoint",
    "waypoint_id": "waypoint_01J6",
    "display_label": "Aussichtspunkt"
  }
]
```

Bei `taxonomy/category` ist `parent_category_id` verboten, bei `taxonomy/subcategory` erforderlich. Taxonomiegründe werden in V1 nur für einen sichtbaren gemappten Knoten geliefert; ein Match auf `unmapped`, `ambiguous` oder einem nicht sichtbaren Alias liefert keinen Grund. Waypoint-ID und -Label sind nur bei Sichtbarkeit des konkreten Waypoints zulässig. Reasons sind nach Suchattributpriorität und ID sortiert, je Variantenschlüssel eindeutig und auf drei pro Hit begrenzt. Freie Engine-Erklärungen oder weitere Felder sind keine V1-Semantik.

## 10. Total, Facetten, Histogramme und Genauigkeit

### 10.1 Gemeinsame Grundmenge

`total`, alle Facetten, alle Histogramme und `hits` verwenden denselben Suchkontext, denselben Actor, dieselben ACL-/Restriction-Fences, dieselbe normalisierte Spec und denselben Spatial-Vertrag. Treffer, Total, Seite und Suchkontext sind der kritische Kern: Scheitert einer davon, folgt eine normale Fehlerantwort. Eine angeforderte Facetten- oder Histogrammgruppe darf unabhängig scheitern; sie erscheint dann an ihrer kanonischen Listenposition mit `status: error`, einem sicheren Fehlerobjekt und ohne `buckets`, `complete` oder Count. Andere Gruppen derselben `200`-Antwort liefern nur vertragstreu vollständig berechnete Bucket-Zahlen und dürfen weder alte noch aus einer Teilmenge extrapolierte Werte verwenden.

`aggregation_status` ist `complete`, wenn jede angeforderte Gruppe `status: ready` besitzt, andernfalls `partial`; ohne angeforderte Gruppen ist es ebenfalls `complete`. Dieser Top-Level-Status beschreibt nur den Berechnungserfolg und ist unabhängig vom Facettenfeld `complete`. Eine Error-Gruppe hat exakt die Form `{field, status: error, error}`. `error` enthält nur `code` und `retryable`; zulässige Codes sind `count_timeout`, `count_backend_unavailable`, `count_query_budget_exceeded`, `count_facet_not_exhaustive` und `count_unsupported_context`. Sie bestätigen weder verborgene Treffer noch interne Engine-, Principal-, Index- oder Filterdaten. `retryable` empfiehlt einen neuen vollständigen Request nach Backoff; es verspricht keine Wiederverwendung eines inzwischen abgelaufenen Kontexts.

`total` und jeder Bucket einer Ready-Gruppe sind exhaustiv für die berechnete Treffermenge. `estimatedTotalHits`, Sampling und ein stilles `1'000`-Treffer-Cap sind für zugesagte Werte verboten. Die Response unterscheidet dabei zwei Aussagen:

- `count_computation: exact_for_result_set`: Die Zahl ist die exakte Kardinalität der tatsächlich berechneten Menge.
- `predicate_accuracy`: Das zugrunde liegende fachliche Prädikat ist `exact` oder eine dokumentierte `bounded_approximate` Berechnung.

Bei `bounded_approximate` bleiben Total und alle Ready-Buckets dennoch exakte Zahlen für ihre jeweils berechnete Direct-Menge. Der Begriff bezeichnet für `route_radius_ux_v1` einen statistisch gegen das festgelegte Qualifikationscorpus geprüften Produktvertrag. Er verspricht weder pro Anfrage eine maximale Trefferabweichung noch eine geometrische Distanz- oder Hausdorff-Grenze. `route_radius_ux_v1` verwendet deshalb `display_qualifier: none`: Wanderer setzt nicht vor jede Zahl ein `≈`, sondern dokumentiert die mögliche räumliche Abweichung einmal zentral in UI und API. Eine Error-Gruppe zeigt `—` und enthält keine Teilzahl. Die Approximation ist keine Schätzung durch Sampling. Der konkrete Spatial-Vertrag und seine Version werden additiv unter `accuracy.contracts` ausgeliefert und durch den Cursor gebunden. Federation definiert das Suchuniversum und ist keine Genauigkeitsstufe.

Die Direct-Releasequalifikation MUSS diese beiden Aussagen getrennt prüfen. Ohne Fehlertoleranz MUSS `total` sowie jeder ausgelieferte Facetten- und Histogrammcount einer Ready-Gruppe der Kardinalität einer unabhängigen Vollmengenauswertung desselben Direct-Prädikats im jeweiligen Filter-, Selektor- beziehungsweise Bucketkontext entsprechen. Eine Abweichung verletzt `exact_for_result_set` unmittelbar und DARF NICHT in das nachfolgende Approximationsbudget eingehen. Die Vollmengenauswertung ist ein Qualifikations-Oracle und verlangt keine Runtime-Materialisierung aller IDs im produktiven Request.

Danach MUSS die räumliche Prädikatsqualität gegen eine unabhängige mathematisch exakte Oracle-Auswertung geprüft werden. Der fachliche Zählauftrag bleibt vollständig gleich; nur das Direct-Raumprädikat wird durch die exakte Auswertung derselben Anker-, Radius- und Segmentbedingung auf der kanonischen Rohgeometrie ersetzt. Für denselben Actor, Snapshot, nicht-räumlichen Filterkontext und gegebenenfalls denselben disjunktiven Selektor beziehungsweise Histogrammbucket bezeichnen `got` den exhaustiven Direct-Count und `oracle` den Count der exakten Oracle-Menge. Pro Beobachtung gelten:

```text
absolute_error = |got - oracle|
relative_error = |got - oracle| / max(1, oracle)
```

Das durch ADR 0002 angenommene G1-Count-Gate besitzt genau eine Messreihe für
`total`; ihr p95 wird nach aufsteigender Sortierung der `n` Werte durch lineare
Interpolation an der nullbasierten Position `0.95 × (n − 1)` bestimmt und
erfüllt `p95(relative_error) <= 0.01`. Facetten- und Histogrammcounts waren
nicht Teil der G1-Evidenz. Führt SRCH3 eine Ready-Gruppe ein, MUSS deren eigene
Qualifikation zusätzlich getrennte Reihen `(field, bucket_id)` für jede
ausgelieferte Facettenoption mit `visible: true` und jeden Histogrammbucket
einschliesslich `unknown_bucket` enthalten. SRCH3 legt vor dieser Qualifikation
ein für spärliche und häufige Buckets sinnvolles Budget, die erforderliche
Stichprobengrösse und die Aggregationsregel fest; G1 schreibt dafür weder die
Ein-Prozent-Schwelle noch ein implizites 100-Prozent-Gate fort. Diese spätere
Qualifikation erweitert die Evidenz für die neue Count-Capability und wird
nicht rückwirkend zur G1-Abnahme von GeoJSON Direct erklärt.
`absolute_error`, exakte Einzelfalltreffer und countbezogene False-Empty-Fälle
bleiben Diagnosen; False-Empty-Anfragen werden zusätzlich durch das
aggregierte Non-Empty-Gate von ADR 0002 bewertet. Die Toleranz qualifiziert
ausschliesslich `predicate_accuracy: bounded_approximate`. Sie erlaubt weder
geschätzte Counts innerhalb der Direct-Menge noch unvollständige
Distributionen.

V1 erlaubt für `accuracy` exakt `result_set_computation: exhaustive` und `count_computation: exact_for_result_set`. Die beiden zulässigen diskriminierten Formen sind:

```json
[
  {
    "result_set_computation": "exhaustive",
    "count_computation": "exact_for_result_set",
    "predicate_accuracy": "exact",
    "display_qualifier": "none",
    "contracts": []
  },
  {
    "result_set_computation": "exhaustive",
    "count_computation": "exact_for_result_set",
    "predicate_accuracy": "bounded_approximate",
    "display_qualifier": "none",
    "contracts": [
      {
        "kind": "spatial_boundary",
        "id": "route_radius_ux_v1",
        "predicate": "route_geometry_distance_lte",
        "plan": "direct",
        "max_radius_m": 100000,
        "product_boundary_tolerance_m": 50,
        "resolution": 100,
        "geometry_profile": "route_geometry_search_v1",
        "simplification_profile": "rdp_s50_v1",
        "max_segment_length_m": 5000
      }
    ]
  }
]
```

`contracts` ist somit immer vorhanden: im exakten Zweig leer, im approximativen nichtleer. Die initiale approximative Variante verlangt alle gezeigten Felder; Integer sind positiv. `product_boundary_tolerance_m` ist die neutrale Produkt- und Auswertungszone aus ADR 0002, ausdrücklich keine geometrische Worst-Case- oder Hausdorff-Garantie. S50, r100 und ihre Kombination veröffentlichen deshalb keinen erfundenen `computed_algorithmic_error_bound_m`. Contracts werden nach `kind`, dann `id` sortiert und vollständig durch den Cursor gebunden. Sind mehrere Prädikate aktiv, bestimmt das schwächste `predicate_accuracy`; eine automatische Kennzeichnung jeder Countzahl folgt daraus nicht. Ein unbekannter oder untypisierter Accuracy-Vertrag ist keine degradierbare Erfolgsantwort, sondern `unsupported_capability`.

### 10.2 Disjunktive Facetten

Eine Facettenzahl bedeutet: Anzahl der Treffer, wenn alle aktiven Filter ausser der jeweils beschriebenen Facettengruppe gelten und der Bucket als Kandidat eingesetzt wird. Freitext, Source, ACL, Datum, Geo und alle anderen Gruppen bleiben wirksam.

Eine erfolgreiche flache Facette verwendet `{field, status: ready, complete, buckets}`. Jeder Bucket besitzt `bucket_id: BucketIdV1`, `value`, lokalisiertes `label`, `selected`, `visible` und den nichtnegativen Integer `count`. `visible` beschreibt die normale Darstellung nach UI-/Taxonomiepräferenzen, nie ACL; eine explizit gewählte ausgeblendete Option bleibt mit `selected: true, visible: false` wirksam und als entfernbarer Chip sichtbar. Für `tags` ist `value` eine stabile Tag-ID, für Access und Difficulty der jeweilige Enumwert. Difficulty folgt `easy, moderate, difficult, unknown`, Access `public, owned_private, shared_with_me`; Tags folgen Label und bei Gleichstand ID. Es gibt keinen labelbasierten Requestwert.

Der Server vereinigt die exhaustive Engineverteilung mit dem autoritativen, für den Actor sichtbaren Optionskatalog und allen normalisierten Auswahlen. Eine ausgewählte gültige Option erscheint deshalb auch mit `count: 0` und `selected: true`; sie bleibt im UI abwählbar. Eine fehlende Engineoption darf nur bei nachgewiesener Exhaustivität oder einer einzigen gebündelten, gruppenweiten Beweisabfrage zu `0` werden; eine Query je Option ist verboten. `complete: true` bedeutet, dass der angebotene Optionsraum vollständig ist. Bei `complete: false`, beispielsweise einer begrenzten Tagliste, sind alle ausgelieferten Counts exakt, aber eine nicht ausgelieferte Option ist niemals implizit `0`; jede ausgewählte Option wird trotzdem injiziert oder die Gruppe wird `count_facet_not_exhaustive`. Nur eine nicht ausgewählte Option aus einer Ready-Gruppe mit belastbarem `count: 0` darf die UI deaktivieren.

- Bei flachen Gruppen wie Schwierigkeit, Tags oder Access wird der gesamte Filter derselben Gruppe entfernt und genau der Bucket eingesetzt.
- Ein Kategorie-Bucket entfernt den vollständigen aktiven Taxonomieausdruck und setzt genau die Kandidatenkategorie ein.
- Ein Subkategorie-Bucket behält die ausgewählten Kategorien, entfernt deren Kindbedingungen und setzt den Kandidaten unter seinem Parent ein. Andere ausgewählte Kategoriebranches ohne Kindbedingung bleiben als OR erhalten. Ist der Parent noch nicht ausgewählt, wird nur `Parent AND Kandidat` gezählt.
- Taxonomie liefert, wo anwendbar, getrennte `absent`-/`unmapped`-/`ambiguous`-Buckets; Difficulty liefert `unknown`. Tags und Access erhalten keinen erfundenen Unknown-Bucket. Künftige nullable Facetten müssen ihren Zustand ausdrücklich in Capabilities deklarieren.
- Eine Facettenresponse enthält stabile Bucket-IDs und zusätzlich die autoritativen Werte beziehungsweise Selektoren; der Client leitet weder IDs noch Requestwerte aus Labels ab.

Die Taxonomiebuckets verwenden exakt diese schema-versionierten IDs:

```text
tax.v1/category/<category-id>
tax.v1/category_absent
tax.v1/category_mapping/unmapped
tax.v1/category_mapping/ambiguous
tax.v1/subcategory/<category-id>/<subcategory-id>
tax.v1/subcategory_absent/<category-id>
tax.v1/subcategory_mapping/unmapped/<category-id>
tax.v1/subcategory_mapping/ambiguous/<category-id>
```

Auch diese IDs sind opak. Kategorie-Zustandsbuckets besitzen keinen Parent; jeder Subkategorie-Zustandsbucket liefert dagegen den typisierten gemappten `parent_category` zusätzlich zur ID.

Kategorie und Subkategorie sind getrennte angeforderte Gruppen, weil ihre Disjunktionskontexte und Fehler unabhängig sind. Ihre Ready-Formen lauten beispielsweise:

```json
[
  {
    "field": "taxonomy/category",
    "status": "ready",
    "complete": true,
    "buckets": [
      {
        "bucket_id": "tax.v1/category/cat_hiking",
        "selector": { "kind": "id", "id": "cat_hiking" },
        "label": "Wandern",
        "selected": true,
        "visible": true,
        "count": 142
      }
    ]
  },
  {
    "field": "taxonomy/subcategory",
    "status": "ready",
    "complete": true,
    "buckets": [
      {
        "bucket_id": "tax.v1/subcategory_absent/cat_hiking",
        "parent_category": { "kind": "id", "id": "cat_hiking" },
        "selector": { "kind": "absent" },
        "label": "Ohne Subkategorie",
        "selected": true,
        "visible": true,
        "count": 19
      }
    ]
  }
]
```

Gemappte Kategorien folgen dem capability-gebundenen administrativen `sort_order` und bei Gleichstand ihrer ID. Subkategoriebuckets werden zuerst nach ihrem Parent in derselben Reihenfolge sortiert; Zustandsknoten folgen je Parent nach gemappten Kindern in der Reihenfolge `absent`, `unmapped`, `ambiguous`.

### 10.3 Histogramme

Eine erfolgreiche Histogrammgruppe verwendet `{field, status: ready, complete: true, bucket_profile, buckets, unknown_bucket}`. `bucket_profile` ist stets `{id, hash}` und bindet den unveränderlichen Profilnamen sowie den Hash seiner registrierten Grenzen und Semantik; Capabilities, Response und `search_context.aggregation_profiles` liefern dieselben Werte. Numerische Buckets besitzen `bucket_id: BucketIdV1`, `kind = range | overflow`, einen nichtnegativen Integer `count` und sind überlappungsfrei sowie vollständig aufsteigend. Range-Buckets haben die Form `[from_inclusive, to_exclusive)`; nur der letzte Bucket darf als `kind: overflow` ein `to_exclusive: null` besitzen. `unknown_bucket` hat `kind: unknown`, eine eigene stabile ID und einen Count, aber keine Grenzen. Er gehört zu keinem numerischen Bucket.

Aktive Filter derselben Rangegruppe werden für das Histogramm entfernt; alle übrigen Filter bleiben wirksam. Bucketgrenzen sind niemals aus dem aktuellen Resultat abgeleitete Zufallsmaxima. Eine Änderung der registrierten Grenzleiter erzeugt ein neues Bucketprofil; das kontextabhängige Zusammenfassen unveränderter Feinbuckets bei `open_from` nicht. Eine Ready-Gruppe enthält höchstens 16 sichtbare Buckets insgesamt: `buckets.length + 1` für den stets vorhandenen `unknown_bucket` darf `max_histogram_buckets_per_group` nicht überschreiten. Kann der Server die exhaustive Verteilung einschliesslich Unknown nicht beweisen, ist die gesamte Gruppe `count_facet_not_exhaustive` statt teilweise befüllt.

V1 liefert für die drei vorhandenen Rangefelder diese metrischen Profile; alle Zahlen sind SI-Meter:

| Profil                    | feste Grenzen                                                                                       | Default-`open_from` |  UI-Cap |
| ------------------------- | --------------------------------------------------------------------------------------------------- | ------------------: | ------: |
| `distance_m_web_v1`       | 0, 2'000, 5'000, 10'000, 15'000, 20'000, 30'000, 50'000, 75'000, 100'000, 150'000, 200'000, 300'000 |              50'000 | 300'000 |
| `elevation_gain_m_web_v1` | 0, 100, 250, 500, 750, 1'000, 1'500, 2'000, 3'000, 4'000, 6'000                                     |               2'000 |   6'000 |
| `elevation_loss_m_web_v1` | identisch zu Aufstieg                                                                               |               2'000 |   6'000 |

`open_from` wird pro Aktivitätsfamilie, normalisiertem Quell-Scope und Feld aus dem p99 des öffentlich auffindbaren Korpus gewählt. Nach aufsteigender Sortierung der `n` validen bekannten Werte ist p99 der deterministische, 1-basiert indizierte Nearest-Rank-Wert `x[ceil(0.99 × n)]`; die bekannte Abdeckung ist `n / Anzahl aller Trails der Domain`. Der Wert wird auf die kleinste registrierte Profilgrenze aufgerundet, die nicht kleiner ist, oder am UI-Cap gesättigt, wenn p99 darüber liegt. Damit gilt weiterhin `open_from = min(UI-Cap, max(Default-open_from, aufgerundetes p99))`. Eine eigene Statistik verlangt mindestens 500 valide bekannte öffentliche Werte und mindestens 80 Prozent bekannte Abdeckung. Darunter gilt zunächst eine ausreichend grosse breitere Domain derselben Familie aus dem öffentlich auffindbaren Gesamtkatalog, danach der ausgelieferte Default. Private, geteilte oder nur actor-sichtbare Werte beeinflussen die Domain nie. Bei mehreren ausgewählten Familien gilt ihr grösster Endanschlag; ohne Auswahl ein versionierter `all`-Snapshot.

Capabilities beziehungsweise der Suchkontext binden Profil, SI-Einheit und effektives `open_from`. Der Suchkontext enthält zusätzlich `statistics_domains` in kanonischer Reihenfolge; jeder Eintrag bindet Aktivitätsfamilie, den tatsächlich verwendeten normalisierten Source-Selector des Statistik-Korpus aus Abschnitt 5.3, `catalog_visibility = public_discoverable`, `domain_source = public_p99 | broader_public_fallback | shipped_default`, seinen berücksichtigten Endanschlag und die Statistikrevision. Der normalisierte Request-Selector bleibt separat unter `normalized_search.filters.source` gebunden; weicht der Statistik-Scope wegen eines dokumentierten Fallbacks ab, sind damit beide sichtbar. So sind auch mehrere ausgewählte Familien und `origin_instance` ohne geratenen Scope reproduzierbar. Feste feinere Projektionsbuckets oberhalb von `open_from` werden serverseitig zum offenen Response-Bucket aggregiert. Der UI-Cap ist keine implizite API-Grenze und schliesst keine bekannten Trailwerte aus; die normalisierte Suche speichert weiterhin ihre tatsächlichen SI-Grenzen und nie eine Histogramm-ID.

## 11. Suchkontext, Snapshot und Cursor

Die technische Ausprägung dieses Abschnitts ist im angenommenen
[ADR 0001: Cursorzustand und Snapshot-Lifecycle](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/)
festgelegt. Die interne PocketBase-Control-Plane, ihre Collections,
Revisionstransaktion und Federation-/Index-/Gateway-Transitionen konkretisiert
der normative
[Federation-, Index- und Gateway-Zustandsvertrag](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/).

### 11.1 Zeit- und Inhaltsstand

Der öffentliche `search_context` besitzt diese Pflichtfelder: `contract`, opakes `context_id`, `as_of`, `valid_until`, `capability_revision`, `time_zone_database_revision`, `content_snapshot`, `ranking_profile`, `sort_profile` und `metric_profile` jeweils aus `id` und `hash`, `aggregation_profiles`, `projection_profile`, `acl_policy_revision`, `category_preference_revision`, `source_scope`, `spatial_contracts` und `accuracy`. `source_scope` ist immer der vollständige kanonische `SourceSelectorV1` aus Abschnitt 5.3 und exakt gleich `normalized_search.filters.source`; bei `origin_instance` enthält er deshalb zwingend `origin_instance_id` und ist nie nur ein String. `aggregation_profiles` enthält jede angeforderte Facetten- und Histogrammgruppe in kanonischer Reihenfolge; jeder Facetteneintrag bindet seine `options_revision` und sein Options-`sort_profile`, auch bei einer festen Enumoption. Das verschachtelte Facettenfeld `sort_profile` bestimmt die Katalogreihenfolge und ist vom gleichnamigen Top-Level-Profil für die Treffernavigation verschieden. Histogrammeinträge binden zusätzlich SI-Einheit, effektives `open_from` und die nichtleere Liste `statistics_domains`. Deren Einträge verwenden die in Abschnitt 10.3 definierte Form und werden nach Aktivitätsfamilie, `source_scope.scope` und gegebenenfalls `origin_instance_id` sortiert. Das gilt auch für eine Gruppe, die in dieser Ausführung `status: error` liefert, damit ein Retry weder Grenzen noch Statistikdomain umdeutet. `spatial_contracts` ist ohne Spatialfilter `[]`; beim V1-Startpunktradius enthält es genau die im Responsebeispiel gezeigte Variante. Bei `route_geometry_radius` enthält es genau die nachfolgende Variante. Die Liste wird nach `kind`, dann `id` sortiert. `federation` ist bei `source_scope.scope = all | federated | origin_instance` erforderlich und bei `source_scope.scope = local` verboten; es enthält `universe`, `snapshot_set_revision` und das in V1 stets falsche `live_remote_queries`. Weitere Profilrevisionen erscheinen nur, wenn die normalisierte Spec ihre Capability tatsächlich aktiviert, und werden dann Cursorbestandteil.

```json
{
  "kind": "route_geometry_radius",
  "id": "route_radius_ux_v1",
  "predicate_accuracy": "bounded_approximate",
  "max_radius_m": 100000,
  "product_boundary_tolerance_m": 50,
  "resolution": 100,
  "geometry_profile": "route_geometry_search_v1",
  "simplification_profile": "rdp_s50_v1",
  "max_segment_length_m": 5000
}
```

Dieser Eintrag, der normalisierte `route_geometry_radius`-Filter, die gleichnamigen Capabilityfelder und der `accuracy.contracts`-Eintrag aus Abschnitt 10 MÜSSEN dieselben Vertrags-, Profil- und Grenzwerte binden. Der Cursor bindet das vollständige Objekt; eine Abweichung ist kein zulässiger Resume-Kontext.

Bei kataloggestützten Facetten sind `options_revision` und das verschachtelte `sort_profile` die für genau diesen Actor, diese Locale und diesen Inhaltsstand verwendeten Bindings des entsprechenden `TrailSearchOptionsV1`-Katalogs. Feste Enumfacetten verwenden stattdessen unveränderliche versionierte Options- und Sortprofilnamen. Beide Formen sind Cursor- und Cachebestandteil.

`search_context.as_of` und `valid_until` sind UTC-RFC-3339-Zeitpunkte mit Millisekunden. Der Server setzt `as_of` einmal vor der ersten Teilabfrage. `valid_until` ist exklusiv: Bei `now >= valid_until` ist der Kontext abgelaufen. Er endet spätestens an der frühesten relevanten Federation-Expiry oder Retentiongrenze. „Relevant“ umfasst die vollständige unpaginierte Treffermenge sowie die durch disjunktive Facetten und Histogramme bewusst verbreiterten Aggregationsuniversen, nicht nur die erste Seite oder die dort sichtbaren Origins.

Bei Federation stammen `snapshot_set_revision` und die früheste Expiry aus dem unveränderlichen Scope-Binding genau des publizierten `content_snapshot`, nicht aus einem neueren mutablen Federation-Head. Ein späterer TTL-Refresh darf `valid_until` eines alten Content-Snapshots deshalb nie verlängern. Der aktuelle Head dient nur zur Restriction-/Stale-Erkennung und zur Vorbereitung einer neuen Epoch.

Das Binding referenziert ein rekonstruierbares kanonisches Membershipmanifest.
Sein Expiryfeld ist bei nichtleerer Membership exakt deren Minimum und nur bei
leerem Manifest leer; eine fehlende oder spätere Grenze macht den Snapshot für
neue föderierte Kontexte fail-closed unbenutzbar.

Alle Treffer- und Aggregationsabfragen einer Response MÜSSEN denselben logischen Inhaltsstand beobachten. `index_generation_id` allein genügt dafür nicht: Es pinnt Schema, Einstellungen und die gemeinsam aktivierte Indexfamilie, aber bei einer veränderlichen Generation nicht automatisch deren Dokumentinhalt. V1 verwendet deshalb ausschliesslich `content_snapshot.mode = revision_fenced`; jeder andere Modewert verletzt das V1-Responseschema.

Der Snapshot verlangt `mode`, eine nichtleere opake `id`, `index_generation_id`, `delivered_catalog_revision`, `security_watermark` und `content_epoch`. Revisionen und Epoch sind kanonische nichtnegative Dezimalstrings ohne führende Nullen. `content_epoch` ist an genau eine Indexgeneration gebunden und bezeichnet dort einen veröffentlichten Inhaltsstand. Eine Snapshot-ID darf nie für ein anderes Tupel aus Kern-/Overlay-/Geometry-Dokumentstand, Generation und Epoch wiederverwendet werden. Mehrere Suchkontexte dürfen denselben Stand mit einem neueren aktuellen Security-Watermark referenzieren, weil Security bewusst nicht gepinnt wird.

Eine stillgelegte Generation darf nach einem atomaren Generationstausch für laufende Cursor und als Quelle eines getrennt aufgeholten Rollbackkandidaten unverändert aufbewahrt werden. Diese operative Versiegelung ist kein zweiter öffentlicher Snapshotmodus: Ihr `content_snapshot.mode` bleibt `revision_fenced`, und der Gateway routet den Cursor über seine gepinnte `index_generation_id` und `content_epoch`. Sie ist nur abfragbar, wenn aktuelle Restriction-Fences vor Ranking und Counts vollständig in jede betroffene Query eingehen; andernfalls bleibt sie fail-closed unbenutzbar. Nach späteren Writes auf der neuen aktiven Generation darf sie nicht direkt als veralteter Rollbackstand aktiviert oder für ein Catch-up mutiert werden.

Epoch-Publikation erfolgt durch genau einen persistent serialisierten Publisher mit atomarem Cutoff und Compare-and-swap: Er darf einen Stand nur publizieren, wenn sämtliche vor dem Cutoff gestarteten beziehungsweise ihm zugeordneten Kern-, Overlay- und Geometry-Mutationen vollständig bestätigt sind. Ein konkurrierender Publish-Versuch muss vor dem ersten Engine-Write scheitern und kann deshalb keinen Zwischenstand als `published` markieren.

Ein Publisher-Failover darf den Writepfad erst übernehmen, wenn der vorherige Prozess durch Infrastruktur oder Credential-Fencing keinen weiteren Meilisearch-Task mehr einreichen kann. Active/Active-Publisher ohne einen von diesem Writepfad durchgesetzten monotonen Fencing-Token sind in V1 nicht zulässig.

Vor dem ersten normalen Content-, Geometry- oder Settings-Engine-Write wechselt die Generation dauerhaft von `published` zu `updating`. Eine neue monotone Epoch wird erst `published`, wenn alle persistierten Taskintents terminal `succeeded` sind, jede möglicherweise angenommene physische Submission selbst terminal bestätigt oder durch den STATE1-konformen Barrier-/Endmarkerbeleg vollständig abgedeckt ist und Delivery- sowie Security-Watermark für sämtliche Bundlebestandteile lückenlos bis zum Cutoff bestätigt sind. HTTP `202`, ein fehlerfreier Transport oder die höchste einzelne Task-ID sind keine Bestätigung. Nach einem Crash wird derselbe Publish-Versuch verifiziert und fortgesetzt oder die Generation bleibt unbenutzbar; ein teilweise mutierter Stand wird nie als alte Epoch freigegeben. Ein `fence_apply` ist die einzige Ausnahme: Während bereits wirksame `exact_target`-Gateway-Fences das Ziel vor Ranking und Counts schützen, darf der serialisierte Publisher nur den vollständig gebundenen kumulativen Visibilitystand aller das physische Ziel überlappenden Subject-Frontiers beobachtungsäquivalent realisieren. Der persistierte Attempt lässt Content-Epoch und -Cutoff unverändert, darf keinen anderen Dokumentwert oder Settings ändern und kann weder Sichtbarkeit noch Ranking, Counts oder DTO-Beobachtungen verändern; eine zwischen Anlage, Submit und Abschluss geänderte Frontier lässt den alten Delivery-/Retirement-Beleg scheitern. Grant/Clear und die Lockerung eines konservativen Whole-Subject-Guards laufen wieder über eine normale neue Epoch.

Der gepinnte Dokumentstand umfasst jedes Feld, das Treffbarkeit, Ranking, Total, Facetten, Histogramme, Highlights, Matchgründe oder das ausgelieferte `trail_hit_v1` beeinflusst. Dazu gehören insbesondere lokalisierte Author-/Tag-/Taxonomielabels, Actor-Likezustand, Asset-/Waypoint-Presence, Thumbnailmetadaten und sichtbare Excerpts. Der Gateway hydriert solche Werte nach der Suche nicht aus einer ungepinnten veränderlichen Datenbank. Ihre Mutation erzeugt eine neue Content-Epoch; eine Sichtbarkeitsverengung setzt zusätzlich sofort den aktuellen Fence. Rein deterministisch aus stabilen IDs gebildete lokale Routen dürfen zur Auslieferung erzeugt werden.

Das Gateway liest den generationengebundenen veröffentlichten Epochwert vor und nach dem gesamten Hits-/Total-/Facetten-/Histogramm-Batch. Ist er ungleich oder zwischenzeitlich `updating`, verwirft die erste Seite den gesamten Kontextkandidaten und beginnt einen begrenzten Full-Context-Retry mit neuem `as_of`, neu berechnetem `valid_until` und allen neuen Revisionsbindungen. Bleibt kein stabiler Stand erreichbar, folgt `search_unavailable`; ein altes `as_of` wird nie mit einer später publizierten Projektion kombiniert. Eine Cursor-Folgeseite wird nicht auf einen neuen Kontext umgebogen: Sie verlangt dieselbe Generation und Epoch und ergibt andernfalls `search_context_stale`. Ein Wechsel des aktiven Generation-Pointers allein macht sie nicht stale, solange ihre gepinnte vorherige Generation unverändert und gemäss Retention erreichbar bleibt.

Zum selben Pre-/Post- und finalen Response-Kontrollstempel gehören Zustand,
Version und Writer-Fence-Tupel des PocketBase-Publishers. Ist dieser
`quarantined`, bleiben neue und bestehende Seiten auch bei weiterhin
`published`er Generation fail-closed mit `search_unavailable`; ein bereits
bewiesener Inkarnations- oder Retentionsverlust behält seinen spezifischen
`search_context_unavailable`-/Stale-Fehler und dessen Präzedenz.

Catalog- und Watermarkrevisionen werden als Strings übertragen, damit 64-Bit-Werte in JavaScript nicht gerundet werden. `content_snapshot.id`, Revisionswerte und Generationen sind opake Diagnosewerte; sie erlauben keinen direkten Indexzugriff.

Sichtbarkeitsverengungen sind nie gepinnt. Das Gateway prüft vor und nach jeder Gesamtantwort den aktuellen Restriction-Fence und Security-Watermark. Public→Private, Share-Widerruf, Löschung oder Sperre setzen den Fence vor der asynchronen Indexkonvergenz. Ändert sich die relevante Sichtbarkeit, verwirft eine erste Seite die Antwort und führt denselben begrenzten Full-Context-Retry gegen den aktuellen Fence aus; bleibt er instabil, folgt `search_unavailable`. Eine Cursor-Folgeseite ergibt sofort `search_context_stale`. Eine verlorene, aber laut `valid_until` noch zugesagte Snapshotressource ergibt `search_context_unavailable`, nicht `search_context_expired`.

Eine Sichtbarkeitserweiterung durch Grant/Clear wird ausschliesslich über eine
normale neue Content-Epoch publiziert. Dasselbe gilt, wenn ein vorübergehend
konservativer Whole-Subject-Gateway-Fence durch den exakten, für einzelne
Principals wieder sichtbaren eingeschränkten Zielstand ersetzt wird; epochloses
`fence_apply` darf nur den vollständig gebundenen kumulativen Stand aller
überlappenden exakten Guards beobachtungsäquivalent physisch realisieren.
Enthält der Snapshot eine für das vollständige Treffer-
oder Aggregationsuniversum relevante `clear_revision` oder
`guard_release_revision`, ist der bisherige Restriction-Fence aber noch nicht
`retired`, gilt bis zum Retirement ein Issuance-Fence: Die erste Seite stellt
weder Suchkontext noch Cursor aus und liefert nach dem begrenzten
Full-Context-Retry `search_unavailable`. Der finale Response-Guard prüft dies
erneut. Ein vor dieser Epoch ausgestellter Cursor wird durch den Epochwechsel
`search_context_stale`; ein Same-Epoch-Cursor darf in diesem Zwischenfenster
niemals existieren. Liegt ein solcher Cursor nach einem Head-Swap auf einer
versiegelten Altgeneration, wird er bei einem relevanten
`guard_release_converged` zusätzlich anhand des retenierten Fencebelegs und
seines vor `guard_release_revision` liegenden Snapshot-Cutoffs stale. Dasselbe
gilt bei Übernahme des Guard-Release durch einen nachweislich engeren
Restriction-Successor; wird stattdessen ein begonnener Clear supersediert, ist
`clear_revision` der Stale-Anker. Vor dem Retirement nach jeder solchen
Successor-Supersession muss die geschlossene normale Successor-Epoch publiziert
sein; der Link selbst entsteht bereits atomar mit der
Successor-Restriction. Der Pointerwechsel allein genügt ausdrücklich nicht.

### 11.2 Actor-, Overlay- und Federation-Invarianten

- Kern- und berechtigte Overlay-Kandidaten werden vollständig vereinigt und anhand der kanonischen Trail-ID dedupliziert, bevor Ranking, Total, Facetten, Histogramme oder Pagination stattfinden.
- Ein nicht berechtigtes Dokument darf weder Treffbarkeit noch Rang, Count, Bucket, Highlight oder Matchgrund beeinflussen.
- Der Security-Fence ist aktueller als ein gecachter Suchkontext; eine Restriction kann die Menge sofort verengen, aber nie durch einen Request erweitert werden.
- Ein publizierter Clear oder Guard-Release erweitert keinen bereits
  ausgestellten Kontext: Vor dem Fence-Retirement blockiert er neue Kontexte,
  danach beginnt die erste Ausstellung bereits unter der endgültigen
  Sichtbarkeit.
- Föderierte Treffer stammen ausschliesslich aus lokal materialisierten, zum `as_of` verifizierten und noch auffindbaren Snapshots. Es gibt keine Live-Remote-Abfrage pro Suche.
- `valid_until` liegt spätestens auf der frühesten relevanten `discoverable_until`. Das epochgebundene Scope-Manifest bindet jede Einzel-Expiry und genau deren Minimum; dieses ist leer genau bei leerer Membership. Der planmässige Ablauf wird deshalb bei Erreichen von `valid_until` zu `search_context_expired`; ein vorzeitiges Delete, Public→Private oder ein anderer Restriction-Fence wird unmittelbar zu `search_context_stale`.
- Der öffentliche Kontext enthält weder Principal-Fingerprint noch Actor-, Share-, Overlay-, Tenant- oder interne Indexnamen. Actor- und Policybindung bleiben als kontextgesalzene keyed Bindings im opaken Cursor; rohe Principalwerte werden dort nicht gespeichert.

### 11.3 Cursorvertrag

V1 verwendet einen selbstenthaltenden, versionierten und mit HMAC-SHA-256 authentisierten Cursor. Derselbe Gateway stellt ihn aus und prüft ihn; der Algorithmus ist serverseitig festgelegt und kann nicht durch ein Tokenfeld gewählt werden. Der Cursor ist für Clients API-opak, aber nicht vertraulich. Clients dürfen ihn nur unverändert zurücksenden. Er bindet mindestens:

```text
Formatversion + kid + contract + context_id + content_snapshot
opake Control-Plane-Inkarnation als Restore-Fence
keyed Request-Binding + effective_sort + ranking/sort/metric profile
page.size + include/projection fingerprint
locale + entity kind + vollständiger source selector
as_of + valid_until
time-zone database revision + compiled date bounds
keyed Principal-Binding + ACL/policy/preference revisions
federation snapshot set
spatial/analysis contracts
next position or offset
```

Die Request-Binding deckt die vollständig normalisierte Spec, `page.size` und alle Includes ab. Die Principal-Binding deckt Tenant, effektiven Actor beziehungsweise `anonymous_public` und Autorisierungsklasse ab. MAC-, Request- und Principal-Key werden aus demselben durch `kid` gewählten Cursor-Root-Key mittels HKDF-SHA-256, festen versionierten Domainlabels und `context_id` als Kontext abgeleitet. Der lesbare Payload enthält weder Suchtext, private Filter- oder Geoangaben, rohe Actor-/Tenant-/Share-/Overlay-IDs, interne Indexnamen noch verborgene Ranking-, Match- oder ACL-Werte. Benötigt ein späterer Execution-Pfad vertrauliche Positionswerte, verlangt er einen neuen Cursoruntervertrag und eine erneute Architekturentscheidung zu AEAD oder serverseitigem Zustand.

Normative Regeln:

- Derselbe Cursor darf innerhalb seiner Gültigkeit mehrfach verwendet werden und liefert idempotent dieselbe Folgeseite. Er ist nicht single-use.
- V1 unterstützt nur Vorwärtsnavigation. Die UI darf bereits gelieferte Seiten lokal beziehungsweise im History-State halten.
- Ein syntaktisch ungültiger, manipulierter oder für einen anderen Actor ausgestellter Cursor ergibt einheitlich `invalid_cursor`, damit keine Actorbindung verraten wird.
- Eine abweichende normalisierte Spec, `page.size` oder `include` ergibt `cursor_request_mismatch`.
- Ein neuer Ranking-, Projection-, Capability-, Histogramm- oder Spatial-Vertrag gilt nur für neue Kontexte und invalidiert einen zugesagten alten Vertrag nicht. Der Server hält dessen Artefakte bis `valid_until` vor; ein unerwarteter Verlust ergibt `search_context_unavailable`.
- Eine neue Content-Epoch in der vom Cursor gepinnten Generation oder eine aktuelle ACL-/Restriction-, Taxonomiepräferenz- beziehungsweise Federationänderung, die das gebundene Suchuniversum verändert, ergibt `search_context_stale`. Epochs einer nach dem Cutover aktiven anderen Generation sind für diesen Cursor unerheblich.
- Ein Kontext ist bei `now >= valid_until` mit `search_context_expired` abgelaufen.
- Cursor stehen nie in einer kanonischen Share-URL, gespeicherten Suche, Analytics-Dimension oder einem Log. Diagnosen verwenden höchstens einen kurzlebigen Hash.
- Ein Cursor autorisiert nichts. Jeder Zugriff durchläuft erneut Authentifizierung, aktuelle Restrictions und DTO-Projektion.
- Eine abweichende Control-Plane-Inkarnation nach Restore führt fail-closed zu `search_context_unavailable`; der Cursor wird nie gegen den zurückgedrehten Stand neu interpretiert.

Die interne Principalbindung besteht aus Tenant, effektivem Actor und Autorisierungsklasse, nicht aus dem kurzlebigen Sessiontoken. Für anonyme Suchen ist der Actor-Marker ausdrücklich `anonymous_public`; ein solcher Cursor ist nur unter derselben anonymen Audience gültig. Login, Logout, Tenant- oder Actorwechsel zwischen Seiten ergibt einheitlich `invalid_cursor` und kann die Treffermenge weder verbreitern noch still auf Public reduzieren. Ein Tokenrefresh desselben Actors ändert die Principalbindung nicht.

Vor Rückgabe eines Cursors erhöht der Gateway dauerhaft und monoton die `issued_valid_until`-High-Watermarks des gewählten Cursor-Root-Keys, der gepinnten Generation und aller benötigten versionierten Vertragsartefakte. Die Antwort darf erst danach sichtbar werden; ein Crash dazwischen führt höchstens zu sicherer Überretention. MAC-Prüfschlüssel werden mindestens bis zu ihrem High-Watermark plus `cursor_key_retention_grace_s` aufbewahrt. Gepinnte Generationen und Vertragsartefakte bleiben mindestens bis zu ihrem High-Watermark plus `context_resource_retention_grace_s` erhalten. Diese interne Resource-Grace ist mindestens maximale Search-Request-Deadline plus zugesagter Clock-Skew und verlängert die fachliche Cursorgültigkeit nicht.

Rotation und Deployment verwenden Expand/Contract: Alle möglichen Leser erhalten einen neuen Root-Key und ein neues Cursorformat, bevor deren Ausstellung beginnt. Die Ausstellung eines alten Formats endet vor dem Entfernen seiner Leserunterstützung; Entfernung ist erst nach dessen High-Watermark plus Grace erlaubt. Ein Rollback auf einen Reader, der bereits ausgestellte Cursor nicht versteht, ist verboten. Scheitert die Integritätsprüfung oder ist die Key-ID unbekannt, gilt `invalid_cursor`. Ist ein als ausgestellt bekannter Schlüssel entgegen der Retentionzusage irreversibel verloren, gilt `search_context_unavailable`.

## 12. Fehlervertrag

Jede nicht erfolgreiche API-Antwort verwendet `Content-Type: application/problem+json` und diese Hülle:

```json
{
  "type": "urn:wanderer:problem:search-context-stale",
  "title": "Search context is stale",
  "status": 409,
  "code": "search_context_stale",
  "detail": "Start the search again.",
  "instance": "/api/v1/trails/search",
  "request_id": "req_opaque",
  "retryable": false,
  "restart_search": true,
  "violations": [
    {
      "path": "/page/cursor",
      "code": "snapshot_changed",
      "message": "The pinned search snapshot is no longer current."
    }
  ]
}
```

`status` und `code` sind die stabile Programmierschnittstelle. `title`, `detail` und `message` sind lokalisierbare Diagnosehinweise und dürfen nicht zur Steuerung verwendet werden. `path` ist ein JSON Pointer nach RFC 6901; `violations` ist bei feldbezogenen Fehlern nichtleer und sonst `[]`. `request_id` ist opak. `retryable` sagt, ob derselbe Request nach Backoff sinnvoll ist; `restart_search` sagt, ob die erste Seite ohne Cursor neu angefordert werden soll.

| HTTP | `code`                                    | `retryable` | `restart_search` | Bedeutung                                                                                                                                                                            |
| ---: | ----------------------------------------- | :---------: | :--------------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
|  400 | `malformed_json`                          |    nein     |       nein       | syntaktisch ungültiges JSON                                                                                                                                                          |
|  400 | `unsupported_contract_version`            |    nein     |       nein       | `contract` fehlt oder ist nicht `trail-search.v1`                                                                                                                                    |
|  400 | `invalid_cursor`                          |    nein     |        ja        | Cursor ist leer, mehr als 4096 Bytes lang, syntaktisch ungültig, MAC-widrig, mit unbekannter Key-ID versehen oder Principal-fremd                                                    |
|  401 | `authentication_required`                 |    nein     |       nein       | actorbezogener Filter oder Scope ohne Anmeldung                                                                                                                                      |
|  403 | `search_scope_forbidden`                  |    nein     |       nein       | authentifizierter Actor darf den ausdrücklich angeforderten Scope nicht verwenden                                                                                                    |
|  409 | `cursor_request_mismatch`                 |    nein     |        ja        | Spec, Seitengrösse oder Includes passen nicht zum Cursor                                                                                                                             |
|  409 | `search_context_stale`                    |    nein     |        ja        | gebundener Inhalts-, ACL-, Preference- oder Federationstand wurde relevant verändert                                                                                                 |
|  410 | `search_context_expired`                  |    nein     |        ja        | `valid_until` ist erreicht                                                                                                                                                           |
|  413 | `search_request_too_large`                |    nein     |       nein       | Transportgrösse überschreitet das Capability-Limit                                                                                                                                   |
|  415 | `unsupported_media_type`                  |    nein     |       nein       | Requestbody ist nicht `application/json`                                                                                                                                             |
|  422 | `invalid_search_request`                  |    nein     |       nein       | schema-unbekanntes oder strukturell ungültiges Feld, Operator, Enum, Typ, Required, Duplicate oder Grenze                                                                            |
|  422 | `query_too_long`                          |    nein     |       nein       | Codepoint- oder Tokenlimit des Textprofils überschritten                                                                                                                             |
|  422 | `unsupported_locale`                      |    nein     |       nein       | Locale wird vom aktiven Profil nicht unterstützt                                                                                                                                     |
|  422 | `unsupported_time_zone`                   |    nein     |       nein       | Zone ist unbekannt, nicht kanonisch oder kein unterstützter IANA-Name                                                                                                                |
|  422 | `unknown_filter_value`                    |    nein     |       nein       | syntaktisch gültige, aber unbekannte stabile ID                                                                                                                                      |
|  422 | `invalid_filter_combination`              |    nein     |       nein       | einzeln gültige Werte sind gemeinsam widersprüchlich                                                                                                                                 |
|  422 | `unsupported_capability`                  |    nein     |       nein       | im implementierten V1-Schema bekanntes Feld, Operator, Facette, Histogramm, Enum oder Untervertrag ist für diesen Actor beziehungsweise diese Instanz oder dieses Profil nicht aktiv |
|  422 | `unsupported_radius_for_spatial_contract` |    nein     |       nein       | Radius liegt ausserhalb der angebotenen Spatial-Grenze                                                                                                                               |
|  422 | `unsupported_sort_for_spatial_contract`   |    nein     |       nein       | Sortierung kann mit dem aktiven Spatial-Modus nicht vertragsgemäss geliefert werden                                                                                                  |
|  422 | `search_space_too_broad`                  |    nein     |       nein       | deterministisches, vorab veröffentlichtes Komplexitätslimit überschritten                                                                                                            |
|  429 | `search_rate_limited`                     |     ja      |       nein       | Actor- oder Instanzbudget temporär erschöpft; `Retry-After` wird gesendet                                                                                                            |
|  500 | `internal_search_error`                   |     ja      |       nein       | unerwarteter interner Fehler ohne Implementierungsdetails                                                                                                                            |
|  503 | `search_unavailable`                      |     ja      |       nein       | konsistenter Suchbatch, gesunde Zeitentscheidung, externer Checkpoint oder benötigte Engine vorübergehend nicht verfügbar                                                            |
|  503 | `search_context_unavailable`              |    nein     |        ja        | zugesagter, noch nicht abgelaufener Kontext ist wegen irreversiblen Snapshot-/Key-/Vertrags-/Control-Plane-Inkarnationsverlusts oder einer nachgewiesenen Issuance-Invarianzverletzung nicht mehr sicher interpretierbar |
|  504 | `search_timeout`                          |     ja      |       nein       | Zeitbudget des kritischen Treffer-/Total-/Snapshotkerns abgelaufen; ein reiner Aggregationstimeout ist stattdessen ein Gruppenfehler unter HTTP 200                                  |

`authentication_required` sendet zusätzlich `WWW-Authenticate: Bearer realm="wanderer"`. `search_rate_limited` sendet `Retry-After` in Sekunden; Clients verwenden zusätzlich exponentiellen Backoff mit Jitter.

Spezifische `violations[].code`-Werte umfassen mindestens `required`, `unknown_field`, `invalid_type`, `duplicate_value`, `invalid_enum`, `invalid_bound`, `empty_selector`, `orphan_subcategory`, `missing_time_zone`, `incomplete_geo_center` und `snapshot_changed`. Derselbe Fehler erhält unabhängig von Engine und Deployment denselben HTTP-/Top-Level-Code.

Abgesehen von vorgelagertem Rate Limiting teilen sich erste Seite und Cursor zunächst diese Präzedenz: Bodygrösse, Media Type, JSON-Syntax, Vorhandensein und Wert von `contract`, strukturelles Requestschema, danach Cursorformat/-integrität/-Actorbindung. Ein fehlendes oder falsches `contract` ist deshalb immer `unsupported_contract_version`; andere fehlende Pflichtfelder sowie schema-unbekannte Elemente sind `invalid_search_request`. Erst nach erfolgreicher Schemaprüfung kann ein schema-bekanntes, aber deaktiviertes Element `unsupported_capability` ergeben. Die spezifischere Kombination aus aktivem `route_geometry_radius` und dem schema-bekannten Ablehnungstoken `sort.by = route_proximity_m` ergibt davor deterministisch `unsupported_sort_for_spatial_contract`; ohne den Routenradius bleibt der Token `unsupported_capability`. Ein vorhandener leerer oder mehr als 4096 Bytes langer Cursor ist `invalid_cursor`; das Requestschema lässt deshalb jeden String zur Cursorprüfung passieren. `search_request_too_large` bezeichnet nur den gesamten HTTP-Body.

Einziger Sonderfall innerhalb der Integritätsstufe: Ist ein syntaktisch valides `kid` als von dieser Control-Plane-Inkarnation ausgestellt bekannt, sein Root-Key aber entgegen einem noch laufenden persistenten HWM irreversibel `lost`, folgt `search_context_unavailable`, obwohl die MAC nicht mehr prüfbar ist. Ein unbekanntes oder nie ausgestelltes `kid` bleibt `invalid_cursor`. Alle anderen Retentionsfehler behalten die unten genannte Präzedenz nach Expiry und Staleness.

Danach verzweigt die Validierung:

- **Erste Seite:** fachlich normalisieren, IDs gegen den aktuellen Actor-Katalog auflösen, aktuelle Capabilities und Kombinationen prüfen.
- **Cursor-Folgeseite:** Spec und Includes ohne aktuelle ID-/Capability-Lookups kanonisieren, zuerst ihre keyed Request-Binding mit dem Cursor vergleichen und bei Abweichung `cursor_request_mismatch` liefern. Danach muss die normative DB-/Instanzuhr innerhalb des zugesagten Skew-Bounds gesund sein; andernfalls folgt `search_unavailable`, weil eine unzuverlässige Uhr weder Expiry behaupten noch den Cursor fortsetzen darf. Erst dann `search_context_expired` bei `now >= valid_until`, danach `search_context_stale` bei einer erkannten Inhalts-/Sicherheitsänderung und zuletzt `search_context_unavailable` bei einem gebrochenen Retentionversprechen prüfen. IDs, Locale, TZDB und Unterverträge werden ausschliesslich gegen die gepinnten Cursorbindungen validiert.

Eine nach Seite eins gelöschte normale Kategorie-ID oder inzwischen deaktivierte neue Discovery-Capability erzeugt dadurch nicht fälschlich `unknown_filter_value` oder `unsupported_capability`: Ein unveränderter generationengebundener Kontext läuft mit seinem gepinnten Wörterbuch weiter; eine neue relevante Content-Epoch oder Sicherheitsänderung macht ihn stale; ein vertragswidrig verlorenes Artefakt ist unavailable. Eine andere, aktuell unbekannte ID zusammen mit einem Cursor ist dagegen zunächst ein Request-Binding-Mismatch.

Fehlerantworten enthalten niemals Suchengine-Ausdrücke, interne Filter, Stacktraces, Principal-/Tenant-IDs, Indexnamen, Cursorinhalt oder verborgene Matchwerte. Ein transienter Ressourcenengpass darf nicht als deterministischer `search_space_too_broad` kaschiert werden; umgekehrt ist eine bekannte, requestabhängige Komplexitätsgrenze kein `503`.

### 12.1 Search-Readiness und Vertragsrevision

Prozess-Liveness und Suchfähigkeit sind getrennte Zustände. Der bestehende
DB-Endpunkt `GET /health` bleibt ein reiner Liveness-Check. Suchfähigkeit wird
über genau diese beiden Endpunkte veröffentlicht:

```text
DB   GET /health/search
Web  GET /api/v1/health/search
```

Beide liefern `Content-Type: application/json`, `Cache-Control: no-store` und
denselben geschlossenen Typ `SearchReadinessV1` mit
`additionalProperties: false`:

| Feld | Pflicht | Typ und Bedeutung |
| --- | :---: | --- |
| `contract` | ja | exakt `search-readiness.v1` |
| `status` | ja | `ready \| not_ready` |
| `code` | ja | einer der unten festgelegten Codes |
| `search_contract_revision` | ja | aktive stabile Request-/ACL-/DTO-Vertragsrevision |
| `profile_revisions` | ja | nichtleeres kanonisches Array aus `SearchProfileRevisionV1` für alle in diesem Deployment erforderlichen logischen Suchprofile |
| `expected_settings_fingerprint` | ja | SHA-256 des erwarteten vollständigen Profilsets |
| `observed_settings_fingerprint` | nein | beobachteter SHA-256, sobald vollständig berechenbar |

Auch die Arrayelemente sind geschlossen und besitzen
`additionalProperties: false`:

```text
SearchProfileRevisionV1 = {
  "logical_role": string,       # ^[a-z][a-z0-9_]{0,63}$
  "profile_revision": string   # 1..128 Unicode-Codepoints
}
```

Das Array hat `minItems: 1` und `uniqueItems: true`; die zusätzliche
Kanonizitätsvalidierung verlangt jede `logical_role` genau einmal und eine nach
ihren UTF-8-Bytes streng aufsteigende Ordnung. Damit ist sein JSON Schema
deploymentunabhängig; welche Rollen ein Deployment verlangt, ist
Laufzeitkonfiguration, keine dynamische Objektproperty.

Ein Fingerprint ist `sha256:` gefolgt von genau 64 kleingeschriebenen
Hexadezimalzeichen. Jedes Profilartefakt enthält die vollständigen wirksamen
Engine-Settings einschliesslich expliziter Defaults als JSON-Objekt.
`settings_digest` ist `sha256:` gefolgt von 64 kleingeschriebenen Hexzeichen
des SHA-256 über dessen RFC-8785-kanonisierte Bytes. Der Settingsfingerprint
ist der SHA-256 über die RFC-8785-kanonisierten Bytes genau dieser geschlossenen
Preimage:

```text
SearchSettingsFingerprintPreimageV1 = {
  "contract": "search-settings-fingerprint.v1",
  "profiles": [{
    "logical_role": string,
    "profile_revision": string,
    "primary_key": non-empty text,
    "settings_digest": sha256-prefixed-lowercase-hex text
  }]
}
```

`profiles` ist nichtleer, folgt exakt Menge und Ordnung von
`profile_revisions` und besitzt ebenfalls geschlossene Elemente. Für
`expected_settings_fingerprint` stammen Primärschlüssel und Settings aus dem
aktiven Profilartefakt; für `observed_settings_fingerprint` aus dem tatsächlich
gelesenen Engineprofil, während Rolle und erwartete Profilrevision unverändert
bleiben. Physische Indexnamen sind nie Teil der Preimage. Kann für nur eine
erforderliche Rolle Primärschlüssel oder vollständiges Settingsobjekt nicht
sicher gelesen werden, wird der beobachtete Fingerprint vollständig
ausgelassen; ein Teilfingerprint ist verboten.

HTTP `200`, `status: ready` und `code: search_ready` sind nur zulässig, wenn
gleichzeitig:

1. die Engine erreichbar ist;
2. der laufende DB- und Webbuild die aktive `search_contract_revision`
   unterstützt;
3. jede erforderliche logische Suchrolle genau einen autoritativen, vollständig
   gebundenen Routingzustand besitzt;
4. jeder von diesem Zustand verpflichtete Index beziehungsweise jede
   verpflichtete Generation existiert und Primärschlüssel sowie Settings dem
   aktiven Profil entsprechen;
5. die für den aktuellen Owner normativ festgelegten Freshness-, Delivery-,
   Security-, Publisher- und Recoverygates grün sind; und
6. kein innerhalb seines deklarierten Scopes unvollständiger, mehrdeutiger oder
   recoverypflichtiger Handoff-, Rebuild-, Swap- oder Publisherzustand
   vorliegt.

Dieser API-Vertrag definiert bewusst nicht, wie ein Owner Quellrevisionen,
Tasks, Generationen, Vergleiche oder Recovery persistiert. Für die drei
heutigen direkten Legacyindizes ist
[IDX0](/develop/specs/trail-search/work-items/engine/idx0/) der konkrete
Produzent beider Endpunkte und seines kleinen Bootstrapzustands. Nach einem
Ownerwechsel sind für die persistente Control Plane der STATE1-Vertrag und die
jeweils implementierenden Indexbausteine normativ. Ein Legacy- oder
Migrationswerkzeug darf seine eigenen strengeren Preflight- und Terminalgates
besitzen, macht deren interne Records aber nicht zu Feldern von
`SearchReadinessV1`.

Unter IDX0 ist der DB-Endpunkt die autoritative Legacyprobe; der Web-Endpunkt
validiert deren geschlossene Wireform, ergänzt ausschliesslich seine eigene
Vertragsunterstützung und veröffentlicht wieder `SearchReadinessV1`.
SRCH-COMP konsumiert diesen Zustand für seine Consumer und erzeugt keine
zweite Readinessquelle. Ein späterer STATE1-Handoff ersetzt nur Owner und
interne Gates, niemals Endpunkte, Wireform oder Fehlerpräzedenz.

DB und Web teilen dafür das statische Erwartungsmanifest des aktiven
Legacyprofils. Kann Web die DB nach Ablauf seines höchstens fünf Sekunden alten
lokalen Zustands nicht erreichen oder ihren Payload nicht validieren, emittiert
es dieses Manifest ohne `observed_settings_fingerprint`. Unterstützt der
Webbuild dessen Vertragsrevision nicht, gilt gemäss Präzedenz
`not_ready/search_contract_unsupported`; andernfalls gilt
`not_ready/search_recovery_required`. Web erfindet weder einen grünen Zustand
noch einen beobachteten Teilfingerprint. IDX0 besitzt diese Produktion; der
Search- und Token-Guard, der den Zustand vor Enginezugriff konsumiert, gehört
zu SRCH-COMP.

Audit- oder Migrationsdateien, Run-IDs, einzelne Engine-Task-IDs und Digests
bestimmter Binary- oder Containerbuilds sind keine Readinessinputs. Ein
kompatibler App-Rollback oder -Rollforward bleibt ohne Reattest zulässig, wenn
der Build dieselbe Vertragsrevision unterstützt. Eine Änderung an
Requestcompiler, ACL-/DTO-Grenze, Projektion oder Indexsettings muss die
entsprechende Vertrags- beziehungsweise Profilrevision erhöhen.

Jeder andere Zustand antwortet mit HTTP `503`, `status: not_ready` und genau
einem Code nach dieser Präzedenz:

| Präzedenz | `code` | Bedingung |
| ---: | --- | --- |
| 1 | `search_engine_unreachable` | Engineverbindung oder notwendiger Engine-Read schlägt fehl |
| 2 | `search_contract_unsupported` | der laufende Build unterstützt die aktive Vertragsrevision nicht |
| 3 | `search_recovery_required` | der autoritative Owner meldet Initialisierung, unvollständige Zustellung, einen nichtterminalen oder mehrdeutigen Betriebszustand oder erforderliches Recovery; Web kann den autoritativen DB-Owner nicht erreichen oder dessen Payload nicht validieren |
| 4 | `search_index_missing` | mindestens ein erforderlicher logischer Index beziehungsweise eine erforderliche Generation fehlt |
| 5 | `search_settings_mismatch` | Primärschlüssel oder Settings weichen vom aktiven Profilset ab, ohne erkannten Recoveryzustand |
| 6 | `search_state_mismatch` | autoritativer Routing-/Control-Plane-Zustand und Engineprofil widersprechen sich, ohne erkannten Recoveryzustand |

`observed_settings_fingerprint` wird gesetzt, sobald das gesamte beobachtete
Profilset sicher berechnet wurde; bei unerreichbarer Engine oder fehlendem
Index bleibt es ausgelassen. Antworten enthalten weder Engine-URL oder -Key,
interne Index-UIDs, Actor-/Principal-IDs, Task-IDs noch
Dokumentfingerprints.

DB und Web halten ihren lokalen Readinesszustand höchstens fünf Sekunden
stale. Jede Search-, Multi-Search-, Profil-, Cluster-, Actor- und Hilfsroute
prüft denselben lokalen Guard vor dem ersten Enginezugriff. Bei `not_ready`
wird Meilisearch nicht aufgerufen; die öffentliche Suchroute liefert das
bestehende Problem Detail mit HTTP `503` und `code: search_unavailable`.
Interne Readinesscodes werden nicht als zusätzliche öffentliche Searchfehler
ausgegeben.

Der Liveness-Endpunkt darf während Initialisierung oder Recovery bereits
`200` liefern. Das öffnet weder Suchrouten noch mutierende Lieferpfade. Wann
Fachwrites, Projektionsworker oder Federation-Worker zulässig sind, bestimmt
ihr jeweiliger Ownervertrag und nicht dieser Such-Wirevertrag.

## 13. Capability Discovery

`GET /api/v1/trails/search/capabilities` liefert die für den aktuellen Actor, die aktive Instanzkonfiguration und die angebotenen Vertragsprofile gültigen Möglichkeiten. Beispiel für einen authentifizierten Actor mit aktiver Assetprojektion:

```json
{
  "contract": "trail-search.v1",
  "capability_revision": "search-capabilities-42",
  "entity_kinds": ["trail"],
  "locales": ["de-CH", "de-DE", "en"],
  "default_locale": "de-CH",
  "time_zone_database_revision": "tzdb-opaque",
  "request_limits": {
    "max_body_bytes": 32768,
    "max_text_codepoints": 256,
    "max_text_tokens": 10,
    "max_filter_values": 100,
    "max_requested_aggregations": 16,
    "max_internal_search_queries": 32,
    "max_histogram_buckets_per_group": 16,
    "max_cursor_bytes": 4096,
    "max_url_bytes": 8192
  },
  "context": {
    "max_ttl_s": 300,
    "cursor_key_retention_grace_s": 60
  },
  "page": { "default_size": 24, "max_size": 100 },
  "filters": {
    "taxonomy": {
      "operators": ["any_of"],
      "states": ["id", "absent", "unmapped", "ambiguous"]
    },
    "tag_ids": { "operators": ["any_of"] },
    "author_ids": { "operators": ["any_of"] },
    "access": {
      "operators": ["any_of"],
      "values": ["public", "owned_private", "shared_with_me"]
    },
    "source": {
      "scopes": ["all", "local", "federated", "origin_instance"]
    },
    "source_difficulty": {
      "operators": ["any_of"],
      "values": ["easy", "moderate", "difficult", "unknown"]
    },
    "distance_m": {
      "operators": ["gte", "lte", "missing"],
      "min_value": 0,
      "max_value": 9007199254740991
    },
    "elevation_gain_m": {
      "operators": ["gte", "lte", "missing"],
      "min_value": 0,
      "max_value": 9007199254740991
    },
    "elevation_loss_m": {
      "operators": ["gte", "lte", "missing"],
      "min_value": 0,
      "max_value": 9007199254740991
    },
    "trail_date": { "operators": ["from", "through", "missing"] },
    "completed": { "operators": ["eq"] },
    "liked_by_me": { "operators": ["eq"], "authentication": "required" },
    "has_photos": { "operators": ["eq"] },
    "start_point_radius": {
      "predicate": "start_point_distance_lte",
      "default_radius_m": 2000,
      "max_radius_m": 10000,
      "predicate_accuracy": "exact",
      "contract": "start_point_radius_v1",
      "distance_profile": "start_point_distance_v1"
    },
    "route_geometry_radius": {
      "predicate": "route_geometry_distance_lte",
      "default_radius_m": 2000,
      "max_radius_m": 100000,
      "predicate_accuracy": "bounded_approximate",
      "display_qualifier": "none",
      "contract": "route_radius_ux_v1",
      "geometry_profile": "route_geometry_search_v1",
      "simplification_profile": "rdp_s50_v1",
      "max_segment_length_m": 5000,
      "resolution": 100
    }
  },
  "sort": {
    "profile": {
      "id": "trail_sort_v1",
      "hash": "sha256:opaque-base64url"
    },
    "fields": [
      "relevance",
      "name",
      "trail_date",
      "created_at",
      "distance_m",
      "elevation_gain_m",
      "elevation_loss_m",
      "source_duration_s",
      "source_difficulty",
      "local_observed_like_count"
    ]
  },
  "ranking_profile": {
    "id": "trail_text_v1",
    "hash": "sha256:opaque-base64url"
  },
  "metric_profile": {
    "id": "trail_metrics_v1",
    "hash": "sha256:opaque-base64url"
  },
  "highlight_profile": {
    "id": "trail_highlight_v1",
    "hash": "sha256:opaque-base64url"
  },
  "response_profile": {
    "id": "trail_hit_v1",
    "hash": "sha256:opaque-base64url",
    "enabled_optional_fields": ["has_photos", "thumbnail"]
  },
  "facets": [
    "taxonomy/category",
    "taxonomy/subcategory",
    "tags",
    "access",
    "source_difficulty"
  ],
  "histograms": [
    {
      "field": "distance_m",
      "bucket_profile": {
        "id": "distance_m_web_v1",
        "hash": "sha256:opaque-base64url"
      },
      "unit": "m",
      "ui_step": 1000,
      "registered_boundaries": [
        0, 2000, 5000, 10000, 15000, 20000, 30000, 50000, 75000, 100000, 150000,
        200000, 300000
      ],
      "default_open_from": 50000,
      "ui_cap": 300000,
      "open_upper": true,
      "statistics_policy": {
        "catalog_visibility": "public_discoverable",
        "percentile": 0.99,
        "quantile_method": "nearest_rank",
        "minimum_known_values": 500,
        "minimum_known_fraction": 0.8,
        "rounding": "next_registered_boundary_saturating_at_ui_cap"
      },
      "domains": [
        {
          "activity_family": "all",
          "source_scope": { "scope": "all" },
          "catalog_visibility": "public_discoverable",
          "domain_source": "shipped_default",
          "revision": "distance-domain-v1-default",
          "open_from": 50000
        }
      ]
    },
    {
      "field": "elevation_gain_m",
      "bucket_profile": {
        "id": "elevation_gain_m_web_v1",
        "hash": "sha256:opaque-base64url"
      },
      "unit": "m",
      "ui_step": 50,
      "registered_boundaries": [
        0, 100, 250, 500, 750, 1000, 1500, 2000, 3000, 4000, 6000
      ],
      "default_open_from": 2000,
      "ui_cap": 6000,
      "open_upper": true,
      "statistics_policy": {
        "catalog_visibility": "public_discoverable",
        "percentile": 0.99,
        "quantile_method": "nearest_rank",
        "minimum_known_values": 500,
        "minimum_known_fraction": 0.8,
        "rounding": "next_registered_boundary_saturating_at_ui_cap"
      },
      "domains": [
        {
          "activity_family": "all",
          "source_scope": { "scope": "all" },
          "catalog_visibility": "public_discoverable",
          "domain_source": "shipped_default",
          "revision": "elevation-gain-domain-v1-default",
          "open_from": 2000
        }
      ]
    },
    {
      "field": "elevation_loss_m",
      "bucket_profile": {
        "id": "elevation_loss_m_web_v1",
        "hash": "sha256:opaque-base64url"
      },
      "unit": "m",
      "ui_step": 50,
      "registered_boundaries": [
        0, 100, 250, 500, 750, 1000, 1500, 2000, 3000, 4000, 6000
      ],
      "default_open_from": 2000,
      "ui_cap": 6000,
      "open_upper": true,
      "statistics_policy": {
        "catalog_visibility": "public_discoverable",
        "percentile": 0.99,
        "quantile_method": "nearest_rank",
        "minimum_known_values": 500,
        "minimum_known_fraction": 0.8,
        "rounding": "next_registered_boundary_saturating_at_ui_cap"
      },
      "domains": [
        {
          "activity_family": "all",
          "source_scope": { "scope": "all" },
          "catalog_visibility": "public_discoverable",
          "domain_source": "shipped_default",
          "revision": "elevation-loss-domain-v1-default",
          "open_from": 2000
        }
      ]
    }
  ],
  "option_lookups": {
    "contract": "trail-search-options.v1",
    "required_query_parameters": ["locale"],
    "sort_profile": {
      "id": "search_options_sort_v1",
      "hash": "sha256:opaque-base64url"
    },
    "endpoints": [
      {
        "kind": "taxonomy",
        "href": "/api/v1/trails/search/options/taxonomy"
      },
      {
        "kind": "tags",
        "href": "/api/v1/trails/search/options/tags"
      },
      {
        "kind": "authors",
        "href": "/api/v1/trails/search/options/authors"
      },
      {
        "kind": "origins",
        "href": "/api/v1/trails/search/options/origins"
      }
    ]
  },
  "saved_searches": {
    "contract": "trail-saved-search-capabilities.v1",
    "href": "/api/v1/trails/search/saved/capabilities"
  },
  "url_codec": "trail_search_url_v1"
}
```

Capabilities sind actorbezogen, enthalten aber keine Trefferzahlen oder verborgenen Objekt-IDs. Sie werden mit `Cache-Control: private` und einem `ETag` ausgeliefert und nie in einem benutzerübergreifenden Cache geteilt. `sort.fields` ist die vollständige Menge ausführbarer Sortierungen; der nur zur stabilen Ablehnung schema-bekannte Token `route_proximity_m` fehlt dort gemäss `route_radius_ux_v1` absichtlich. `option_lookups` annonciert die vier getrennten, autorisierten Endpunkte und deren DTO-/Sortvertrag; Abschnitt 13.1 definiert ihre Selector- und Revisionsbindung. Das optionale Feld `saved_searches` erscheint ausschliesslich für authentifizierte Actors und erst nach vollständiger Freigabe des [Saved-Search-Vertrags](/develop/specs/trail-search/contracts/saved-trail-search-v1/). Sein Fehlen bedeutet, dass Clients weder Saved-Search-Verwaltung noch Defaultauflösung anbieten dürfen.

Die im Beispiel je Histogramm gekürzte `domains`-Liste zeigt nur den ausgelieferten Default für Aktivitätsfamilie `all` und Source-Scope `all`. Produktiv kann sie die endliche Matrix vorberechneter kanonischer Aktivitätsfamilien mit den Scopes `all`, `local` und `federated` enthalten. Ein potenziell unbeschränkter `origin_instance`-Katalog wird hier nicht vollständig aufgebläht: Seine konkrete Domain wird beim autorisierten Suchrequest über den normalisierten Selector `{scope: origin_instance, origin_instance_id}` aufgelöst und stets unter `search_context.aggregation_profiles[].statistics_domains` gebunden. Der Server wählt für einen SearchRequest den normalisierten Source-Scope sowie den `all`-Stand oder den grössten Endanschlag der durch die normalisierte Taxonomieauswahl bestimmten Familien. Eine unbekannte beziehungsweise nicht eindeutig gemappte Familie verwendet `all`; ein Scope oder eine Origin-ID wird nie geraten.

`max_filter_values` zählt jedes Element eines flachen `any_of`. In Taxonomie zählt ein Branch ohne Subcategory-Auswahl beziehungsweise mit nicht-ID-Kategorie als ein Leaf, andernfalls jeder Subcategory-Selector als ein Leaf; der Parent wird dann nicht zusätzlich gezählt. `max_requested_aggregations` ist `include.facets.length + include.histograms.length`; Highlights zählen nicht. `max_histogram_buckets_per_group` zählt numerische Buckets und den verpflichtenden `unknown_bucket` gemeinsam. Eine Überschreitung dieser deterministischen Komplexitätsgrenzen ergibt `search_space_too_broad`; ein Server darf kein zu grosses Bucketprofil als Capability publizieren. `max_internal_search_queries` begrenzt den serverseitig kompilierten Plan aus Treffer-, Total- und Aggregationsqueries. Benötigt bereits der kritische Treffer-/Totalkern mehr als dieses Budget, ergibt der Request `422 search_space_too_broad`. Andernfalls weist der Planer das Restbudget den Aggregationsgruppen in kanonischer Feldreihenfolge zu: Eine Gruppe wird nur geplant, wenn alle von ihr zusätzlich benötigten Queries hineinpassen; sonst plant sie keine davon, liefert `count_query_budget_exceeded` und die nächste Gruppe wird gegen das unveränderte Restbudget geprüft. Gemeinsam genutzte Queries werden beim ersten kanonischen Verbraucher budgetiert und dürfen von späteren Gruppen mitverwendet werden. Eine Option oder ein Histogrammbucket erzeugt niemals je eine eigene Query; versionierte Projektionstokens und exhaustive Distributionen bündeln die Gruppe. Body- und URL-Limits ergeben `search_request_too_large` beziehungsweise den URL-Code `search_url_too_large`. Ein Wert ausserhalb `min_value..max_value` ergibt `invalid_search_request` mit Violation `invalid_bound`; für Radius gelten die spezielleren Spatial-Fehler.

Der Server validiert jeden Request unabhängig von einer vorherigen Capability-Abfrage. Eine Capability ist keine Autorisierung und kein dauerhafter Snapshot. Ihre Revision wird im Suchkontext gespiegelt. Eine neue Discovery-Revision ändert einen laufenden Kontext nicht; dessen tatsächlich verwendete Unterverträge werden bis `valid_until` weiter bedient. Nur eine aktuelle Sicherheits-/Policyänderung darf ihn fail-closed stale machen, ein unerwartet verlorenes Vertragsartefakt ergibt `search_context_unavailable`.

### 13.1 Versionierter Options-Lookup `TrailSearchOptionsV1`

Die vier in `option_lookups.endpoints` angebotenen, autorisierten GET-Endpunkte liefern Kategorien, Tags, Autoren beziehungsweise Origin-Instanzen ohne Counts. Jeder Request verlangt den Queryparameter `locale` als kanonischen BCP-47-Tag, beispielsweise `GET /api/v1/trails/search/options/taxonomy?locale=de-CH`; ein fehlender oder nicht kanonischer Wert ergibt `invalid_search_request`, ein kanonischer, aber nicht angebotener Wert `unsupported_locale`. Es gibt bei diesen GETs keinen impliziten Benutzer-, Browser- oder Instanzlocale-Default. Alle verwenden dieselbe diskriminierte Responsehülle `TrailSearchOptionsV1`; ihr eigenständiger Vertrag lautet `trail-search-options.v1`. Eine Taxonomieantwort sieht beispielsweise so aus:

```json
{
  "contract": "trail-search-options.v1",
  "search_contract": "trail-search.v1",
  "kind": "taxonomy",
  "locale": "de-CH",
  "complete": true,
  "revision_binding": {
    "capability_revision": "search-capabilities-42",
    "options_revision": "taxonomy-options-opaque",
    "sort_profile": {
      "id": "search_options_sort_v1",
      "hash": "sha256:opaque-base64url"
    }
  },
  "items": [
    {
      "bucket_id": "tax.v1/category/cat_hiking",
      "selector": { "kind": "id", "id": "cat_hiking" },
      "label": "Wandern",
      "icon": "hiking",
      "sort_order": 0,
      "visible": true,
      "subcategories": [
        {
          "bucket_id": "tax.v1/subcategory/cat_hiking/sub_family",
          "parent_category_id": "cat_hiking",
          "selector": { "kind": "id", "id": "sub_family" },
          "label": "Familienwanderung",
          "icon": "family",
          "sort_order": 0,
          "visible": true
        },
        {
          "bucket_id": "tax.v1/subcategory_absent/cat_hiking",
          "parent_category_id": "cat_hiking",
          "selector": { "kind": "absent" },
          "label": "Ohne Subkategorie",
          "sort_order": 1,
          "visible": true
        }
      ]
    },
    {
      "bucket_id": "tax.v1/category_absent",
      "selector": { "kind": "absent" },
      "label": "Ohne Kategorie",
      "sort_order": 1,
      "visible": true,
      "subcategories": []
    }
  ]
}
```

`contract`, `search_contract`, `kind = taxonomy | tags | authors | origins`, die normalisierte `locale`, das in V1 stets wahre `complete`, `revision_binding` und `items` sind Pflichtfelder. Die bekannten Pflichtkinder von `revision_binding` sind exakt `capability_revision`, `options_revision` und `sort_profile`; `sort_profile` verlangt exakt `id` und `hash` und entspricht beiden Werten aus `capabilities.option_lookups.sort_profile`. Clients ignorieren additive unbekannte Responsefelder gemäss Abschnitt 2. V1-Optionsantworten sind vollständig und werden nie still gekürzt oder paginiert. Kann ein Endpunkt seinen autorisierten Optionsraum nicht vollständig liefern, antwortet er mit einem normalen Problem Detail statt mit `complete: false`. Ein künftiger suchender oder paginierter Lookup benötigt einen additiven Capability- und DTO-Vertrag.

Taxonomieitems verwenden den vollständigen autoritativen Selector und `bucket_id`, nicht eine vom Client aus Pfad oder Label abgeleitete ID. Kategorien enthalten die vollständig sortierte Liste `subcategories`; jeder Kindknoten nennt zusätzlich `parent_category_id`. `icon` ist optional, alle anderen gezeigten Knotenfelder sind Pflicht. Gemappte und synthetische Knoten verwenden die acht Bucket-IDs aus Abschnitt 10.2. Top-Level-Zustände besitzen `subcategories: []`; unter einem gemappten Parent können die drei Kindzustände stehen.

Die anderen drei Varianten verwenden diese Itemformen:

- `tags`: `{id, label, sort_order, visible}`;
- `authors`: `{id, display_name, handle, avatar_url, sort_order, visible}`; fehlt ein Avatar, ist `avatar_url` wie im Treffer-DTO `""` und niemals `null`;
- `origins`: `{selector, label, sort_order, visible}`, wobei `selector` immer vollständig `{ "scope": "origin_instance", "origin_instance_id": "..." }` ist. Eine freie Domain, URL oder nackte, vom Client einzusetzende Origin-ID wird nicht geliefert.

`sort_order` ist ein eindeutiger, bei `0` beginnender sicherer Integer innerhalb derselben Geschwisterliste und entspricht exakt der Arrayposition. Das Profil `search_options_sort_v1` bildet die fachliche Reihenfolge deterministisch darauf ab: gemappte Kategorien und Subkategorien nach administrativer Reihenfolge und ID, ihre Zustandsknoten danach als `absent`, `unmapped`, `ambiguous`; Tags nach lokalisiertem Label und ID; Autoren nach `display_name`, `handle`, ID; Origins nach Label und Origin-Instanz-ID. Die Taxonomie-Facetten aus Abschnitt 10.2 verwenden dieselbe Hierarchie und Reihenfolge. Eine andere Collation, Tie-Break-Regel oder Reihenfolge erzeugt einen neuen Profilhash und eine neue `options_revision`.

`visible` beschreibt nur die normale Darstellung nach den aktuellen UI-/Taxonomiepräferenzen. Eine für den Actor weiterhin gültige explizite Auswahl bleibt als `visible: false` im Lookup enthalten; eine wegen ACL, Tenant, Restriction oder Föderationsstand nicht autorisierte ID fehlt vollständig und wird niemals durch `visible: false` bestätigt. Der Lookup autorisiert keinen späteren SearchRequest: Jede Suche löst IDs und Origin-Selector erneut gegen ihren aktuellen Actor-Kontext auf.

`options_revision` ist opak und bindet mindestens Contract, `kind`, Locale, Capability-, Katalog-, Policy-, Security-, Preference- und Federationstand, den vollständigen Sortprofilhash sowie die daraus resultierende actor-autorisierte Optionssicht. Sie enthält weder Actor-/Tenant-/Principal-ID noch einen actorspezifischen Salt und darf deshalb nicht als Principal-Fingerprint wirken; zwei Actors mit derselben autorisierten Sicht und denselben gebundenen Ständen können dieselbe Revision erhalten. Actor, Tenant und Principalbindung bleiben separat im privaten Cache-Key beziehungsweise Cursor. Die Revision ändert sich, sobald sich Optionsmenge, stabile ID beziehungsweise Selector, Label, Icon, Hierarchie, Reihenfolge oder Sichtbarkeit ändert. Der `ETag` wird aus `contract`, `kind`, `locale` und der vollständigen `revision_binding` abgeleitet. Optionsantworten sind `Cache-Control: private`, unterstützen `If-None-Match` und dürfen nie zwischen Actors geteilt werden.

Ein Client darf Facetten-Counts nur mit einem Optionskatalog kombinieren, wenn dessen `revision_binding.capability_revision` exakt `search_context.capability_revision` und dessen `revision_binding.options_revision` sowie `revision_binding.sort_profile` exakt den gleichnamigen Bindings des zugehörigen Facetteneintrags in `search_context.aggregation_profiles[]` entsprechen. Dabei verwenden `taxonomy/category` und `taxonomy/subcategory` den Lookup `kind = taxonomy`, `tags` den Lookup `kind = tags`; feste Enumfacetten benötigen keinen externen Lookup. Bei irgendeiner Abweichung verwirft der Client die Katalog-/Count-Kombination und lädt den Lookup neu. Passt auch die neu geladene Revision nicht zum gepinnten Suchkontext, wiederholt er denselben vollständigen SearchRequest ohne Cursor ab Seite eins. Labels, Hierarchie, Reihenfolge oder Sichtbarkeit einer Revision werden nie mit Counts einer anderen Revision angezeigt.

## 14. Teilbare URL `trail_search_url_v1`

### 14.1 Grammatik

Die kanonische URL kodiert ausschliesslich `TrailSearchSpecV1`, nicht Transport- oder Darstellungszustand. Beispiel:

```text
/trails?sv=1&locale=de-CH&q=linde+aussicht
&tax=cat_hiking%2Fsub_family&tax=cat_hiking%2F%40absent
&tag=tag_panorama&difficulty=easy&difficulty=unknown
&access=public&access=shared_with_me
&distance.gte_m=5000&distance.lte_m=20000
&date.from=2026-09-05&date.through=2026-09-06
&date.tz=Europe%2FZurich&liked=true
&near.lat=46.8&near.lon=8.2&near.radius_m=10000
&sort=distance_m%3Aasc
```

Die Zeilenumbrüche dienen nur der Lesbarkeit; eine echte URL besitzt keine. Zulässige Parameter sind:

| Parameter                                              | Kardinalität | Abbildung                                                                      |
| ------------------------------------------------------ | -----------: | ------------------------------------------------------------------------------ |
| `sv`                                                   |      genau 1 | exakt `1`                                                                      |
| `locale`                                               |      genau 1 | kanonischer BCP-47-Tag                                                         |
| `q`                                                    |         0..1 | `text`; leer wird ausgelassen                                                  |
| `tax`                                                  |         0..n | hierarchische Taxonomieauswahl gemäss untenstehender Grammatik                 |
| `tag`                                                  |         0..n | `tag_ids.any_of`                                                               |
| `author_id`                                            |         0..n | `author_ids.any_of`                                                            |
| `access`                                               |         0..n | `public \| owned_private \| shared_with_me`                                    |
| `source`                                               |         0..1 | `all \| local \| federated \| origin_instance`; Default `all` wird ausgelassen |
| `origin`                                               |         0..1 | `origin_instance_id`; genau bei `source=origin_instance` erforderlich          |
| `difficulty`                                           |         0..n | `easy \| moderate \| difficult \| unknown`                                     |
| `distance.gte_m`, `distance.lte_m`, `distance.missing` |      je 0..1 | Distanzrange; Missing `exclude \| include \| only`                             |
| `gain.gte_m`, `gain.lte_m`, `gain.missing`             |      je 0..1 | Aufstiegsrange                                                                 |
| `loss.gte_m`, `loss.lte_m`, `loss.missing`             |      je 0..1 | Abstiegsrange                                                                  |
| `date.from`, `date.through`, `date.tz`, `date.missing` |      je 0..1 | Datumsrange und IANA-Zone                                                      |
| `completed`, `liked`, `photos`                         |      je 0..1 | exaktes `true \| false`                                                        |
| `near.lat`, `near.lon`, `near.radius_m`                |      je 0..1 | atomarer Startpunkt-Radius                                                     |
| `route_near.lat`, `route_near.lon`, `route_near.radius_m` |   je 0..1 | atomarer, capability-gebundener Routenradius gemäss `route_radius_ux_v1`       |
| `sort`                                                 |         0..1 | `relevance` oder `<field>:<asc\|desc>`; Relevanzdefault wird ausgelassen       |

`near.*` und `route_near.*` sind jeweils atomar und gegenseitig ausgeschlossen. Die URL-Normalisierung bildet `route_near.*` ausschliesslich auf `route_geometry_radius` ab; sie darf es weder als Startpunktradius interpretieren noch bei fehlender Capability ignorieren. `tax` besitzt genau diese Formen:

```text
<category-id>
<category-id>/<subcategory-id>
<category-id>/@absent
<category-id>/@unmapped
<category-id>/@ambiguous
@absent
@unmapped
@ambiguous
```

Kanonische Taxonomie-IDs enthalten kein `/` und beginnen nicht mit `@`. Ein Wert ohne Slash wählt die ganze Kategorie beziehungsweise einen nicht gemappten Hauptkategoriezustand. Ist `<category-id>` als ganzer Branch vorhanden, sind zusätzliche Kindwerte desselben Parents redundant und werden bei der Kanonisierung entfernt.

### 14.2 Parser und Kanonisierung

- Vor `URLSearchParams` validiert der Codec die rohe Query: Jedes `%` muss von genau zwei Hexziffern gefolgt sein und der pro Komponente dekodierte Bytestrom muss valides UTF-8 ohne NUL- oder C0-Steuerzeichen sein. Das kompensiert das absichtlich fehlertolerante WHATWG-Decoding; Ersatzzeichen sind kein Fallback.
- Danach gilt die WHATWG-`URLSearchParams`-Semantik: `+` bedeutet Leerzeichen, Prozentsequenzen werden genau einmal dekodiert, `%2B` bedeutet ein literales Plus.
- Malformed Encoding, unbekannte Parameter, unbekannte IDs, doppelte Scalar-Parameter und unvollständige Gruppen sind ungültig. Die UI führt dann keine möglicherweise breitere Suche aus.
- `sort=route_proximity_m:asc` und `sort=route_proximity_m:desc` sind schema-bekannte Ablehnungsformen. Zusammen mit einer vollständigen `route_near.*`-Gruppe ergeben sie `unsupported_sort_for_spatial_contract`; ohne Routenradius ergeben sie `unsupported_capability`.
- Wiederholbare Mengenparameter werden dedupliziert, nach ihrem normalisierten stabilen Wert sortiert und erneut serialisiert.
- Kanonische Ausgabe verwendet UTF-8, `+` für Leerzeichen und grossgeschriebene Hexziffern in Prozentsequenzen.
- Die feste Parameterreihenfolge lautet: `sv`, `locale`, `q`, `tax`, `tag`, `author_id`, `access`, `source`, `origin`, `difficulty`, Distanz, Aufstieg, Abstieg, Datum, Booleans, Geo (`near.*`, danach `route_near.*`), `sort`.
- Defaults werden ausgelassen: leerer Text, `source=all`, `sort=relevance`, alle Access- oder Difficulty-Werte sowie `missing=exclude` bei einer aktiven Range.
- Ein offenes oberes Sliderende wird durch fehlendes `lte` dargestellt. Ein historisches oder aktuelles UI-Maximum wird nicht serialisiert.
- `page`, `page.size`, `cursor`, `include`, Kartenbounds, Scrollposition, View-Modus, Panelzustand, Saved-Search-ID und Default-Revision gehören nicht in die Share-URL. Cursor und bereits gelieferte Seiten sowie nichtfachliche Saved-Search-Provenienz dürfen nur in `history.state` liegen.
- Der Request-Fingerprint entsteht aus der normalisierten Spec und nie aus roher Parameterreihenfolge oder URL-Encoding.
- `sv` fehlt: ausschliesslich Legacyparser. `sv` ist vorhanden, aber ungleich `1`: `unsupported_search_url_version`; kein Legacyfallback.
- `sv=1` zusammen mit einem Legacy-eigenen Suchparameter ist `mixed_search_url_versions`. V1- und Legacyzustand werden nie kombiniert.

Legacy-eigene URL-Keys sind in diesem Routingkontext `author`, `category`, `subcategory`, `page`, `tl_lat`, `tl_lon`, `br_lat`, `br_lon`, `lat` und `lon`. Die V1-Keys `q` und `sort` sind bei `sv=1` ausdrücklich erlaubt, obwohl gleichnamige Parameter ohne `sv` historisch wirkungslos waren.

Integerparameter verwenden bei der Eingabe und Ausgabe exakt `0|[1-9][0-9]*`; feldabhängig unzulässige Nullwerte scheitern danach fachlich. Latitude und Longitude verwenden `-?(0|[1-9][0-9]*)(\.[0-9]+)?`, nie Exponenten, führendes `+`, führende Nullen oder eine fehlende Ziffer vor/nach dem Punkt. Die kanonische Ausgabe ist die kürzeste dezimale Schreibweise desselben IEEE-754-Werts ohne Exponent, entfernt nachfolgende Nachkommanullen und normalisiert `-0` zu `0`. Bereichsprüfung erfolgt erst nach verlustfreier Parse-Prüfung.

URL-Fehler verwenden ausserhalb der Search-API einen kleinen `SearchUrlProblemV1` mit `code`, optionalem `parameter`, lokalisiertem `message` und `status_hint`. Stabile Codes sind:

| `code`                           | `status_hint` | Bedeutung                                             |
| -------------------------------- | ------------: | ----------------------------------------------------- |
| `malformed_search_url_encoding`  |           400 | ungültiges Percent-Encoding, UTF-8 oder Steuerzeichen |
| `unknown_search_url_parameter`   |           400 | Key gehört nicht zur V1-Grammatik                     |
| `duplicate_search_url_parameter` |           400 | Scalar-Key mehrfach vorhanden                         |
| `incomplete_search_url_group`    |           400 | Date-, Source- oder Geo-Gruppe unvollständig          |
| `unsupported_search_url_version` |           400 | `sv` ist nicht `1`                                    |
| `mixed_search_url_versions`      |           400 | V1- und Legacy-Key gemeinsam                          |
| `search_url_too_large`           |           414 | rohe URL überschreitet `max_url_bytes`                |
| `invalid_search_url_value`       |           422 | Typ, Lexik, Enum oder Grenze ungültig                 |
| `unknown_filter_value`           |           422 | formatgültige ID nicht auflösbar                      |
| `invalid_filter_combination`     |           422 | gemeinsam widersprüchliche Werte                      |
| `unsupported_capability`         |           422 | schema-bekanntes Element ist aktuell nicht angeboten  |
| `unsupported_sort_for_spatial_contract` |    422 | Route-Proximity ist für `route_radius_ux_v1` unzulässig |
| `unsupported_radius_for_spatial_contract` |  422 | Radius liegt ausserhalb der angebotenen Spatial-Grenze |

Der Browser zeigt das Problem am betroffenen Control und startet keine Suche. `status_hint` ist der Status, falls SSR oder ein URL-Auflösungsendpunkt den Fehler über HTTP ausliefert; die Domain-Search-API selbst verwendet weiterhin Abschnitt 12.

Die URL hat Vorrang vor gespeicherten Präferenzen. Der vollständige Lebenszyklus
steht im [Vertrag für gespeicherte Trail-Suchen](/develop/specs/trail-search/contracts/saved-trail-search-v1/).
Die Auflösungsreihenfolge lautet:

1. gültige V1-URL;
2. Legacy-URL über den Legacyadapter;
3. einmalige Legacy-LocalStorage-Migration dieses Browserprofils nur, wenn keinerlei Suchparameter vorhanden sind;
4. authentifizierter, für `trail_list` konfigurierter Benutzerdefault;
5. dokumentierte V1-Defaults.

Eine vorhandene, aber ungültige oder nicht unterstützte explizite URL fällt nie
auf den Benutzer- oder Systemdefault zurück. Eine V1-URL liest keine alten
Suchfilter- oder Sortierkeys aus LocalStorage. Zurück-/Vorwärtsnavigation stellt
die Spec aus der URL und optional die bisherige Cursorfolge aus `history.state`
wieder her. Ist der Kontext nicht mehr gültig, startet sie dieselbe Spec bei
Seite eins. Eine erfolgreich aufgelöste Standardsuche wird vor ihrer ersten
Ausführung vollständig und ohne neuen History-Eintrag in die kanonische V1-URL
materialisiert; sie wird nicht mit URL-Filtern gemischt.

## 15. Migration bestehender URLs und Browserzustände

### 15.1 Legacy-Auswertung vor der Übersetzung

Der Legacyadapter bildet zuerst das tatsächlich bisherige Verhalten nach und übersetzt erst dessen Ergebnis. Er verwendet diese Präzedenz:

1. Historische Defaultfilter einschliesslich der damals gespeicherten dynamischen Range-Limits erzeugen.
2. Die URL-Keys `author`, `category` und `subcategory` jeweils mit First-Value-Semantik lesen.
3. Sobald mindestens einer dieser drei Keys vorhanden ist, den gesamten gespeicherten `trailListFilter` ignorieren.
4. Andernfalls `trailListFilter` lesen und mit der historischen Sanitize-Logik auswerten.
5. Die separaten LocalStorage-Keys `sort`, `sort_order` und `paginationItems` danach anwenden.
6. Den separaten URL-Key `page` zuletzt als Legacyseite auswerten.
7. Den historischen `displayOption` nur für die damals abgeleitete Seitengrösse berücksichtigen; er ist kein Suchfilter.

Ohne gespeicherte Seitengrösse gilt für eine alte Seite der historische Default `25`. Alte URL-Keys, die die Oberfläche nie gelesen hat, erhalten nachträglich keine neue Bedeutung: `?q=foo&sort=-distance` ohne `sv=1` aktiviert weder Text noch Sortierung.

Ist `trailListFilter` kein gültiges JSON-Objekt, entfernt der Adapter nur diesen defekten Key, verwendet die Legacydefaults und meldet `legacy_storage_invalid`. Strukturierte, aber ungültige Felder durchlaufen weiterhin die historische Sanitize-/Clamp-Logik; mengenartige Duplikate werden erst danach vor Erstellung der V1-Spec dedupliziert.

Auch oberflächenabhängige Altdefaults bleiben im Adapter sichtbar: Die bisherige Listenansicht startet mit `created ASC`, die Kartenansicht mit `created DESC`. Da keine einzelne normale V1-Spec beide Zustände zugleich reproduziert, werden sie nicht automatisch auf den neuen Empty-Query-Default umgeschrieben. Ein expliziter Nutzerwechsel auf V1 verwendet danach den V1-Vertrag.

### 15.2 Feldabbildung und bewusste Korrekturen

| Legacyzustand                                                                   | V1-Abbildung                                                                                                                          |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Kategorie-/Subkategorie- oder Tagname                                           | nur bei genau einem normalisierten ID-Treffer; sonst Migrationfehler                                                                  |
| Legacy-`author`                                                                 | exakte opake Actor-ID in `author_ids.any_of`; unbekannte ID blockiert die Migration                                                   |
| `category=X&subcategory=Y`, gleicher Parent                                     | nur Branch `X/Y`, nicht zusätzlich die ganze Kategorie X                                                                              |
| Kategorie X und Subkategorie Y mit anderem Parent                               | `X/* OR parent(Y)/Y`                                                                                                                  |
| `__no_subcategory__:<categoryId>`                                               | Branch `<categoryId>/@absent`; unbekannter oder mehrdeutiger Parent blockiert mit entsprechender Legacydiagnose                       |
| `public/private/shared`-Default alle aktiv                                      | Accessfilter ausgelassen; gesamtes autorisiertes Universum                                                                            |
| echte Teilmenge aus `public/private/shared`                                     | entsprechende Kombination aus `public`, `owned_private`, `shared_with_me`, sofern das SRCH0-Fixture Äquivalenz beweist                |
| nicht äquivalente oder historisch widersprüchliche Accesskombination            | im Legacyadapter belassen; keine still breitere V1-Spec                                                                               |
| Difficulty-Default `[0,1,2]`                                                    | Filter ausgelassen; echtes `unknown` bleibt dadurch im Universum                                                                      |
| echte Difficulty-Teilmenge                                                      | entsprechende V1-Enums                                                                                                                |
| `liked=false`                                                                   | Filter ausgelassen                                                                                                                    |
| `liked=true`                                                                    | `liked_by_me: true`                                                                                                                   |
| `completed=true/false`                                                          | exakter V1-Boolean                                                                                                                    |
| `distanceMax == distanceLimit`                                                  | `lte` ausgelassen, auch wenn das heutige Katalogmaximum höher ist                                                                     |
| `ascentMax == ascentLimit`, `descentMax == descentLimit`                        | jeweiliges `lte` ausgelassen                                                                                                          |
| Legacy-Minimum `0` plus offenes Maximum                                         | Rangefilter vollständig ausgelassen                                                                                                   |
| positives Legacy-Minimum                                                        | `gte = floor(sanitizedMin)`                                                                                                           |
| explizites Maximum unter historischem Limit                                     | `lte = ceil(sanitizedMax)`                                                                                                            |
| beide aktiven Grenzen                                                           | beide obigen inklusiven Grenzen plus `missing: exclude`                                                                               |
| nach Legacy-Sanitizing Minimum grösser als Maximum                              | Minimum zuerst auf das rohe Maximum setzen, dann `gte=floor(value)`, `lte=ceil(value)`; nicht als widersprüchlichen V1-Request senden |
| Legacy-Start-/Enddatum                                                          | inklusive lokale Kalendertage in der gewählten Benutzer-/Instanzzone                                                                  |
| Legacy-Radius mit Latitude oder Longitude `0`                                   | aktiver gültiger Radiusfilter                                                                                                         |
| doppelt erzeugte identische `_geoRadius`-Bedingung                              | einmaliger V1-Startpunktradius                                                                                                        |
| Startpunkt-Radius                                                               | bleibt fachlich Startpunkt; bis bewiesener Metrikparität `legacy_start_point_radius_v0`, niemals `route_geometry_radius`              |
| Kartenbounds `tl_lat/tl_lon/br_lat/br_lon` oder alte View-Koordinaten `lat/lon` | ausschliesslich Karten-/History-State; nie Suchradius oder Share-Spec                                                                 |
| Sortkey `name`, `distance`, `duration`, `difficulty`                            | `name`, `distance_m`, `source_duration_s`, `source_difficulty`                                                                        |
| Sortkey `elevation_gain`, `elevation_loss`, `like_count`                        | `elevation_gain_m`, `elevation_loss_m`, `local_observed_like_count`                                                                   |
| Sortkey `created`, `date`; Order `+`, `-`                                       | `created_at`, `trail_date`; Direction `asc`, `desc`                                                                                   |
| beliebige Legacy-Sortierung bei nichtleerem Text                                | `legacy_ranking_v0`, falls eine V1-Sortierung die Reihenfolge ändern würde                                                            |

Die Datumsumstellung von inklusiver UTC-Mitternacht auf inklusive lokale Kalendertage, die Einbeziehung echter Unknown-Difficulty beim bisherigen „alle“-Default und die Korrektur von Koordinate `0` sind **bewusste Fehlerkorrekturen**. Sie erhalten jeweils Release-Hinweis, Migrationwarning und Golden Fixture. Die doppelte identische Radiusbedingung war wirkungslos und wird ohne sichtbaren Hinweis entfernt.

Für Distanz, Aufstieg und Abstieg gilt dieselbe Legacy-Rundung. Sobald nach der obigen Open-End-Erkennung mindestens eine Grenze aktiv bleibt, setzt der Adapter `missing: exclude`; auch ein positives Rohminimum, das durch `floor` zu `0` wird, bleibt dadurch von einem vollständig ausgelassenen Rangefilter unterscheidbar.

Nicht auflösbare Altwerte ergeben `legacy_filter_unresolved`, mehrdeutige Namen oder Übersetzungen `legacy_filter_ambiguous`. Die Oberfläche zeigt Feld und sichtbaren Altwert und führt keine still verbreiterte Suche aus. Sie darf einen Nutzer eine konkrete neue Auswahl treffen lassen.

### 15.3 URL-, Paging- und Rolloutregeln

Eine semantisch identische Legacyseite darf nach erfolgreicher Übersetzung einmalig mit `history.replaceState` in die kanonische V1-URL überführt werden. Eine Seite mit bewusster Korrektur, Legacyranking oder nicht exakt reproduzierbarer Position bleibt im Legacyadapter, bis der Nutzer sichtbar „mit neuer Suche fortfahren“ wählt.

Ein gültiges `page=N` wird für die erste Adapterausführung mit der ermittelten historischen Seitengrösse in einen Offset übersetzt. Fehlt die Grösse, gelten `25` und Warning `legacy_page_size_assumed`. Der Adapter darf die benötigten vorherigen Cursor intern erzeugen; weder Seitennummer noch Cursor gelangen in die kanonische Share-URL. Ein Deep Link wird nicht automatisch umgeschrieben, solange seine Position unter V1 nicht identisch reproduziert ist.

Stabile nicht blockierende Warningcodes sind:

```text
legacy_date_semantics_corrected
legacy_unknown_difficulty_corrected
legacy_zero_coordinate_corrected
legacy_page_size_assumed
legacy_ranking_preserved
legacy_storage_invalid
```

`legacy_filter_unresolved` und `legacy_filter_ambiguous` sind dagegen blockierende Migrationsdiagnosen und führen ohne Nutzerauswahl zu keiner Suche.

Die Migration alter Browserzustände ist einmalig und idempotent. Nach einer gültigen V1-URL werden Legacykeys weder gelesen noch zurückgeschrieben. Der rohe generische Search-API-Pfad und externe Integrationen erhalten einen getrennten Deprecation-Zeitraum; eine Web-URL-Migration beweist nicht automatisch API-Kompatibilität.

## 16. Sicherheits-, Datenschutz- und Lastgrenzen

- Browser und Chat erzeugen als fachlichen Suchanteil ausschliesslich `TrailSearchSpecV1`; der Transportadapter kapselt ihn in `SearchRequestV1`. Freie Enginefilter, Rankingregeln, Attribute, Indexnamen oder Projektionen werden nie übernommen.
- Schema und Compiler verwenden Allowlisting. Escaping allein macht einen unbekannten Filter nicht zulässig.
- ACL, Tenant, aktuelle Restriction-Fences und Sharezustand werden serverseitig ergänzt und können den Request nur verengen. Lokale Taxonomiepräferenzen gelten als UI-/Browse-Default; eine explizite V1-Taxonomieauswahl überschreibt ihr Ausblenden, aber niemals ACL oder Restriction-Fences.
- Overlay- und Kernmengen werden vor Pagination vereinigt und dedupliziert. Ein separater Page- oder Countpfad pro Overlay ist verboten.
- Responses, Caches und Capabilities mit Actor-Kontext sind `private`. `search_response_cache_v1` bildet seinen Key aus dem vollständig defaultmaterialisierten und kanonisch serialisierten `SearchRequestV1`: `contract`, normalisierte `search`, `page.size`, Erstseitenmarker beziehungsweise nicht umkehrbarer Cursor-Digest samt gebundener Position sowie die vollständigen normalisierten Includes `facets`, `histograms` und `highlights`. Der `normalized_search_fingerprint` allein ist ausdrücklich kein ausreichender Cache-Key.
- Zusätzlich bindet derselbe Key effektive Actor-ID, Tenant und Principalbindung, `capability_revision`, vollständigen `content_snapshot`, aktuelle Policy-/Security-/Preference-/Federationrevisionen und die vollständigen Bindings aller aktiven Ranking-, Sort-, Metrik-, Projektions-, Spatial-, Accuracy- und Aggregationsverträge. Wo ein Profil `id` und `hash` definiert, gehen beide ein; Spatial- und Accuracy-Verträge werden in ihrer vollständigen typisierten Form gebunden. Für jede angeforderte Histogrammgruppe gehören insbesondere das zugehörige `aggregation_profiles[]`-Binding mit Profil-ID und -Hash, Einheit, `open_from` und allen `statistics_domains` samt Source-Selector und Revision hinein; Facetten binden ihre `options_revision`. Zwei Requests mit verschiedenen Includes, Capability- oder Bucketprofilrevisionen dürfen daher nie denselben Cacheeintrag treffen.
- Ein Search-Cacheeintrag endet spätestens an `search_context.valid_until` beziehungsweise der frühesten relevanten Federation-Expiry. Vor jeder Auslieferung werden aktueller Restriction-Fence und Security-Watermark erneut geprüft; der rohe Cursor wird weder im Key noch in Diagnosen gespeichert.
- Search-Responses dürfen nicht in gemeinsam genutzten Proxies gespeichert werden. Der Server setzt einen angemessenen `Cache-Control`-Header und `Vary` für die tatsächlich verwendete Authentifizierung.
- Suchtext, Filter auf private IDs und Cursor gelten als potenziell personenbezogen. Produktionslogs speichern standardmässig nur Request-ID, normalisierten Struktur-/Kostenhash, Latenzen, Ergebnisgrössen und kurzlebige opake Fingerprints, nicht den Rohtext oder Cursor.
- Trail-Suchseiten setzen mindestens `Referrer-Policy: strict-origin-when-cross-origin`; externe Navigationen und Ressourcenrequests erhalten dadurch nie die Suchquery mit Text, IDs oder Koordinaten. Eine strengere Policy bleibt zulässig.
- Vom Index gelieferte URLs werden nicht direkt ausgegeben. Detail-, Avatar- und Thumbnail-URLs entstehen über lokale, autorisierte Router beziehungsweise eine geprüfte Media-Policy.
- Highlighttexte werden als Daten und niemals als Trusted HTML behandelt. Content-Security-Policy bleibt zusätzlich aktiv.
- Bevor actor-sensitive Overlays oder neue interne Indexfelder produktiv werden, ist `/api/v1/search/{index}` entweder deaktiviert oder vollständig hinter denselben aktuellen ACL-/Restriction-Fences und einer festen DTO-/Attribut-Allowlist geführt. Freie `attributesToRetrieve`, Filter, Sorts oder Indexnamen werden dort dann nicht mehr akzeptiert; alte Search-/Tenant-Tokens sind widerrufen und können den Domain-Gateway nicht umgehen.
- Der produktive Cutover auf diesen Vertrag setzt die Gates aus dem [Vertrag der Federation-Sicherheits- und Publikationsvoraussetzungen](/develop/specs/trail-search/contracts/federation-security/) für jeden betroffenen Datenpfad voraus. Browser, App, Chat, Legacyadapter und externe Integrationen dürfen weder Meilisearch direkt erreichen noch über alte Schlüssel, Search-/Tenant-Tokens oder einen generischen Enginepfad die Gateway-, DTO-, ACL-, Publication- oder Restriction-Regeln umgehen. Diese Freigabeabhängigkeit ändert weder Requestschema noch Source-Scope, Filter oder Sortierung und hindert interne Implementierungs- und Shadowarbeit nicht.
- `max_body_bytes`, `max_filter_values`, `max_requested_aggregations`, `max_internal_search_queries`, `max_histogram_buckets_per_group`, Page- und Cursorgrenzen werden vor teuren Backendabfragen beziehungsweise vor Ausführung des kompilierten Plans geprüft.
- Ein versionierter Komplexitätsprüfer bewertet Anzahl alternativer Taxonomiezweige, Aggregationsqueries, Spatial-Plan und Overlayfanout. Nur eine deterministisch requestabhängige Überschreitung ergibt `search_space_too_broad`; der Server clampet keine Auswahl still.
- Rate Limit, Gesamtdeadline und Abbruchsignal gelten für den gesamten Suchrequest. Treffer-/Totalkern und jede Aggregationsgruppe besitzen darin getrennte Teilbudgets; das Überschreiten eines Aggregationsbudgets erzeugt den dokumentierten Gruppenfehler, während ein abgebrochener Browserrequest weiterhin alle noch laufenden Enginequeries beendet, soweit die Engine dies unterstützt.
- Fehler, Timingdiagnosen und Telemetrie dürfen keine Existenz enger sichtbarer Treffer bestätigen. Sicherheitsrelevante Tests vergleichen deshalb nicht nur Hits, sondern auch Total, Facetten, Histogramme, Reihenfolge, Highlights, Warnings und Fehlerform.

## 17. Normative Akzeptanzmatrix

Jede Zeile wird mindestens als Compiler-/Gateway-Test umgesetzt. Mit `E2E` markierte Fälle laufen zusätzlich gegen Web-URL, Browsernavigation und eine reale Suchengine. IDs beziehen sich dauerhaft auf diesen Vertrag.

### 17.1 Normalisierung und Filter

| ID      | Eingabe beziehungsweise Ereignis                                                                                      | Erwartung                                                                                                                                                            |
| ------- | --------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| REQ-01  | Feld ausgelassen, `null`, leere Selektorliste                                                                         | ausgelassen nutzt dokumentierten Default; `null` und leere Liste ergeben `invalid_search_request`                                                                    |
| REQ-02  | dieselben IDs in anderer Reihenfolge                                                                                  | identische `normalized_search` und identischer Fingerprint                                                                                                           |
| REQ-03  | schema-unbekanntes Requestfeld beziehungsweise Operator/Enum                                                          | `422 invalid_search_request`, Violation `unknown_field` beziehungsweise `invalid_enum`                                                                               |
| REQ-03b | im V1-Schema bekanntes, aber in den aktuellen Capabilities deaktiviertes Element                                      | `422 unsupported_capability`; nichts wird ignoriert                                                                                                                  |
| REQ-04  | syntaktisch gültige unbekannte ID                                                                                     | `422 unknown_filter_value`; keine Suche ohne diesen Filter                                                                                                           |
| REQ-05  | 257 Codepoints oder elf analysierte Tokens                                                                            | `422 query_too_long`; keine Kürzung                                                                                                                                  |
| REQ-06  | `{search:{}, page:{}, include:{}}` versus `tag_ids:{}` oder `taxonomy:{}`                                             | erste Objekte normalisieren zu Defaults; Selektorwrapper ohne `any_of` sind ungültig                                                                                 |
| FLT-01  | zwei Tags, Difficulty und Datum                                                                                       | Tags OR, Difficulty OR, die drei Gruppen untereinander AND                                                                                                           |
| TAX-01  | Kategorie A vollständig OR Kategorie B mit B1 und `absent`                                                            | exakt `A/* OR B/(B1 OR absent)`                                                                                                                                      |
| TAX-02  | Subkategorie-ID unter falschem Parent                                                                                 | `invalid_filter_combination` mit `orphan_subcategory`                                                                                                                |
| TAX-03  | `absent`, `unmapped`, `ambiguous` auf Kategorie- und Subkategorieebene                                                | je Ebene drei disjunkte Treffermengen; Presence-/Mappingfakten und die acht festgelegten Taxonomie-Bucket-IDs bleiben eindeutig                                      |
| RNG-01  | Wert exakt auf `gte` oder `lte`                                                                                       | Treffer; beide Requestgrenzen sind inklusive                                                                                                                         |
| RNG-02  | Auf-/Abstiegsrange mit `exclude`, `include`, `only`; belegter Wert `0`                                                | Unknown ausgeschlossen, zusätzlich eingeschlossen beziehungsweise allein; der feldfachlich gültige Nullwert bleibt bekannt                                           |
| RNG-03  | historischer Default `0` ohne Presence; expliziter/berechneter Nullwert bei Auf-/Abstieg; Distanz-/Quelldauerwert `0` | historischer Default wird `unknown`; nur Auf-/Abstieg wird `{state: known, value: 0}`; Distanz und `source_duration_s` bleiben `unknown`                             |
| RNG-04  | Sliderreset, oberer Handle am Domainende sowie beide Handles bei `[50 km, ∞)`                                         | Reset lässt das ganze Filterfeld aus; der obere Endanschlag entfernt nur `lte` und ändert `gte` nicht; `[50 km, ∞)` normalisiert zu `{gte: 50000, missing: exclude}` |
| DIF-01  | fehlende oder ungültige Quellschwierigkeit                                                                            | Projektion und Trefferwert `unknown`, nie `easy`                                                                                                                     |
| DAT-01  | ein Tag beim Zürcher DST-Start                                                                                        | genau der lokale 23-Stunden-Kalendertag über halboffene UTC-Grenzen                                                                                                  |
| DAT-02  | ein Tag beim Zürcher DST-Ende                                                                                         | genau der lokale 25-Stunden-Kalendertag                                                                                                                              |
| DAT-03  | `2028-02-29`                                                                                                          | gültiger inklusiver Schalttag; `2027-02-29` ist ungültig                                                                                                             |
| DAT-04  | unbekannte Zone, fester Offset oder IANA-Alias statt kanonischem Namen                                                | `422 unsupported_time_zone`; keine Betriebssystem-Defaultzone                                                                                                        |
| DAT-05  | bekannter Trail-Instant mit Submillisekunden und zonenloser Altwert                                                   | UTC-Millisekundenwert für Filter/Sort/DTO; zonenlos ohne belegte Zone bleibt Unknown                                                                                 |
| GEO-01  | Mittelpunkt `(0, 0)` und Trail exakt auf Radiusgrenze                                                                 | Filter aktiv; Grenztrail enthalten                                                                                                                                   |
| GEO-02  | nur Latitude oder Radius über Capability-Maximum                                                                      | `incomplete_geo_center` beziehungsweise `unsupported_radius_for_spatial_contract`                                                                                    |
| GEO-03  | capability-aktiver `route_geometry_radius`; Route berührt den Kreis nur in einem mittleren Segment                    | Direct-Ausführung gemäss `route_radius_ux_v1`; kein Startpunktfallback; Response bindet `bounded_approximate` mit `display_qualifier: none`                           |
| GEO-04  | `start_point_radius` zusammen mit `route_geometry_radius` beziehungsweise `{by: route_proximity_m, direction: asc}` im Direct-Modus | `invalid_filter_combination` beziehungsweise `unsupported_sort_for_spatial_contract`; niemals still gewählter Modus oder degradierte Erfolgsantwort                  |
| BOOL-01 | `completed: false`, `liked_by_me: false`                                                                              | echte Negativfilter; liked ohne Authentifizierung ergibt `authentication_required`                                                                                   |

### 17.2 Text, Ranking und Highlights

| ID     | Eingabe beziehungsweise Ereignis                                                                               | Erwartung                                                                    |
| ------ | -------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| TXT-01 | fehlender, leerer oder nur aus Whitespace bestehender Text; Sortierung ausgelassen beziehungsweise `relevance` | normalisiert `""`; Reihenfolge `created_at DESC, trail_id ASC`               |
| TXT-02 | `linde auss` versus `lind aussicht`                                                                            | nur jeweils letzter Term darf Präfix sein; alle Terme müssen matchen         |
| TXT-03 | Tokens mit Länge 4/5/8/9 und numerisches Token                                                                 | Typobudgets exakt 0/1/1/2; numerisch immer 0                                 |
| TXT-04 | gleicher Typo-/Nähegrad in Name, Ort und Beschreibung                                                          | Attributreihenfolge entscheidet Name vor Ort vor Beschreibung                |
| TXT-05 | Anführungszeichen, führendes Minus, unbekanntes Wort                                                           | keine versteckte Phrase-/Negationssyntax und keine Synonym-/Relaxationssuche |
| SRT-01 | nichtleere Query plus explizite Distanzsortierung                                                              | Distanz ist primär; Relevanz bricht erst gleiche Distanzen                   |
| SRT-02 | bekannte und unbekannte Distanz bei ASC und DESC                                                               | Unknown in beiden Richtungen zuletzt                                         |
| SRT-03 | identische Relevanz, Primärwerte und Zeit; andere Einspielreihenfolge                                          | immer `trail_id ASC`                                                         |
| HLT-01 | Match hinter Emoji/Mehrbytezeichen                                                                             | valide Graphemgrenzen; Segmentverkettung ergibt exakt den Ausschnitt         |
| HLT-02 | Dokumenttext `<script>alert(1)</script> Linde`                                                                 | reiner Segmenttext, kein HTML/DOM-Code                                       |
| HLT-03 | Match nur auf unsichtbarem Alias oder nicht sichtbarem Waypoint                                                | kein Fragment, Grund oder Rangsignal; keine Countdifferenz                   |

### 17.3 Kontext, Pagination, ACL und Federation

| ID      | Eingabe beziehungsweise Ereignis                                                                     | Erwartung                                                                                                                                                                                                                                                                                                                                        |
| ------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| CTX-01  | Mutation zwischen Treffer- und Facettenquery eines revision-fenced Batches                           | gesamter Batch Retry oder `503`; niemals gemischte `200`                                                                                                                                                                                                                                                                                         |
| CTX-01b | konkurrierender oder supersedierter Publisher                                                        | verliert vor dem nächsten Engine-Write sein monotones Fencing-Recht; nur vollständig alte oder vollständig neue veröffentlichte Epoch, nie Zwischenstand                                                                                                                                                                                         |
| CTX-01c | Label-, Like-, Excerpt- oder Thumbnailmutation zwischen identischen Cursor-Replays                   | gebundene Content-Epoch wird stale; keine mutable Nachhydration                                                                                                                                                                                                                                                                                  |
| CTX-02  | neue Content-Epoch in der gepinnten Generation zwischen Seite eins und zwei                          | `409 search_context_stale`; eine Epoch einer anderen, später aktivierten Generation ist dagegen unerheblich                                                                                                                                                                                                                                      |
| CTX-03  | aktiver Generationsswap bei unverändert aufbewahrter vorheriger Generation                           | Folgeseite wird über ihre gepinnte Generation bis `valid_until` identisch bedient; neue Kontexte verwenden die neue Generation                                                                                                                                                                                                                   |
| CTX-03b | Index-/Schema-Rollback nach späteren Writes auf der neuen Generation                                 | alter cursorgepinnter Index wird nicht direkt aktiviert oder mutiert; ein getrennter Kandidat wird bis zum aktuellen Cutoff aufgeholt und erst nach Task-/Coverage-/Watermark-Gates aktiviert                                                                                                                                                    |
| CTX-04  | Request exakt vor und bei `valid_until`                                                              | vorher zulässig; bei Gleichheit `410 search_context_expired`                                                                                                                                                                                                                                                                                     |
| CUR-01  | derselbe Cursor zweimal ohne Zustandsänderung                                                        | bytegleich relevante Folgeseite; Cursor ist nicht verbraucht                                                                                                                                                                                                                                                                                     |
| CUR-01b | vorhandener leerer oder mehr als 4096 Bytes langer Cursor                                            | `400 invalid_cursor`; `null` oder Nicht-String bleibt `422 invalid_search_request`                                                                                                                                                                                                                                                               |
| CUR-02  | andere Spec, Seitengrösse oder Includes mit Cursor                                                   | `409 cursor_request_mismatch`                                                                                                                                                                                                                                                                                                                    |
| CUR-03  | manipulierter Cursor und valider Cursor eines anderen Actors                                         | beide identisch `400 invalid_cursor`                                                                                                                                                                                                                                                                                                             |
| CUR-04  | anonymer Cursor nach Login, Actor-/Tenantwechsel oder Logout                                         | `400 invalid_cursor`; Tokenrefresh desselben Actors bleibt gültig                                                                                                                                                                                                                                                                                |
| CUR-05  | MAC-Key-/Formatrotation, Gatewaywechsel oder Deployment zwischen Seiten                              | Expand aller Reader vor neuer Ausstellung; alter Root-Key, Cursorformat, Vertragsartefakte und gepinnte Generation bleiben bis zum jeweiligen `issued_valid_until`-High-Watermark plus Grace verfügbar; erlaubte alte→neue und neue→alte Readerwechsel bleiben gültig, Manipulation bleibt `400`                                                 |
| CUR-06  | maximaler gültiger Request mit allen 16 Aggregationsgruppen und aktiven Federation-/Spatialbindungen | ausgegebener Cursor bleibt höchstens 4096 Bytes; decodierter Payload enthält keinen Suchtext, keine privaten Filter-/Principalwerte, internen Indexnamen oder verborgenen Rankingwerte                                                                                                                                                           |
| CUR-07  | Crash vor/nach Reservierung der Key-/Generations-/Artefakt-High-Watermarks und vor Cursorantwort     | Cursorantwort wird nie vor dauerhafter Reservierung sichtbar; der Crash erzeugt höchstens sichere Überretention                                                                                                                                                                                                                                  |
| PAG-01  | mehr als 1'000 Treffer vollständig bis `total` traversieren                                          | jede ID genau einmal in totaler Reihenfolge; `next_cursor` ist auf jeder nicht letzten Seite gesetzt und erst nach dem letzten Treffer `null`; kein Engine-Cap wird still sichtbar                                                                                                                                                               |
| ACL-01  | Public→Private oder Share-Widerruf während Batch                                                     | erste Seite baut neuen Kontext oder liefert `503`, Cursorseite wird stale; kein Hit, Count, Bucket oder Snippet leakt                                                                                                                                                                                                                            |
| ACL-02  | Kern- und erlaubtes Overlay enthalten dieselbe Trail-ID                                              | genau ein Kandidat vor Ranking, Count und Pagination                                                                                                                                                                                                                                                                                             |
| ACL-03  | Versuch über generischen Legacypfad oder alten Search-/Tenant-Token                                  | kein Umgehen von Fence oder DTO-Allowlist; vor Overlay-Cutover deaktiviert oder gleichwertig gegated                                                                                                                                                                                                                                             |
| ACL-04 (E2E) | Anonymous/authenticated × local/federated × public/private/Direct-Share/revoked/deleted | Public liefert nur lokale öffentliche Inhalte und aktive `verified_public`-Publikationen; private und Direct-Share erscheinen ausschliesslich im belegten Actor-/ACL-Scope; revoked und deleted fehlen aus Hits, Total, Facetten, Histogrammen, Highlights und Matchgründen; anonyme und unberechtigte Requests bestätigen ihre Existenz auch nicht über Fehler oder Timingdiagnosen |
| FED-01  | föderierter Snapshot läuft vor Folgeseite ab                                                         | Kontext endet spätestens an Expiry; kein abgelaufener Treffer                                                                                                                                                                                                                                                                                    |
| FED-02  | Peer ist offline, materialisierter Snapshot weiterhin gültig                                         | Treffer und Counts bleiben stabil; keine Live-Remote-Abfrage                                                                                                                                                                                                                                                                                     |
| CTX-05  | Snapshotressource vor zugesagtem `valid_until` verloren                                              | `503 search_context_unavailable`, nicht `410`                                                                                                                                                                                                                                                                                                    |
| CTX-06  | Suche mit `{scope: origin_instance, origin_instance_id: X}` und angefordertem Histogramm             | Request, `normalized_search.filters.source`, `search_context.source_scope` und Cursor binden denselben vollständigen `SourceSelectorV1`; `statistics_domains[].source_scope` bindet den tatsächlich verwendeten vollständigen Selector und darf nur bei `broader_public_fallback` abweichen; nirgends wird `origin_instance` ohne ID gespeichert |
| CTX-07  | Failover auf eine Replica mit älterer Generation, Epoch, Watermark oder Settingshash                 | Replica bleibt fail-closed; kein Fallback auf einen ungepinnten oder nur teilweise bestätigten Stand                                                                                                                                                                                                                                             |
| CTX-08  | `route_geometry_radius` auf erster Seite und Cursorfolge                                              | `normalized_search`, `search_context.spatial_contracts`, `accuracy.contracts`, Cache-Key und Cursor binden übereinstimmend `route_radius_ux_v1`, S50, r100, 5-km-Segmente, 100-km-Grenze, 50-m-Neutralband und dieselbe snapshotgebundene Route-Shardfamilie; kein Neuauflösen gegen aktuelle Shards |

### 17.4 Counts, URL und Legacy

| ID           | Eingabe beziehungsweise Ereignis                                                                                                               | Erwartung                                                                                                                                                                                     |
| ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CNT-01       | 1'237 passende Trails                                                                                                                          | `total=1237`; kein 1'000er-Cap oder `estimatedTotalHits`                                                                                                                                      |
| CNT-02       | aktive Kategorie und Difficulty, Category-Facette                                                                                              | Kategoriecount entfernt Taxonomie, behält Difficulty und übrige Gruppen                                                                                                                       |
| CNT-03       | aktive Kategorien mit Kindauswahl, Subcategory-Facette                                                                                         | Parentauswahl bleibt; Kindbedingungen werden vertragsgemäss ersetzt                                                                                                                           |
| CNT-04       | Histogrammwert genau auf Bucketgrenze, Overflow plus Unknown                                                                                   | jeder bekannte Wert liegt in genau einem halboffenen beziehungsweise offenen Bucket; Unknown besitzt einen getrennten stabilen Bucket                                                         |
| CNT-04b      | Bucketprofil mit 16 beziehungsweise 17 sichtbaren Buckets einschliesslich Unknown                                                              | 16 ist publizierbar; 17 wird vom Capability-/Startup-Preflight abgelehnt                                                                                                                      |
| CNT-05       | eine angeforderte Facettenquery schlägt fehl                                                                                                   | `200`, `aggregation_status=partial`, nur diese Gruppe ist `error`; keine gecachte Altzahl oder leere Bucketliste                                                                              |
| CNT-06       | `bounded_approximate` Spatial-Vertrag                                                                                                          | Total und jeder Ready-Bucket stimmen ohne Toleranz mit der unabhängigen Direct-Vollmengenauswertung überein; die mögliche räumliche Abweichung wird zentral statt mit `≈` vor jeder Zahl erklärt; G1 prüft gegen die exakte Rohgeometrie ausschliesslich die Total-Reihe mit p95 relativ ≤ 1 %; bei Einführung von Facetten oder Histogrammen entscheidet und qualifiziert SRCH3 deren sichtbare Options- und Bucketreihen mit einem eigenen, für spärliche Buckets geeigneten Budget; diese spätere Evidenz wird nicht rückwirkend G1 zugerechnet; absolute Abweichung bleibt Diagnose; Error-Gruppen zeigen `—` |
| CNT-07       | ausgewählte bekannte Option unter allen übrigen Filtern ohne Treffer                                                                           | Bucket wird mit `selected=true` und `count=0` ausgeliefert und bleibt abwählbar                                                                                                               |
| CNT-07b      | explizit ausgewählte, durch UI-Präferenz ausgeblendete Kategorie                                                                               | Bucket bleibt wirksam und erscheint mit `selected=true`, `visible=false` sowie seinem exakten Count                                                                                           |
| CNT-07c      | mehrere im Engine-Ergebnis fehlende Katalogoptionen                                                                                            | `0` nur nach exhaustiver Distribution oder einer gebündelten gruppenweiten Beweisquery; nie eine Query pro Option                                                                             |
| CNT-08       | begrenzte Tagfacette mit `complete=false`                                                                                                      | jeder gelieferte Count ist exakt; aus einer nicht gelieferten Option wird nie `0` abgeleitet; Auswahlen sind injiziert oder die Gruppe ist Error                                              |
| CNT-09       | Capability-Fixture mit einem SearchRequest aus 17 angeforderten Aggregationen beziehungsweise einem gültigen Plan mit einer 33. internen Query | erster Fall `422 search_space_too_broad`; im zweiten Fall genau die budgetüberschreitende gültige Gruppe `count_query_budget_exceeded`                                                        |
| CNT-09b      | kritischer Treffer-/Totalkern benötigt allein 33 interne Queries                                                                               | gesamte Suche ist `422 search_space_too_broad`; der Kern wird nicht als Aggregationsfehler ausgegeben                                                                                         |
| CNT-10       | Locale- und Labeländerung bei unverändertem Selector/Profil                                                                                    | identische `bucket_id`; Änderung der registrierten Grenzleiter oder Bucketsemantik erzeugt ein neues Profil beziehungsweise eine neue ID                                                      |
| CNT-10b      | anderer p99-Endanschlag bei unveränderter registrierter Grenzleiter                                                                            | gleiches `bucket_profile`; `open_from` und `statistics_domains[].revision` binden den neuen Kontext                                                                                           |
| CNT-11       | 499/500 bekannte Werte und 79,9/80 Prozent Coverage                                                                                            | unter einer Grenze versionierter Fallback; ab beiden Grenzen p99-Aufrundung gemäss festem Profil und UI-Cap                                                                                   |
| CNT-11b      | Nearest-Rank-p99 oberhalb des UI-Caps                                                                                                          | `open_from` entspricht dem UI-Cap; höhere bekannte Werte bleiben vollständig im Overflow-Bucket                                                                                               |
| CNT-11c      | gleiche Familie mit `local`, `federated`, `all` und konkreter Origin                                                                           | jede p99-Auswahl bindet den tatsächlich verwendeten normalisierten Statistik-Scope; ein abweichender Request-Scope ist nur mit `broader_public_fallback` zulässig und bleibt separat sichtbar |
| CNT-12       | Treffer-, Total-, Snapshot- oder Fence-Auswertung schlägt fehl                                                                                 | gesamte Suche ist Fehlerantwort; der kritische Kern wird nie zum Gruppenfehler degradiert                                                                                                     |
| URL-01 (E2E) | beliebige gültige Spec → URL → Spec                                                                                                            | identische normalisierte Spec und deterministisch identische kanonische URL                                                                                                                   |
| URL-02 (E2E) | doppelte Scalar-Keys, unbekannter Key oder malformed Escape                                                                                    | typisierte ungültige URL; keine Suche mit entfernten Parametern                                                                                                                               |
| URL-03 (E2E) | Cursor, Seite oder Includes vorhanden                                                                                                          | nicht in Share-URL; nur Cursorfolge optional in `history.state`                                                                                                                               |
| URL-04 (E2E) | vollständiges `route_near.*` plus `sort=route_proximity_m%3Aasc`                                                                               | `unsupported_sort_for_spatial_contract`; Browser, SSR und URL-Auflöser führen keine degradierte Suche aus                                                                                    |
| URL-05 (E2E) | vollständiges `route_near.*` mit `route_near.radius_m=100001`                                                                                  | `unsupported_radius_for_spatial_contract`; kein Clamp auf 100 km, kein Startpunkt- oder Legacyfallback                                                                                       |
| LEG-00 (E2E) | alte Liste beziehungsweise Karte ohne gespeicherten Zustand                                                                                    | `created ASC` beziehungsweise `created DESC` im Legacyadapter; kein stiller V1-Default                                                                                                        |
| LEG-01 (E2E) | keine Such-URL, gültiger `trailListFilter`, separate Sort-/Page-Keys                                                                           | historische Präzedenz, danach eindeutige V1-Übersetzung                                                                                                                                       |
| LEG-02 (E2E) | `author`, `category` oder `subcategory` in Legacy-URL plus Storagefilter                                                                       | kompletter `trailListFilter` ignoriert; separate Sort-/Page-Keys bleiben wirksam                                                                                                              |
| LEG-03       | gleiche versus verschiedene Parents bei Kategorie/Subkategorie                                                                                 | nur Kindbranch beziehungsweise `X/* OR parent(Y)/Y`                                                                                                                                           |
| LEG-03b      | `__no_subcategory__:cat_hiking` und unbekannter Parent                                                                                         | `cat_hiking/@absent` beziehungsweise blockierende Unresolved-Diagnose                                                                                                                         |
| LEG-04       | eindeutiger, unbekannter und mehrdeutiger Altname                                                                                              | ID-Auflösung beziehungsweise `legacy_filter_unresolved`/`legacy_filter_ambiguous`; nie still entfernen                                                                                        |
| LEG-05       | Difficulty `[0,1,2]`, `liked=false`                                                                                                            | beide Filter ausgelassen; Unknown-Korrektur als Warning                                                                                                                                       |
| LEG-06       | Range-Maximum gleich historischem Limit                                                                                                        | offene Obergrenze statt festem alten Maximalwert                                                                                                                                              |
| LEG-06b      | Slider-Minimum `1234.7`, Maximum `9876.2` unter Limit                                                                                          | inklusive `gte=1234`, `lte=9877`, `missing=exclude`                                                                                                                                           |
| LEG-07 (E2E) | Legacydatum über DST oder Koordinate `0`                                                                                                       | dokumentierte neue Semantik plus jeweiliges Warning                                                                                                                                           |
| LEG-08 (E2E) | `page=3`, keine gespeicherte Seitengrösse                                                                                                      | Offset 50 mit Grösse 25 und `legacy_page_size_assumed`                                                                                                                                        |
| LEG-09       | `?q=foo&sort=-distance` ohne `sv`                                                                                                              | beide im Legacy-URL-Kontext wirkungslos                                                                                                                                                       |
| LEG-10 (E2E) | `sv=1` plus Legacy-Suchkey                                                                                                                     | `mixed_search_url_versions`; keine Zustandskombination                                                                                                                                        |
| LEG-11 (E2E) | alte Kartenbounds über Antimeridian oder mit Koordinate `0`                                                                                    | korrekter View-State, aber keinerlei Suchfilterwirkung                                                                                                                                        |
| LEG-12 (E2E) | ungültiges `trailListFilter`-JSON                                                                                                              | nur defekten Key entfernen, Legacydefault und `legacy_storage_invalid`                                                                                                                        |
| ENG-01       | gleiche Golden Fixtures vor/nach Engine-, Settings- oder Compilerwechsel                                                                       | identische normalisierte Spec, Treffermenge, Reihenfolge, Counts, Highlights und Fehlercodes                                                                                                  |

### 17.5 Options-Lookup und Cache

| ID     | Eingabe beziehungsweise Ereignis                                                             | Erwartung                                                                                                                                                                                     |
| ------ | -------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| OPT-01 | Taxonomielookup mit gemappten, `absent`-, `unmapped`- und `ambiguous`-Knoten                 | vollständige Parent-Kind-Hierarchie, autoritative Selector und Bucket-IDs, lückenlose `sort_order` je Geschwisterliste                                                                        |
| OPT-02 | Label-, Icon-, Hierarchie-, Reihenfolge- oder Sichtbarkeitsänderung                          | neue `options_revision` und neuer `ETag`; unveränderte Revision mit passendem `If-None-Match` ergibt `304`                                                                                    |
| OPT-03 | explizit ausgewählte UI-verdeckte Option und ACL-verbotene Option                            | erste bleibt `visible=false` enthalten; zweite fehlt vollständig und wird nicht bestätigt                                                                                                     |
| OPT-04 | Origin-Option in Lookup und anschließendem SearchRequest                                     | Lookup liefert vollständigen `SourceSelectorV1`; Request und Suchkontext reproduzieren ihn bytegleich kanonisch                                                                               |
| OPT-05 | Count-Gruppe mit einem Lookup, dessen Capability-, Options- oder Sortprofilrevision abweicht | keine Kombination; Lookup neu laden, bei fortbestehendem Mismatch denselben vollständigen SearchRequest ohne Cursor ab Seite eins wiederholen; erst bei drei exakten Bindings Counts anzeigen |
| CAC-01 | gleiche Spec mit anderem Include, anderer Capabilityrevision oder anderem Bucketprofilhash   | getrennte `search_response_cache_v1`-Keys; kein Treffer auf die jeweils andere Response                                                                                                       |

### 17.6 Search-Readiness

| ID | Zustand beziehungsweise Ereignis | Erwartung |
| --- | --- | --- |
| RDY-01 | Engine, vollständiges Profilset, autoritativer Routingzustand und vom Build unterstützte Vertragsrevision stimmen; alle vom aktuellen Owner verlangten Gates sind grün | beide Endpunkte liefern `200`, `search-readiness.v1`, `ready` und `search_ready` |
| RDY-02 | anderer Binary-/Containerbuild unterstützt dieselbe aktive Vertragsrevision | Suche bleibt ohne buildgebundene Reattestierung ready |
| RDY-03 | laufender Build unterstützt die aktive Vertragsrevision nicht | `503 search_contract_unsupported`; keine Suchroute erreicht die Engine |
| RDY-04 | Engine unerreichbar, erforderlicher Index fehlt oder Settings weichen ab | exakt der gemäss Präzedenz bestimmte Readinesscode; kein nachrangiger Zustand leakt in den Body |
| RDY-05 | DB-Prozess lebt, Suche ist nicht ready | DB `GET /health` bleibt Liveness; beide Search-Readiness-Endpunkte bleiben `503` |
| RDY-06 | der autoritative Owner meldet Initialisierung, unvollständige Zustellung oder Recovery | `503 search_recovery_required`, bis ausschliesslich dieser Owner seine normativen Gates wieder grün setzt |
| RDY-07 | externe Audit-, Reindex- oder Run-Dateien fehlen, der autoritative Control-Plane-Zustand ist jedoch vollständig und grün | Readiness bleibt grün; externe Dateien sind kein System of Record |
| RDY-08 | Settings passen, der autoritative Zustand nennt aber eine andere Profil- oder Vertragsrevision | `503 search_state_mismatch`; Settingsgleichheit repariert den Zustand nicht still |
| RDY-09 | partieller oder mehrdeutiger Handoff-, Swap-, Rebuild- oder Publisherzustand | `503 search_recovery_required`; kein API-Readinesspfad führt selbst eine Kompensation aus |
| RDY-10 | ein zuvor grüner Owner schliesst sein Freshness-, Delivery- oder Securitygate | beide Readiness-Endpunkte werden `not_ready`; laufende und neue Searchconsumer folgen der vom Owner zugesagten Invalidierungsgrenze |
| RDY-11 | Web-Guard ist `not_ready` | Einzel-, Multi-, Profil-, Cluster-, Actor- und Hilfsrouten liefern `503 search_unavailable` ohne Engineaufruf |
| RDY-12 | Engine ist unerreichbar oder ein erforderlicher Index fehlt, sodass das beobachtete vollständige Profilset nicht berechnet werden kann | `observed_settings_fingerprint` fehlt; die Antwort erfindet keinen Teilfingerprint |
| RDY-13 | Deployment mit mehreren erforderlichen Rollen, vertauschter Eingabereihenfolge, duplizierter Rolle sowie semantisch gleichen Settingsobjekten mit anderer Propertyreihenfolge | beide Endpunkte emittieren dasselbe streng sortierte, duplikatfreie `profile_revisions`-Array und denselben RFC-8785-Golden-Fingerprint; die Schema- plus Kanonizitätsvalidierung verwirft ein dupliziertes oder unsortiertes Wirearray, eine Änderung von Rolle, Profilrevision, Primärschlüssel, explizitem Default oder wirksamem Setting ändert den Fingerprint |

IDX0 besitzt für den Legacyowner die Produceranteile von `RDY-01` bis
`RDY-09` sowie `RDY-12` und `RDY-13`. SRCH-COMP besitzt die Consumeranteile
von `RDY-03`, `RDY-10` und `RDY-11`, insbesondere den Nachweis, dass bei Rot
kein Token- oder Enginezugriff stattfindet.

## 18. Lieferartefakte und Änderungsregel

Vor produktiver Freigabe von `trail-search.v1` existieren aus demselben Quellvertrag generiert oder gegeneinander geprüft:

1. JSON Schema für Request, Response, Capabilities, `TrailSearchOptionsV1`, `SearchReadinessV1` und Problem Details mit den jeweils oben festgelegten `additionalProperties`-Regeln;
2. OpenAPI-Beschreibung des Search-, Capability-, der vier Options- und beider Search-Readiness-Endpunkte sowie aller stabilen HTTP-/Fehlercodes;
3. eine einzige serverseitige Normalisierungs-/Compilerbibliothek für Treffer, Total, Facetten und Histogramme;
4. URL-Codec-Fixtures mit Parse-, Canonical- und Roundtripfällen;
5. Ranking-/Highlight-Golden-Fixtures gegen die konkret gepinnte Engineversion;
6. Legacy-Fixtures aus realen URLs und serialisierten Browserzuständen;
7. ACL-/Overlay-/Federation-Integrationstests und die Akzeptanzmatrix aus Abschnitt 17;
8. Lasttests über Meilis Standard-Paginierungsgrenze hinaus bis zum Ende der exakten Treffermenge sowie über dem maximal angebotenen Multi-Select-/Aggregationsbudget;
9. die im [Cursor-/Snapshot-ADR](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/) festgelegten Deep-Paging-, Retention-, Deployment-, Mutation-Soak- und Failover-Tests;
10. die Transition-, Invarianten-, Crash-Recovery- und Cutover-Suite des [Federation-, Index- und Gateway-Zustandsvertrags](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/);
11. die `SEC-VIS`-/`SEC-AUTH`-/`SEC-DUR`-/`SEC-SCOPED`-, P1/P2- und Legacy-Migrationssuite aus den [Federation-Sicherheits- und Publikationsvoraussetzungen](/develop/specs/trail-search/contracts/federation-security/).

Eine Änderung, die eine bestehende gültige Spec anders matcht, filtert, sortiert, zählt, paginiert oder migriert, ist kein redaktioneller Fix. Sie benötigt mindestens einen neuen Untervertrag und Golden Fixtures; bei beobachtbarer Request-/Response-Semantik eine neue Search-Contract-Version. Additive Responsefelder bleiben zulässig, sofern alte Clients sie ignorieren können und weder Fingerprint noch bestehende Feldbedeutung verändert werden.
