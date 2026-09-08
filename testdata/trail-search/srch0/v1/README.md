# SRCH0 – Suchbestand nach Themen

Automatisch aus den Fallgruppen erzeugt. `node scripts/srch0/manifest.mjs --write` aktualisiert Digests und Übersicht, keine Beobachtungen.

Bestandsanker: `e9b7a8cade980002acbcf2e2f5b2a083934f29d2` · 2026-09-07 · 280 Fälle in 28 Gruppen.

## Review-Einstieg

1. Die [fachliche Bewertung](../../../../scripts/srch0/BEFUNDE.md) erklärt Fehler und offene Produktfragen.
2. In der betroffenen Gruppe stehen gemeinsame Metadaten einmal, Eingabe und Beobachtung vollständig beim jeweiligen Fall.
3. Die [Anleitung](../../../../scripts/srch0/README.md) ordnet Adapter und Solländerungen ein. Generierte Dateidigests stehen im [Manifest](manifest.json).

**Bekannte fachliche Fehler sind Merge-Blocker.** Historische Beobachtungen erlauben keine grünen Tests bei verletzten Eigenschaften. Tests und korrekte Sollwerte liegen auf SRCH0, Produktfixes auf einem separaten Branch.

## Fallgruppen

| Gruppe | Fälle | Consumer |
| --- | ---: | --- |
| [cases/browser/navigation.json](cases/browser/navigation.json) | 11 | trail-list-browser, map-browser |
| [cases/compiler/api-actors.json](cases/compiler/api-actors.json) | 6 | actor-search |
| [cases/compiler/api-consumers.json](cases/compiler/api-consumers.json) | 9 | profile-trails, profile-lists, recommendation, trail-bounding-box, map-cluster, trail-filter-values |
| [cases/compiler/api-proxy.json](cases/compiler/api-proxy.json) | 6 | search-proxy, map-id-details, multi-search-proxy |
| [cases/compiler/api-upload.json](cases/compiler/api-upload.json) | 3 | upload-duplicate |
| [cases/compiler/dto.json](cases/compiler/dto.json) | 6 | trail-dto |
| [cases/compiler/filters.json](cases/compiler/filters.json) | 30 | trail-list |
| [cases/compiler/requests.json](cases/compiler/requests.json) | 7 | list-edit-trail-picker, homepage-global-search, map-global-search, map-list, list-search |
| [cases/compiler/sorting.json](cases/compiler/sorting.json) | 18 | trail-list |
| [cases/mutation/hooks.json](cases/mutation/hooks.json) | 17 | pocketbase-mutation |
| [cases/mutation/startup.json](cases/mutation/startup.json) | 4 | startup-index-initialization |
| [cases/mutation/tokens.json](cases/mutation/tokens.json) | 4 | search-token |
| [cases/projection/actors.json](cases/projection/actors.json) | 3 | projector-actors |
| [cases/projection/lists.json](cases/projection/lists.json) | 5 | projector-lists, projector-remote-lists |
| [cases/projection/trails.json](cases/projection/trails.json) | 13 | projector-trails |
| [cases/search/access.json](cases/search/access.json) | 31 | trail-list, search-proxy |
| [cases/search/consumers.json](cases/search/consumers.json) | 16 | profile-trails, recommendation, list-search, actor-search, upload-duplicate, map-cluster, map-id-details, multi-search-proxy, trail-bounding-box, list-edit-trail-picker |
| [cases/search/dates.json](cases/search/dates.json) | 4 | trail-list |
| [cases/search/errors.json](cases/search/errors.json) | 5 | search-proxy |
| [cases/search/filters.json](cases/search/filters.json) | 21 | trail-list |
| [cases/search/geo.json](cases/search/geo.json) | 3 | trail-list, map-list |
| [cases/search/paging.json](cases/search/paging.json) | 4 | trail-list |
| [cases/search/scale.json](cases/search/scale.json) | 2 | map-cluster, trail-list |
| [cases/search/sorting.json](cases/search/sorting.json) | 17 | trail-list |
| [cases/search/taxonomy-tags.json](cases/search/taxonomy-tags.json) | 13 | trail-list |
| [cases/search/text.json](cases/search/text.json) | 11 | trail-list |
| [cases/state/defaults.json](cases/state/defaults.json) | 6 | list-defaults, map-defaults, trail-list |
| [cases/state/sanitizing.json](cases/state/sanitizing.json) | 5 | filter-sanitizer |

