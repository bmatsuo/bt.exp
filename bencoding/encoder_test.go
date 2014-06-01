package bencoding

import (
	"fmt"
	"testing"
)

func TestMarshal_success(t *testing.T) {
	type MyString string
	type MyInt int32
	for _, test := range []struct {
		v      interface{}
		expect string
	}{
		{[]byte("hello"), "5:hello"},
		{"hello", "5:hello"},
		{MyString("world!"), "6:world!"},
		{-13, "i-13e"},
		{uint32(10), "i10e"},
		{MyInt(0), "i0e"},
		{true, "i1e"},
		{false, "i0e"},
		{[]interface{}{1, "2", struct{ A int64 }{3}}, "li1e1:2d1:Ai3eee"},
		{map[string]interface{}{
			"hello":   "world",
			"charset": "utf-8",
		}, "d7:charset5:utf-85:hello5:worlde"},
		{struct {
			A string `bencoding:"a,omitempty"`
			B int64  `bencoding:"b"`
			C bool   `bencoding:"c"`
		}{}, "d1:bi0e1:ci0ee"},
	} {
		p, err := Marshal(test.v)
		if err != nil {
			t.Errorf("marshal %#v: %v", test.v, err)
			continue
		}
		if string(p) != test.expect {
			t.Errorf("marshal %#v got %q (expect %q)", test.v, p, test.expect)
			continue
		}
	}
}

func TestMarshal_failure(t *testing.T) {
	for _, test := range []struct {
		v interface{}
	}{
		{func() { fmt.Println("hello, bencoding") }},
		{make(chan int)},
	} {
		p, err := Marshal(test.v)
		if err == nil {
			t.Errorf("marshal %#v: unexpected value %q", test.v, p)
			continue
		}
	}
}
