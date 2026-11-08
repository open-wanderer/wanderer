import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { expect, it } from "vitest";

it("rejects valid but empty generated artifacts in the actual validator process", () => {
    const fixture = mkdtempSync(join(tmpdir(), "wanderer-openapi-validation-"));
    try {
        for (const paths of [{}, { "/photos": { parameters: [] } }]) {
            const file = join(fixture, "empty.json");
            writeFileSync(file, JSON.stringify({ openapi: "3.0.3", info: { title: "Empty", version: "1" }, paths }));
            const result = spawnSync(process.execPath, [resolve("scripts/validate-openapi.js"), file], { encoding: "utf8" });
            expect(result.status).toBe(1);
            expect(result.stderr).toContain("at least one operation");
        }
        const file = join(fixture, "one-operation.json");
        writeFileSync(file, JSON.stringify({ openapi: "3.0.3", info: { title: "API", version: "1" }, paths: { "/photos": { get: { responses: { "200": { description: "OK" } } } } } }));
        const result = spawnSync(process.execPath, [resolve("scripts/validate-openapi.js"), file], { encoding: "utf8" });
        expect(result.status).toBe(0);
        expect(result.stdout).toContain("1 operations");
    } finally {
        rmSync(fixture, { recursive: true, force: true });
    }
});
