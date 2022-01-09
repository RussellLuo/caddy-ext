package requestbodyvar

import (
	"bytes"
	"testing"
)

func TestJSON_Query(t *testing.T) {
	j := JSON{buf: bytes.NewBufferString(`{"name":{"first":"Janet","last":"Prichard"},"age":47}`)}
	want := "Prichard"
	result := j.Query("name.last")
	if result != want {
		t.Fatalf("Result: got (%#v), want (%#v)", result, want)
	}
}

func TestXML_Query(t *testing.T) {
	x := XML{buf: bytes.NewBufferString(`<?xml version="1.0" encoding="UTF-8"?>
  <name>
    <first>Janet</first>
    <last>Prichard</last>
  </name>
  <age>47</age>`)}
	want := "Prichard"
	result := x.Query("name.last")
	if result != want {
		t.Fatalf("Result: got (%#v), want (%#v)", result, want)
	}
}
