// For a disposable stack: start this file with `upstream 127.0.0.1 <port>`,
// set NOMINATIM_URL=http://127.0.0.1:<port>/fixture on db, then run
// `node tests/geocoding_smoke.mjs check <web-origin> <db-origin>`.
import assert from "node:assert/strict";
import { createServer } from "node:http";

const query = "Fixture + 湖";
const collection = {
    type: "FeatureCollection",
    licence: "Geocoding smoke fixture",
    features: [{
        type: "Feature",
        geometry: { type: "Point", coordinates: [11, 47.5] },
        properties: { name: query, address: { town: query } },
    }],
};
const failure = { message: "Nominatim request failed" };
const noAddress = { error: "Unable to geocode" };

function fixture(req, res) {
    function json(status, payload, headers = {}) {
        res.writeHead(status, { "Content-Type": "application/json", ...headers });
        res.end(JSON.stringify(payload));
    }

    const { pathname, searchParams: params } = new URL(req.url, "http://fixture");
    if (pathname === "/health") {
        return json(200, {});
    }

    const search = pathname === "/fixture/search";
    if (!search && pathname !== "/fixture/reverse") {
        return json(404, {});
    }
    const keys = [...params.keys()];
    const allowed = ["format", "addressdetails", ...(search ? ["q", "limit"] : ["lat", "lon"])];
    const privateHeaders = ["cookie", "authorization", "x-internal-secret", "x-forwarded-for"];
    if (req.method !== "GET" || keys.some(key => !allowed.includes(key)) || new Set(keys).size !== keys.length ||
        params.get("format") !== "geojson" || params.get("addressdetails") !== "1" ||
        privateHeaders.some(key => Object.hasOwn(req.headers, key))) {
        return json(400, { message: "Unexpected upstream request" });
    }

    if (search && params.get("q") === "fixture rate limited") {
        return json(429, { error: "private upstream detail" }, { "Retry-After": "3600" });
    }
    if (search && params.get("q") === "fixture empty") {
        res.writeHead(204);
        return res.end();
    }
    if (search && params.get("q") === "fixture empty 200") {
        res.writeHead(200);
        return res.end();
    }
    if (!search && params.get("lat") === "0" && params.get("lon") === "0") {
        return json(200, noAddress);
    }
    if (!search && params.get("lat") === "1" && params.get("lon") === "1") {
        return json(404, { error: "private upstream detail" });
    }

    const matches = search
        ? params.get("q") === query && params.get("limit") === "0x10"
        : params.get("lat") === "+4.75e1" && params.get("lon") === "0xB";
    return json(search && matches ? 202 : 200, matches ? collection : { type: "FeatureCollection", features: [] });
}

async function check(web, db) {
    async function get(base, path, params = [], headers = {}) {
        const url = new URL(path, base);
        url.search = new URLSearchParams(params).toString();
        return fetch(url, { headers, redirect: "error", signal: AbortSignal.timeout(10_000) });
    }

    const headers = { "X-Internal-Secret": "visitor-supplied", "X-Forwarded-For": "192.0.2.1" };
    for (const [endpoint, args, missing, upstreamStatus, upstreamArgs] of [
        ["search", [["q", query], ["q", "ignored"], ["limit", "0x10"]], "q", 429, [["q", "fixture rate limited"]]],
        ["reverse", [["lat", "+4.75e1"], ["lon", "0xB"], ["lat", "invalid"]], "lat or lon", 404, [["lat", "1"], ["lon", "1"]]],
    ]) {
        const api = `/api/v1/geocoding/${endpoint}`;
        const valid = await get(web, api, [...args, ["format", "json"], ["addressdetails", "0"], ["url", "http://ignored.local"]], headers);
        assert.equal(valid.status, 200, `${endpoint}: valid request failed`);
        assert.deepEqual(await valid.json(), collection);

        const invalid = await get(web, api);
        assert.equal(invalid.status, 400);
        assert.deepEqual(await invalid.json(), { message: `Missing query parameter: ${missing}` });

        const error = await get(web, api, upstreamArgs);
        assert.equal(error.status, upstreamStatus);
        assert.deepEqual(await error.json(), failure);

        const direct = await get(db, `/geocoding/${endpoint}`, args, headers);
        assert.equal(direct.status, 401, `${endpoint}: DB accepted a request with no valid secret`);
        await direct.body?.cancel();
    }
    const noResult = await get(web, "/api/v1/geocoding/reverse", [["lat", "0"], ["lon", "0"]]);
    assert.equal(noResult.status, 200);
    assert.deepEqual(await noResult.json(), noAddress);
    for (const query of ["fixture empty", "fixture empty 200"]) {
        const empty = await get(web, "/api/v1/geocoding/search", [["q", query]]);
        assert.equal(empty.status, 200);
        assert.deepEqual(await empty.json(), {});
    }
    console.log("Search and reverse geocoding smoke checks passed.");
}

const [command, first, second] = process.argv.slice(2);
if (command === "upstream" && first && second) {
    createServer(fixture).listen(Number(second), first, () => console.log("Geocoding smoke upstream ready."));
} else if (command === "check" && first && second) {
    await check(first, second);
} else {
    console.error("Usage: node geocoding_smoke.mjs upstream <host> <port> | check <web-origin> <db-origin>");
    process.exitCode = 1;
}
