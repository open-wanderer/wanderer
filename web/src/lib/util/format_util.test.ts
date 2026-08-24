import { describe, expect, it, vi } from "vitest";

const appState = vi.hoisted(() => ({ settings: { unit: "metric" } }));

vi.mock("$app/state", () => ({
    page: { data: { settings: appState.settings } },
}));

import {
    formatHTMLAsText,
    formatHTMLAsTextPreview,
    formatSpeed,
} from "./format_util";

describe("formatHTMLAsText", () => {
    it("returns an empty string for missing input", () => {
        expect(formatHTMLAsText()).toBe("");
        expect(formatHTMLAsText("")).toBe("");
    });

    it("strips inline markup but keeps the text", () => {
        expect(formatHTMLAsText("<p>a <strong>bold</strong> word</p>")).toBe(
            "a bold word",
        );
    });

    it("separates block elements by a blank line", () => {
        expect(formatHTMLAsText("<p>one</p><p>two</p>")).toBe("one\n\ntwo");
        expect(formatHTMLAsText("<ul><li>one</li><li>two</li></ul>")).toBe(
            "one\n\ntwo",
        );
    });

    it("turns line breaks into single newlines", () => {
        expect(formatHTMLAsText("one<br>two<br />three")).toBe(
            "one\ntwo\nthree",
        );
    });

    it("keeps torn markup out of the output", () => {
        // What `description.substring(0, 100)` used to produce (#1128)
        expect(formatHTMLAsText("<p>text <strong>bold and it")).toBe(
            "text bold and it",
        );
    });

    it("survives attribute values containing angle brackets", () => {
        expect(formatHTMLAsText('<a href="/x?a=1&b=2" title="a > b">link</a>')).toBe(
            "link",
        );
    });

    it("drops script and style content", () => {
        expect(
            formatHTMLAsText("<p>keep</p><script>alert('x')</script>"),
        ).toBe("keep");
        expect(formatHTMLAsText("<style>p { color: red; }</style>keep")).toBe(
            "keep",
        );
    });

    it("decodes named and numeric entities", () => {
        expect(formatHTMLAsText("a &amp; b")).toBe("a & b");
        expect(formatHTMLAsText("&lt;p&gt;")).toBe("<p>");
        expect(formatHTMLAsText("&quot;q&quot; &#39;a&#39; &#x27;b&#x27;")).toBe(
            "\"q\" 'a' 'b'",
        );
        expect(formatHTMLAsText("caf&#233; &#x2014; open")).toBe("café — open");
    });

    it("decodes entities exactly once", () => {
        // The author wrote "&lt;p&gt;", which the editor stored as "&amp;lt;p&amp;gt;".
        // Decoding twice would turn that back into a tag and lose it.
        expect(formatHTMLAsText("<p>&amp;lt;p&amp;gt; stays</p>")).toBe(
            "&lt;p&gt; stays",
        );
    });

    it("leaves unknown entities alone", () => {
        expect(formatHTMLAsText("&unknownentity; &#xZZ;")).toBe(
            "&unknownentity; &#xZZ;",
        );
    });

    it("collapses redundant whitespace", () => {
        expect(formatHTMLAsText("<p>a</p><p></p><p></p><p>b</p>")).toBe(
            "a\n\nb",
        );
        expect(formatHTMLAsText("<p>  padded  </p>")).toBe("padded");
        expect(formatHTMLAsText("a    b")).toBe("a  b");
    });
});

describe("formatHTMLAsTextPreview", () => {
    it("reports untruncated text unchanged", () => {
        const result = formatHTMLAsTextPreview("<p>short</p>", 100);

        expect(result).toEqual({ text: "short", truncated: false });
    });

    it("truncates to the visible character count, not the HTML length", () => {
        // 30 characters of text behind far more than 30 characters of markup
        const html = `<p><strong><em>${"a".repeat(30)}</em></strong></p>`;
        const result = formatHTMLAsTextPreview(html, 10);

        expect(result).toEqual({ text: "a".repeat(10), truncated: true });
    });

    it("counts code points so the cutoff never splits an astral character", () => {
        const result = formatHTMLAsTextPreview("<p>🏔️🏔️🏔️</p>", 2);

        expect(result.truncated).toBe(true);
        expect(Array.from(result.text)).toHaveLength(2);
        expect(result.text).toBe("🏔️");
    });

    it("treats a boundary-length description as untruncated", () => {
        const result = formatHTMLAsTextPreview(`<p>${"a".repeat(100)}</p>`, 100);

        expect(result.truncated).toBe(false);
    });

    it("handles missing descriptions", () => {
        expect(formatHTMLAsTextPreview(undefined, 100)).toEqual({
            text: "",
            truncated: false,
        });
    });
});

describe("formatSpeed", () => {
    it("formats routing speeds in the user's configured unit", () => {
        appState.settings.unit = "metric";
        expect(formatSpeed(5.1 / 3.6)).toBe("5.1 km/h");

        appState.settings.unit = "imperial";
        expect(formatSpeed(5.1 / 3.6)).toBe("3.2 mph");
    });
});
