package bencoding

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

//Encoder takes care of encoding objects into byte streams.
//The result of the encoding operation is available in Encoder.Bytes.
//Consecutive operations are appended to the byte stream.
//
//Accepts only string, int/int64, []interface{} and map[string]interface{} as input.
type Encoder struct {
	w io.Writer //the result byte stream
}

// NewEncoder allocates and returns an Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

//Marshal wrapps Encoder.Encode.
func Marshal(in interface{}) ([]byte, error) {
	return encodeObject(in, false)
}

type Marshaller interface {
	MarshalBencoding() ([]byte, error)
}

//Encode encodes an object into a bencoded byte stream.
//The result of the operation is accessible through Encoder.Bytes.
//
//Example:
//	enc.Encode(23)
//	enc.Encode("test")
//	enc.Result //contains 'i23e4:test'
func (enc *Encoder) Encode(in interface{}) error {
	p, err := encodeObject(in, false)
	if err != nil {
		return err
	}
	_, err = enc.w.Write(p)
	return err
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
	switch t := reflect.TypeOf(in); t.Kind() {
	case reflect.String:
		return encodeString(in.(string)), nil
	case reflect.Int64:
		return encodeInteger(in.(int64)), nil
	case reflect.Int:
		return encodeInteger(int64(in.(int))), nil
	case reflect.Bool:
		if in.(bool) {
			return []byte("i1e"), nil
		} else {
			return []byte("i0e"), nil
		}
	case reflect.Slice:
		return encodeSlice(reflect.ValueOf(in))
	case reflect.Struct:
		return encodeStruct(reflect.ValueOf(in))
	case reflect.Ptr:
		val := reflect.ValueOf(in)
		if val.IsNil() && !omitable {
			return nil, fmt.Errorf("nil value")
		}
		return encodeObject(reflect.Indirect(val).Interface(), omitable)
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
	n := typ.NumField()
	var fs fields
	for i := 0; i < n; i++ {
		ftyp := typ.Field(i)
		if ftyp.PkgPath != "" {
			continue
		}
		var fname string
		var tag, opts string
		pieces := strings.SplitN(ftyp.Tag.Get("bencoding"), ",", 2)
		tag = pieces[0]
		if len(pieces) > 1 {
			opts = pieces[1]
		}
		if tag != "" {
			fname = tag
		} else {
			fname = ftyp.Name
		}
		fs = append(fs, field{i, fname, opts == "omitempty"})
	}
	sort.Sort(fs)
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
