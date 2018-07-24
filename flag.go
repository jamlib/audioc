package main

import (
  "fmt"
  "flag"
)

const (
  version = "0.0.0"
  description = "Clean up audio collection setting metadata & embedding artwork"
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

type Flags struct {
  Artist, Bitrate, ModTime string
  Collection, Fast, Fix, Force, Version, Write bool
}

var flags = Flags{}

func init() {
  // setup options
  flag.StringVar(&flags.Artist, "artist", "", "treat as specific artist")
  flag.StringVar(&flags.Bitrate, "bitrate", "V0", "convert to mp3 (V0=variable 256kbps, 320=constant 320kbps)")
  flag.BoolVar(&flags.Collection, "collection", false, "treat as collection of artists")
  flag.BoolVar(&flags.Fast, "fast", false, "skips album directory if starts w/ year")
  flag.BoolVar(&flags.Fix, "fix", false, "fixes incorrect track length, ie 1035:36:51")
  flag.BoolVar(&flags.Force, "force", false, "processes all files, even if path info matches tag info")
  flag.StringVar(&flags.ModTime, "modtime", "", "set modified timestamp of updated files")
  flag.BoolVar(&flags.Version, "version", false, "print program version, then exit")
  flag.BoolVar(&flags.Write, "write", false, "write changes to disk")

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
  if flags.Version {
    fmt.Printf("%s\n", version)
    return a, false
  }

  // show --help unless args
  if len(a) != 1 {
    flag.Usage()
    return a, false
  }

  // must specify --artist OR --collection
  if flags.Artist == "" && !flags.Collection {
    fmt.Printf("\nError: Must provide option --artist OR --collection\n")
    flag.Usage()
    return a, false
  }

  // default to V0 unless 320 specified
  if flags.Bitrate != "320" {
    flags.Bitrate = "V0"
  }

  return a, true
}