## Fälle je Gruppe

<details>
<summary>cases/browser/navigation.json (11 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-BROWSER-001 | browser-list-default | preserve |
| SRCH0-BROWSER-002 | ignored-url-storage | preserve |
| SRCH0-BROWSER-003 | url-storage-precedence | preserve |
| SRCH0-BROWSER-004 | malformed-storage | preserve |
| SRCH0-BROWSER-005 | duplicate-page-key | preserve |
| SRCH0-BROWSER-006 | separate-page-key | preserve |
| SRCH0-BROWSER-007 | cards-bucket | preserve |
| SRCH0-BROWSER-008 | table-bucket | preserve |
| SRCH0-BROWSER-009 | raw-sort | known_gap |
| SRCH0-BROWSER-010 | snapshot, history, navigation | preserve |
| SRCH0-BROWSER-011 | map, map-list, cluster | preserve |

</details>

<details>
<summary>cases/compiler/api-actors.json (6 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-060 | self-true | preserve |
| SRCH0-COMPILER-061 | self-false | preserve |
| SRCH0-COMPILER-062 | limit-string | preserve |
| SRCH0-COMPILER-063 | actor-rejection | preserve |
| SRCH0-COMPILER-064 | actor-rejection | preserve |
| SRCH0-COMPILER-065 | remote-handle | preserve |

</details>

<details>
<summary>cases/compiler/api-consumers.json (9 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-066 | profile-preferences | preserve |
| SRCH0-COMPILER-067 | profile-replaces-filter | preserve |
| SRCH0-COMPILER-068 | recommend-eligibility | preserve |
| SRCH0-COMPILER-069 | zero-size | preserve |
| SRCH0-COMPILER-070 | anonymous-bounds | preserve |
| SRCH0-COMPILER-071 | empty-bounds | preserve |
| SRCH0-COMPILER-072 | cluster-cap, preferences | preserve |
| SRCH0-COMPILER-073 | invalid-cluster-input | preserve |
| SRCH0-COMPILER-074 | anonymous-fixed-limits | preserve |

</details>

<details>
<summary>cases/compiler/api-proxy.json (6 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-054 | preferences | preserve |
| SRCH0-COMPILER-055 | detail-preference-bypass | preserve |
| SRCH0-COMPILER-056 | free-proxy-input | preserve |
| SRCH0-COMPILER-057 | free-proxy-input | preserve |
| SRCH0-COMPILER-058 | free-proxy-input | preserve |
| SRCH0-COMPILER-059 | preferences, discard-extra-fields | preserve |

</details>

<details>
<summary>cases/compiler/api-upload.json (3 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-083 | duplicate-position-21 | preserve |
| SRCH0-COMPILER-084 | duplicate-found | preserve |
| SRCH0-COMPILER-085 | no-visible-duplicate | preserve |

</details>

<details>
<summary>cases/compiler/dto.json (6 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-075 | unknown-projection-display | known_gap |
| SRCH0-COMPILER-076 | raw-difficulty-display | known_gap |
| SRCH0-COMPILER-077 | raw-difficulty-display | known_gap |
| SRCH0-COMPILER-078 | raw-difficulty-display | known_gap |
| SRCH0-COMPILER-079 | full-dto, relation-expansion | preserve |
| SRCH0-COMPILER-080 | empty-dto | preserve |

</details>

<details>
<summary>cases/compiler/filters.json (30 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-001 | default-filter | preserve |
| SRCH0-COMPILER-002 | raw-query | preserve |
| SRCH0-COMPILER-003 | anonymous-browser-scope | preserve |
| SRCH0-COMPILER-004 | parent | preserve |
| SRCH0-COMPILER-005 | same-parent | preserve |
| SRCH0-COMPILER-006 | different-parent | preserve |
| SRCH0-COMPILER-007 | no-child | preserve |
| SRCH0-COMPILER-008 | unknown-child | preserve |
| SRCH0-COMPILER-009 | tags-or | preserve |
| SRCH0-COMPILER-010 | liked | preserve |
| SRCH0-COMPILER-011 | liked-neutral | preserve |
| SRCH0-COMPILER-012 | completed-true | preserve |
| SRCH0-COMPILER-013 | completed-false | preserve |
| SRCH0-COMPILER-014 | dst, inclusive-day-start | known_gap |
| SRCH0-COMPILER-015 | end-date | known_gap |
| SRCH0-COMPILER-016 | duplicate-geo | known_gap |
| SRCH0-COMPILER-017 | zero-lat | known_gap |
| SRCH0-COMPILER-018 | zero-lon | known_gap |
| SRCH0-COMPILER-019 | difficulty-subset | preserve |
| SRCH0-COMPILER-020 | difficulty-empty | preserve |
| SRCH0-COMPILER-021 | author | preserve |
| SRCH0-COMPILER-022 | visibility | preserve |
| SRCH0-COMPILER-023 | floor-ceil, distance | preserve |
| SRCH0-COMPILER-024 | floor-ceil, elevationGain | preserve |
| SRCH0-COMPILER-025 | floor-ceil, elevationLoss | preserve |
| SRCH0-COMPILER-026 | raw-inverted-range | preserve |
| SRCH0-COMPILER-045 | raw-sort | known_gap |
| SRCH0-COMPILER-046 | pagination | preserve |
| SRCH0-COMPILER-047 | pagination | preserve |
| SRCH0-COMPILER-048 | pagination | preserve |

</details>

<details>
<summary>cases/compiler/requests.json (7 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-049 | retrieval-outside-options | known_gap |
| SRCH0-COMPILER-050 | multi-search, locations-dispatch | preserve |
| SRCH0-COMPILER-051 | multi-search, locations-dispatch | preserve |
| SRCH0-COMPILER-052 | bounds, map-ignores-radius | preserve |
| SRCH0-COMPILER-053 | antimeridian, map-ignores-radius | preserve |
| SRCH0-COMPILER-081 | list-index | preserve |
| SRCH0-COMPILER-082 | list-index | preserve |

</details>

<details>
<summary>cases/compiler/sorting.json (18 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-COMPILER-027 | sort, name, + | preserve |
| SRCH0-COMPILER-028 | sort, name, - | preserve |
| SRCH0-COMPILER-029 | sort, distance, + | preserve |
| SRCH0-COMPILER-030 | sort, distance, - | preserve |
| SRCH0-COMPILER-031 | sort, duration, + | preserve |
| SRCH0-COMPILER-032 | sort, duration, - | preserve |
| SRCH0-COMPILER-033 | sort, difficulty, + | preserve |
| SRCH0-COMPILER-034 | sort, difficulty, - | preserve |
| SRCH0-COMPILER-035 | sort, elevation_gain, + | preserve |
| SRCH0-COMPILER-036 | sort, elevation_gain, - | preserve |
| SRCH0-COMPILER-037 | sort, elevation_loss, + | preserve |
| SRCH0-COMPILER-038 | sort, elevation_loss, - | preserve |
| SRCH0-COMPILER-039 | sort, like_count, + | preserve |
| SRCH0-COMPILER-040 | sort, like_count, - | preserve |
| SRCH0-COMPILER-041 | sort, created, + | preserve |
| SRCH0-COMPILER-042 | sort, created, - | preserve |
| SRCH0-COMPILER-043 | sort, date, + | preserve |
| SRCH0-COMPILER-044 | sort, date, - | preserve |

</details>

<details>
<summary>cases/mutation/hooks.json (17 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-MUTATION-001 | core-update, visibility, difficulty, defensive-source | non_contract |
| SRCH0-MUTATION-002 | create | preserve |
| SRCH0-MUTATION-003 | delete | preserve |
| SRCH0-MUTATION-004 | likes, relation-removal | preserve |
| SRCH0-MUTATION-005 | shares, revoked | known_gap |
| SRCH0-MUTATION-006 | shares | preserve |
| SRCH0-MUTATION-007 | actor, denormalization | known_gap |
| SRCH0-MUTATION-008 | tags, denormalization | known_gap |
| SRCH0-MUTATION-009 | taxonomy, denormalization | known_gap |
| SRCH0-MUTATION-010 | taxonomy, denormalization | known_gap |
| SRCH0-MUTATION-011 | actor-create-delete | preserve |
| SRCH0-MUTATION-012 | list-share | preserve |
| SRCH0-MUTATION-013 | list-create | preserve |
| SRCH0-MUTATION-014 | list-update, partial-update | preserve |
| SRCH0-MUTATION-015 | list-delete | preserve |
| SRCH0-MUTATION-016 | taxonomy, tags | preserve |
| SRCH0-MUTATION-017 | core-update, visibility, difficulty | preserve |

</details>

<details>
<summary>cases/mutation/startup.json (4 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-MUTATION-090 | startup, request-availability, startup-api | known_gap |
| SRCH0-MUTATION-091 | startup, request-availability, startup-api | known_gap |
| SRCH0-MUTATION-092 | startup, request-availability, startup-api | known_gap |
| SRCH0-MUTATION-093 | startup, request-availability, startup-api | known_gap |

</details>

<details>
<summary>cases/mutation/tokens.json (4 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-MUTATION-080 | tenant-scope | preserve |
| SRCH0-MUTATION-081 | tenant-scope | preserve |
| SRCH0-MUTATION-082 | tenant-scope | preserve |
| SRCH0-MUTATION-083 | tenant-scope | preserve |

</details>

<details>
<summary>cases/projection/actors.json (3 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-PROJECTION-018 | actor | preserve |
| SRCH0-PROJECTION-019 | actor | preserve |
| SRCH0-PROJECTION-020 | actor | preserve |

</details>

<details>
<summary>cases/projection/lists.json (5 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-PROJECTION-013 | relations | preserve |
| SRCH0-PROJECTION-014 | relations | preserve |
| SRCH0-PROJECTION-015 | remote-live-aggregate | known_gap |
| SRCH0-PROJECTION-016 | remote-live-aggregate | known_gap |
| SRCH0-PROJECTION-017 | remote-live-aggregate | known_gap |

</details>

<details>
<summary>cases/projection/trails.json (13 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-PROJECTION-001 | relations | preserve |
| SRCH0-PROJECTION-002 | relations | preserve |
| SRCH0-PROJECTION-003 | relations | preserve |
| SRCH0-PROJECTION-004 | relations | preserve |
| SRCH0-PROJECTION-005 | relations | preserve |
| SRCH0-PROJECTION-006 | difficulty | preserve |
| SRCH0-PROJECTION-007 | difficulty | preserve |
| SRCH0-PROJECTION-008 | difficulty | known_gap |
| SRCH0-PROJECTION-009 | difficulty, defensive-source | non_contract |
| SRCH0-PROJECTION-010 | difficulty | known_gap |
| SRCH0-PROJECTION-011 | partial-update | preserve |
| SRCH0-PROJECTION-012 | thumbnail | preserve |
| SRCH0-PROJECTION-021 | thumbnail, validated-source, projection-panic | known_gap |

</details>

<details>
<summary>cases/search/access.json (31 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-011 | access, access-public-alpine | preserve |
| SRCH0-SEARCH-012 | access, access-private-alice | preserve |
| SRCH0-SEARCH-013 | access, access-private-bob | preserve |
| SRCH0-SEARCH-014 | access, access-shared-alice | preserve |
| SRCH0-SEARCH-015 | access, access-revoked-alice | preserve |
| SRCH0-SEARCH-016 | access, access-remote-public | preserve |
| SRCH0-SEARCH-017 | access, access-remote-private | preserve |
| SRCH0-SEARCH-018 | access, access-remote-shared | preserve |
| SRCH0-SEARCH-019 | access, access-unverified-public | preserve |
| SRCH0-SEARCH-020 | access, access-deleted-trail | preserve |
| SRCH0-SEARCH-021 | access, access-public-alpine | preserve |
| SRCH0-SEARCH-022 | access, access-private-alice | preserve |
| SRCH0-SEARCH-023 | access, access-private-bob | preserve |
| SRCH0-SEARCH-024 | access, access-shared-alice | preserve |
| SRCH0-SEARCH-025 | access, access-revoked-alice | preserve |
| SRCH0-SEARCH-026 | access, access-remote-public | preserve |
| SRCH0-SEARCH-027 | access, access-remote-private | preserve |
| SRCH0-SEARCH-028 | access, access-remote-shared | preserve |
| SRCH0-SEARCH-029 | access, access-unverified-public | preserve |
| SRCH0-SEARCH-030 | access, access-deleted-trail | preserve |
| SRCH0-SEARCH-031 | access, access-public-alpine | preserve |
| SRCH0-SEARCH-032 | access, access-private-alice | preserve |
| SRCH0-SEARCH-033 | access, access-private-bob | preserve |
| SRCH0-SEARCH-034 | access, access-shared-alice | preserve |
| SRCH0-SEARCH-035 | access, access-revoked-alice | preserve |
| SRCH0-SEARCH-036 | access, access-remote-public | preserve |
| SRCH0-SEARCH-037 | access, access-remote-private | preserve |
| SRCH0-SEARCH-038 | access, access-remote-shared | preserve |
| SRCH0-SEARCH-039 | access, access-unverified-public | preserve |
| SRCH0-SEARCH-040 | access, access-deleted-trail | preserve |
| SRCH0-SEARCH-041 | link-share-search-scope | preserve |

</details>

<details>
<summary>cases/search/consumers.json (16 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-108 | profile-author | preserve |
| SRCH0-SEARCH-109 | recommendation-eligibility | non_contract |
| SRCH0-SEARCH-110 | lists-tenant | preserve |
| SRCH0-SEARCH-111 | lists-anonymous | preserve |
| SRCH0-SEARCH-112 | actor-self-included | preserve |
| SRCH0-SEARCH-113 | actor-self-excluded | preserve |
| SRCH0-SEARCH-114 | actor-local-lookup | preserve |
| SRCH0-SEARCH-115 | actor-other-lookup | preserve |
| SRCH0-SEARCH-116 | actor-anonymous-tenant | preserve |
| SRCH0-SEARCH-117 | duplicate-limit-20, duplicate-position-21 | known_gap |
| SRCH0-SEARCH-118 | duplicate-tenant | preserve |
| SRCH0-SEARCH-119 | cluster-summary | preserve |
| SRCH0-SEARCH-120 | multi-details-tenant | preserve |
| SRCH0-SEARCH-121 | multi-search | preserve |
| SRCH0-SEARCH-122 | bounding-box-extrema | preserve |
| SRCH0-SEARCH-123 | generic-retrieval, SRCH0-GAP-DTO-001 | known_gap |

</details>

<details>
<summary>cases/search/dates.json (4 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-070 | date-start | preserve |
| SRCH0-SEARCH-071 | date-end-midnight, SRCH0-GAP-DATE-001 | known_gap |
| SRCH0-SEARCH-072 | date-end-of-day | preserve |
| SRCH0-SEARCH-073 | date-dst | preserve |

</details>

<details>
<summary>cases/search/errors.json (5 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-124 | negative-unknown-index | non_contract |
| SRCH0-SEARCH-125 | negative-unknown-filter | non_contract |
| SRCH0-SEARCH-126 | negative-unknown-sort | non_contract |
| SRCH0-SEARCH-127 | negative-unknown-direction | non_contract |
| SRCH0-SEARCH-128 | negative-free-retrieval | non_contract |

</details>

<details>
<summary>cases/search/filters.json (21 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-055 | liked-neutral | preserve |
| SRCH0-SEARCH-056 | liked-active | preserve |
| SRCH0-SEARCH-057 | completed-true | preserve |
| SRCH0-SEARCH-058 | completed-false | preserve |
| SRCH0-SEARCH-059 | completed-omitted, difficulty-default, sort-created-asc, sort-ties | preserve |
| SRCH0-SEARCH-060 | range-distance, range-floor-ceil | preserve |
| SRCH0-SEARCH-061 | range-distance-open-max | preserve |
| SRCH0-SEARCH-062 | range-distance-reversed | preserve |
| SRCH0-SEARCH-063 | range-elevation_gain, range-floor-ceil | preserve |
| SRCH0-SEARCH-064 | range-elevation_gain-open-max | preserve |
| SRCH0-SEARCH-065 | range-elevation_gain-reversed | preserve |
| SRCH0-SEARCH-066 | range-elevation_loss, range-floor-ceil | preserve |
| SRCH0-SEARCH-067 | range-elevation_loss-open-max | preserve |
| SRCH0-SEARCH-068 | range-elevation_loss-reversed | preserve |
| SRCH0-SEARCH-069 | range-combined | preserve |
| SRCH0-SEARCH-074 | difficulty-0 | preserve |
| SRCH0-SEARCH-075 | difficulty-1 | preserve |
| SRCH0-SEARCH-076 | difficulty-2 | preserve |
| SRCH0-SEARCH-077 | difficulty-unknown-collapse, SRCH0-GAP-DIFF-001 | known_gap |
| SRCH0-SEARCH-079 | difficulty-subset | preserve |
| SRCH0-SEARCH-080 | difficulty-empty-selection | preserve |

</details>

<details>
<summary>cases/search/geo.json (3 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-081 | geo-radius, SRCH0-GAP-GEO-001 | known_gap |
| SRCH0-SEARCH-084 | geo-bounds, map-default | preserve |
| SRCH0-SEARCH-085 | geo-antimeridian | preserve |

</details>

<details>
<summary>cases/search/paging.json (4 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-104 | page-1, pagination | preserve |
| SRCH0-SEARCH-105 | page-2, pagination | preserve |
| SRCH0-SEARCH-106 | page-3, pagination | preserve |
| SRCH0-SEARCH-107 | page-99, pagination | preserve |

</details>

<details>
<summary>cases/search/scale.json (2 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-129 | scale-10001, cluster-limit | preserve |
| SRCH0-SEARCH-130 | scale-pagination-limit | preserve |

</details>

<details>
<summary>cases/search/sorting.json (17 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-086 | sort-name-asc, sort-ties | preserve |
| SRCH0-SEARCH-087 | sort-name-desc, sort-ties | preserve |
| SRCH0-SEARCH-088 | sort-distance-asc, sort-ties | preserve |
| SRCH0-SEARCH-089 | sort-distance-desc, sort-ties | preserve |
| SRCH0-SEARCH-090 | sort-duration-asc, sort-ties | preserve |
| SRCH0-SEARCH-091 | sort-duration-desc, sort-ties | preserve |
| SRCH0-SEARCH-092 | sort-difficulty-asc, sort-ties | preserve |
| SRCH0-SEARCH-093 | sort-difficulty-desc, sort-ties | preserve |
| SRCH0-SEARCH-094 | sort-elevation_gain-asc, sort-ties | preserve |
| SRCH0-SEARCH-095 | sort-elevation_gain-desc, sort-ties | preserve |
| SRCH0-SEARCH-096 | sort-elevation_loss-asc, sort-ties | preserve |
| SRCH0-SEARCH-097 | sort-elevation_loss-desc, sort-ties | preserve |
| SRCH0-SEARCH-098 | sort-like_count-asc, sort-ties | preserve |
| SRCH0-SEARCH-099 | sort-like_count-desc, sort-ties | preserve |
| SRCH0-SEARCH-101 | sort-created-desc, sort-ties | preserve |
| SRCH0-SEARCH-102 | sort-date-asc, sort-ties | preserve |
| SRCH0-SEARCH-103 | sort-date-desc, sort-ties | preserve |

</details>

<details>
<summary>cases/search/taxonomy-tags.json (13 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-042 | taxonomy-parent | preserve |
| SRCH0-SEARCH-043 | taxonomy-child | preserve |
| SRCH0-SEARCH-044 | taxonomy-same-parent | preserve |
| SRCH0-SEARCH-045 | taxonomy-other-parent | preserve |
| SRCH0-SEARCH-046 | taxonomy-no-child | preserve |
| SRCH0-SEARCH-047 | taxonomy-unknown | preserve |
| SRCH0-SEARCH-048 | taxonomy-preference | preserve |
| SRCH0-SEARCH-049 | taxonomy-child-preference | preserve |
| SRCH0-SEARCH-050 | taxonomy-detail-bypass | preserve |
| SRCH0-SEARCH-051 | tags-one | preserve |
| SRCH0-SEARCH-052 | tags-multiple | preserve |
| SRCH0-SEARCH-053 | tags-and-group | preserve |
| SRCH0-SEARCH-054 | tags-unknown | preserve |

</details>

<details>
<summary>cases/search/text.json (11 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-SEARCH-001 | text-empty, list-default | preserve |
| SRCH0-SEARCH-002 | text-author_name | preserve |
| SRCH0-SEARCH-003 | text-name | preserve |
| SRCH0-SEARCH-004 | text-description | preserve |
| SRCH0-SEARCH-005 | text-location | preserve |
| SRCH0-SEARCH-006 | text-tags | preserve |
| SRCH0-SEARCH-007 | text-multiple | preserve |
| SRCH0-SEARCH-008 | text-empty-result | preserve |
| SRCH0-SEARCH-009 | text-ranking-tie | preserve |
| SRCH0-SEARCH-010 | text-with-visible-sort | preserve |
| SRCH0-SEARCH-131 | text-relevance-before-sort, ranking-sort-conflict | preserve |

</details>

<details>
<summary>cases/state/defaults.json (6 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-STATE-001 | defaults | preserve |
| SRCH0-STATE-002 | defaults | preserve |
| SRCH0-STATE-003 | ignored-query, duplicate-page | preserve |
| SRCH0-STATE-004 | url-normalization, first-query-key | preserve |
| SRCH0-STATE-005 | unknown-taxonomy | preserve |
| SRCH0-STATE-006 | asymmetric-limits | known_gap |

</details>

<details>
<summary>cases/state/sanitizing.json (5 Fälle)</summary>

| Case-ID | Prüfmerkmal | Einordnung |
| --- | --- | --- |
| SRCH0-STATE-007 | wrong-types | preserve |
| SRCH0-STATE-008 | min-greater-max, clamp | preserve |
| SRCH0-STATE-009 | legacy-difficulty, zero-coordinates | preserve |
| SRCH0-STATE-010 | null-storage | preserve |
| SRCH0-STATE-011 | date-preservation | preserve |

</details>

## Inventar und Bezugsdaten

[inventory.json](inventory.json) enthält die unabhängig gepflegten Consumer und Abdeckungspflichten. Gruppen und Fälle werden dagegen geprüft.

- [datasets/reference.json](datasets/reference.json), Revision 1
- [datasets/scale-10001.json](datasets/scale-10001.json), Revision 1
- [profiles/legacy-actor-v0.json](profiles/legacy-actor-v0.json), Revision 1
- [profiles/lists-legacy-v0.json](profiles/lists-legacy-v0.json), Revision 1
- [profiles/trails-legacy-v0.json](profiles/trails-legacy-v0.json), Revision 1
