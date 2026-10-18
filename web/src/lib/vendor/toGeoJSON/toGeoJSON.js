function $(element, tagName) {
  return Array.from(element.getElementsByTagName(tagName));
}
function normalizeId(id) {
  return id[0] === "#" ? id : `#${id}`;
}
function $ns(element, tagName, ns) {
  return Array.from(element.getElementsByTagNameNS(ns, tagName));
}
/**
   * get the content of a text node, if any
   */
function nodeVal(node) {
  node === null || node === void 0 ? void 0 : node.normalize();
  return node && node.textContent || "";
}
/**
   * Get one Y child of X, if any, otherwise null
   */
function get1(node, tagName, callback) {
  const n = node.getElementsByTagName(tagName);
  const result = n.length ? n[0] : null;
  if (result && callback)
    callback(result);
  return result;
}
function get(node, tagName, callback) {
  const properties = {};
  if (!node)
    return properties;
  const n = node.getElementsByTagName(tagName);
  const result = n.length ? n[0] : null;
  if (result && callback) {
    return callback(result, properties);
  }
  return properties;
}
function val1(node, tagName, callback) {
  const val = nodeVal(get1(node, tagName));
  if (val && callback)
    return callback(val) || {};
  return {};
}
function $num(node, tagName, callback) {
  const val = parseFloat(nodeVal(get1(node, tagName)));
  if (isNaN(val))
    return undefined;
  if (val && callback)
    return callback(val) || {};
  return {};
}
function num1(node, tagName, callback) {
  const val = parseFloat(nodeVal(get1(node, tagName)));
  if (isNaN(val))
    return undefined;
  if (callback)
    callback(val);
  return val;
}
function getMulti(node, propertyNames) {
  const properties = {};
  for (const property of propertyNames) {
    val1(node, property, val => {
      properties[property] = val;
    });
  }
  return properties;
}
function isElement(node) {
  return (node === null || node === void 0 ? void 0 : node.nodeType) === 1;
}

function getLineStyle(node) {
  return get(node, "line", lineStyle => {
    const val = Object.assign({}, val1(lineStyle, "color", color => {
      return { stroke: `#${color}` };
    }), $num(lineStyle, "opacity", opacity => {
      return { "stroke-opacity": opacity };
    }), $num(lineStyle, "width", width => {
      // GPX width is in mm, convert to px with 96 px per inch
      return { "stroke-width": width * 96 / 25.4 };
    }));
    return val;
  });
}

function getExtensions(node) {
  let values = [];
  if (node === null)
    return values;
  for (const child of Array.from(node.childNodes)) {
    if (!isElement(child))
      continue;
    const name = abbreviateName(child.nodeName);
    if (name === "gpxtpx:TrackPointExtension") {
      // loop again for nested garmin extensions (eg. "gpxtpx:hr")
      values = values.concat(getExtensions(child));
    } else {
      // push custom extension (eg. "power")
      const val = nodeVal(child);
      values.push([name, parseNumeric(val)]);
    }
  }
  return values;
}
function abbreviateName(name) {
  return ["heart", "gpxtpx:hr", "hr"].includes(name) ? "heart" : name;
}
function parseNumeric(val) {
  const num = parseFloat(val);
  return isNaN(num) ? val : num;
}

function coordPair$1(node) {
  const ll = [
    parseFloat(node.getAttribute("lon") || ""),
    parseFloat(node.getAttribute("lat") || "")];

  if (isNaN(ll[0]) || isNaN(ll[1])) {
    return null;
  }
  num1(node, "ele", val => {
    ll.push(val);
  });
  const time = get1(node, "time");
  return {
    coordinates: ll,
    time: time ? nodeVal(time) : null,
    extendedValues: getExtensions(get1(node, "extensions"))
  };

}

