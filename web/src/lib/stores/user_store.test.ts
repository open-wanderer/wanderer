import { get } from "svelte/store";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "$lib/models/user";
import { APIError } from "$lib/util/api_util";
import { currentUser, users_update } from "./user_store";

vi.mock("$lib/pocketbase", () => ({ getPb: vi.fn() }));

const user = {
    id: "user00000000001",
    username: "tester",
    avatar: "old-avatar.png",
} as User;

describe("users_update error responses", () => {
    beforeEach(() => {
        currentUser.set(user);
    });

    afterEach(() => {
        vi.unstubAllGlobals();
        currentUser.set(null);
    });

    it.each([
        [413, "<html>Request too large</html>"],
        [429, ""],
        [503, "null"],
        [403, "[]"],
    ])("preserves HTTP %s when the upload error body is not usable JSON", async (status, body) => {
        const fetchMock = vi.fn()
            .mockResolvedValueOnce(new Response(JSON.stringify(user)))
            .mockResolvedValueOnce(new Response(body, { status }));
        vi.stubGlobal("fetch", fetchMock);

        const error = await users_update(user, new File(["image"], "avatar.png"))
            .catch(error => error);

        expect(error).toBeInstanceOf(APIError);
        expect(error.status).toBe(status);
        expect(error.detail).toBeUndefined();
        expect(get(currentUser)).toEqual(user);
        expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    it("preserves the status of a non-JSON error before the file request", async () => {
        const fetchMock = vi.fn().mockResolvedValue(new Response("Unavailable", { status: 503 }));
        vi.stubGlobal("fetch", fetchMock);

        const error = await users_update(user, new File(["image"], "avatar.png"))
            .catch(error => error);

        expect(error).toBeInstanceOf(APIError);
        expect(error.status).toBe(503);
        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(get(currentUser)).toEqual(user);
    });

    it("retains structured avatar validation parameters", async () => {
        const detail = {
            data: {
                avatar: {
                    code: "validation_file_size_limit",
                    params: { maxSize: 5242880 },
                },
            },
        };
        vi.stubGlobal("fetch", vi.fn()
            .mockResolvedValueOnce(new Response(JSON.stringify(user)))
            .mockResolvedValueOnce(new Response(JSON.stringify({ message: "Invalid image", detail }), { status: 400 })));

        const error = await users_update(user, new File(["image"], "avatar.png"))
            .catch(error => error);

        expect(error).toBeInstanceOf(APIError);
        expect(error.detail).toEqual(detail);
        expect(get(currentUser)).toEqual(user);
    });
});
