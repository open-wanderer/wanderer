import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import { loadCorpus, corpusRoot, repoRoot, readJSON } from './corpus.mjs';
import { validateInventory } from './inventory.mjs';

const inventory = readJSON(path.join(corpusRoot, 'inventory.json'));
const fixtures = () => loadCorpus().cases.map(item => item.fixture);

test('ein unabhängig ergänzter Consumer verlangt einen ausführbaren Fall', () => {
  const next = structuredClone(inventory);
  next.consumers.push({ id: 'new-surface', description: 'Neu gefundene Suchoberfläche', source: 'web/src/lib/stores/trail_store.ts' });
  assert.ok(validateInventory(next, fixtures(), repoRoot).includes('Consumer ohne Fall: new-surface'));
});

test('generische Randfall-Tags können fehlende DST-Abdeckung nicht ersetzen', () => {
  const remaining = fixtures().filter(fixture => !fixture.evidence.coverage.includes('date-dst'));
  assert.ok(remaining.some(fixture => fixture.family === 'search' && fixture.evidence.coverage.includes('boundary')));
  assert.ok(validateInventory(inventory, remaining, repoRoot).includes('Pflichtmerkmal ohne Fall: search.date-dst'));
});

test('die Access-Matrix prüft die tatsächliche Anfrage trotz unveränderter Tags', () => {
  const changed = fixtures();
  for (const fixture of changed) {
    if (fixture.family === 'search' && fixture.context.principal === 'anonymous' && fixture.input.request?.filter === "id = 'private-bob'") fixture.input.request.filter = "id = 'public-alpine'";
  }
  assert.ok(validateInventory(inventory, changed, repoRoot).includes('Access-Matrix unvollständig: anonymous/private-bob'));
});

test('jede Sortierrichtung benötigt sowohl einen Compiler- als auch einen Enginefall', () => {
  const changed = fixtures();
  for (const fixture of changed) {
    if (fixture.input.adapter === 'list-search' && fixture.input.filter?.sort === 'distance' && fixture.input.filter.sortOrder === '-') fixture.input.filter.sortOrder = '+';
  }
  assert.ok(validateInventory(inventory, changed, repoRoot).some(error => error.includes('distance:desc (Compiler=false, Engine=true)')));
});
