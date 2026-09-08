import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { loadChanges, resolveObservation } from './expectations.mjs';
export { checkGoldenEvolution } from './evolution.mjs';
export { renderMatrix } from './matrix.mjs';

export const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
export const corpusRoot = path.join(repoRoot, 'testdata/trail-search/srch0/v1');
export const families = ['state', 'projection', 'compiler', 'search', 'mutation', 'browser'];
export const baselineCommit = 'e9b7a8cade980002acbcf2e2f5b2a083934f29d2';
export const engineProfiles = ['meilisearch-1.11.3', 'meilisearch-1.36.0'];
export const sha256 = (value) => crypto.createHash('sha256').update(value).digest('hex');
export const canonical = (value) => JSON.stringify(sortKeys(value));
export function sortKeys(value) {
  if (Array.isArray(value)) return value.map(sortKeys);
  if (value && typeof value === 'object') return Object.fromEntries(Object.keys(value).sort().map((key) => [key, sortKeys(value[key])]));
  return value;
}
export function readJSON(file) { return JSON.parse(fs.readFileSync(file, 'utf8')); }
export function jsonFiles(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(directory, entry.name);
    if (entry.isSymbolicLink()) throw new Error(`Symlinks are not corpus artifacts: ${file}`);
    return entry.isDirectory() ? jsonFiles(file) : entry.name.endsWith('.json') ? [file] : [];
  }).sort();
}
export function safePath(root, relative) {
  if (typeof relative !== 'string' || relative.includes('\\') || path.isAbsolute(relative) || relative.split('/').includes('..')) throw new Error(`Unsafe artifact path: ${relative}`);
  const file = path.resolve(root, relative);
  if (!file.startsWith(path.resolve(root) + path.sep)) throw new Error(`Artifact escapes corpus: ${relative}`);
  if (fs.existsSync(file) && fs.realpathSync(file) !== file) throw new Error(`Artifact is a symlink: ${relative}`);
  return file;
}
export function gitFile(ref, relative) {
  return execFileSync('git', ['show', `${ref}:${relative}`], { cwd: repoRoot, maxBuffer: 20 * 1024 * 1024, stdio: ['ignore', 'pipe', 'pipe'] });
}
export function settingsFingerprint(root = corpusRoot) {
  return sha256(canonical(jsonFiles(path.join(root, 'profiles')).map((file) => ({
    path: path.relative(root, file).split(path.sep).join('/'),
    sha256: sha256(fs.readFileSync(file)),
  }))));
}
function allowedKeys(value, keys, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value) || Object.keys(value).some(key => !keys.includes(key))) throw new Error(`${label}: unbekannte Felder oder ungültiges Objekt`);
}

/** Nur Metadaten haben Gruppendefaults; Eingabe und Beobachtung stehen vollständig im Fall. */
export function expandGroup(group, baseline) {
  allowedKeys(group, ['family', 'dataset_ref', 'context', 'code', 'cases'], 'Fallgruppe');
  if (!Array.isArray(group.cases) || !group.cases.length) throw new Error('Fallgruppe muss Fälle enthalten');
  const contextKeys = ['principal', 'locale', 'timezone', 'preferences', 'surface', 'consumer'];
  allowedKeys(group.context, contextKeys, 'Gruppenkontext');
  return group.cases.map(item => {
    allowedKeys(item, ['case_id', 'input', 'observed', 'stability', 'successor_refs', 'context', 'evidence'], 'Gruppenfall');
    if (item.context !== undefined) allowedKeys(item.context, contextKeys, 'Fallkontext');
    allowedKeys(item.evidence, ['code', 'method', 'coverage'], 'Fallevidenz');
    return {
      schema_version: 'wanderer.srch0/v1', case_id: item.case_id, family: group.family,
      baseline, dataset_ref: group.dataset_ref, context: { ...group.context, ...item.context },
      input: item.input, observed: item.observed, stability: item.stability,
      successor_refs: item.successor_refs ?? [], evidence: { ...item.evidence, code: item.evidence.code ?? group.code },
    };
  });
}

export function readGroups(root, baseline) {
  return jsonFiles(path.join(root, 'cases')).map(file => ({
    path: path.relative(root, file).split(path.sep).join('/'),
    sha256: sha256(fs.readFileSync(file)), fixtures: expandGroup(readJSON(file), baseline),
  }));
}

export function loadCorpus(root = corpusRoot, { expectations = false } = {}) {
  const bytes = fs.readFileSync(path.join(root, 'manifest.json'));
  const manifest = JSON.parse(bytes);
  const verify = entry => {
    if (sha256(fs.readFileSync(safePath(root, entry.path))) !== entry.sha256) throw new Error(`${entry.path}: Dateidigest unterscheidet sich vom Manifest`);
  };
  for (const entry of [...manifest.groups, ...manifest.datasets, ...manifest.profiles]) verify(entry);
  verify({ path: 'changes.json', sha256: manifest.changes_sha256 });
  const changes = loadChanges(root);
  const cases = manifest.groups.flatMap(group => expandGroup(readJSON(safePath(root, group.path)), manifest.baseline).map(fixture => {
    const digest = sha256(canonical(fixture));
    if (manifest.cases[fixture.case_id] !== digest) throw new Error(`${fixture.case_id}: Falldigest unterscheidet sich vom Manifest`);
    const entry = { case_id: fixture.case_id, family: fixture.family, path: group.path, sha256: digest, dataset_ref: fixture.dataset_ref, consumer: fixture.context.consumer };
    return { entry, fixture: expectations ? { ...fixture, baseline_observed: fixture.observed, observed: resolveObservation(fixture, digest, changes) } : fixture };
  }));
  const ids = cases.map(({ fixture }) => fixture.case_id).sort();
  if (new Set(ids).size !== ids.length || canonical(ids) !== canonical(Object.keys(manifest.cases).sort())) throw new Error('Fallinventar unterscheidet sich vom Manifest');
  return { root, manifest, manifestDigest: sha256(bytes), cases };
}
export function formatCase(fixture, digest, profile = 'none') {
  return `${fixture.case_id} family=${fixture.family} basis=${digest} engine=${profile}`;
}
export function structuralDiff(actual, observed, at = '$') {
  if (canonical(actual) === canonical(observed)) return null;
  if (actual && observed && typeof actual === 'object' && typeof observed === 'object') {
    for (const key of [...new Set([...Object.keys(actual), ...Object.keys(observed)])].sort()) {
      const diff = structuralDiff(actual[key], observed[key], `${at}.${key}`);
      if (diff) return diff;
    }
  }
  return `${at}: observed=${JSON.stringify(observed)} actual=${JSON.stringify(actual)}`;
}
export function compileSchema(root = corpusRoot) {
  const require = createRequire(path.join(repoRoot, 'web/package.json'));
  const Ajv = require('ajv');
  return new Ajv({ allErrors: true, strict: true }).compile(readJSON(path.join(root, 'schema.json')));
}
