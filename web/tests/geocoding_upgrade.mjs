// Linux regression with built binaries: (cd db && go build -o /path/to/pocketbase .)
// and (cd web && npm run build), then `node web/tests/geocoding_upgrade.mjs
// /path/to/pocketbase [path/to/meilisearch]` from the repo root. Requires Node 22,
// the v0.21.2 Git tag, Docker Compose (config only), OpenSSL, ip and
// unprivileged user namespaces.
// Only loopback is available to the running stack; even the public hostname
// resolves to our TLS fixture. No real-provider traffic is possible.
import assert from "node:assert/strict";
import { execFileSync, spawn } from "node:child_process";
import { lookup } from "node:dns/promises";
import { once } from "node:events";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createServer } from "node:http";
import { createServer as createHttpsServer } from "node:https";
import { networkInterfaces, tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { setTimeout as delay } from "node:timers/promises";
import { fileURLToPath } from "node:url";
import { check, collection, failure, fixture, query } from "./geocoding_smoke.mjs";

const root = fileURLToPath(new URL("../../", import.meta.url));
const host = "nominatim.openstreetmap.org";
const sharedSecret = "geocoding-upgrade-fixture-proxy-secret";
const fixturePort = 18808;
const dbOrigin = "http://127.0.0.1:18809";
const webOrigin = "http://127.0.0.1:18810";
const meiliOrigin = "http://127.0.0.1:18811";
const privateURL = `http://127.0.0.1:${fixturePort}/fixture`;
const legacyPrivateURL = "https://legacy-geocoding-fixture.invalid/nominatim";

function run(db, meili) {
    const dir = mkdtempSync(join(tmpdir(), "wanderer-geocoding-"));
    try {
        const historical = join(dir, "historical.yml");
        writeFileSync(historical, execFileSync("git", ["show", "v0.21.2:docker-compose.yml"], { cwd: root }));
        const { services } = JSON.parse(execFileSync("docker", [
            "compose", "--env-file", "/dev/null", "-f", historical, "config", "--format", "json",
        ], { env: { PATH: process.env.PATH, HOME: process.env.HOME, POCKETBASE_PROXY_SECRET: sharedSecret } }));
        for (const name of ["db", "web"]) {
            assert.equal(services[name].environment.POCKETBASE_PROXY_SECRET, sharedSecret);
        }
        assert.equal(services.web.environment.NOMINATIM_URL, `https://${host}`);
        assert(!Object.hasOwn(services.db.environment, "NOMINATIM_URL"));
        assert(!Object.hasOwn(services.db.environment, "PUBLIC_NOMINATIM_URL"));
        writeFileSync(join(dir, "compose.json"), JSON.stringify(services));
        writeFileSync(join(dir, "hosts"), `127.0.0.1 localhost ${host}\n`);
        execFileSync("openssl", [
            "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
            "-subj", `/CN=${host}`, "-addext", `subjectAltName=DNS:${host}`,
            "-keyout", join(dir, "tls.key"), "-out", join(dir, "tls.pem"),
        ], { stdio: "pipe" });
        execFileSync("unshare", ["-Urnmp", "--kill-child", "sh", "-euc", `
            ip link set lo up
            mount --bind "$1/hosts" /etc/hosts
            exec "$2" "$3" --isolated "$1" "$4" "$5"
        `, "geocoding-upgrade", dir, process.execPath, fileURLToPath(import.meta.url), resolve(db), meili],
        { env: { PATH: process.env.PATH }, stdio: "inherit", timeout: 180_000 });
    } finally {
        rmSync(dir, { force: true, recursive: true });
    }
}

function service(bin, args, cwd, env) {
    const child = spawn(bin, args, { cwd, env: { PATH: process.env.PATH, ...env } });
    child.output = "";
    child.on("error", err => { child.output += err.message; });
    for (const stream of [child.stdout, child.stderr]) {
        stream.on("data", chunk => { child.output += chunk; });
    }
    return child;
}

async function ready(child, url) {
    for (let i = 0; i < 150 && child.pid && child.exitCode === null; i++) {
        const res = await fetch(url, { redirect: "error", signal: AbortSignal.timeout(500) }).catch(() => null);
        await res?.body?.cancel();
        if (res?.ok) return;
        await delay(200);
    }
    throw new Error(`Service did not become ready at ${url}:\n${child.output}`);
}

async function stop(child) {
    if (child.pid && child.exitCode === null && child.signalCode === null) {
        const exit = once(child, "exit");
        child.kill();
        await exit;
    }
}

async function checkDevDotenv(dir) {
    const { createServer: createViteServer } = await import("vite");
    const { sveltekit } = await import("@sveltejs/kit/vite");
    const values = { NOMINATIM_URL: privateURL, PUBLIC_NOMINATIM_URL: legacyPrivateURL };
    writeFileSync(join(dir, ".env.local"), Object.entries(values).map(([key, value]) => `${key}=${value}\n`).join(""));
    const outDir = join(dir, "dev-kit");
    const server = await createViteServer({
        configFile: false, mode: "development", cacheDir: join(dir, "vite-cache"),
        plugins: [sveltekit({ outDir, env: { dir } })],
        server: { host: "127.0.0.1", port: 18812, strictPort: true, hmr: false, watch: null },
    });
    try {
        await server.listen();
        const read = file => readFileSync(join(outDir, "generated/dev/env", file), "utf8");
        const config = read("config.js");
        for (const [key, value] of Object.entries(values)) {
            assert.equal(process.env[key], undefined);
            assert(config.includes(`dynamic_private_env.${key} = ${key};`) && config.includes(value));
            assert(read("private/server.js").includes(`export const ${key} = env.${key};`));
            for (const file of ["public/server.js", "public/client.js", "service-worker.js"]) {
                assert(!read(file).includes(key) && !read(file).includes(value));
            }
        }
        console.log("Vite .env.local: both Nominatim settings loaded only in private generated modules.");
    } finally {
        await server.close();
    }
}

async function isolated(dir, dbBin, meiliBin) {
    assert.deepEqual(Object.keys(networkInterfaces()), ["lo"]);
    assert.equal((await lookup(host)).address, "127.0.0.1");
    process.chdir(join(root, "web"));
    await checkDevDotenv(dir);
    const { db, web } = JSON.parse(readFileSync(join(dir, "compose.json")));
    const cert = join(dir, "tls.pem");
    const publicFixture = createHttpsServer({ key: readFileSync(join(dir, "tls.key")), cert: readFileSync(cert) },
        (req, res) => fixture(req, res, ""));
    const privateFixture = createServer(fixture);
    let publicDials = 0;
    let privateDials = 0;
    publicFixture.on("connection", () => publicDials++);
    privateFixture.on("connection", () => privateDials++);
    await Promise.all([
        once(publicFixture.listen(443, "127.0.0.1"), "listening"),
        once(privateFixture.listen(fixturePort, "127.0.0.1"), "listening"),
    ]);

    const shared = {
        HOME: dir, ORIGIN: webOrigin, MEILI_URL: meiliOrigin, MEILI_MASTER_KEY: "geocoding-upgrade-fixture-meili-master-key",
    };
    const meili = service(meiliBin, [], dir, {
        ...shared, MEILI_HTTP_ADDR: new URL(meiliOrigin).host, MEILI_NO_ANALYTICS: "true",
    });
    try {
        await ready(meili, `${meiliOrigin}/health`);
        for (const [name, dbEnv, webEnv, expectedStatus, expectedDials] of [
            ["web-only", {}, { NOMINATIM_URL: privateURL }, 502, [0, 0]],
            ["legacy-web-only", {}, { NOMINATIM_URL: undefined, PUBLIC_NOMINATIM_URL: legacyPrivateURL }, 502, [0, 0]],
            ["blank-web-canonical", {}, { NOMINATIM_URL: "", PUBLIC_NOMINATIM_URL: `https://${host}` }, 502, [0, 0]],
            ["explicit-public-db", { NOMINATIM_URL: `https://${host}` }, { NOMINATIM_URL: privateURL }, 502, [0, 0]],
            ["migrated", { NOMINATIM_URL: privateURL }, { NOMINATIM_URL: privateURL }, 200, [0, 2]],
            ["public-default", {}, {}, 200, [2, 0]],
            ["wrong-proxy-secret", { NOMINATIM_URL: privateURL }, { POCKETBASE_PROXY_SECRET: "wrong-fixture-secret" }, 502, [0, 0]],
        ]) {
            publicDials = 0;
            privateDials = 0;
            const backend = service(dbBin, ["serve", "--dev", `--http=${new URL(dbOrigin).host}`, `--dir=${join(dir, name)}`],
                join(root, "db"), { ...db.environment, ...shared, SSL_CERT_FILE: cert, ...dbEnv });
            const frontend = service(process.execPath, ["server.js"], join(root, "web"), {
                ...web.environment, ...shared, PUBLIC_POCKETBASE_URL: dbOrigin,
                HOST: "127.0.0.1", PORT: new URL(webOrigin).port, ...webEnv,
            });
            try {
                await ready(backend, `${dbOrigin}/health`);
                await ready(frontend, `${webOrigin}/api/v1/health`);
                const envModule = await fetch(`${webOrigin}/_app/env.js`, { redirect: "error", signal: AbortSignal.timeout(5_000) });
                assert.equal(envModule.status, 200);
                const publicEnv = await envModule.text();
                assert.match(publicEnv, /\bPUBLIC_POCKETBASE_URL\b/);
                assert.doesNotMatch(publicEnv, /NOMINATIM_URL|private-required|X-Wanderer-Geocoding|internal_proxy_auth/i);
                assert(!publicEnv.includes(privateURL) && !publicEnv.includes(legacyPrivateURL));
                for (const endpoint of [`search?q=${encodeURIComponent(query)}&limit=0x10`, "reverse?lat=%2B4.75e1&lon=0xB"]) {
                    const res = await fetch(`${webOrigin}/api/v1/geocoding/${endpoint}`, {
                        redirect: "error", signal: AbortSignal.timeout(10_000),
                    });
                    assert.equal(res.status, expectedStatus, name);
                    assert.deepEqual(await res.json(), expectedStatus === 200 ? collection : failure, name);
                    assert.equal(res.headers.get("X-Wanderer-Geocoding-Failure"), null);
                    assert.equal(res.headers.get("X-Wanderer-Geocoding-Privacy"), null);
                }
                assert.deepEqual([publicDials, privateDials], expectedDials, `${name}: unexpected fixture connections`);
                if (expectedStatus === 502 && name !== "wrong-proxy-secret") {
                    assert.equal(backend.output.split("private_nominatim_required").length - 1, 1);
                    assert(!backend.output.includes("q=Fixture") && !backend.output.includes("lon=0xB"));
                } else if (name === "migrated") {
                    await check(webOrigin, dbOrigin);
                } else if (name === "wrong-proxy-secret") {
                    assert(frontend.output.includes("db rejected the internal proxy secret"));
                }
                console.log(`${name}: both routes passed (public/private fixture dials: ${expectedDials.join("/")})`);
            } finally {
                await Promise.all([stop(frontend), stop(backend)]);
            }
        }
    } finally {
        await stop(meili);
        publicFixture.close();
        privateFixture.close();
    }
}

const [db, meili, ...args] = process.argv.slice(2);
if (db === "--isolated") {
    await isolated(meili, ...args);
} else {
    assert(db, "Usage: node web/tests/geocoding_upgrade.mjs <pocketbase-binary> [meilisearch-binary]");
    run(db, meili ? resolve(meili) : "meilisearch");
}
