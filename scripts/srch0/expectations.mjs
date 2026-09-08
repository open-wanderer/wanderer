import fs from 'node:fs';
import path from 'node:path';

/** Eine Solländerung ersetzt die vollständige Beobachtung; die Basis bleibt erhalten. */
export function loadChanges(root) {
  const document = JSON.parse(fs.readFileSync(path.join(root, 'changes.json'), 'utf8'));
  if (document.schema_version !== 'wanderer.srch0.changes/v1' || !Array.isArray(document.changes)
    || Object.keys(document).some(key => !['schema_version', 'changes'].includes(key))) throw new Error('changes.json: unbekanntes Änderungsformat');
  const keys = ['id', 'case_id', 'basis_digest', 'reason', 'evidence', 'observed'];
  for (const change of document.changes) {
    if (!change || Object.keys(change).some(key => !keys.includes(key)) || keys.some(key => !Object.hasOwn(change, key))
      || !['id', 'case_id', 'reason'].every(key => typeof change[key] === 'string' && change[key].trim())
      || !/^[a-f0-9]{64}$/.test(change.basis_digest)
      || !Array.isArray(change.evidence) || !change.evidence.length || change.evidence.some(file => typeof file !== 'string' || !file)
      || !change.observed || typeof change.observed !== 'object' || Array.isArray(change.observed)) throw new Error('changes.json: unvollständige oder unbekannte Änderungsfelder');
  }
  return document.changes;
}

export function resolveObservation(fixture, basisDigest, changes) {
  const matching = changes.filter(change => change.case_id === fixture.case_id);
  if (matching.length > 1) throw new Error(`${fixture.case_id}: mehrere aktive Änderungen`);
  const change = matching[0];
  if (!change) return structuredClone(fixture.observed);
  if (change.basis_digest !== basisDigest) throw new Error(`${fixture.case_id}: Änderung gehört zu einem anderen Basisdigest`);
  if (!change.observed || typeof change.observed !== 'object' || Array.isArray(change.observed)) throw new Error(`${fixture.case_id}: Ersatzbeobachtung muss ein Objekt sein`);
  return structuredClone(change.observed);
}
