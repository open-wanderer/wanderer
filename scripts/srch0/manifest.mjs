import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { repoRoot, corpusRoot, families, baselineCommit, engineProfiles, canonical, sha256, jsonFiles, readJSON, safePath, settingsFingerprint, loadCorpus, readGroups, expandGroup, compileSchema, renderMatrix, gitFile, checkGoldenEvolution } from './corpus.mjs';
import { loadChanges, resolveObservation } from './expectations.mjs';
import { validateInventory } from './inventory.mjs';

const relative = (root, file) => path.relative(root, file).split(path.sep).join('/');
const sortedUnique = (list) => [...new Set(list)].sort();
const adapters = {
  state: ['route-load', 'sanitize'],
  projection: ['go-projection'],
  compiler: ['trail-dto', 'list-search', 'map-search', 'generic-search', 'global-multi', 'list-index-search', 'api-proxy', 'api-multi', 'api-actor', 'api-profile-trails', 'api-profile-lists', 'api-recommendation', 'api-bounds', 'api-cluster', 'api-filter-values', 'api-upload'],
  search: ['engine-search', 'engine-multi'],
  mutation: ['go-mutation', 'go-search-token', 'go-startup'],
  browser: ['browser'],
};

export function buildManifest(root = corpusRoot) {
  const old = fs.existsSync(path.join(root, 'manifest.json')) ? readJSON(path.join(root, 'manifest.json')) : null;
  const baseline = { commit: baselineCommit, observed_at: '2026-09-07', engine_profiles: engineProfiles, settings_fingerprint: settingsFingerprint(root) };
  const groups = readGroups(root, baseline);
  const cases = groups.flatMap(group => group.fixtures).sort((a, b) => a.case_id.localeCompare(b.case_id));
  if (new Set(cases.map(fixture => fixture.case_id)).size !== cases.length) throw new Error('Doppelte Case-ID in Fallgruppen');
  const artifacts = (directory) => jsonFiles(path.join(root, directory)).map((file) => ({ path: relative(root, file), revision: readJSON(file).revision, sha256: sha256(fs.readFileSync(file)) }));
  const inventory = readJSON(path.join(root, 'inventory.json'));
  return {
    schema_version: 'wanderer.srch0.manifest/v1', revision: old?.revision ?? 1,
    baseline,
    schema_sha256: sha256(fs.readFileSync(path.join(root, 'schema.json'))),
    inventory_sha256: sha256(fs.readFileSync(path.join(root, 'inventory.json'))),
    changes_sha256: sha256(fs.readFileSync(path.join(root, 'changes.json'))),
    sources_sha256: sha256(fs.readFileSync(path.join(root, 'sources.json'))),
    datasets: artifacts('datasets'), profiles: artifacts('profiles'),
    groups: groups.map(({ path, sha256 }) => ({ path, sha256 })),
    cases: Object.fromEntries(cases.map(fixture => [fixture.case_id, sha256(canonical(fixture))])),
    requirements: { families: inventory.families, consumers: inventory.consumers.map(item => item.id).sort() },
    retired: old?.retired ?? [],
  };
}

