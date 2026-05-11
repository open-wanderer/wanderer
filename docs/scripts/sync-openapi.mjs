import { copyFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const docsDir = resolve(scriptDir, '..');
const repoDir = resolve(docsDir, '..');

const source = resolve(repoDir, 'web/static/docs/api/wanderer.openapi.json');
const destination = resolve(docsDir, 'wanderer.openapi.json');

if (!existsSync(source)) {
  console.error(`OpenAPI schema not found at ${source}. Run the web build first so it can be generated.`);
  process.exit(1);
}

copyFileSync(source, destination);
console.log(`Synced OpenAPI schema to ${destination}`);
