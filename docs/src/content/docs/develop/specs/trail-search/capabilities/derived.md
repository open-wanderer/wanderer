---
title: DERIVED — Lokale Analyse und Personalisierung
description: Produktanforderungen für versionierte Routenfakten, Ausflugsprofile, persönliche Dauer und Eignung sowie lokale Bewertungen
editUrl: false
sidebar:
  order: 2
  badge: Entwurf
spec:
  id: CAP-DERIVED
  kind: capability
  status: draft
  capability: DERIVED
  lastReviewed: '2026-08-30'
---

## Rolle und Normativität

Diese Seite ist die fachliche Capability-Spezifikation für Fakten und
Auswertungen, die Wanderer aus lokal verfügbaren Trail-, GPX- und Nutzerdaten
ableiten kann. Sie ist ein **normativer Entwurf für Bedeutung und Grenzen der
Capability DERIVED**. Die nachfolgend gezeigten logischen Datensätze beschreiben
die benötigten Informationen, schreiben aber kein physisches PocketBase-Schema
vor, sofern ein referenzierter Vertrag oder Entscheid nichts anderes festlegt.

Querschnittliche Regeln für unbekannte Werte, Provenienz, Versionierung,
Persistenz, Backfills, Aktivierung, Rollback, Kompatibilität und Sicherheit sind
ausschliesslich in den [gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/)
festgelegt. Exakte Requestfelder, Operatoren, Defaults, Counts, Sortierung,
Pagination und URL-Kodierung definiert ausschliesslich der
[Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/). Mögliche Bausteinschnitte
und ihre Abhängigkeiten werden ausschliesslich unter
[Delivery und Beiträge](/develop/specs/trail-search/delivery/) geführt.

## Capability-Grenze

DERIVED verwendet lokale Trail-, GPX- und Nutzerdaten. Die Capability darf
versionierte abgeleitete Datensätze, wiederverwendbare Vorlagen für
Ausflugsprofile und fortsetzbare Analysearbeit ergänzen. Sie führt kein
Route-zu-OSM-Matching durch und ruft im Suchrequest keine zeitabhängigen
Provider auf. Anstiegs-, Wegbeschaffenheits-, Wetter-, Saison- und
ÖV-Evaluatoren können ihre Modelle später erweitern; ihr Fehlen darf aber nicht
durch erfundene Defaultwerte kaschiert werden.

Die zentrale fachliche Trennung lautet:

- Routenfakten beschreiben die Route;
- Teilnehmer-Templates beschreiben Fähigkeiten und Präferenzen;
- ein Ausflugsprofil beschreibt, für wen und mit welcher Ausrüstung aktuell
  geplant wird;
- zeitabhängige Bedingungen gehören in die Kontextauswertung und nicht in den
  gespeicherten Trail.

## Kanonisches Aktivitätsmodell

Referenzdauer und Teilnehmerprofile benötigen stabile interne
Aktivitätsschlüssel. Frei konfigurierbare lokale Kategorien und heterogene
föderierte Taxonomien können nicht direkt als Modellschlüssel dienen. Familie,
Disziplin und Unterstützung sind deshalb getrennte Achsen.

Gemäss
[ADR 0003](/develop/specs/trail-search/decisions/0003-walk-hike-cycle-personal-evaluation/)
unterstützt die erste produktive Referenzdauer-, ETA- und Konditionsbewertung
die Familien `walk`, `hike` und `cycle`. Für `cycle` gilt zusätzlich:

```text
activity_discipline = touring | road | gravel | mtb | unknown
```

Elektrische Unterstützung ist weder Familie noch Disziplin. Ein E-Bike-
Quellwert wird als `cycle`, eine soweit eindeutig bestimmbare Disziplin und ein
separater Unterstützungsclaim erhalten. Die beim konkreten Ausflug tatsächlich
verwendete Unterstützung gehört in Teilnehmer- und Ausrüstungskontext.

