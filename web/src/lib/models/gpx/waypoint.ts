import Link from './link';
import { parseGpxNumber } from './number';

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
 * V8's `Date` still rejects after normalisation (a 6-digit year, an
 * out-of-range month that Dart's `DateTime` would normalise) yields "no time"
 * here and a real time in Dart. That class is not reachable from any GPX a
 * real exporter produces.
 */
function parseGpxTime(raw: unknown): Date | undefined {
  if (raw instanceof Date) {
    return Number.isNaN(raw.getTime()) ? undefined : raw;
  }
  if (typeof raw !== "string") {
    return undefined;
  }
  const match = DART_DATETIME_GRAMMAR.exec(raw.trim());
  if (!match) {
    return undefined;
  }
  const parsed = new Date(normalizedIsoString(match));
  return Number.isNaN(parsed.getTime()) ? undefined : parsed;
}

/**
 * Rewrites a grammar match into the one ISO-8601 form V8's `Date` reliably
 * reads. The grammar also admits forms `Date` rejects but Dart reads: the
 * basic format (`20240501T100000Z`), a comma fraction (`10:00:00,5Z`) and an
 * hour-only offset (`+02`). Without a zone the time stays local, as in Dart.
 */
function normalizedIsoString(match: RegExpExecArray): string {
  const [, year, month, day, hour, minute, second, fraction, zone, sign, zoneHour, zoneMinute] = match;
  const date = `${year}-${month}-${day}`;
  if (hour === undefined) {
    return date;
  }
  const millis = fraction ? `.${fraction.slice(0, 3).padEnd(3, "0")}` : "";
  let offset = "";
  if (sign) {
    offset = `${sign}${zoneHour}:${zoneMinute ?? "00"}`;
  } else if (zone) {
    offset = "Z";
  }
  return `${date}T${hour}:${minute ?? "00"}:${second ?? "00"}${millis}${offset}`;
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
    // A missing or unparseable coordinate stays undefined, as in the app's
    // reader. It used to fall back to -1, which put the point near the Gulf
    // of Guinea and added thousands of kilometres to the trail's distance.
    this.$.lat = parseGpxNumber(object.$?.lat) ?? parseGpxNumber(object.lat);
    this.$.lon = parseGpxNumber(object.$?.lon) ?? parseGpxNumber(object.lon);
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