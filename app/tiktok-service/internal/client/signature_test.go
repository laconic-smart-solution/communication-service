package client

import "testing"

func TestSignRequestStable(t *testing.T) {
	query := map[string]string{
		"app_key":   "12345",
		"timestamp": "1610000000",
		"sign":      "should-be-ignored",
	}
	got := signRequest("/order/202309/orders/search", query, `{"page_size":20}`, "secret")
	if got == "" {
		t.Fatal("expected non-empty signature")
	}
	again := signRequest("/order/202309/orders/search", query, `{"page_size":20}`, "secret")
	if got != again {
		t.Fatalf("signature not stable: %s vs %s", got, again)
	}
}
