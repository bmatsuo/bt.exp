package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bmatsuo/torrent/bencoding"
	"github.com/bmatsuo/torrent/metainfo"
)

func main() {
	force := flag.Bool("f", false, "overwrite existing torrent file")
	outpath := flag.String("o", "", "path of output torrent file")
	private := flag.Bool("p", false, "make a private torrent")
	comment := flag.String("c", "", "comment text")
	rec := flag.Bool("r", false, "recursively add files in directories")
	id := flag.String("id", "com.github.bmatsuo.torrent.cmd.mktorrent/0.0", "program identity")
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		log.Fatal("usage: %s [flags] <announce> <file> ...")
	}
	announce, files := args[0], args[1:]
	w, err := metainfo.NewWriter(512 << 10)
	if err != nil {
		log.Fatal("couldn't created torrent writer: %v", err)
	}
	for _, filename := range files {
		info, err := os.Stat(filename)
		if err != nil {
			log.Fatal("%q %v", filename, err)
		}
		if !*rec && info.IsDir() {
			log.Fatal("directory specified without -r: %q ", filename)
		}
	}
	for _, filename := range files {
		err := filepath.Walk(filename, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() { // rec check would be redundant
				return nil
			}

			metap, err := filepath.Rel(filename, path)
			if err != nil {
				return err
			}
			var metaps []string
			var base string
			for metap != "" {
				metap, base = filepath.Split(metap)
				metaps = append(metaps, "")
				copy(metaps, metaps[1:])
				metaps[0] = base
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			err = w.Open(metaps...)
			if err != nil {
				return err
			}
			_, err = io.Copy(w, f)
			if err != nil {
				return err
			}
			f.Close()

			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	name := filepath.Base(files[0])
	meta, err := w.Metainfo(name, announce)
	if err != nil {
		log.Fatal("could not create torrent: %v", err)
	}
	meta.CreationDate = time.Now().Unix()
	meta.CreatedBy = *id
	meta.Comment = *comment
	meta.Info.Private = *private
	if *outpath == "" {
		*outpath = fmt.Sprintf("%s.torrent", name)
	}
	mode := os.O_WRONLY | os.O_CREATE | os.O_APPEND | os.O_EXCL
	if *force {
		mode ^= os.O_EXCL
	}
	outf, err := os.OpenFile(*outpath, mode, 0755)
	outbuf := bufio.NewWriter(outf)
	if err != nil {
		log.Fatal(err)
	}
	defer outf.Close()
	err = bencoding.NewEncoder(outbuf).Encode(meta)
	if err != nil {
		log.Fatal("could not write torrent: %v", err)
	}
	err = outbuf.Flush()
	if err != nil {
		log.Fatal("could not flush torrent content: %v", err)
	}
}
