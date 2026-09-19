import { page } from "$app/state";
import { Parser } from "htmlparser2";

export function formatTimeHHMM(seconds?: number) {
    if (seconds == null || isNaN(seconds)) {
        return "-";
    }

    const totalMinutes = Math.floor(seconds / 60);
    const m = totalMinutes % 60;
    const h = Math.floor(totalMinutes / 60);

    return (h < 10 ? "0" : "") + h.toString() + "h " + (m < 10 ? "0" : "") + m.toString() + "m";
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

export function formatSpeed(speed?: number, fractionDigits?: number) {
    if (speed === undefined) {
        return "-";
    }

    const unit = page.data.settings?.unit ?? "metric";

    if (unit == "metric") {
        return `${(speed * 3.6).toFixed(fractionDigits ?? 2)} km/h`
    } else {
        const mph = speed * 3.6 * 0.621371;

        const formattedMph =
            fractionDigits === undefined
                ? Math.round(mph)
                : mph.toFixed(fractionDigits);
        return `${formattedMph} mp/h`;
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

const blockTags = new Set([
    "address",
    "article",
    "aside",
    "blockquote",
    "div",
    "figure",
    "footer",
    "h1",
    "h2",
    "h3",
    "h4",
    "h5",
    "h6",
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
]);

/**
 * Converts rich text to plain text.
 *
 * Use the same parser during SSR and in the browser to avoid hydration differences.
 * The result is plain text, including decoded entities, and must be rendered as
 * text rather than inserted as HTML.
 */
export function formatHTMLAsText(html?: string) {
    if (!html) {
        return "";
    }

    const text: string[] = [];
    let ignoredDepth = 0;
    const parser = new Parser({
        onopentag(name) {
            if (ignoredDepth > 0 || name === "script" || name === "style") {
                ignoredDepth++;
            } else if (name === "br" || blockTags.has(name)) {
                text.push("\n");
            }
        },
        ontext(value) {
            if (ignoredDepth === 0) text.push(value);
        },
        onclosetag(name) {
            if (ignoredDepth > 0) {
                ignoredDepth--;
            } else if (blockTags.has(name)) {
                text.push("\n");
            }
        },
    });
    parser.end(html);

    return text.join("")
        .replace(/\u00a0/g, " ")
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
