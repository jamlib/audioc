package audioc

import (
  "os"
  "fmt"
  "runtime"
  "strings"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/libaudio/fsutil"
)

type Config struct {
  Dir, Artist, Album, Bitrate string
  Collection, Fix, Force, Write bool
}

type audioc struct {
  Config *Config
  Ffmpeg ffmpeg.Ffmpeger
  Ffprobe ffprobe.Ffprober
  Image string
  Files []string
  Workers int
  Workdir string
}

func New(c *Config, ffm ffmpeg.Ffmpeger, ffp ffprobe.Ffprober) *audioc {
  return &audioc{ Config: c, Ffmpeg: ffm, Ffprobe: ffp,
    Workers: runtime.NumCPU() }
}

func (a *audioc) Process() error {
  if !a.Config.Write {
    fmt.Printf("\n* To write changes to disk, please provide flag: --write\n")
  }

  // ensure path is is valid directory
  fi, err := os.Stat(a.Config.Dir)
  if err != nil || !fi.IsDir() {
    return fmt.Errorf("Invalid directory: %s", a.Config.Dir)
  }

  // obtain audio file list
  a.Files = fsutil.FilesAudio(a.Config.Dir)

  // if --artist mode, move innermost dir from a.Config.Dir and add to
  // each file path within a.Files since this folder could be the album name.
  // if it is only the artist name, then will mimic --collection
  if a.Config.Artist != "" {
    bpa := strings.Split(a.Config.Dir, fsutil.PathSep)
    a.Config.Dir = strings.Join(bpa[:len(bpa)-1], fsutil.PathSep)

    for i := range a.Files {
      a.Files[i] = bpa[len(bpa)-1] + fsutil.PathSep + a.Files[i]
    }
  }

  // group files by parent directory; call a.processBundle
  // a.processBundle found within bundle.go
  err = fsutil.BundleFiles(a.Config.Dir, a.Files, a.processBundle)
  if err != nil {
    return err
  }

  fmt.Printf("\naudioc finished.\n")
  return nil
}
