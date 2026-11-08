import { expect, it } from "vitest";
import { generateSpec } from "../../../node_modules/sveltekit-openapi-generator/dist/generator.js";
import { openapiOptions } from "../../../openapi.config.js";
import { TrailUpdateSchema } from "#lib/models/api/trail_schema.ts";

it("documents trail writes and the successful GPX upload status", () => {
    const { outputPath, ...options } = openapiOptions("1.0.0");
    const spec = generateSpec({ rootDir: process.cwd(), ...options, failOnErrors: true });
    expect(spec.paths?.["/api/v1/trail/{id}"]?.post).toMatchObject({
        summary: "Update a trail",
        responses: { "200": { description: "Updated trail" } },
    });
    expect(spec.paths?.["/api/v1/trail/{id}"]?.delete).toMatchObject({
        summary: "Delete a trail",
        responses: { "200": { description: "Deleted trail" } },
    });
    const responses = spec.paths?.["/api/v1/trail/upload"]?.put?.responses;
    expect(responses).toHaveProperty("200");
    expect(responses).not.toHaveProperty("201");
});

it("matches required JSON update fields without requiring them on multipart routes", () => {
    const { outputPath, ...options } = openapiOptions("1.0.0");
    const spec = generateSpec({ rootDir: process.cwd(), ...options, failOnErrors: true });
    type InputSchema = { $ref?: string; required?: string[]; allOf?: InputSchema[] };
    function requiredFields(schema: InputSchema): string[] {
        const referenced = schema.$ref
            ? spec.components?.schemas?.[schema.$ref.split("/").at(-1)!]
            : undefined;
        return [
            ...(schema.required ?? []),
            ...(referenced ? requiredFields(referenced) : []),
            ...(schema.allOf ?? []).flatMap(requiredFields),
        ];
    }
    function requestInput(path: string, contentType: string): InputSchema {
        const body = spec.paths?.[path]?.post?.requestBody;
        if (!body || !("content" in body)) throw new Error(`Missing request body for ${path}`);
        const schema = body.content[contentType]?.schema;
        if (!schema) throw new Error(`Missing ${contentType} schema for ${path}`);
        return schema;
    }

    const input = requestInput("/api/v1/trail/{id}", "application/json");
    const runtimeRequired = Object.entries(TrailUpdateSchema.shape)
        .filter(([, field]) => !field.isOptional())
        .map(([name]) => name);
    expect([...new Set(requiredFields(input))].sort()).toEqual(runtimeRequired.sort());

    for (const path of ["/api/v1/trail/{id}/file", "/api/v1/trail/form/{id}"]) {
        const multipartInput = requestInput(path, "multipart/form-data");
        expect(multipartInput).toEqual({ $ref: "#/components/schemas/TrailUpdateInput" });
        expect(requiredFields(multipartInput)).not.toContain("name");
    }
});
