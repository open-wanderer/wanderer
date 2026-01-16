import GPX from "$lib/models/gpx/gpx";
import { Trail } from "$lib/models/trail";
import { gpx, kml, tcx } from "$lib/vendor/toGeoJSON/toGeoJSON";
import cryptoRandomString from "crypto-random-string";
//@ts-ignore
import { browser } from "$app/environment";
import Track from "$lib/models/gpx/track";
import TrackSegment from "$lib/models/gpx/track-segment";
import GPXWaypoint from "$lib/models/gpx/waypoint";
import EasyFit from "$lib/vendor/easy-fit/easy-fit";
import type { Feature, FeatureCollection, GeoJSON, GeoJsonProperties, Position } from 'geojson';
import JSZip from "jszip";
import type { AuthRecord } from "pocketbase";
import * as xmldom from 'xmldom';
import { bbox, splitMultiLineStringToLineStrings } from "./geojson_util";
import { trails_show } from "$lib/stores/trail_store";
import { handleFromRecordWithIRI } from "./activitypub_util";
import { Waypoint } from "$lib/models/waypoint";


export async function gpx2trail(gpxString: string, fallbackName?: string, correctElevation: boolean = false, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const gpx = GPX.parse(gpxString);

    if (gpx instanceof Error) {
        throw gpx;
    }
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
        const wp = new Waypoint(wpt.$.lat ?? 0, wpt.$.lon ?? 0);
        wp.id = cryptoRandomString({ length: 15 });
        wp.name = wpt.name ?? ""
        wp.description = wpt.desc;
        trail.expand!.waypoints_via_trail?.push(wp);
    }

    const totals = gpx.features

    const trackPoints = gpx.trk?.at(0)?.trkseg?.at(0)?.trkpt
    const routePoints = gpx.rte?.at(0)?.rtept;

    const startPoint = trackPoints?.at(0) ?? routePoints?.at(0);
    if (startPoint) {
        trail.lat = startPoint.$.lat
        trail.lon = startPoint.$.lon
    }

    const startTime = trackPoints?.at(0)?.time;
    const endTime = trackPoints?.at((trackPoints?.length ?? 1) - 1)?.time

    if (startTime && endTime) {
        trail.date = startTime.toISOString()
            .substring(0, 10);
    }

    trail.duration = totals.duration / 1000
    trail.elevation_gain = totals.elevationGain;
    trail.elevation_loss = totals.elevationLoss;
    trail.distance = totals.distance

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

    gpx.metadata = {
        name: trail.name,
        desc: trail.description ?? "",
        time: trail.date ? new Date(trail.date) : new Date(),
        keywords: `${trail.category ?? ""}, ${trail.location ?? ""}`,
        author: { name: trail.author ?? "", email: user?.email ?? "" }
    }

    if (!gpx.wpt) {
        gpx.wpt = [];
    }

    for (const wp of gpxTrail.expand!.waypoints_via_trail ?? []) {
        const gpxWpt = gpx.wpt.find((w) => w.$.lat == wp.lat && w.$.lon == wp.lon)
        if (!gpxWpt) {
            gpx.wpt.push(new GPXWaypoint({
                $: {
                    lat: wp.lat,
                    lon: wp.lon
                }
            }))
        }
    }

    return gpx.toString();
}

export async function fromFile(file: File | Blob) {
    let gpxData = "";
    let gpxFile: Blob;
    const fileContent = await file.text();
    const fileBuffer = await file.arrayBuffer();

    if (isFITFile(fileBuffer)) {
        gpxData = await fromFIT(fileBuffer);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });

    } else if (isKMZFile(fileBuffer)) {
        gpxData = await fromKMZ(fileBuffer);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });

    } else if (fileContent.includes("http://www.opengis.net/kml")) {
        gpxData = fromKML(fileContent);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
    } else if (fileContent.includes("TrainingCenterDatabase")) {
        gpxData = fromTCX(fileContent);
        gpxFile = new Blob([gpxData], {
            type: "application/gpx+xml",
        });
    } else {
        gpxData = fileContent;
        gpxFile = file;
    }


    return { gpxData, gpxFile };
}

export function fromKML(kmlData: string) {
    const parser = browser ? new DOMParser() : new xmldom.DOMParser();
    const nodes = parser.parseFromString(kmlData, "text/xml")
    const geojson = kml(nodes) as Feature | FeatureCollection

    const gpx = fromGeoJSON(geojson);

    return gpx
}

export async function fromKMZ(kmzData: ArrayBuffer) {
    const zip = new JSZip()
    const zipContents = await zip.loadAsync(kmzData)
    const kmlFile = await zip.file('doc.kml')?.async('string');
    return fromKML(kmlFile ?? "")
}

export function fromTCX(tcxData: string) {
    const parser = browser ? new DOMParser() : new xmldom.DOMParser();
    const nodes = parser.parseFromString(tcxData, "text/xml")
    const geojson = tcx(nodes) as Feature | FeatureCollection

    const gpx = fromGeoJSON(geojson);

    return gpx
}

export async function fromFIT(fitData: ArrayBuffer) {
    const easyFit = new EasyFit();
    return new Promise<string>((resolve, reject) => easyFit.parse(fitData, function (error, data) {

        if (error) {
            console.error(error);
            reject(error)
        }

        const gpx = new GPX({
            metadata: {
                name: "",
                desc: "",
                time: data.activity?.timestamp ?? new Date(),
                keywords: "wanderer"
            },
            trk: [
                new Track({
                    trkseg: new TrackSegment({
                        trkpt: data.records.flatMap(((d: { position_lat: any; position_long: any; timestamp: any; altitude: any; }) => {
                            return (d.position_lat && d.position_long) ? new GPXWaypoint({
                                $: { lat: d.position_lat, lon: d.position_long },
                                time: d.timestamp,
                                ele: d.altitude

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