function extractProperties(node) {
  const properties = getMulti(node, [
    "name",
    "cmt",
    "desc",
    "type",
    "time",
    "keywords"]);

  const extensions = Array.from(node.getElementsByTagNameNS("http://www.garmin.com/xmlschemas/GpxExtensions/v3", "*"));
  for (const child of extensions) {
    var _child$parentNode;
    if (((_child$parentNode = child.parentNode) === null || _child$parentNode === void 0 ? void 0 : _child$parentNode.parentNode) === node) {
      properties[child.tagName.replace(":", "_")] = nodeVal(child);
    }
  }
  const links = $(node, "link");
  if (links.length) {
    properties.links = links.map(link => Object.assign({ href: link.getAttribute("href") }, getMulti(link, ["text", "type"])));
  }
  return properties;
}

/**
   * Extract points from a trkseg or rte element.
   */
function getPoints$1(node, pointname) {
  const pts = $(node, pointname);
  const line = [];
  const times = [];
  const extendedValues = {};
  for (let i = 0; i < pts.length; i++) {
    const c = coordPair$1(pts[i]);
    if (!c) {
      continue;
    }
    line.push(c.coordinates);
    if (c.time)
      times.push(c.time);
    for (const [name, val] of c.extendedValues) {
      const plural = name === "heart" ? name : name.replace("gpxtpx:", "") + "s";
      if (!extendedValues[plural]) {
        extendedValues[plural] = Array(pts.length).fill(null);
      }
      extendedValues[plural][i] = val;
    }
  }
  if (line.length < 2)
    return; // Invalid line in GeoJSON
  return {
    line: line,
    times: times,
    extendedValues: extendedValues
  };

}
/**
   * Extract a LineString geometry from a rte
   * element.
   */
function getRoute(node) {
  const line = getPoints$1(node, "rtept");
  if (!line)
    return;
  return {
    type: "Feature",
    properties: Object.assign({ _gpxType: "rte" }, extractProperties(node), getLineStyle(get1(node, "extensions"))),
    geometry: {
      type: "LineString",
      coordinates: line.line
    }
  };


}
function getTrack(node) {
  const segments = $(node, "trkseg");
  const track = [];
  const times = [];
  const extractedLines = [];
  for (const segment of segments) {
    const line = getPoints$1(segment, "trkpt");
    if (line) {
      extractedLines.push(line);
      if (line.times && line.times.length)
        times.push(line.times);
    }
  }
  if (extractedLines.length === 0)
    return null;
  const multi = extractedLines.length > 1;
  const properties = Object.assign({ _gpxType: "trk" }, extractProperties(node), getLineStyle(get1(node, "extensions")), times.length ?
    {
      coordinateProperties: {
        times: multi ? times : times[0]
      }
    } :


    {});
  for (const line of extractedLines) {
    track.push(line.line);
    if (!properties.coordinateProperties) {
      properties.coordinateProperties = {};
    }
    const props = properties.coordinateProperties;
    const entries = Object.entries(line.extendedValues);
    for (let i = 0; i < entries.length; i++) {
      const [name, val] = entries[i];
      if (multi) {
        if (!props[name]) {
          props[name] = extractedLines.map(line => new Array(line.line.length).fill(null));
        }
        props[name][i] = val;
      } else {
        props[name] = val;
      }
    }
  }
  return {
    type: "Feature",
    properties: properties,
    geometry: multi ?
      {
        type: "MultiLineString",
        coordinates: track
      } :

      {
        type: "LineString",
        coordinates: track[0]
      }
  };


}
/**
   * Extract a point, if possible, from a given node,
   * which is usually a wpt or trkpt
   */
function getPoint(node) {
  const properties = Object.assign(extractProperties(node), getMulti(node, ["sym"]));
  const pair = coordPair$1(node);
  if (!pair)
    return null;
  return {
    type: "Feature",
    properties,
    geometry: {
      type: "Point",
      coordinates: pair.coordinates
    }
  };


}
/**
   * Convert GPX to GeoJSON incrementally, returning
   * a [Generator](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Iterators_and_Generators)
   * that yields output feature by feature.
   */
