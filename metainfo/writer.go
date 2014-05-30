package metainfo

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash"
	"sync"
)

var errClosed = fmt.Errorf("closed")

type pieceWriter struct {
	mut    sync.Mutex
	pieces []byte
	plen   int64
	offset int64
	sha    hash.Hash
	closed bool
}

func newPieceWriter(plen int64) *pieceWriter {
	return &pieceWriter{
		plen: plen,
		sha:  sha1.New(),
	}
}

func (w *pieceWriter) nonnil() {
	if w == nil {
		panic("nil receiver")
	}
}

func (w *pieceWriter) Pieces() []byte {
	w.nonnil()
	w.mut.Lock()
	defer w.mut.Unlock()
	return append([]byte(nil), w.pieces...)
}

func (w *pieceWriter) Close() error {
	w.nonnil()
	w.mut.Lock()
	defer w.mut.Unlock()
	if w.closed {
		return errClosed
	}
	if w.sha == nil {
		w.sha = sha1.New()
	}
	w.pieces = append(w.pieces, w.sha.Sum(nil)...)
	w.sha = nil
	return nil
}

func (w *pieceWriter) Write(p []byte) (int, error) {
	w.nonnil()
	w.mut.Lock()
	defer w.mut.Unlock()
	return w.write(p)
}

func (w *pieceWriter) write(p []byte) (int, error) {
	if w.closed {
		return 0, errClosed
	}
	var prefix, suffix []byte
	cut := w.plen - w.offset
	n := len(p)
	if int64(n) > cut {
		prefix, suffix = p[:int(cut)], p[int(cut):]
	} else {
		prefix = p
	}
	if w.sha == nil {
		w.sha = sha1.New()
	}
	w.sha.Write(prefix)
	if len(suffix) > 0 {
		w.pieces = append(w.pieces, w.sha.Sum(nil)...)
		w.sha = sha1.New()
		w.offset = 0
		_n, err := w.write(suffix)
		return n + _n, err
	}
	return n, nil
}

type fileInfoWriter struct {
	path   []string
	mut    sync.Mutex
	w      *pieceWriter
	length int64
	md5    hash.Hash
	closed bool
}

func newFileInfoWriter(w *pieceWriter, path []string) *fileInfoWriter {
	w.nonnil()
	info := &fileInfoWriter{
		path: path,
		w:    w,
		md5:  md5.New(),
	}
	return info
}

func (h *fileInfoWriter) nonnil() {
	if h == nil {
		panic("nil header")
	}
}

func (h *fileInfoWriter) Write(p []byte) (int, error) {
	h.nonnil()
	h.mut.Lock()
	defer h.mut.Unlock()
	n, err := h.w.Write(p)
	if n > 0 {
		h.md5.Write(p[:n])
	}
	h.length += int64(n)
	return n, err
}

func (h *fileInfoWriter) Close() error {
	h.nonnil()
	h.mut.Lock()
	defer h.mut.Unlock()
	h.closed = true
	return h.w.Close()
}

func (h *fileInfoWriter) MD5Sum() []byte {
	return h.md5.Sum(nil)
}

// Writer is used to compute file checksums and create Metainfo objects.
type Writer struct {
	mut    sync.Mutex
	closed bool
	files  []*fileInfoWriter
	file   *fileInfoWriter
	single bool
	plen   int64
	w      *pieceWriter
}

// NewWriter allocates and returns a new Writer.
func NewWriter(plen int64) (*Writer, error) {
	t := &Writer{
		plen: plen,
		w:    newPieceWriter(plen),
	}
	return t, nil
}

// NewWriterSingle returns a writer in single-file mode.  Writers retured by
// NewWriterSingle can be written to without opening a path.
func NewWriterSingle(plen int64, name string) (*Writer, error) {
	t, err := NewWriter(plen)
	if err != nil {
		return nil, err
	}
	t.mut.Lock()
	defer t.mut.Unlock()
	err = t.Open(name)
	t.single = true
	return t, nil
}

func (t *Writer) nonnil() {
	if t == nil {
		panic("nil torrent")
	}
}

// Open creates a new file entry in t.  Subsequent calls to Write increment
// the file's length counter.
func (t *Writer) Open(path ...string) error {
	t.nonnil()
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.closed {
		return errClosed
	}
	if t.file != nil && t.single {
		return fmt.Errorf("single-file writer cannot create new files")
	}
	if t.file != nil {
		t.file.Close()
	}
	file := newFileInfoWriter(t.w, path)
	t.files = append(t.files, file)
	t.file = file
	return nil
}

// Write adds bytes to t's open file.  Write returns an error t if t.Open() has
// not been called.
func (t *Writer) Write(p []byte) (int, error) {
	t.nonnil()
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.closed {
		return 0, errClosed
	}
	if t.file == nil {
		return 0, fmt.Errorf("no open file")
	}
	return t.file.Write(p)
}

// Close flushes checksum buffers and prevents future write operations on t.
func (t *Writer) Close() error {
	t.nonnil()
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.closed {
		return errClosed
	}
	if t.file != nil {
		t.file.Close()
		t.file = nil
	}
	t.w.Close()
	return nil
}

// Metainfo returns a skeleton Metainfo object from bytes written to t.
// Metainfo closes t if it is not already closed.  If t is in single-file mode,
// dir is ignored.  Otherwise it is used as the metainfo's Name field.
func (t *Writer) Metainfo(dir, announce string) (*Metainfo, error) {
	err := t.Close()
	if err != nil && err != errClosed {
		return nil, err
	}
	if t.single {
		return t.metainfoSingle(dir, announce)
	} else {
		return t.metainfoMulti(dir, announce)
	}
}

func (t *Writer) metainfoMulti(dir, announce string) (*Metainfo, error) {
	var info Info
	info.Name = dir
	for _, file := range t.files {
		fileinfo := FileInfo{
			Path:   file.path,
			Length: file.length,
		}
		if t.single {
			fileinfo.MD5Sum = fmt.Sprintf("%x", file.md5.Sum(nil))
		}
		info.Files = append(info.Files, fileinfo)
	}
	info.Pieces = t.w.Pieces()
	return &Metainfo{Info: &info}, nil
}

func (t *Writer) metainfoSingle(_, announce string) (*Metainfo, error) {
	var info Info
	info.Name = t.files[0].path[0]
	info.Length = t.files[0].length
	info.MD5Sum = fmt.Sprintf("%x", t.files[0].MD5Sum())
	info.Pieces = t.w.Pieces()
	return &Metainfo{Info: &info}, nil
}
