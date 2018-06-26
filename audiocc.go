package main

import (
  "os"
  "fmt"
  "log"
  "image"
  "strings"
  "runtime"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
)

type audiocc struct {
  DirCur, DirEntry string
  Image string
  Ffmpeg ffmpeger
  Ffprobe ffprober
  Files []string
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

  ffm, err := ffmpeg.New()
  if err != nil {
    log.Fatal(err)
  }

  ffp, err := ffprobe.New()
  if err != nil {
    log.Fatal(err)
  }

  a := &audiocc{ Ffmpeg: ffm, Ffprobe: ffp, DirEntry: args[0] }
  err = a.process()
  if err != nil {
    log.Fatal(err)
  }
}

func (a *audiocc) process() error {
  if !flagWrite {
    fmt.Printf("\n* To write changes to disk, please provide flag: --write\n")
  }

  // ensure path is is valid directory
  _, err := checkDir(a.DirEntry)
  if err != nil {
    return err
  }

  // obtain audio file list
  a.Files = filesByExtension(a.DirEntry, audioExts)

  fmt.Printf("\nPath: %v\n\n", a.DirEntry)

  bundle := []int{}

  for x := range a.Files {
    pi := getPathInfo(a.DirEntry, a.Files[x])

    // when to skip ahead
    if skipArtistOnCollection(pi.Dir) {
      continue
    }

    if a.DirCur == "" {
      a.DirCur = string(pi.Dir)
    }

    if pi.Dir != a.DirCur || x >= len(a.Files) - 1 {
      err := a.processBundle(bundle)
      if err != nil {
        return err
      }

      bundle = []int{}
      a.DirCur = string(pi.Dir)
    }

    bundle = append(bundle, x)
  }

  fmt.Printf("audiocc finished.\n")
  return nil
}

func (a *audiocc) processBundle(indexes []int) error {
  // process album art
  art := &artwork{ Ffmpeg: a.Ffmpeg, Ffprobe: a.Ffprobe,
    ImgDecode: image.DecodeConfig,
    PathInfo: getPathInfo(a.DirEntry, a.Files[indexes[0]]) }

  if flagWrite {
    var err error
    a.Image, err = art.process()
    if err != nil {
      return err
    }
  }

  // determine to skip processing if first file looks good (and not --force)
  skip, err := a.processIndex(indexes[0])
  if skip {
    return nil
  }
  if err != nil {
    return err
  }

  // remove this index
  indexes = append(indexes[:0], indexes[1:]...)

  // process files using multiple cores
  var workers = runtime.NumCPU()

  jobs := make(chan int)
  done := make(chan bool, workers)

  // iterate through files sending them to worker processes
  go func() {
    for x := range indexes {
      jobs <- indexes[x]
    }
    close(jobs)
  }()

  // start worker processes
  for i := 0; i < workers; i++ {
    go func() {
      for job := range jobs {
        _, err := a.processIndex(job)
        if err != nil {
          fmt.Printf("Error: %s\n\n", err.Error())
        }
      }

      // when jobs channel is closed
      done <- true
    }()
  }

  // wait for all workers to finish
  for i := 0; i < workers; i++ {
    <-done
  }

  return nil
}

func (a *audiocc) processIndex(index int) (bool, error) {
  pi := getPathInfo(a.DirEntry, a.Files[index])

  // info from path & filename
  i := &info{}
  i.fromFile(pi.File)
  i.fromPath(pi.Dir, string(os.PathSeparator))

  // info from embedded tags within audio file
  d, err := a.Ffprobe.GetData(pi.Fullpath)
  if err != nil {
    return false, err
  }

  // skip if sources match (unless --force)
  if i.matchProbe(d.Format.Tags) && !flagForce {
    return true, nil
  }

  fmt.Printf("File: %v\n", a.Files[index])
  fmt.Printf("Info: %#v\n", i)
  fmt.Printf("Tags: %#v\n\n", d.Format.Tags)

  // build info from probe tags
  tagInfo := &info{
    Disc: d.Format.Tags.Disc,
    Track: d.Format.Tags.Track,
    Title: matchAlbumOrTitle(d.Format.Tags.Title),
  }
  tagInfo.fromAlbum(d.Format.Tags.Album)

  if *i != *tagInfo {
    fmt.Printf("*** Info Diff: %v, %v\n\n", i, tagInfo)
  }

  // TODO: convert audio & update tags (if necessary)
  // TODO: rename file (if necessary)

  if a.lastOfCurrentDir(index) {
    // TODO: move files to updated path (if necessary)
  }

  return false, nil
}

// compares paths with DirCur to determine if all files have been processed
func (a *audiocc) lastOfCurrentDir(i int) bool {
  if i <= len(a.Files)-1 {
    last := true
    if i < len(a.Files)-1 {
      ni := getPathInfo(a.DirEntry, a.Files[i+1])
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
