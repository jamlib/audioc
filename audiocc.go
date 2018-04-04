package main

import (
  "os"
  "fmt"
  "strings"

  "github.com/JamTools/goff/ffprobe"
)

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
  process(dir)
}

func process(dirEntry string) {
  currentDir := ""
  images := []string{}

  audio := filesByExtension(dirEntry, audioExts)
  for x := range audio {
    pi := getPathInfo(dirEntry, audio[x])

    if skipArtistOnCollection(pi.Dir) {
      continue
    }

    if pi.Dir != currentDir {
      // only need to process images once
      images = filesByExtension(pi.Fulldir, imageExts)
      currentDir = string(pi.Dir)
    }

    i, _ := infoFromFile(pi.File)
    i2 := infoFromPath(pi.Dir, string(os.PathSeparator))
    fmt.Printf("File: %#v\nPath: %#v\n", i, i2)

    d, err := ffprobe.GetData(pi.Fullpath)
    if err != nil {
      fmt.Errorf("%v\n\n", err)
      return
    }
    fmt.Printf("Tags: %#v\n\n", d.Format.Tags)

    // wait to move files until all files within path have been processed
    if x <= len(audio)-1 {
      save := true
      if x < len(audio)-1 {
        ni := getPathInfo(dirEntry, audio[x+1])
        if ni.Dir == currentDir {
          save = false
        }
      }
      if save {
        // TODO: move files to updated path
        fmt.Printf("Images:\n%v\n\n", images)
        images = []string{}
      }
    }

    // debug
    if x >= 50 {
      break
    }

  }
}

func skipArtistOnCollection(p string) bool {
  if flagCollection {
    pa := strings.Split(p, string(os.PathSeparator))
    if strings.Index(pa[0], " - ") != -1 {
      return true
    }
  }
  return false
}
