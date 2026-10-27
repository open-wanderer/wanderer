package hooks

import (
	"net/http"
	"os"
	"testing"
	"time"

	"pocketbase/federation"
)

// Tests here receive deliveries on httptest servers, which the production
// delivery client refuses as loopback.
func TestMain(m *testing.M) {
	federation.SetDeliveryClientForTesting(&http.Client{Timeout: 10 * time.Second})
	os.Exit(m.Run())
}
