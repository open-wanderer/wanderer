import GPX from "#lib/models/gpx/gpx.ts";
import Person from "#lib/models/gpx/person.ts";
import { Trail } from "#lib/models/trail.ts";
import { kml, tcx } from "#lib/vendor/toGeoJSON/toGeoJSON.js";
import cryptoRandomString from "crypto-random-string";
//@ts-ignore
import { browser } from "$app/env";
import Track from "#lib/models/gpx/track.ts";
import TrackSegment from "#lib/models/gpx/track-segment.ts";
import GPXWaypoint, { hasUsablePosition } from "#lib/models/gpx/waypoint.ts";
import { Waypoint } from "#lib/models/waypoint.ts";
import { trails_show } from "#lib/stores/trail_store.ts";
import FitParser from "#lib/vendor/fit-parser/fit_parser.ts";
import { DOMParser as XMLDOMParser } from "@xmldom/xmldom";
import type { Feature, FeatureCollection, GeoJsonProperties, Position } from 'geojson';
import JSZip from "jszip";
import { ClientResponseError, type AuthRecord } from "pocketbase";
import { handleFromRecordWithIRI } from "./activitypub_util";
import { joinContiguousTrackSegments } from "./gpx_segment_util";
import { icons } from "./icon_util";


export async function gpx2trail(gpxString: string, fallbackName?: string, correctElevation: boolean = false, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const gpx = GPX.parse(gpxString);

    if (gpx instanceof Error) {
        throw gpx;
    }

    // Files are converted when they are imported (see fromFile). This covers
    // route-only files that were stored before that.
    gpx.convertRoutesToTracks();
    if (correctElevation) {
        try {
            await gpx.correctElevation(f)
        } catch (e) {
            console.warn("Unable to correct elevation: " + e)
        }
    }


    const trail = new Trail("");

    trail.name = gpx.metadata?.name || gpx.trk?.at(0)?.name || gpx.rte?.at(0)?.name || fallbackName || `trail-${new Date().toISOString()}`;

    trail.description = gpx.metadata?.desc;

    for (const wpt of gpx.wpt ?? []) {
        if (!hasUsablePosition(wpt)) {
            continue;
        }
        const wp = new Waypoint(wpt.$.lat!, wpt.$.lon!);
        wp.id = cryptoRandomString({ length: 15 });
        wp.name = wpt.name ?? ""
        wp.description = wpt.desc;
        if(wpt.sym && icons.includes(wpt.sym as typeof icons[number])) {
            wp.icon = wpt.sym as typeof icons[number];
        }
        trail.expand!.waypoints_via_trail?.push(wp);
    }

    const totals = gpx.features

    const firstSegment = gpx.trk?.at(0)?.trkseg?.at(0);
    const trackPoints = firstSegment?.trkpt
    const routePoints = gpx.rte?.at(0)?.rtept?.filter(hasUsablePosition);

    const startPoint = trackPoints?.at(0) ?? routePoints?.at(0);
    if (startPoint) {
        trail.lat = startPoint.$.lat
        trail.lon = startPoint.$.lon
    }

    const startTime = firstSegment?.fileTimes ? firstSegment.fileTimes.start : trackPoints?.at(0)?.time;
    const endTime = firstSegment?.fileTimes ? firstSegment.fileTimes.end : trackPoints?.at(-1)?.time;

    if (startTime && endTime) {
        trail.date = startTime.toISOString()
            .substring(0, 10);
    }

    trail.duration = totals.duration / 1000
    trail.elevation_gain = totals.elevationGain;
    trail.elevation_loss = totals.elevationLoss;
    trail.distance = totals.distance

    const bbox = totals.boundingBox;
    if (Number.isFinite(bbox.minLat) && Number.isFinite(bbox.maxLat)) {
        trail.min_lat = bbox.minLat;
        trail.max_lat = bbox.maxLat;
        trail.min_lon = bbox.minLon;
        trail.max_lon = bbox.maxLon;
    }

    return { gpx: gpx, trail: trail }
}

