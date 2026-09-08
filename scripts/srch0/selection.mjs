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
    'SRCH0-SEARCH-116': 'Anonyme Akteursuche erhält den Engine-Status 403',
    'SRCH0-SEARCH-120': 'Multi-Search mit expliziten IDs hält die Tenant-Grenze',
    'SRCH0-SEARCH-121': 'Multi-Search liefert Trails und Listen getrennt',
    'SRCH0-SEARCH-124': 'Unbekannter Index erhält den Engine-Status 403',
    'SRCH0-SEARCH-125': 'Ungültiger Filter erhält den Engine-Status 400',
    'SRCH0-SEARCH-128': 'Text und Feldauswahl werden an das SDK weitergegeben',
});

export const API_COMPLETENESS_CASES = Object.freeze({
    'SRCH0-SEARCH-117': 'Upload prüft auch Duplikatkandidaten nach der ersten Engineantwort',
    'SRCH0-SEARCH-129': 'Cluster bildet den vollständigen sichtbaren Bestand auch oberhalb maxTotalHits ab',
});

export const API_STARTUP_CASES = Object.freeze({
    'SRCH0-MUTATION-090': 'Bestehende Indizes bleiben nach abgeschlossenem Startup erhalten',
    'SRCH0-MUTATION-091': 'Fehlende Indizes sind vor Suchbereitschaft vollständig aufgebaut',
});

export const PROJECTION_ENGINE_CASE_IDS = Object.freeze(['SRCH0-PROJECTION-008', 'SRCH0-PROJECTION-010']);

export const API_CASE_IDS = Object.freeze([...Object.keys(API_SEARCH_CASES), ...Object.keys(API_COMPLETENESS_CASES), ...Object.keys(API_STARTUP_CASES)]);
export const isAPIIntegrationCase = fixture => API_CASE_IDS.includes(fixture.case_id);
