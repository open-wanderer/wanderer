---
title: CONTEXT — Räumliche und externe Kontexte
description: Ortsanker, Routengeometrie, Anstiege, Zugang, Wetter und Wegbeschaffenheit für die Trail-Suche.
editUrl: false
sidebar:
  order: 3
  badge: Entwurf
spec:
  id: CAP-CONTEXT
  kind: capability
  status: draft
  capability: CONTEXT
  lastReviewed: '2026-08-30'
---

Dieses Dokument spezifiziert `CONTEXT`: räumliche, segmentbasierte und externe
Fakten, die den stabilen Trail-Katalog ergänzen. Allgemeine Request-, Count-,
Snapshot-, ACL-, Federation-, Cursor- und Publikationsinvarianten stehen in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/), im
[Trail-Suchvertrag](/develop/specs/trail-search/contracts/trail-search-v1/) und
im [Control-Plane-Vertrag](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/).
Lieferreihenfolge und Beitragsschnitte gehören in den
[Delivery-Katalog](/develop/specs/trail-search/delivery/). Messhistorie und noch
zu kalibrierende Werte stehen in
[Evidenz und Kalibrierung](/develop/specs/trail-search/evidence/evidence-and-calibration/).

## Ortssuche und räumliche Anker

Das Ortsfeld löst Text zu einem typisierten räumlichen Anker auf und ist nicht
auf Siedlungen beschränkt. Der Benutzer wählt einen harten Scope:

- alle;
- Orte und Regionen;
- Unterkünfte;
- Sehenswürdigkeiten und Kultur;
- Naturziele;
- Essen und Trinken;
- Haltestellen und Bahnhöfe;
- benannte Parkplätze.

Bei `Linde` im Scope `Unterkünfte` darf beispielsweise „Zur Linde“ erscheinen,
nicht aber die Siedlung „Lindenberg“. Der Scope wird beim Provider angewendet,
nicht nachträglich auf eine bereits gekappte Ergebnismenge. `Alle` ist eine
einzige unbeschränkte Suche mit typisierten, gruppierten Treffern und kein
Fan-out über sämtliche Scopes.

```text
PlaceSearchRequest
  query
  scope
  locale / limit
  proximity?              # nur Ranking-Bias
  viewport?
  viewport_mode?          # bias | bounded

PlaceSearchResponse
  provider / dataset_version / attribution
  results[]
    id                    # provider-namespaced, opak
    osm_ref?
    name / label
    point                 # {lat, lon}, verpflichtend
    bbox?
    anchor_kind           # point | area
    geometry_ref?
    scope / subtype
```

Wanderer besitzt eine versionierte Place-Taxonomie. Adapter bilden
Providerwerte darauf ab; dieselbe Taxonomie steuert die Icons. Der URL-State
einer Auswahl enthält provider-namespaced ID, optionalen OSM-Verweis, Typ,
Label, Punkt, Bounding Box und Geometriereferenz.

Der Mindestsatz für `accommodation` umfasst die OSM-Werte
`tourism=hotel`, `hostel`, `guest_house`, `apartment`, `chalet`, `camp_site`,
`alpine_hut` und `wilderness_hut`. Ein Adapter darf weitere Werte versioniert
ergänzen, aber keinen dieser Werte still einer anderen Hauptklasse zuordnen
oder aus einem als vollständig deklarierten Accommodation-Scope entfernen.

Eine langlebige Flächensemantik verlangt einen unveränderlichen
`place_anchor_snapshot` mit Provider-/Objekt-ID, Datasetversion, Geometrie und
Hash, Attribution/Lizenz sowie Erstell- und Retentionsdaten. Eine gespeicherte
Suche geocodiert beim Laden nicht erneut. Ist ein bewusst abgelaufener Snapshot
nicht mehr auflösbar, verlangt die UI eine neue Auswahl; sie fällt nie still
von einer Fläche auf Bounding Box oder Repräsentativpunkt zurück.

Die sichtbare Wahl zwischen Startpunkt- und Routenmodus ist ein
UI-/Compiler-Discriminator, kein zusätzliches öffentliches Suchvertragsfeld:

- der Startpunktmodus kompiliert zum bestehenden `filters.start_point_radius`;
- der Routenmodus kompiliert einen Punktanker zu
  `filters.route_geometry_radius`.

Bestehende Startpunkt-URLs behalten ihre Bedeutung. Der Routenmodus wird immer
explizit und additiv gewählt.

### Geocoding- und POI-Domänen

`geocoding` und `pois` sind getrennte Plugin-Domänen mit gemeinsamer
Place-Taxonomie:

- `geocoding` löst Namen über `search.v1`, `autocomplete.v1`, `reverse.v1` und
  `area_geometry.v1` in wenige auswählbare Anker auf;
- `pois` durchsucht oder synchronisiert Inventare, Kategorien, Gebiete und
  Routenkorridore über `search.v1`, `dataset_sync.v1`, `detail.v1` und optional
  `route_corridor.v1`.

Produktpfade prüfen effektive Capabilities, nie eine Provider-ID. Effektive
Capabilities sind die Schnittmenge aus Typvertrag, Deployment-/Importprofil
und Nutzungsrichtlinie. Kategorisiertes Autocomplete wird nur freigegeben, wenn
das gepinnte Profil positive und negative Fixtures, Abdeckung, Aktualisierung
und Betriebsziele erfüllt.

