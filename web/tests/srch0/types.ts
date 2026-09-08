import type { Category } from '$lib/models/category';
import type { Subcategory } from '$lib/models/subcategory';
import type { Trail, TrailFilter, TrailSearchResult } from '$lib/models/trail';
import type { ListFilter } from '$lib/models/list';
import type { ActorSearchResult } from '$lib/models/activitypub/actor';
import type { ListSearchResult } from '$lib/stores/search_store';
import type { MultiSearchParams, SearchParams } from 'meilisearch';

export type JsonRecord = Record<string, unknown>;
export type Principal = 'anonymous' | 'alice' | 'bob';
export interface Context {
    principal: Principal;
    timezone: string;
    locale: string;
    surface: string;
    consumer: string;
    preferences: Record<string, unknown[]>;
}
export interface Fixture<Input = JsonRecord, Observation = JsonRecord> {
    case_id: string;
    family: string;
    baseline: { engine_profiles: string[] };
    dataset_ref: string;
    context: Context;
    input: Input;
    observed: Observation;
    baseline_observed: Observation;
    digest: string;
    path: string;
}
export interface Dataset {
    principals: Record<Principal, { id?: string; user_id?: string }>;
    categories: Category[];
    subcategories: Subcategory[];
    trails: TrailSearchResult[];
    lists: ListSearchResult[];
    actors: ActorSearchResult[];
}

export interface FilterInput {
    url?: string;
    limits?: { max_distance: number; max_elevation_gain: number; max_elevation_loss: number };
    // Ungültige Legacy-Sortwerte müssen als rohe Eingaben testbar bleiben.
    filter?: Omit<Partial<TrailFilter>, 'sort' | 'sortOrder'> & { sort?: string; sortOrder?: string };
}
export type StateInput = FilterInput & (
    { adapter: 'route-load' } | { adapter: 'sanitize'; candidate: unknown }
);
export interface DTOInput {
    adapter: 'trail-dto';
    ids: string[];
    hits?: TrailSearchResult[];
    fields?: (keyof Trail)[];
    hit_overrides?: Record<string, JsonRecord>;
    omit_fields?: string[];
}
interface Paging { page?: number; per_page?: number }
export type CompilerInput =
    | (FilterInput & Paging & { adapter: 'list-search' })
    | (FilterInput & Paging & {
        adapter: 'map-search'; north_east: { lat: number; lng: number }; south_west: { lat: number; lng: number };
        zoom?: number; load_map_data?: boolean; cluster_response?: JsonRecord;
    })
    | { adapter: 'generic-search'; q: string; options: SearchParams }
    | { adapter: 'global-multi'; options: MultiSearchParams }
    | (Paging & { adapter: 'list-index-search'; filter: ListFilter });

export type ApiAdapter = 'api-proxy' | 'api-multi' | 'api-actor' | 'api-profile-trails' | 'api-profile-lists'
    | 'api-recommendation' | 'api-bounds' | 'api-cluster' | 'api-filter-values' | 'api-upload';
export interface ApiInput {
    adapter: ApiAdapter;
    url?: string;
    params?: Record<string, string>;
    body?: JsonRecord;
    responses?: JsonRecord[];
    random?: number;
    actor?: JsonRecord;
    filter_values?: JsonRecord;
    settings?: JsonRecord;
    remote_response?: JsonRecord;
    search_ids?: string[];
    trail?: JsonRecord;
}
export type WebInput = StateInput | DTOInput | CompilerInput | ApiInput;
export interface BrowserInput { adapter: 'browser'; url: string; storage: Record<string, string>; navigation?: 'history-back' }
export interface BrowserObservation {
    browser_state: { url?: string; url_path?: string; query_keys?: string[]; sort?: string; filter?: TrailFilter; page_size?: number };
}
export interface RecordedRequest { url: string; body: unknown }
