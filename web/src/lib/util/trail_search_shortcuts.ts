import type { ActorSearchResult } from "$lib/models/activitypub/actor";
import type { Tag } from "$lib/models/tag";

export type TrailSearchShortcut = {
    kind: "author" | "tag";
    start: number;
    end: number;
    query: string;
};

export type TrailSearchShortcutChoice =
    | { kind: "author"; author: ActorSearchResult }
    | { kind: "tag"; tag: Tag };

/** Only the expression at the caret is an input aid; the query stays literal. */
export function findTrailSearchShortcut(value: string, caret = value.length, selectionEnd = caret): TrailSearchShortcut | undefined {
    if (caret !== selectionEnd || caret < 0 || caret > value.length) return;
    const matches = [...value.matchAll(/(^|\s)(@|#)/g)];
    for (let index = matches.length - 1; index >= 0; index--) {
        const match = matches[index];
        const start = match.index! + match[1].length;
        const queryStart = start + match[2].length;
        if (queryStart > caret) continue;
        const kind = match[2] === "@" ? "author" : "tag";
        const whitespace = value.slice(queryStart).search(/\s/);
        const end = whitespace === -1 ? value.length : queryStart + whitespace;
        if (caret > end) continue;
        return { kind, start, end, query: value.slice(queryStart, caret).trim() };
    }
}

/** Remove a confirmed expression while retaining text on both sides. */
export function consumeTrailSearchShortcut(value: string, shortcut: TrailSearchShortcut) {
    const before = value.slice(0, shortcut.start).trimEnd();
    const after = value.slice(shortcut.end).trimStart();
    const separator = before && after ? " " : "";
    return {
        value: before + separator + after,
        caret: before.length + separator.length,
    };
}

export function trailSearchAuthorHandle(author: ActorSearchResult) {
    return `@${author.preferred_username}${!author.is_local && author.domain ? `@${author.domain}` : ""}`;
}
