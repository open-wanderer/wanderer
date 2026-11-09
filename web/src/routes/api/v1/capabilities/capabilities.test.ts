import { expect, it } from "vitest";
import { GET } from "./+server";

it("advertises the asset API without authentication", async () => {
 const response = GET();
 expect(response.status).toBe(200);
 expect(response.headers.get("content-type")).toContain("application/json");
 expect(await response.json()).toEqual({assetApi:true});
});