function* gpxGen(node) {
  for (const track of $(node, "trk")) {
    const feature = getTrack(track);
    if (feature)
      yield feature;
  }
  for (const route of $(node, "rte")) {
    const feature = getRoute(route);
    if (feature)
      yield feature;
  }
  for (const waypoint of $(node, "wpt")) {
    const point = getPoint(waypoint);
    if (point)
      yield point;
  }
}
/**
   *
   * Convert a GPX document to GeoJSON. The first argument, `doc`, must be a GPX
   * document as an XML DOM - not as a string. You can get this using jQuery's default
   * `.ajax` function or using a bare XMLHttpRequest with the `.response` property
   * holding an XML DOM.
   *
   * The output is a JavaScript object of GeoJSON data, same as `.kml` outputs, with the
   * addition of a `_gpxType` property on each `LineString` feature that indicates whether
   * the feature was encoded as a route (`rte`) or track (`trk`) in the GPX document.
   */
function gpx(node) {
  return {
    type: "FeatureCollection",
    features: Array.from(gpxGen(node))
  };

}

const EXTENSIONS_NS = "http://www.garmin.com/xmlschemas/ActivityExtension/v2";
const TRACKPOINT_ATTRIBUTES = [
  ["heartRate", "heartRates"],
  ["Cadence", "cadences"],
  // Extended Trackpoint attributes
  ["Speed", "speeds"],
  ["Watts", "watts"]];

const LAP_ATTRIBUTES = [
  ["TotalTimeSeconds", "totalTimeSeconds"],
  ["DistanceMeters", "distanceMeters"],
  ["MaximumSpeed", "maxSpeed"],
  ["AverageHeartRateBpm", "avgHeartRate"],
  ["MaximumHeartRateBpm", "maxHeartRate"],
  // Extended Lap attributes
  ["AvgSpeed", "avgSpeed"],
  ["AvgWatts", "avgWatts"],
  ["MaxWatts", "maxWatts"]];

function getProperties(node, attributeNames) {
  const properties = [];
  for (const [tag, alias] of attributeNames) {
    let elem = get1(node, tag);
    if (!elem) {
      const elements = node.getElementsByTagNameNS(EXTENSIONS_NS, tag);
      if (elements.length) {
        elem = elements[0];
      }
    }
    const val = parseFloat(nodeVal(elem));
    if (!isNaN(val)) {
      properties.push([alias, val]);
    }
  }
  return properties;
}
function coordPair(node) {
  const ll = [num1(node, "LongitudeDegrees"), num1(node, "LatitudeDegrees")];
  if (ll[0] === undefined ||
    isNaN(ll[0]) ||
    ll[1] === undefined ||
    isNaN(ll[1])) {
    return null;
  }
  const heartRate = get1(node, "HeartRateBpm");
  const time = nodeVal(get1(node, "Time"));
  get1(node, "AltitudeMeters", alt => {
    const a = parseFloat(nodeVal(alt));
    if (!isNaN(a)) {
      ll.push(a);
    } else {
      ll.push(null)
    }
    ll.push(time);
  });
  return {
    coordinates: ll,
    time: time || null,
    heartRate: heartRate ? parseFloat(nodeVal(heartRate)) : null,
    extensions: getProperties(node, TRACKPOINT_ATTRIBUTES)
  };

}
function getPoints(node) {
  const pts = $(node, "Trackpoint");
  const line = [];
  const times = [];
  const heartRates = [];
  if (pts.length < 2)
    return null; // Invalid line in GeoJSON
  const extendedProperties = {};
  const result = { extendedProperties };
  for (let i = 0; i < pts.length; i++) {
    const c = coordPair(pts[i]);
    if (c === null)
      continue;
    line.push(c.coordinates);
    const { time, heartRate, extensions } = c;
    if (time)
      times.push(time);
    if (heartRate)
      heartRates.push(heartRate);
    for (const [alias, value] of extensions) {
      if (!extendedProperties[alias]) {
        extendedProperties[alias] = Array(pts.length).fill(null);
      }
      extendedProperties[alias][i] = value;
    }
  }
  if (line.length < 2)
    return null;
  return Object.assign(result, {
    line: line,
    times: times,
    heartRates: heartRates
  });

}
function getLap(node) {
  const segments = $(node, "Track");
  const track = [];
  const times = [];
  const heartRates = [];
  const allExtendedProperties = [];
  let line;
  const properties = Object.assign(Object.fromEntries(getProperties(node, LAP_ATTRIBUTES)), get(node, "Name", nameElement => {
    return { name: nodeVal(nameElement) };
  }));
  for (const segment of segments) {
    line = getPoints(segment);
    if (line) {
      track.push(line.line);
      if (line.times.length)
        times.push(line.times);
      if (line.heartRates.length)
        heartRates.push(line.heartRates);
      allExtendedProperties.push(line.extendedProperties);
    }
  }
  for (let i = 0; i < allExtendedProperties.length; i++) {
    const extendedProperties = allExtendedProperties[i];
    for (const property in extendedProperties) {
      if (segments.length === 1) {
        if (line) {
          properties[property] = line.extendedProperties[property];
        }
      } else {
        if (!properties[property]) {
          properties[property] = track.map(track => Array(track.length).fill(null));
        }
        properties[property][i] = extendedProperties[property];
      }
    }
  }
  if (track.length === 0)
    return null;
  if (times.length || heartRates.length) {
    properties.coordinateProperties = Object.assign(times.length ?
      {
        times: track.length === 1 ? times[0] : times
      } :

      {}, heartRates.length ?
      {
        heart: track.length === 1 ? heartRates[0] : heartRates
      } :

      {});
  }
  return {
    type: "Feature",
    properties: properties,
    geometry: track.length === 1 ?
      {
        type: "LineString",
        coordinates: track[0]
      } :

      {
        type: "MultiLineString",
        coordinates: track
      }
  };


}
/**
   * Incrementally convert a TCX document to GeoJSON. The
   * first argument, `doc`, must be a TCX
   * document as an XML DOM - not as a string.
   */