Walk und Hike werden nur durch explizite versionierte Quell-, Kategorie- oder
manuelle Regeln unterschieden, nie heuristisch aus Distanz, Höhenmetern oder
Surface. Familie und Disziplin besitzen getrennte Auflösungszustände;
`configured` ist eine Mappingquelle und kein Auflösungszustand:

```text
family_mapping.presence     = present | absent
discipline_mapping.presence = present | absent | not_applicable
family_mapping.status / discipline_mapping.status =
  exact | ambiguous | unmapped                 # nur bei presence = present
family_mapping.source / discipline_mapping.source =
  source_claim | built_in_rule | instance_config | manual
family_mapping.version / discipline_mapping.version = <immutable version>
model_support_status      = supported | unsupported
```

- Instanzadministratoren dürfen benutzerdefinierte lokale Kategorien zuordnen.
- Bekannte föderierte Taxonomien dürfen über eine explizite versionierte Regel
  zugeordnet werden.
- Rohwert, Mappingrevisionen, Presence, Auflösungszustände und Mappingquellen
  bleiben nachvollziehbar.
- Eine allgemeine Angabe wie „Biking“ ergibt exakt `cycle`, aber einen
  `absent`-Disziplinclaim und damit eine unbekannte Disziplin; sie wird nie
  still als `touring` interpretiert.
- Eine eindeutig bekannte, aber noch nicht produktiv bewertete Familie bleibt
  erhalten und setzt `model_support_status = unsupported`; `unknown` bezeichnet
  dagegen eine nicht bestimmbare Familie.
- Interne Modellachsen bleiben unabhängig von Kategorie und Subkategorie, die
  dem Nutzer angezeigt und von ihm ausgewählt werden.

## Gemeinsame Routenanalyse

Upload, manuelle Planung, Plugin-Import, föderierter Ingest und Backfill dürfen
keine voneinander abweichenden Formeln für denselben abgeleiteten Fakt
etablieren. Sie verwenden einen gemeinsamen serverseitigen Analysepfad und eine
gemeinsame richtungssensitive Routen-/Chainage-Basis.

Ein logischer `trail_analysis`-Datensatz oder ein gleichwertiges versioniertes
Modell enthält mindestens:

```text
trail_id
geometry_hash
analysis_version
analysis_status          pending | ready | partial | failed
analysis_error
source_activity_claim
activity_family
family_mapping             presence / status? / source / version
activity_discipline
discipline_mapping         presence / status? / source / version
source_assistance_claim
model_support_status
route_chainage_version
start_lat/lon
finish_lat/lon
route_shape              loop | point_to_point | unknown
min_elevation_m
max_elevation_m
reference_duration_s
reference_duration_status
reference_duration_components
reference_model_version
reference_duration_quality
elevation_source
elevation_coverage
updated_at
```

Echte GPX-`trk`- und -`rte`-Segmente bleiben getrennt. Jeder Punkt oder Abschnitt
der gemeinsamen Chainage-Repräsentation behält sein Quellsegment, seine lokale
Distanz, seine routenglobale Distanz, seine Richtung und den Geometrie-Hash.
Suchgeometrie, Höhen- und Anstiegsanalyse sowie späteres Map-Matching leiten
ihre eigenen Repräsentationen aus dieser gemeinsamen Basis ab, statt leicht
abweichende Segmentierungen zu erzeugen.

Eine geänderte GPX-Geometrie oder Analyseversion invalidiert die zugehörige
Analyse. Nur kompakte filterbare Skalare werden in den Suchindex projiziert;
GPX-Rohdaten und detaillierte geordnete Segmente bleiben ausserhalb des
Suchdokuments. Den allgemeinen Lifecycle und die atomaren Aktivierungsregeln
definieren die [gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/). Für diese
Capability gilt zusätzlich: Ein aktiver Suchkontext darf Fakten oder
Modellkomponenten verschiedener Analyseversionen nie still mischen.

## Routenform und Höhenextrema

