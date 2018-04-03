package main

import (
  "os"
  "fmt"

  "github.com/JamTools/goff/ffprobe"
)

func main() {
  args, cont := processFlags()
  if !cont {
    os.Exit(1)
  }

  fmt.Printf("\nFlag collection: %v\n", flagCollection)
  if flagArtist != "" {
    fmt.Printf("Flag artist: %v\n\n", flagArtist)
  }

  dirEntry := args[0]
  fi, err := os.Stat(dirEntry)
  if err != nil || !fi.IsDir() {
    fmt.Printf("Error: not a directory.\n%v\n\n", dirEntry)
    os.Exit(1)
  }

  process(dirEntry)
}

func process(dirEntry string) {
  fmt.Printf("Path:\n%v\n\n", dirEntry)

  audio := filesByExtension(dirEntry, audioExts)
  for x := range audio {
    p, dir, file := pathInfo(dirEntry, audio[x])

    images := filesByExtension(dir, imageExts)
    fmt.Printf("Images:\n%v\n\n", images)

    fmt.Printf("File: %v\n", file)

    i, r := infoFromFile(file)
    fmt.Printf("Date: %s-%s-%s\n", i.Year, i.Month, i.Day)
    fmt.Printf("Disc/Track: %s/%s\n", i.Disc, i.Track)
    fmt.Printf("Remain: %v\n\n", r)

    fmt.Printf("Path[]:\n")
    infoFromPath(dir, string(os.PathSeparator))

    fmt.Println()
    probeData(p)

    break // debug
  }
}

func probeData(p string) {
  d, err := ffprobe.GetData(p)
  if err != nil {
    fmt.Errorf("%v", err)
  }

  for i := range d.Streams {
    fmt.Printf("Stream %v: %#v\n", i, d.Streams[i])
  }

  fmt.Printf("Format: %#v\n", d.Format)
  fmt.Printf("Tags: %#v\n\n", d.Format.Tags)
}
