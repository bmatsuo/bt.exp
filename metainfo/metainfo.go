// Copyright 2012, Bryan Matsuo. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package metainfo provides utilities to work with torrent metainfo files.

This package API is unstable and may change without notice.
*/
package metainfo

/*  Filename:    metadata.go
 *  Author:      Bryan Matsuo <bmatsuo@soe.ucsc.edu>
 *  Created:     2012-03-04 20:29:46.043613 -0800 PST
 *  Description:
 */

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
