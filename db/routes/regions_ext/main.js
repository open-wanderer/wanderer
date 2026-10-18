app.store.headerLinks.push({
  href: "/region-catalog/",
  icon: "ri-map-2-line",
  label: "Region Catalog",
});

// PocketBase's admin UI hardcodes target="_blank" + rel="noopener noreferrer"
// for any headerLinks entry whose href doesn't start with "#/" (an internal
// SPA route) — see ui/dist/assets/index-*.js: `let n = e.href.startsWith("#/")`.
// There's no per-link override for that, so intercept the click here instead
// and navigate same-tab before the browser honors target="_blank".
document.addEventListener('click', function (e) {
  var link = e.target.closest && e.target.closest('a[href="/region-catalog/"]');
  if (!link) return;
  e.preventDefault();
  window.location.href = '/region-catalog/';
}, true);
