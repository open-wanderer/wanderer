import { vi } from 'vitest';
import { currentUser } from '$lib/stores/user_store';
import { categories } from '$lib/stores/category_store';
import { subcategories } from '$lib/stores/subcategory_store';
import { observeState } from './state-adapter';
import { observeCompiler, observeDTO } from './compiler-adapter';
import { observeAPI } from './api-adapter';
import { principal } from './api-context';
import type { Dataset, Fixture, WebInput } from './types';

export function resetAdapters() {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
    currentUser.set(null);
}

export async function observeWeb(fixture: Fixture<WebInput>, source: Dataset) {
    const { input, context } = fixture;
    vi.stubEnv('TZ', context.timezone);
    currentUser.set(principal(context.principal) as unknown as Parameters<typeof currentUser.set>[0]);
    categories.set(source.categories);
    subcategories.set(source.subcategories);

    switch (input.adapter) {
        case 'route-load': case 'sanitize': return observeState(context, input, source);
        case 'trail-dto': return observeDTO(input, source);
        case 'list-search': case 'map-search': case 'generic-search': case 'global-multi': case 'list-index-search':
            return observeCompiler(context, input, source);
        case 'api-proxy': case 'api-multi': case 'api-actor': case 'api-profile-trails': case 'api-profile-lists':
        case 'api-recommendation': case 'api-bounds': case 'api-cluster': case 'api-filter-values': case 'api-upload':
            return observeAPI(context, input, source);
        default: throw new Error(`Unbekannter Webadapter: ${JSON.stringify(input)}`);
    }
}
