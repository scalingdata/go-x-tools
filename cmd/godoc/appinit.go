// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build appengine

package main

// This file replaces main.go when running godoc under app-engine.
// See README.godoc-app for details.

import (
	"archive/zip"
	"log"
	"net/http"
	"path"
	"regexp"

	"github.com/scalingdata/go-x-tools/godoc"
	"github.com/scalingdata/go-x-tools/godoc/dl"
	"github.com/scalingdata/go-x-tools/godoc/proxy"
	"github.com/scalingdata/go-x-tools/godoc/short"
	"github.com/scalingdata/go-x-tools/godoc/static"
	"github.com/scalingdata/go-x-tools/godoc/vfs"
	"github.com/scalingdata/go-x-tools/godoc/vfs/mapfs"
	"github.com/scalingdata/go-x-tools/godoc/vfs/zipfs"

	"google.golang.org/appengine"
)

func init() {
	enforceHosts = !appengine.IsDevAppServer()
	playEnabled = true

	log.Println("initializing godoc ...")
	log.Printf(".zip file   = %s", zipFilename)
	log.Printf(".zip GOROOT = %s", zipGoroot)
	log.Printf("index files = %s", indexFilenames)

	goroot := path.Join("/", zipGoroot) // fsHttp paths are relative to '/'

	// read .zip file and set up file systems
	const zipfile = zipFilename
	rc, err := zip.OpenReader(zipfile)
	if err != nil {
		log.Fatalf("%s: %s\n", zipfile, err)
	}
	// rc is never closed (app running forever)
	fs.Bind("/", zipfs.New(rc, zipFilename), goroot, vfs.BindReplace)
	fs.Bind("/lib/godoc", mapfs.New(static.Files), "/", vfs.BindReplace)

	corpus := godoc.NewCorpus(fs)
	corpus.Verbose = false
	corpus.MaxResults = 10000 // matches flag default in main.go
	corpus.IndexEnabled = true
	corpus.IndexFiles = indexFilenames
	if err := corpus.Init(); err != nil {
		log.Fatal(err)
	}
	corpus.IndexDirectory = indexDirectoryDefault
	go corpus.RunIndexer()

	pres = godoc.NewPresentation(corpus)
	pres.TabWidth = 8
	pres.ShowPlayground = true
	pres.ShowExamples = true
	pres.DeclLinks = true
	pres.NotesRx = regexp.MustCompile("BUG")

	readTemplates(pres, true)

	mux := registerHandlers(pres)
	dl.RegisterHandlers(mux)
	short.RegisterHandlers(mux)

	// Register /compile and /share handlers against the default serve mux
	// so that other app modules can make plain HTTP requests to those
	// hosts. (For reasons, HTTPS communication between modules is broken.)
	proxy.RegisterHandlers(http.DefaultServeMux)

	log.Println("godoc initialization complete")
}
