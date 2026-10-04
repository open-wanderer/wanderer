// Remove C0 and DEL only; retain whitespace outside that range and all other
// Unicode, punctuation, literal markup and empty names.
export function normalizeTagName(name: string): string {
    return name.replace(/[\u0000-\u001f\u007f]/gu, "");
}
