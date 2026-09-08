import type { RequestEvent } from '@sveltejs/kit';
import type { Principal, JsonRecord } from './types';

export function principal(name: Principal) {
    return name === 'anonymous' ? undefined : { id: `user${name}`, actor: name };
}

interface ApiContext {
    principal: Principal;
    ms: unknown;
    preferences?: Record<string, unknown[]>;
    params?: Record<string, string>;
    url?: string;
    body?: unknown;
    settings?: JsonRecord;
    filterValues?: JsonRecord;
    fetch?: typeof fetch;
}

export function apiEvent(context: ApiContext): RequestEvent {
    const user = principal(context.principal);
    const preferences = context.preferences ?? {};
    const records = {
        user_category_preferences: preferences.user_category_preferences ?? preferences.hidden_categories?.map(category => ({ category, visible: false })) ?? [],
        user_subcategory_preferences: preferences.user_subcategory_preferences ?? preferences.hidden_subcategories?.map(subcategory => ({ subcategory, visible: false })) ?? [],
    };
    const pb = {
        authStore: { record: user }, filter: () => '',
        collection: (name: keyof typeof records) => ({ getFullList: async () => records[name], getOne: async () => context.filterValues }),
    };
    // Gemeinsame externe Testgrenze: Router, SDK und Browseradapter lesen
    // verschiedene Ausschnitte des grossen SvelteKit-RequestEvent-Typs.
    return {
        params: context.params ?? { index: 'trails' },
        url: new URL(context.url ?? '/api/v1/search/trails', 'http://srch0.invalid'),
        request: new Request('http://srch0.invalid', { method: 'POST', body: JSON.stringify(context.body ?? {}) }),
        locals: { ms: context.ms, pb, user, settings: context.settings ?? {} },
        fetch: context.fetch,
    } as unknown as RequestEvent;
}

export function routeError(error: unknown) {
    const response = error as { status?: number; body?: { code?: string; cause?: { code?: string } } };
    return { status: response.status ?? 500, category: response.body?.code ?? response.body?.cause?.code };
}
