package util

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCheckRequestSignatureDigestAndCoverage(t *testing.T) {
	now := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
	body := []byte("{\n  \"type\": \"Create\", \"name\": \"Weg\"\n}\n")
	sum := sha256.Sum256(body)
	digest := base64.StdEncoding.EncodeToString(sum[:])
	shortDigest := base64.StdEncoding.EncodeToString(sum[:31])
	longDigest := base64.StdEncoding.EncodeToString(append(append([]byte{}, sum[:]...), 0))
	for _, tc := range []struct {
		name    string
		method  string
		headers string
		digest  string
		body    []byte
		wantErr bool
	}{
		{name: "matching body", digest: "SHA-256=" + digest, body: body},
		{name: "different body", digest: "SHA-256=" + digest, body: append(append([]byte{}, body...), ' '), wantErr: true},
		{name: "header absent", body: body, wantErr: true},
		{name: "post digest uncovered", headers: "(request-target) date", digest: "SHA-256=" + digest, body: body, wantErr: true},
		{name: "time uncovered", headers: "(request-target) digest", digest: "SHA-256=" + digest, body: body, wantErr: true},
		{name: "target and digest uncovered", method: http.MethodPut, headers: "date", body: body, wantErr: true},
		{name: "several algorithms", digest: "SHA-512=c2hvcnQ=, SHA-256=" + digest, body: body},
		{name: "lowercase algorithm", digest: "sha-256=" + digest, body: body},
		{name: "mixed case covered headers", headers: "(REQUEST-TARGET) DaTe DiGeSt", digest: "SHA-256=" + digest, body: body},
		{name: "other algorithm alone", digest: "SHA-512=c2hvcnQ=", body: body, wantErr: true},
		{name: "invalid base64", digest: "SHA-256=***", body: body, wantErr: true},
		{name: "short digest", digest: "SHA-256=" + shortDigest, body: body, wantErr: true},
		{name: "long digest", digest: "SHA-256=" + longDigest, body: body, wantErr: true},
		{name: "get with host", method: http.MethodGet, headers: "(request-target) date host", body: nil},
		{name: "get without host", method: http.MethodGet, headers: "(request-target) date", body: nil, wantErr: true},
		{name: "get with signed digest", method: http.MethodGet, headers: "date host digest", digest: "SHA-256=" + digest, body: body},
		{name: "get with different signed body", method: http.MethodGet, headers: "date host digest", digest: "SHA-256=" + digest, body: []byte("{}"), wantErr: true},
		{name: "post without host", headers: "date digest", digest: "SHA-256=" + digest, body: body},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = http.MethodPost
			}
			headers := tc.headers
			if headers == "" {
				headers = "(request-target) date digest"
			}
			req, err := http.NewRequest(method, "https://local.example/inbox", bytes.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Date", now.Format(http.TimeFormat))
			if tc.digest != "" {
				req.Header.Set("Digest", tc.digest)
			}
			req.Header.Set("Signature", fmt.Sprintf(`keyId="https://remote.example/actor#main-key",algorithm="rsa-sha256",headers="%s",signature="c2lnbmF0dXJl"`, headers))
			signature, err := CheckRequestSignature(req, tc.body, now)
			if (err != nil) != tc.wantErr {
				t.Fatalf("CheckRequestSignature error = %v, want error %v", err, tc.wantErr)
			}
			if !tc.wantErr && (signature == nil || signature.KeyID != "https://remote.example/actor#main-key") {
				t.Fatalf("unexpected signature metadata: %#v", signature)
			}
		})
	}
}

