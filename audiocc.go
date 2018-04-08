package main

import (
  "os"
  "fmt"
  "log"
  "image"
  "strings"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
)

type audiocc struct {
  DirCur, DirEntry string
  SkipDir bool
  Image string
  Ffmpeg ffmpeger
  Ffprobe ffprober
}

type ffmpeger interface {
  ToMp3(i, q string, m ffmpeg.Metadata, o string) (string, error)
  OptimizeAlbumArt(s, d string) (string, error)
  Exec(args ...string) (string, error)
}

type ffprober interface {
  GetData(filePath string) (*ffprobe.Data, error)
  EmbeddedImage() (int, int, bool)
}

func main() {
  args, cont := processFlags()
  if !cont {
    os.Exit(0)
  }

  if !flagWrite {
    fmt.Printf("\n* To write changes to disk, please provide flag: --write\n")
  }

  // ensure path is is valid directory
  fmt.Printf("\nPath: %v\n\n", args[0])
  dir, err := checkDir(args[0])
  if err != nil {
    log.Fatal(err)
  }

  ffm, err := ffmpeg.New()
  if err != nil {
    log.Fatal(err)
  }

  ffp, err := ffprobe.New()
  if err != nil {
    log.Fatal(err)
  }

  a := &audiocc{ Ffmpeg: ffm, Ffprobe: ffp, DirEntry: dir }
  err = a.process()
  if err != nil {
    log.Fatal(err)
  }

  fmt.Printf("audiocc finished.\n")
}

func (a *audiocc) process() error {
  // iterate through all nested audio extensions within dir
  audio := filesByExtension(a.DirEntry, audioExts)
  for x := range audio {
    pi := getPathInfo(a.DirEntry, audio[x])

    // when to skip ahead
    if skipArtistOnCollection(pi.Dir) || a.SkipDir {
      continue
    }

    // info from path & filename
    i := &info{}
    i.fromFile(pi.File)
    i.fromPath(pi.Dir, string(os.PathSeparator))

    // info from embedded tags within audio file
    d, err := a.Ffprobe.GetData(pi.Fullpath)
    if err != nil {
      return err
    }

    // reset skip on dir change
    if pi.Dir != a.DirCur {
      a.SkipDir = false
    }

    // skip if sources match (unless --force)
    if i.matchProbe(d.Format.Tags) && !flagForce {
      a.SkipDir = true
      continue
    }

    art := &artwork{ Ffmpeg: a.Ffmpeg, Ffprobe: a.Ffprobe,
      PathInfo: pi, ImgDecode: image.DecodeConfig }

    // if current dir changed
    if pi.Dir != a.DirCur {
      if flagWrite {
        a.Image, err = art.process()
        if err != nil {
          return err
        }
      }
      a.DirCur = string(pi.Dir)
    }

    // prints
    fmt.Printf("Info: %#v\n", i)
    fmt.Printf("Tags: %#v\n\n", d.Format.Tags)

    if a.lastOfCurrentDir(x, audio) {
      // TODO: move files to updated path
    }

    // debug
    if x >= 50 {
      break
    }

  }
  return nil
}

// compares paths with DirCur to determine if all files have been processed
func (a *audiocc) lastOfCurrentDir(i int, paths []string) bool {
  if i <= len(paths)-1 {
    last := true
    if i < len(paths)-1 {
      ni := getPathInfo(a.DirEntry, paths[i+1])
      if ni.Dir == a.DirCur {
        last = false
      }
    }
    return last
  }
  return false
}

// true if --collection & artist path contains " - "
func skipArtistOnCollection(p string) bool {
  if flagCollection {
    pa := strings.Split(p, string(os.PathSeparator))
    if strings.Index(pa[0], " - ") != -1 {
      return true
    }
  }
  return false
}
