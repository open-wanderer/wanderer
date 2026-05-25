import ExifReader from 'exifreader';

function removeEmpty(obj: Record<string, any>) {
  Object.entries(obj).forEach(([key, val]) => {
    if (val && val instanceof Object) {
      removeEmpty(val);
    } else if (val == null) {
      delete obj[key];
    }
  });
}

function allDatesToISOString(obj: Record<string, any>) {
  Object.entries(obj).forEach(([key, val]) => {
    if (val) {
      if (val instanceof Date) {
        obj[key] = val.toISOString().split('.')[0] + 'Z';
      } else if (val instanceof Object) {
        allDatesToISOString(val);
      }
    }
  });
}

function haversineDistance(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371; // Radius of the Earth in km
  const dLat = (lat2 - lat1) * (Math.PI / 180); // Convert degrees to radians
  const dLon = (lon2 - lon1) * (Math.PI / 180);
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos((lat1 * (Math.PI / 180))) * Math.cos((lat2 * (Math.PI / 180))) *
    Math.sin(dLon / 2) * Math.sin(dLon / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  const distance = R * c * 1000; // Distance in km
  return distance;
}

function rationalToNumber(r: unknown): number | undefined {
  if (typeof r === "number") return Number.isFinite(r) ? r : undefined;
  if (
    Array.isArray(r) &&
    r.length === 2 &&
    typeof r[0] === "number" &&
    typeof r[1] === "number"
  ) {
    return r[1] === 0 ? undefined : r[0] / r[1];
  }
  return undefined;
}

// Converts a [deg, min, sec] rational array plus a N/S/E/W reference into signed
// decimal degrees. Works on ExifReader's raw GPSLatitude/GPSLongitude `.value`.
function dmsArrayToDecimal(value: unknown, ref: unknown): number | undefined {
  if (!Array.isArray(value) || value.length < 3) return undefined;
  const d = rationalToNumber(value[0]);
  const m = rationalToNumber(value[1]);
  const s = rationalToNumber(value[2]);
  if (d === undefined || m === undefined || s === undefined) return undefined;
  let dd = d + m / 60 + s / 3600;
  const r = Array.isArray(ref) ? ref[0] : ref;
  if (r === "S" || r === "W") dd = -dd;
  return dd;
}

// Reads GPS coordinates from a photo's EXIF data. Accepts a File or a URL
// (including data URLs). Returns decimal degrees, or null if the image has no
// usable GPS tags. Tries multiple shapes defensively: ExifReader's computed
// `gps` group (already signed), then the raw Exif GPS rationals, then a
// non-expanded read - so it is robust across builds/bundlers.
async function gpsFromPhoto(source: string | File): Promise<{ lat: number; lon: number } | null> {
  const buffer =
    typeof source === "string"
      ? await (await fetch(source)).arrayBuffer()
      : await source.arrayBuffer();

  let lat: number | undefined;
  let lon: number | undefined;
  let tags: any;

  try {
    tags = ExifReader.load(buffer, { expanded: true });
  } catch {
    tags = undefined;
  }

  if (tags) {
    // 1) computed gps group (signed decimals)
    if (typeof tags.gps?.Latitude === "number" && typeof tags.gps?.Longitude === "number") {
      lat = tags.gps.Latitude;
      lon = tags.gps.Longitude;
    }
    // 2) raw Exif GPS rationals
    if (lat === undefined || lon === undefined) {
      const ex = tags.exif ?? tags;
      lat = dmsArrayToDecimal(ex?.GPSLatitude?.value, ex?.GPSLatitudeRef?.value);
      lon = dmsArrayToDecimal(ex?.GPSLongitude?.value, ex?.GPSLongitudeRef?.value);
    }
  }

  // 3) non-expanded read as a last resort
  if (lat === undefined || lon === undefined) {
    try {
      const flat: any = ExifReader.load(buffer);
      lat = dmsArrayToDecimal(flat?.GPSLatitude?.value, flat?.GPSLatitudeRef?.value);
      lon = dmsArrayToDecimal(flat?.GPSLongitude?.value, flat?.GPSLongitudeRef?.value);
    } catch {
      // no usable EXIF GPS data
    }
  }

  return typeof lat === "number" && typeof lon === "number" ? { lat, lon } : null;
}

export { removeEmpty, allDatesToISOString, haversineDistance, gpsFromPhoto };
