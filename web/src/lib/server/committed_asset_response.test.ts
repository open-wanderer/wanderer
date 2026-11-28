import { expect, it } from "vitest";
import { ClientResponseError } from "pocketbase";
import { committedAssetResponseError } from "./committed_asset_response.ts";
import { responseOverHttp } from "./response_wire_test_util.ts";

it("delivers the committed identity and complete JSON through the Node adapter", async () => {
    const error = new ClientResponseError({ status: 503, response: { message: "refresh failed", data: {} } });
    const record = { id: "trail0000000001", name: "Saved trail", photos: ["incomplete"], expand: { incomplete: true } };
    const response = await responseOverHttp(await committedAssetResponseError(error, record));
    expect(response.status).toBe(503);
    expect(response.headers.get("Content-Type")).toContain("application/json");
    const body = await response.text();
    expect(Number(response.headers.get("Content-Length"))).toBe(Buffer.byteLength(body));
    expect(JSON.parse(body)).toMatchObject({ detail: { recordCommitted: true, savedRecord: { id: record.id, name: record.name } } });
    expect(JSON.parse(body).detail.savedRecord).not.toHaveProperty("photos");
});
