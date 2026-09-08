import fs from 'node:fs';
import path from 'node:path';

/** Das Inventar ist die Verpflichtung; die vorhandenen Fälle sind ihr Nachweis. */
export function validateInventory(inventory, fixtures, repoRoot) {
  const errors = [];
  if (inventory.schema_version !== 'wanderer.srch0.inventory/v1') return ['Unbekanntes Inventarformat'];
  const consumers = new Set();
  for (const consumer of inventory.consumers) {
    if (consumers.has(consumer.id)) errors.push(`Doppelter Consumer: ${consumer.id}`);
    consumers.add(consumer.id);
    if (!consumer.description || !fs.existsSync(path.join(repoRoot, consumer.source))) errors.push(`Consumer ohne überprüfbaren Codeanker: ${consumer.id}`);
    if (!fixtures.some(fixture => fixture.context.consumer === consumer.id)) errors.push(`Consumer ohne Fall: ${consumer.id}`);
  }
  for (const [family, obligations] of Object.entries(inventory.obligations)) for (const [id, tags] of Object.entries(obligations)) {
    const covered = fixtures.some(fixture => fixture.family === family && tags.every(tag => fixture.evidence.coverage.includes(tag)));
    if (!covered) errors.push(`Pflichtmerkmal ohne Fall: ${family}.${id}`);
  }
  // Diese beiden Matrizen prüfen tatsächliche Eingaben statt frei vergebener Tags.
  for (const principal of inventory.access.principals) for (const target of inventory.access.targets) {
    if (!fixtures.some(fixture => fixture.family === 'search' && fixture.context.principal === principal && fixture.input.request?.filter === `id = '${target}'`)) {
      errors.push(`Access-Matrix unvollständig: ${principal}/${target}`);
    }
  }
  for (const key of inventory.sort_keys) for (const direction of ['asc', 'desc']) {
    const compiled = fixtures.some(fixture => fixture.input.adapter === 'list-search' && fixture.input.filter?.sort === key && fixture.input.filter?.sortOrder === (direction === 'asc' ? '+' : '-'));
    const executed = fixtures.some(fixture => fixture.family === 'search' && fixture.input.request?.q === '' && fixture.input.request?.sort?.[0] === `${key}:${direction}`);
    if (!compiled || !executed) errors.push(`Sortiermatrix unvollständig: ${key}:${direction} (Compiler=${compiled}, Engine=${executed})`);
  }
  return errors;
}