Die Routenform wird aus dem echten ersten und letzten Punkt der geordneten Route
abgeleitet. Die Rundwegschwelle ist versioniert und wird mit realen Daten
kalibriert. Sie darf eine absolute Obergrenze mit einer zur Routendistanz
relativen Komponente kombinieren.

Ein begründetes manuelles Override ist zulässig, weil eine Aufzeichnung kleine
Transfers enthalten kann. Abgeleiteter Wert und Override bleiben getrennt und
nachvollziehbar.

- Mehrere GPX-Segmente bleiben getrennt; zwischen dem Ende eines Segments und
  dem Anfang des nächsten darf keine fiktive Kante entstehen.
- GPX-`rte`-Punkte werden ebenso berücksichtigt wie `trk`-Punkte.
- Die maximale Höhe ist das Maximum der validen Höhensamples.
- Fehlende oder unzureichend abgedeckte Höhendaten ergeben einen unbekannten
  Wert.
- Eine Route auf Meereshöhe und eine Route ohne Höhendaten sind verschiedene
  Fälle.
- Valide negative Höhen sind weder ungültig noch unbekannt.

Die exakte öffentliche Range- und Missing-Repräsentation gehört in den
[Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/).

## Dauermodell

Ein einzelner untypisierter Wert `duration` reicht für Suche und
Personalisierung nicht aus. Rohbeobachtungen unterscheiden:

- `observed_moving`: gemessene Zeit in Bewegung;
- `observed_elapsed`: gesamte verstrichene Zeit;
- `provider`: vom Quellsystem gelieferte Planzeit;
- `routing_engine`: von einer Routing-Engine gelieferte Dauer;
- `estimated`: das neutrale Wanderer-Referenzmodell;
- `unknown`.

Diese Werte bleiben Beobachtungen. Sie werden nicht opportunistisch als
`reference_duration_s` eines Trails übernommen, weil sonst die Leistung des
Erstellers oder eine unbekannte Providerannahme als Routeneigenschaft
gespeichert würde.

`reference_duration_s` wird immer durch ein neutrales, aktivitätsspezifisches
und versioniertes Wanderer-Modell aus statischen Routenmerkmalen berechnet.
Qualifizierte beobachtete Dauern dürfen eine spätere Modellversion kalibrieren,
aber eine Beobachtung wird nie als Referenzdauer desselben Trails zurückkopiert.
Eine manuell gezeichnete Route ohne Zeitstempel wird bei ausreichenden
Routenfakten geschätzt; andernfalls bleibt ihre Referenzdauer unbekannt, statt
null zu werden.

Das Referenzmodell ist eine Registry unabhängig versionierter Komponenten. Das
vollständige Basismodell weist die je Familie anwendbaren distanzbezogenen,
aufwärts- und gegebenenfalls abwärtsbezogenen Komponenten sowie Anspruch,
Konfidenz und Komponenten-Fingerprint aus. Spätere Komponenten für
Anstiegsverteilung und Wegbeschaffenheit dürfen sich nach ihren eigenen Daten-
und Qualitätsgates unabhängig registrieren. Für fehlende oder nicht anwendbare
Komponenten entstehen keine synthetischen Defaults. Teilnehmerleistung,
Gruppeneffekte, Wetter und Pausen sind nie Teil der stabilen Referenzdauer.

Der erste produktive Modellumfang enthält Walk, Hike und Cycle gemeinsam. Walk
und Hike verwenden je ein versioniertes flach-, aufwärts- und
abwärtsbezogenes Komponentenmodell; die erste Modellversion darf dafür
denselben Parametersatz referenzieren, ohne beide Familienwerte
zusammenzulegen. Cycle verwendet je Disziplin eine Distanz- und eine
Aufstiegskomponente mit eigenem Kombinator:

```text
distance_component_s = distance_m / distance_rate_m_per_s
ascent_component_s   = elevation_gain_m / ascent_rate_m_per_s

reference_duration_s =
  round_half_up_seconds(
    max(distance_component_s, ascent_component_s)
    + 0.5 * min(distance_component_s, ascent_component_s)
  )
```