Eine konfigurierte Geocoding-Instanz besitzt mindestens diese explizite
Betriebs- und Capability-Semantik:

```text
plugin.type = geocoding
plugin.id = nominatim | photon | ...
instance.scope = system
instance.usage_profile = osmf_public | managed | self_hosted
instance.endpoint = ...

EffectiveCapabilities
  = Typvertrag ∩ Deployment-/Importprofil ∩ Nutzungsrichtlinie

search.v1 / autocomplete.v1 / reverse.v1 / area_geometry.v1
category_filter:
  scope_taxonomy_version
  supported_scopes[]
  supports_scope_union
  filter_precision
  dataset_import_profile_version
```

`category_filter` und `autocomplete.v1` sind unabhängige Zusagen. Ein Adapter
darf einen harten Scope nur dann anbieten, wenn genau dieser Scope im aktiven
Import-/Deploymentprofil qualifiziert ist; ein Kategorievertrag verleiht ihm
nicht automatisch Autocomplete. Die breiten Nominatim-Filter `layer` und
`featureType` reichen insbesondere nicht automatisch für exakte Unterkunfts-,
Sehenswürdigkeits- oder Parkplatz-Scopes.

Photon ist der vorgesehene First-Party-Referenzadapter für kategorisiertes
Autocomplete, aber keine Wanderer-, GeoJSON- oder Routenradius-Voraussetzung.
Pelias oder ein geeignetes selbst gehostetes Nominatim-Profil können dieselbe
Produktscheibe freischalten, wenn sie denselben Capabilityvertrag bestehen.
Öffentliche Demoangebote sind keine Produktionsgarantie. Fehlt eine zugesagte
Capability, bleibt nur die davon abhängige neue Scheibe verborgen; die
Administrationsdiagnose nennt Instanz, fehlende Capability und Profilgrund.

Der öffentliche OSMF-Nominatim-Dienst ist ein Submit-only-Kompatibilitätsprofil.
Er verlangt einen wirklich instanzweiten, über Benutzer und Replikas
serialisierten Egress-Limiter, Cache/Singleflight, Identifikation, Attribution
und korrektes `429`-/`Retry-After`-Verhalten. Verboten sind verstecktes
Autocomplete, systematisches Kategorie-Browsing, Backfills und periodische
Jobs. Auch andere Endpoints erhalten ein explizites Nutzungsprofil und gelten
nicht automatisch als unbegrenzt. UI und Betriebsdokumentation nennen
Provider, Attribution und den möglichen Abfluss des Suchtexts.

## Routeweiter Punktradius

Eine Route ist ein Treffer, sobald mindestens ein reales Routensegment den
Radius um den gewählten Punkt berührt. Start und Ziel allein genügen nicht.
Leere Segmente werden beim Aufbau verworfen. Enthält eine GPX-Datei weniger
als zwei valide Punkte, ist ausschliesslich die Routenberührung `unavailable`;
ein weiterhin bekannter Startpunkt bleibt über den getrennten
`start_point_radius` auffindbar.

