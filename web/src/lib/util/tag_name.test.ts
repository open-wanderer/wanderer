import { describe, expect, it } from "vitest";
import { normalizeNewTagName, normalizeTagName } from "./tag_name";

describe("tag-name normalization", () => {
    it.each([
        ["all whitespace controls", "a\tb\nc\vd\fe\rf", "a b c d e f"],
        ["CRLF", "a\r\nb", "a  b"],
        ["other C0 and DEL", "a\u0000\u0008\u000e\u001f\u007fb", "ab"],
        ["repeated spaces", "  a\t\tb  ", "  a  b  "],
        ["Unicode and literal text", "  e\u0301 👩‍👩‍👧‍👦 <b> &amp; \u0085\u00a0\u2028  ", "  e\u0301 👩‍👩‍👧‍👦 <b> &amp; \u0085\u00a0\u2028  "],
        ["empty API name", "\u0000\u007f", ""],
    ])("preserves the contract for %s", (_label, input, expected) => {
        expect(normalizeTagName(input)).toBe(expected);
        expect(normalizeTagName(expected)).toBe(expected);
    });

    it.each(["", "   ", "\u0000\u007f", "\t\n\v\f\r"])("skips an empty new UI tag", input => {
        expect(normalizeNewTagName(input)).toBeNull();
    });

    it.each(["\u00a0", "\u0085", "\u2028", "  word\tword  "])("retains non-ASCII whitespace and meaningful spacing", input => {
        expect(normalizeNewTagName(input)).toBe(normalizeTagName(input));
    });
});
