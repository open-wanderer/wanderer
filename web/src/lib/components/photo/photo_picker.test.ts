import { render } from "svelte/server";
import { addMessages, init } from "svelte-i18n";
import { beforeAll, describe, expect, it } from "vitest";
import type { Asset } from "#lib/models/asset.ts";
import PhotoPicker from "./photo_picker.svelte";

beforeAll(() => {
    addMessages("en", {});
    init({ fallbackLocale: "en", initialLocale: "en" });
});

function asset(name: string, metadata?: Record<string, unknown>): Asset {
    return {
        id: name.padEnd(15, "0"), collectionId: "assets", collectionName: "assets",
        type: "photo", author: "actor", file: `${name}.webp`, metadata,
    };
}

const preview = asset("preview", { generated: { kind: "route-preview" } });
const summit = asset("summit");
const photoURL = (value: Asset) => `/api/v1/files/${value.collectionId}/${value.id}/${value.file}`;

// Tells for each photo card, in order, whether it offers "make thumbnail".
function thumbnailControls(photos: Asset[], props: Record<string, unknown> = {}): boolean[] {
    const { body } = render(PhotoPicker, {
        props: {
            id: "trail",
            photos: photos.map(photoURL),
            photoFiles: [],
            parent: { expand: { assets_via_trail: [preview, summit] } },
            ...props,
        },
    });
    return body
        .split("background-image: url(")
        .slice(1)
        .map((card) => card.includes('aria-label="Make thumbnail"'));
}

describe("PhotoPicker", () => {
    it("offers a route preview as thumbnail only while there is no other photo", () => {

        expect(thumbnailControls([summit, preview]), "next to a saved photo").toEqual([true, false]);
        expect(thumbnailControls([preview]), "on its own").toEqual([true]);
        expect(thumbnailControls([preview], { photoFiles: [new File(["photo"], "summit.jpg")] }), "next to an upload")
            .toEqual([false]);
        expect(thumbnailControls([summit, preview], { showThumbnailControls: false }), "without thumbnail controls")
            .toEqual([false, false]);
    });
});