function* tcxGen(node) {
  for (const lap of $(node, "Lap")) {
    const feature = getLap(lap);
    if (feature)
      yield feature;
  }
  for (const course of $(node, "Courses")) {
    const feature = getLap(course);
    if (feature)
      yield feature;
  }
}
/**
   * Convert a TCX document to GeoJSON. The first argument, `doc`, must be a TCX
   * document as an XML DOM - not as a string.
   */
function tcx(node) {
  return {
    type: "FeatureCollection",
    features: Array.from(tcxGen(node))
  };

}

function fixColor(v, prefix) {
  const properties = {};
  const colorProp = prefix == "stroke" || prefix === "fill" ? prefix : prefix + "-color";
  if (v[0] === "#") {
    v = v.substring(1);
  }
  if (v.length === 6 || v.length === 3) {
    properties[colorProp] = "#" + v;
  } else
    if (v.length === 8) {
      properties[prefix + "-opacity"] = parseInt(v.substring(0, 2), 16) / 255;
      properties[colorProp] =
        "#" + v.substring(6, 8) + v.substring(4, 6) + v.substring(2, 4);
    }
  return properties;
}

function numericProperty(node, source, target) {
  const properties = {};
  num1(node, source, val => {
    properties[target] = val;
  });
  return properties;
}
function getColor(node, output) {
  return get(node, "color", elem => fixColor(nodeVal(elem), output));
}
function extractIconHref(node) {
  return get(node, "Icon", (icon, properties) => {
    val1(icon, "href", href => {
      properties.icon = href;
    });
    return properties;
  });
}
function extractIcon(node) {
  return get(node, "IconStyle", iconStyle => {
    return Object.assign(getColor(iconStyle, "icon"), numericProperty(iconStyle, "scale", "icon-scale"), numericProperty(iconStyle, "heading", "icon-heading"), get(iconStyle, "hotSpot", hotspot => {
      const left = parseFloat(hotspot.getAttribute("x") || "");
      const top = parseFloat(hotspot.getAttribute("y") || "");
      const xunits = hotspot.getAttribute("xunits") || "";
      const yunits = hotspot.getAttribute("yunits") || "";
      if (!isNaN(left) && !isNaN(top))
        return {
          "icon-offset": [left, top],
          "icon-offset-units": [xunits, yunits]
        };

      return {};
    }), extractIconHref(iconStyle));
  });
}
function extractLabel(node) {
  return get(node, "LabelStyle", labelStyle => {
    return Object.assign(getColor(labelStyle, "label"), numericProperty(labelStyle, "scale", "label-scale"));
  });
}
function extractLine(node) {
  return get(node, "LineStyle", lineStyle => {
    return Object.assign(getColor(lineStyle, "stroke"), numericProperty(lineStyle, "width", "stroke-width"));
  });
}
function extractPoly(node) {
  return get(node, "PolyStyle", (polyStyle, properties) => {
    return Object.assign(properties, get(polyStyle, "color", elem => fixColor(nodeVal(elem), "fill")), val1(polyStyle, "fill", fill => {
      if (fill === "0")
        return { "fill-opacity": 0 };
    }), val1(polyStyle, "outline", outline => {
      if (outline === "0")
        return { "stroke-opacity": 0 };
    }));
  });
}
function extractStyle(node) {
  return Object.assign({}, extractPoly(node), extractLine(node), extractLabel(node), extractIcon(node));
}