Beide Raten sind endlich und strikt positiv. Faktor `0.5`, Kombinator und die
Rundung des finalen Werts auf die nächste ganze Sekunde, wobei Halbwerte
aufgerundet werden, sind stabil; Distanz- und Steigraten werden empirisch
kalibriert. Die Cycle-Distanzkomponente umfasst bereits die gesamte Route. Eine
zusätzliche Abstiegskomponente ist im Cycle-Basismodell
`not_applicable`, nicht null; spätere Anstiegs- oder Surface-Korrekturen bleiben
getrennt versioniert. Eine Cycle-Referenzdauer setzt bekannte Distanz,
bekannten Gesamtaufstieg und eine unterstützte Disziplin voraus. Bekannte null
Höhenmeter sind valide. Eine unbekannte Disziplin darf höchstens eine
ausdrücklich grobe Spanne liefern, aber keine einzelne Referenzdauer für
Filter, Sortierung, Counts oder Eignungsklassen.

Trail und aktives Ausflugsprofil müssen für eine produktive Bewertung denselben
Familien- und Disziplinschlüssel besitzen. Ein Mismatch ergibt
`not_applicable`; ein Gravel-Profil konsumiert insbesondere nie still eine
Road-Referenzdauer. Eine spätere explizite Kompatibilitätsmatrix darf diese
Regel versioniert erweitern.

## Teilnehmer-Templates und Ausflugsprofile

Ein implizites Profil des angemeldeten Benutzers genügt nicht. Das Modell trennt
wiederverwendbare Teilnehmer-Templates von dem für eine konkrete Suche
gewählten Ausflugsprofil.

Teilnehmer-Templates existieren in drei Scopes:

- **System:** mit Wanderer ausgelieferte, versionierte Ausgangspunkte;
- **Instanz:** von einem Administrator an Region oder Zielgruppe angepasste
  Vorlagen;
- **Benutzer:** private Kopien, Overrides und gelernte Profile.

Systemtemplates können einen neutralen Erwachsenen, einen Wander-Einstieg, einen
konservativen Ausgangspunkt für ein Kind oder disziplinspezifische Radprofile
enthalten. Systemweite Ausflugsvorlagen können diese als „Allein“, „Familie“,
„Mit Kinderwagen“, „Mit Anhänger“ oder mit elektrischer Unterstützung
kombinieren. Alter allein wird nie als Fähigkeit behandelt. Vorlagen sind
transparente Annahmen, die Benutzer klonen, benennen und mit realen Fähigkeiten
präzisieren können.

Ein logisches Teilnehmer-Template enthält:

```text
performance_profile_template
  id / scope / owner
  base_template_id?
  name / activity_family / activity_discipline?
  pace_factors             distance / ascent / descent?
  comfortable_duration_s / comfortable_gain_m
  climb_capacity / grade_capacity
  technical_capacities
  environmental_limits
  allowed_surface_and_way_constraints
  equipment_defaults
  rest_model
  source                  system | instance | manual | learned
  model_version / base_version / override_diff
  sample_count / confidence
```

Die `pace_factors` sind dimensionslos. `1.0` entspricht der jeweiligen
Komponente des neutralen Referenzmodells; ein Wert grösser als `1.0` bedeutet
schneller. Cycle benötigt mindestens getrennte Distanz- und aufwärts gerichtete
Faktoren, weil dieselbe Person nicht zwingend in beiden Situationen
relativ gleich schnell ist. Walk/Hike dürfen zusätzlich einen Abwärtsfaktor
verwenden. Zulässige Grenzen gehören zur Modellversion. Ein fehlender Wert
fällt sichtbar auf ein neutrales Profil mit niedriger Konfidenz zurück, während
null und negative Werte ungültig sind. Körperliche Kapazität und technische
Fähigkeit bleiben getrennte Achsen.

