import { Parser } from 'isomorphic-xml2js';

type BrowserXmlParser = Parser & {
  domToObject(node: Node): any;
};

function isGpxNamespace(namespace: string | null | undefined): boolean {
  return namespace === 'http://www.topografix.com/GPX/1/0' ||
    namespace === 'http://www.topografix.com/GPX/1/1';
}

function parseAttribute(value: string): string | number {
  if (value.length && !isNaN(Number(value))) {
    return Number.isInteger(Number(value)) ? parseInt(value, 10) : parseFloat(value);
  }
  return value;
}

function normalizeElement(element: Element): Element {
  if (element.getAttribute('xmlns') === '') {
    element.removeAttribute('xmlns');
  }
  if (element.prefix && isGpxNamespace(element.namespaceURI)) {
    const replacement = element.ownerDocument.createElementNS(element.namespaceURI, element.localName);
    while (element.attributes.length) {
      const attribute = element.attributes.item(0)!;
      element.removeAttributeNode(attribute);
      replacement.setAttributeNodeNS(attribute);
    }
    while (element.firstChild) {
      replacement.appendChild(element.firstChild);
    }
    element.parentNode!.replaceChild(replacement, element);
    element = replacement;
  }
  for (let child = element.firstChild; child;) {
    const next = child.nextSibling;
    if (child.nodeType === 8) {
      element.removeChild(child);
    } else if (child.nodeType === 1) {
      normalizeElement(child as Element);
    }
    child = next;
  }
  return element;
}

/** Uses each platform's existing XML parser once, preserving its import behavior. */
export function parseGpxXml(source: string): any {
  if (source.charCodeAt(0) === 0xFEFF) {
    source = source.slice(1);
  }

  // The browser export provides a public DOM converter. Calling it directly
  // avoids parseString's first-child assumption for comments before the root.
  if ('domToObject' in Parser.prototype) {
    const parser = new Parser({
      explicitArray: false,
      attrValueProcessors: [parseAttribute]
    }) as BrowserXmlParser;
    const document = new DOMParser().parseFromString(source, 'application/xml');
    for (const namespace of [
      'http://www.w3.org/1999/xhtml',
      'http://www.mozilla.org/newlayout/xml/parsererror.xml'
    ]) {
      const error = document.getElementsByTagNameNS(namespace, 'parsererror').item(0);
      if (error) {
        throw new Error(error.textContent || 'Invalid XML');
      }
    }
    if (!document.documentElement) {
      throw new Error('Missing XML root element');
    }
    // Normalize before conversion groups equal names, preserving the order of
    // interleaved prefixed/unprefixed GPX elements. Literal text stays untouched.
    const root = normalizeElement(document.documentElement);
    return { [root.nodeName]: parser.domToObject(root) };
  }

  // xml2js processes attributes before the tag name, and calls the validator
  // on closing each element. Keep namespace bindings scoped to that element.
  const emptyScope: Record<string, string> = Object.create(null);
  const scopes: Record<string, string>[] = [];
  let pendingScope: Record<string, string> | undefined;
  const parser = new Parser({
    explicitArray: false,
    attrValueProcessors: [(value: string, name: string) => {
      if (name.startsWith('xmlns:')) {
        pendingScope ??= Object.create(scopes[scopes.length - 1] ?? emptyScope);
        pendingScope![name.slice('xmlns:'.length)] = value;
      }
      return parseAttribute(value);
    }],
    tagNameProcessors: [(name: string) => {
      const scope = pendingScope ?? scopes[scopes.length - 1] ?? emptyScope;
      scopes.push(scope);
      pendingScope = undefined;
      const separator = name.indexOf(':');
      return separator > 0 && isGpxNamespace(scope[name.slice(0, separator)])
        ? name.slice(separator + 1)
        : name;
    }],
    validator: (_path: string, _current: unknown, value: any) => {
      scopes.pop();
      if (value?.$?.xmlns === '') {
        delete value.$.xmlns;
        if (Object.keys(value.$).length === 0) {
          delete value.$;
          const keys = Object.keys(value);
          if (keys.length === 0) {
            return '';
          }
          if (keys.length === 1 && keys[0] === '_') {
            return value._;
          }
        }
      }
      return value;
    }
  });
  let result: any;
  let error: Error | null = null;
  parser.parseString(source, (parseError, parsed) => {
    error = parseError;
    result = parsed;
  });
  if (error) {
    throw error;
  }
  return result;
}
