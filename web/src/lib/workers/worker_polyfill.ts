import { DOMParser as XMLDOMParser } from '@xmldom/xmldom';

/**
 * Web Worker Environment Polyfill for GPX Parsing.
 *
 * Background context & rationale:
 * 1. isomorphic-xml2js compatibility:
 *    When isomorphic-xml2js is imported, its internal `builder.js` immediately evaluates
 *    top-level code expecting `document.implementation.createDocument()` and `new XMLSerializer()`.
 *    In Web Workers, `document` and `XMLSerializer` are not in the global scope. We stub them here
 *    so isomorphic-xml2js can be safely imported without ReferenceErrors. Note that the worker only
 *    parses GPX (reading XML), it never calls builder/serializer methods.
 *
 * 2. Cross-browser DOMParser support:
 *    Some browser engines (e.g. WebKit/Safari or specific worker execution contexts) do not expose
 *    the standard `DOMParser` in `DedicatedWorkerGlobalScope`. We fall back to `@xmldom/xmldom`
 *    (already a project dependency).
 *
 * 3. Silent error handling:
 *    `isomorphic-xml2js` intentionally parses an invalid string ('INVALID') on startup to detect
 *    browser parsererror namespaces. We pass `onError: () => {}` to xmldom to silence internal
 *    fatalError console noise caused by this probe. Real GPX syntax errors are still explicitly
 *    checked, caught, and handled downstream in `parse-xml.ts` and `gpx_parser.worker.ts`.
 */

class SilentDOMParser extends XMLDOMParser {
    constructor(options?: any) {
        super({
            onError: () => {},
            ...options
        });
    }
}

if (typeof self !== 'undefined') {
    if (typeof (self as any).DOMParser === 'undefined') {
        (self as any).DOMParser = SilentDOMParser;
    }

    if (typeof (self as any).document === 'undefined') {
        try {
            const ParserClass = (self as any).DOMParser || SilentDOMParser;
            const doc = new ParserClass().parseFromString('<xml/>', 'application/xml');
            (self as any).document = doc;
        } catch {
            (self as any).document = {
                implementation: {
                    createDocument: () => ({
                        createAttribute: (name: string) => ({ name, value: '' }),
                        createElement: () => ({
                            textContent: '',
                            attributes: { setNamedItem: () => {} },
                            appendChild: () => {}
                        })
                    })
                },
                createTextNode: (text: string) => ({ textContent: text })
            };
        }
    }

    if (typeof (self as any).XMLSerializer === 'undefined') {
        (self as any).XMLSerializer = class XMLSerializer {
            serializeToString(): string {
                return '';
            }
        };
    }

    if (typeof (self as any).Node === 'undefined') {
        (self as any).Node = {
            ELEMENT_NODE: 1,
            ATTRIBUTE_NODE: 2,
            TEXT_NODE: 3,
            CDATA_SECTION_NODE: 4,
            ENTITY_REFERENCE_NODE: 5,
            ENTITY_NODE: 6,
            PROCESSING_INSTRUCTION_NODE: 7,
            COMMENT_NODE: 8,
            DOCUMENT_NODE: 9,
            DOCUMENT_TYPE_NODE: 10,
            DOCUMENT_FRAGMENT_NODE: 11,
            NOTATION_NODE: 12
        };
    }
}

export {};