Ein Ausflugsprofil beschreibt den konkreten Planungskontext und referenziert ein
oder mehrere Teilnehmer-Templates:

```text
outing_profile
  id / scope / owner / name         # „Allein“, „Mit Mia“, „Familie“
  base_template_id? / override_diff
  activity_family / activity_discipline?
  participants[]                    # Template, Overrides, Ausrüstungszuordnung
  group_size
  equipment[]                       # typisiert und Teilnehmern zugeordnet
    kind / assistance_mode          # none | electric_assist | unknown
  luggage
  group_pace_factor / rest_overhead
  hard_constraints
  version
```

Ein Kind benötigt kein Benutzerkonto; Alias und Parameter bleiben privat.
Benutzer dürfen globale Vorlagen erweitern, während ihre Overrides als Diff
gespeichert werden. Ändert sich eine Basisvorlage, zeigt Wanderer eine
Migrationsvorschau, statt lokale Änderungen still zu überschreiben. Vor einer
sichtbaren Freigabe müssen das aktive Ausflugsprofil, seine unveränderliche
Revision sowie Request-, URL- und Responsebindung ausdrücklich im
[Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/)
ergänzt werden. Bis dahin behauptet weder URL noch dialogbasierte Suche, diesen
Zustand bereits reproduzierbar zu transportieren.

Ein Kinder-Template darf konservativere Wind-, Hitze-, Kälte- und
Tageslichtgrenzen vorschlagen. Benutzer können jede Grenze nachvollziehbar als
versionierten Diff überschreiben. Solche Werte sind transparente
Planungsannahmen; sie behaupten weder medizinische Eignung noch objektive
Sicherheit und ersetzen keine situationsbezogene Beurteilung.

Aus Summit Logs gelernte Werte aktualisieren ein Benutzer-Template, nie ein
Systemtemplate und nie einen Trail. Ein robuster Schätzer filtert nach
Aktivitätsfamilie, Disziplin, Datenqualität und Kontext. Der Median von
`reference_duration_s / observed_moving_duration_s` darf als Gesamtdiagnose
dienen, identifiziert aber keine getrennten Distanz- und Aufstiegsfaktoren.
Diese benötigen ein nach Disziplin und Routenkomponenten stratifiziertes,
identifizierbares Modell. Elektrisch unterstützte Beobachtungen kalibrieren nie
die neutrale menschliche Referenz oder unassistierte Teilnehmerfaktoren,
sondern höchstens einen getrennten Unterstützungseffekt. Kleine, alte oder
atypische Stichproben bleiben sichtbar niedrig in der Konfidenz und ersetzen
nie manuell gesetzte harte Grenzen.

Falls Activity Metrics später Herzfrequenz, Leistung oder Kadenz kanonisch
speichern, darf eine getrennte, einwilligungsbasierte Erweiterung diese Signale
nach Qualitätsprüfung zur Kalibrierung verwenden. Solche Metriken sind keine
Voraussetzung für System- oder Benutzer-Templates: Beobachtete Dauer und
manuelle Overrides bilden bereits eine ehrliche, vollständige Basis.

## Erwartete Dauer für den gewählten Ausflug

Für jeden Teilnehmer wird je anwendbarer Referenzkomponente eine individuelle
Bewegungszeit geschätzt. In einer Gruppe bestimmt der langsamste relevante
Teilnehmer jede Komponente und nicht der Durchschnitt. Der aktive
Familienkombinator bildet daraus die Bewegungszeit; Gruppen- und Pauseneffekte
werden danach angewandt:

```text
participant_component_axis_i =
  reference_component_axis / pace_factor_axis_i
group_component_axis = max(participant_component_axis_i)
movement_time = family_combiner(group_components)
expected_group_duration = movement_time * group_pace_factor
                        + rest_overhead(route, participants, equipment)
```

