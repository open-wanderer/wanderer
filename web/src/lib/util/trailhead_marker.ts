import type { Subcategory } from "#lib/models/subcategory.ts";
import { displayCategoryIcon, displaySubcategoryIcon } from "#lib/util/category_util.ts";
import type { FeatureCollection } from "geojson";
import type * as M from "maplibre-gl";

// Matches the app's trailhead markers: a primary-colored circle with a white
// border and the white category icon.
const MARKER_SIZE = 28;
const ICON_SIZE = 13;
const BORDER_WIDTH = 2;
const MARKER_COLOR = "#242734";

export function trailheadImageName(icon: string): string {
    return `trailhead-${icon}`;
}

/** The subcategory icon, falling back to the category icon and then a generic one. */
export function trailheadIcon(
    properties: { category_icon?: string | null; subcategory_id?: string | null },
    subcategories: Pick<Subcategory, "id" | "icon">[] = [],
): string {
    const subcategory = properties.subcategory_id
        ? subcategories.find((candidate) => candidate.id === properties.subcategory_id)
        : undefined;
    const category = { icon: properties.category_icon ?? "" };
    return subcategory ? displaySubcategoryIcon(subcategory, category) : displayCategoryIcon(category);
}

/** Add the marker image name of every trailhead to its properties. */
export function withTrailheadIcons(
    geojson: FeatureCollection,
    subcategories: Pick<Subcategory, "id" | "icon">[] = [],
): FeatureCollection {
    return {
        ...geojson,
        features: geojson.features.map((feature) => feature.properties?.cluster ? feature : {
            ...feature,
            properties: {
                ...feature.properties,
                marker_icon: trailheadImageName(trailheadIcon(feature.properties ?? {}, subcategories)),
            },
        }),
    };
}

/** Font Awesome renders icons as a glyph in `::before`; read it to draw the same glyph on a canvas. */
function iconGlyph(icon: string) {
    const element = document.createElement("i");
    element.className = `fa ${icon}`;
    element.style.cssText = "position:absolute;visibility:hidden;pointer-events:none";
    document.body.append(element);
    const style = getComputedStyle(element, "::before");
    // Font Awesome 7 adds alternative text: `"\uf6ec" / ""`. Only the first string is the glyph.
    // The computed style is live: read it before the element goes away.
    const glyph = { text: style.content.match(/^["'](.*?)["']/)?.[1], family: style.fontFamily, weight: style.fontWeight };
    element.remove();
    return glyph.text ? { ...glyph, text: glyph.text } : undefined;
}

async function drawMarker(icon: string, ratio: number) {
    const size = MARKER_SIZE * ratio;
    const canvas = document.createElement("canvas");
    canvas.width = canvas.height = size;
    const context = canvas.getContext("2d");
    if (!context) return;
    context.beginPath();
    context.arc(size / 2, size / 2, size / 2 - BORDER_WIDTH * ratio / 2, 0, Math.PI * 2);
    context.fillStyle = MARKER_COLOR;
    context.fill();
    context.lineWidth = BORDER_WIDTH * ratio;
    context.strokeStyle = "#fff";
    context.stroke();
    const glyph = iconGlyph(icon) ?? iconGlyph(displayCategoryIcon());
    if (glyph) {
        const font = `${glyph.weight} ${ICON_SIZE * ratio}px ${glyph.family}`;
        try { await document.fonts.load(font, glyph.text); } catch { /* Draw with whatever font is available. */ }
        context.font = font;
        context.fillStyle = "#fff";
        context.textAlign = "center";
        context.textBaseline = "middle";
        context.fillText(glyph.text, size / 2, size / 2 + ratio / 2);
    }
    return context.getImageData(0, 0, size, size);
}

/** Register a marker image for every trailhead icon the map does not have yet. */
export async function ensureTrailheadImages(map: M.Map, geojson: FeatureCollection) {
    const style = map.style;
    if (!style) return;
    const ratio = Math.max(1, Math.ceil(window.devicePixelRatio || 1));
    const names = new Set(geojson.features.map((feature) => feature.properties?.marker_icon).filter(Boolean));
    await Promise.all([...names].map(async (name: string) => {
        if (map.hasImage(name)) return;
        const image = await drawMarker(name.replace(/^trailhead-/, ""), ratio);
        if (image && map.style === style && !map.hasImage(name)) map.addImage(name, image, { pixelRatio: ratio });
    }));
}