const toNumber = x => Number(x);
const typeConverters = {
  string: x => x,
  int: toNumber,
  uint: toNumber,
  short: toNumber,
  ushort: toNumber,
  float: toNumber,
  double: toNumber,
  bool: x => Boolean(x)
};

function extractExtendedData(node, schema) {
  return get(node, "ExtendedData", (extendedData, properties) => {
    for (const data of $(extendedData, "Data")) {
      properties[data.getAttribute("name") || ""] = nodeVal(get1(data, "value"));
    }
    for (const simpleData of $(extendedData, "SimpleData")) {
      const name = simpleData.getAttribute("name") || "";
      const typeConverter = schema[name] || typeConverters.string;
      properties[name] = typeConverter(nodeVal(simpleData));
    }
    return properties;
  });
}
function getMaybeHTMLDescription(node) {
  const descriptionNode = get1(node, "description");
  for (const c of Array.from((descriptionNode === null || descriptionNode === void 0 ? void 0 : descriptionNode.childNodes) || [])) {
    if (c.nodeType === 4) {
      return {
        description: {
          "@type": "html",
          value: nodeVal(c)
        }
      };


    }
  }
  return {};
}
function extractTimeSpan(node) {
  return get(node, "TimeSpan", timeSpan => {
    return {
      timespan: {
        begin: nodeVal(get1(timeSpan, "begin")),
        end: nodeVal(get1(timeSpan, "end"))
      }
    };


  });
}
function extractTimeStamp(node) {
  return get(node, "TimeStamp", timeStamp => {
    return { timestamp: nodeVal(get1(timeStamp, "when")) };
  });
}
function extractCascadedStyle(node, styleMap) {
  return val1(node, "styleUrl", styleUrl => {
    styleUrl = normalizeId(styleUrl);
    if (styleMap[styleUrl]) {
      return Object.assign({ styleUrl }, styleMap[styleUrl]);
    }
    // For backward-compatibility. Should we still include
    // styleUrl even if it's not resolved?
    return { styleUrl };
  });
}

const removeSpace = /\s*/g;
const splitSpace = /\s+/;
/**
 * Get one coordinate from a coordinate array, if any
 */
function coord1(value) {
  return value.
    replace(removeSpace, "").
    split(",").
    map(parseFloat).
    filter(num => !isNaN(num)).
    slice(0, 3);
}
/**
   * Get all coordinates from a coordinate array as [[],[]]
   */
