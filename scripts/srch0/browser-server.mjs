// Eigene externe Dienste für den echten SvelteKit-Browsertest. Leere Antworten
// machen den UI-Zustand sichtbar; Enginefilterung prüft die separate Matrix.
import { createServer } from 'node:http';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const webRequire = createRequire(new URL('../../web/package.json', import.meta.url));
const { createServer: createViteServer } = await import(webRequire.resolve('vite'));

const root = fileURLToPath(new URL('../../', import.meta.url));
process.chdir(`${root}web`);
const dataset = JSON.parse(readFileSync(new URL('../../testdata/trail-search/srch0/v1/datasets/reference.json', import.meta.url)));
const server = createServer(async (request, response) => {
    const path = new URL(request.url, 'http://localhost').pathname;
    let body = '';
    for await (const chunk of request) body += chunk;
    const input = body ? JSON.parse(body) : {};
    let result;
    if (path === '/search/token') result = { token: 'srch0-synthetic-browser-token' };
    else if (path === '/api/collections/users/auth-methods') result = { password: { enabled: true }, oauth2: { enabled: false, providers: [] } };
    else if (path === '/api/collections/categories/records' || path === '/api/collections/subcategories/records') {
        const items = path.includes('/subcategories/') ? dataset.subcategories : dataset.categories;
        result = { page: 1, perPage: 500, totalItems: items.length, totalPages: 1, items };
    } else if (path === '/api/collections/tags/records') {
        result = { page: 1, perPage: 500, totalItems: dataset.tags.length, totalPages: 1, items: dataset.tags };
    } else if (path === '/indexes/trails/search') {
        result = { hits: [], query: input.q ?? '', page: input.page ?? 1, hitsPerPage: input.hitsPerPage ?? 25, totalPages: 5, totalHits: 100 };
    } else if (path === '/multi-search') {
        result = { results: input.queries.map(query => ({ indexUid: query.indexUid, hits: [], estimatedTotalHits: 0 })) };
    } else {
        console.error(`Unexpected synthetic dependency request: ${request.method} ${path}`);
        response.writeHead(404, { 'Content-Type': 'application/json' });
        response.end(JSON.stringify({ message: 'Unknown synthetic endpoint' }));
        return;
    }
    response.writeHead(200, { 'Content-Type': 'application/json' });
    response.end(JSON.stringify(result));
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
let middleware;
const app = createServer((request, response) => {
    if (middleware) middleware(request, response);
    else { response.writeHead(503); response.end(); }
});
await new Promise(resolve => app.listen(0, '127.0.0.1', resolve));
const baseURL = `http://127.0.0.1:${app.address().port}`;
Object.assign(process.env, {
    MEILI_URL: `http://127.0.0.1:${server.address().port}`,
    PUBLIC_POCKETBASE_URL: `http://127.0.0.1:${server.address().port}`,
    ORIGIN: baseURL, PUBLIC_PRIVATE_INSTANCE: 'false', PUBLIC_IS_DEMO: 'false', TZ: 'Europe/Zurich',
});
// Der äussere HTTP-Server besitzt seinen zufällig vergebenen Port durchgehend.
// Es gibt kein Reservieren/Freigeben und keinen Zugriff auf bestehende Dienste.
const vite = await createViteServer({ root: `${root}web`, server: { middlewareMode: true, hmr: false } });
middleware = vite.middlewares;
let stopping = false;
const stop = async () => {
    if (stopping) return;
    stopping = true;
    await vite.close();
    app.close();
    app.closeAllConnections();
    server.close();
    server.closeAllConnections();
    if (process.connected) process.disconnect();
};
process.on('SIGTERM', stop);
process.on('SIGINT', stop);
process.on('message', message => { if (message?.stop) void stop(); });
process.on('disconnect', stop);
process.send?.({ baseURL });
