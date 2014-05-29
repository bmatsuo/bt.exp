/*
Package bencoding provides bencoding serialization.

The specification can be found at
https://wiki.theory.org/BitTorrentSpecification#Bencoding
*/
package bencoding

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// Unmarshal decodes the bencoded content of p into dst.
// p must contain exactly one bencoded value.
func Unmarshal(dst interface{}, p []byte) error {
	dec := NewDecoderBytes(p)
	err := dec.nextObject(reflect.ValueOf(dst))
	if err != nil {
		return err
	}
	if dec.pos < len(dec.stream) {
		return fmt.Errorf("trailing bytes")
	}
	return nil
}

//A Decoder reads and decodes bencoded objects from an input stream.
//It returns objects that are either an "Integer", "String", "List" or "Dict".
//
//Example usage:
//	d := bencode.NewDecoder([]byte("i23e4:testi123e"))
//	for !p.Consumed {
//		o, _ := p.Decode()
//		fmt.Printf("obj(%s): %#v\n", reflect.TypeOf(o).Name, o)
//	}
type Decoder struct {
	stream []byte
	pos    int
}

//NewDecoder creates a new decoder for the given token stream
func NewDecoderBytes(b []byte) *Decoder {
	return &Decoder{b, 0}
}

//Decode reads one object from the input stream
func (dec *Decoder) Decode(dst interface{}) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("destination is not a pointer")
	}
	if !val.IsNil() {
		return dec.nextObject(reflect.Indirect(val))
	}
	return fmt.Errorf("nil destination")
}

var (
	EOF = errors.New("the token stream is consumed")
)

/*
//DecodeAll reads all objects from the input stream
func (dec *Decoder) DecodeAll(res []interface{}) ([]interface{}, error) {
	for {
		var obj interface{}
		err := dec.Decode(&obj)
		if err != nil {
			return nil, err
		}
		res = append(res, obj)
	}
	return res, nil
}
*/

//fetch the next object at position 'pos' in 'stream'
func (self *Decoder) nextObject(val reflect.Value) error {
	if self.pos >= len(self.stream) {
		return EOF
	}
	switch c := self.stream[self.pos]; c {
	case 'i':
		return self.nextInteger(val)
	case 'l':
		return self.nextList(val)
	case 'd':
		return self.nextDict(val)
	default:
		if c >= '0' && c <= '9' {
			return self.nextString(val)
		}
		return fmt.Errorf("couldn't parse '%s' index %d (%s)", self.stream, self.pos, string(self.stream[self.pos]))
	}
}

var okInt = map[reflect.Kind]bool{
	reflect.Complex128: true,
	reflect.Complex64:  true,
	reflect.Float64:    true,
	reflect.Float32:    true,
	reflect.Int64:      true,
	reflect.Int32:      true,
	reflect.Int16:      true,
	reflect.Int8:       true,
	reflect.Uint64:     true,
	reflect.Uint32:     true,
	reflect.Uint16:     true,
	reflect.Uint8:      true,
}

//fetches next integer from stream and advances pos pointer
func (dec *Decoder) nextInteger(val reflect.Value) error {
	if dec.pos >= len(dec.stream) {
		return EOF
	}

	if dec.stream[dec.pos] != 'i' {
		return fmt.Errorf("not an integer")
	}
	dec.pos++

	typ := derefType(val.Type())
	if ok := okInt[typ.Kind()] || isEmptyInterface(typ); !ok {
		return fmt.Errorf("cannot decode integer to %T", val.Interface())
	}

	var neg bool
	if dec.pos >= len(dec.stream) {
		return fmt.Errorf("unterminated integer")
	}
	if dec.stream[dec.pos] == '-' {
		neg = true
		dec.pos++
	}
	start := dec.pos

	i := bytes.IndexFunc(dec.stream[dec.pos:], func(c rune) bool {
		return c < '0' || c > '9'
	})
	if i < 0 {
		return fmt.Errorf("unterminated integer")
	}
	dec.pos += i
	if dec.stream[dec.pos] != 'e' {
		return fmt.Errorf("unexpected byte %x", dec.stream[dec.pos])
	}
	intstr := string(dec.stream[start:dec.pos])
	dec.pos++
	if len(intstr) == 0 {
		return fmt.Errorf("unexpected integer terminator")
	}
	if intstr[0] == '0' {
		if len(intstr) == 1 && neg {
			return fmt.Errorf("invalid integer -0")
		}
		if len(intstr) > 1 {
			return fmt.Errorf("leading zero")
		}
	}
	x, err := strconv.ParseInt(intstr, 10, 64)
	if err != nil {
		return err
	}
	if neg {
		x = -x
	}

	val, _ = derefVal(val, true)
	val.Set(reflect.ValueOf(x))
	return nil
}

