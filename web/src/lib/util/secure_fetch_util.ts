import dns from "node:dns/promises";
import net from "node:net";
import http from "node:http";
import https from "node:https";

// Defense-in-depth cap: we buffer the response in memory before returning it.
const MAX_DOWNLOAD_BYTES = 100 * 1024 * 1024; // 100 MB
const REQUEST_TIMEOUT_MS = 15_000;
const MAX_REDIRECTS = 5;

/**
 * Error raised while fetching an external URL. `status` is the HTTP status the
 * caller should surface (validation/redirect failures are 400, timeouts 504).
 */
export class DownloadError extends Error {
    status: number;
    constructor(message: string, status = 400) {
        super(message);
        this.status = status;
    }
}

// ---------------------------------------------------------------------------
// IP range validation (SSRF — GHSA-7vqq-mjjr-h9j5)
// ---------------------------------------------------------------------------

function ipv4ToBytes(ip: string): number[] | null {
    const parts = ip.split(".");
    if (parts.length !== 4) return null;
    const bytes = parts.map((p) => (/^\d{1,3}$/.test(p) ? Number(p) : NaN));
    if (bytes.some((b) => Number.isNaN(b) || b < 0 || b > 255)) return null;
    return bytes;
}

function isBlockedIPv4(ip: string): boolean {
    const b = ipv4ToBytes(ip);
    if (!b) return true;
    const [a, x, c] = b;
    if (a === 0) return true; // 0.0.0.0/8 "this network"
    if (a === 10) return true; // 10.0.0.0/8 private
    if (a === 100 && x >= 64 && x <= 127) return true; // 100.64.0.0/10 CGNAT
    if (a === 127) return true; // 127.0.0.0/8 loopback
    if (a === 169 && x === 254) return true; // 169.254.0.0/16 link-local
    if (a === 172 && x >= 16 && x <= 31) return true; // 172.16.0.0/12 private
    if (a === 192 && x === 0 && c === 0) return true; // 192.0.0.0/24 IETF protocol
    if (a === 192 && x === 0 && c === 2) return true; // 192.0.2.0/24 TEST-NET-1
    if (a === 192 && x === 168) return true; // 192.168.0.0/16 private
    if (a === 198 && (x === 18 || x === 19)) return true; // 198.18.0.0/15 benchmarking
    if (a === 198 && x === 51 && c === 100) return true; // 198.51.100.0/24 TEST-NET-2
    if (a === 203 && x === 0 && c === 113) return true; // 203.0.113.0/24 TEST-NET-3
    if (a >= 224 && a <= 239) return true; // 224.0.0.0/4 multicast
    if (a >= 240) return true; // 240.0.0.0/4 reserved (incl. 255.255.255.255 broadcast)
    return false;
}

// Parse an IPv6 literal (incl. `::` compression and embedded IPv4) into 16 bytes.
function ipv6ToBytes(ip: string): number[] | null {
    let s = ip;
    const zone = s.indexOf("%");
    if (zone !== -1) s = s.slice(0, zone); // strip scope/zone id
    if (s.length === 0) return null;

    // Convert a trailing embedded IPv4 (e.g. ::ffff:1.2.3.4) into two hextets.
    const lastColon = s.lastIndexOf(":");
    if (lastColon !== -1 && s.slice(lastColon + 1).includes(".")) {
        const v4 = ipv4ToBytes(s.slice(lastColon + 1));
        if (!v4) return null;
        const hi = ((v4[0] << 8) | v4[1]).toString(16);
        const lo = ((v4[2] << 8) | v4[3]).toString(16);
        s = s.slice(0, lastColon + 1) + hi + ":" + lo;
    }

    const halves = s.split("::");
    if (halves.length > 2) return null;

    const parseGroups = (part: string): number[] | null => {
        if (part === "") return [];
        const out: number[] = [];
        for (const g of part.split(":")) {
            if (!/^[0-9a-fA-F]{1,4}$/.test(g)) return null;
            const v = parseInt(g, 16);
            out.push((v >> 8) & 0xff, v & 0xff);
        }
        return out;
    };

    const head = parseGroups(halves[0]);
    if (head === null) return null;

    if (halves.length === 2) {
        const tail = parseGroups(halves[1]);
        if (tail === null) return null;
        const missing = 16 - head.length - tail.length;
        if (missing < 0) return null;
        return [...head, ...new Array(missing).fill(0), ...tail];
    }

    return head.length === 16 ? head : null;
}

