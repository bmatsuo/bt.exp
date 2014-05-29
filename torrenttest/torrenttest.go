// Package torrenttest provides utilities for creating test torrent files.
package torrenttest

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash"
	"sync"
)

type PieceWriter struct {
	mut    sync.Mutex
	pieces []byte
	plen   int64
	offset int64
	sha    hash.Hash
	closed bool
}

func newPieceWriter(plen int64) *PieceWriter {
	return &PieceWriter{
		plen: plen,
		sha:  sha1.New(),
	}
}

func (w *PieceWriter) nonnil() {
	if w == nil {
		panic("nil receiver")
	}
}

func (w *PieceWriter) Close() error {
	w.nonnil()
	w.mut.Lock()
	defer w.mut.Unlock()
	if w.closed {
		return fmt.Errorf("closed")
	}
	w.pieces = append(w.pieces, w.sha.Sum(nil)...)
	w.sha = nil
	return nil
}

func (w *PieceWriter) Write(p []byte) (int, error) {
	w.nonnil()
	w.mut.Lock()
	defer w.mut.Unlock()
	var prefix, suffix []byte
	cut := w.plen - w.offset
	n := len(p)
	if int64(n) > cut {
		prefix, suffix = p[:int(cut)], p[int(cut):]
	} else {
		prefix = p
	}
	w.sha.Write(prefix)
	if suffix != nil {
		w.pieces = append(w.pieces, w.sha.Sum(nil)...)
		w.sha = sha1.New()
		w.offset = 0
		_n, err := w.Write(suffix)
		return n + _n, err
	}
	return n, nil
}

type FileInfo struct {
	path   string
	mut    sync.Mutex
	w      *PieceWriter
	md5    hash.Hash
	closed bool
}

func newFileInfo(path string, w *PieceWriter) *FileInfo {
	w.nonnil()
	info := &FileInfo{
		path: path,
		w:    w,
		md5:  md5.New(),
	}
	return info
}

func (h *FileInfo) nonnil() {
	if h == nil {
		panic("nil header")
	}
}

func (h *FileInfo) Write(p []byte) (int, error) {
	h.nonnil()
	h.mut.Lock()
	defer h.mut.Unlock()
	n, err := h.w.Write(p)
	if n > 0 {
		h.md5.Write(p[:n])
	}
	return n, err
}

func (h *FileInfo) Close() error {
	h.nonnil()
	h.mut.Lock()
	defer h.mut.Unlock()
	h.closed = true
	return h.w.Close()
}

func (h *FileInfo) MD5Sum() []byte {
	return h.md5.Sum(nil)
}

type Torrent struct {
	mut   sync.Mutex
	files []*FileInfo
	plen  int64
	w     *PieceWriter
}

func NewTorrent(announce string, plen int64) *Torrent {
	t := &Torrent{
		plen: plen,
		w:    newPieceWriter(plen),
	}
	return t
}

func (t *Torrent) nonnil() {
	if t == nil {
		panic("nil torrent")
	}
}

func (t *Torrent) NewFile(path string) *FileInfo {
	t.nonnil()
	t.mut.Lock()
	defer t.mut.Unlock()
	return newFileInfo(path, t.w)
}
