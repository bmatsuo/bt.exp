package bencoding

import (
	"fmt"
	"strconv"
	"testing"
)

func DecodingError(t *testing.T, typ, msg, exp, recv string) {
	t.Errorf("Decoding %s: %s (expected %s) %s", typ, msg, exp, recv)
}

func it(t *testing.T, in string, exp int64, exp_err bool) {
	// Summarize a decoding, either expected or observed.
	dsumm := func(s string, d interface{}) string { return fmt.Sprintf("%s->%d", s, d) }

	d := NewDecoderBytes([]byte(in))
	var i int64
	err := d.Decode(&i)
	if !exp_err {
		if err != nil {
			DecodingError(t, "int", "unexpected error", dsumm(in, exp), err.Error())
		}
		if i != exp {
			DecodingError(t, "int", "unexpected result", strconv.FormatInt(exp, 10), dsumm(in, i))
		}
	} else {
		if err == nil {
			DecodingError(t, "int", "unexpected result", "Error", dsumm(in, i))
		}
	}
}

func TestInteger(t *testing.T) {
	it(t, "i23e", 23, false)
	it(t, "i124145124e", 124145124, false)
	it(t, "i0e", 0, false)
	it(t, "ie", 0, true)
	it(t, "i-e", 0, true)
	it(t, "i15155", 0, true)
	it(t, "55", 55, true)
}

func st(t *testing.T, in string, exp string, exp_err bool) {
	// Summarize a decoding, either expected or observed.
	dsumm := func(s string, d interface{}) string { return fmt.Sprintf("%s->%s", s, d) }

	d := NewDecoderBytes([]byte(in))
	var s string
	err := d.Decode(&s)
	if !exp_err {
		if err != nil {
			DecodingError(t, "string", "unexpected error", dsumm(in, exp), err.Error())
		}
		if s != exp {
			DecodingError(t, "string", "unexpected result", exp, dsumm(in, s))
		}
	} else {
		if err == nil {
			DecodingError(t, "string", "unexpected result", "Error", dsumm(in, s))
		}
	}
}

func TestString(t *testing.T) {
	st(t, "5:hello", "hello", false)
	st(t, "6:world", "world", true)
}

func lt(t *testing.T, in string, exp []interface{}, exp_err bool) {
	// Summarize a decoding, either expected or observed.
	dsumm := func(s string, list []interface{}) string { return fmt.Sprintf("%s->%v", s, list) }

	d := NewDecoderBytes([]byte(in))
	var list []interface{}
	err := d.Decode(&list)
	if exp_err {
		if err == nil {
			DecodingError(t, "string", "unexpected result", "Error", dsumm(in, list))
		}
		return
	}
	if err != nil {
		DecodingError(t, "list", "unexpected error", dsumm(in, exp), err.Error())
		return
	}
	if len(list) != len(exp) {
		DecodingError(t, "list", "unexpected result", fmt.Sprintf("%v", exp), dsumm(in, list))
		return
	}
	for i := range list {
		if list[i] != exp[i] {
			DecodingError(t, "list", "unexpected result", fmt.Sprintf("%v", exp), dsumm(in, list))
			break
		}
	}
}

func TestList(t *testing.T) {
	lt(t, "li124145124ee", []interface{}{int64(124145124)}, false)
	lt(t, "li15155ee", []interface{}{int64(15155)}, false)
	lt(t, "le", []interface{}{}, false)
	lt(t, "li15155e", []interface{}{}, true)
}

func dt(t *testing.T, in string, exp map[string]interface{}, exp_err bool) {
	// Summarize a decoding, either expected or observed.
	dsumm := func(s string, dict map[string]interface{}) string { return fmt.Sprintf("%s->%v", s, dict) }

	d := NewDecoderBytes([]byte(in))
	var dict map[string]interface{}
	err := d.Decode(&dict)
	if exp_err {
		if err == nil {
			DecodingError(t, "string", "unexpected result", "Error", dsumm(in, dict))
		}
		return
	}
	if err != nil {
		DecodingError(t, "list", "unexpected error", dsumm(in, exp), err.Error())
	}
	if len(dict) != len(exp) {
		DecodingError(t, "list", "unexpected result", fmt.Sprintf("%v", exp), dsumm(in, dict))
	}
	for i := range dict {
		if dict[i] != exp[i] {
			DecodingError(t, "list", "unexpected result", fmt.Sprintf("%v", exp), dsumm(in, dict))
			break
		}
	}
}

func TestDict(t *testing.T) {
	dt(t, "d4:blahi124145124ee", map[string]interface{}{"blah": int64(124145124)}, false)
	dt(t, "d5:hello5:worlde", map[string]interface{}{"hello": "world"}, false)
	dt(t, "de", map[string]interface{}{}, false)
	dt(t, "d4:highi5e", map[string]interface{}{}, true)
	dt(t, "d5:highi5ee", map[string]interface{}{}, true)
}
