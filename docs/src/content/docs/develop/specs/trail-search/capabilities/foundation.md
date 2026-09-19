---
title: 'FOUNDATION: Bestands- und Suchfundament'
description: Produktvertrag für Bestandsparität, Volltext, Taxonomie, Filterpanel, Counts, Rangefilter, Startpunktsuche und den unterstützten Meilisearch-Betrieb.
editUrl: false
sidebar:
  order: 1
  badge: Entwurf
spec:
  id: CAP-FOUNDATION
  kind: capability
  status: draft
  capability: FOUNDATION
  lastReviewed: '2026-09-19'
---

> **Rolle:** Dieses Dokument beschreibt die fachliche Capability
> `FOUNDATION`: das verlässliche Suchfundament aus bereits vorhandenen
> Trail-Daten, Bestandsfiltern, Sortierungen, Suchprojektion und Oberfläche.
>
> **Normativität:** Die hier formulierten Produktregeln und Qualitätsgrenzen
> sind für `FOUNDATION` verbindlich. Feldtypen, Operatoren, Defaults,
> Fehlercodes und Responsezustände definiert ausschliesslich der
> [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/).
> Cursor- und Snapshot-Lifecycle gehören in
> [ADR 0001](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/),
> Federation-Sicherheit in den
> [Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/)
> und die persistente Control-Plane in
> [STATE1](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/).
> Dieses Dokument wiederholt deren Wire- und Zustandsautomaten nicht als
> zweite Vertragsquelle.

Capability-übergreifende Regeln zu Bestandsparität, Datenqualität,
Provenienz, Jobs, Plugins, Suchkontext und Tests stehen in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/).
Mögliche Arbeitsschnitte, Codes und Abhängigkeiten stehen nur im
[Delivery- und Beitragskatalog](/develop/specs/trail-search/delivery/). Sie
sind weder zugesagte PRs noch zusätzliche Produktsemantik.

## 1. Zweck und Abgrenzung

`FOUNDATION` macht die vorhandene Suche zuverlässig und verständlich. Die
Capability darf dafür bestehende PocketBase- und Meilisearch-Daten,
Suchkonfiguration, eine deterministische Suchprojektion, Reindex und
denormalisierte Suchbegriffe verwenden. Sie verspricht noch keine neue
Routenanalyse, kein persönliches Leistungsmodell und keinen externen
Zeitkontext.

Zum eigenständig nutzbaren Ergebnis gehören:

- ein stabiler, für UI, URL, App und spätere Dialogadapter gemeinsamer
  Suchvertrag;
- eine nachvollziehbare Volltextreihenfolge;
- eine funktionierende Kategorie-/Subkategorie-Hierarchie;
- ein zugängliches Filterpanel ohne Fake-Controls oder Beispielzahlen;
- owner-private gespeicherte Suchaufträge und eine sichtbare Standardsuche für
  den Einstieg in die Trail-Liste;
- reale disjunktive Counts und robuste Range-Domains, sobald ihre
  Datenscheibe freigegeben ist;
- billige Filter und vorhandene Sortierungen mit ehrlicher Semantik;
- die unverändert erhaltene Startpunkt-Ortssuche samt Radius; sowie
- ein konkret gepinntes und betreibbar migrierbares Meilisearch-Profil.

`has_photos` gehört fachlich zu `FOUNDATION`, wird aber erst aktiviert, wenn
das kanonische Assetmodell und ein actor-sicheres Overlay beweisen, dass nur
für den Suchenden sichtbare Trail- und Waypoint-Assets einfliessen. Enger
sichtbare Inhalte werden nie der Einfachheit halber in das globale
Traildokument kopiert.

## 2. SRCH0 und Trail-Suchvertrag v1

