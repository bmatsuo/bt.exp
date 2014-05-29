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
	Path   []string `bencoding:"path"`             // File path components.
	Length int64    `bencoding:"length"`           // Length in bytes.
	MD5Sum string   `bencoding:"md5sum,omitempty"` // Optional.
}

// The main contents of a Metadata type.
type TorrentInfo struct {
	Name        string      `bencoding:"name"`              // Name of file (single-file mode) or directory (multi-file mode)
	Files       []*FileInfo `bencoding:"files,omitempty"`   // Nil if and only if single-file mode
	Length      int64       `bencoding:"length,omitempty"`  // 0 if and only if in multi-file mode
	MD5Sum      string      `bencoding:"md5sum,omitempty"`  // Empty if and only if multi-file mode (optional).
	Pieces      []byte      `bencoding:"pieces"`            // SHA-1 hash values of all pieces
	PieceLength int64       `bencoding:"piece length"`      // Length in bytes.
	Private     bool        `bencoding:"private,omitempty"` // Optional
}

// Returns true if info is in Single file mode.
func (info *TorrentInfo) SingleFileMode() bool { return info.Files == nil }

// The contents of a .torrent file.
type Metainfo struct {
	Info         *TorrentInfo `bencoding:"info"`                    // Required
	Announce     string       `bencoding:"announce"`                // Required
	CreationDate int64        `bencoding:"creation date,omitempty"` // Optional
	Encoding     string       `bencoding:"encoding,omitempty"`      // Optional
	CreatedBy    string       `bencoding:"created by,omitempty"`    // Optional
	Comment      string       `bencoding:"comment,omitempty"`       // Optional
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

func ParseMetainfo(p []byte) (meta *Metainfo, err error) {
	var dict map[string]interface{}
	err = bencoding.Unmarshal(&dict, p)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("type error: %v", e)
		}
	}()
	wrapKey := func(key string, err error) error {
		if err == nil {
			return nil
		}
		return fmt.Errorf("%q %v", key, err)
	}
	meta = new(Metainfo)
	meta.Announce, err = StringKey(dict, "announce")
	if err != nil {
		return nil, wrapKey("announce", err)
	}
	meta.Encoding, err = StringKey(dict, "encoding")
	if err != nil && err != ErrNotFound {
		return nil, wrapKey("encoding", err)
	}
	meta.Comment, err = StringKey(dict, "comment")
	if err != nil && err != ErrNotFound {
		return nil, wrapKey("comment", err)
	}
	meta.CreatedBy, err = StringKey(dict, "created by")
	if err != nil && err != ErrNotFound {
		return nil, wrapKey("created by", err)
	}
	meta.CreationDate, err = Int64Key(dict, "creation date")
	if err != nil && err != ErrNotFound {
		return nil, wrapKey("creation date", err)
	}
	infodict, err := Key(dict, "info")
	if err != nil {
		return nil, wrapKey("info", err)
	}
	info := new(TorrentInfo)
	meta.Info = info
	info.Name, err = StringKey(infodict, "name")
	if err != nil {
		return nil, wrapKey("name", err)
	}
	info.Pieces, err = BytesKey(infodict, "pieces")
	if err != nil {
		return nil, wrapKey("pieces", err)
	}
	info.MD5Sum, err = StringKey(infodict, "md5sum")
	if err != nil && err != ErrNotFound {
		return nil, wrapKey("md5sum", err)
	}
	switch privbit, err := Int64Key(infodict, "private"); {
	case err == ErrNotFound:
		break
	case err != nil && err != ErrNotFound:
		return nil, wrapKey("private", err)
	case privbit == 0:
		break
	case privbit == 1:
		info.Private = true
	default:
		return nil, fmt.Errorf("\"private\" value invalid")
	}
	info.PieceLength, err = Int64Key(infodict, "piece length")
	if err != nil {
		return nil, wrapKey("piece length", err)
	}
	files, err := SliceKey(infodict, "files")
	if err == ErrNotFound {
		return meta, nil
	}
	if err != nil {
		return nil, wrapKey("files", err)
	}
	for _, filedict := range files {
		file := new(FileInfo)
		file.MD5Sum, err = StringKey(filedict, "md5sum")
		if err != nil {
			return nil, wrapKey("md5sum", err)
		}
		file.Length, err = Int64Key(filedict, "length")
		if err != nil {
			return nil, wrapKey("length", err)
		}
		path, err := SliceKey(filedict, "path")
		if err != nil {
			return nil, wrapKey("path", err)
		}
		for _, elem := range path {
			pathseg, err := String(elem)
			if err != nil {
				return nil, err
			}
			file.Path = append(file.Path, pathseg)
		}
		info.Files = append(info.Files, file)
	}
	return meta, nil
}

func ReadFile(torrent string) (*Metainfo, error) {
	p, err := ioutil.ReadFile(torrent)
	if err != nil {
		return nil, err
	}
	return ParseMetainfo(p)
}
