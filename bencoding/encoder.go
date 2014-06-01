package bencoding

import (
	"fmt"
	"io"
	"reflect"
	"sort"
)

// Encoder writes bencoded objects into an io.Writer.
type Encoder struct {
	w io.Writer //the result byte stream
}

// NewEncoder allocates and returns an Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

// Marshal wraps Encoder.Encode.
func Marshal(in interface{}) ([]byte, error) {
	return encodeObject(in, false)
}

// Marshaller implements custom marshalling of Bencoded values.
type Marshaller interface {
	MarshalBencoding() ([]byte, error)
}

// Encode bencodes an object and writes it to enc's output stream.  If v
// implements Marshaller, v.Marshaller() is written to the output stream.
// Otherwise a default encoding is of v is performed using runtime reflection.
func (enc *Encoder) Encode(v interface{}) error {
	p, err := encodeObject(v, false)
	if err != nil {
		return err
	}
	_, err = enc.w.Write(p)
	return err
}

var intKind = map[reflect.Kind]bool{
	reflect.Int:   true,
	reflect.Int64: true,
	reflect.Int32: true,
	reflect.Int16: true,
	reflect.Int8:  true,
}
var uintKind = map[reflect.Kind]bool{
	reflect.Uint:   true,
	reflect.Uint64: true,
	reflect.Uint32: true,
	reflect.Uint16: true,
	reflect.Uint8:  true,
}

func encodeObject(in interface{}, omitable bool) ([]byte, error) {
	if m, ok := in.(Marshaller); ok {
		return m.MarshalBencoding()
	}
	if as, ok := in.([]interface{}); ok {
		return encodeList(as)
	}
	if m, ok := in.(map[string]interface{}); ok {
		return encodeDict(m)
	}
	if p, ok := in.([]byte); ok {
		return encodeBytes(p), nil
	}
	t := reflect.TypeOf(in)
	k := t.Kind()
	switch {
	case k == reflect.Ptr:
		val := reflect.ValueOf(in)
		if val.IsNil() && !omitable {
			return nil, fmt.Errorf("nil value")
		}
		return encodeObject(reflect.Indirect(val).Interface(), omitable)
	case k == reflect.Struct:
		return encodeStruct(reflect.ValueOf(in))
	case k == reflect.String:
		return encodeString(reflect.ValueOf(in).String()), nil
	case k == reflect.Slice:
		return encodeSlice(reflect.ValueOf(in))
	case intKind[k]:
		return encodeInteger(reflect.ValueOf(in).Int()), nil
	case uintKind[k]:
		// TODO prevent overflow
		return encodeInteger(int64(reflect.ValueOf(in).Uint())), nil
	case k == reflect.Bool:
		if in.(bool) {
			return []byte("i1e"), nil
		} else {
			return []byte("i0e"), nil
		}
	default:
		return nil, fmt.Errorf("invalid type %T", in)
	}
}

type field struct {
	i         int
	name      string
	omitempty bool
}
type fields []field

func (fs fields) Len() int           { return len(fs) }
func (fs fields) Less(i, j int) bool { return fs[i].name < fs[j].name }
func (fs fields) Swap(i, j int)      { fs[i], fs[j] = fs[j], fs[i] }

// BUG: dictionary keys cannot contain commas
func encodeStruct(v reflect.Value) ([]byte, error) {
	typ := v.Type()
	fs := structFields(typ)
	var benc []byte
	benc = append(benc, 'd')
	for _, f := range fs {
		p, err := encodeObject(v.Field(f.i).Interface(), f.omitempty)
		if err != nil {
			return nil, err
		}
		if f.omitempty {
			if len(p) < 2 {
				panic("empty byte slice")
			}
			switch {
			case p[0] == '0' && p[1] == ':':
				continue
			case p[0] == 'l' && p[1] == 'e':
				continue
			case p[0] == 'd' && p[1] == 'e':
				continue
			}
		}
		namep := encodeString(f.name)
		benc = append(benc, namep...)
		benc = append(benc, p...)
	}
	benc = append(benc, 'e')
	return benc, nil
}

func encodeString(s string) []byte {
	if len(s) <= 0 {
		return []byte{'0', ':'}
	}
	return []byte(fmt.Sprintf("%d:%s", len(s), s))
}

func encodeBytes(p []byte) []byte {
	if len(p) <= 0 {
		return []byte{'0', ':'}
	}
	return []byte(fmt.Sprintf("%d:%s", len(p), p))
}

func encodeInteger(i int64) []byte {
	return []byte(fmt.Sprintf("i%de", i))
}

func encodeSlice(val reflect.Value) ([]byte, error) {
	n := val.Len()
	if n == 0 {
		return []byte{'l', 'e'}, nil
	}
	ret := []byte("l")
	for i := 0; i < n; i++ {
		p, err := encodeObject(val.Index(i).Interface(), false)
		if err != nil {
			return nil, err
		}
		ret = append(ret, p...)
	}
	ret = append(ret, 'e')
	return ret, nil
}

func encodeList(list []interface{}) ([]byte, error) {
	if len(list) <= 0 {
		return []byte{'l', 'e'}, nil
	}
	ret := []byte("l")
	for _, obj := range list {
		p, err := encodeObject(obj, false)
		if err != nil {
			return nil, err
		}
		ret = append(ret, p...)
	}
	ret = append(ret, 'e')
	return ret, nil
}

func encodeDict(m map[string]interface{}) ([]byte, error) {
	if len(m) <= 0 {
		return []byte{'d', 'e'}, nil
	}
	//sort the map >.<
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := []byte("d")
	for _, k := range keys {
		p, err := encodeObject(m[k], false)
		if err != nil {
			return nil, err
		}
		ret = append(ret, encodeString(k)...)
		ret = append(ret, p...)
	}
	ret = append(ret, 'e')
	return ret, nil
}
