package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// signRequest builds the TikTok Shop Open API HMAC-SHA256 signature.
// See: https://partner.tiktokshop.com/docv2/page/sign-your-api-request
func signRequest(path string, query map[string]string, body string, appSecret string) string {
	keys := make([]string, 0, len(query))
	for k := range query {
		if k == "sign" || k == "access_token" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(path)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(query[k])
	}
	b.WriteString(body)

	payload := appSecret + b.String() + appSecret
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