Die angenommene
[GeoJSON-Direct-ADR](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
definiert `route_radius_ux_v1`:

```text
0 < radius_m <= 100000
```

Grössere oder ungültige Radien ergeben
`unsupported_radius_for_spatial_contract`. GeoJSON Direct ist das Endprädikat;
es gibt im Requestpfad keine SQLite-Hydration und keine versteckte exakte
Punkt-zu-Polyline-Nachprüfung. Die getrennte Route-Indexfamilie verwendet
S50-RDP-Vereinfachung, geodätische Verdichtung auf höchstens 5 km lange
Segmente, Antimeridian-Splitting und `_geoRadius(..., 100)` ohne Padding.

Das Ergebnis ist bewusst `bounded_approximate`: eine empirische
Produktqualifikation gegen ein festgelegtes Korpus, keine geometrische
Einzelfehlergrenze. Das neutrale 50-m-Grenzband ist eine Auswertungsregel und
keine Hausdorff-Garantie. Response und Suchkontext binden Vertragsversion,
`max_radius_m`, `product_boundary_tolerance_m`, Plan/Resolution, Geometry-Hash
sowie Vereinfachungs- und Segmentierungsversion, aber keinen erfundenen
`computed_algorithmic_error_bound_m`.

`total`, zurückgegebene Seite, `next_cursor` und Reihenfolge sind exakt für die
Direct-Menge desselben gepinnten Kontexts; V1 gibt keine Seitenzahl oder
Gesamtseitenzahl aus. Gegenüber dem mathematischen Rohgeometrie-Oracle dürfen
sie abweichen. Linien-Proximity-Sortierung ist nicht unterstützt und ergibt den
typisierten Spatial-Fehler, statt still nach Startpunkt zu sortieren. Echte
Flächengeometrie und Kartenausschnitt-Schnitt benötigen eigene qualifizierte
Verträge.

## Anstiege

Gesamthöhenmeter pro Kilometer verdecken einzelne harte Rampen. `CONTEXT`
modelliert deshalb richtungssensitive Anstiege statt einer globalen
„Steilheit“. Eine Richtungsumkehr ändert den Geometry-Hash und erzeugt eine
neue Analyse.

Der Server validiert je geordnetem GPX-Segment die Höhenabdeckung, resamplet
und glättet distanzbasiert, erkennt Täler/Gipfel über Hysterese und Prominenz,
führt begrenzte Flachstücke und Gegenabfahrten zusammen und klassifiziert mit
einem versionierten Aktivitätsprofil. Dieses bindet die Familie und, wo das
Modell sie benötigt, die Disziplin aus
[ADR 0003](/develop/specs/trail-search/decisions/0003-walk-hike-cycle-personal-evaluation/).

Das Analyseprofil legt insbesondere Resamplingdistanz, Glättung,
Mindestprominenz, maximale Gegenabfahrt und maximale Flachdistanz fest. Als zu
kalibrierender Ausgangsbereich gelten 20 bis 30 m Resamplingdistanz. Für jeden
erkannten Anstieg gilt:

```text
length_m       = horizontale Wegdistanz von Start bis Gipfel
avg_grade_pct  = 100 * net_gain_m / length_m
gross_gain_m   = Summe aller positiven Höhenänderungen nach der Glättung
```

Die Definition des 100-m-Fensters, der Umgang mit kürzeren Randfenstern,
doppelten Punkten und fehlenden Höhenwerten sowie Merge-Grenzen für
Gegenabfahrt und Flachdistanz sind explizite Bestandteile der
`analysis_version`. Eine Versionsänderung erzeugt eine neue Analyse und deutet
alte Werte nicht still um.

```text
ordinal
start_distance_m / end_distance_m
start_coordinate / summit_coordinate
start_elevation_m / summit_elevation_m
length_m / net_gain_m / gross_gain_m
avg_grade_pct / max_grade_100m_pct / p95_grade_100m_pct
score / score_model / climb_class
activity_family / activity_discipline
classification_profile / classification_version
source_segment
```

Nur Rollups wie Anzahl, maximale Höhe/Länge, härtester Score, anhaltende
Steigung, Klassen, Coverage, Quelle und Analyse-/Profilversion gelangen in den
Hauptindex. `climb_count = 0` bedeutet eine erfolgreiche Analyse ohne Anstieg;
fehlende Höhendaten ergeben `partial` oder `unavailable`.

Schwellen, die für denselben Anstieg gelten, dürfen nicht als unabhängige
Trail-Prädikate kompiliert werden. Feste UI-Kombinationen können versionierte
Compound-Tokens verwenden. Frei kombinierbare kontinuierliche Schwellen
benötigen einen Anstiegsindex oder den gemeinsamen Search-Orchestrator.
Herstellerformeln sind Kalibrierreferenzen und werden nie als identische
Wanderer-Skala ausgegeben.

## Zugang, Transit und Parken

„ÖV erreichbar“ ist zu grob. Das Modell trennt:

- Start, Ziel, beide oder eine ausdrücklich separate Ausstiegsoption entlang
  der Route;
- Bahn, Bus, Tram/Metro, Seil-/Standseilbahn und Fähre;
- ÖV, Auto- und Veloparkplatz;
- geroutete Gehdistanz/-zeit von Luftlinie als Vorfilter;
- physische Infrastruktur, planmässige Bedienung und konkrete Verbindung;
- statische Erreichbarkeit von einer Verbindung für Datum und Uhrzeit.

Die Routenform stammt als objektiver Fakt aus `DERIVED`; `CONTEXT` verwendet
sie nur für die Zugangssemantik. Bei einem Rundweg genügt typischerweise ein
geeigneter Zugang am kollokierten Start-/Zielpunkt. Bei einer Streckentour
bedeutet „mit ÖV machbar“ standardmässig, dass Start **und** Ziel passend
bedient werden. Haltestellen entlang der Route sind getrennte
Ausstiegsoptionen und dürfen dieses Endpunktfilterergebnis nicht verändern.

```text
trail_id
endpoint                  # start | finish | along_route
access_kind               # transit | car_parking | bike_parking
mode
facility_id / facility_name
air_distance_m
walk_distance_m / walk_duration_s
service_status            # scheduled | physical_only | unknown
public_access
capacity? / fee?
source / source_updated_at
geometry_hash / algorithm_version
```

OSM liefert physische Infrastruktur; statische Transitdatasets liefern
Haltestellen, Modi und planmässige Bedienung. Sie werden periodisch importiert
und lokal angereichert. Suchrequests rufen weder Overpass noch einen
Fahrplanprovider pro Trail auf. Ein verzeichneter Parkplatz verspricht keinen
aktuell freien Platz.

Statischer Transit und Live-Journey-Planning sind getrennte, vollständige
Aussagen. Ein statischer Treffer darf bediente Endpunkte und Counts je Modus
ausweisen. Eine Live-Verbindung verlangt `journey_plan.v1`, Datum, Zeitzone,
Provider-Gültigkeit und explizite Fehlersemantik. Fehlende Livedaten löschen
die statische Aussage nicht und verwandeln sie nicht in eine konkrete
Verbindung.

Der Journey-Provider darf eine andere `transit`-Instanz als das statische
Schedule-Dataset sein. In diesem Fall muss er dieselbe Region abdecken und die
Abbildung seiner Stop- und Agency-Namespaces auf das aktive statische Dataset
vollständig erklären. Ein atomarer Kompatibilitätsbeleg verhindert die
Auswertung von Realtime- oder Journey-IDs gegen den falschen Fahrplan.
Ergebnisliste und Detailseite lesen je freigegebener Produktscheibe dieselbe
persistierte beziehungsweise kontextgebundene Berechnung und dürfen nicht zu
unterschiedlichen Erreichbarkeitsaussagen gelangen.

## Provider- und Plugin-Domänen

Top-Level-Typen stehen für stabile Host-Subsysteme, Capabilities für optionale
versionierte Operationen und Plugin-IDs für Implementierungen. Es gibt keine
providerbezogenen Top-Level-Typen.

| Typ | Typischer Scope | Wesentliche Capabilities | Beispielimplementierungen |
| --- | --- | --- | --- |
| `trails` | Benutzer | bestehende Trail-/Activity-Import- und Sendefunktionen | Komoot, Strava, Hammerhead |
| `assets` | Benutzer | `asset_library.v1` | Immich; später weitere Fotoquellen |
| `routing` | Systemangebot plus Benutzerwahl/-profile | `route.v1`, `round_trip.v1`, `elevation.v1`, `maneuvers.v1`, `profile_introspect.v1`, `profile_prepare.v1`, später `map_match_attributes.v1` und `map_dataset_metadata.v1` | Valhalla, BRouter, später GraphHopper |
| `geocoding` | primär System, optional Benutzer-Credentials | `search.v1`, `autocomplete.v1`, `reverse.v1`, `area_geometry.v1` | Nominatim, Photon, Pelias |
| `pois` | primär System | `search.v1`, `dataset_sync.v1`, `detail.v1`, optional `route_corridor.v1` | Overpass, 1NiteTent, iOverlander, Watrify |
| `weather` | primär System | `forecast.v1`, `seasonal_conditions.v1`, optional `observations.v1` | Wetter- und Klimadatenprovider |
| `transit` | System, mehrere regionale Instanzen | `dataset_sync.v1`, `realtime_sync.v1`, `journey_plan.v1` | GTFS-Feeds, OpenTripPlanner, regionale APIs |

Diese Registry beschreibt stabile fachliche Domänen und beispielhafte
Adapter. Sie verpflichtet weder eine Installation noch einen Releaseschnitt,
alle Typen oder Adapter gemeinsam auszuliefern.

`transit` ist der Domainname, weil GTFS Schedule/Realtime Formate sind und
Journey-Provider andere Schnittstellen besitzen können. Realtime bindet über
`dataset_ref` an ein kompatibles aktives Schedule-Dataset samt Feedversion und
Agency-/Namespace-Mapping.

Grosse GTFS-, POI-, Karten- oder Klimadatasets verwenden begrenzte
Artefaktpläne. Der Host validiert unveränderliche Manifeste mit Quelle,
Revision, Datenhorizont, Hash und Coverage und aktiviert sie atomar; sie werden
nicht als unbegrenzte synchrone WASM-Antwort übertragen. Gemeinsame
Infrastruktur liefert typisierte Manifestvalidierung, System-/User-Scopes,
durable Lease-Jobs, TTL-/Singleflight-Caches, sichere Connectoren, Deadlines
und Ressourcenbudgets.

## Saison und Wetter

Wetter wird nie als dauerhafte Trail-Eigenschaft in Meilisearch gespeichert.
Es kombiniert stabilen Routendemand, Richtung, Anstiege und Surface-Fakten mit
Ausflugsprofil, Startzeit, erwartetem Fortschritt und versionierten
räumlich-zeitlichen Providerdaten.

- Der **Saisonmodus** beschreibt typische Bedingungen nach Monat, Höhe,
  Exposition, Tageslicht, Klimatologie und statistischen
  Schnee-/Hitze-/Nässe-/Eisindikatoren.
- Der **Prognosemodus** beschreibt eine konkrete Startzeit und erwartete
  Routenpositionen anhand von Windrichtung/-stärke, Temperatur, Niederschlag
  und unterstützten Zustandsrisiken.

Beide nennen Quelle, Gültigkeitszeitraum, räumliche Abdeckung, Unsicherheit und
beitragende Faktoren. Eine Kurzfristprognose ersetzt kein Saisondataset. Es
sind Planungshilfen, keine Sicherheitsgarantien.

Unabhängige Evaluatoren laufen im gemeinsamen Orchestrator. Er wendet zuerst
stabile ACL-/Federation-/Textfilter an, berechnet Basis-ETA für den vollständigen
Kandidatenraum, liest Evaluator-Caches gebündelt entlang der Route und
erwarteten Durchgangszeit, iteriert ETA/Demand bis zur versionierten Grenze und
wendet erst danach dynamische Filter, globale Sortierung, Counts, Histogramme
und Pagination an. Er bewertet nie nur die erste Seite. Budgetüberschreitung
liefert einen expliziten Hinweis zur Eingrenzung des Suchraums.

Fehlende, abgelaufene oder räumlich unzureichende Daten ergeben
`context_status = unknown`; unbekannter Kontext kann sichtbar einbezogen
werden. Die statische ETA bleibt sichtbar, wird aber nicht als wetterangepasst
bezeichnet. Harte Gruppengrenzen verwenden die strengste bekannte Grenze von
Teilnehmenden und Ausrüstung, nicht den Mittelwert. Vom gewählten Template
vorgeschlagene oder als Diff überschriebene Wind-, Hitze-, Kälte- und
Tageslichtgrenzen werden als Planungsannahmen erklärt; der Wetterkontext
erhebt daraus keine medizinische oder objektive Sicherheitszusage.

## Wegbeschaffenheit und Wegtyp

Surface ist eine asynchrone, serverseitige und persistierte Routenanalyse.
Suche, Indexrebuild, Detailansicht und Browser lösen weder Map-Matching,
Overpass noch Providerzugriffe aus.

```text
kanonische GPX-Segmente
  -> routing/map_match_attributes.v1
  -> OSM-Attribute aus lokalem oder gecachtem Snapshot
  -> versionierte Normalisierung
  -> persistierte Segment-Runs und Rollups
  -> Suchprojektion, Filter, Profil und Erklärung
```

Der Matcher nutzt permissive, versionierte Aktivitätsprofile; Wandern umfasst
den vollständig vorgesehenen `sac_scale`-Bereich, Cycle bindet zusätzlich die
Disziplin. Lokale Kategorie-IDs, elektrische Unterstützung und die Fähigkeit
des aktuellen Benutzers beeinflussen das Matching nie. GPX-Segmente behalten
die gemeinsame Chainage. Für die Aktivitätsfamilie `unknown` sowie für Cycle
mit unbekannter Disziplin wird ein definiertes Profil-Ensemble ausgewertet,
statt willkürlich einen Strassen- oder Touring-Typ zu bevorzugen;
Parallelweg-Fixtures prüfen die Auswahl. Segmente werden mit kontrollierter
Überlappung
geteilt und so dedupliziert, dass gilt:

```text
matched_m + unmatched_m = analysed_gpx_m
```

Valhallas `edge.length` und durch das Matching verursachte Umwege sind reine
Diagnosewerte und nie Nenner dieser Gleichung. Gaps, wiederholte Kanten oder
Batchüberlappungen dürfen die auf der analysierten GPX-Chainage basierenden
Anteile nicht verändern.

Matcher-Graph und OSM-Tags sind getrennte Snapshots. Providerclaims und
Overrides erfüllen dieselben Taxonomie-, Provenienz-, Coverage- und
Konfliktregeln. Die deterministische Priorität lautet: explizites versioniertes
Segment-Override, kompatibler exakter OSM-Snapshot, vertrauenskonfigurierter
Providerclaim, grober Matcher-Fallback. Föderierte Geometrie wird lokal
analysiert; Remoteanalysen bleiben getrennte attributierte Claims und werden
nicht unmarkiert in lokale Counts gemischt.

Ein föderierter Trail wird nur dann lokal mit der aktiven Surface-Version
analysiert, wenn seine Geometrie lokal verfügbar ist. Ohne lokale Geometrie ist
die lokale Analyse `unavailable`; ein fremder Surface-Claim bleibt mit Herkunft
getrennt und fliesst nicht unmarkiert in lokale Filter oder Counts ein.

Ist ein referenzierter OSM-Way zwischen Graph- und Tag-Snapshot gelöscht oder
gesplittet, oder überschreitet deren Datenstandsabstand das erlaubte Budget,
wird der betroffene Abschnitt `partial` statt scheinbar exakt. Konflikte,
verdrängte Claims und die Urheberschaft eines Overrides bleiben in Provenienz
und Diagnose sichtbar.

### Objektive Dimensionen

Die versionierte Leaf-Taxonomie trennt mindestens:

- Wegfamilie: Strasse, Serviceweg, Radweg, Track, Pfad, Fussweg, Stufen,
  Fähre und unbekannt;
- Material: Asphalt, Beton, Pflaster, Kopfsteinpflaster, verdichtet, Feinkies,
  Kies, Erde, Gras, Sand, Schlamm, Fels/Geröll, Holz, Schnee/Eis, sonstig und
  unbekannt;
- Befestigung: befestigt, teilbefestigt, unbefestigt und unbekannt;
- `tracktype=grade1..grade5` und unbekannt;
- `smoothness=excellent|good|intermediate|bad|very_bad|horrible|very_horrible|impassable`
  als eigene geordnete Acht-Stufen-Skala sowie unbekannt;
- SAC- und MTB-Skala, Stufen, Furt, Fähre, Brücke, Tunnel,
  Zugangsrestriktion und bei ausreichender Datenqualität Exposition und
  Wegbreite.

Jede Dimension behält Quell-Rohwert und getrennten normalisierten Rang, etwa
`mtb_scale_raw=2+` neben `mtb_scale_rank`. Richtungstags wie
`mtb:scale:uphill` werden als eigene Rohclaims und normalisierte Werte
gespeichert, nicht in die allgemeine MTB-Skala eingerechnet. Sobald ein
Richtungstag ETA, Eignung, Filter oder Darstellung beeinflusst, wird die
betroffene Analyseversion richtungssensitiv; ihre Aktivierung erzeugt für die
Gegenrichtung einen eigenen Input-Fingerprint und Run, statt einen alten
richtungslosen Wert umzudeuten.

„Singletrail“ ist eine versionierte Ableitung mit Coverage und Konfidenz, kein
Alias für `highway=path`. Unbekannt unterscheidet unmatched Geometrie von
gematchten Features ohne das gesuchte Attribut.

Jeder Anteil verwendet die gesamte analysierte GPX-Distanz inklusive Unknown
als Nenner. Unabhängige Leaf-Verteilungen summieren jeweils mit ihrem Unknown
auf 100 Prozent. Überlappende Obermengen wie `paved` und `asphalt` werden nicht
in dieselbe Verteilung addiert. Negative oder Maximalfilter sind nur bei
ausreichender Coverage wahr, andernfalls unknown.

### Dataset- und Analysepersistenz

„Kartenversion“ ist kein Datum. Ein unveränderlicher `map_dataset_snapshot`
speichert Datasetart/-namespace/-revision, Providerinstanz/-version, Region und
Coverage, Beobachtungs- und Quelldatenzeitpunkt, Replikationsbasis/-sequenz,
Upstream-/Parent-Snapshot, Artefakthash/Build/Import, Builder/Konfiguration,
Qualität (`exact | opaque`), Attribution und Lizenz. Ein kleiner
`map_dataset_binding` aktiviert atomar den aktuellen Snapshot und auditiert
jede Aktivierung.

Ein exakter OSM-Snapshot braucht stabile Namespace-/Revisionsidentität und
einen belastbaren Quelldatenhorizont. Nur beobachtete Providerzeit ist `opaque`
und erfüllt keinen Filter. Graph und Tagstore bleiben getrennt referenziert,
auch wenn sie denselben OSM-Parent besitzen.

`map_match_attributes.v1` liefert mit jedem Matchergebnis eine stabile
`dataset_ref` auf genau den verwendeten Graphsnapshot.
`map_dataset_metadata.v1` beschreibt den aktuell aktiven Matcher-Graph samt
Identität und Datenhorizont ohne erneute Trailanalyse. Eine nachträgliche
Metadatenabfrage darf ein bereits erzeugtes Ergebnis nie still auf einen
anderen Datasetstand umhängen.

Valhallas `tileset_last_modified` bezeichnet ausschliesslich den
Änderungszeitpunkt des Graphartefakts. Er darf weder als OSM-Datenzeitpunkt
noch als `source_data_as_of` gespeichert oder zur Verlängerung von
`usable_until` verwendet werden. Bei kontrollierten OSM-PBF-Importen stammen
Replikationsbasis, Sequenznummer und Replikationszeitpunkt aus den dafür
vorgesehenen OSMHeader-Feldern; Importzeit, Dateizeit und Artefakt-Buildzeit
ersetzen diesen Quelldatenhorizont nicht. Fehlen diese Belege, bleibt die
Qualität `opaque`.

`trail_surface_analysis` ist ein unveränderlicher Run mit Trail,
richtungssensitivem Geometry-Hash, Analyseprofil/Input-Fingerprint,
Matcher-/Normalisierungsversion, Graph-/Tag-Snapshots, Coverage,
Berechnungs-/Verifikationszeiten, `refresh_due_at` und `usable_until`.
`trail_surface_segment` speichert Chainage, Quellsegment, optionale
OSM-Featureidentität/-version, normalisierte und rohe Dimensionen und
Confidence. Ein Run pinnt seine Zielsnapshots. Nur ein vollständiger,
validierter Run wird per kurzem Compare-and-swap aktiv; bei Fehlern bleiben
vorheriger Run und Projektion aktiv.

Routenglobale Distanz und `source_segment` bleiben auch bei wiederholten
Koordinaten, Out-and-back-Routen und GPX-Segmentgrenzen eindeutig. Ein
unveränderlicher, je Dataset-Snapshot und Way deduplizierter
`map_feature_snapshot` speichert die verwendeten OSM-Tags, Way-ID,
Objektversion, letzten Changeset und seinen Inhalts-Hash. Trailsegmente
referenzieren ihn, statt dieselben Tags zu duplizieren.

Der `input_fingerprint` umfasst mindestens kanonischen richtungssensitiven
Geometry-Hash, Analyseprofil und -version, Matcher-Plugin/-Version und
-Konfiguration, Matcher-Graph-Snapshot, OSM-Tag-Snapshot,
Normalisierungsversion sowie explizite Overrides. Der Runstatus unterscheidet
`pending`, `ready`, `partial`, `unavailable` und `failed`. Persistiert werden
mindestens analysierte, gematchte und nicht gematchte Distanz, Match-Coverage,
dimensionsspezifische Material-, Wegtyp-, Smoothness-, Tracktype-, SAC-, MTB-
und Stufen-Coverage sowie deren versionierte Distanzverteilungen. `partial`
und `unavailable` werden nie als Nullanteil oder leichte Ausprägung indexiert.

Die Suchprojektion enthält kompakte Rollups sowie aktiven Run/Fingerprint,
Analyse-/Normalisierungsversion, exakte Snapshotidentitäten/-qualitäten,
Coverage, Datenhorizonte, `computed_at` und `usable_until`. Jeder Surfacefilter
verlangt zusätzlich exakte Snapshots, ausreichende Coverage und
`surface_usable_until > as_of`. Abgelaufene Fakten werden für Surfacefilter
unknown, entfernen den Trail aber nie aus anderen Suchen. Treffer binden den
Run, damit Detailansicht und Höhenprofil dieselben Fakten wie die gepinnte
Indexgeneration lesen.

Die kompakten Rollups umfassen mindestens Anteile für Asphalt, befestigt,
unbefestigt, Gravel, natürlich, rau und Material-Unknown, Anteile für Strasse,
Track, Pfad und Singletrail, Singletrail-/Stufen-Coverage, `has_steps`,
`has_ferry` sowie maximale normalisierte SAC-, MTB- und
Smoothness-Schweregrade. Alle tragen Run-, Fingerprint-, Analyse-,
Normalisierungs- und Snapshotversion.

Produkte formulieren überprüfbare Bedingungen wie „überwiegend asphaltiert“,
„höchstens 10 % unbefestigt“, „mindestens 30 % Gravel“, „keine Stufen“,
„maximal SAC T2“, „maximal MTB S1“ oder „mindestens 90 % bekannte
Oberfläche“. `has_steps=false` bei unzureichender `steps_coverage`, ein
SAC-Maximum bei unbekannter SAC-Strecke und ein MTB-Maximum bei unzureichender
MTB-Coverage sind `unknown`, nie „keine“ oder „leicht“.

### Refresh-Lifecycle

Die DB-Queue, nicht Cron, ist das System of Record. Cron beansprucht anhand der
Datenbankzeit nur ein kurzes Scheduler-Lease und bewegt je Partition einen
persistenten, fairen, umlaufenden Keyset-Cursor. `trail_surface_state` hält
genau eine monoton steigende Sollrevision samt Fingerprint, Geometry-Hash,
Zielsnapshots, aktivem Run und optionalem Reconcile-/Fehlerzeitpunkt.

```text
surface_analysis_job
  trail_id / desired_revision / desired_input_fingerprint
  reason / priority
  target_matcher_snapshot_id / target_osm_tag_snapshot_id
  status                  queued | running | retry | superseded | dead | done
  available_at / attempts
  lease_owner / lease_token / lease_until
  last_error_code / last_error_message
```

Ein partieller Unique-Index erlaubt je Trail und Sollrevision genau einen
lebenden `queued`-/`running`-/`retry`-Job; terminale Historie verhindert keinen
späteren Refresh. Der Sweep überspringt eine bereits lebend eingereihte
Sollrevision. Nur tatsächlich neu eingefügte Jobs verbrauchen das Batchlimit,
sodass bereits eingereihte oder dauerhaft fehlgeschlagene Trails den Cursor
nicht vor dahinterliegenden Kandidaten festhalten.

Worker claimen atomar mit Lease-Token, führen Providerarbeit ausserhalb von
Transaktionen aus und erneuern bei langen Läufen ihr Lease. Die Aktivierung
vergleicht Lease-Token, aktuelle Sollrevision/Fingerprint, Zielsnapshots und
aktuellen Geometry-Hash atomar. Ein veralteter Worker kann nur `superseded`
werden. Abgelaufene Leases werden zurückgeholt; temporäre Fehler nutzen
begrenztes exponentielles Backoff mit Jitter und beachten `Retry-After`;
Dead-Jobs benötigen einen persistenten Cooldown oder bewussten Retry. Zeit-,
Punkt-, Meter-, Provider- und Parallelitätsbudgets begrenzen die Last.

Refresh wird fällig bei fehlender Analyse, geänderter Geometrie/Override,
neuer Analyse-/Profil-/Normalisierungsversion, Admininvalidierung oder
abgelaufener Policy gegenüber einem neueren aktiven Kartensnapshot. Eine neue
globale OSM-Revision reiht nicht sofort den ganzen Katalog ein. Optionale
Replikationsdiffs und Way-/Korridor-Reverse-Indizes priorisieren Betroffene;
der Alterssweep bleibt Reconciliation und Safety-Net.

`refresh_due_at` ist die weiche Refreshschwelle, `usable_until` der harte
Behauptungshorizont. Dieser wird sowohl vom Matcher-Graph- als auch vom
OSM-Tag-Datenhorizont begrenzt. Eine heutige Neuberechnung auf derselben zwei
Jahre alten Karte verlängert ihn nicht. Nur eine verifiziert lückenlose
Deltakette darf einen Datenhorizont anheben; `last_verified_at` allein nie.
Ein abgelaufenes Dataset erzeugt keinen neuen `ready`-Run, ein `opaque`-
Snapshot nur eine kürzere Diagnosefrist.

Eine neue Refreshpolicy berechnet Due-/Usable-Zeiten ohne unnötiges
Rematching neu. Das Batchlimit ist ein Lastbudget, keine Freshness-Garantie.
Schedule und Workerdurchsatz müssen für den grössten unterstützten Katalog
Neuzugang, Invalidierungen, Fehler und den vollständigen TTL-Sweep bewältigen.
Diagnosen zeigen aktive Snapshots, Lag, Queue-Tiefe, ältesten fälligen Job,
Durchsatz und terminale Fehler.

Nach `usable_until` bleibt der letzte Run auf der Detailseite mit seinem
Berechnungs- und Datenstandsdatum sichtbar. Er erfüllt Surface- und
Technikfilter jedoch nur noch als `unknown`; alle anderen Bestandsfilter
behalten den Trail unverändert im Suchuniversum.

Bei Surface-Filter, -Sortierung oder -Counts endet
`search_context.valid_until` spätestens an der frühesten für die vollständige
Ergebnis- und Aggregationsmenge relevanten `surface_usable_until`-Grenze. Kann
dieses Minimum nicht exakt bestimmt werden, gilt ein konservativer kürzerer
Kontext-TTL. Eine Fortsetzung nach der Grenze ergibt
`search_context_expired`, nicht Resultate mit einem neueren oder veralteten
Surface-Stand.

Aktive Runs sowie die von einer gepinnten Indexgeneration oder einem lebenden
Suchkontext referenzierten Runs, Feature- und Dataset-Snapshots bleiben bis zum
Ende aller Referenzen erhalten. Dasselbe gilt für rollbackfähige inaktive Runs
und für Zielsnapshots lebender `queued`-/`running`-/`retry`-Jobs. GC darf diese
Artefakte erst nach belegter Referenzfreiheit entfernen.

## Capability-spezifische Abnahme

Die folgenden Gates ergänzen die allgemeinen Tests, ohne deren Wire- oder
State-Semantik zu duplizieren:

- Der G0-Smoke-Contract prüft Linienmitte, zweiten MultiLine-Teil,
  MultiLine-Gap, Koordinatenreihenfolge `[lon, lat]`, Viewport-Crossing,
  disjunkte Linien trotz überlappender Bounding Box, Boundary-Touch und einen
  explizit am Antimeridian gesplitteten Fall. Hinzu kommen Auflösungs- und
  Near-Boundary-Regressionen. Ein Treffer in der Routenmitte wird gefunden;
  getrennte GPX-Segmente werden nie durch eine fiktive Linie verbunden.
- Der Geocoding-Contract-Test läuft mindestens gegen GEO-PHOTON und einen
  providerneutralen Fake-Adapter. Er prüft dieselben positiven und negativen
  Scope-Fixtures, Capability-Metadaten und Fehlerfälle; kein Produktpfad darf
  einen Sonderfall `provider = photon` benötigen.
- Direct-Reports weisen finale Responsegrösse, Boundary-Misses,
  Countabweichung, Page-/Top-10-/Nearest-Metriken und Warm-Stabilität aus.
  Zweistufige Diagnosepfade weisen Kandidaten, geladene DB-Zeilen/-Bytes und
  exakte Checks jeweils mindestens als p95 und Maximum aus.
- Ressourcen-Caps werden bei N−1/N/N+1 geprüft. Kein zweistufiger Pfad setzt
  `maxTotalHits` auf Kataloggrösse oder hält unbeschränkt die Vollmenge im RAM;
  Streaming/Spill und Capfehler liefern niemals partielle Treffer oder Counts
  als exakt.
- Anstiegs-Fixtures belegen Same-Climb-Semantik, Richtungsumkehr,
  Segmentgrenzen und Unknown bei fehlender Höhe.
- Zugangs-Fixtures unterscheiden Start/Ziel, Rundweg/Streckentour,
  Ausstiegsoptionen entlang der Route, Modus, Zugangsart, physische
  Infrastruktur, statische Bedienung und konkrete Verbindung.
- Surface-Fixtures decken alpine SAC-Wege, MTB, Parallelwege, Out-and-back und
  mehrere GPX-Segmente ab. Jeder Run partitioniert die A1-Route exakt einmal;
  Overlap, Gaps und getrennte Graph-/Tag-Snapshots bleiben sichtbar.
- Filterbare Runs referenzieren zwei `exact`-Snapshots und bestehen die
  vereinbarte Match- und Attribut-Coverage. `opaque`, abgelaufen und
  dimensionsspezifisch unzureichend ergeben reproduzierbar `unknown`.
- Tests für überlappende Scheduler, Prozessabbruch, Lease-Ablauf, Retry,
  permanente Fehler und veraltete Sollrevision beweisen: kein verlorener Job,
  kein doppelter aktiver Run und kein Rollback durch einen alten Worker.
- Der persistente Partitionssweep erreicht Kandidaten hinter bereits
  eingereihten oder permanent fehlgeschlagenen Trails; nur erfolgreiche
  Inserts verbrauchen das Batchbudget.
- Der gemessene Refreshdurchsatz hält im grössten unterstützten Katalog den
  ältesten fälligen Run vor `usable_until`. Reicht der TTL-Sweep nicht, wird
  Delta-Invalidierung vor Freigabe verpflichtend.
- Expiry-/Retentiontests belegen Unknown nach `usable_until`, das Ende des
  Suchkontexts vor der ersten relevanten Expiry sowie den Schutz aller aktiven,
  rollbackfähigen, kontext- und jobreferenzierten Artefakte.
- Wetter-/Fahrplanausfall ergibt unbekannten Kontext; statische Werte werden
  nie als dynamische Ergebnisse umetikettiert. Providerzugriffe erfolgen
  weder pro Trail noch pro Suchkandidat im Requestpfad.
