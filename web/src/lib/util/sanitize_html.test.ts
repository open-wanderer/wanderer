import { describe, expect, it } from "vitest";
import { sanitizeHTML } from "./sanitize_html";

describe("safe rich-text rendering", () => {
    it("keeps formatted international text and safe links", () => {
        const input = '<p>Grüezi <strong>山歩き 🚲</strong> &amp; friends</p><a class="mention" href="https://example.test/profile">@friend</a>';
        expect(sanitizeHTML(input)).toBe(input);
        expect(sanitizeHTML(sanitizeHTML(input))).toBe(sanitizeHTML(input));
    });

    it("sanitizes new and previously cached profile HTML on the server", () => {
        const input = '<p onclick="blocked()">Profile</p><img src="x" onerror="blocked()"><script>blocked()</script><a href="javascript:blocked()">Link</a>';
        const safe = sanitizeHTML(input);
        expect(safe).toContain("Profile");
        expect(safe).toContain("Link");
        expect(safe).not.toMatch(/onclick|onerror|<script|javascript:|blocked\(\)/i);
    });

    it.each([
        '<svg><g onload="blocked()"></g></svg>',
        '<math><mtext><table><mglyph><style><!--</style><img title="--><img src=x onerror=blocked()>">',
        '<a href="&#x6a;avascript:blocked()">Link</a>',
        '<iframe srcdoc="<script>blocked()</script>"></iframe>',
    ])("rejects foreign namespaces and unsafe parser variants", (input) => {
        expect(sanitizeHTML(input)).not.toMatch(/<svg|<math|<iframe|<style|onerror=|onload=|href="javascript:/i);
    });

    it("does not render injected controls or inline styles", () => {
        expect(sanitizeHTML('<form><input name="password"><button>Submit</button></form><p style="position:fixed">Text</p>'))
            .not.toMatch(/<form|<input|<button|style=/i);
        expect(sanitizeHTML(null)).toBe("");
    });
});
