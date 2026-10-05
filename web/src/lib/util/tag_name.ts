// Preserve word boundaries: each ASCII whitespace control becomes one space.
// Remove other C0/DEL without trimming, collapsing or changing other Unicode.
export function normalizeTagName(name: string): string {
    return name.replace(/[\u0009-\u000d]/gu, " ").replace(/[\u0000-\u001f\u007f]/gu, "");
}

// Skip empty new UI tags only. Existing tags and API empty names stay valid.
export function normalizeNewTagName(name: string): string | null {
    const normalized = normalizeTagName(name);
    return /^ *$/u.test(normalized) ? null : normalized;
}