function coord(value) {
  // trim() instead of the original /^\s*|\s*$/g replace, which takes
  // quadratic time on long runs of whitespace (a ~100 KB KML stalled the
  // server for seconds).
  return value.
    trim().
    split(splitSpace).
    map(coord1).
    filter(coord => {
      return coord.length >= 2;
    });
}
function gxCoords(node) {
  let elems = $(node, "coord");
  if (elems.length === 0) {
    elems = $ns(node, "coord", "*");
  }
  let coordinates = elems.map(elem => {
    return nodeVal(elem).split(" ").map(parseFloat);
  });
  if (coordinates.length === 0) {
    return null;
  }

  let times = $(node, "when").map(elem => nodeVal(elem));

  const invalidCoordinateIndices = []
  coordinates.forEach((c, i) => {
    if (c.length < 2) {
      invalidCoordinateIndices.push(i)
    } else {
      c.push(times[i])
    }
  })
  coordinates = coordinates.filter((_, i) => !invalidCoordinateIndices.includes(i))

  return {
    geometry: coordinates.length > 2 ?
      {
        type: "LineString",
        coordinates
      } :

      {
        type: "Point",
        coordinates: coordinates[0]
      },

    times: times
  };

}
function fixRing(ring) {
  if (ring.length === 0)
    return ring;
  const first = ring[0];
  const last = ring[ring.length - 1];
  let equal = true;
  for (let i = 0; i < Math.max(first.length, last.length); i++) {
    if (first[i] !== last[i]) {
      equal = false;
      break;
    }
  }
  if (!equal) {
    return ring.concat([ring[0]]);
  }
  return ring;
}
function getCoordinates(node) {
  return nodeVal(get1(node, "coordinates"));
}
function getGeometry(node) {
  let geometries = [];
  let coordTimes = [];
  for (let i = 0; i < node.childNodes.length; i++) {
    const child = node.childNodes.item(i);
    if (isElement(child)) {
      switch (child.tagName) {
        case "MultiGeometry":
        case "MultiTrack":
        case "gx:MultiTrack": {
          const childGeometries = getGeometry(child);
          geometries = geometries.concat(childGeometries.geometries);
          coordTimes = coordTimes.concat(childGeometries.coordTimes);
          break;
        }
        case "Point": {
          const coordinates = coord1(getCoordinates(child));
          if (coordinates.length >= 2) {
            geometries.push({
              type: "Point",
              coordinates
            });

          }
          break;
        }
        case "LinearRing":
        case "LineString": {
          const coordinates = coord(getCoordinates(child));
          if (coordinates.length >= 2) {
            geometries.push({
              type: "LineString",
              coordinates
            });

          }
          break;
        }
        case "Polygon": {
          const coords = [];
          for (const linearRing of $(child, "LinearRing")) {
            const ring = fixRing(coord(getCoordinates(linearRing)));
            if (ring.length >= 4) {
              coords.push(ring);
            }
          }
          if (coords.length) {
            geometries.push({
              type: "Polygon",
              coordinates: coords
            });

          }
          break;
        }
        case "Track":
        case "gx:Track": {
          const gx = gxCoords(child);
          if (!gx)
            break;
          const { times, geometry } = gx;
          geometries.push(geometry);
          if (times.length)
            coordTimes.push(times);
          break;
        }
      }

    }
  }
  return {
    geometries,
    coordTimes
  };

}

function geometryListToGeometry(geometries) {
  return geometries.length === 0 ?
    null :
    geometries.length === 1 ?
      geometries[0] :
      {
        type: "GeometryCollection",
        geometries
      };

}
function getPlacemark(node, styleMap, schema, options) {
  var _feature$properties;
  const { coordTimes, geometries } = getGeometry(node);
  const geometry = geometryListToGeometry(geometries);
  if (!geometry && options.skipNullGeometry) {
    return null;
  }
  const feature = {
    type: "Feature",
    geometry,
    properties: Object.assign(getMulti(node, [
      "name",
      "address",
      "visibility",
      "open",
      "phoneNumber",
      "description"]),
      getMaybeHTMLDescription(node),
      extractCascadedStyle(node, styleMap),
      extractStyle(node),
      extractExtendedData(node, schema),
      extractTimeSpan(node),
      extractTimeStamp(node),
      coordTimes.length ?
        {
          coordinateProperties: {
            times: coordTimes.length === 1 ? coordTimes[0] : coordTimes
          }
        } : {})
  };

  if (((_feature$properties = feature.properties) === null || _feature$properties === void 0 ? void 0 : _feature$properties.visibility) !== undefined) {
    feature.properties.visibility = feature.properties.visibility !== "0";
  }
  const id = node.getAttribute("id");
  if (id !== null && id !== "")
    feature.id = id;
  return feature;
}

