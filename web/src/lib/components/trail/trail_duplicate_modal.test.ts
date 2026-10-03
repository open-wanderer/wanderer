import { render } from "svelte/server";
import { addMessages, init } from "svelte-i18n";
import { beforeAll, describe, expect, it } from "vitest";
import ModalPair from "./trail_duplicate_modal_pair.fixture.svelte";

beforeAll(() => {
    addMessages("en", {});
    init({ fallbackLocale: "en", initialLocale: "en" });
});

describe("TrailDuplicateModal", () => {
    it("gives every instance its own dialog id", () => {
        const { body } = render(ModalPair);
        const ids = [...body.matchAll(/<dialog[^>]*\sid="([^"]+)"/g)].map((match) => match[1]);

        expect(ids).toHaveLength(2);
        expect(new Set(ids).size).toBe(2);
    });
});
