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

var flagVersion, flagCollection bool
var flagArtist string

func init() {
  // setup options
  flag.BoolVar(&flagVersion, "version", false, "print program version, then exit")
  flag.BoolVar(&flagCollection, "collection", false, "treat as collection of artists")
  flag.StringVar(&flagArtist, "artist", "", "treat as specific artist")

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

  return a, true
}
