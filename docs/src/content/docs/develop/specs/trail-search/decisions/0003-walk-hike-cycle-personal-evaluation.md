---
title: "ADR 0003: Walk, Hike und Cycle in der persönlichen Bewertung"
description: Angenommene Produktentscheidung zum Aktivitätsmodell und zum ersten produktiven Umfang von Referenzdauer, ETA und Eignung.
editUrl: false
sidebar:
  order: 3
  badge: Angenommen
spec:
  id: ADR-0003
  kind: decision
  status: accepted
  capability: DERIVED
  lastReviewed: '2026-08-30'
---

- Status: angenommen
- Datum: 30. August 2026
- Produktumfang: erste produktive Referenzdauer-, ETA- und
  Konditionsbewertung
- Delivery-Einordnung: konkrete Bausteine und Abhängigkeiten stehen
  ausschliesslich im
  [Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/)
- Fachliche Einordnung:
  [DERIVED](/develop/specs/trail-search/capabilities/derived/)

## Kontext

Die persönliche Bewertung benötigt einen stabilen Aktivitätsschlüssel. Ein
flaches Enum mit `road_bike`, `gravel`, `mtb` und `ebike` vermischt jedoch drei
verschiedene Aussagen: die allgemeine Aktivitätsfamilie, die Rad-Disziplin und
eine mögliche elektrische Unterstützung. Insbesondere ist ein E-MTB zugleich
MTB und elektrisch unterstützt.

Walk und Hike allein würden ausserdem Wanderers primäres Einsatzgebiet
unzureichend abdecken. Cycle muss deshalb bereits im ersten produktiven
Bewertungsschnitt dieselbe vollständige Basis aus neutraler Referenzdauer,
persönlicher beziehungsweise gruppenbezogener ETA und körperlicher Eignung
erhalten. Technische Eignung benötigt dagegen zusätzliche Routenfakten und darf
nicht aus der gewählten Rad-Disziplin erfunden werden.

## Entscheidung

### 1. Getrennte Modellachsen

Die erste produktive Bewertung unterstützt diese kanonischen
Aktivitätsfamilien gemeinsam:

```text
activity_family = walk | hike | cycle
```

Die Registry darf weitere bekannte Familien bereits aufnehmen. Solange für
eine solche Familie kein freigegebenes Modell existiert, lautet der
`model_support_status = unsupported`. Der Familienwert bleibt erhalten.
`unknown` bedeutet dagegen, dass keine eindeutige Familie bestimmt werden
konnte; beide Zustände werden nie gleichgesetzt.

Für `cycle` ist die Disziplin eine eigene Achse:

```text
activity_discipline = touring | road | gravel | mtb | unknown
```

Für `walk` und `hike` ist die Cycle-Disziplin `not_applicable`. Spätere Familien
dürfen eigene Disziplin-Registries ergänzen, ohne die Familienwerte umzudeuten.

Walk und Hike werden nur durch explizite, versionierte Quell-, Kategorie- oder
manuelle Regeln unterschieden. Distanz, Höhenmeter und Surface dürfen die
Familie nie heuristisch umdeuten. Beide Familien dürfen in der ersten
Modellversion denselben Parametersatz und Kombinator referenzieren, behalten
aber getrennte Modellschlüssel und nachvollziehbare Mappings.

Elektrische Unterstützung ist weder Aktivitätsfamilie noch Disziplin. Ein
importierter E-Bike-Typ bleibt als Quellclaim erhalten, wird fachlich aber als
`activity_family = cycle` mit separater Unterstützungsinformation abgebildet.
Das für einen konkreten Ausflug tatsächlich verwendete

```text
assistance_mode = none | electric_assist | unknown
```

gehört in die Ausrüstungszuordnung des jeweiligen Teilnehmers. Die neutrale
Cycle-Referenzdauer bleibt eine menschliche Basis ohne elektrische
Unterstützung. Eine Unterstützung darf die persönliche ETA genau einmal
beeinflussen und nie zugleich Modellschlüssel und Geschwindigkeitsbonus sein.

Quellwert, Mappingrevisionen, Presence, Auflösungszustand je Achse und
Mappingquelle bleiben erhalten:

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

Eine allgemeine Quellangabe wie „Biking“ wird nicht still als `touring`
interpretiert: Die Familie kann exakt `cycle` sein, während der Disziplin-
Quellwert `absent` und die kanonische Disziplin `unknown` bleibt. `cycle +
unknown` darf höchstens eine klar als grob bezeichnete Spanne oder Annotation
liefern. Sie autorisiert keine einzelne produktive Referenzdauer, keinen harten
Dauer- oder Eignungsfilter, keine entsprechende Sortierung und keine Counts.
Bei manueller Anlage wird die Disziplin deshalb explizit gewählt.

Trail und aktives Ausflugsprofil müssen für die produktive Bewertung denselben
Familien- und Disziplinschlüssel besitzen. Ein Mismatch ergibt
`not_applicable`; insbesondere konsumiert ein Gravel-Profil nie still eine
Road-Referenzdauer. Eine spätere Kompatibilitätsmatrix wäre ein eigener,
versionierter Entscheid und kein implizites Fallback.

### 2. Cycle-Referenzdauer

Alle Referenzmodelle geben ihre berechenbaren Zeitkomponenten sowie
Anwendbarkeit, Qualität, Modellversion und Fingerprint aus. Walk und Hike
verwenden ihre jeweils versionierte flach-, aufwärts- und abwärtsbezogene
Komponentenstruktur. Für Cycle gelten eine eigene distanzbezogene und eine
eigene Aufstiegskomponente:

