package main

import (
  "fmt"
  "flag"
)

const (
  version = "0.0.0"
  description = "Clean up audio collection setting id3 tags & embedding artwork"
)

const args = `
Positional Args:
  PATH           directory path
`

const printUsage = `
audiocc v%s
%s

Usage: audiocc [OPTIONS] PATH
%s
Options:
`

var flagCollection, flagForce, flagVersion, flagWrite bool
var flagArtist string

func init() {
  // setup options
  flag.StringVar(&flagArtist, "artist", "", "treat as specific artist")
  flag.BoolVar(&flagCollection, "collection", false, "treat as collection of artists")
  flag.BoolVar(&flagForce, "force", false, "probes all files, even if path looks good")
  flag.BoolVar(&flagVersion, "version", false, "print program version, then exit")
  flag.BoolVar(&flagWrite, "write", false, "write changes to disk")

  // --help
  flag.Usage = func() {
    fmt.Printf(printUsage, version, description, args)
    // print options from built-in flag helper
    flag.PrintDefaults()
    fmt.Println()
  }
}

func processFlags() ([]string, bool) {
  flag.Parse()
  a := flag.Args()

  // --version
  if flagVersion {
    fmt.Printf("%s\n", version)
    return a, false
  }

  // show --help unless args
  if len(a) != 1 {
    flag.Usage()
    return a, false
  }

  // must specify --artist OR --collection
  if flagArtist == "" && !flagCollection {
    fmt.Printf("\nError: Must provide option --artist OR --collection\n")
    flag.Usage()
    return a, false
  }

  return a, true
}
