import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import SwaggerParser from '@apidevtools/swagger-parser';
import { openapiOptions } from '../openapi.config.js';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const packageJson = JSON.parse(readFileSync(resolve(webDir, 'package.json'), 'utf-8'));
const specPath = resolve(webDir, process.argv[2] ?? openapiOptions(packageJson.version).outputPath);

try {
    // Validate the generated artifact, including every local reference.
    // CI validation does not fetch schemas from the network.
    const spec = await SwaggerParser.validate(specPath, { resolve: { http: false } });
    const operationMethods = new Set(['get', 'put', 'post', 'delete', 'options', 'head', 'patch', 'trace']);
    const operations = Object.values(spec.paths ?? {}).reduce(
        (count, path) => count + Object.keys(path).filter((method) => operationMethods.has(method)).length,
        0,
    );
    if (operations < 1) throw new Error("OpenAPI must document at least one operation");
    console.log(`OpenAPI valid: ${Object.keys(spec.paths ?? {}).length} paths, ${operations} operations.`);
} catch (error) {
    console.error(`OpenAPI validation failed: ${error instanceof Error ? error.message : String(error)}`);
    process.exitCode = 1;
}
