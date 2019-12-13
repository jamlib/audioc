package main

import (
  "os"
  "fmt"
  "flag"
  "path/filepath"

  "github.com/jamlib/audioc"
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
audioc v%s
%s

Usage: audioc [MODE] [OPTIONS] PATH
%s
MODE (specify only one):
  --artist "ARTIST" --album "ALBUM"
    treat as specific album belonging to specific artist

  --artist "NAME"
    treat as specific artist

  --collection
    treat as collection of artists

OPTIONS:
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

func configFromFlags() (*audioc.Config, bool) {
  c := audioc.Config{}
  flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

  // set mode
  flags.StringVar(&c.Album, "album", "", "")
  flags.StringVar(&c.Artist, "artist", "", "")
  flags.BoolVar(&c.Collection, "collection", false, "")

  // set options
  flags.StringVar(&c.Bitrate, "bitrate", "V0", "")
  flags.BoolVar(&c.Fix, "fix", false, "")
  flags.BoolVar(&c.Force, "force", false, "")
  flags.BoolVar(&c.Write, "write", false, "")

  // set debug options
  var printVersion bool
  flags.BoolVar(&printVersion, "version", false, "")

  // create --help closure
  flags.Usage = func() {
    fmt.Printf(printUsage, version, description, args)
    fmt.Println()
  }

  // process flags
  flags.Parse(os.Args[1:])
  a := flags.Args()

  // --version
  if printVersion {
    fmt.Printf("%s\n", version)
    return &c, false
  }

  // show --help unless args
  if len(a) != 1 {
    flags.Usage()
    return &c, false
  }

  // must specify proper MODE
  if !c.Collection && c.Artist == "" {
    fmt.Printf("\nError: Must provide a valid MODE\n")
    flags.Usage()
    return &c, false
  }

  // default to V0 unless 320 specified
  if c.Bitrate != "320" {
    c.Bitrate = "V0"
  }

  c.Dir = filepath.Clean(a[0])
  return &c, true
}
