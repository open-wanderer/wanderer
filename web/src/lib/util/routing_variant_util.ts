import type {
    RoutingCandidate,
} from "$lib/models/routing";

export const ROUTING_MAX_VARIANT_ANCHORS = 16;
export const ROUTING_MAX_VARIANTS = 3;
export const ROUTING_MAX_ENGINES = 4;

export function routingCandidateEngineLabel(candidate: RoutingCandidate) {
    const engines: {
        aliases: Set<string>;
        label: string;
        hasProvider: boolean;
    }[] = [];

    function addEngine(provider?: string, pluginId?: string) {
        const providerLabel = provider?.trim() ?? "";
        const pluginLabel = pluginId?.trim() ?? "";
        const aliases = new Set(
            [providerLabel, pluginLabel]
                .filter(Boolean)
                .map((value) => value.toLowerCase()),
        );
        if (!aliases.size) return;

        const matchingEngines = engines.filter((engine) =>
            [...aliases].some((alias) => engine.aliases.has(alias)),
        );
        if (!matchingEngines.length) {
            engines.push({
                aliases,
                label: providerLabel || pluginLabel,
                hasProvider: Boolean(providerLabel),
            });
            return;
        }

        const target = matchingEngines[0];
        for (const alias of aliases) target.aliases.add(alias);
        for (const duplicate of matchingEngines.slice(1)) {
            for (const alias of duplicate.aliases) target.aliases.add(alias);
            engines.splice(engines.indexOf(duplicate), 1);
        }
        if (providerLabel && !target.hasProvider) {
            target.label = providerLabel;
            target.hasProvider = true;
        }
    }

    addEngine(candidate.provider, candidate.pluginId);
    for (const segment of candidate.segments) {
        addEngine(segment.provenance?.provider, segment.provenance?.pluginId);
    }
    return engines.map((engine) => engine.label).join(" · ");
}
