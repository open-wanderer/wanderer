import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { afterEach, beforeAll, expect, it, vi } from "vitest";
import messages from "#lib/i18n/locales/en.json";
import Settings from "./plugin_instance_settings_modal.svelte";
import type { PluginProvider } from "#lib/models/plugin_provider.ts";

vi.mock("$app/env/public", () => ({
    PUBLIC_POCKETBASE_URL: "https://a.example",
    PUBLIC_MAP_MAX_POLYLINES: undefined,
    PUBLIC_PRIVATE_INSTANCE: undefined,
}));
vi.mock("#lib/stores/theme_store.ts", async () => {
    const { writable } = await import("svelte/store");
    return { theme: writable("light") };
});

const mounted: ReturnType<typeof mount>[] = [];
beforeAll(() => { addMessages("en", messages); init({ initialLocale: "en", fallbackLocale: "en" }); });
afterEach(async () => {
    await Promise.all(mounted.splice(0).map((component) => unmount(component)));
    document.body.replaceChildren();
    vi.restoreAllMocks();
    Reflect.deleteProperty(HTMLDialogElement.prototype, "showModal");
    Reflect.deleteProperty(HTMLDialogElement.prototype, "close");
});

it("keeps a remap dismissal when the settings dialog saves unchanged configuration", async () => {
    Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value() { this.open = true; } });
    Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value() { this.open = false; } });
    vi.spyOn(window, "scrollTo").mockImplementation(() => {});
    const timestamp = "2026-10-03T12:00:00Z";
    const onsave = vi.fn();
    const target = document.createElement("div");
    document.body.append(target);
    const component = mount(Settings, { target, props: {
        plugin: { id: "komoot", type: "trails", name: "komoot", auth: { type: "none" }, status: "available" } as PluginProvider,
        instance: { id: "instance", enabled: true, config: { host: { categoryRemapDismissedAt: timestamp } } } as never,
        onsave,
    } });
    mounted.push(component);
    flushSync();
    component.openModal();
    flushSync();
    const save = [...target.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === "Save")!;
    save.click();
    await vi.waitFor(() => expect(onsave).toHaveBeenCalledOnce());
    expect(onsave.mock.calls[0][0].config.host.categoryRemapDismissedAt).toBe(timestamp);
});