function getGroundOverlayBox(node) {
  const latLonQuad = get1(node, "gx:LatLonQuad");
  if (latLonQuad) {
    const ring = fixRing(coord(getCoordinates(node)));
    return {
      geometry: {
        type: "Polygon",
        coordinates: [ring]
      }
    };


  }
  return getLatLonBox(node);
}
const DEGREES_TO_RADIANS = Math.PI / 180;
function rotateBox(bbox, coordinates, rotation) {
  const center = [(bbox[0] + bbox[2]) / 2, (bbox[1] + bbox[3]) / 2];
  return [
    coordinates[0].map(coordinate => {
      const dy = coordinate[1] - center[1];
      const dx = coordinate[0] - center[0];
      const distance = Math.sqrt(Math.pow(dy, 2) + Math.pow(dx, 2));
      const angle = Math.atan2(dy, dx) + rotation * DEGREES_TO_RADIANS;
      return [
        center[0] + Math.cos(angle) * distance,
        center[1] + Math.sin(angle) * distance];

    })];

}
function getLatLonBox(node) {
  const latLonBox = get1(node, "LatLonBox");
  if (latLonBox) {
    const north = num1(latLonBox, "north");
    const west = num1(latLonBox, "west");
    const east = num1(latLonBox, "east");
    const south = num1(latLonBox, "south");
    const rotation = num1(latLonBox, "rotation");
    if (typeof north === "number" &&
      typeof south === "number" &&
      typeof west === "number" &&
      typeof east === "number") {
      const bbox = [west, south, east, north];
      let coordinates = [
        [
          [west, north],
          [east, north],
          [east, south],
          [west, south],
          [west, north]]];


      if (typeof rotation === "number") {
        coordinates = rotateBox(bbox, coordinates, rotation);
      }
      return {
        bbox,
        geometry: {
          type: "Polygon",
          coordinates
        }
      };


    }
  }
  return null;
}
function getGroundOverlay(node, styleMap, schema, options) {
  var _feature$properties2;
  const box = getGroundOverlayBox(node);
  const geometry = (box === null || box === void 0 ? void 0 : box.geometry) || null;
  if (!geometry && options.skipNullGeometry) {
    return null;
  }
  const feature = {
    type: "Feature",
    geometry,
    properties: Object.assign(
      /**
                                  * Related to
                                  * https://gist.github.com/tmcw/037a1cb6660d74a392e9da7446540f46
                                  */
      { "@geometry-type": "groundoverlay" }, getMulti(node, [
        "name",
        "address",
        "visibility",
        "open",
        "phoneNumber",
        "description"]),
      getMaybeHTMLDescription(node), extractCascadedStyle(node, styleMap), extractStyle(node), extractIconHref(node), extractExtendedData(node, schema), extractTimeSpan(node), extractTimeStamp(node))
  };

  if (box !== null && box !== void 0 && box.bbox) {
    feature.bbox = box.bbox;
  }
  if (((_feature$properties2 = feature.properties) === null || _feature$properties2 === void 0 ? void 0 : _feature$properties2.visibility) !== undefined) {
    feature.properties.visibility = feature.properties.visibility !== "0";
  }
  const id = node.getAttribute("id");
  if (id !== null && id !== "")
    feature.id = id;
  return feature;
}

function getStyleId(style) {
  let id = style.getAttribute("id");
  const parentNode = style.parentNode;
  if (!id &&
    isElement(parentNode) &&
    parentNode.localName === "CascadingStyle") {
    id = parentNode.getAttribute("kml:id") || parentNode.getAttribute("id");
  }
  return normalizeId(id || "");
}
function buildStyleMap(node) {
  const styleMap = {};
  for (const style of $(node, "Style")) {
    styleMap[getStyleId(style)] = extractStyle(style);
  }
  for (const map of $(node, "StyleMap")) {
    const id = normalizeId(map.getAttribute("id") || "");
    val1(map, "styleUrl", styleUrl => {
      styleUrl = normalizeId(styleUrl);
      if (styleMap[styleUrl]) {
        styleMap[id] = styleMap[styleUrl];
      }
    });
  }
  return styleMap;
}
function buildSchema(node) {
  const schema = {};
  for (const field of $(node, "SimpleField")) {
    schema[field.getAttribute("name") || ""] =
      typeConverters[field.getAttribute("type") || ""] ||
      typeConverters["string"];
  }
  return schema;
}
const FOLDER_PROPS = [
  "name",
  "visibility",
  "open",
  "address",
  "description",
  "phoneNumber",
  "visibility"];

