package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func sign(apiPath string, params map[string]string, appSecret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(apiPath)
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(params[k])
	}

	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(b.String()))
	return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
}
