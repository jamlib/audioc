package main

import (
  "os"
  "log"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/audioc"
)

func main() {
  c, cont := configFromFlags()
  if !cont {
    os.Exit(0)
  }

  ffm, err := ffmpeg.New()
  if err != nil {
    log.Fatal(err)
  }

  ffp, err := ffprobe.New()
  if err != nil {
    log.Fatal(err)
  }

  // audioc.New & a.Process found within ../audioc.go
  a := audioc.New(c, ffm, ffp)

  err = a.Process()
  if err != nil {
    log.Fatal(err)
  }
}
