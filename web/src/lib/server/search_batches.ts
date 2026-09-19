import type { Meilisearch, SearchParams } from 'meilisearch';

/** Read all matching documents through the caller's tenant-scoped search API. */
export async function* searchBatches<T extends { id: string }>(
    index: ReturnType<Meilisearch['index']>,
    q: string,
    options: Pick<SearchParams, 'filter' | 'attributesToRetrieve'> = {},
): AsyncGenerator<T[]> {
    const seen = new Set<string>();
    const filter = options.filter ? (Array.isArray(options.filter) ? options.filter : [options.filter]) : [];
    const fields = options.attributesToRetrieve;
    while (true) {
        // Offset pagination stops at maxTotalHits. Excluding the documents
        // already consumed opens a new search window without widening ACLs.
        const response = await index.search<T>(q, {
            ...options,
            ...(fields ? { attributesToRetrieve: [...new Set([...fields, 'id'])] } : {}),
            filter: [...filter, ...(seen.size ? ['id NOT IN ' + JSON.stringify([...seen])] : [])],
            offset: 0,
            limit: 500,
        });
        if (!response.hits.length) return;
        for (const hit of response.hits) {
            if (typeof hit.id !== 'string' || seen.has(hit.id)) throw new Error('Search did not advance to new document IDs');
            seen.add(hit.id);
        }
        yield response.hits;
        // A short page can still be capped by the engine's configuration.
        // Only an empty remaining set proves that the traversal is complete.
    }
}