export async function trail2gpx(trail: Trail, user?: AuthRecord) {
    let gpxTrail = trail;

    if (!trail.expand?.gpx_data) {
        // no gpx_data -> empty trail?
        // or just not expanded? -> expand now
        const response = await trails_show(trail.id!, handleFromRecordWithIRI(trail), undefined, true);

        if (!response.expand?.gpx_data) {
            throw Error("Trail has no GPX data")
        } else {
            gpxTrail = response;
        }
    }

    const gpx = GPX.parse(gpxTrail.expand!.gpx_data!);

    if (gpx instanceof Error) {
        throw gpx;
    }

    // In the element order of the GPX 1.1 schema.
    gpx.metadata = {
        name: trail.name,
        desc: trail.description ?? "",
        author: new Person({ name: trail.author ?? "", email: user?.email ?? "" }),
        time: trail.date ? new Date(trail.date) : new Date(),
        keywords: `${trail.category ?? ""}, ${trail.location ?? ""}`,
    }

    if (gpx.trk && gpx.trk.length > 0) {
        gpx.trk[0].name = trail.name
    }

    if (!gpx.wpt) {
        gpx.wpt = [];
    }

    for (const wp of gpxTrail.expand!.waypoints_via_trail ?? []) {
        let gpxWpt = gpx.wpt.find((w) => w.$.lat == wp.lat && w.$.lon == wp.lon)
        if (!gpxWpt) {
            gpxWpt = new GPXWaypoint({
                $: {
                    lat: wp.lat,
                    lon: wp.lon
                }
            })
            gpx.wpt.push(gpxWpt)
        }
        gpxWpt.desc = wp.description
        gpxWpt.name = wp.name
        gpxWpt.sym = wp.icon
    }

    joinContiguousTrackSegments(gpx);

    return gpx.toString();
}

export const trailGpxFields = ["distance", "duration", "elevation_gain", "elevation_loss", "lat", "lon"] as const;
export const summitLogGpxFields = ["distance", "duration", "elevation_gain", "elevation_loss"] as const;
const positionFields: readonly string[] = ["lat", "lon"];

export async function applyGpxToForm(data: FormData, fields: readonly (typeof trailGpxFields)[number][], correctElevation: boolean, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const file = data.get("gpx");
    if (!(file instanceof Blob) || file.size === 0) {
        return;
    }

    let gpxFile: Blob;
    let gpx: GPX;
    let trail: Trail;
    try {
        const converted = await fromUpload(file);
        gpxFile = converted.gpxFile;
        ({ gpx, trail } = await gpx2trail(converted.gpxData, undefined, correctElevation, f));
    } catch (e) {
        if (e instanceof FileTooLargeError || e instanceof ClientResponseError) {
            throw e;
        }
        console.error(e);
        throw new ClientResponseError({ status: 400, response: { message: "Invalid file" } });
    }

    if (gpxFile !== file) {
        const name = file instanceof File ? file.name.replace(/\.[^.]*$/, "") + ".gpx" : "trail.gpx";
        data.set("gpx", gpxFile, name);
    }

    // Statistics are computed from track points only. A route-only file
    // (<rte>/<rtept>) yields zeros, which must not replace the stored values.
    const hasTrackPoints = gpx.trk?.some((trk) => trk.trkseg?.some((seg) => (seg.trkpt?.length ?? 0) > 0)) ?? false;

    // Moving time describes the track that was replaced, and nothing here can
    // compute a new one. Cleared as 0 because only a value overwrites the record.
    if (fields === trailGpxFields && !data.has("moving_duration")) {
        data.set("moving_duration", "0");
    }

    for (const field of fields) {
        if (!hasTrackPoints && !positionFields.includes(field)) {
            continue;
        }
        if (!data.has(field) && trail[field] !== undefined) {
            data.set(field, String(trail[field]));
        }
    }
}

/**
 * `fromFile` for a request: a file that cannot be read is the client's error,
 * and what the parsers have to say about it stays in the log.
 */
export async function fromUpload(file: File | Blob) {
    try {
        return await fromFile(file);
    } catch (e) {
        if (e instanceof FileTooLargeError) {
            throw e;
        }
        console.error(e);
        throw new ClientResponseError({ status: 400, response: { message: "Invalid file" } });
    }
}

