import { describe, expect, it } from "vitest";
import { ClientResponseError } from "pocketbase";
import { APIError } from "./api_util";
import { remoteContentError } from "./remote_content_error";
import { GET as trailGET } from "../../routes/api/v1/trail/[id]/+server";
import { GET as listGET } from "../../routes/api/v1/list/[id]/+server";
import type { RequestEvent } from "@sveltejs/kit";

const message = "This content is too large to load here. You can view it on its original instance.";

describe("remote content errors", () => {
    it.each(["https://www.remote.example:8443/trail/view/@alice/trail0000000001", "http://remote.example:8080/lists/@alice/list00000000001"])("offers the original view %s", async (originURL) => {
        const error = new APIError(422, message, { code: "remote_document_limit", origin_url: originURL });
        expect(remoteContentError(error)).toEqual({ message, originURL });
    });

    it.each(["javascript:alert(1)", "data:text/html,text", "//remote.example/lists/@alice/id", "https://user:pass@remote.example/lists/@alice/id", "https://remote.example/api/v1/list/id", "invalid"])("omits unsupported original links %s", (originURL) => {
        expect(remoteContentError(new APIError(422, message, { code: "remote_document_limit", origin_url: originURL }))).toEqual({ message });
    });

    it.each([400, 404, 500, 503])("leaves ordinary status %s without a link", (status) => {
        expect(remoteContentError(new APIError(status, message, { code: "remote_document_limit", origin_url: "https://remote.example/lists/@alice/id" }))).toEqual({ message: status === 404 ? "Not found" : message });
    });

    it.each([trailGET, listGET])("carries the PocketBase response through the API route", async (GET) => {
        const detail = { code: "remote_document_limit", origin_url: "https://remote.example/lists/@alice/list00000000001" };
        const event = {
            params: { id: "list00000000001" }, url: new URL("https://local.example/api/v1/list/list00000000001?handle=@alice@remote.example"),
            request: new Request("https://local.example/api/v1/list/list00000000001"),
            locals: { pb: { send: async () => { throw new ClientResponseError({ status: 422, response: { status: 422, message, data: detail } }); } } },
        } as unknown as RequestEvent;
        const response = await GET(event);
        expect(response.status).toBe(422);
        const body = await response.json();
        expect(body.message).toBe(message);
        expect(body.data).toEqual(detail);
        expect(remoteContentError(new APIError(response.status, body.message, body.detail ?? body.data))).toEqual({ message, originURL: detail.origin_url });
    });
});
