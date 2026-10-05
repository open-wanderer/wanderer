import DOMPurify from "isomorphic-dompurify";

// The same parser-backed sanitizer runs during SSR and immediately before
// rendering cached/provider content in the browser. Sanitize after Markdown.
export function sanitizeHTML(value: unknown): string {
    return DOMPurify.sanitize(typeof value === "string" ? value : "", {
        USE_PROFILES: { html: true },
        FORBID_TAGS: ["style", "form", "input", "button", "select", "textarea"],
        FORBID_ATTR: ["style"],
    });
}
