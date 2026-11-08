import { mkdtempSync, mkdirSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { expect, it } from "vitest";

it("fails the actual generator process when a route contains malformed Swagger YAML", () => {
    const fixture = mkdtempSync(join(tmpdir(), "wanderer-openapi-"));
    try {
        mkdirSync(join(fixture, "scripts"));
        mkdirSync(join(fixture, "src/routes/broken"), { recursive: true });
        symlinkSync(resolve("node_modules"), join(fixture, "node_modules"));
        writeFileSync(join(fixture, "package.json"), JSON.stringify({ type: "module", version: "1.0.0" }));
        writeFileSync(join(fixture, "openapi.config.js"), `export const openapiOptions = () => ({outputPath:"spec.json",info:{title:"Fixture",version:"1"}});`);
        writeFileSync(join(fixture, "scripts/generate-openapi.js"), readFileSync(resolve("scripts/generate-openapi.js")));
        writeFileSync(join(fixture, "src/routes/broken/+server.ts"), `/**
 * @swagger
 * /broken:
 *   get:
 *     summary: [unterminated
 *     responses:
 *       200: {description: OK}
 */
export function GET() { return new Response("ok"); }
`);
        const result = spawnSync(process.execPath, [join(fixture, "scripts/generate-openapi.js")], { encoding: "utf8" });
        expect(result.status).not.toBe(0);
        expect(result.stderr).toMatch(/YAML|unterminated|unexpected|insufficient|Errors in/);
    } finally {
        rmSync(fixture, { recursive: true, force: true });
    }
});

it("fails the actual generator process when shared schemas contain malformed Swagger YAML", () => {
    const fixture = mkdtempSync(join(tmpdir(), "wanderer-openapi-"));
    try {
        mkdirSync(join(fixture, "scripts"));
        mkdirSync(join(fixture, "src/lib/models/api"), { recursive: true });
        mkdirSync(join(fixture, "src/routes/valid"), { recursive: true });
        symlinkSync(resolve("node_modules"), join(fixture, "node_modules"));
        writeFileSync(join(fixture, "package.json"), JSON.stringify({ type: "module", version: "1.0.0" }));
        writeFileSync(join(fixture, "openapi.config.js"), `export const openapiOptions = () => ({outputPath:"spec.json",info:{title:"Fixture",version:"1"},baseSchemasPath:"src/lib/models/api/openapi_schemas.ts"});`);
        writeFileSync(join(fixture, "scripts/generate-openapi.js"), readFileSync(resolve("scripts/generate-openapi.js")));
        writeFileSync(join(fixture, "src/routes/valid/+server.ts"), `/**
 * @swagger
 * /valid:
 *   get:
 *     responses:
 *       200: {description: OK}
 */
export function GET() { return new Response("ok"); }
`);
        writeFileSync(join(fixture, "src/lib/models/api/openapi_schemas.ts"), `/**
 * @swagger
 * components:
 *   schemas:
 *     Unused:
 *       description: [unterminated
 */
`);
        const result = spawnSync(process.execPath, [join(fixture, "scripts/generate-openapi.js")], { encoding: "utf8" });
        expect(result.status).not.toBe(0);
        expect(result.stderr).toMatch(/YAML|unterminated|unexpected|insufficient|Errors in/);
    } finally {
        rmSync(fixture, { recursive: true, force: true });
    }
});
