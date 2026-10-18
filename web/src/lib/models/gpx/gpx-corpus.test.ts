import { existsSync, readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { gpx2trail } from "$lib/util/gpx_util";
import GPX from "./gpx";

// The corpus is a single, on-disk, language-neutral fixture set at the repo root,
// sibling to web/, app/, db/ — not duplicated per language. See fixtures/gpx-corpus/README.md
// for the full contract this file implements.
const corpusRoot = path.resolve(process.cwd(), "..", "fixtures", "gpx-corpus");

// A wrong CWD must fail loudly (zero fixtures discovered), not silently pass an
// empty/vacuous suite. Vitest's CWD must be web/ for this relative resolution to work.
if (!existsSync(corpusRoot)) {
  throw new Error(
    `GPX corpus not found at resolved path "${corpusRoot}". ` +
      `This test must be run with Vitest's working directory set to "web/" ` +
      `(e.g. "cd web && npx vitest run"), not the repo root.`,
  );
}

// Absolute tolerances, declared once here and never inline in a test body.
// See fixtures/gpx-corpus/README.md's tolerance table for the full field-by-field breakdown
// and the empirical justification (a 5000-point haversine accumulation was bit-identical
// between Node/V8 and the Dart VM to 17 significant digits).
const METRE_TOLERANCE = 1e-6; // metres — distance and elevation fields
const DEGREE_TOLERANCE = 1e-9; // degrees — centroid fields only

type CorpusWaypoint = {
  lat: number;
  lon: number;
  name: string;
  description: string | null;
  icon: string | null;
};

type CorpusExpected = {
  id: string;
  covers: string[];
  derivation: "hand" | "seeded";
  notes: string;
  metrics: {
    distance: number;
    elevationGain: number;
    elevationLoss: number;
    durationMs: number;
    pointCount: number;
    boundingBox: { minLat: number; maxLat: number; minLon: number; maxLon: number };
    centroid: { lat: number; lon: number };
  };
  trail: {
    name: string;
    description: string | null;
    lat: number;
    lon: number;
    date: string | null;
    distance: number;
    elevationGain: number;
    elevationLoss: number;
    duration: number;
    minLat: number;
    maxLat: number;
    minLon: number;
    maxLon: number;
    waypoints: CorpusWaypoint[];
  };
};

type CorpusFixture = {
  dir: string;
  inputGpx: string;
  expected: CorpusExpected;
};

/**
 * Reads every fixture directory under fixtures/gpx-corpus/ from disk. Asserts at least 12
 * fixtures are discovered so a corpus that silently shrinks (a fixture accidentally deleted
 * or excluded) fails this suite rather than passing vacuously.
 */
function loadCorpusFixtures(): CorpusFixture[] {
  const dirs = readdirSync(corpusRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort();

  const fixtures = dirs.map((dir): CorpusFixture => {
    const inputGpx = readFileSync(path.join(corpusRoot, dir, "input.gpx"), "utf8");
    const expected = JSON.parse(
      readFileSync(path.join(corpusRoot, dir, "expected.json"), "utf8"),
    ) as CorpusExpected;
    return { dir, inputGpx, expected };
  });

  expect(fixtures.length).toBeGreaterThanOrEqual(12);
  return fixtures;
}

/**
 * Compares an "actual" record against an "expected" record, one field at a time, using an
 * absolute tolerance per field name (README's tolerance table, implemented exactly once here).
 * A field with no entry in toleranceByField — or a tolerance of 0 — is compared exactly via
 * toEqual; a field with a positive tolerance is compared via an absolute-difference check.
 */
function expectCorpusMatch(
  actual: Record<string, number | string | null | undefined>,
  expected: Record<string, number | string | null | undefined>,
  toleranceByField: Partial<Record<string, number>> = {},
) {
  for (const key of Object.keys(expected)) {
    const tolerance = toleranceByField[key];
    const actualValue = actual[key];
    const expectedValue = expected[key];

    if (typeof tolerance === "number" && tolerance > 0) {
      expect(
        Math.abs((actualValue as number) - (expectedValue as number)),
        `field "${key}": ${actualValue} not within ${tolerance} of ${expectedValue}`,
      ).toBeLessThanOrEqual(tolerance);
    } else {
      expect(actualValue, `field "${key}" mismatch`).toEqual(expectedValue);
    }
  }
}

const fixtures = loadCorpusFixtures();

describe("GPX corpus - metrics", () => {
  for (const fixture of fixtures) {
    it(`${fixture.dir}: GPX.parse() reproduces the corpus's metrics`, () => {
      const gpx = GPX.parse(fixture.inputGpx);

      expectCorpusMatch(
        {
          distance: gpx.features.distance,
          elevationGain: gpx.features.elevationGain,
          elevationLoss: gpx.features.elevationLoss,
          durationMs: gpx.features.duration,
        },
        {
          distance: fixture.expected.metrics.distance,
          elevationGain: fixture.expected.metrics.elevationGain,
          elevationLoss: fixture.expected.metrics.elevationLoss,
          durationMs: fixture.expected.metrics.durationMs,
        },
        { distance: METRE_TOLERANCE, elevationGain: METRE_TOLERANCE, elevationLoss: METRE_TOLERANCE },
      );

      expectCorpusMatch(gpx.features.boundingBox, fixture.expected.metrics.boundingBox);

      expectCorpusMatch(gpx.features.centroid, fixture.expected.metrics.centroid, {
        lat: DEGREE_TOLERANCE,
        lon: DEGREE_TOLERANCE,
      });

      // Deliberately not asserted anywhere in this file: the raw index-aligned
      // per-point distance array, or the track-shape hash. Both exist on the TS class but
      // are outside the corpus's public-metrics-only contract.
      expect(gpx.flatten().length).toBe(fixture.expected.metrics.pointCount);
    });
  }
});

describe("GPX corpus - trail assembly", () => {
  for (const fixture of fixtures) {
    it(`${fixture.dir}: gpx2trail() reproduces the corpus's trail block`, async () => {
      // correctElevation left at its default `false` so gpx2trail performs no fetch.
      const { trail } = await gpx2trail(fixture.inputGpx);

      const nameAndLocation: Record<string, string | number | null> = {
        name: trail.name,
        description: trail.description ?? null,
        lat: trail.lat ?? null,
        lon: trail.lon ?? null,
      };
      const expectedNameAndLocation = {
        name: fixture.expected.trail.name,
        description: fixture.expected.trail.description,
        lat: fixture.expected.trail.lat,
        lon: fixture.expected.trail.lon,
      };
      expectCorpusMatch(nameAndLocation, expectedNameAndLocation);

      // trail.date is only asserted when the fixture supplies GPX timestamps. The Trail
      // constructor defaults `date` to *today's* date (a non-deterministic value) whenever
      // gpx2trail finds no start/end trkpt time to derive it from — that default is a Trail
      // model implementation detail, not part of the GPX-derived contract this corpus pins.
      if (fixture.expected.trail.date !== null) {
        expect(trail.date).toBe(fixture.expected.trail.date);
      }

      expectCorpusMatch(
        {
          distance: trail.distance,
          elevationGain: trail.elevation_gain,
          elevationLoss: trail.elevation_loss,
        },
        {
          distance: fixture.expected.trail.distance,
          elevationGain: fixture.expected.trail.elevationGain,
          elevationLoss: fixture.expected.trail.elevationLoss,
        },
        { distance: METRE_TOLERANCE, elevationGain: METRE_TOLERANCE, elevationLoss: METRE_TOLERANCE },
      );

      expect(trail.duration).toBe(fixture.expected.trail.duration);

      expectCorpusMatch(
        {
          minLat: trail.min_lat ?? null,
          maxLat: trail.max_lat ?? null,
          minLon: trail.min_lon ?? null,
          maxLon: trail.max_lon ?? null,
        },
        {
          minLat: fixture.expected.trail.minLat,
          maxLat: fixture.expected.trail.maxLat,
          minLon: fixture.expected.trail.minLon,
          maxLon: fixture.expected.trail.maxLon,
        },
      );

      const actualWaypoints = trail.expand?.waypoints_via_trail ?? [];
      expect(actualWaypoints.length).toBe(fixture.expected.trail.waypoints.length);

      fixture.expected.trail.waypoints.forEach((expectedWaypoint, index) => {
        const actualWaypoint = actualWaypoints[index];
        expectCorpusMatch(
          {
            lat: actualWaypoint.lat,
            lon: actualWaypoint.lon,
            name: actualWaypoint.name ?? null,
            description: actualWaypoint.description ?? null,
            icon: actualWaypoint.icon ?? null,
          },
          expectedWaypoint,
        );
      });
    });
  }
});

describe("GPX corpus - integrity", () => {
  // Iterated per fixture (rather than one aggregate assertion loop) so a failure identifies
  // the offending fixture by name in the test title, matching the other two describes' style.
  for (const fixture of fixtures) {
    it(`${fixture.dir}: expected.json id matches its directory name and, if hand-derived, has a DERIVATION.md sibling`, () => {
      expect(fixture.expected.id, `fixture directory "${fixture.dir}"`).toBe(fixture.dir);

      if (fixture.expected.derivation === "hand") {
        const derivationPath = path.join(corpusRoot, fixture.dir, "DERIVATION.md");
        expect(existsSync(derivationPath), `${fixture.dir}/DERIVATION.md`).toBe(true);
      }
    });
  }
});
