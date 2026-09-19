import { syntaxTree } from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import { classHighlighter, highlightTree } from "@lezer/highlight";
import { describe, expect, it } from "vitest";
import {
    codeEditorLanguage,
    codeEditorLanguageDefinition,
} from "./code_editor_language";

describe("code editor language", () => {
    it("highlights a plugin-provided declarative language", () => {
        const document = [
            "---context:global",
            "assign validForBikes = true",
            "assign cost = multiply 2.5 # comment",
        ].join("\n");
        const state = EditorState.create({
            doc: document,
            extensions: [
                codeEditorLanguage({
                    id: "example",
                    lineComment: "#",
                    keywords: ["assign"],
                    atoms: ["true", "false"],
                    builtins: ["multiply"],
                    directives: ["---context:global"],
                }),
            ],
        });
        const highlighted = new Map<string, string>();

        highlightTree(syntaxTree(state), classHighlighter, (from, to, classes) => {
            highlighted.set(document.slice(from, to), classes);
        });

        expect(highlighted.get("---context:global")).toContain("tok-meta");
        expect(highlighted.get("assign")).toContain("tok-keyword");
        expect(highlighted.get("true")).toContain("tok-bool");
        expect(highlighted.get("multiply")).toContain("tok-typeName");
        expect(highlighted.get("2.5")).toContain("tok-number");
        expect(highlighted.get("# comment")).toContain("tok-comment");
    });

    it("sanitizes language metadata supplied by a plugin", () => {
        expect(
            codeEditorLanguageDefinition({
                id: " brouter ",
                name: 42,
                lineComment: "#",
                keywords: ["assign", 42, "if"],
            }),
        ).toEqual({
            id: "brouter",
            name: undefined,
            caseSensitive: undefined,
            lineComment: "#",
            keywords: ["assign", "if"],
            atoms: [],
            builtins: [],
            directives: [],
        });
        expect(codeEditorLanguageDefinition("brouter")).toBeUndefined();
    });
});