function isBlockedIPv6(ip: string): boolean {
    const b = ipv6ToBytes(ip);
    if (!b) return true;

    const isZero = (from: number, to: number) => b.slice(from, to).every((x) => x === 0);
    const embeddedV4 = () => b.slice(12, 16).join(".");

    // ::/128 unspecified, ::1/128 loopback
    if (isZero(0, 15) && (b[15] === 0 || b[15] === 1)) return true;
    if (b[0] === 0xff) return true; // ff00::/8 multicast
    if (b[0] === 0xfe && (b[1] & 0xc0) === 0x80) return true; // fe80::/10 link-local
    if ((b[0] & 0xfe) === 0xfc) return true; // fc00::/7 unique local
    if (b[0] === 0x01 && b[1] === 0x00 && isZero(2, 8)) return true; // 100::/64 discard

    // 2001:db8::/32 documentation, 2001:2::/48 benchmarking, 2001:20::/28 ORCHIDv2
    if (b[0] === 0x20 && b[1] === 0x01 && b[2] === 0x0d && b[3] === 0xb8) return true;
    if (b[0] === 0x20 && b[1] === 0x01 && b[2] === 0x00 && b[3] === 0x02 && isZero(4, 6)) return true;
    if (b[0] === 0x20 && b[1] === 0x01 && b[2] === 0x00 && (b[3] & 0xf0) === 0x20) return true;

    // IPv4-mapped ::ffff:0:0/96 → check the embedded IPv4
    if (isZero(0, 10) && b[10] === 0xff && b[11] === 0xff) return isBlockedIPv4(embeddedV4());
    // 64:ff9b::/96 NAT64 → embedded IPv4
    if (b[0] === 0x00 && b[1] === 0x64 && b[2] === 0xff && b[3] === 0x9b && isZero(4, 12)) {
        return isBlockedIPv4(embeddedV4());
    }
    // ::/96 IPv4-compatible (deprecated) → embedded IPv4
    if (isZero(0, 12)) return isBlockedIPv4(embeddedV4());
    // 2002::/16 6to4 → IPv4 encoded in bytes 2..6
    if (b[0] === 0x20 && b[1] === 0x02) return isBlockedIPv4(b.slice(2, 6).join("."));
    // 2001::/32 Teredo → client IPv4 = last 4 bytes XOR 0xff
    if (b[0] === 0x20 && b[1] === 0x01 && b[2] === 0x00 && b[3] === 0x00) {
        return isBlockedIPv4(b.slice(12, 16).map((n) => n ^ 0xff).join("."));
    }

    return false;
}

function isBlockedAddress(ip: string, family: number): boolean {
    return family === 6 ? isBlockedIPv6(ip) : isBlockedIPv4(ip);
}

// Docker Compose service names for internal-only backends (db:8090, search:7700).
const BLOCKED_HOSTNAMES = new Set(["localhost", "db", "search"]);

function isBlockedHostname(hostname: string): boolean {
    const h = hostname.toLowerCase().replace(/\.$/, ""); // drop a trailing FQDN dot
    return h === "" || BLOCKED_HOSTNAMES.has(h) || h.endsWith(".localhost");
}

type ValidatedTarget = { parsed: URL; ip: string; family: number };

/** Parses a candidate URL (initial request or redirect target) and rejects
 * disallowed protocols/hostnames before any DNS lookup is attempted. */
function parseAndCheckUrl(rawUrl: unknown): URL {
    if (typeof rawUrl !== "string" || rawUrl.length === 0) throw new DownloadError("invalid_url");

    let parsed: URL;
    try {
        parsed = new URL(rawUrl);
    } catch {
        throw new DownloadError("invalid_url");
    }

    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") throw new DownloadError("invalid_url");
    if (isBlockedHostname(parsed.hostname)) throw new DownloadError("invalid_url");

    return parsed;
}

/**
 * Guards against SSRF (GHSA-7vqq-mjjr-h9j5). Resolves the hostname, throws if
 * ANY resolved address is private/reserved, and returns the exact IP to connect
 * to. The caller pins the socket to this IP (see fetchSingleHop) so the
 * validated address is the one actually used — closing the DNS-rebinding
 * time-of-check/time-of-use window. Called again for every redirect hop, so a
 * chain can't bounce through validation on the first URL and land elsewhere.
 */
async function resolveAndValidate(parsed: URL): Promise<ValidatedTarget> {
    // URL hostnames wrap IPv6 literals in brackets; strip them for net/dns.
    const host = parsed.hostname.replace(/^\[|\]$/g, "");

    const literal = net.isIP(host);
    let addresses: { address: string; family: number }[];
    try {
        addresses = literal
            ? [{ address: host, family: literal }]
            : await dns.lookup(host, { all: true });
    } catch {
        throw new DownloadError("invalid_url");
    }

    if (addresses.length === 0) throw new DownloadError("invalid_url");

    // Strict: reject if ANY resolved address is internal (blocks mixed public/
    // private answers where happy-eyeballs might pick the private one).
    for (const { address, family } of addresses) {
        if (isBlockedAddress(address, family)) throw new DownloadError("invalid_url");
    }

    return { parsed, ip: addresses[0].address, family: addresses[0].family };
}

