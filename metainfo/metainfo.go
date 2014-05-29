// Copyright 2012, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metainfo

/*  Filename:    metadata.go
 *  Author:      Bryan Matsuo <bmatsuo@soe.ucsc.edu>
 *  Created:     2012-03-04 20:29:46.043613 -0800 PST
 *  Description:
 */

import (
	"fmt"
	"io/ioutil"

	"github.com/bmatsuo/torrent/bencoding"
)

// One file in a multi-file Metadata object.
type FileInfo struct {
	Path   []string `bencoding:"path"`
	Length int64    `bencoding:"length"`
	MD5Sum string   `bencoding:"md5sum,omitempty"`
}

// The main contents of a Metadata type.
type TorrentInfo struct {
	Name        string      `bencoding:"name"`
	Files       []*FileInfo `bencoding:"files,omitempty"`
	Length      int64       `bencoding:"length,omitempty"`
	MD5Sum      string      `bencoding:"md5sum,omitempty"`
	Pieces      []byte      `bencoding:"pieces"`
	PieceLength int64       `bencoding:"piece length"`
	Private     bool        `bencoding:"private,omitempty"`
}

// Returns true if info is in Single file mode.
func (info *TorrentInfo) SingleFileMode() bool { return info.Files == nil }

// The contents of a .torrent file.
type Metainfo struct {
	Info         *TorrentInfo `bencoding:"info"`
	Announce     string       `bencoding:"announce"`
	CreationDate int64        `bencoding:"creation date,omitempty"`
	Encoding     string       `bencoding:"encoding,omitempty"`
	CreatedBy    string       `bencoding:"created by,omitempty"`
	Comment      string       `bencoding:"comment,omitempty"`
}

var ErrNotFound = fmt.Errorf("key not found")
var ErrInvalidType = fmt.Errorf("value has the wrong type")

func Key(v interface{}, key string) (interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidType
	}
	if m == nil {
		return nil, ErrNotFound
	}
	v, ok = m[key]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func Bytes(v interface{}) ([]byte, error) {
	s, ok := v.(string)
	if !ok {
		return nil, ErrInvalidType
	}
	return []byte(s), nil
}

func String(v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", ErrInvalidType
	}
	return s, nil
}

func Int64(v interface{}) (int64, error) {
	x, ok := v.(int64)
	if !ok {
		return 0, ErrInvalidType
	}
	return x, nil
}

func Map(v interface{}) (map[string]interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidType
	}
	return m, nil
}

func Slice(v interface{}) ([]interface{}, error) {
	a, ok := v.([]interface{})
	if !ok {
		return nil, ErrInvalidType
	}
	return a, nil
}

func BytesKey(m interface{}, k string) ([]byte, error) {
	v, err := Key(m, k)
	if err != nil {
		return nil, err
	}
	return Bytes(v)
}

func StringKey(m interface{}, k string) (string, error) {
	v, err := Key(m, k)
	if err != nil {
		return "", err
	}
	return String(v)
}

func Int64Key(m interface{}, k string) (int64, error) {
	v, err := Key(m, k)
	if err != nil {
		return 0, err
	}
	return Int64(v)
}

func MapKey(v interface{}, k string) (map[string]interface{}, error) {
	v, err := Key(v, k)
	if err != nil {
		return nil, err
	}
	return Map(v)
}

func SliceKey(v interface{}, k string) ([]interface{}, error) {
	v, err := Key(v, k)
	if err != nil {
		return nil, err
	}
	return Slice(v)
}

func ReadFile(torrent string) (*Metainfo, error) {
	var meta Metainfo
	p, err := ioutil.ReadFile(torrent)
	if err != nil {
		return nil, err
	}
	err = bencoding.Unmarshal(&meta, p)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
