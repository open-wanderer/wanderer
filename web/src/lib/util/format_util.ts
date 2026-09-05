import { page } from "$app/state";

export function formatTimeHHMM(seconds?: number) {
    if (seconds == null || isNaN(seconds)) {
        return "-";
    }

    const totalMinutes = Math.floor(seconds / 60);
    const m = totalMinutes % 60;
    const h = Math.floor(totalMinutes / 60);

    return (h < 10 ? "0" : "") + h.toString() + "h " + (m < 10 ? "0" : "") + m.toString() + "m";
}

/**
 * The single implementation of the display rule: show `moving_duration`
 * (moving time, set only by app recordings) when present, otherwise fall
 * back to `duration` (GPX-derived elapsed time, always present when the
 * trail has a route). Nothing may write `moving_duration` from a GPX
 * recompute -- this function only reads it for display.
 *
 * Typed structurally (not as `Trail`) so partially-expanded records (e.g.
 * feed/search results) can be passed without importing the full model and
 * risking an import cycle.
 */
export function trailDisplayDuration(trail: {
    duration?: number;
    moving_duration?: number;
}): number | undefined {
    return trail.moving_duration != null && trail.moving_duration > 0
        ? trail.moving_duration
        : trail.duration;
}

export function formatDistance(
    meters?: number,
    options: { compact?: boolean } = {},
) {
    if (meters === undefined) {
        return "-";
    }

    const unit = page.data.settings?.unit ?? "metric";

    if (unit == "metric") {
        if (meters >= 1000) {
            const kilometers = meters / 1000;
            if (options.compact && kilometers >= 100) {
                return `${kilometers.toFixed(0)} km`;
            }
            if (options.compact && kilometers >= 10) {
                return `${kilometers.toFixed(1)} km`;
            }
            return `${kilometers.toFixed(2)} km`
        } else {
            return meters % 1 == 0 ? `${meters} m` : `${Math.round(meters)} m`;
        }
    } else {
        const miles = meters * 0.000621371;
        const roundedMiles = miles.toFixed(2);

        return `${roundedMiles} mi`;
    }
}

export function formatElevation(meters?: number) {
    if (meters === undefined) {
        return "-";
    }

    const unit = page.data.settings?.unit ?? "metric";

    if (unit == "metric") {
        return `${Math.round(meters)} m`
    } else {
        const feet = meters * 3.28084;

        return `${Math.round(feet)} ft`;
    }
}

export function formatSpeed(speed?: number) {
    if (speed === undefined) {
        return "-";
    }

    const unit = page.data.settings?.unit ?? "metric";

    if (unit == "metric") {
        return `${(speed * 3.6).toFixed(2)} km/h`
    } else {
        const mph = speed * 3.6 * 0.621371;

        return `${Math.round(mph)} mp/h`;
    }
}

export function formatTimeSince(date: Date) {
    const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);

    let interval = seconds / 31536000;
    if (interval > 1) {
        return { unit: "years", value: Math.floor(interval) };
    }
    interval = seconds / 2592000;
    if (interval > 1) {
        return { unit: "months", value: Math.floor(interval) };
    }
    interval = seconds / 86400;
    if (interval > 1) {
        return { unit: "days", value: Math.floor(interval) };
    }
    interval = seconds / 3600;
    if (interval > 1) {
        return { unit: "hours", value: Math.floor(interval) };
    }
    interval = seconds / 60;
    if (interval > 1) {
        return { unit: "minutes", value: Math.floor(interval) };
    }
    return { unit: "seconds", value: seconds };
}

const blockTags = [
    "address",
    "article",
    "aside",
    "blockquote",
    "div",
    "figure",
    "footer",
    "h[1-6]",
    "header",
    "li",
    "main",
    "nav",
    "ol",
    "p",
    "pre",
    "section",
    "table",
    "td",
    "th",
    "tr",
    "ul",
].join("|");

const blockTagRegex = new RegExp(`</?(?:${blockTags})(?:\\s[^>]*)?/?>`, "gi");

const namedEntities: Record<string, string> = {
    amp: "&",
    apos: "'",
    gt: ">",
    lt: "<",
    nbsp: " ",
    quot: '"',
};

function decodeEntities(text: string) {
    // Single pass, so a decoded "&amp;lt;" stays as the literal text "&lt;"
    return text.replace(
        /&(#\d+|#[xX][0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);/g,
        (entity, body: string) => {
            if (body.startsWith("#")) {
                const codePoint =
                    body[1] === "x" || body[1] === "X"
                        ? parseInt(body.slice(2), 16)
                        : parseInt(body.slice(1), 10);
                if (
                    !Number.isFinite(codePoint) ||
                    codePoint < 0 ||
                    codePoint > 0x10ffff ||
                    (codePoint >= 0xd800 && codePoint <= 0xdfff)
                ) {
                    return entity;
                }
                return String.fromCodePoint(codePoint);
            }
            return namedEntities[body.toLowerCase()] ?? entity;
        },
    );
}

/**
 * Converts rich text to plain text.
 *
 * Deliberately string-based rather than DOM-based: it has to produce identical
 * output during SSR and in the browser, otherwise the two renders disagree and
 * hydration re-renders the subtree.
 */
export function formatHTMLAsText(html?: string) {
    if (!html) {
        return "";
    }

    return decodeEntities(
        html
            .replace(/<(script|style)\b[^>]*>[\s\S]*?<\/\1\s*>/gi, "")
            .replace(/<br\s*\/?>/gi, "\n")
            .replace(blockTagRegex, "\n")
            // Attribute values may contain ">", so match quoted runs explicitly
            .replace(/<(?:[^>"']|"[^"]*"|'[^']*')*>/g, ""),
    )
        .replace(/\r\n?/g, "\n")
        .replace(/[ \t]+\n/g, "\n") // trailing spaces
        .replace(/\n[ \t]+/g, "\n") // leading spaces
        .replace(/\n{3,}/g, "\n\n") // collapse 3+ newlines
        .replace(/[ \t]{2,}/g, "  ") // collapse multiple spaces to two
        .trim();
}

/**
 * Plain-text preview of rich text, truncated to `maxLength` characters.
 *
 * Truncating the HTML itself would tear tags in half, which is what broke
 * shared list rendering (#1128). Counts code points so the cutoff never splits
 * an astral character such as an emoji.
 */
export function formatHTMLAsTextPreview(
    html: string | undefined,
    maxLength: number,
): { text: string; truncated: boolean } {
    const text = formatHTMLAsText(html);
    const characters = Array.from(text);

    if (characters.length <= maxLength) {
        return { text, truncated: false };
    }

    return { text: characters.slice(0, maxLength).join(""), truncated: true };
}