Für Cycle nimmt der Kombinator das Maximum aus Distanz- und
Aufstiegskomponente plus die Hälfte der kleineren Komponente. Dadurch dürfen
über die Distanz und im Aufstieg unterschiedliche Personen die Gruppe
limitieren.
Für eine einzelne Person wird derselbe Kombinator auf deren Komponenten
angewandt. Ein skalares `reference_duration_s / pace_factor` ist nicht für alle
Familien gleichwertig.

Eine elektrische Unterstützung verändert die effektiven Teilnehmerkomponenten
über genau einen versionierten Ausrüstungseffekt; der E-Bike-Quellclaim eines
Trails gewährt keinen zweiten Bonus. Unterstützung wird je Teilnehmer und
zugeordnetem Fahrrad ausgewertet; eine Gruppe mit unterstützten und nicht
unterstützten Rädern erhält keinen pauschalen E-Bike-Faktor. Statische monotone
Profilkomponenten dürfen in gleichwertige indexierte Schwellen je
Aktivitätsfamilie und Disziplin übersetzt werden. Gemischte Familien verwenden
für jeden Zweig die korrekte familienspezifische Schwelle; Vergleiche verwenden
erwartete Sekunden.
Gruppenpausen, Ausrüstung und aktivierte Kontext-Evaluatoren werden vom
Search-Orchestrator über die vollständige Kandidatenmenge ausgewertet. Es
entstehen weder benutzerspezifische Suchdokumente noch ein Profil-Reindex.

Ein reiner, versionierter Teilnehmerdauer-Evaluator liefert die individuelle
Dauer für Dauerfilter, Gruppen-ETA und persönliche Eignung; diese Verbraucher
dürfen keine konkurrierenden Zeitformeln implementieren. Wird eine neue
statische Referenzmodellkomponente aktiviert, ändern sich ihre Fakten und ihre
Projektion nach den gemeinsamen Versionierungsregeln, während der Evaluator
weiterhin genau das aktive Komponenten-Set samt `reference_duration_s`
konsumiert.

Das normative Verhalten von Filterung, Sortierung, Counts und Pagination legt
ausschliesslich der [Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/) fest.

## Eignung für den gewählten Ausflug

Die persönliche Eignung verwendet nicht die gespeicherte oder importierte
Quellschwierigkeit des Trails. Das bestehende Konzept der angegebenen
Schwierigkeit bleibt getrennt. Die neue Auswertung vergleicht einen objektiven
Routen-Demand-Vektor mit jedem Teilnehmer und der Ausrüstung des aktiven
Ausflugsprofils.

Der vollständige DERIVED-Basisvektor lautet:

```text
reference_duration_s
elevation_gain_m
```

Diese körperliche Basis wird im ersten produktiven Slice für Walk, Hike und
Cycle ausgewertet. Bei Cycle wählt die Disziplin das Referenzmodell, ist aber
kein Beweis für technische Fahrradkompatibilität.

Kontext-Capabilities dürfen später isolierte Anstiegsachsen wie
`hardest_climb_score` und `max_sustained_grade_100m_pct` sowie unabhängige
Surface- oder Technikachsen wie Oberflächenanteile, `smoothness`, `tracktype`,
Stufen, SAC-/MTB-Skala und bekannte Exposition registrieren. Jede Erweiterung
deklariert Namen, Einheit, anwendbare Aktivitätsfamilien, Datenabdeckung,
Teilnehmer-Capability und Erklärungslogik. Fehlende Erweiterungen blockieren die
DERIVED-Basis nicht und erzeugen keine Scheinwerte. Wetter und Saison dürfen die
registrierte Nachfrage für einen gewählten Zeitpunkt verändern, überschreiben
aber nie objektive Trail-Fakten.

Distanz und höchste Höhe bleiben unabhängige Routenfakten. Die höchste Höhe wird
erst dann zu einer persönlichen Belastungsachse, wenn ein fachlich belastbares
Höhenverträglichkeits- oder Akklimatisierungsmodell existiert.

Eine erklärbare körperliche Klassifikation verwendet den grössten registrierten
Belastungsquotienten:

