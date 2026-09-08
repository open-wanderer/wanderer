/** A versioned, deterministic synthetic expansion; never reads clock or instance data. */
export const GENERATOR_VERSION = 'wanderer.srch0.scale/v1';

export function generateScaleDataset(reference, descriptor) {
    if (descriptor.generator !== GENERATOR_VERSION || descriptor.count !== 10001) {
        throw new Error('Unsupported SRCH0 scale dataset descriptor');
    }
    const template = reference.trails.find(trail => trail.id === descriptor.template_id);
    if (!template) throw new Error(`Missing scale template ${descriptor.template_id}`);
    return {
        ...reference,
        trails: Array.from({ length: descriptor.count }, (_, index) => ({
            ...template,
            id: `scale-${String(index + 1).padStart(5, '0')}`,
            name: `Scale trail ${String(index + 1).padStart(5, '0')}`,
            distance: index + 1,
            created: 1767225600 + index,
            _geo: { lat: 47 + (index % 100) / 10000, lng: 8 + Math.floor(index / 100) / 10000 },
        })),
    };
}
