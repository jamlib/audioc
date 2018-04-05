package main

import (
  "os"
  "fmt"
  "log"
  "strings"

  "github.com/JamTools/goff/ffprobe"
)

type audiocc struct {
  DirCur, DirEntry string
  Images []string
}

func main() {
  args, cont := processFlags()
  if !cont {
    os.Exit(1)
  }

  dir, err := checkDir(args[0])
  if err != nil {
    fmt.Printf("Error: %v\n%v\n\n", err.Error(), dir)
    os.Exit(1)
  }

  fmt.Printf("\nFlag collection: %v\n", flagCollection)
  if flagArtist != "" {
    fmt.Printf("Flag artist: %v\n\n", flagArtist)
  }

  fmt.Printf("Path: %v\n\n", dir)
  a := &audiocc{ DirEntry: dir }
  err = a.process()
  if err != nil {
    log.Fatal(err)
  }
}

func (a *audiocc) process() error {
  audio := filesByExtension(a.DirEntry, audioExts)
  for x := range audio {
    pi := getPathInfo(a.DirEntry, audio[x])

    if skipArtistOnCollection(pi.Dir) {
      continue
    }

    if pi.Dir != a.DirCur {
      // process images once per directory
      a.processImages(pi.Fulldir)
      a.DirCur = string(pi.Dir)
    }

    i := &info{}
    i.fromFile(pi.File)
    i.fromPath(pi.Dir, string(os.PathSeparator))
    fmt.Printf("Info: %#v\n", i)

    d, err := ffprobe.GetData(pi.Fullpath)
    if err != nil {
      return err
    }
    fmt.Printf("Tags: %#v\n\n", d.Format.Tags)

    if a.lastOfCurrentDir(x, audio) {
      // TODO: move files to updated path
      fmt.Printf("Images:\n%v\n\n", a.Images)
    }

    // debug
    if x >= 50 {
      break
    }

  }
  return nil
}

func (a *audiocc) processImages(dir string) {
  a.Images = filesByExtension(dir, imageExts)
  // TODO: iterate to find best image, then optimize as 'folder.jpg'
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
