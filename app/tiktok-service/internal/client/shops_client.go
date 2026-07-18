package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *tiktokClientImpl) ListAuthorizedShops(
	ctx context.Context,
	creds Credentials,
	accessToken string,
) ([]AuthorizedShop, error) {
	path := "/authorization/" + apiVersion + "/shops"
	data, err := c.doSignedJSON(ctx, creds, http.MethodGet, path, accessToken, "", nil, nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Shops []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Region string `json:"region"`
			Cipher string `json:"cipher"`
		} `json:"shops"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode shops: %w", err)
	}

	out := make([]AuthorizedShop, 0, len(payload.Shops))
	for _, shop := range payload.Shops {
		out = append(out, AuthorizedShop{
			ID:     strings.TrimSpace(shop.ID),
			Name:   strings.TrimSpace(shop.Name),
			Region: strings.TrimSpace(shop.Region),
			Cipher: strings.TrimSpace(shop.Cipher),
		})
	}
	return out, nil
}

func (c *tiktokClientImpl) ResolveShopCipher(
	ctx context.Context,
	creds Credentials,
	accessToken, preferredShopID string,
) (*AuthorizedShop, error) {
	shops, err := c.ListAuthorizedShops(ctx, creds, accessToken)
	if err != nil {
		return nil, err
	}
	if len(shops) == 0 {
		return nil, fmt.Errorf("no authorized tiktok shops for this token")
	}

	preferred := strings.TrimSpace(preferredShopID)
	if preferred != "" {
		for i := range shops {
			if shops[i].ID == preferred {
				return &shops[i], nil
			}
		}
	}
	return &shops[0], nil
}
