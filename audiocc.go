package main

import (
  "os"
  "fmt"

  "github.com/JamTools/goff/ffprobe"
)

var imageExts = []string{ "jpeg", "jpg", "png" }
var audioExts = []string{ "flac", "m4a", "mp3", "mp4", "shn", "wav" }

func main() {
  args, cont := processFlags()
  if !cont {
    os.Exit(1)
  }

  fmt.Printf("\nFlag collection: %v\n", flagCollection)
  if flagArtist != "" {
    fmt.Printf("Flag artist: %v\n\n", flagArtist)
  }

  dir := args[0]
  fi, err := os.Stat(dir)
  if err != nil || !fi.IsDir() {
    fmt.Printf("Error: not a directory.\n%v\n\n", dir)
    os.Exit(1)
  }

  fmt.Printf("Path:\n%v\n\n", dir)

  audio := filesByExtension(dir, audioExts)
  for i := range audio {
    infoFromPath(audio[i])
    fmt.Println()
    probeData(audio[i])
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
