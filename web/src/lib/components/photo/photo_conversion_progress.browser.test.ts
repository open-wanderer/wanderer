import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { expect, it } from "vitest";
import Harness from "./photo_conversion_progress_harness.svelte";

it("keeps its live region mounted before, during and after photo processing", async () => {
    addMessages("en", { "photo-conversion-progress": "Processing {current}/{total}: {file}" });
    init({ fallbackLocale: "en", initialLocale: "en" });
    const target = document.createElement("div");
    document.body.append(target);
    const component = mount(Harness, { target });
    try {
        flushSync();
        const region = target.querySelector('[role="status"]');
        expect(region).not.toBeNull();
        expect(region?.textContent?.trim()).toBe("");
        component.setProgress({ file: "first.heic", current: 1, total: 2 });
        flushSync();
        expect(target.querySelector('[role="status"]')).toBe(region);
        expect(region?.textContent?.trim()).toBe("Processing 1/2: first.heic");
        component.setProgress({ file: "second.heic", current: 2, total: 2 });
        flushSync();
        expect(region?.textContent?.trim()).toBe("Processing 2/2: second.heic");
        component.setProgress(undefined);
        flushSync();
        expect(target.querySelector('[role="status"]')).toBe(region);
        expect(region?.textContent?.trim()).toBe("");
    } finally {
        await unmount(component);
        target.remove();
    }
});
