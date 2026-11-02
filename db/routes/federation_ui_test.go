package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/ui"
)

// newFederationUIHandler builds a router with the two federation page routes
// registered exactly as db/main.go registers them.
func newFederationUIHandler(t *testing.T) http.Handler {
	t.Helper()

	app := newFederationAdminTestApp(t)
	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	r.GET("/federation/", FederationDashboard)
	r.GET("/federation/assets/{path...}", FederationAssets())

	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("build mux: %v", err)
	}
	return mux
}

func federationGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func readFederationExtFile(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(FederationExtFS(), name)
	if err != nil {
		t.Fatalf("read %s from FederationExtFS: %v", name, err)
	}
	return string(b)
}

// cspDirective returns the space separated tokens of one CSP directive.
func cspDirective(csp, name string) []string {
	for _, part := range strings.Split(csp, ";") {
		fields := strings.Fields(part)
		if len(fields) > 0 && fields[0] == name {
			return fields[1:]
		}
	}
	return nil
}

func TestFederationAssetsServesEmbeddedFiles(t *testing.T) {
	h := newFederationUIHandler(t)

	css := federationGet(t, h, "/federation/assets/federation.css")
	if css.Code != http.StatusOK {
		t.Fatalf("federation.css status = %d, want 200", css.Code)
	}
	if ct := css.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("federation.css Content-Type = %q, want text/css", ct)
	}
	if cc := css.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("federation.css Cache-Control = %q, want no-cache", cc)
	}
	if got, want := css.Body.String(), readFederationExtFile(t, "federation.css"); got != want {
		t.Errorf("federation.css body differs from the embedded file")
	}

	font := federationGet(t, h, "/federation/assets/fonts/ibm-plex-sans-latin-400-normal.woff2")
	if font.Code != http.StatusOK {
		t.Fatalf("woff2 status = %d, want 200", font.Code)
	}
	if ct := font.Header().Get("Content-Type"); ct != "font/woff2" {
		t.Errorf("woff2 Content-Type = %q, want font/woff2", ct)
	}

	if missing := federationGet(t, h, "/federation/assets/missing.js"); missing.Code != http.StatusNotFound {
		t.Errorf("missing.js status = %d, want 404", missing.Code)
	}

	page := federationGet(t, h, "/federation/")
	if page.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200", page.Code)
	}
	if !strings.Contains(page.Body.String(), "<html") {
		t.Errorf("dashboard body does not look like the dashboard HTML")
	}
}

