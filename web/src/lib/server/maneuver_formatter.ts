import {
    LOCALE_LOADERS,
    SUPPORTED_LOCALES,
    defaultLocale,
    normalizeLocale,
    type SupportedLocale,
} from "$lib/i18n/locales";
import type {
    RoutingInternalManeuverResponse,
    RoutingManeuver,
    RoutingManeuverResponse,
    RoutingManeuverType,
} from "$lib/models/routing";

const MAX_INSTRUCTION_CHARACTERS = 500;

const maneuverMessageKeys: Record<RoutingManeuverType, string> = {
    start: "maneuver-start",
    destination: "maneuver-destination",
    continue: "maneuver-continue",
    turn_left: "maneuver-turn-left",
    turn_right: "maneuver-turn-right",
    turn_slight_left: "maneuver-turn-slight-left",
    turn_slight_right: "maneuver-turn-slight-right",
    turn_sharp_left: "maneuver-turn-sharp-left",
    turn_sharp_right: "maneuver-turn-sharp-right",
    keep_left: "maneuver-keep-left",
    keep_right: "maneuver-keep-right",
    uturn_left: "maneuver-uturn-left",
    uturn_right: "maneuver-uturn-right",
    uturn: "maneuver-uturn",
    roundabout_enter: "maneuver-roundabout-enter",
    roundabout_exit: "maneuver-roundabout-exit",
    exit_left: "maneuver-exit-left",
    exit_right: "maneuver-exit-right",
    ramp_straight: "maneuver-ramp-straight",
    ramp_left: "maneuver-ramp-left",
    ramp_right: "maneuver-ramp-right",
    merge: "maneuver-merge",
    merge_left: "maneuver-merge-left",
    merge_right: "maneuver-merge-right",
    ferry: "maneuver-ferry",
    unknown: "maneuver-continue",
};

export interface ManeuverLanguageInput {
    explicit?: string | null;
    settingsLanguage?: string | null;
    acceptLanguage?: string | null;
}

export interface ResolvedManeuverLanguage {
    language: SupportedLocale;
    languageFallback: boolean;
}

export function resolveManeuverLanguage(input: ManeuverLanguageInput): ResolvedManeuverLanguage {
    const explicit = input.explicit?.trim();
    if (explicit) {
        const canonical = canonicalizeLocale(explicit);
        const language = normalizeLocale(canonical);
        return {
            language,
            languageFallback: canonical.toLowerCase() !== language.toLowerCase(),
        };
    }

    for (const candidate of [input.settingsLanguage, ...acceptLanguageCandidates(input.acceptLanguage)]) {
        if (!candidate?.trim()) continue;
        const supported = supportedLocale(candidate);
        if (supported) {
            return { language: supported, languageFallback: false };
        }
    }
    return { language: defaultLocale, languageFallback: false };
}

export async function formatManeuverResponse(
    internal: RoutingInternalManeuverResponse,
    resolved: ResolvedManeuverLanguage,
): Promise<RoutingManeuverResponse> {
    const loadMessages = LOCALE_LOADERS[resolved.language] ?? LOCALE_LOADERS[defaultLocale];
    const messages = loadMessages ? await loadMessages() : {};
    const maneuvers = internal.maneuvers.map((maneuver) =>
        formatManeuver(maneuver, messages),
    );
    const warnings = uniqueStrings([
        ...(internal.warnings ?? []),
        ...(resolved.languageFallback ? ["language_fallback"] : []),
    ]);
    return {
        language: resolved.language,
        geometry: internal.geometry,
        maneuvers,
        ...(warnings.length > 0 ? { warnings } : {}),
    };
}

