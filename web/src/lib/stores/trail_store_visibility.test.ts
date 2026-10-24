import { afterEach, describe, expect, it, vi } from "vitest";
import type { TrailFilter } from "$lib/models/trail";
import type { User } from "$lib/models/user";
import { trails_search_filter } from "./trail_store";
import { currentUser } from "./user_store";

const filter: TrailFilter = {
    q: "", category: [], subcategory: [], tags: [], difficulty: [],
    near: { radius: 2_000 },
    distanceMin: 0, distanceMax: 50_000, distanceLimit: 50_000,
    elevationGainMin: 0, elevationGainMax: 2_000, elevationGainLimit: 2_000,
    elevationLossMin: 0, elevationLossMax: 2_000, elevationLossLimit: 2_000,
    sort: "created", sortOrder: "+",
};
const userFixture: User = {
    id: "user1", actor: "actor1", collectionId: "users", collectionName: "users", password: "test-only",
};

describe("trail search visibility", () => {
    afterEach(() => currentUser.set(null));

    it.each([
        { private: false, shared: false, expected: "public = TRUE AND public = FALSE" },
        { private: false, shared: true, expected: "AND (shares = actor1)" },
        { private: true, shared: false, expected: "public = FALSE AND author = actor1" },
    ])("preserves explicit visibility choices: %j", async ({ private: privateChoice, shared, expected }) => {
        currentUser.set(userFixture);
        const request = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => Response.json({
            hits: [], page: 1, hitsPerPage: 48, totalPages: 0, totalHits: 0,
        }));
        await trails_search_filter({ ...filter, public: false, private: privateChoice, shared }, 1, 48, request);
        const expression = JSON.parse(String(request.mock.calls[0][1]?.body)).options.filter;
        expect(expression).toContain(expected);
        if (!privateChoice) expect(expression).not.toContain("public = FALSE AND author = actor1");
    });
});