// A cheap check on the raw text, so that ordinary track files are not parsed twice.
function isRouteOnlyGPX(content: string) {
    return /<(?:[\w.-]+:)?rte[\s>]/.test(content) && !/<(?:[\w.-]+:)?trk[\s>]/.test(content);
}

export async function fromFile(file: File | Blob) {
    assertServerLimit(file.size, SERVER_LIMITS.fileBytes);

    let gpxData = "";
    let gpxFile: Blob;
    const fileBuffer = await file.arrayBuffer();

    if (isFITFile(fileBuffer)) {
        assertServerLimit(file.size, SERVER_LIMITS.fitBytes);
        gpxData = await fromFIT(fileBuffer);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
        return { gpxData, gpxFile };
    }

    if (isKMZFile(fileBuffer)) {
        assertServerLimit(file.size, SERVER_LIMITS.kmzBytes);
        gpxData = await fromKMZ(fileBuffer);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
        return { gpxData, gpxFile };
    }

    const fileContent = new TextDecoder().decode(fileBuffer);

    if (fileContent.includes("http://www.opengis.net/kml")) {
        assertServerLimit(file.size, SERVER_LIMITS.kmlBytes);
        gpxData = fromKML(fileContent);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
    } else if (fileContent.includes("TrainingCenterDatabase")) {
        gpxData = fromTCX(fileContent);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
    } else if (isRouteOnlyGPX(fileContent)) {
        assertGpxWithinServerLimits(fileContent);
        // GaiaGPS and similar planners export saved routes as <rte> with no
        // <trk>. Store them as tracks so metrics, the map and export work (#1356).
        const gpx = GPX.parse(fileContent);
        gpx.convertRoutesToTracks();
        assertServerLimit(gpx.flatten().length, SERVER_LIMITS.convertedPoints);
        gpxData = gpx.toString();
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
    } else {
        assertGpxWithinServerLimits(fileContent);
        gpxData = fileContent;
        gpxFile = file;
    }


    return { gpxData, gpxFile };
}

export function fromKML(kmlData: string) {
    assertServerLimit(countTags(kmlData), SERVER_LIMITS.domTags);
    const parser = browser ? new DOMParser() : new XMLDOMParser();
    const nodes = parser.parseFromString(kmlData, "text/xml")
    const geojson = kml(nodes) as Feature | FeatureCollection

    const gpx = fromGeoJSON(geojson);

    return gpx
}

const MiB = 1024 * 1024;

// What one file may cost on the server, where parsing runs in the same process
// as everything else and running out of heap kills it. Measured on a production
// build with --max-old-space-size=512; see the numbers in PR #1397. Browsers
// parse their own uploads, so only the KMZ bound applies to them.
const SERVER_LIMITS = {
    fileBytes: 25 * MiB,
    // JSZip keeps an object per zip entry, and an empty entry costs ~80 bytes.
    kmzBytes: 8 * MiB,
    // Every "1,2 " in a KML becomes an array in toGeoJSON.
    kmlBytes: 4 * MiB,
    fitBytes: 8 * MiB,
    // Elements in the xmldom tree that KML and TCX are read from.
    domTags: 400_000,
    // Elements and points in xml2js' objects, which GPX is read into.
    gpxTags: 750_000,
    gpxPoints: 150_000,
    // Points that a conversion may hand to the GPX serializer.
    convertedPoints: 60_000,
};
const MAX_BROWSER_KMZ_DOCUMENT_BYTES = 100 * MiB;

export class FileTooLargeError extends Error {
    status = 413;

    constructor(message = "File too large") {
        super(message);
    }
}

function assertServerLimit(value: number, limit: number) {
    if (!browser && value > limit) {
        throw new FileTooLargeError();
    }
}

function countTags(xml: string) {
    let count = 0;
    for (let i = xml.indexOf("<"); i !== -1; i = xml.indexOf("<", i + 1)) {
        count++;
    }
    return count;
}

