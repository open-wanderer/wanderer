import { afterEach, describe, expect, it, vi } from "vitest";
import { notifications_index } from "./notification_store";

afterEach(() => {
    vi.restoreAllMocks();
});

describe("notifications_index", () => {
    it("filters by recipient without excluding notifications by month", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({
            items: [],
            page: 1,
            perPage: 10,
            totalItems: 0,
            totalPages: 0,
        }));

        await notifications_index({ recipient: "actor123" }, 1, 10, request);

        const [url] = request.mock.calls[0];
        const filter = new URL(String(url), "http://localhost").searchParams.get("filter");
        expect(filter).toBe("recipient='actor123'");
    });

    it("keeps the seen filter without imposing a month cutoff", async () => {
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({
            items: [],
            page: 1,
            perPage: 10,
            totalItems: 0,
            totalPages: 0,
        }));

        await notifications_index({ recipient: "actor123", seen: false }, 1, 10, request);

        const [url] = request.mock.calls[0];
        const filter = new URL(String(url), "http://localhost").searchParams.get("filter");
        expect(filter).toBe("recipient='actor123'&&seen=false");
    });
});
