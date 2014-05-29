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
	Path   []string // File path components.
	Length int64    // Length in bytes.
	MD5Sum string   // Optional.
}

// The main contents of a Metadata type.
type TorrentInfo struct {
	Name        string      // Name of file (single-file mode) or directory (multi-file mode)
	Files       []*FileInfo // Nil if and only if single-file mode
	MD5Sum      string      // Optional -- Non-empty if and only if single-file mode.
	Pieces      string      // SHA-1 hash values of all pieces
	PieceLength int64       // Length in bytes.
	Private     bool        // Optional
}

// Returns true if info is in Single file mode.
func (info *TorrentInfo) SingleFileMode() bool { return info.Files == nil }

// The contents of a .torrent file.
type Metainfo struct {
	Info         *TorrentInfo // Required
	Announce     string       // Required
	CreationDate int64        // Optional
	Encoding     string       // Optional
	CreatedBy    string       // Optional
	Comment      string       // Optional
}

func tryCastKey(m map[string]interface{}, key string, action func(interface{}), required bool) {
	tryCast(key, m[key], action, required)
}

func tryCast(name string, v interface{}, action func(interface{}), required bool) {
	defer func() {
		if e := recover(); e != nil {
			if required {
				panic(fmt.Errorf("%s: %v", name, e))
			}
		}
	}()
	action(v)
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
	meta = new(Metainfo)
	tryCastKey(dict, "announce", func(v interface{}) { meta.Announce = v.(string) }, true)
	tryCastKey(dict, "encoding", func(v interface{}) { meta.Encoding = v.(string) }, false)
	tryCastKey(dict, "comment", func(v interface{}) { meta.Comment = v.(string) }, false)
	tryCastKey(dict, "created by", func(v interface{}) { meta.CreatedBy = v.(string) }, false)
	tryCastKey(dict, "creation date", func(v interface{}) { meta.CreationDate = v.(int64) }, false)
	var _info map[string]interface{}
	tryCastKey(dict, "info", func(v interface{}) { _info = v.(map[string]interface{}) }, true)
	info := new(TorrentInfo)
	meta.Info = info
	tryCastKey(dict, "name", func(v interface{}) { info.Name = v.(string) }, true)
	tryCastKey(dict, "pieces", func(v interface{}) { info.Pieces = v.(string) }, true)
	tryCastKey(dict, "md5sum", func(v interface{}) { info.MD5Sum = v.(string) }, false)
	tryCastKey(dict, "private", func(v interface{}) { info.Private = v.(int64) == 1 }, false)
	tryCastKey(dict, "piece length", func(v interface{}) { info.PieceLength = v.(int64) }, true)
	var _fileIs []interface{}
	tryCastKey(_info, "files", func(v interface{}) { _fileIs = v.([]interface{}) }, false)
	for i, _fileI := range _fileIs {
		var _file map[string]interface{}
		tryCast(fmt.Sprintf("file %d", i), _fileI,
			func(v interface{}) { _file = v.(map[string]interface{}) }, true)
		file := new(FileInfo)
		tryCastKey(_file, "md5sum", func(v interface{}) { file.MD5Sum = v.(string) }, false)
		tryCastKey(_file, "length", func(v interface{}) { file.Length = v.(int64) }, true)
		var path []interface{}
		tryCastKey(_file, "path", func(v interface{}) { path = v.([]interface{}) }, true)
		for j, elem := range path {
			tryCast(fmt.Sprintf("file %d: path element %d", i, j), elem,
				func(v interface{}) { file.Path = append(file.Path, v.(string)) }, true)
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