```text
participant_base_ratio_i = max(
  participant_duration_i / comfortable_duration_s_i,
  elevation_gain_m / comfortable_gain_m_i
)

participant_physical_ratio_i = max(
  participant_base_ratio_i,
  registered_extension_ratios_i...
)
group_physical_ratio = max(participant_physical_ratio_i)
technical_allowed = all(registered_participant_and_equipment_constraints)
```

Referenzdauer und Gesamtaufstieg werden nicht addiert; das Maximum bezeichnet
den dominierenden Engpass des am stärksten eingeschränkten Teilnehmers.
Technische Eignung ist eine AND-Bedingung: Eine für einen Erwachsenen geeignete
Passage ist dadurch nicht automatisch für ein Kind, einen Kinderwagen oder
einen Anhänger geeignet. Ein zusätzlicher `endurance_load` ist nur zulässig,
wenn seine Einheit und seine Abgrenzung zur Referenzdauer unabhängig validiert
wurden.

Bei statischen Klassen bedeutet „leicht“, dass alle erforderlichen Grenzen für
alle Teilnehmer eingehalten werden. „Schwer“ bedeutet, dass mindestens eine
anwendbare harte Grenze überschritten wird; „mittel“ ist der versionierte
Bereich dazwischen. Gruppen- und Kontextbedingungen müssen über dieselbe
vollständige Kandidatenmenge wie die Treffer ausgewertet werden. Die exakte
Kompilierung der Suche bleibt Aufgabe des
[Trail-Suchvertrags V1](/develop/specs/trail-search/contracts/trail-search-v1/).

Schwellen und Achsen sind aktivitätsspezifisch und versioniert. Eine unbekannte
Achse wird nie als leicht oder kompatibel interpretiert. Statt eines
undurchsichtigen Gesamtscores trennt das Ergebnis körperliche Klasse,
technische Kompatibilität, dominante Ursache, Datenabdeckung und Konfidenz.

Eine persönliche Aussage über isolierte Anstiege benötigt eine freigegebene
Anstiegs-Profilintegration; technische Fahrradkompatibilität für Road, Gravel
oder MTB benötigt eine freigegebene Surface-/Technikintegration. Fehlt die
jeweils erforderliche Integration, bleibt diese Achse `unknown`. Anstiegs- oder
Surface-Erweiterungen blockieren die Cycle-Basis aus Referenzdauer, ETA,
Gesamtaufstieg und körperlicher Kondition nicht.

Das UI hält mindestens diese Begriffe getrennt:

- **Kondition für uns:** leicht, mittel oder schwer, mit betroffenem Teilnehmer,
  Begründung und Konfidenz;
- **Technische Eignung:** objektiver Routendemand gegen Fähigkeiten von
  Teilnehmern und Ausrüstung; unbekannt, wenn die benötigten Fakten fehlen;
- **Angegebene Schwierigkeit:** der bestehende Quellwert, der nie als Grundlage
  der persönlichen Eignung ausgegeben wird.

