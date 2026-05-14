import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { generateSpec, writeSpec } from '../node_modules/sveltekit-openapi-generator/dist/generator.js';
import { openapiOptions } from '../openapi.config.js';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const webDir = resolve(scriptDir, '..');
const packageJson = JSON.parse(readFileSync(resolve(webDir, 'package.json'), 'utf-8'));
const { outputPath, ...generatorOptions } = openapiOptions(packageJson.version);

const spec = generateSpec({
  rootDir: webDir,
  ...generatorOptions,
});

writeSpec(spec, resolve(webDir, outputPath));
