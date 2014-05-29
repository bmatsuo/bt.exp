package bencode

import (
	"fmt"
	"io"
	"reflect"
	"sort"
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
	return encodeObject(in)
}

//Encode encodes an object into a bencoded byte stream.
//The result of the operation is accessible through Encoder.Bytes.
//
//Example:
//	enc.Encode(23)
//	enc.Encode("test")
//	enc.Result //contains 'i23e4:test'
func (enc *Encoder) Encode(in interface{}) error {
	p, err := encodeObject(in)
	if err != nil {
		return err
	}
	_, err = enc.w.Write(p)
	return err
}

func encodeObject(in interface{}) ([]byte, error) {
	switch t := reflect.TypeOf(in); t.Kind() {
	case reflect.String:
		return encodeString(in.(string)), nil
	case reflect.Int64:
		return encodeInteger(in.(int64)), nil
	case reflect.Int:
		return encodeInteger(int64(in.(int))), nil
	case reflect.Slice:
		return encodeList(in.([]interface{}))
	case reflect.Map:
		return encodeDict(in.(map[string]interface{}))
	default:
		return nil, fmt.Errorf("invalid type %T", in)
	}
}

func encodeString(s string) []byte {
	if len(s) <= 0 {
		return []byte{'0', ':'}
	}
	return []byte(fmt.Sprintf("%d:%s", len(s), s))
}

func encodeInteger(i int64) []byte {
	return []byte(fmt.Sprintf("i%de", i))
}

func encodeList(list []interface{}) ([]byte, error) {
	if len(list) <= 0 {
		// is this right?
		return []byte{'l', 'e'}, nil
	}
	ret := []byte("l")
	for _, obj := range list {
		p, err := encodeObject(obj)
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
		p, err := encodeObject(m[k])
		if err != nil {
			return nil, err
		}
		ret = append(ret, encodeString(k)...)
		ret = append(ret, p...)
	}
	ret = append(ret, 'e')
	return ret, nil
}
