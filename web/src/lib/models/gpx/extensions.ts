// Extracts per-trackpoint sensor metrics (heart rate, cadence, power, air
// temperature) from a GPX <extensions> tree. GPX exporters disagree on the
// namespace prefix (gpxtpx:, ns3:, gpxpx: ...) and on nesting depth, so we match
// purely on the local element name and walk the tree recursively.

export type PointMetrics = {
  heartRate?: number;
  cadence?: number;
  power?: number;
  temperature?: number;
};

export const POINT_METRIC_KEYS = ["heartRate", "cadence", "power", "temperature"] as const;
export type PointMetricKey = (typeof POINT_METRIC_KEYS)[number];

const HR_NAMES = new Set(["hr", "heartrate", "heart_rate"]);
const CADENCE_NAMES = new Set(["cad", "cadence"]);
const POWER_NAMES = new Set(["power", "powerinwatts", "watts"]);
const TEMP_NAMES = new Set(["atemp", "temp", "temperature"]);

function localName(key: string): string {
  const colon = key.lastIndexOf(":");
  return (colon >= 0 ? key.slice(colon + 1) : key).toLowerCase();
}

function toNumber(value: unknown): number | undefined {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : undefined;
  }
  if (typeof value === "string") {
    const n = parseFloat(value);
    return Number.isFinite(n) ? n : undefined;
  }
  // xml2js represents an element that has both text and attributes as { _: text, $: attrs }
  if (value && typeof value === "object" && "_" in (value as Record<string, unknown>)) {
    return toNumber((value as Record<string, unknown>)._);
  }
  return undefined;
}

// Builds a plain object that the xml2js GPX builder serializes into a
// TrackPointExtension <extensions> block readable again by extractPointMetrics().
// Returns undefined when there is nothing to write, so callers can leave the
// point's extensions unset.
export function buildExtensionsObject(metrics: PointMetrics): Record<string, unknown> | undefined {
  const tpx: Record<string, number> = {};
  if (metrics.heartRate !== undefined) tpx.hr = metrics.heartRate;
  if (metrics.cadence !== undefined) tpx.cad = metrics.cadence;
  if (metrics.temperature !== undefined) tpx.atemp = metrics.temperature;

  const ext: Record<string, unknown> = {};
  if (Object.keys(tpx).length > 0) {
    ext.TrackPointExtension = tpx;
  }
  if (metrics.power !== undefined) {
    ext.power = metrics.power;
  }

  return Object.keys(ext).length > 0 ? ext : undefined;
}

export function extractPointMetrics(extensions: unknown): PointMetrics {
  const metrics: PointMetrics = {};
  if (!extensions || typeof extensions !== "object") {
    return metrics;
  }

  const visit = (node: unknown) => {
    if (!node || typeof node !== "object") {
      return;
    }
    for (const [key, value] of Object.entries(node as Record<string, unknown>)) {
      if (key === "$") {
        continue; // xml2js attribute bag
      }
      const num = toNumber(value);
      if (num !== undefined) {
        const name = localName(key);
        if (metrics.heartRate === undefined && HR_NAMES.has(name)) metrics.heartRate = num;
        else if (metrics.cadence === undefined && CADENCE_NAMES.has(name)) metrics.cadence = num;
        else if (metrics.power === undefined && POWER_NAMES.has(name)) metrics.power = num;
        else if (metrics.temperature === undefined && TEMP_NAMES.has(name)) metrics.temperature = num;
      } else if (value && typeof value === "object") {
        visit(value);
      }
    }
  };

  visit(extensions);
  return metrics;
}
