import type { ConfigField, ConfigFieldOption, LocalizedTextMap, PluginProvider } from "$lib/models/plugin_provider";

export function localizedText(
    texts: LocalizedTextMap | undefined,
    currentLocale: string | null | undefined,
    fallback = "",
): string {
    return localizedTextForCandidates(texts, [...localeCandidates(currentLocale), "en"])
        || fallback.trim();
}

export function pluginTitle(plugin: PluginProvider, currentLocale: string | null | undefined): string {
    return localizedText(plugin.displayNames, currentLocale, plugin.displayName || plugin.name);
}

export function pluginDescription(plugin: PluginProvider, currentLocale: string | null | undefined): string {
    return localizedText(plugin.descriptions, currentLocale, plugin.description ?? "");
}

export function pluginInformation(plugin: PluginProvider, currentLocale: string | null | undefined): string {
    const information = localizedTextForCurrentLocale(plugin.information, currentLocale);
    if (information) {
        return information;
    }

    const description = localizedTextForCurrentLocale(plugin.descriptions, currentLocale);
    if (description) {
        return description;
    }

    return localizedTextForCandidates(plugin.information, ["en"])
        || pluginDescription(plugin, currentLocale);
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

export function providerCategoryLabel(
    plugin: PluginProvider,
    providerCategory: string,
    currentLocale: string | null | undefined,
): string {
    const providerCategories = plugin.metadata?.providerCategories;
    if (!providerCategories || typeof providerCategories !== "object" || Array.isArray(providerCategories)) {
        return providerCategory;
    }

    const category = (providerCategories as Record<string, unknown>)[providerCategory];
    if (!category || typeof category !== "object" || Array.isArray(category)) {
        return providerCategory;
    }

    const labels = (category as Record<string, unknown>).labels;
    if (!labels || typeof labels !== "object" || Array.isArray(labels)) {
        return providerCategory;
    }

    return localizedText(labels as LocalizedTextMap, currentLocale, providerCategory);
}

function normalizeLocale(value: string | null | undefined): string {
    return normalizeLocaleKey(value) || "en";
}

function normalizeLocaleKey(value: string | null | undefined): string {
    return (value || "").trim().toLowerCase().replaceAll("_", "-");
}

function localeCandidates(currentLocale: string | null | undefined): string[] {
    const locale = normalizeLocale(currentLocale);
    return [...new Set([locale, locale.split("-")[0]])];
}

function localizedTextForCurrentLocale(
    texts: LocalizedTextMap | undefined,
    currentLocale: string | null | undefined,
): string {
    return localizedTextForCandidates(texts, localeCandidates(currentLocale));
}

function localizedTextForCandidates(
    texts: LocalizedTextMap | undefined,
    candidates: string[],
): string {
    if (!texts) {
        return "";
    }
    for (const candidate of candidates) {
        for (const [key, text] of Object.entries(texts)) {
            if (normalizeLocaleKey(key) === candidate) {
                const value = text.trim();
                if (value) {
                    return value;
                }
            }
        }
    }
    return "";
}
