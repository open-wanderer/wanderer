import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { corpusRoot, loadCorpus, checkGoldenEvolution, safePath, compileSchema, expandGroup } from './corpus.mjs';
import { buildManifest, validateCorpus } from './manifest.mjs';
import { resolveObservation } from './expectations.mjs';

test('the canonical corpus is complete, schema valid and bound to reproducible source', () => {
  assert.deepEqual(validateCorpus().errors, []);
});

test('changing a published observation is rejected even with new evidence and regenerated digests', () => {
  const before = loadCorpus();
  const after = structuredClone(before);
  after.cases[0].fixture.observed = { diagnostics: { category: 'deliberate-corruption' } };
  after.cases[0].fixture.evidence.method = 'Another capture cannot rewrite published history';
  assert.ok(checkGoldenEvolution(before, after).some((error) => error.includes('historische Beobachtung geändert')));
});

test('removing a case or reusing an ID is rejected during review', () => {
  const before = loadCorpus();
  const after = structuredClone(before);
  after.cases.shift();
  assert.ok(checkGoldenEvolution(before, after).some((error) => error.includes('without an explicit retirement')));
  const changed = structuredClone(before);
  changed.cases[0].fixture.input.adapter = 'different-meaning';
  assert.ok(checkGoldenEvolution(before, changed).some((error) => error.includes('assign a new case ID')));
  const rebound = structuredClone(before);
  rebound.cases[0].fixture.dataset_ref = 'datasets/different-history.json';
  assert.ok(checkGoldenEvolution(before, rebound).some((error) => error.includes('dataset_ref changed')));
});

test('changing a dataset in place cannot silently reuse its revision', () => {
  const before = loadCorpus();
  const after = structuredClone(before);
  after.manifest.datasets[0].sha256 = '0'.repeat(64);
  assert.ok(checkGoldenEvolution(before, after).some(error => error.includes('changed artifact requires a new revision')));
  assert.ok(checkGoldenEvolution(before, after).some(error => error.includes('new manifest revision')));
});

test('a missing physical case does not become a green reduced corpus', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'wanderer-srch0-manifest-'));
  try {
    fs.cpSync(corpusRoot, directory, { recursive: true });
    const { manifest } = loadCorpus(directory);
    fs.unlinkSync(safePath(directory, manifest.groups[0].path));
    assert.throws(() => validateCorpus(directory), /ENOENT/);
  } finally { fs.rmSync(directory, { recursive: true, force: true }); }
});

test('unknown successor fields and unsafe artifact paths are rejected', () => {
  const validate = compileSchema();
  const fixture = structuredClone(loadCorpus().cases[0].fixture);
  fixture.expected = { success: true };
  assert.equal(validate(fixture), false);
  assert.ok(validate.errors.some(error => error.keyword === 'additionalProperties'));
  assert.throws(() => safePath(corpusRoot, '../outside.json'), /Unsafe/);
  assert.throws(() => safePath(corpusRoot, '/tmp/outside.json'), /Unsafe/);
});

test('successor fields cannot enter baseline observations, but settings changes resolve separately', () => {
  const validate = compileSchema();
  const { entry, fixture } = loadCorpus().cases.find(({ fixture }) => fixture.case_id === 'SRCH0-MUTATION-090');
  for (const key of ['expected', 'allowed_delta', 'delivery_owner', 'activation_gate', 'settings_overrides']) {
    const changed = structuredClone(fixture);
    changed.observed[key] = {};
    assert.equal(validate(changed), false, `baseline must reject observed.${key}`);
  }
  const settings = { trails: { filterableAttributes: ['author', 'new_field'] } };
  const resolved = resolveObservation(fixture, entry.sha256, [{
    case_id: fixture.case_id, basis_digest: entry.sha256,
    observed: { ...fixture.observed, settings_overrides: settings },
  }]);
  assert.deepEqual(resolved.settings_overrides, settings);
  assert.equal(Object.hasOwn(fixture.observed, 'settings_overrides'), false);
  assert.equal(validate(fixture), true);
});

test('eine Gruppenänderung verändert nur den Digest des tatsächlich betroffenen Falls', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'wanderer-srch0-group-'));
  try {
    fs.cpSync(corpusRoot, directory, { recursive: true });
    const before = buildManifest(directory);
    const file = path.join(directory, 'cases/compiler/sorting.json');
    const group = JSON.parse(fs.readFileSync(file));
    const [changed, sibling] = group.cases;
    changed.observed = { deliberately: 'changed' };
    fs.writeFileSync(file, JSON.stringify(group));
    assert.throws(() => loadCorpus(directory), /Dateidigest/);
    const after = buildManifest(directory);
    assert.notEqual(after.cases[changed.case_id], before.cases[changed.case_id]);
    assert.equal(after.cases[sibling.case_id], before.cases[sibling.case_id]);
  } finally { fs.rmSync(directory, { recursive: true, force: true }); }
});

test('vollständige Ersatzbeobachtungen erfüllen das Schema; nur Startup erlaubt zusätzliche Settings', () => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'wanderer-srch0-replacement-'));
  try {
    fs.cpSync(corpusRoot, directory, { recursive: true });
    const { entry, fixture } = loadCorpus(directory).cases.find(({ fixture }) => fixture.case_id === 'SRCH0-MUTATION-090');
    const check = observed => {
      fs.writeFileSync(path.join(directory, 'changes.json'), JSON.stringify({
        schema_version: 'wanderer.srch0.changes/v1',
        changes: [{ id: 'SRCH-STARTUP-001', case_id: fixture.case_id, basis_digest: entry.sha256,
          reason: 'Prüfe die aktive Beobachtung', evidence: ['db/main.go'], observed }],
      }));
      fs.writeFileSync(path.join(directory, 'manifest.json'), JSON.stringify(buildManifest(directory)));
      return validateCorpus(directory, { checkMatrix: false }).errors;
    };
    assert.ok(check({}).some(error => error.includes('ungültige Ersatzbeobachtung')));
    for (const key of ['expected', 'allowed_delta', 'delivery_owner', 'activation_gate']) {
      assert.ok(check({ ...fixture.observed, [key]: {} }).some(error => error.includes('ungültige Ersatzbeobachtung')), key);
    }
    assert.ok(check({ ...fixture.observed, settings_overrides: [] }).some(error => error.includes('Startup-Objekt')));
    assert.deepEqual(check({ ...fixture.observed, settings_overrides: { trails: { filterableAttributes: ['author', 'new_field'] } } }), []);
  } finally { fs.rmSync(directory, { recursive: true, force: true }); }
});

test('Gruppen erlauben keine versteckten Erwartungsdefaults und ersetzen Kontextfelder nur flach', () => {
  const group = JSON.parse(fs.readFileSync(path.join(corpusRoot, 'cases/compiler/sorting.json')));
  const baseline = loadCorpus().manifest.baseline;
  group.observed = { unexpected: 'group default' };
  assert.throws(() => expandGroup(group, baseline), /unbekannte Felder/);
  delete group.observed;
  group.context.preferences = { old: true };
  group.cases[0].context = { preferences: { replacement: true } };
  const [fixture] = expandGroup(group, baseline);
  assert.deepEqual(fixture.context.preferences, { replacement: true });
  assert.equal(fixture.context.principal, group.context.principal);
});
