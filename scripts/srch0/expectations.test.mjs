import test from 'node:test';
import assert from 'node:assert/strict';
import { resolveObservation } from './expectations.mjs';
import { loadCorpus } from './corpus.mjs';

test('eine Ersatzbeobachtung erlaubt die Radiuskorrektur und erhält die historische Basis', () => {
  const { entry, fixture } = loadCorpus().cases.find(({ fixture }) => fixture.case_id === 'SRCH0-COMPILER-016');
  const observed = structuredClone(fixture.observed);
  const options = observed.engine_requests[0].body.options;
  const clauses = options.filter.match(/_geoRadius\([^)]*\)/g);
  assert.equal(clauses.length, 2);
  assert.equal(clauses[0], clauses[1]);
  options.filter = options.filter.replace(` AND ${clauses[0]}`, '');
  const change = { case_id: fixture.case_id, basis_digest: entry.sha256, observed };
  const resolved = resolveObservation(fixture, entry.sha256, [change]);
  assert.equal(resolved.engine_requests[0].body.options.filter.match(/_geoRadius\(/g).length, 1);
  assert.equal(fixture.observed.engine_requests[0].body.options.filter.match(/_geoRadius\(/g).length, 2);
  resolved.engine_requests.length = 0;
  assert.equal(observed.engine_requests.length, 1);
  assert.throws(() => resolveObservation(fixture, 'wrong-base', [change]), /Basisdigest/);
  assert.throws(() => resolveObservation(fixture, entry.sha256, [change, change]), /mehrere aktive/);
});

test('vollständiger Ersatz kann alte Felder entfernen; eine fehlende Beobachtung wird abgewiesen', () => {
  const fixture = { case_id: 'SRCH0-MUTATION-090', observed: { available: true, old_field: 'legacy' } };
  const change = { case_id: fixture.case_id, basis_digest: 'test-basis', observed: { available: false } };
  assert.deepEqual(resolveObservation(fixture, 'test-basis', [change]), { available: false });
  assert.deepEqual(fixture.observed, { available: true, old_field: 'legacy' });
  assert.throws(() => resolveObservation(fixture, 'test-basis', [{ ...change, observed: null }]), /Objekt/);
});
