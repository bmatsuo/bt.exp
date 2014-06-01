// Copyright 2012, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*  Filename:    metadata.go
 *  Author:      Bryan Matsuo <bmatsuo@soe.ucsc.edu>
 *  Created:     2012-03-04 20:29:46.043613 -0800 PST
 *  Description:
 */

/*
Package metainfo provides utilities to work with torrent metainfo files.

This package API is unstable and may change without notice.
*/
package metainfo

import (
	"crypto/sha1"
	"io/ioutil"
	"os"

	"github.com/bmatsuo/torrent/bencoding"
)

// FileInfo serializes one file's metadata in a multi-file Info.
type FileInfo struct {
	Path   []string `bencoding:"path"`
	Length int64    `bencoding:"length"`
	MD5Sum string   `bencoding:"md5sum,omitempty"`
}

// Info serializes the BitTorrent info dictionary.
// Info represents both single-file and multi-file torrents.
// See the specification for information about modes and optional values:
// https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
type Info struct {
	Name        string     `bencoding:"name"`
	Files       []FileInfo `bencoding:"files,omitempty"`
	Length      int64      `bencoding:"length,omitempty"`
	MD5Sum      string     `bencoding:"md5sum,omitempty"`
	Pieces      []byte     `bencoding:"pieces"`
	PieceLength int64      `bencoding:"piece length"`
	Private     bool       `bencoding:"private,omitempty"`
}

// Returns true if info is in single-file mode.
func (info Info) SingleFileMode() bool {
	return len(info.Files) == 0
}

// Hash returns the (20 byte) SHA-1 hash of info.
func (info Info) Hash() ([]byte, error) {
	p, err := bencoding.Marshal(info)
	if err != nil {
		return nil, err
	}
	h := sha1.New()
	_, err = h.Write(p)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// Metainfo serializes the BitTorrent metainfo dictionary.
type Metainfo struct {
	Info         Info   `bencoding:"info"`
	Announce     string `bencoding:"announce"`
	CreationDate int64  `bencoding:"creation date,omitempty"`
	Encoding     string `bencoding:"encoding,omitempty"`
	CreatedBy    string `bencoding:"created by,omitempty"`
	Comment      string `bencoding:"comment,omitempty"`
}

// WriteFile creates a (.torrent) metainfo file.
func WriteFile(filename string, meta *Metainfo, perm os.FileMode) error {
	p, err := bencoding.Marshal(meta)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, p, perm)
}

// ReadFile reads a (.torrent) metainfo file.
func ReadFile(filename string) (*Metainfo, error) {
	p, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var meta Metainfo
	err = bencoding.Unmarshal(p, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
