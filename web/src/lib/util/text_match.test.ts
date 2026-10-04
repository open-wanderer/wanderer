import { describe, expect, it } from "vitest";
import { textMatchSegments } from "./text_match";

describe("literal text matches", () => {
    it.each(["[route]", ".*", "(", "\\", "$", "+", "?"])("treats %s as text", query => {
        expect(textMatchSegments(`before ${query} after`, query)).toEqual([
            { text: "before ", matched: false }, { text: query, matched: true }, { text: " after", matched: false },
        ]);
    });

    it("finds repeated case-insensitive matches while preserving source spelling", () => {
        expect(textMatchSegments("ALP alpine alp", "alp")).toEqual([
            { text: "ALP", matched: true }, { text: " ", matched: false }, { text: "alp", matched: true },
            { text: "ine ", matched: false }, { text: "alp", matched: true },
        ]);
    });

    it.each(["", "missing"])("leaves unmatched text intact for query %s", query => {
        expect(textMatchSegments("<img> &lt;tag&gt; &amp;", query)).toEqual([
            { text: "<img> &lt;tag&gt; &amp;", matched: false },
        ]);
    });

    it("preserves Unicode offsets, combining marks and surrogate pairs", () => {
        expect(textMatchSegments("İstanbul 🌍 A", "a")).toEqual([
            { text: "İst", matched: false }, { text: "a", matched: true },
            { text: "nbul 🌍 ", matched: false }, { text: "A", matched: true },
        ]);
        for (const query of ["İ", "🌍", "e\u0301"]) {
            const text = `prefix ${query} suffix`;
            const segments = textMatchSegments(text, query);
            expect(segments.filter(segment => segment.matched).map(segment => segment.text)).toEqual([query]);
            expect(segments.map(segment => segment.text).join("")).toBe(text);
        }
    });

    it("handles empty text without empty-query loops", () => {
        expect(textMatchSegments("", "x")).toEqual([{ text: "", matched: false }]);
        expect(textMatchSegments("", "")).toEqual([{ text: "", matched: false }]);
    });
});
