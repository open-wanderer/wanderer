export type TextMatchSegment = { text: string; matched: boolean };

// Literal case-insensitive matches. Map folded character boundaries back to
// the source so Unicode case expansion never shifts or changes displayed text.
export function textMatchSegments(text: string, query: string): TextMatchSegment[] {
    if (!query) return [{ text, matched: false }];
    let foldedText = "";
    const offsets = [0];
    let sourceOffset = 0;
    for (const character of text) {
        const folded = character.toLowerCase();
        foldedText += folded;
        sourceOffset += character.length;
        for (let i = 1; i < folded.length; i++) offsets.push(-1);
        offsets.push(sourceOffset);
    }
    const foldedQuery = Array.from(query, character => character.toLowerCase()).join("");
    const segments: TextMatchSegment[] = [];
    let cursor = 0;
    let searchFrom = 0;
    while (searchFrom < foldedText.length) {
        const index = foldedText.indexOf(foldedQuery, searchFrom);
        if (index < 0) break;
        const start = offsets[index];
        const end = offsets[index + foldedQuery.length];
        searchFrom = index + 1;
        if (start < 0 || end < 0) continue;
        if (start > cursor) segments.push({ text: text.slice(cursor, start), matched: false });
        segments.push({ text: text.slice(start, end), matched: true });
        cursor = end;
        searchFrom = index + foldedQuery.length;
    }
    if (cursor < text.length || !segments.length) segments.push({ text: text.slice(cursor), matched: false });
    return segments;
}
