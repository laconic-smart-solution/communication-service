package client

import (
	"encoding/json"
	"testing"
)

func TestParseOrderItemsGetBody_ArrayData(t *testing.T) {
	body := []byte(`{
		"code": "0",
		"data": [
			{"sku":"SKU-1","name":"Widget","item_price":"100.50","paid_price":"100.50","currency":"THB"},
			{"sku":"SKU-1","name":"Widget","item_price":100.5,"paid_price":100.5,"currency":"THB"}
		]
	}`)
	items, err := parseOrderItemsGetBody(body)
	if err != nil {
		t.Fatal(err)
	}
	grouped := groupLazadaOrderItems(items)
	if len(grouped) != 1 {
		t.Fatalf("expected 1 grouped line, got %d", len(grouped))
	}
	if grouped[0].Quantity != 2 {
		t.Fatalf("quantity = %d", grouped[0].Quantity)
	}
	if grouped[0].SKU != "SKU-1" {
		t.Fatalf("sku = %q", grouped[0].SKU)
	}
	if grouped[0].Currency != "THB" {
		t.Fatalf("currency = %q", grouped[0].Currency)
	}
}

func TestParseOrderItemsGetBody_NumericCode(t *testing.T) {
	body := []byte(`{"code":0,"data":[{"shop_sku":"S1","name":"A","item_price":10}]}`)
	items, err := parseOrderItemsGetBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
}

func TestParseOrdersItemsGetBody(t *testing.T) {
	body := []byte(`{
		"code":"0",
		"data":[
			{"order_id":1107581273868367,"order_items":[
				{"sku":"A","name":"Product A","item_price":"50"}
			]}
		]
	}`)
	result, err := parseOrdersItemsGetBody(body)
	if err != nil {
		t.Fatal(err)
	}
	items := result["1107581273868367"]
	if len(items) != 1 {
		t.Fatalf("items=%v", items)
	}
	if items[0].ProductName != "Product A" {
		t.Fatalf("name=%q", items[0].ProductName)
	}
}

func TestFlexibleString(t *testing.T) {
	var s flexibleString
	if err := json.Unmarshal([]byte(`"abc"`), &s); err != nil || s.String() != "abc" {
		t.Fatalf("string: %v %q", err, s)
	}
	if err := json.Unmarshal([]byte(`12.5`), &s); err != nil || s.String() != "12.5" {
		t.Fatalf("number: %v %q", err, s)
	}
}