export function validateCorpus(root = corpusRoot, { checkMatrix = true, checkSources = false } = {}) {
  const errors = [];
  const { manifest, cases, manifestDigest } = loadCorpus(root);
  if (manifest.schema_version !== 'wanderer.srch0.manifest/v1') errors.push('Unknown manifest schema');
  if (!Number.isSafeInteger(manifest.revision) || manifest.revision < 1) errors.push('Manifest revision must be positive');
  const built = buildManifest(root);
  if (canonical(manifest) !== canonical(built)) errors.push('Manifest artifacts/digests are stale; inspect the change before regenerating');
  if (!manifest.datasets.length || manifest.profiles.length !== 3) errors.push('Require a versioned dataset and exactly three baseline index profiles');
  const datasets = new Map();
  for (const entry of manifest.datasets) {
    const dataset = readJSON(safePath(root, entry.path));
    datasets.set(entry.path, dataset);
    if (dataset.schema_version !== 'wanderer.srch0.dataset/v1' || !Number.isSafeInteger(dataset.revision) || dataset.revision < 1) errors.push(`${entry.path}: missing dataset schema/revision binding`);
    if (dataset.extends) {
      if (!manifest.datasets.some(item => item.path === dataset.extends) || dataset.extends === entry.path) errors.push(`${entry.path}: invalid parent dataset`);
      if (dataset.generator !== 'wanderer.srch0.scale/v1' || dataset.count !== 10001) errors.push(`${entry.path}: invalid versioned scale generator`);
    } else {
      if (!dataset.principals || !['anonymous', 'alice', 'bob'].every(key => key in dataset.principals)) errors.push(`${entry.path}: missing synthetic principals`);
      for (const key of ['actors', 'trails', 'lists', 'categories', 'subcategories', 'tags']) {
        if (!Array.isArray(dataset[key]) || !dataset[key].length) { errors.push(`${entry.path}: empty/missing ${key} collection`); continue; }
        const ids = dataset[key].map(item => item.id);
        if (ids.some(id => typeof id !== 'string' || !id) || new Set(ids).size !== ids.length) errors.push(`${entry.path}: invalid/duplicate ${key} IDs`);
      }
    }
  }
  const indexes = [];
  for (const entry of manifest.profiles) {
    const profile = readJSON(safePath(root, entry.path));
    indexes.push(profile.index_uid);
    if (profile.schema_version !== 'wanderer.srch0.profile/v1' || profile.baseline_commit !== baselineCommit || !Number.isSafeInteger(profile.revision) || profile.revision < 1) errors.push(`${entry.path}: invalid profile schema/revision/baseline binding`);
    if (profile.primary_key !== 'id' || !profile.profile_id || canonical(profile.engine_versions) !== canonical(['1.11.3', '1.36.0'])) errors.push(`${entry.path}: invalid profile identity or engine versions`);
    for (const key of ['searchableAttributes', 'filterableAttributes', 'sortableAttributes', 'rankingRules']) if (!Array.isArray(profile.settings?.[key]) || profile.settings[key].some(value => typeof value !== 'string')) errors.push(`${entry.path}: invalid ${key} settings`);
    if (!Array.isArray(profile.document_fields) || !profile.document_fields.includes('id')) errors.push(`${entry.path}: missing document form`);
  }
  if (canonical(indexes.sort()) !== canonical(['actors', 'lists', 'trails'])) errors.push('Profiles must cover trails, lists and actors exactly once');
  if (canonical(Object.keys(manifest.requirements.families).sort()) !== canonical([...families].sort())) errors.push('All six mandatory fixture families must be declared');
  const schema = compileSchema(root);
  const ids = new Set();
  const sources = readJSON(path.join(root, 'sources.json'));
  if (sources.commit !== baselineCommit || sources.schema_version !== 'wanderer.srch0.sources/v1') errors.push('Ungültige Quellenprovenienz');
  const sourcePaths = new Set(sources.files.map(source => source.path));
  for (const source of sources.files) {
    safePath(repoRoot, source.path);
    if (!/^[a-f0-9]{64}$/.test(source.sha256)) errors.push(`Ungültiger Quellenhash: ${source.path}`);
    if (checkSources && sha256(gitFile(sources.commit, source.path)) !== source.sha256) errors.push(`Quellenprovenienz stimmt nicht: ${source.path}`);
  }
  for (const { entry, fixture } of cases) {
    const prefix = `${fixture.case_id} (${entry.path})`;
    if (!schema(fixture)) { errors.push(`${prefix}: ${JSON.stringify(schema.errors)}`); continue; }
    if (ids.has(fixture.case_id)) errors.push(`${prefix}: duplicate ID`);
    ids.add(fixture.case_id);
    if (!fixture.case_id.startsWith(`SRCH0-${fixture.family.toUpperCase()}-`)) errors.push(`${prefix}: ID/family mismatch`);
    if (!adapters[fixture.family].includes(fixture.input.adapter)) errors.push(`${prefix}: no executable ${fixture.family} adapter for ${fixture.input.adapter}`);
    if ('url' in fixture.input && typeof fixture.input.url !== 'string') errors.push(`${prefix}: URL must retain its raw string representation`);
    if ('storage' in fixture.input && (!fixture.input.storage || typeof fixture.input.storage !== 'object' || Object.values(fixture.input.storage).some(value => typeof value !== 'string'))) errors.push(`${prefix}: LocalStorage values must retain raw strings`);
    if (!entry.path.startsWith(`cases/${fixture.family}/`)) errors.push(`${prefix}: incorrect family directory`);
    if (canonical(fixture.baseline) !== canonical(manifest.baseline)) errors.push(`${prefix}: baseline differs from manifest`);
    if (canonical(fixture.successor_refs) !== canonical(sortedUnique(fixture.successor_refs))) errors.push(`${prefix}: successor refs must be sorted and unique`);
    if (!manifest.datasets.some((item) => item.path === fixture.dataset_ref)) errors.push(`${prefix}: dataset is not bound by manifest`);
    const dataset = datasets.get(fixture.dataset_ref);
    const baseDataset = dataset?.extends ? datasets.get(dataset.extends) : dataset;
    if (!baseDataset?.principals || !(fixture.context.principal in baseDataset.principals)) errors.push(`${prefix}: principal is not in the bound synthetic dataset`);
    if (!manifest.requirements.consumers.includes(fixture.context.consumer)) errors.push(`${prefix}: consumer missing from declared inventory`);
    for (const code of fixture.evidence.code) {
      safePath(repoRoot, code.path);
      if (!sourcePaths.has(code.path)) errors.push(`${prefix}: Codeanker fehlt in sources.json: ${code.path}`);
    }
  }
  for (const family of families) {
    const selected = cases.filter(({ fixture }) => fixture.family === family);
    const requirement = manifest.requirements.families[family];
    if (!requirement || requirement.minimum < 3 || selected.length < requirement.minimum) errors.push(`${family}: below declared minimum coverage`);
    const coverage = new Set(selected.flatMap(({ fixture }) => fixture.evidence.coverage));
    for (const tag of ['positive', 'negative', 'boundary', ...(requirement?.coverage ?? [])]) if (!coverage.has(tag)) errors.push(`${family}: missing ${tag} coverage`);
  }
  for (const consumer of manifest.requirements.consumers) if (!cases.some(({ fixture }) => fixture.context.consumer === consumer)) errors.push(`Uncovered First-Party consumer: ${consumer}`);
  errors.push(...validateInventory(readJSON(path.join(root, 'inventory.json')), cases.map(item => item.fixture), repoRoot));
  const changes = loadChanges(root);
  const changedCases = new Set();
  const changeIDs = new Set();
  for (const change of changes) {
    const record = cases.find(({ fixture }) => fixture.case_id === change.case_id);
    if (!change.id || changeIDs.has(change.id) || changedCases.has(change.case_id)) errors.push('Änderungs-IDs und betroffene Fälle müssen eindeutig sein');
    changeIDs.add(change.id); changedCases.add(change.case_id);
    if (!record) { errors.push(`${change.id}: ungebundene Solländerung`); continue; }
    for (const file of change.evidence) if (!fs.existsSync(safePath(repoRoot, file))) errors.push(`${change.id}: Evidenzdatei fehlt: ${file}`);
    try {
      const { settings_overrides, ...observed } = resolveObservation(record.fixture, record.entry.sha256, changes);
      if (!schema({ ...record.fixture, observed })) errors.push(`${change.id}: ungültige Ersatzbeobachtung: ${JSON.stringify(schema.errors)}`);
      if (settings_overrides !== undefined && (record.fixture.input.adapter !== 'go-startup'
        || !settings_overrides || typeof settings_overrides !== 'object' || Array.isArray(settings_overrides))) errors.push(`${change.id}: settings_overrides ist nur als Startup-Objekt zulässig`);
    }
    catch (error) { errors.push(`${change.id}: ${error.message}`); }
  }
  if (canonical(manifest.requirements.consumers) !== canonical(sortedUnique(manifest.requirements.consumers))) errors.push('Consumer inventory must be sorted and unique');
  for (const item of manifest.retired) {
    if (ids.has(item.case_id) || !item.reason || !item.successor_refs?.length) errors.push(`${item.case_id}: invalid retirement`);
    if (ids.has(`retired:${item.case_id}`)) errors.push(`${item.case_id}: duplicate retirement`);
    ids.add(`retired:${item.case_id}`);
  }
  if (checkMatrix && fs.readFileSync(path.join(root, 'README.md'), 'utf8') !== renderMatrix({ manifest, cases })) errors.push('Generated README matrix drift');
  return { errors, manifestDigest, count: cases.length };
}

