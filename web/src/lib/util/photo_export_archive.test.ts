import JSZip from "jszip";
import { afterEach, expect, it, vi } from "vitest";
import { appendExportPhotos, addMissingPhotoReport, missingExportPhotoNames } from "./photo_export_archive";

afterEach(() => vi.useRealTimers());
it.each(["single trail", "account"])("finishes the actual %s photo loop with GPX and readable photos despite unavailable media", async (kind) => {
    const zip = new JSZip();
    const tracks = kind === "account" ? ["First trail", "Second trail"] : ["First trail"];
    const missing = [];
    for (const name of tracks) {
        const folder = kind === "account" ? zip.folder(name)! : zip;
        folder.file(`${name}.gpx`, "<gpx/>");
        const request = vi.fn(async (url: RequestInfo | URL) => {
            if (String(url).includes("good")) return new Response("real original", { headers: { "Content-Type": "image/jpeg" } });
            const code = String(url).includes("busy") ? "asset_media_busy" : "not_found";
            return Response.json({ status: code === "asset_media_busy" ? 503 : 404, message: code, data: {} }, { status: code === "asset_media_busy" ? 503 : 404 });
        });
        const pending = appendExportPhotos(folder.folder("photos"), { name, photos: ["/missing.jpg", "/good.jpg", "/busy.jpg", "/good.jpg"] }, new Set(), request);
        missing.push(...await pending);
        expect(request.mock.calls.filter(([url]) => String(url).includes("busy"))).toHaveLength(1);
    }
    addMissingPhotoReport(zip, missing);
    const result = await JSZip.loadAsync(await zip.generateAsync({ type: "uint8array" }));
    for (const name of tracks) {
        const prefix = kind === "account" ? `${name}/` : "";
        expect(await result.file(`${prefix}${name}.gpx`)!.async("string")).toBe("<gpx/>");
        expect(await result.file(`${prefix}photos/good.jpg`)!.async("string")).toBe("real original");
        expect(await result.file(`${prefix}photos/good-2.jpg`)!.async("string")).toBe("real original");
        expect(result.file(`${prefix}photos/missing.jpg`)).toBeNull();
    }
    expect(await result.file("missing-photos.txt")!.async("string")).toContain("First trail: missing.jpg");
    expect(missing).toHaveLength(tracks.length * 2);
});


it("bounds export warning names while the archive report retains every entry", async () => {
    const { addMessages, init, locale } = await import("svelte-i18n");
    const { get } = await import("svelte/store");
    const { _ } = await import("svelte-i18n");
    const en = (await import("#lib/i18n/locales/en.json")).default;
    const de = (await import("#lib/i18n/locales/de.json")).default;
    addMessages("en", en); addMessages("de", de); init({ initialLocale: "en", fallbackLocale: "en" });
    for (const language of ["en", "de"]) {
        locale.set(language);
        const missing = Array.from({ length: 20 }, (_, i) => ({ trail: "Trail", filename: i === 0 ? "asset0000000001.bin" : `photo-${i}.jpg` }));
        const names = missingExportPhotoNames(missing);
        expect(names).not.toContain(".bin");
        expect(names).toContain(language === "en" ? "and 15 more" : "und 15 weitere");
        expect(names).not.toContain("photo-5.jpg");
        expect(get(_)("export-photos-skipped", { values: { n: 1, names: "summit.jpg" } })).toContain(language === "en" ? "1 photo was skipped" : "1 Foto wurde ausgelassen");
        const zip = new JSZip(); addMissingPhotoReport(zip, missing);
        expect(await zip.file("missing-photos.txt")!.async("string")).toContain("photo-19.jpg");
    }
});


it("keeps unknown exported photo identities distinguishable in the bounded warning", () => {
    const names = missingExportPhotoNames([{ trail: "Hike", filename: "000000000000001.bin" }, { trail: "Hike", filename: "000000000000002.bin" }]);
    expect(names).toContain("000000000000001");
    expect(names).toContain("000000000000002");
});
