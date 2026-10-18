import Link from './link';

/**
 * The grammar Dart's `DateTime.parse`/`DateTime.tryParse` accepts, transcribed
 * verbatim from the Dart SDK's `DateTime._parseFormat`.
 *
 * `new Date(...)` is far more permissive than this — it happily parses
 * `"Jan 1 2024"` and other locale/legacy formats — so gating on this pattern
 * is what keeps the web's `<time>` semantics aligned with the Dart port's
 *. GPX 1.1 mandates ISO 8601 for `<time>` anyway, so nothing a
 * conforming exporter emits is excluded by it.
 */
const DART_DATETIME_GRAMMAR =
  /^([+-]?\d{4,6})-?(\d\d)-?(\d\d)(?:[ T](\d\d)(?::?(\d\d)(?::?(\d\d)(?:[.,](\d+))?)?)?( ?[zZ]| ?([-+])(\d\d)(?::?(\d\d))?)?)?$/;

/**
 * Parses a `<time>` element body the same way the Dart port does: anything
 * that is not a well-formed ISO-8601 instant is "no time", not a time.
 *
 * Before this existed the constructor did a bare `new Date(object.time)`, and
 * because an `Invalid Date` is a truthy object, `GPX.getTotals()`'s
 * `startTime && endTime` guard passed and computed
 * `endTime.getTime() - startTime.getTime()` — i.e. `NaN` — as the trail's
 * duration. The Dart side, whose sanitizer rewrites an unparseable `<time>`
 * to a self-closing tag, reported `0` for the same document.
 *
 * Residual, deliberately-accepted asymmetry: a string this grammar admits but
 * V8's `Date` rejects (a 6-digit year, an out-of-range month that Dart's
 * `DateTime` would normalise) yields "no time" here and a real time in Dart.
 * That class is not reachable from any GPX a real exporter produces.
 */
function parseGpxTime(raw: unknown): Date | undefined {
  if (raw instanceof Date) {
    return Number.isNaN(raw.getTime()) ? undefined : raw;
  }
  if (typeof raw !== "string") {
    return undefined;
  }
  const trimmed = raw.trim();
  if (!DART_DATETIME_GRAMMAR.test(trimmed)) {
    return undefined;
  }
  const parsed = new Date(trimmed);
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}

export default class Waypoint {
  $: {
    lat?: number;
    lon?: number;
  }
  ele?: number;
  time?: Date;
  magvar?: string;
  geoidheight?: string;
  name?: string;
  cmt?: string
  desc?: string;
  src?: string;
  sym?: string;
  type?: string;
  sat?: string;
  hdop?: string;
  vdop?: string;
  pdop?: string;
  ageofdgpsdata?: string;
  dgpsid?: string;
  extensions?: string;
  link?: Link[];
  constructor(object: {
    lat?: number,
    lon?: number,
    $: {
      lat?: number,
      lon?: number
    },
    ele?: number,
    time?: Date,
    magvar?: string,
    geoidheight?: string,
    name?: string,
    cmt?: string,
    desc?: string,
    src?: string,
    sym?: string,
    type?: string,
    sat?: string,
    hdop?: string,
    vdop?: string,
    pdop?: string,
    ageofdgpsdata?: string,
    dgpsid?: string,
    extensions?: string,
    link?: Link[]
  }) {
    this.$ = {};
    this.$.lat = object.$.lat === 0 || object.lat === 0 ? 0 : object.$.lat || object.lat || -1;
    this.$.lon = object.$.lon === 0 || object.lon === 0 ? 0 : object.$.lon || object.lon || -1;
    this.ele = object.ele;
    if (object.time) {
      this.time = parseGpxTime(object.time);
    }
    this.magvar = object.magvar;
    this.geoidheight = object.geoidheight;
    this.name = object.name;
    this.cmt = object.cmt;
    this.desc = object.desc;
    this.src = object.src;
    this.sym = object.sym;
    this.type = object.type;
    this.sat = object.sat;
    this.hdop = object.hdop;
    this.vdop = object.vdop;
    this.pdop = object.pdop;
    this.ageofdgpsdata = object.ageofdgpsdata;
    this.dgpsid = object.dgpsid;
    this.extensions = object.extensions;
    if (object.link) {
      if (!Array.isArray(object.link)) {
        object.link = [object.link];
      }
      this.link = object.link.map(l => new Link(l));
    }
  }

  toGeoJSON(): GeoJSON.Feature {
    return {
      type: "Feature",
      geometry: {
        type: "Point",
        coordinates: [this.$.lon ?? 0, this.$.lat ?? 0, this.ele ?? 0]
      },
      properties: {
        name: this.name,
        desc: this.desc,
        time: this.time?.toISOString(),
        type: this.type,
        sym: this.sym
      }
    };
  }

}