function formatManeuver(
    internal: RoutingInternalManeuverResponse["maneuvers"][number],
    messages: Record<string, unknown>,
): RoutingManeuver {
    const { providerInstruction, warnings: _internalWarnings, ...publicFields } = internal;
    const warnings = [...(internal.warnings ?? [])];
    let instruction: string;
    if (internal.type === "unknown" && providerInstruction?.trim()) {
        instruction = providerInstruction.trim();
        warnings.push("provider_instruction_fallback");
    } else {
        const key = maneuverMessageKeys[internal.type];
        let template = message(messages, key, defaultTemplate(internal.type));
        if (
            (internal.type === "roundabout_enter" || internal.type === "roundabout_exit") &&
            internal.roundaboutExit != null &&
            internal.roundaboutExit > 0
        ) {
            template = message(messages, "maneuver-roundabout-enter-exit", template);
        }
        const street = internal.streetNames?.find((name) => name.trim())?.trim() ??
            message(messages, "maneuver-route", "the route");
        instruction = interpolate(template, {
            street,
            exit: String(internal.roundaboutExit ?? ""),
        });
    }
    const bounded = truncateInstruction(instruction);
    if (bounded.truncated) warnings.push("instruction_truncated");
    const normalizedWarnings = uniqueStrings(warnings);
    return {
        ...publicFields,
        instruction: bounded.value,
        ...(normalizedWarnings.length > 0 ? { warnings: normalizedWarnings } : {}),
    };
}

function message(messages: Record<string, unknown>, key: string, fallback: string): string {
    const value = messages[key];
    return typeof value === "string" && value ? value : fallback;
}

function interpolate(template: string, values: Record<string, string>): string {
    return template.replace(/\{(street|exit)\}/g, (_, key: string) => values[key] ?? "");
}

function truncateInstruction(value: string): { value: string; truncated: boolean } {
    const characters = Array.from(value);
    if (characters.length <= MAX_INSTRUCTION_CHARACTERS) {
        return { value, truncated: false };
    }
    return {
        value: `${characters.slice(0, MAX_INSTRUCTION_CHARACTERS - 1).join("")}…`,
        truncated: true,
    };
}

function canonicalizeLocale(raw: string): string {
    const candidate = raw.trim().split(";", 1)[0].replace("_", "-");
    try {
        return Intl.getCanonicalLocales(candidate)[0] ?? candidate.toLowerCase();
    } catch {
        return candidate.toLowerCase();
    }
}

function supportedLocale(raw: string): SupportedLocale | undefined {
    const canonical = canonicalizeLocale(raw);
    const base = canonical.toLowerCase().split("-", 1)[0];
    return SUPPORTED_LOCALES.includes(base) ? base : undefined;
}

function acceptLanguageCandidates(value: string | null | undefined): string[] {
    if (!value) return [];
    return value
        .split(",")
        .map((part) => part.trim())
        .filter((part) => Boolean(part) && quality(part) > 0)
        .sort((left, right) => quality(right) - quality(left))
        .map((part) => part.split(";", 1)[0]);
}

function quality(value: string): number {
    const match = value.match(/;\s*q=([0-9.]+)/i);
    const parsed = match ? Number(match[1]) : 1;
    return Number.isFinite(parsed) ? parsed : 0;
}

function uniqueStrings(values: string[]): string[] {
    return Array.from(new Set(values.filter(Boolean)));
}

function defaultTemplate(type: RoutingManeuverType): string {
    const templates: Record<RoutingManeuverType, string> = {
        start: "Start on {street}.",
        destination: "You have reached your destination.",
        continue: "Continue on {street}.",
        turn_left: "Turn left onto {street}.",
        turn_right: "Turn right onto {street}.",
        turn_slight_left: "Turn slightly left onto {street}.",
        turn_slight_right: "Turn slightly right onto {street}.",
        turn_sharp_left: "Turn sharply left onto {street}.",
        turn_sharp_right: "Turn sharply right onto {street}.",
        keep_left: "Keep left on {street}.",
        keep_right: "Keep right on {street}.",
        uturn_left: "Make a left U-turn onto {street}.",
        uturn_right: "Make a right U-turn onto {street}.",
        uturn: "Make a U-turn onto {street}.",
        roundabout_enter: "Enter the roundabout toward {street}.",
        roundabout_exit: "Exit the roundabout onto {street}.",
        exit_left: "Take the left exit onto {street}.",
        exit_right: "Take the right exit onto {street}.",
        ramp_straight: "Continue on the ramp toward {street}.",
        ramp_left: "Take the left ramp toward {street}.",
        ramp_right: "Take the right ramp toward {street}.",
        merge: "Merge onto {street}.",
        merge_left: "Merge left onto {street}.",
        merge_right: "Merge right onto {street}.",
        ferry: "Continue via the ferry toward {street}.",
        unknown: "Continue on {street}.",
    };
    return templates[type];
}
