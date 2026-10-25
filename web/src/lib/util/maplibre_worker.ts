import { setWorkerUrl } from "maplibre-gl";
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";

// MapLibre 6 ships its worker as a separate ES module and resolves it relative to
// import.meta.url. Under Vite that URL points at the pre-bundled (dev) or chunked
// (build) copy of maplibre-gl, next to which no worker file exists, so vector
// styles never render. Hand MapLibre the URL of the worker Vite serves instead.
setWorkerUrl(workerUrl);
