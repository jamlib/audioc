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

Usage: audiocc [MODE] [OPTIONS] PATH
%s
Mode (specify only one):
  --artist "NAME"
    treat as specific artist

  --collection
    treat as collection of artists

Options:
  --bitrate "BITRATE"
    V0 (default)
      convert to variable 256kbps mp3
    320
      convert to constant 320kbps mp3

  --fix
    fixes incorrect track length, ie 1035:36:51

  --force
    processes all files, even if path info matches tag info

  --write
    write changes to disk

Debug:
  --version
    print program version, then exit
`

type Flags struct {
  Artist, Bitrate string
  Collection, Fix, Force, Version, Write bool
}

var flags = Flags{}

func init() {
  // setup mode
  flag.StringVar(&flags.Artist, "artist", "", "")
  flag.BoolVar(&flags.Collection, "collection", false, "")
  // setup options
  flag.StringVar(&flags.Bitrate, "bitrate", "V0", "")
  flag.BoolVar(&flags.Fix, "fix", false, "")
  flag.BoolVar(&flags.Force, "force", false, "")
  flag.BoolVar(&flags.Write, "write", false, "")
  // detup debug options
  flag.BoolVar(&flags.Version, "version", false, "")

  // --help
  flag.Usage = func() {
    fmt.Printf(printUsage, version, description, args)
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
