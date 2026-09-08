// Die breite Suchmatrix läuft direkt gegen beide Engines. Die API-Auswahl
// ergänzt nur Übergänge, die Route, SDK oder Anmeldekontext verändern können.
export const API_SEARCH_CASES = Object.freeze({
    'SRCH0-SEARCH-001': 'Anonyme Standardsuche mit Seiten- und Sortieroptionen',
    'SRCH0-SEARCH-012': 'Anonyme Suche darf keinen privaten Trail liefern',
    'SRCH0-SEARCH-022': 'Alice kann ihren eigenen privaten Trail suchen',
    'SRCH0-SEARCH-024': 'Alice kann einen mit ihr geteilten Trail suchen',
    'SRCH0-SEARCH-033': 'Bob verwendet seinen eigenen Tenant',
    'SRCH0-SEARCH-048': 'Die Route setzt ausgeblendete Kategorien ein',
    'SRCH0-SEARCH-049': 'Die Route setzt ausgeblendete Unterkategorien ein',
    'SRCH0-SEARCH-050': 'Explizite IDs umgehen Kategoriepräferenzen',
    'SRCH0-SEARCH-110': 'Angemeldete Listensuche berücksichtigt Freigaben',
    'SRCH0-SEARCH-111': 'Anonyme Listensuche bleibt öffentlich',
    'SRCH0-SEARCH-112': 'Angemeldete Akteursuche verwendet Text und Limit',
    'SRCH0-SEARCH-116': 'Anonyme Akteursuche: Engine 403 wird API 500',
    'SRCH0-SEARCH-117': 'Fehlende Seitenoptionen behalten Offset 0 und Limit 20',
    'SRCH0-SEARCH-120': 'Multi-Search mit expliziten IDs hält die Tenant-Grenze',
    'SRCH0-SEARCH-121': 'Multi-Search liefert Trails und Listen getrennt',
    'SRCH0-SEARCH-124': 'Unbekannter Index: Engine 403 wird API 500',
    'SRCH0-SEARCH-125': 'Ungültiger Filter: Engine 400 wird API 500',
    'SRCH0-SEARCH-128': 'Text und Feldauswahl werden an das SDK weitergegeben',
});

export const API_STARTUP_CASES = Object.freeze({
    'SRCH0-MUTATION-090': 'Bestehende Indizes während Leerung und Teilaufbau',
    'SRCH0-MUTATION-091': 'Fehlende Indizes während Erstellung und Teilaufbau',
});

export const API_CASE_IDS = Object.freeze([...Object.keys(API_SEARCH_CASES), ...Object.keys(API_STARTUP_CASES)]);
export const isAPIIntegrationCase = fixture => API_CASE_IDS.includes(fixture.case_id);
