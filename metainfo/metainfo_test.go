package metainfo

/*  Filename:    metadata_test.go
 *  Author:      Bryan Matsuo <bmatsuo@soe.ucsc.edu>
 *  Created:     2012-03-04 20:29:46.043866 -0800 PST
 *  Description: For testing metadata.go
 */

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bmatsuo/torrent/bencoding"
)

// TestBencoding tests the bencoding tags on metainfo struct types.
func TestBencoding(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to determine working directory: %v", err)
	}
	testfiles, err := filepath.Glob(filepath.Join(cwd, "test", "torrents", "*"))
	if err != nil {
		t.Fatalf("failed to find test torrent files: %v", err)
	}
	if len(testfiles) == 0 {
		t.Fatalf("no test files found")
	}
	for _, filename := range testfiles {
		if !strings.HasSuffix(filename, ".torrent") {
			t.Logf("skipping non-torrent %q", filename)
			continue
		}
		base := filepath.Base(filename)
		origp, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Errorf("failed to read file: %v", err)
			continue
		}
		var meta Metainfo
		err = bencoding.Unmarshal(origp, &meta)
		if err != nil {
			t.Errorf("failed to read file: %v", err)
			continue
		}
		p, err := bencoding.Marshal(meta)
		if err != nil {
			t.Errorf("unable to marshal metainfo for %q: %v", base, err)
			continue
		}
		meta = Metainfo{}
		err = bencoding.Unmarshal(p, &meta)
		if err != nil {
			t.Errorf("unable to parse marshalled output for %q: %v", base, err)
			continue
		}
		cpp, _ := bencoding.Marshal(meta)
		if len(p) != len(cpp) {
			t.Errorf("unexpected output size %d for %q (expected %d)", len(p), base, len(cpp))
			continue
		}
		if !reflect.DeepEqual(p, cpp) {
			t.Logf("expected: %q", p)
			t.Logf("received: %q", cpp)
			t.Fatalf("unexpected serialization output for %q", base)
		}
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	cwd, err := os.Getwd()
	if err != nil {
		b.Fatalf("failed to determine working directory: %v", err)
	}
	allfiles, err := filepath.Glob(filepath.Join(cwd, "test", "torrents", "*"))
	if err != nil {
		b.Fatalf("failed to find test torrent files: %v", err)
	}
	type torrentBytes struct {
		name string
		p    []byte
	}
	torrents := make([]torrentBytes, 0, len(allfiles))
	for _, name := range allfiles {
		if filepath.Ext(name) == ".torrent" {
			p, err := ioutil.ReadFile(name)
			if err != nil {
				b.Fatal(err)
			}
			name := filepath.Base(name)
			torrents = append(torrents, torrentBytes{name, p})
		}
	}
	b.ResetTimer()
	var nbytes int64
	for i := 0; i < b.N; i++ {
		t := torrents[i%len(torrents)]
		err := bencoding.Unmarshal(t.p, new(Metainfo))
		if err != nil {
			b.Fatal(err)
		}
		nbytes += int64(len(t.p))
	}
	b.StopTimer()
	b.SetBytes(nbytes)
}

func BenchmarkMarshal(b *testing.B) {
	cwd, err := os.Getwd()
	if err != nil {
		b.Fatalf("failed to determine working directory: %v", err)
	}
	allfiles, err := filepath.Glob(filepath.Join(cwd, "test", "torrents", "*"))
	if err != nil {
		b.Fatalf("failed to find test torrent files: %v", err)
	}
	type torrent struct {
		name string
		meta *Metainfo
	}
	torrents := make([]torrent, 0, len(allfiles))
	for _, name := range allfiles {
		if filepath.Ext(name) == ".torrent" {
			p, err := ioutil.ReadFile(name)
			if err != nil {
				b.Fatal(err)
			}
			meta := new(Metainfo)
			err = bencoding.Unmarshal(p, meta)
			if err != nil {
				b.Fatal(name, err)
			}
			name := filepath.Base(name)
			_, err = bencoding.Marshal(meta)
			if err != nil {
				b.Fatal(err)
			}
			torrents = append(torrents, torrent{name, meta})
		}
	}
	b.ResetTimer()
	var nbytes int64
	for i := 0; i < b.N; i++ {
		t := torrents[i%len(torrents)]
		p, err := bencoding.Marshal(t.meta)
		if err != nil {
			b.Fatal(err)
		}
		nbytes += int64(len(p))
	}
	b.StopTimer()
	b.SetBytes(nbytes)
}