Frühere Arbeit in [PR 594](https://github.com/open-wanderer/wanderer/pull/594)
enthält nützliche Ideen zu kategorienspezifischen Grenzen und Benutzerprofilen,
ist aber Hintergrund und kein unverändert wiederaufzunehmender Patch. Ein
Nachfolger muss Einheiten, flache Routen, Update- und Backfill-Verhalten,
Kategoriezuordnung, Defaults und Tests neu lösen. Insbesondere dürfen die
Leistungsgrenzen eines Erstellers nie global auf einem Trail persistiert werden.

## Bewertungen

Bewertungen bilden einen unabhängigen DERIVED-Produktbereich. Das Modell
benötigt:

- genau eine aktuelle Bewertung pro Benutzer und Trail;
- einen begrenzten Wert, beispielsweise von eins bis fünf;
- die Aggregate `rating_count`, `rating_sum` und `rating_average`;
- einen optionalen getrennten Review-Text;
- eine gewichtete oder bayessche Rangfolge statt eines unqualifizierten
  arithmetischen Mittels.

Ein Mindestbewertungsfilter trifft nur Trails mit mindestens einer Bewertung
und zeigt immer die Anzahl Bewertungen. Likes bleiben ein anderes Konzept.
Bewertungen dieser Spezifikation sind instanzlokal und zählen nur lokal
akzeptierte Stimmen. Sie werden nicht über ActivityPub föderiert: Ohne
gemeinsamen Identitäts-, Trust-, Moderations- und Deduplizierungsvertrag wäre
ein globaler Mittelwert irreführend. Föderierte Bewertungen benötigen eine
eigene Protokollspezifikation und dürfen diese Felder nicht still erweitern.

Die exakte öffentliche Repräsentation für Filter, Counts und Rangfolge legt der
[Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/) fest.

## Capability-spezifische Freigabegates

Routenfakten, Referenzdauer, Profile, persönliche Kondition und Bewertungen
dürfen unabhängig erscheinen, sobald ihre eigenen Abhängigkeiten unter
[Delivery und Beiträge](/develop/specs/trail-search/delivery/) sowie die folgenden Gates erfüllt sind:

- Die Routenanalyse ist reproduzierbar und wird bei fehlerhaften GPX-Eingaben
  fortgesetzt; ihre capability-spezifische Invalidierung und ihr
  Coverage-Verhalten sind getestet.
- Routenform, Endpunkt und höchste Höhe funktionieren für GPX-Tracks, -Routen
  und mehrere Segmente ohne erfundene Verbindungen.
- Dauer null wird nie als bekannte Dauer interpretiert.
- Walk-, Hike-, Touring-, Road-, Gravel- und MTB-Fixtures sowie Cycle mit
  unbekannter Disziplin prüfen Modellwahl, Komponenten, Qualität und
  `unsupported`/`unknown` ohne stilles Fallback; sie übernehmen keine
  unqualifizierten Kalibrierseeds als Produktionsdefaults.
- Einzel-, Kinder- und Gruppenprofile erzeugen ohne Trail-Reindex
  unterschiedliche Dauer- und Konditionsergebnisse.
- Gruppenfixtures prüfen getrennt limitierende Personen auf Distanz und
  Aufstieg;
  E-Bike-Quellclaim und tatsächliche Unterstützung dürfen nie doppelt wirken.
- System-, Instanz- und Benutzertemplates sind versioniert; Benutzer-Overrides
  überstehen eine kontrollierte Aktualisierung der Basisvorlage.
- Jede Einzel- oder Gruppeneinstufung nennt den betroffenen Teilnehmer, die
  dominante Ursache, Datenabdeckung und Konfidenz.
- Controls für technische Eignung erscheinen nur, wenn die entsprechende
  Kontextanalyse registriert und bereit ist; fehlende technische Fakten gelten
  nie als leicht.
- Kinder-Aliasse, Profilparameter, gelernte Stichproben und Activity Metrics
  halten die dokumentierten Datenschutz- und Einwilligungsgrenzen ein.
- Bewertungen besitzen von Likes getrennte Modelle, Filter und Sortierungen und
  können unabhängig von Routenanalysejobs aktiviert werden.

Allgemeine Aktivierungs-, Rollback-, Kompatibilitäts-, Sicherheits- und
Beobachtbarkeitsgates werden hier nicht wiederholt; sie stehen in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/).

## Quellen

- [ADR 0003: Walk, Hike und Cycle in der persönlichen Bewertung](/develop/specs/trail-search/decisions/0003-walk-hike-cycle-personal-evaluation/)
- [Trail-Suchvertrag V1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [Gemeinsame Invarianten der Trail-Suche](/develop/specs/trail-search/shared-invariants/)
- [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
- [Historische Profil- und Schwierigkeitsarbeit, PR 594](https://github.com/open-wanderer/wanderer/pull/594)
