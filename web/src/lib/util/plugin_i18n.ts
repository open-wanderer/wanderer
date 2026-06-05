import type { ConfigField, ConfigFieldOption, LocalizedTextMap, PluginProvider } from "$lib/models/plugin_provider";

export function localizedText(
    texts: LocalizedTextMap | undefined,
    currentLocale: string | null | undefined,
    fallback = "",
): string {
    if (!texts) {
        return fallback;
    }

    const locale = normalizeLocale(currentLocale);
    const language = locale.split("-")[0];
    const candidates = [locale, language, "en"];
    for (const candidate of candidates) {
        const value = texts[candidate]?.trim();
        if (value) {
            return value;
        }
    }

    const trimmedFallback = fallback.trim();
    if (trimmedFallback) {
        return trimmedFallback;
    }

    return "";
}

export function pluginTitle(plugin: PluginProvider, currentLocale: string | null | undefined): string {
    return localizedText(plugin.displayNames, currentLocale, plugin.displayName || plugin.name);
}

export function pluginDescription(plugin: PluginProvider, currentLocale: string | null | undefined): string {
    return localizedText(plugin.descriptions, currentLocale, plugin.description ?? "");
}

export function configFieldLabel(
    field: ConfigField,
    currentLocale: string | null | undefined,
    fallback: string,
): string {
    return localizedText(field.labels, currentLocale, field.label || fallback);
}

export function configFieldDescription(
    field: ConfigField,
    currentLocale: string | null | undefined,
): string | undefined {
    return localizedText(field.descriptions, currentLocale, field.description ?? "") || undefined;
}

export function configFieldOptionLabel(
    option: ConfigFieldOption,
    currentLocale: string | null | undefined,
    fallback: string,
): string {
    return localizedText(option.labels, currentLocale, option.label || fallback);
}

function normalizeLocale(value: string | null | undefined): string {
    return (value || "en").trim().toLowerCase().replace("_", "-");
}