function previousCorpus(ref) {
  let bytes;
  try { bytes = gitFile(ref, 'testdata/trail-search/srch0/v1/manifest.json'); }
  catch {
    // An absent file is legitimate on the initial SRCH0 PR; an invalid ref is not.
    gitFile(ref, 'web/package.json');
    return null;
  }
  const manifest = JSON.parse(bytes);
  return { manifest, cases: manifest.groups.flatMap(group => expandGroup(JSON.parse(gitFile(ref, `testdata/trail-search/srch0/v1/${group.path}`)), manifest.baseline).map(fixture => ({
    entry: { case_id: fixture.case_id, path: group.path, sha256: manifest.cases[fixture.case_id] }, fixture,
  }))) };
}

export function main(args = process.argv.slice(2)) {
  if (args.includes('--write')) {
    const manifest = buildManifest();
    fs.writeFileSync(path.join(corpusRoot, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n');
    fs.writeFileSync(path.join(corpusRoot, 'README.md'), renderMatrix(loadCorpus()));
    console.log('Wrote artifact digests and matrix; observed fixtures were not changed.');
    return;
  }
  const result = validateCorpus(corpusRoot, { checkSources: args.includes('--verify-baseline') });
  const baseIndex = args.indexOf('--base-ref');
  if (baseIndex >= 0) {
    if (!args[baseIndex + 1]) throw new Error('--base-ref requires a Git revision');
    const previous = previousCorpus(args[baseIndex + 1]);
    if (previous) result.errors.push(...checkGoldenEvolution(previous, loadCorpus()));
  }
  if (result.errors.length) throw new Error(result.errors.join('\n'));
  console.log(`SRCH0: ${result.count} schema-valid cases; manifest=${result.manifestDigest}; all families, consumers, digests and matrix verified.`);
}
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { console.error(error.message); process.exitCode = 1; }
}