export type ExternalFile = { contentType: string | null; body: Buffer };

type HopResult = { type: "redirect"; location: string } | ({ type: "final" } & ExternalFile);

/**
 * Performs a single request with the socket pinned to `ip` (the validated
 * address). A 3xx response is surfaced as a `redirect` result rather than
 * being followed automatically — the caller re-validates the target and pins
 * a fresh connection for it (see fetchExternalFile).
 */
function fetchSingleHop(target: ValidatedTarget): Promise<HopResult> {
    const { parsed, ip, family } = target;
    const isHttps = parsed.protocol === "https:";
    const client = isHttps ? https : http;

    return new Promise<HopResult>((resolve, reject) => {
        let settled = false;
        const finish = (fn: () => void) => {
            if (settled) return;
            settled = true;
            fn();
        };

        const req = client.request(
            {
                method: "GET",
                protocol: parsed.protocol,
                hostname: parsed.hostname,
                port: parsed.port || (isHttps ? 443 : 80),
                path: `${parsed.pathname}${parsed.search}`,
                servername: isHttps ? parsed.hostname.replace(/^\[|\]$/g, "") : undefined,
                // Pin the socket to the pre-validated IP; never re-resolve. This is
                // what closes the DNS-rebinding TOCTOU window. Node's connect path
                // may call lookup with `{ all: true }` (expecting an address array)
                // or the classic 3-arg form — support both.
                lookup: (_hostname: string, options: { all?: boolean }, cb: (err: NodeJS.ErrnoException | null, address: string | { address: string; family: number }[], family?: number) => void) => {
                    if (options && options.all) {
                        cb(null, [{ address: ip, family }]);
                    } else {
                        cb(null, ip, family);
                    }
                },
                headers: { host: parsed.host, accept: "*/*", "user-agent": "wanderer" },
            },
            (res) => {
                const status = res.statusCode ?? 0;
                if (status >= 300 && status < 400) {
                    const location = res.headers.location;
                    res.destroy();
                    if (!location) {
                        finish(() => reject(new DownloadError("redirect_not_allowed")));
                    } else {
                        finish(() => resolve({ type: "redirect", location }));
                    }
                    return;
                }
                if (status < 200 || status >= 300) {
                    res.destroy();
                    finish(() => reject(new DownloadError("download_failed")));
                    return;
                }

                const chunks: Buffer[] = [];
                let total = 0;
                res.on("data", (chunk: Buffer) => {
                    total += chunk.length;
                    if (total > MAX_DOWNLOAD_BYTES) {
                        res.destroy();
                        finish(() => reject(new DownloadError("file_too_large")));
                        return;
                    }
                    chunks.push(chunk);
                });
                res.on("end", () =>
                    finish(() =>
                        resolve({
                            type: "final",
                            contentType: (res.headers["content-type"] as string | undefined) ?? null,
                            body: Buffer.concat(chunks),
                        }),
                    ),
                );
                res.on("error", (e) => finish(() => reject(e)));
            },
        );

        req.setTimeout(REQUEST_TIMEOUT_MS, () => {
            req.destroy(new DownloadError("request_timeout", 504));
        });
        req.on("error", (e) => finish(() => reject(e)));
        req.end();
    });
}

/**
 * SSRF-safe fetch of a user-supplied URL. Validates the URL and its resolved
 * IPs, pins each connection to its validated address, and re-validates every
 * redirect hop the same way (capped at {@link MAX_REDIRECTS}) rather than
 * either blindly following redirects or banning them outright — a redirect
 * chain can't bypass validation on any hop.
 * Throws {@link DownloadError} on any validation/transport failure.
 */
export async function fetchExternalFile(rawUrl: unknown): Promise<ExternalFile> {
    let parsed = parseAndCheckUrl(rawUrl);

    for (let hop = 0; ; hop++) {
        const target = await resolveAndValidate(parsed);
        const result = await fetchSingleHop(target);

        if (result.type === "final") {
            return { contentType: result.contentType, body: result.body };
        }

        if (hop >= MAX_REDIRECTS) {
            throw new DownloadError("too_many_redirects");
        }

        // Resolve a possibly-relative Location against the current URL, then
        // re-run the full protocol/hostname/DNS/IP validation on the target —
        // exactly as if it were the original user-supplied URL.
        let next: URL;
        try {
            next = new URL(result.location, parsed);
        } catch {
            throw new DownloadError("invalid_url");
        }
        parsed = parseAndCheckUrl(next.toString());
    }
}
