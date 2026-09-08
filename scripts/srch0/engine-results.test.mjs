import test from 'node:test';
import assert from 'node:assert/strict';
import { join } from 'node:path';
import { corpusRoot, readJSON } from './corpus.mjs';
import { assertSearchPlausibility, normalizeResult } from './engine-results.mjs';

const dataset = readJSON(join(corpusRoot, 'datasets/reference.json'));
const trail = id => dataset.trails.find(document => document.id === id);
const response = hits => ({ hits, estimatedTotalHits: hits.length, offset: 0, limit: 20 });

test('an arbitrary unsorted candidate is not an identity golden', () => {
    const first = response([trail('duplicate-01')]);
    const other = response([trail('duplicate-21')]);
    const order = { eligibility_only: true };
    assert.deepEqual(normalizeResult(first, order), normalizeResult(other, order));
    assertSearchPlausibility(other, { q: '' }, dataset, 'alice', 'trails');
});

test('equal sort values may exchange positions without changing the observation', () => {
    const hits = [trail('tie-a'), trail('tie-b')];
    const order = { tie_fields: ['distance'] };
    assert.deepEqual(normalizeResult(response(hits), order), normalizeResult(response([...hits].reverse()), order));
});

test('the full DTO digest detects field changes without a separate field list', () => {
    const original = normalizeResult(response([{ id: 'one', name: 'Trail' }]));
    const renamed = normalizeResult(response([{ id: 'one', title: 'Trail' }]));
    assert.equal(Object.hasOwn(original, 'document_fields'), false);
    assert.notEqual(original.documents_sha256, renamed.documents_sha256);
    const eligibility = normalizeResult(response([{ id: 'one', name: 'Trail' }]), { eligibility_only: true });
    assert.deepEqual(eligibility.document_fields, ['id', 'name']);
});

test('a plausible snapshot cannot authorize an invisible or fabricated hit', () => {
    assert.throws(() => assertSearchPlausibility(response([trail('private-bob')]), { q: '' }, dataset, 'anonymous', 'trails'), /visible dataset/);
    const fabricated = { ...trail('public-alpine'), distance: -123 };
    assert.throws(() => assertSearchPlausibility(response([fabricated]), { q: '' }, dataset, 'anonymous', 'trails'), /retrieved value/);
});

test('counts, uniqueness and numeric order are checked independently of goldens', () => {
    const hits = [trail('range-above'), trail('range-below')];
    assert.throws(() => assertSearchPlausibility(response(hits), { q: '', sort: ['distance:asc'] }, dataset, 'anonymous', 'trails'), /monotone/);
    assert.throws(() => assertSearchPlausibility(response([hits[0], hits[0]]), { q: '' }, dataset, 'anonymous', 'trails'), /repeat a document/);
    assert.throws(() => assertSearchPlausibility({ ...response(hits), estimatedTotalHits: 1 }, { q: '' }, dataset, 'anonymous', 'trails'), /Total must cover/);
});

test('source plausibility tolerates a floating-point roundtrip, not changed measurements', () => {
    const list = { ...dataset.lists.find(document => document.id === 'list-local'), distance: 20.49 };
    assertSearchPlausibility(response([list]), { q: '' }, dataset, 'anonymous', 'lists');
    assert.throws(() => assertSearchPlausibility(response([{ ...list, distance: 20.5 }]), { q: '' }, dataset, 'anonymous', 'lists'), /retrieved value/);
});