```text
distance_component_s = distance_m / distance_rate_m_per_s
ascent_component_s   = elevation_gain_m / ascent_rate_m_per_s

reference_duration_s =
  round_half_up_seconds(
    max(distance_component_s, ascent_component_s)
    + 0.5 * min(distance_component_s, ascent_component_s)
  )
```

Beide Raten sind endlich und strikt positiv. Der Faktor `0.5`, der Kombinator
und die Rundung des finalen Werts auf die nächste ganze Sekunde, wobei
Halbwerte aufgerundet werden, sind Teil dieser Entscheidung und keine
Kalibrierseeds. Die gesamte Routendistanz steckt bereits in der
Distanzkomponente. Eine
zusätzliche Cycle-Abstiegskomponente ist im Basismodell deshalb
`not_applicable`, nicht die bekannte Zahl null. Technische, Anstiegs- oder
oberflächenbedingte Korrekturen dürfen später nur als getrennt versionierte und
coverage-geprüfte Komponenten hinzukommen.

Eine produktive Cycle-Referenzdauer setzt eine bekannte positive Distanz,
einen bekannten Gesamtaufstieg und eine unterstützte Disziplin voraus.
Bekannte null Höhenmeter sind ein valider flacher Fall; fehlende Höhenfakten
ergeben `unknown` statt null. Konkrete Geschwindigkeiten, Steigraten und
Klassengrenzen werden vor Freigabe je Disziplin kalibriert. Die Entscheidung
für Cycle als Produktumfang hängt nicht von einem bestimmten Seedwert ab.

### 3. Teilnehmer und Gruppen

Ein einzelner skalarer `pace_factor` ist für Cycle nicht ausreichend: Eine
Person kann über die gesamte Distanz und im Aufstieg relativ zu einer anderen
Person unterschiedlich leistungsfähig sein. Teilnehmerprofile halten deshalb
mindestens getrennte Distanz- und Aufstiegsfaktoren; Walk/Hike dürfen zusätzlich
einen Abwärtsfaktor verwenden.

Bei Gruppen wird der limitierende Teilnehmer komponentenweise bestimmt. Für
Cycle gilt:

```text
participant_distance_i = distance_component_s / distance_factor_i
participant_ascent_i   = ascent_component_s / ascent_factor_i

group_distance = max(participant_distance_i)
group_ascent   = max(participant_ascent_i)

group_movement_time =
  round_half_up_seconds(
    max(group_distance, group_ascent)
    + 0.5 * min(group_distance, group_ascent)
  )
```

Erst danach werden koordinations- und pausenbedingte Effekte des
Ausflugsprofils angewandt. So kann auf Distanz und Aufstieg jeweils eine andere
Person limitieren, ohne Teilnehmerleistungen zu mitteln. Dasselbe Prinzip gilt
für Walk/Hike mit deren aktivem Familienkombinator.

### 4. Umfang der ersten Cycle-Bewertung

Die produktive Cycle-Basis liefert:

- eine neutrale Referenzdauer je unterstützter Disziplin;
- persönliche und gruppenbezogene Bewegungs- und Gesamtdauer;
- eine körperliche Konditionsklasse aus erwarteter Dauer und Gesamtaufstieg;
- eine Erklärung mit limitierender Person, dominanter Achse, Datenabdeckung
  und Konfidenz.

Sie behauptet noch keine technische Road-, Gravel- oder MTB-Eignung. Ohne eine
freigegebene persönliche Anstiegsintegration entstehen keine Aussagen über
Einzelanstiege, Rampen oder anhaltende Maximalsteigung. Ohne eine freigegebene
Surface-/Technikintegration entstehen keine Aussagen über
Fahrradkompatibilität aus Oberfläche, `smoothness`, `tracktype`, MTB-Skala,
Stufen oder Wegbreite. Die Disziplin wählt ein Referenz- und Planungsmodell;
sie beweist nicht, dass die Route für das entsprechende Fahrrad technisch
geeignet ist.

Fehlende Anstiegs- oder Surface-/Technikfakten blockieren Referenzdauer, ETA und
körperliche Kondition der Cycle-Basis nicht. Die getrennte technische Eignung
bleibt bis zur benötigten, ausreichend abgedeckten Analyse ausdrücklich
`unknown`.

## Folgen und nicht entschiedene Werte

- Familie, Disziplin, Unterstützungsclaim, Auflösungszustände und
  Modellunterstützung werden getrennt modelliert.
- Referenzdauer, Profile, Gruppen-ETA und körperliche Eignung müssen Walk, Hike
  und Cycle gemeinsam als ersten produktiven Slice qualifizieren; Cycle ist
  keine spätere optionale Familie.
- Das Ausflugsprofil ordnet Fahrrad und tatsächliche Unterstützung einzelnen
  Teilnehmern zu;
  eine Gruppe mit unterstützten und nicht unterstützten Rädern erhält keinen
  pauschalen E-Bike-Faktor.
- Semantische Golden Fixtures decken mindestens Touring, Road, Gravel, MTB,
  unbekannte Disziplin, E-Bike-Quellclaim, flache Routen und widersprüchliche
  Teilnehmerstärken ab; sie pinnen keine unqualifizierten Kalibrierseeds als
  Produktionsdefaults.
- Disziplinspezifische Distanz- und Steigraten, Walk-/Hike-Koeffizienten,
  Profildefaults sowie die Grenzen für leicht, mittel und schwer bleiben
  Kalibriergegenstand. Seedwerte stehen nur unter
  [Evidenz und Kalibrierung](/develop/specs/trail-search/evidence/evidence-and-calibration/).
- Öffentliche Request-, URL- und Responsefelder werden nicht durch dieses ADR
  nebenbei erweitert. Sie benötigen vor sichtbarer Freigabe eine ausdrückliche
  Ergänzung des Trail-Suchvertrags.
