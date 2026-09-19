import { StreamLanguage, type StringStream } from "@codemirror/language";
import type { Extension } from "@codemirror/state";

export interface CodeEditorLanguageDefinition {
    id: string;
    name?: string;
    caseSensitive?: boolean;
    lineComment?: string;
    keywords?: string[];
    atoms?: string[];
    builtins?: string[];
    directives?: string[];
}

export function codeEditorLanguageDefinition(
    value: unknown,
): CodeEditorLanguageDefinition | undefined {
    if (!value || typeof value !== "object" || Array.isArray(value)) return undefined;
    const input = value as Record<string, unknown>;
    const id = safeString(input.id, 64);
    if (!id) return undefined;

    return {
        id,
        name: safeString(input.name, 64) || undefined,
        caseSensitive:
            typeof input.caseSensitive === "boolean" ? input.caseSensitive : undefined,
        lineComment: safeString(input.lineComment, 8) || undefined,
        keywords: stringList(input.keywords),
        atoms: stringList(input.atoms),
        builtins: stringList(input.builtins),
        directives: stringList(input.directives),
    };
}

export function codeEditorLanguage(
    definition?: CodeEditorLanguageDefinition,
): Extension {
    definition = codeEditorLanguageDefinition(definition);
    if (!definition) return [];
    const caseSensitive = definition.caseSensitive !== false;
    const normalized = (word: string) => (caseSensitive ? word : word.toLowerCase());
    const keywords = wordSet(definition.keywords, normalized);
    const atoms = wordSet(definition.atoms, normalized);
    const builtins = wordSet(definition.builtins, normalized);
    const directives = stringList(definition.directives);
    const lineComment = safeString(definition.lineComment, 8);

    return StreamLanguage.define<null>({
        name: safeString(definition.id, 64) || "profile",
        startState: () => null,
        languageData: lineComment
            ? { commentTokens: { line: lineComment } }
            : undefined,
        token(stream: StringStream) {
            if (stream.eatSpace()) return null;
            if (lineComment && stream.match(lineComment)) {
                stream.skipToEnd();
                return "comment";
            }
            for (const directive of directives) {
                if (stream.match(directive, false, !caseSensitive)) {
                    stream.match(/^\S+/);
                    return "meta";
                }
            }
            if (stream.match(/^"(?:[^"\\]|\\.)*(?:"|$)/)) return "string";
            if (stream.match(/^'(?:[^'\\]|\\.)*(?:'|$)/)) return "string";
            if (stream.match(/^[+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?\b/i)) {
                return "number";
            }
            if (stream.match(/^[A-Za-z_][\w:-]*/)) {
                const candidate = normalized(stream.current());
                if (keywords.has(candidate)) return "keyword";
                if (atoms.has(candidate)) return "bool";
                if (builtins.has(candidate)) return "typeName";
                return null;
            }
            if (stream.match(/^[=<>!+*/|&-]+/)) return "operator";
            stream.next();
            return null;
        },
    });
}

function wordSet(
    values: unknown,
    normalize: (value: string) => string,
): Set<string> {
    return new Set(stringList(values).map(normalize));
}

function stringList(value: unknown): string[] {
    if (!Array.isArray(value)) return [];
    return value
        .filter((item): item is string => typeof item === "string")
        .map((item) => safeString(item, 64))
        .filter(Boolean)
        .slice(0, 256);
}

function safeString(value: unknown, maxLength: number): string {
    return typeof value === "string" ? value.trim().slice(0, maxLength) : "";
}