SRCH0 liefert die fachlich geprüfte Regressionsbaseline des heutigen Produkts.
Es erfasst vor jeder UI-, Gateway-, Projektions- oder Indexmigration die
tatsächlich unterstützte Matrix aus Filtern, Sortierungen, ACL-Kontexten,
API- und URL-Zuständen und sichert deren korrektes Verhalten ab. Bekannte
Fehler werden vor der Abnahme korrigiert. Die vollständige Bestandsliste
besitzt SRCH0 selbst; die capability-übergreifenden Cutover-/Rollbackregeln
stehen getrennt in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/).
Der nicht normative
[Bestandsaudit vom 30. August 2026](/develop/specs/trail-search/evidence/evidence-and-calibration/#bestandsaudit-vom-30-august-2026)
liefert dafür den datierten Ausgangspunkt mit Codeankern; SRCH0 verifiziert ihn
neu und ergänzt ihn durch ausführbare Evidenz.

Jeder Altfall wird im SRCH0-Korpus als eine der folgenden Klassen erfasst:

1. semantisch unverändert zu erhalten;
2. als nachgewiesener Bestandsfehler vor der SRCH0-Abnahme zu korrigieren; oder
3. als wirkungsloser Implementierungsunfall zu entfernen, mit Nachweis
   unveränderter fachlicher Ergebnisse.

Historische Referenzdaten, beobachtete Ergebnisse und stabile Case-IDs bleiben
unverändert. Die aktiven Erwartungen beschreiben das fachlich korrekte
Ergebnis, einschliesslich begründeter Korrekturen auf denselben Case-IDs.
Unabhängige fachliche Eigenschaften sichern diese Erwartungen zusätzlich ab.
Ein Vergleich mit historischen Fehlresultaten oder ein automatisch
aktualisiertes Golden ist kein Korrektheitsnachweis.

Jeder nachgewiesene Fehler im SRCH0-Bestandsumfang blockiert Abnahme und Merge,
bis Produktkorrektur und Regressionstests auf demselben Integrationsstand
bestehen. `knownViolation`, Skip oder erwartete Fehlschläge sind keine
zulässigen Ausnahmen. Testkorpus und Korrekturen dürfen getrennte PRs sein;
die Korrekturen bleiben Voraussetzungen der SRCH0-Abnahme.

Damit gehören etwa fehlerhafte Radiusgrenzen, Unknown-Difficulty-Ergebnisse,
Listenprojektionen oder ein destruktiver normaler Indexstart nicht erst in
einen späteren Baselineverbraucher. [SRCH-COMP](/develop/specs/trail-search/work-items/search/srch-comp/),
[SRCH2](/develop/specs/trail-search/work-items/search/srch2/) und `SEC-VIS-0`
konsumieren die korrigierte Baseline und besitzen ihre weiterführenden
Vertrags-, Filter- und Sicherheitsänderungen. Neue Semantik benötigt ein
begründetes Delta-Overlay; bereits bekannte Bestandsfehler dürfen darüber
nicht vertagt werden.

Die angegebene beziehungsweise importierte Schwierigkeit bleibt ein
Bestandsfeld und wird nie in persönliche Schwierigkeit umgedeutet.

IDX0 bleibt unabhängig implementierbar und besitzt den heutigen Indexbootstrap
sowie den Legacy-Produzenten von `SearchReadinessV1`. Nötige Korrekturen des
Bestandsstarts werden bereits für SRCH0 nachgewiesen; die vollständige
Readinessproduktion und ihre Betriebsregeln bleiben IDX0-Scope. Daraus
entsteht keine Implementierungsabhängigkeit von IDX0 auf SRCH0, und weder
STATE1 noch die M1-Enginemigration werden in den normalen App-Start gezogen.

Der [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
trennt den reproduzierbaren fachlichen Suchauftrag, die
transport-/paginierungsbezogene Anfrage und die Antwort. Ein Legacyadapter
übersetzt alte Requests in diese Semantik; nur semantisch identische
Zustände dürfen automatisch in eine kanonische v1-URL überführt werden. Der
Browser erzeugt keine freien Meilisearch-Ausdrücke.

Die erste Implementierung dieses Vertrags auf dem heutigen Backend ist kein
fachlich reduzierter Zwischenvertrag. Ein späterer Gateway- oder
Indexwechsel ersetzt nur die interne Ausführung. Für bereits unterstützte
Requests bleibt der Wechsel unsichtbar und wartet auf die einschlägigen
Security-, Paritäts- und Zustandsfreigaben.

## 3. Volltext und Ranking

Die Zielreihenfolge der durchsuchbaren Attribute ist:

1. `name`
2. `location`
3. `taxonomy_search_terms`
4. `tags`
5. `waypoint_names`
6. `description`
7. `author_name`

Lokale und föderierte Taxonomien liefern dieselben Arten von Suchbegriffen.
Alle bekannten Übersetzungen dürfen in einem gemeinsamen Suchterm-Array
liegen; die angezeigte Übersetzung bleibt davon unabhängig. Namen heutiger
Waypoints dürfen denormalisiert werden, solange sie die Sichtbarkeit des
Trails vollständig erben. Erhält ein Waypoint später eine engere
Sichtbarkeit, wandert sein Name in ein scopegebundenes Overlay. GPX-Rohtext
gehört nicht in den Volltextindex.

„Routenname zuerst“ ist ein Attribut-Tie-Breaker, keine absolute
Ranggarantie. Wortabdeckung, Tippfehler und Wortnähe können davor wirken.
„Relevanz“ übergibt keine explizite Sortierung; bei leerer Query gilt die im
Suchvertrag festgelegte stabile Defaultsortierung nach Erstellzeit und
Trail-ID.

Änderungen und Löschungen von Waypoints, Kategorien, Subkategorien und deren
Übersetzungen invalidieren alle betroffenen Kern- oder
Overlayprojektionen. Tests decken diese Pfade zusätzlich zu Trail-Create und
-Update ab.

Die UI unterscheidet zwei heute leicht verwechselbare Begriffe:

- `location` ist frei erfasster Text und Teil der Volltextsuche.
- „Ort und Umkreis“ löst einen Namen zu Koordinaten auf und setzt einen
  räumlichen Filter.

## 4. Kategorien und Subkategorien

Kategorie- und Subkategoriedaten sowie die wesentlichen Filterfelder sind
bereits vorhanden. Die hierarchische Auswahl verwendet folgende Semantik:

- unterschiedliche Filtergruppen werden mit AND verbunden;
- mehrere Kategorien oder Optionen derselben Gruppe werden mit OR
  verbunden;
- „ohne Subkategorie“ für Kategorie X bedeutet Kategorie X plus fachlich
  fehlende Subkategorie;
- Kategorien ohne passende Unterkategorien bleiben auswählbar; und
- Auswahl und Sortierung werden mit stabilen IDs in der URL serialisiert.

Presence und Mapping sind getrennte Aussagen. Für Kategorie und
Subkategorie unterscheidet die Projektion jeweils „fehlt“ von „vorhanden“;
ein vorhandener fremder Begriff ist zusätzlich `mapped`, `unmapped` oder
`ambiguous`. Mappingstatus wird nur bei vorhandener Relation ausgewertet.
Der Subkategoriebranch existiert nur bei vorhandener, eindeutig gemappter
Kategorie.

Damit gilt insbesondere:

- ein fremder, nicht zuordenbarer Begriff ist nie „ohne Kategorie“ oder
  „ohne Subkategorie“;
- `unmapped` und `ambiguous` bleiben eigene, ausblendbare Buckets; und
- das Ausblenden einer lokalen Kategorie versteckt fremde, nicht gemappte
  Werte nicht implizit.

Wird eine Kategorie abgewählt, entfernt die UI ihre nun verwaisten
Subkategoriebedingungen. Eine unbekannte oder inzwischen gelöschte ID darf
die Suche nicht still verbreitern, sondern bleibt als entfernbarer aktiver
Zustand mit verständlichem Fehler sichtbar.

## 5. Filterpanel

Das visuelle Redesign ist eine eigenständig nützliche Produktscheibe. Es ist
nicht davon abhängig, dass alle späteren Filter, Counts und Histogramme
gleichzeitig freigegeben werden. Es zeigt jedoch ausschliesslich reale
Funktionen und Daten.

Der Produktvertrag lautet:

- Freitext, Kategorie mit Subkategorie, Ort/Umkreis und Sortierung sind ohne
  Suche in einer langen Liste erreichbar. Weitere prominente Gruppen dürfen
  responsiv und nach belegtem Nutzungsbedarf variieren.
- Sämtliche Bestandsfilter bleiben unter „Weitere Filter“ vollständig
  erreichbar. Einklappen löscht keinen Suchzustand.
- Aktive Bedingungen, Anzahl aktiver Filter, „alle zurücksetzen“ und die
  aktuelle Gesamttrefferzahl sind ausserhalb einzelner Akkordeons erkennbar.
  Der Scope einer Reset-Aktion ist sichtbar und eindeutig.
- Aktive eingeklappte Gruppen bleiben sichtbar oder werden als aktive Chips
  zusammengefasst.
- Kategorie-Icons besitzen einen sichtbaren zugänglichen Namen; unklare
  Symbolik hängt nicht allein von Hover-Tooltips ab.
- Tastatur, Screenreader, Fokusführung und ausreichend grosse Touch-Ziele
  gehören zum Slice.
- Desktop-Seitenpanel und mobiles Sheet teilen URL-/Store-Zustand und
  fachliche Gruppenreihenfolge. Schliessen, Browser-Zurück und erneutes
  Öffnen verlieren keine Auswahl.
- Rangefilter zeigen Einheit, Handles und offenen Endbereich verständlich.
  Counts und Histogramme erscheinen erst aus echten API-Daten.
- Lade-, Fehler- und fehlende-Capability-Zustände sind definiert.
  Ausgefallene Count-Gruppen sperren die zugrunde liegenden Bestandsfilter
  nicht.
- Noch nicht freigegebene Capabilities erscheinen nicht als dauerhaft
  deaktivierte Versprechen.

### Gespeicherte Suchen und Standardsuche

Der [Vertrag für gespeicherte Trail-Suchen](/develop/specs/trail-search/contracts/saved-trail-search-v1/)
ergänzt das Panel um benannte, geräteübergreifende Suchaufträge. Persistiert
wird die normalisierte `TrailSearchSpecV1`, nicht ein Browserfilter-Blob und
nicht die Resultate einer Ausführung. Ein separater Benutzer-/Surface-Pointer
kann genau eine Suche als Default für `trail_list` auswählen.

Der Default gilt nur beim frischen Einstieg über eine leere Trail-Listenroute.
Explizite URLs, Deep Links, Reload und Browser-Zurück/-Vorwärts gewinnen und
werden nie mit ihm gemischt. Nach erfolgreicher Auflösung wird die vollständige
Spec per Replace in die kanonische V1-URL geschrieben. Sämtliche Bedingungen
bleiben dadurch sichtbar, editierbar und teilbar; der Default ist kein
unsichtbarer Filteroverlay.

„Alle Filter entfernen“ und „Meine Standardsuche wiederherstellen“ sind
getrennte Aktionen. Die erste erzeugt die fachlich uneingeschränkte V1-Suche
und darf den Default im selben Seitenbesuch nicht sofort erneut anwenden. Eine
inzwischen ungültige gespeicherte ID oder Capability führt in einen sichtbaren
Reparaturzustand und niemals in eine still verbreiterte Suche.

Der Prototyp wird mindestens mit den Aufgaben „Kategorie plus Subkategorie
wählen“, „bestehenden Startpunktradius setzen“, „Sortierung ändern“,
„aktiven Filter wiederfinden“ und „Nulltrefferkombination korrigieren“
getestet. Counts und Histogramme sind eine additive Datenscheibe, keine
Voraussetzung für die neue Informationsarchitektur.

## 6. Trefferzahlen

Trefferzahlen sind disjunktiv. Die Zahl neben einer Option berücksichtigt
alle aktiven Bedingungen ausser der eigenen Facettengruppe und fügt dann die
betrachtete Option hinzu. „Radfahren (42)“ unter aktivem „Wandern“ bedeutet
somit 42 Radfahr-Trails unter allen übrigen Filtern und nicht einen globalen
oder statischen Beispielwert.

Für die Taxonomiehierarchie gilt:

- Subkategorie-Counts entfernen den Subkategorienfilter, behalten aber die
  ausgewählten Kategorien.
- Kategorie-Counts entfernen auch die von dieser Gruppe abhängigen
  Subkategoriebedingungen.
- „Ohne Subkategorie“ und normale Subkategorien derselben Kategorie werden
  innerhalb der Gruppe mit OR verbunden.

Ergebnis, Total, angeforderte Facetten und Histogramme entstehen aus demselben
normalisierten Suchauftrag und Suchkontext. Der Server bündelt die nötigen
Operationen; eine Option oder ein Bucket erzeugt nie eine eigene
Backendquery. Die UI drosselt Interaktionen sinnvoll und verwirft veraltete
Antworten.

`estimatedTotalHits` ist für zugesagte Counts unzulässig. Kategoriegruppen
verwenden exhaustive Verteilungen, Range-Buckets einen nachweislich exakten
Totalmechanismus für die tatsächlich berechnete Ergebnismenge.
Integrationstests überschreiten bewusst Meilisearchs Standardgrenze von
1'000 Treffern.

Für UI-Zustände gilt:

- `0` ist eine berechnete Zahl, nicht „unbekannt“.
- Angeforderte Gruppen sind vollständig `ready` oder `error`; ein Fehler
  enthält weder Teilzahl noch alten Cachewert oder erfundene Null.
- Treffer und Total bleiben atomar. Ein Aggregationsfehler lässt Filter und
  bestehende Auswahl bedienbar.
- Eine gültige ausgewählte Option mit Count 0 bleibt sichtbar und
  abwählbar.
- Bucket-IDs sind schema-versioniert und unabhängig von Übersetzung,
  Position und aktuellem Count. URLs speichern stabile IDs oder SI-Grenzen.

Der Server veröffentlicht ein deterministisches Budget. Im v1-Profil sind
höchstens 16 angeforderte Aggregationsgruppen, 32 interne Suchoperationen und
16 sichtbare Histogrammbuckets pro Gruppe zulässig. Passt bereits der
Treffer-/Totalkern nicht, wird der Request typisiert abgelehnt. Passt nur
eine optionale Gruppe nicht vollständig, fällt ausschliesslich diese Gruppe
aus. Die genauen Planungs- und Fehlerregeln bleiben Eigentum des
[Trail-Suchvertrags v1](/develop/specs/trail-search/contracts/trail-search-v1/).

Bei Federation bedeutet ein Count die Anzahl autorisiert auffindbarer,
lokal materialisierter Trail-Einträge im gebundenen Instanz-Snapshot, nicht
eine Live-Zahl des Fediverse. Die UI erklärt dies als „Treffer im Suchbestand
dieser Instanz; föderierte Inhalte entsprechen dem zuletzt verifizierten
Stand“. Ein Peer-Ausfall ändert eine gebundene Zahl nicht spontan; eine
Snapshotänderung oder Expiry beendet stattdessen den bisherigen Kontext. Die
vollständige Suchuniversum-, Expiry-,
ACL- und Cache-Semantik steht in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/#6-search-context-counts-und-federation)
und im
[Security-Vertrag](/develop/specs/trail-search/contracts/federation-security/).

Ein aktiver Routenradius zählt exakt die durch das angenommene
GeoJSON-Direct-Prädikat gelieferte Ergebnismenge desselben Snapshots. Die
mögliche Abweichung dieses räumlichen Prädikats von der mathematisch exakten
Rohgeometrie wird einmal zentral im
[CONTEXT-Vertrag](/develop/specs/trail-search/capabilities/context/) und in
[ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
erklärt; die UI versieht nicht jede ansonsten exakte Zahl pauschal mit
`≈`. Facetten- und Histogrammserien erhalten bei ihrer Einführung einen
eigenen Offline-Audit und ein geeignetes Fehlerbudget.

## 7. Rangefilter und Histogramme

Globale Datenmaxima bestimmen nie direkt die Slider-Domain. API,
Suchprojektion, URL und Bucketgrenzen verwenden SI-Einheiten; Kilometer,
Meilen, Meter, Fuss und lokalisierte Dauer sind Präsentation. Die v1-Leitern
sind feste, halboffene Buckets mit einem offenen letzten Bucket und keine
Quantil-Buckets:

| Filter | Kanonischer Wert und UI-Schritt | Feste Bucketgrenzen v1 | Default-Endanschlag | UI-Cap |
| --- | --- | --- | ---: | ---: |
| Distanz | `distance_m`; 1 km oder äquivalent | 0/2/5/10/15/20/30/50/75/100/150/200/300/+ km | 50 km | 300 km |
| Referenzdauer | `reference_duration_s`; 30 Minuten | 0/0,5/1/2/3/4/6/8/10/12/16/24/+ h | 10 h | 24 h |
| Aufstieg | `elevation_gain_m`; 50 m | 0/100/250/500/750/1'000/1'500/2'000/3'000/4'000/6'000/+ m | 2'000 m | 6'000 m |
| Abstieg | `elevation_loss_m`; 50 m | wie Aufstieg | 2'000 m | 6'000 m |
| Höchster Punkt | signiertes `max_elevation_m`; 100 m | −500/0/500/1'000/1'500/2'000/2'500/3'000/3'500/4'000/4'500/5'000/6'000/+ m | 3'500 m | 6'000 m |

Der UI-Cap begrenzt nur die Sliderdarstellung. Höhere valide Werte bleiben
im offenen Bucket. Beim höchsten Punkt sind plausible bekannte Rohwerte von
−500 bis 9'000 Metern zulässig; Werte oberhalb des Caps bleiben bekannt,
nicht plausible oder unzureichend belegte Werte werden unknown. Der
signierte Höhentyp und die genaue Range-Wiresemantik stehen im
[Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/).

Der erste `FOUNDATION`-Slice kann Distanz, Aufstieg und Abstieg aus
vorhandenen, nachgewiesen bekannten Daten bedienen. Referenzdauer und
höchster Punkt werden erst freigeschaltet, wenn ihre in
[DERIVED](/develop/specs/trail-search/capabilities/derived/) definierten
Fakten, Presence-Regeln und Backfills vorliegen. Die registrierten Leitern
bleiben trotzdem der gemeinsame v1-Vertrag.

### 7.1 Perzentilbasierter Endanschlag

Ein versionierter Statistikjob berechnet p99 nur aus bekannten, validen
Werten des zum Snapshot öffentlich auffindbaren Basiskatalogs. Private,
geteilte oder nur für einen Actor sichtbare Trails fliessen nie ein. Für
eine Domain aus Aktivitätsfamilie, normalisiertem Quell-Scope und Feld gelten
gleichzeitig:

- mindestens 500 bekannte Werte;
- mindestens 80 Prozent bekannte Feldabdeckung;
- deterministischer Nearest-Rank-Wert `x[ceil(0.99 × n)]` der aufsteigend
  sortierten bekannten Werte;
- Aufrundung auf die nächste registrierte Grenze oder Sättigung am UI-Cap;
- effektiver offener Endanschlag als kleinerer Wert aus UI-Cap und dem
  grösseren Wert aus ausgeliefertem Default und gerundetem p99;
- bei mehreren Aktivitätsfamilien der grösste gültige Endanschlag; ohne
  Familienauswahl ein eigener versionierter `all`-Snapshot.

Bei zu kleinem oder zu lückenhaftem Korpus wird nicht extrapoliert. Zuerst
gilt eine ausreichend grosse breitere öffentliche Domain derselben
Aktivitätsfamilie, danach der ausgelieferte versionierte Default. Antwort und
Suchkontext nennen Bucketprofil, SI-Einheit, effektiven Endanschlag,
Statistikrevision und tatsächlich verwendete Domain samt Fallbackgrund.
Counts und Histogramme werden weiterhin für den aktuellen Request, Actor,
ACL-Scope und Suchkontext berechnet; sie werden nie aus der öffentlichen
p99-Statistik übernommen.

### 7.2 Missing, Unknown und offener Bereich

Ein ausgelassener Rangefilter kompiliert kein Zahlenprädikat und lässt
bekannte wie unbekannte Werte zu. Ein aktiver Rangefilter kann unbekannte
Werte ausschliessen, einbeziehen oder ausschliesslich unknown auswählen. Die
genauen Requestformen definiert der Suchvertrag.

Steht der obere Handle am offenen Endanschlag, entfällt nur die Obergrenze;
eine gesetzte Untergrenze bleibt erhalten. Der Overflow-Bucket enthält nur
bekannte endliche Werte und ist auch allein auswählbar. Unknown besitzt eine
eigene stabile ID und wird nie in den Overflow-Bucket gemischt.

Ein historischer Datenbank- oder Clientdefault `0` beweist keinen bekannten
Wert. Autoritativ gelieferte oder erfolgreich versioniert berechnete null
Höhenmeter bleiben dagegen bekannt. Distanz 0 und Referenzdauer 0 sind keine
validen bekannten Trailmetriken; Höhe 0 kann ein bekannter Wert auf
Meereshöhe sein. Vor der Freigabe eines Histogramms braucht jedes Feld eine
Presence-/Quality-Aussage, Provenienz, Validitätsregel und einen Backfill.

Räumliche Radien sind keine Korpus-Rangefilter: Sie besitzen weder p99 noch
Histogramm, Overflow oder „Unknown einbeziehen“. Schwierigkeit, Bewertung
und der frühere Prototypwert „Steilheit“ werden ebenfalls nicht als
scheinbar kontinuierliche Slider modelliert.

## 8. Billige Filter und Sortierung

Folgende Erweiterungen nutzen weitgehend vorhandene Daten, behalten aber
ihre fachlichen Grenzen:

- **Hat Fotos:** mindestens ein für den Suchenden sichtbares direktes
  Trail- oder Waypoint-Asset. Summit-Log-Fotos gehören zur Aktivität und
  bleiben getrennt. Link/Unlink sowie eigenständige Asset- oder
  Waypoint-Sichtbarkeit invalidieren Kern beziehungsweise Overlay.
- **Von mir geliked:** der lokale Like-Zustand des aktuellen Actors.
- **Mindest-Likes und beliebt:** `local_observed_like_count` macht die
  Semantik ausdrücklich „auf dieser Instanz beobachtet“. Ein behauptetes
  Origin-Aggregat bleibt getrennte, provenancebehaftete Detailinformation.
- **Sortierung:** Relevanz ohne explizites `sort` sowie alle bereits
  vorhandenen Sortierungen für Name, Datum, Erstellzeit, Distanz,
  Höhenmeter, Likes, Dauer und angegebene Schwierigkeit bleiben erhalten.

Die heutigen Sortierungen nach Dauer und Schwierigkeit beziehen sich weiter
auf ihre bisherigen Quellwerte. Neutrale Referenzdauer, erwartete Dauer und
persönliche Eignung erhalten später eigene Keys aus
[DERIVED](/develop/specs/trail-search/capabilities/derived/) und deuten die
Bestandskeys nicht um.

Likes und Bewertungen sind verschiedene Konzepte: Ein Like ist eine binäre
soziale Aktion; eine Bewertung ist ein Skalenurteil mit Stimmenzahl,
Mittelwert und Schutz vor verzerrter Sortierung bei sehr wenigen Stimmen.
Ratings werden daher nicht aus Likes abgeleitet und gehören als eigene
Produktscheibe zu `DERIVED`.

Solange ihre Gates nicht erfüllt sind, zeigt die UI weder Sterne, ÖV,
Rundweg, Anstiege noch persönliche Schwierigkeit als scheinbar
funktionierende Controls. Alle bereits produktiven Filter, insbesondere
angegebene Schwierigkeit und Startpunkt-Umkreis, bleiben sichtbar und
wirksam.

## 9. Erhaltene Startpunktsuche und L1

Der heutige Modus „Startpunkt im Umkreis“ bleibt fachlich und in gespeicherten
URLs erhalten. Sein Radius bietet die diskreten UI-Stops 0,5, 1, 2, 5 und
10 Kilometer, startet bei 2 Kilometern und besitzt einen harten Cap von
10 Kilometern. Der Request enthält trotzdem immer einen expliziten Radius.

Der spätere Modus „Route berührt den Radius“ ist eine additive
`CONTEXT`-Capability mit eigenem Requestfeld und eigenem Spatial-Vertrag.
Eine alte Startpunkt-URL wird nie still in den Routenmodus umgedeutet. Die
vollständige Routenradius-Semantik steht in
[CONTEXT](/develop/specs/trail-search/capabilities/context/) und
[ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/).

Die bestehende Konfiguration `NOMINATIM_URL` kann bereits den öffentlichen
OSMF-Dienst oder einen anderen beziehungsweise selbst gehosteten Endpoint
ansprechen. Der heutige Limiter erkennt den öffentlichen Dienst jedoch nur
per URL-Substring, und der Store überträgt das gewünschte Ergebnislimit
nicht zuverlässig. L1 korrigiert diese Bestandsprobleme, bevor eine davon
betroffene Produktscheibe veröffentlicht wird.

Im offiziellen OSMF-Nominatim-Modus gelten verbindlich:

- Submit-only statt Autocomplete oder versteckter Debounce-Abfragen;
- ein wirklich serialisierter instanzweiter Egress-Limiter über alle
  Benutzer, Prozesse und Replikas;
- Cache und Singleflight, identifizierender User-Agent, Attribution und
  dokumentierter Datenschutz;
- keine Backfills, periodischen Jobs, systematischen Kategorieabfragen oder
  POI-Browsing-Requests;
- Respekt vor `429` und `Retry-After` ohne Retry-Burst; und
- die sichtbare Angabe von Provider, Attribution und möglicher Übertragung
  des Suchtexts an diesen Provider.

Ein clientseitiger Abbruch verhindert veraltete UI-Antworten, macht einen
bereits gestarteten Upstreamrequest aber nicht ungeschehen. Auch andere
Endpoints gelten nicht automatisch als unbegrenzt; ihr Usage-Profil
bestimmt Rate und erlaubte Funktionen.

Typisierte Scopes wie „Unterkünfte“, Search-as-you-type und providerneutrale
Place-Anker gehören zu
[CONTEXT](/develop/specs/trail-search/capabilities/context/). Fehlt die dafür
nötige Provider-Capability, bleibt nur diese neue Scheibe verborgen. Die
bestehende Submit-Suche und der Startpunktradius bleiben ohne Photon oder
einen anderen Autocomplete-Provider funktionsfähig.

## 10. M1: unterstütztes Meilisearch-Profil

Die aktuellen Ausgangsprofile sind uneinheitlich:

- Root-Compose und Quick-Install verwenden Meilisearch 1.36.0;
- Dev-, Prod-Compose und das minimale Docker-Beispiel verwenden 1.11.3; und
- frühere Benchmarkevidenz mit 1.44.0 bleibt als Vergleichsstand relevant.

Die für GeoJSON Direct angenommene Evidenz in
[ADR 0002](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
qualifiziert das offizielle Meilisearch-Image 1.53.1 samt Image-Digest. M1
operationalisiert diesen Entscheid; G1 liefert die Evidenz für M1 und wartet
nicht umgekehrt auf den bereits vollzogenen Betriebsrollout.

Das Upgrade ist eine Breaking Requirement. Kompatibilität zu 1.11.3 bleibt
nicht zugesagt. M1:

- inventarisiert 1.11.3 und 1.36.0 als reale Ausgangsprofile;
- ermittelt und testet den tatsächlich unterstützten Upgradepfad bis zum
  konkret gepinnten 1.53.1-Profil einschliesslich nötiger Zwischenstände;
- wechselt Root, Dev, Prod, CI, Quick-Install und Dokumentation gemeinsam;
- lehnt nicht qualifizierte Server im Startup-/Health-Preflight mit einer
  klaren Migrationsmeldung ab; und
- veröffentlicht Dump/Restore-, Upgrade/Rollback- und Soak-Nachweise sowie
  ein betreibbares Runbook.

M1 ist ein eigener Betriebsrelease mit Vorlauf für Self-Hoster. Es wird nicht
mit Geo-Integration, Filterpanel oder einer anderen sichtbaren Suchfunktion
gebündelt. Eine spätere Engineversion erhält ein neues unveränderliches
Engineprofil und eine neue Indexgeneration; sie ändert nicht allein deshalb
den öffentlichen Spatial- oder Payloadvertrag. Gleichwertigkeit verlangt
erneut bestandene GeoJSON-, Known-Limitation-, Dump/Restore-,
Upgrade/Rollback- und Soak-Tests.

`_geojson` existiert erst seit Meilisearch 1.22. Neuere Versionen gelten
nicht automatisch als korrekt. Die Diagnosen für bekannte
GeoJSON-Zellabdeckungs- und Hierarchieeffekte bleiben bei Upgrades aktiv:
eine bekannte Enginegrenze ist sichtbar, aber allein nicht gating; ein
Diagnosefehler blockiert die Freigabe. Wanderer trägt dafür keinen privaten
Meilisearch-Fork.

## 11. Qualitätsgates

Eine `FOUNDATION`-Produktscheibe wird nur freigegeben, wenn alle für sie
einschlägigen Punkte belegt sind:

- Alle berührten aktiven SRCH0-Erwartungen und unabhängigen fachlichen
  Eigenschaften bestehen, einschliesslich der korrigierten Bestandsfälle.
  Historische Fehlresultate sind kein Paritätsziel. Bewusste neue Semantik
  benötigt ein grünes, ownergebundenes Delta-Overlay auf derselben Case-ID;
  ein bekannter Bestandsfehler darf dadurch nicht freigestellt werden.
- Ein sichtbares Control verändert reale Suchresultate oder stellt einen
  klaren realen Zustand dar; Beispielwerte und funktionslose Controls sind
  ausgeschlossen.
- Kategorie und Subkategorie funktionieren gemeinsam, behalten stabile
  URLs und unterscheiden absent, unmapped und ambiguous.
- Counts sind disjunktiv, exhaustive für ihre tatsächlich berechnete Menge,
  budgetiert und unterscheiden 0, Ladezustand und Fehler.
- Range-Domains werden nicht von einem Ausreisser unbenutzbar. p99-Domains
  erfüllen Mindestkorpus und Coverage; Caps erhalten höhere bekannte Werte
  im offenen Bucket.
- Unknown wird weder zu 0 noch zu „leicht“, und Presence-/Provenienzregeln
  sind vor Histogrammfreigabe nachgewiesen.
- Alle vorhandenen Sortierungen bleiben wirksam; neue Sortierschlüssel
  deuten ihre Semantik nicht um.
- Das Panel besteht Tastatur-, Screenreader-, Responsive- und die in
  Abschnitt 5 genannten Aufgaben-Tests.
- Gespeicherte Suchen bestehen Owner-Isolation, optimistische Nebenläufigkeit,
  URL-vor-Default-Präzedenz, History/Reload und den Reparaturzustand ohne
  stilles Entfernen ungültiger Bedingungen.
- Der öffentliche OSMF-Nominatim-Modus sendet weder Autocomplete noch
  systematische Requests und besteht Limiter-, Cache-, Retry-, Attribution-
  und Datenschutztests.
- Eine Installation ohne typisierten Autocomplete-Provider behält die
  bestehende Submit- und Startpunktsuche.
- IDX0 lässt einen passenden vorhandenen Bestandsindex beim Neustart
  unangetastet, stellt fehlende oder unerwartet leere Legacyindizes vor der
  Suchfreigabe wieder her und produziert beide Search-Readiness-Endpunkte.
  Ein dauerhaft nicht ableitbarer Polyline-Cache degradiert nur den betroffenen
  Trail und nie die globale Readiness. SRCH-COMP konsumiert den Zustand für
  sämtliche First-Party-Suchwege.
- M1 besteht den unterstützten Migrationspfad, Startup-Preflight,
  Dump/Restore, Rollback und Soak für das gepinnte Engineprofil.
- Gateway-, Cursor-, Federation- und Generationen-Cutover erfüllen die
  verlinkten Security-, ADR- und STATE1-Gates; ein Browser-Direktzugriff auf
  Meilisearch ist danach kein unterstützter Suchpfad.

Die capability-übergreifenden Negativ-, Reconciliation-, Backfill- und
Securitytestregeln werden nicht hier dupliziert; sie stehen in den
[gemeinsamen Invarianten](/develop/specs/trail-search/shared-invariants/#8-freigaben-und-testverantwortung).

## 12. Verwandte Dokumente

- [Trail-Search-Übersicht](/develop/specs/trail-search/)
- [Gemeinsame Invarianten](/develop/specs/trail-search/shared-invariants/)
- [Trail-Suchvertrag v1](/develop/specs/trail-search/contracts/trail-search-v1/)
- [Gespeicherte Trail-Suchen v1](/develop/specs/trail-search/contracts/saved-trail-search-v1/)
- [Federation-Security](/develop/specs/trail-search/contracts/federation-security/)
- [Federation-, Index- und Gateway-Zustandsvertrag v1](/develop/specs/trail-search/contracts/federation-index-gateway-state-machines-v1/)
- [ADR 0001: Search-Cursor und Snapshot-Lifecycle](/develop/specs/trail-search/decisions/0001-search-cursor-and-snapshot-lifecycle/)
- [ADR 0002: GeoJSON Direct für den Routenradius](/develop/specs/trail-search/decisions/0002-geojson-direct-route-radius/)
- [Delivery und Beiträge](/develop/specs/trail-search/delivery/)
- [DERIVED](/develop/specs/trail-search/capabilities/derived/)
- [CONTEXT](/develop/specs/trail-search/capabilities/context/)
- [DIALOG](/develop/specs/trail-search/capabilities/dialog/)
