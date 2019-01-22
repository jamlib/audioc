package audioc

import (
  "os"
  "fmt"
  "runtime"
  "strings"
  "path/filepath"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/libaudio/fsutil"
)

type Config struct {
  DirEntry string
  Flags flags
}

type audioc struct {
  DirEntry string
  Flags flags

  Ffmpeg ffmpeg.Ffmpeger
  Ffprobe ffprobe.Ffprober
  Image string
  Files []string
  Workers int
  Workdir string
}

type flags struct {
  Artist, Album, Bitrate string
  Collection, Fix, Force, Version, Write bool
}

func New(c *Config, ffm ffmpeg.Ffmpeger, ffp ffprobe.Ffprober) *audioc {
  return &audioc{ DirEntry: filepath.Clean(c.DirEntry), Flags: c.Flags,
    Ffmpeg: ffm, Ffprobe: ffp, Workers: runtime.NumCPU() }
}

func (a *audioc) Process() error {
  if !a.Flags.Write {
    fmt.Printf("\n* To write changes to disk, please provide flag: --write\n")
  }

  // ensure path is is valid directory
  fi, err := os.Stat(a.DirEntry)
  if err != nil || !fi.IsDir() {
    return fmt.Errorf("Invalid directory: %s", a.DirEntry)
  }

  // obtain audio file list
  a.Files = fsutil.FilesAudio(a.DirEntry)

  // if --artist mode, move innermost dir from a.DirEntry and add to each
  // file path within a.Files since this folder could be the album name.
  // if it is only the artist name, then will mimic --collection
  if a.Flags.Artist != "" {
    bpa := strings.Split(a.DirEntry, fsutil.PathSep)
    a.DirEntry = strings.Join(bpa[:len(bpa)-1], fsutil.PathSep)

    for i := range a.Files {
      a.Files[i] = bpa[len(bpa)-1] + fsutil.PathSep + a.Files[i]
    }
  }

  // group files by parent directory; call a.processBundle
  // a.processBundle found within bundle.go
  err = fsutil.BundleFiles(a.DirEntry, a.Files, a.processBundle)
  if err != nil {
    return err
  }

  fmt.Printf("\naudioc finished.\n")
  return nil
}