function getFolder(node) {
  const meta = {};
  for (const child of Array.from(node.childNodes)) {
    if (isElement(child) && FOLDER_PROPS.includes(child.tagName)) {
      meta[child.tagName] = nodeVal(child);
    }
  }
  return {
    type: "folder",
    meta,
    children: []
  };

}
/**
   * Yield a nested tree with KML folder structure
   *
   * This generates a tree with the given structure:
   *
   * ```js
   * {
   *   "type": "root",
   *   "children": [
   *     {
   *       "type": "folder",
   *       "meta": {
   *         "name": "Test"
   *       },
   *       "children": [
   *          // ...features and folders
   *       ]
   *     }
   *     // ...features
   *   ]
   * }
   * ```
   *
   * ### GroundOverlay
   *
   * GroundOverlay elements are converted into
   * `Feature` objects with `Polygon` geometries,
   * a property like:
   *
   * ```json
   * {
   *   "@geometry-type": "groundoverlay"
   * }
   * ```
   *
   * And the ground overlay's image URL in the `href`
   * property. Ground overlays will need to be displayed
   * with a separate method to other features, depending
   * on which map framework you're using.
   */
function kmlWithFolders(node, options = {
  skipNullGeometry: false
}) {
  const styleMap = buildStyleMap(node);
  const schema = buildSchema(node);
  const tree = { type: "root", children: [] };
  function traverse(node, pointer, options) {
    if (isElement(node)) {
      switch (node.tagName) {
        case "GroundOverlay": {
          const placemark = getGroundOverlay(node, styleMap, schema, options);
          if (placemark) {
            pointer.children.push(placemark);
          }
          break;
        }
        case "Placemark": {
          const placemark = getPlacemark(node, styleMap, schema, options);
          if (placemark) {
            pointer.children.push(placemark);
          }
          break;
        }
        case "Folder": {
          const folder = getFolder(node);
          pointer.children.push(folder);
          pointer = folder;
          break;
        }
      }

    }
    if (node.childNodes) {
      for (let i = 0; i < node.childNodes.length; i++) {
        traverse(node.childNodes[i], pointer, options);
      }
    }
  }
  traverse(node, tree, options);
  return tree;
}
/**
   * Convert KML to GeoJSON incrementally, returning
   * a [Generator](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Iterators_and_Generators)
   * that yields output feature by feature.
   */
function* kmlGen(node, options = {
  skipNullGeometry: false
}) {
  const styleMap = buildStyleMap(node);
  const schema = buildSchema(node);
  for (const placemark of $(node, "Placemark")) {
    const feature = getPlacemark(placemark, styleMap, schema, options);
    if (feature)
      yield feature;
  }
  for (const groundOverlay of $(node, "GroundOverlay")) {
    const feature = getGroundOverlay(groundOverlay, styleMap, schema, options);
    if (feature)
      yield feature;
  }
}
/**
   * Convert a KML document to GeoJSON. The first argument, `doc`, must be a KML
   * document as an XML DOM - not as a string. You can get this using jQuery's default
   * `.ajax` function or using a bare XMLHttpRequest with the `.response` property
   * holding an XML DOM.
   *
   * The output is a JavaScript object of GeoJSON data. You can convert it to a string
   * with [JSON.stringify](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/stringify)
   * or use it directly in libraries.
   */
function kml(node, options = {
  skipNullGeometry: false
}) {
  return {
    type: "FeatureCollection",
    features: Array.from(kmlGen(node, options))
  };

}

export { gpx, gpxGen, kml, kmlGen, kmlWithFolders, tcx, tcxGen };