//fetches next string from stream and advances pos pointer
func (dec *Decoder) nextString(val reflect.Value) error {
	if dec.pos >= len(dec.stream) {
		return EOF
	}
	if dec.stream[dec.pos] < '0' || dec.stream[dec.pos] > '9' {
		return fmt.Errorf("not a string")
	}
	typ := derefType(val.Type())
	if ok := typ.Kind() == reflect.String || isEmptyInterface(typ); !ok {
		return fmt.Errorf("cannot decode string to %T", val.Interface())
	}

	// scan length
	start := dec.pos
	i := bytes.IndexFunc(dec.stream[start:], func(c rune) bool {
		return c < '0' || c > '9'
	})
	if i < 0 {
		return fmt.Errorf("unterminated string length specifier")
	}
	dec.pos += i
	if dec.stream[dec.pos] != ':' {
		return fmt.Errorf("unexpected byte %x", dec.stream[dec.pos])
	}
	slen, err := strconv.Atoi(string(dec.stream[start:dec.pos]))
	if err != nil {
		return err
	}
	dec.pos++

	// slice data
	if slen > len(dec.stream[dec.pos:]) {
		return fmt.Errorf("unexpected end of string")
	}
	res := string(dec.stream[dec.pos : dec.pos+slen])
	dec.pos += slen

	val, _ = derefVal(val, true)
	val.Set(reflect.ValueOf(res))
	return nil
}

//fetches a list (and its contents) from stream and advances pos
func (dec *Decoder) nextList(val reflect.Value) error {
	if dec.pos >= len(dec.stream) {
		return EOF
	}
	if derefKind(val) != reflect.Slice {
		return fmt.Errorf("cannot decode list to %T", val.Interface())
	}

	if dec.stream[dec.pos] != 'l' {
		return fmt.Errorf("not a list")
	}
	dec.pos++ //skip 'l'

	val, _ = derefVal(val, true)

	for {
		if dec.pos >= len(dec.stream) {
			return fmt.Errorf("unterminated list")
		}
		if dec.stream[dec.pos] == 'e' {
			dec.pos++ //skip 'e'
			return nil
		}
		elem := newElem(val)
		err := dec.nextObject(elem)
		if err != nil {
			return err
		}
		val.Set(reflect.Append(val, reflect.Indirect(elem)))
	}
	panic("unreachable")
}

//fetches a dict
//bencoded dicts must have their keys sorted lexically. but I guess
//we can ignore that and work with unsorted maps. (wtf?! sorted maps ...)
func (dec *Decoder) nextDict(val reflect.Value) error {
	if dec.pos >= len(dec.stream) {
		return EOF
	}
	typ := derefType(val.Type())
	if typ.Kind() != reflect.Map {
		return fmt.Errorf("cannot decode dictionary to %T", val.Interface())
	}
	if typ.Key().Kind() != reflect.String {
		return fmt.Errorf("cannot decode dictionary to %T", val.Interface())
	}
	vtyp := derefType(typ.Elem())
	if vtyp.Kind() != reflect.Interface || vtyp.NumMethod() > 0 {
		return fmt.Errorf("cannot decode dictionary to %T", val.Interface())
	}

	if dec.stream[dec.pos] != 'd' {
		return fmt.Errorf("not a dict")
	}
	dec.pos++ //skip 'd'

	var derref bool
	val, _ = derefVal(val, true)

	for {
		if dec.pos >= len(dec.stream) {
			return fmt.Errorf("unterminated dictionary")
		}
		if dec.stream[dec.pos] == 'e' {
			dec.pos++ //skip 'e'
			return nil
		}
		key := newKey(val)
		err := dec.nextString(key)
		if err != nil {
			return err
		}
		elem := newElem(val)
		err = dec.nextObject(elem)
		if err != nil {
			return err
		}
		if !derref {
			val, _ = derefVal(val, true)
			val.Set(reflect.MakeMap(val.Type()))
		}
		val.SetMapIndex(reflect.Indirect(key), reflect.Indirect(elem))
	}

	panic("unreachable")
}

func derefKind(val reflect.Value) reflect.Kind {
	k := val.Kind()
	if k != reflect.Ptr {
		return k
	}
	return derefType(val.Type().Elem()).Kind()
}

func derefType(t reflect.Type) reflect.Type {
	k := t.Kind()
	for k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return t
}

func isEmptyInterface(t reflect.Type) bool {
	return t.Kind() == reflect.Interface && t.NumMethod() == 0
}

func derefVal(val reflect.Value, create bool) (dval reflect.Value, foundnil bool) {
	if val.Kind() != reflect.Ptr {
		return val, false
	}
	if !val.IsNil() {
		return derefVal(reflect.Indirect(val), create)
	}
	if !create {
		return val, true
	}
	child := reflect.New(val.Type().Elem())
	val.Set(child)
	return derefVal(child, true)
}

func newElem(val reflect.Value) reflect.Value {
	return reflect.New(val.Type().Elem())
}

func newKey(val reflect.Value) reflect.Value {
	return reflect.New(val.Type().Key())
}