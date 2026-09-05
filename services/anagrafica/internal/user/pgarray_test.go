package user

import (
	"reflect"
	"testing"
)

func TestStringArray_ValueScanRoundTrip(t *testing.T) {
	cases := []StringArray{
		nil,
		{},
		{"shifts:read"},
		{"shifts:read", "shifts:write", "users:manage"},
		{`has "quotes"`, `has\backslash`, "has,comma"},
	}
	for _, c := range cases {
		value, err := c.Value()
		if err != nil {
			t.Fatalf("Value(%v): %v", c, err)
		}
		var got StringArray
		if err := got.Scan(value); err != nil {
			t.Fatalf("Scan(%v) from %v: %v", c, value, err)
		}
		want := c
		if want == nil {
			want = StringArray{}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("round trip mismatch: want %#v, got %#v (postgres literal: %q)", want, got, value)
		}
	}
}

func TestStringArray_ScanNil(t *testing.T) {
	var a StringArray = StringArray{"leftover"}
	if err := a.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if a != nil {
		t.Fatalf("expected nil after Scan(nil), got %#v", a)
	}
}
