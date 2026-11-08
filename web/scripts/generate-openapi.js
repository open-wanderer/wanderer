import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import swaggerJSDoc from 'swagger-jsdoc';

import { generateSpec, writeSpec } from '../node_modules/sveltekit-openapi-generator/dist/generator.js';
import { openapiOptions } from '../openapi.config.js';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const webDir = resolve(scriptDir, '..');
const packageJson = JSON.parse(readFileSync(resolve(webDir, 'package.json'), 'utf-8'));
const { outputPath, ...generatorOptions } = openapiOptions(packageJson.version);

if (generatorOptions.baseSchemasPath) {
  swaggerJSDoc({
    definition: { openapi: '3.0.0', info: generatorOptions.info },
    apis: [resolve(webDir, generatorOptions.baseSchemasPath)],
    failOnErrors: true,
  });
}

const spec = generateSpec({
  rootDir: webDir,
  ...generatorOptions,
  failOnErrors: true,
});

writeSpec(spec, resolve(webDir, outputPath));
