import { normalizeSearchLayout, type SearchLayout } from "$lib/models/search_layout";
import { writable } from "svelte/store";

const storageKey = "searchLayout";
export const searchLayout = writable(normalizeSearchLayout(undefined));
let initialized = false;

export function loadSearchLayout() {
    if (initialized || typeof window === "undefined") return;
    initialized = true;
    try {
        searchLayout.set(normalizeSearchLayout(JSON.parse(localStorage.getItem(storageKey) ?? "null")));
    } catch {
        searchLayout.set(normalizeSearchLayout(undefined));
    }
}

export function updateSearchLayout(change: Partial<SearchLayout>) {
    searchLayout.update(current => {
        const next = normalizeSearchLayout({ ...current, ...change });
        try { localStorage.setItem(storageKey, JSON.stringify(next)); } catch { /* Keep the current session usable without storage. */ }
        return next;
    });
}
