// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/googlecodelabs/tools/claat/types"
)

// CmdUpdate is the "claat update ..." subcommand.
// prefix is a URL prefix to prepend when using HTML format.
func CmdUpdate(prefix string) {
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}
	dirs, err := scanPaths(roots)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(dirs) == 0 {
		log.Fatalf("no codelabs found in %s", strings.Join(roots, ", "))
	}

	type result struct {
		dir  string
		meta *types.Meta
		err  error
	}
	ch := make(chan *result, len(dirs))
	for _, d := range dirs {
		go func(d string) {
			// random sleep up to 1 sec
			// to reduce number of rate limit errors
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
			meta, err := updateCodelab(d, prefix)
			ch <- &result{d, meta, err}
		}(d)
	}
	for range dirs {
		res := <-ch
		if res.err != nil {
			errorf(reportErr, res.dir, res.err)
		} else {
			log.Printf(reportOk, res.meta.ID)
		}
	}
}

// updateCodelab reads metadata from a dir/codelab.json file,
// re-exports the codelab just like it normally would in exportCodelab,
// and removes assets (images) which are not longer in use.
func updateCodelab(dir, prefix string) (*types.Meta, error) {
	// get stored codelab metadata and fail early if we can't
	meta, err := readMeta(filepath.Join(dir, metaFilename))
	if err != nil {
		return nil, err
	}
	// override allowed options from cli
	if prefix != "" {
		meta.Prefix = prefix
	}
	if *globalGA != "" {
		meta.MainGA = *globalGA
	}

	// fetch and parse codelab source
	clab, err := slurpCodelab(meta.Source)
	if err != nil {
		return nil, err
	}
	updated := types.ContextTime(clab.mod)
	meta.Context.Updated = &updated

	basedir := filepath.Join(dir, "..")
	newdir := codelabDir(basedir, &clab.Meta)
	imgdir := filepath.Join(newdir, imgDirname)

	// slurp codelab assets to disk and rewrite image URLs
	var client *http.Client
	if clab.typ == srcGoogleDoc {
		client, err = driveClient()
		if err != nil {
			return nil, err
		}
	}
	imgmap, err := slurpImages(client, meta.Source, imgdir, clab.Steps)
	if err != nil {
		return nil, err
	}

	// write codelab and its metadata
	if err := writeCodelab(newdir, clab.Codelab, &meta.Context); err != nil {
		return nil, err
	}

	// cleanup:
	// - remove original dir if codelab ID has changed and so has the output dir
	// - otherwise, remove images which are not in imgs
	old := codelabDir(basedir, &meta.Meta)
	if old != newdir {
		return &meta.Meta, os.RemoveAll(old)
	}
	visit := func(p string, fi os.FileInfo, err error) error {
		if err != nil || p == imgdir {
			return err
		}
		if fi.IsDir() {
			return filepath.SkipDir
		}
		if _, ok := imgmap[filepath.Base(p)]; !ok {
			return os.Remove(p)
		}
		return nil
	}
	return &meta.Meta, filepath.Walk(imgdir, visit)
}

// scanPaths looks for codelab metadata files in roots, recursively.
// The roots argument can contain overlapping directories as the return
// value is always de-duped.
func scanPaths(roots []string) ([]string, error) {
	type result struct {
		root string
		dirs []string
		err  error
	}
	ch := make(chan *result, len(roots))
	for _, r := range roots {
		go func(r string) {
			dirs, err := walkPath(r)
			ch <- &result{r, dirs, err}
		}(r)
	}
	var dirs []string
	for range roots {
		res := <-ch
		if res.err != nil {
			return nil, fmt.Errorf("%s: %v", res.root, res.err)
		}
		dirs = append(dirs, res.dirs...)
	}
	return unique(dirs), nil
}

// walkPath walks root dir recursively, looking for metaFilename files.
func walkPath(root string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		if filepath.Base(p) == metaFilename {
			dirs = append(dirs, filepath.Dir(p))
		}
		return nil
	})
	return dirs, err
}

// readMeta reads codelab metadata from file.
// It will convert legacy fields to the actual.
func readMeta(file string) (*types.ContextMeta, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var cm types.ContextMeta
	if err := json.Unmarshal(b, &cm); err != nil {
		return nil, err
	}
	if cm.Format == "" {
		cm.Format = "html"
	}
	return &cm, nil
}