/** Rejects a GPX document that the server could not parse without running out of memory. */
export function assertGpxWithinServerLimits(gpxData: string) {
    assertServerLimit(countTags(gpxData), SERVER_LIMITS.gpxTags);
    let points = 0;
    for (const _ of gpxData.matchAll(/<(?:[\w.-]+:)?(?:trkpt|rtept|wpt)[\s/>]/g)) {
        points++;
    }
    assertServerLimit(points, SERVER_LIMITS.gpxPoints);
}

type ZipStream = {
    on(event: "data", listener: (chunk: Uint8Array) => void): ZipStream;
    on(event: "error", listener: (error: Error) => void): ZipStream;
    on(event: "end", listener: () => void): ZipStream;
    pause(): void;
    resume(): void;
};

// Counts the bytes as they inflate, because the size in the zip directory is
// whatever the uploader wrote. Not nodeStream: this also runs in the browser.
function inflateEntry(entry: JSZip.JSZipObject, maxBytes: number) {
    return new Promise<string>((resolve, reject) => {
        const decoder = new TextDecoder();
        const stream = (entry as unknown as { internalStream(type: "uint8array"): ZipStream }).internalStream("uint8array");
        let total = 0;
        let text = "";
        stream
            .on("data", (chunk) => {
                total += chunk.length;
                if (total > maxBytes) {
                    stream.pause();
                    text = "";
                    reject(new FileTooLargeError());
                    return;
                }
                text += decoder.decode(chunk, { stream: true });
            })
            .on("error", reject)
            .on("end", () => resolve(text + decoder.decode()))
            .resume();
    });
}

export async function fromKMZ(kmzData: ArrayBuffer, maxDocumentBytes = browser ? MAX_BROWSER_KMZ_DOCUMENT_BYTES : SERVER_LIMITS.kmlBytes) {
    const zip = new JSZip()
    await zip.loadAsync(kmzData)
    const entry = zip.file('doc.kml');
    if (!entry) {
        throw new Error("KMZ has no doc.kml");
    }
    return fromKML(await inflateEntry(entry, maxDocumentBytes))
}

export function fromTCX(tcxData: string) {
    assertServerLimit(countTags(tcxData), SERVER_LIMITS.domTags);
    const parser = browser ? new DOMParser() : new XMLDOMParser();
    const nodes = parser.parseFromString(tcxData, "text/xml")
    const geojson = tcx(nodes) as Feature | FeatureCollection

    const gpx = fromGeoJSON(geojson);

    return gpx
}

export async function fromFIT(fitData: ArrayBuffer) {
    const fitParser = new FitParser();
    return new Promise<string>((resolve, reject) => fitParser.parse(fitData, function (error, data) {

        if (error) {
            console.error(error);
            reject(error)
        }

        assertServerLimit(data?.records?.length ?? 0, SERVER_LIMITS.convertedPoints);

        const gpx = new GPX({
            metadata: {
                name: "",
                desc: "",
                time: data?.activity?.timestamp ?? new Date(),
                keywords: "wanderer"
            },
            trk: [
                new Track({
                    trkseg: new TrackSegment({
                        trkpt: data?.records?.flatMap(((d) => {
                            return (d.position_lat && d.position_long) ? new GPXWaypoint({
                                $: { lat: d.position_lat, lon: d.position_long },
                                time: d.timestamp,
                                ele: d.altitude ?? d.enhanced_altitude

                            }) : []
                        }))
                    })
                })
            ]
        });
        resolve(gpx.toString());
    }));
}

