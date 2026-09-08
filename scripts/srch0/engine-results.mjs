import assert from 'node:assert/strict';
import { canonical, sha256 } from './corpus.mjs';

export const digest = value => sha256(typeof value === 'string' ? value : canonical(value));

/** Keep totals and DTO shape even when an unsorted, truncated selection is not a contract. */
export function normalizeResult(body, order = {}) {
    const hits = body.hits;
    const result = {
        total: body.totalHits ?? body.estimatedTotalHits,
        page: body.page ?? null,
        total_pages: body.totalPages ?? null,
        hits_per_page: body.hitsPerPage ?? null,
        offset: body.offset ?? null,
        limit: body.limit ?? null,
        returned: hits.length,
    };
    if (order.eligibility_only) {
        result.document_fields = [...new Set(hits.flatMap(hit => Object.keys(hit)))].sort();
        return result;
    }

    const hasIDs = hits.every(hit => typeof hit.id === 'string');
    const groups = [];
    for (const hit of hasIDs ? hits : []) {
        const key = order.tie_fields?.length
            ? canonical(order.tie_fields.map(field => hit[field]))
            : order.exact ? hit.id : 'unordered';
        if (groups.at(-1)?.key === key) groups.at(-1).ids.push(hit.id);
        else groups.push({ key, ids: [hit.id] });
    }
    const documents = [...hits].sort((a, b) => hasIDs
        ? a.id.localeCompare(b.id)
        : canonical(a).localeCompare(canonical(b)));
    result.hit_groups = groups.map(group => group.ids.sort());
    result.documents_sha256 = digest(documents);
    if (!hasIDs) result.documents = documents;
    return result;
}

// This predicate expresses fixture relationships, independently of the tenant
// token's Meilisearch expression. It catches a broad or incorrectly signed scope.
export function eligibleDocuments(dataset, index, principal) {
    assert.ok(Object.hasOwn(dataset.principals, principal), `Unknown dataset principal ${principal}`);
    const actor = dataset.principals[principal].id;
    const documents = dataset[index];
    assert.ok(Array.isArray(documents), `Unknown dataset index ${index}`);
    if (index === 'actors') return actor ? documents : [];
    return documents.filter(document => document.public === true
        || (actor && (document.author === actor || document.shares?.includes(actor))));
}

function assertSourceValue(actual, source, path) {
    if (typeof actual === 'number' && typeof source === 'number' && !Number.isInteger(source)) {
        // Meilisearch's numeric roundtrip can differ by one representable float
        // from a JavaScript aggregate (20.49 vs 20.490000000000002). The separate
        // DTO golden remains exact; source plausibility tolerates only this scale.
        const tolerance = Number.EPSILON * Math.max(1, Math.abs(actual), Math.abs(source));
        assert.ok(Math.abs(actual - source) <= tolerance, `${path}: retrieved value must match its source document`);
    } else if (actual && source && typeof actual === 'object' && typeof source === 'object') {
        assert.equal(Array.isArray(actual), Array.isArray(source), `${path}: source container type`);
        assert.deepEqual(Object.keys(actual).sort(), Object.keys(source).sort(), `${path}: source field shape`);
        for (const field of Object.keys(actual)) assertSourceValue(actual[field], source[field], `${path}.${field}`);
    } else {
        assert.deepEqual(actual, source, `${path}: retrieved value must match its source document`);
    }
}

/** Assertions against source facts; no observed snapshot is read here. */
export function assertSearchPlausibility(body, request, dataset, principal, index) {
    assert.ok(Array.isArray(body.hits), 'Search must return an array of hits');
    const eligible = eligibleDocuments(dataset, index, principal);
    const eligibleByID = new Map(eligible.map(document => [document.id, document]));
    const total = body.totalHits ?? body.estimatedTotalHits;
    assert.ok(Number.isSafeInteger(total) && total >= body.hits.length, 'Total must cover returned hits');
    assert.ok(total <= eligible.length, 'Total must not exceed the tenant search universe');
    const capacity = body.hitsPerPage ?? body.limit;
    assert.ok(Number.isSafeInteger(capacity) && body.hits.length <= capacity, 'Page must respect its requested capacity');

    const ids = body.hits.flatMap(hit => typeof hit.id === 'string' ? [hit.id] : []);
    assert.equal(new Set(ids).size, ids.length, 'A page must not repeat a document');
    for (const hit of body.hits) {
        if (hit.id === undefined) continue; // The bounding-box consumer requests only coordinates.
        const source = eligibleByID.get(hit.id);
        assert.ok(source, `${index}/${hit.id} must belong to principal ${principal}'s visible dataset`);
        for (const [field, value] of Object.entries(hit)) {
            assertSourceValue(value, source[field], `${index}/${hit.id}.${field}`);
        }
    }

    // Numeric ordering is unambiguous only for an empty query. With text, the
    // baseline places relevance before the explicit sort rule.
    if (!request.q && request.sort?.length === 1) {
        const [field, direction] = request.sort[0].split(':');
        const values = body.hits.map(hit => hit[field]);
        if (values.every(value => typeof value === 'number')) {
            for (let i = 1; i < values.length; i++) {
                assert.ok(direction === 'asc' ? values[i] >= values[i - 1] : values[i] <= values[i - 1], `${field}:${direction} must be monotone for an empty query`);
            }
        }
    }
}
