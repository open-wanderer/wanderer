import { isDeepStrictEqual as same } from 'node:util';

/** Prüft Struktur und Identität; fachliche Richtigkeit bleibt Reviewaufgabe. */
export function checkGoldenEvolution(previous, current) {
  const errors = [];
  const after = new Map(current.cases.map(({ fixture }) => [fixture.case_id, fixture]));
  const retired = new Map(current.manifest.retired.map(item => [item.case_id, item]));
  for (const { fixture: old } of previous.cases) {
    const next = after.get(old.case_id);
    if (!next) {
      const retirement = retired.get(old.case_id);
      if (!retirement?.reason || !retirement.successor_refs?.length) errors.push(`${old.case_id}: removed without an explicit retirement and successor`);
      continue;
    }
    for (const field of ['input', 'context', 'family', 'baseline', 'dataset_ref']) {
      if (!same(old[field], next[field])) errors.push(`${old.case_id}: ${field} changed; assign a new case ID`);
    }
    if (!same(old.observed, next.observed)) errors.push(`${old.case_id}: historische Beobachtung geändert; Produktkorrekturen gehören in changes.json, neue Beobachtungen in einen neuen Fall`);
  }
  for (const old of previous.manifest.retired) {
    if (after.has(old.case_id) || !retired.has(old.case_id)) errors.push(`${old.case_id}: retired IDs must remain reserved`);
  }
  let artifactsChanged = false;
  for (const family of ['datasets', 'profiles']) {
    const oldArtifacts = new Map(previous.manifest[family].map(item => [item.path, item]));
    const nextArtifacts = new Map(current.manifest[family].map(item => [item.path, item]));
    if (!same([...oldArtifacts.keys()].sort(), [...nextArtifacts.keys()].sort())) artifactsChanged = true;
    for (const entry of nextArtifacts.values()) {
      const old = oldArtifacts.get(entry.path);
      if (!old || old.sha256 !== entry.sha256) artifactsChanged = true;
      if (old && old.sha256 !== entry.sha256 && !(entry.revision > old.revision)) errors.push(`${entry.path}: changed artifact requires a new revision`);
    }
  }
  const idsChanged = !same(previous.cases.map(({ fixture }) => fixture.case_id).sort(), [...after.keys()].sort());
  if ((idsChanged || artifactsChanged) && current.manifest.revision <= previous.manifest.revision) errors.push('Added/removed cases or changed artifacts require a new manifest revision');
  return errors;
}
