import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { generateSpec, writeSpec } from '../node_modules/sveltekit-openapi-generator/dist/generator.js';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const webDir = resolve(scriptDir, '..');
const packageJson = JSON.parse(readFileSync(resolve(webDir, 'package.json'), 'utf-8'));

const spec = generateSpec({
  rootDir: webDir,
  info: {
    title: 'Wanderer API',
    version: `${packageJson.version}`,
    description: 'API documentation for wanderer backend',
  },
  outputPath: 'static/docs/api/wanderer.openapi.json',
  include: ['src/routes/api/v1/**/*.{js,ts}'],
  baseSchemasPath: 'src/lib/models/api/openapi_schemas.ts',
});

writeSpec(spec, resolve(webDir, 'static/docs/api/wanderer.openapi.json'));