// The dashboard HTML is only safe behind the headers FederationDashboard sets
// (CSP, X-Frame-Options), so the asset route must not hand it out, nor the notes.
func TestFederationAssetsRefuseNonAssetFiles(t *testing.T) {
	h := newFederationUIHandler(t)

	for _, name := range []string{"federation_ui.html", "FEDERATION_UI.HTML", "THIRD_PARTY.md"} {
		rec := federationGet(t, h, "/federation/assets/"+name)
		if rec.Code != http.StatusNotFound {
			t.Errorf("/federation/assets/%s status = %d, want 404", name, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "<html") {
			t.Errorf("/federation/assets/%s leaked the dashboard HTML", name)
		}
	}

	for _, name := range []string{
		"federation.css",
		"federation_app.js",
		"federation_theme.js",
		"vendor/alpine-csp-3.17.4.min.js",
		"fonts/ibm-plex-sans-latin-500-normal.woff2",
	} {
		rec := federationGet(t, h, "/federation/assets/"+name)
		if rec.Code != http.StatusOK {
			t.Errorf("/federation/assets/%s status = %d, want 200", name, rec.Code)
			continue
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("/federation/assets/%s X-Content-Type-Options = %q, want nosniff", name, got)
		}
	}
}

var federationAssetRefRe = regexp.MustCompile(`/federation/assets/([A-Za-z0-9_./-]+)`)

// assertFederationAssetRefsExist checks that every /federation/assets/ reference
// in content names a file that FederationExtFS can open.
func assertFederationAssetRefsExist(t *testing.T, label, content string) {
	t.Helper()
	for _, m := range federationAssetRefRe.FindAllStringSubmatch(content, -1) {
		if _, err := fs.Stat(FederationExtFS(), m[1]); err != nil {
			t.Errorf("%s references /federation/assets/%s which is not in FederationExtFS: %v", label, m[1], err)
		}
	}
}

func TestFederationDashboardStylesAndFontsAreSameOrigin(t *testing.T) {
	h := newFederationUIHandler(t)
	rec := federationGet(t, h, "/federation/")
	csp := rec.Header().Get("Content-Security-Policy")

	for _, name := range []string{"style-src", "font-src"} {
		got := cspDirective(csp, name)
		if len(got) != 1 || got[0] != "'self'" {
			t.Errorf("%s = %v, want exactly ['self']", name, got)
		}
	}

	html := readFederationExtFile(t, "federation_ui.html")
	for _, banned := range []string{"<style", "style=", ":style=", "fonts.googleapis.com", "fonts.gstatic.com", "cdn.jsdelivr.net/npm/remixicon"} {
		if strings.Contains(html, banned) {
			t.Errorf("federation_ui.html still contains %q", banned)
		}
	}
	if !strings.Contains(html, "/federation/assets/federation.css") {
		t.Errorf("federation_ui.html does not link /federation/assets/federation.css")
	}

	assertFederationAssetRefsExist(t, "federation_ui.html", html)
	assertFederationAssetRefsExist(t, "federation.css", readFederationExtFile(t, "federation.css"))
}

var (
	// .ri-name:before{content:"X"} as written by PocketBase's bundler (literal glyph)
	// and by federation.css (CSS escape such as \f0d1).
	iconRuleRe   = regexp.MustCompile(`\.(ri-[a-z0-9-]+):before\s*\{\s*content:\s*"([^"]*)"`)
	cssEscapeRe  = regexp.MustCompile(`^\\([0-9a-fA-F]{1,6})\s?$`)
	fontFaceURLs = regexp.MustCompile(`@font-face\s*\{[^}]*font-family:\s*['"]?remixicon['"]?[^}]*\}`)
)

// decodeCSSContent turns a CSS content string (either a literal glyph or a
// single backslash-hex escape) into the rune it denotes.
func decodeCSSContent(t *testing.T, name, raw string) rune {
	t.Helper()
	if m := cssEscapeRe.FindStringSubmatch(raw); m != nil {
		n, err := strconv.ParseInt(m[1], 16, 32)
		if err != nil {
			t.Fatalf("%s: bad escape %q: %v", name, raw, err)
		}
		return rune(n)
	}
	runes := []rune(raw)
	if len(runes) != 1 {
		t.Fatalf("%s: content %q is not a single code point", name, raw)
	}
	return runes[0]
}

// TestFederationIconFontMatchesPocketBase: the page declares its own
// @font-face for PocketBase's bundled remixicon font and its own codepoints. A
// PocketBase upgrade that moves the font or renumbers an icon fails here first.
func TestFederationIconFontMatchesPocketBase(t *testing.T) {
	if _, err := fs.Stat(ui.DistDirFS, "fonts/remixicon/remixicon.woff2"); err != nil {
		t.Fatalf("PocketBase's embedded UI no longer has fonts/remixicon/remixicon.woff2: %v", err)
	}

	css := readFederationExtFile(t, "federation.css")

	face := fontFaceURLs.FindString(css)
	if face == "" {
		t.Fatalf("federation.css has no @font-face for family remixicon")
	}
	if !strings.Contains(face, "url('/_/fonts/remixicon/remixicon.woff2')") {
		t.Errorf("remixicon @font-face does not use url('/_/fonts/remixicon/remixicon.woff2'): %s", face)
	}

	bundles, err := fs.Glob(ui.DistDirFS, "assets/index-*.css")
	if err != nil || len(bundles) == 0 {
		t.Fatalf("no assets/index-*.css in PocketBase's embedded UI (err=%v)", err)
	}
	pbIcons := map[string]rune{}
	for _, name := range bundles {
		f, err := ui.DistDirFS.Open(name)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		b, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, m := range iconRuleRe.FindAllStringSubmatch(string(b), -1) {
			runes := []rune(m[2])
			if len(runes) == 1 {
				pbIcons[m[1]] = runes[0]
			}
		}
	}
	if len(pbIcons) == 0 {
		t.Fatalf("found no .ri-*:before rules in PocketBase's CSS; the rule format changed")
	}

	ours := iconRuleRe.FindAllStringSubmatch(css, -1)
	if len(ours) != 14 {
		t.Fatalf("federation.css declares %d ri-* rules, want exactly 14", len(ours))
	}
	for _, m := range ours {
		name := m[1]
		want, ok := pbIcons[name]
		if !ok {
			t.Errorf("PocketBase's CSS has no .%s:before rule", name)
			continue
		}
		if got := decodeCSSContent(t, name, m[2]); got != want {
			t.Errorf("%s: federation.css has U+%04X, PocketBase has U+%04X", name, got, want)
		}
	}
}

const (
	federationCSP = "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; " +
		"connect-src 'self'; img-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

	federationAlpineFile   = "vendor/alpine-csp-3.17.4.min.js"
	federationAlpineSHA256 = "0d18d7f8d7910e2e0212f0f056b12f50bebc3abb7d88d2f7c7cb4c336fe4519a"
)

func TestFederationDashboardCSP(t *testing.T) {
	h := newFederationUIHandler(t)
	csp := federationGet(t, h, "/federation/").Header().Get("Content-Security-Policy")

	if csp != federationCSP {
		t.Errorf("Content-Security-Policy =\n  %q\nwant\n  %q", csp, federationCSP)
	}
	if got := cspDirective(csp, "script-src"); len(got) != 1 || got[0] != "'self'" {
		t.Errorf("script-src = %v, want exactly ['self']", got)
	}
	for _, banned := range []string{"http://", "https://", "unsafe-inline", "unsafe-eval"} {
		if strings.Contains(csp, banned) {
			t.Errorf("Content-Security-Policy contains %q", banned)
		}
	}
}

var (
	scriptTagRe  = regexp.MustCompile(`(?s)<script\b[^>]*>`)
	eventAttrRe  = regexp.MustCompile(`(?s)<[a-zA-Z][^>]*\s(on[a-zA-Z]+)\s*=`)
	alpineAttrRe = regexp.MustCompile(`\s((?:x-[a-z][a-z0-9:.\-]*)|(?:@[a-z][a-z0-9:.\-]*)|(?::[a-z][a-z0-9\-]*))="([^"]*)"`)
)

func TestFederationDashboardHTMLIsCSPSafe(t *testing.T) {
	html := readFederationExtFile(t, "federation_ui.html")

	for _, tag := range scriptTagRe.FindAllString(html, -1) {
		if !strings.Contains(tag, "src=") {
			t.Errorf("inline script tag %q", tag)
		}
	}
	if m := eventAttrRe.FindStringSubmatch(html); m != nil {
		t.Errorf("inline event handler attribute %q", m[1])
	}
	if strings.Contains(html, "x-init") {
		t.Errorf("html still uses x-init")
	}
	if !strings.Contains(html, `<body x-data="federationApp">`) {
		t.Errorf(`body does not use x-data="federationApp"`)
	}

	order := []string{
		`<script src="/federation/assets/federation_theme.js"></script>`,
		`<link rel="stylesheet" href="/federation/assets/federation.css">`,
		`<script defer src="/federation/assets/federation_app.js"></script>`,
		`<script defer src="/federation/assets/` + federationAlpineFile + `"></script>`,
	}
	prev := -1
	for _, want := range order {
		i := strings.Index(html, want)
		if i < 0 {
			t.Errorf("head is missing %s", want)
			continue
		}
		if i < prev {
			t.Errorf("%s is out of order", want)
		}
		prev = i
	}

	for _, m := range alpineAttrRe.FindAllStringSubmatch(html, -1) {
		name, val := m[1], strings.TrimSpace(m[2])
		for _, banned := range []string{"?.", "`", "=>"} {
			if strings.Contains(val, banned) {
				t.Errorf("%s=%q contains %q, which the Alpine CSP build cannot evaluate", name, val, banned)
			}
		}
		if strings.HasPrefix(val, "if ") || strings.HasPrefix(val, "if(") {
			t.Errorf("%s=%q starts with a statement", name, val)
		}
	}

	rest := strings.ReplaceAll(html, `xmlns="http://www.w3.org/2000/svg"`, "")
	rest = strings.ReplaceAll(rest, `placeholder="https://other.example.com"`, "")
	if strings.Contains(rest, "http://") || strings.Contains(rest, "https://") {
		t.Errorf("html references an absolute URL beyond the svg xmlns and the placeholder")
	}

	assertFederationAssetRefsExist(t, "federation_ui.html", html)

	app := readFederationExtFile(t, "federation_app.js")
	listener := strings.Index(app, "document.addEventListener('alpine:init'")
	register := strings.Index(app, "Alpine.data('federationApp'")
	if listener < 0 || register < listener {
		t.Errorf("federation_app.js must register Alpine.data('federationApp', ...) inside an alpine:init listener")
	}
}

func TestFederationVendoredAlpineHash(t *testing.T) {
	b, err := fs.ReadFile(FederationExtFS(), federationAlpineFile)
	if err != nil {
		t.Fatalf("read %s: %v", federationAlpineFile, err)
	}
	sum := sha256.Sum256(b)
	if got := hex.EncodeToString(sum[:]); got != federationAlpineSHA256 {
		t.Errorf("%s sha256 = %s, want %s", federationAlpineFile, got, federationAlpineSHA256)
	}
	if !strings.Contains(readFederationExtFile(t, "THIRD_PARTY.md"), federationAlpineSHA256) {
		t.Errorf("THIRD_PARTY.md does not record the sha256 %s", federationAlpineSHA256)
	}
	for _, banned := range []string{"new Function", "AsyncFunction"} {
		if strings.Contains(string(b), banned) {
			t.Errorf("%s contains %q", federationAlpineFile, banned)
		}
	}
}