export function fromGeoJSON(geoJson: Feature | FeatureCollection) {
    const gpx = new GPX({
        metadata: {
            name: "",
            desc: "",
            time: new Date(),
            keywords: "wanderer"
        },
        trk: [],
        wpt: []
    });

    const type = geoJson.type;

    function addSupportedPropertiesFromObject(el: any, supports: string[], properties: GeoJsonProperties) {
        if (properties && typeof properties === 'object') {
            supports.forEach(function (key: any) {
                var value = properties[key];
                if (value && typeof value === 'string' && supports.includes(key)) {
                    el[key] = value;
                }
            });
        }
    }
    function createTrk(properties: GeoJsonProperties) {
        var trk = new Track({ trkseg: [] });
        var supports = ['name', 'desc', 'src', 'type'];
        addSupportedPropertiesFromObject(trk, supports, properties);
        return trk;
    }
    function createPt(position: Position, properties?: GeoJsonProperties) {
        const pt = new GPXWaypoint({ $: { lat: position[1], lon: position[0] }, ele: position[2], time: position[3] as any });
        const supports = ['name', 'desc', 'src', 'type'];
        if (properties) {
            addSupportedPropertiesFromObject(pt, supports, properties);
        }
        return pt;
    }
    function createTrkSeg(coordinates: Position[]) {
        var trkSeg = new TrackSegment({ trkpt: [] });
        coordinates.forEach(function (point) {
            trkSeg.trkpt!.push(createPt(point));
        });
        return trkSeg;
    }
    function interpretFeature(feature: Feature) {
        const geometry = feature.geometry
        if (!geometry) {
            return
        }
        const properties = feature.properties;
        const type = geometry.type;
        switch (type) {
            case 'Polygon':
                break;
            case 'Point': {
                gpx.wpt!.push(createPt(geometry.coordinates, properties));
                break;
            }
            case 'MultiPoint': {
                geometry.coordinates.forEach(function (coord) {
                    gpx.wpt!.push(createPt(coord, properties));
                });
                break;
            }
            case 'LineString': {
                var lineTrk = createTrk(properties);
                var trkseg = createTrkSeg(geometry.coordinates);
                lineTrk.trkseg!.push(trkseg);
                gpx.trk!.push(lineTrk);
                break;
            }
            case 'MultiLineString': {
                var trk_1 = createTrk(properties);
                geometry.coordinates.forEach(function (pos) {
                    var trkseg = createTrkSeg(pos);
                    trk_1.trkseg!.push(trkseg);
                });
                gpx.trk!.push(trk_1);
                break;
            }
            default:
                break;
        }
    }
    switch (type) {
        case 'Feature': {
            interpretFeature(geoJson);
            break;
        }
        case 'FeatureCollection': {
            var features = geoJson.features;
            features.forEach(function (feature) {
                interpretFeature(feature);
            });
            break;
        }
        default:
            break;
    }

    assertServerLimit(gpx.flatten().length + (gpx.wpt?.length ?? 0), SERVER_LIMITS.convertedPoints);

    return gpx.toString()
}

export function isFITFile(buffer: ArrayBuffer) {
    var blob = new Uint8Array(buffer);

    if (blob.length < 12) {
        return false
    }

    var headerLength = blob[0];
    if (headerLength !== 14 && headerLength !== 12) {
        return false;
    }

    var fileTypeString = '';
    for (var i = 8; i < 12; i++) {
        fileTypeString += String.fromCharCode(blob[i]);
    }
    if (fileTypeString !== '.FIT') {
        return false
    }

    return true;
}

function isKMZFile(buffer: ArrayBuffer) {
    const blob = new Uint8Array(buffer);
    return blob[0] === 0x50 && blob[1] === 0x4B && blob[2] === 0x03 && blob[3] === 0x04;
}

export function cropGPX(start: GPXWaypoint, end: GPXWaypoint, gpx: GPX): GPX {
    let foundStart = false;
    let done = false;

    const croppedTrk = gpx.trk?.map(track => {
        const croppedSegments: TrackSegment[] = [];

        track.trkseg?.forEach(seg => {
            const newPoints: GPXWaypoint[] = [];

            for (const pt of seg.trkpt ?? []) {
                if (!foundStart) {
                    if (pt === start) {
                        foundStart = true;
                        newPoints.push(pt);
                    }
                    continue;
                }

                if (foundStart && !done) {
                    newPoints.push(pt);
                    if (pt === end) {
                        done = true;
                        break;
                    }
                }
            }

            if (newPoints.length > 0) {
                croppedSegments.push(new TrackSegment({ trkpt: newPoints }));
            }
        });

        return {
            ...track,
            trkseg: croppedSegments,
        };
    }).filter(track => track.trkseg.length > 0);

    return new GPX({ ...gpx, trk: croppedTrk?.map(t => new Track({ ...t })) ?? [], })
}