func TestCheckRequestSignatureTimeWindow(t *testing.T) {
	now := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)
	body := []byte("{}")
	sum := sha256.Sum256(body)
	digest := "SHA-256=" + base64.StdEncoding.EncodeToString(sum[:])
	seconds := func(value time.Time) string { return strconv.FormatInt(value.Unix(), 10) }
	for _, tc := range []struct {
		name      string
		algorithm string
		headers   string
		date      string
		created   string
		expires   string
		wantErr   bool
	}{
		{name: "current date", date: now.Format(http.TimeFormat)},
		{name: "future date at margin", date: now.Add(time.Hour).Format(http.TimeFormat)},
		{name: "future date beyond margin", date: now.Add(time.Hour + time.Second).Format(http.TimeFormat), wantErr: true},
		{name: "default expiry at margin", date: now.Add(-65 * time.Minute).Format(http.TimeFormat)},
		{name: "default expiry beyond margin", date: now.Add(-65*time.Minute - time.Second).Format(http.TimeFormat), wantErr: true},
		{name: "explicit expiry extends default", date: now.Add(-2 * time.Hour).Format(http.TimeFormat), expires: seconds(now.Add(time.Minute))},
		{name: "explicit expiry at margin", date: now.Add(-2 * time.Hour).Format(http.TimeFormat), expires: seconds(now.Add(-time.Hour))},
		{name: "explicit expiry beyond margin", date: now.Add(-2 * time.Hour).Format(http.TimeFormat), expires: seconds(now.Add(-time.Hour - time.Second)), wantErr: true},
		{name: "expiry cap at margin", date: now.Add(-13 * time.Hour).Format(http.TimeFormat), expires: seconds(now.Add(time.Hour))},
		{name: "expiry capped", date: now.Add(-13*time.Hour - time.Second).Format(http.TimeFormat), expires: seconds(now.Add(time.Hour)), wantErr: true},
		{name: "expiry before creation within margin", date: now.Add(-time.Minute).Format(http.TimeFormat), expires: seconds(now.Add(-10 * time.Minute))},
		{name: "invalid date", date: "not a date", wantErr: true},
		{name: "date absent", wantErr: true},
		{name: "obsolete date format", date: now.Format(time.RFC850)},
		{name: "asctime date format", date: now.Format(time.ANSIC)},
		{name: "hs2019 created", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now)},
		{name: "default algorithm created", algorithm: "absent", headers: "(request-target) (created) digest", created: seconds(now)},
		{name: "hs2019 falls back to date", algorithm: "hs2019", date: now.Format(http.TimeFormat)},
		{name: "hs2019 created overrides date", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now), date: now.Add(-24 * time.Hour).Format(http.TimeFormat)},
		{name: "hs2019 unused invalid date", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now), date: "not a date"},
		{name: "hs2019 future created within margin", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now.Add(30 * time.Minute))},
		{name: "hs2019 future created at margin", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now.Add(time.Hour))},
		{name: "hs2019 future created beyond margin", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now.Add(time.Hour + time.Second)), wantErr: true},
		{name: "hs2019 default expiry beyond margin", algorithm: "hs2019", headers: "(request-target) (created) digest", created: seconds(now.Add(-65*time.Minute - time.Second)), wantErr: true},
		{name: "hs2019 explicit expiry within margin", algorithm: "hs2019", headers: "(request-target) (created) (expires) digest", created: seconds(now.Add(-2 * time.Hour)), expires: seconds(now.Add(-30 * time.Minute))},
		{name: "created absent for pseudoheader", algorithm: "hs2019", headers: "(request-target) (created) digest", date: now.Format(http.TimeFormat), wantErr: true},
		{name: "invalid created", algorithm: "hs2019", headers: "(request-target) (created) digest", created: "today", wantErr: true},
		{name: "invalid expires", date: now.Format(http.TimeFormat), expires: "later", wantErr: true},
		{name: "rsa date determines creation", date: now.Format(http.TimeFormat), created: seconds(now.Add(24 * time.Hour))},
		{name: "rsa created pseudoheader", headers: "(request-target) (created) digest", date: now.Format(http.TimeFormat), created: seconds(now), wantErr: true},
		{name: "rsa expires pseudoheader", headers: "(request-target) date (expires) digest", date: now.Format(http.TimeFormat), expires: seconds(now.Add(5 * time.Minute)), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "https://local.example/inbox", bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Digest", digest)
			if tc.date != "" {
				req.Header.Set("Date", tc.date)
			}
			algorithm := tc.algorithm
			if algorithm == "" {
				algorithm = "rsa-sha256"
			}
			headers := tc.headers
			if headers == "" {
				headers = "(request-target) date digest"
			}
			parameters := []string{`keyId="https://remote.example/actor#main-key"`, fmt.Sprintf(`headers="%s"`, headers), `signature="c2lnbmF0dXJl"`}
			if algorithm != "absent" {
				parameters = append(parameters, fmt.Sprintf(`algorithm="%s"`, algorithm))
			}
			if tc.created != "" {
				parameters = append(parameters, "created="+tc.created)
			}
			if tc.expires != "" {
				parameters = append(parameters, "expires="+tc.expires)
			}
			req.Header.Set("Signature", strings.Join(parameters, ","))
			_, err = CheckRequestSignature(req, body, now)
			if (err != nil) != tc.wantErr {
				t.Fatalf("CheckRequestSignature error = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

func TestVerifySignatureChecksBodyAndTime(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	now := time.Now().UTC().Truncate(time.Second)
	body := []byte("{\n  \"type\": \"Create\"\n}\n")
	actor := "https://remote.example/actor"
	for _, tc := range []struct {
		name      string
		date      time.Time
		created   time.Time
		expires   time.Time
		replace   bool
		relabel   bool
		wantValid bool
	}{
		{name: "current date", date: now, wantValid: true},
		{name: "different received body", date: now, replace: true},
		{name: "expired date", date: now.Add(-2 * time.Hour)},
		{name: "future date within margin", date: now.Add(30 * time.Minute), wantValid: true},
		{name: "hs2019 future created within margin", created: now.Add(30 * time.Minute), wantValid: true},
		{name: "hs2019 elapsed expiry within margin", created: now.Add(-2 * time.Hour), expires: now.Add(-30 * time.Minute), wantValid: true},
		{name: "hs2019 elapsed expiry beyond margin", created: now.Add(-2 * time.Hour), expires: now.Add(-2 * time.Hour)},
		{name: "changed algorithm with unsigned date", created: now.Add(-2 * time.Hour), relabel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "https://local.example/inbox", bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Forwarded-Path", "/inbox")
			sum := sha256.Sum256(body)
			req.Header.Set("Digest", "SHA-256="+base64.StdEncoding.EncodeToString(sum[:]))
			headers := []string{"(request-target)"}
			lines := []string{"(request-target): post /inbox"}
			parameters := []string{fmt.Sprintf(`keyId="%s#main-key"`, actor), `algorithm="hs2019"`}
			if !tc.created.IsZero() {
				value := strconv.FormatInt(tc.created.Unix(), 10)
				headers = append(headers, "(created)")
				lines = append(lines, "(created): "+value)
				parameters = append(parameters, "created="+value)
			} else {
				req.Header.Set("Date", tc.date.Format(http.TimeFormat))
				headers = append(headers, "date")
				lines = append(lines, "date: "+req.Header.Get("Date"))
			}
			if !tc.expires.IsZero() {
				value := strconv.FormatInt(tc.expires.Unix(), 10)
				headers = append(headers, "(expires)")
				lines = append(lines, "(expires): "+value)
				parameters = append(parameters, "expires="+value)
			}
			headers = append(headers, "digest")
			lines = append(lines, "digest: "+req.Header.Get("Digest"))
			if tc.relabel {
				headers = append(headers, "host")
				lines = append(lines, "host: "+req.Host)
			}
			base := sha256.Sum256([]byte(strings.Join(lines, "\n")))
			signed, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, base[:])
			if err != nil {
				t.Fatal(err)
			}
			parameters = append(parameters, fmt.Sprintf(`headers="%s"`, strings.Join(headers, " ")), fmt.Sprintf(`signature="%s"`, base64.StdEncoding.EncodeToString(signed)))
			req.Header.Set("Signature", strings.Join(parameters, ","))
			if tc.relabel {
				valid, err := VerifySignature(nil, req, publicKey, body, actor)
				if valid || err == nil {
					t.Fatalf("expired hs2019 request accepted: valid=%v error=%v", valid, err)
				}
				req.Header.Set("Signature", strings.Replace(req.Header.Get("Signature"), `algorithm="hs2019"`, `algorithm="rsa-sha256"`, 1))
				req.Header.Set("Date", now.Format(http.TimeFormat))
			}
			received := body
			if tc.replace {
				received = append(append([]byte{}, body...), ' ')
			}
			valid, err := VerifySignature(nil, req, publicKey, received, actor)
			if tc.wantValid && (!valid || err != nil) {
				t.Fatalf("valid request rejected: %v", err)
			}
			if !tc.wantValid && (valid || err == nil) {
				t.Fatalf("request accepted: valid=%v error=%v", valid, err)
			}
		})
	}
}

func TestVerifySignatureHeaderAndParameterCoverage(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	actor := "https://remote.example/actor"
	body := []byte(`{"type":"Create"}`)
	for _, tc := range []struct {
		name      string
		method    string
		path      string
		headers   string
		keyID     string
		otherKey  bool
		change    func(*testing.T, *http.Request)
		wantValid bool
	}{
		{name: "complete post", wantValid: true},
		{name: "unsupported algorithm label", change: func(t *testing.T, req *http.Request) {
			req.Header.Set("Signature", strings.Replace(req.Header.Get("Signature"), `algorithm="rsa-sha256"`, `algorithm="rsa-sha512"`, 1))
		}},
		{name: "forwarded escaped path", path: "/inbox/%61", wantValid: true},
		{name: "different signing key", otherKey: true},
		{name: "different signature bytes", change: func(t *testing.T, req *http.Request) {
			raw := req.Header.Get("Signature")
			start := strings.LastIndex(raw, `signature="`) + len(`signature="`)
			value, err := base64.StdEncoding.DecodeString(raw[start : len(raw)-1])
			if err != nil {
				t.Fatal(err)
			}
			value[0] ^= 1
			req.Header.Set("Signature", raw[:start]+base64.StdEncoding.EncodeToString(value)+`"`)
		}},
		{name: "different signed host", change: func(t *testing.T, req *http.Request) {
			t.Setenv("ORIGIN", "https://other.local.example")
		}},
		{name: "different signed target", change: func(t *testing.T, req *http.Request) {
			req.Header.Set("X-Forwarded-Path", "/other-inbox")
		}},
		{name: "different signed method", change: func(t *testing.T, req *http.Request) {
			req.Method = http.MethodGet
		}},
		{name: "different signed content type", change: func(t *testing.T, req *http.Request) {
			req.Header.Set("Content-Type", "application/json")
		}},
		{name: "get with signed host", method: http.MethodGet, headers: "(request-target) date host", wantValid: true},
		{name: "get without signed host", method: http.MethodGet, headers: "(request-target) date"},
		{name: "post without signed host", headers: "(request-target) date digest content-type", wantValid: true},
		{name: "quoted key id with comma", keyID: actor + "/key,archive#main", wantValid: true},
		{name: "optional parameter whitespace", wantValid: true, change: func(t *testing.T, req *http.Request) {
			raw := req.Header.Get("Signature")
			for _, parameter := range []string{"keyId", "algorithm", "headers", "signature"} {
				raw = strings.Replace(raw, parameter+"=", parameter+" \t= ", 1)
			}
			req.Header.Set("Signature", strings.ReplaceAll(raw, `",`, "\", \t"))
		}},
		{name: "duplicate key parameter", change: func(t *testing.T, req *http.Request) {
			req.Header.Set("Signature", req.Header.Get("Signature")+`,keyId="https://remote.example/actor#other"`)
		}},
		{name: "duplicate header parameter", change: func(t *testing.T, req *http.Request) {
			req.Header.Set("Signature", req.Header.Get("Signature")+`,headers="(request-target) date digest host content-type"`)
		}},
		{name: "authorization signature", wantValid: true, change: func(t *testing.T, req *http.Request) {
			req.Header.Set("Authorization", "Signature "+req.Header.Get("Signature"))
			req.Header.Del("Signature")
		}},
		{name: "two signature locations", change: func(t *testing.T, req *http.Request) {
			req.Header.Set("Authorization", "Signature "+req.Header.Get("Signature"))
		}},
		{name: "unterminated quoted parameter", change: func(t *testing.T, req *http.Request) {
			raw := req.Header.Get("Signature")
			req.Header.Set("Signature", raw[:len(raw)-1])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ORIGIN", "https://local.example")
			method := tc.method
			if method == "" {
				method = http.MethodPost
			}
			headers := tc.headers
			if headers == "" {
				headers = "(request-target) date digest host content-type"
			}
			keyID := tc.keyID
			if keyID == "" {
				keyID = actor + "#main-key"
			}
			req, err := http.NewRequest(method, "https://local.example/inbox", bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			path := tc.path
			if path == "" {
				path = "/inbox"
			}
			req.Header.Set("X-Forwarded-Path", path)
			req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
			req.Header.Set("Content-Type", "application/activity+json")
			sum := sha256.Sum256(body)
			req.Header.Set("Digest", "SHA-256="+base64.StdEncoding.EncodeToString(sum[:]))
			lines := make([]string, 0)
			for _, header := range strings.Fields(headers) {
				value := req.Header.Get(header)
				switch header {
				case "(request-target)":
					value = strings.ToLower(method) + " " + path
				case "host":
					value = "local.example"
				}
				lines = append(lines, header+": "+value)
			}
			base := sha256.Sum256([]byte(strings.Join(lines, "\n")))
			signingKey := key
			if tc.otherKey {
				signingKey = otherKey
			}
			signed, err := rsa.SignPKCS1v15(rand.Reader, signingKey, crypto.SHA256, base[:])
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Signature", fmt.Sprintf(`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`, keyID, headers, base64.StdEncoding.EncodeToString(signed)))
			if tc.change != nil {
				tc.change(t, req)
			}
			valid, err := VerifySignature(nil, req, publicKey, body, actor)
			if tc.wantValid && (!valid || err != nil) {
				t.Fatalf("valid request rejected: %v", err)
			}
			if !tc.wantValid && (valid || err == nil) {
				t.Fatalf("request accepted: valid=%v error=%v", valid, err)
			}
		})
	}
}
