package main

import (
  "os"
  "fmt"
  "log"
  "image"
  "regexp"
  "strings"
  "strconv"
  "runtime"
  "io/ioutil"
  "path/filepath"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
  "github.com/JamTools/goff/fsutil"
  "github.com/JamTools/audiocc/albumart"
  "github.com/JamTools/audiocc/metadata"
)

type audiocc struct {
  DirEntry string
  Image string
  Ffmpeg ffmpeg.Ffmpeger
  Ffprobe ffprobe.Ffprober
  Files []string
  Workers int
  Workdir string
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

  // process files using multiple cores
  var workers = runtime.NumCPU()
  //workers = 1 // DEBUG in single-threaded mode

  a := &audiocc{ Ffmpeg: ffm, Ffprobe: ffp, Workers: workers,
    DirEntry: filepath.Clean(args[0]) }

  err = a.process()
  if err != nil {
    log.Fatal(err)
  }
}

// true if --collection & artist path contains " - "
func skipArtistOnCollection(p string) bool {
  if flags.Collection {
    pa := strings.Split(p, fsutil.PathSep)
    // first folder is artist
    if strings.Index(pa[0], " - ") != -1 {
      return true
    }
  }
  return false
}

// skip if album folder name contains year (--fast)
func skipFast(p string) bool {
  if flags.Fast {
    pa := strings.Split(p, fsutil.PathSep)
    // last folder is album
    t := metadata.NewInfo(pa[len(pa)-1], "")
    if len(t.Year) > 0 && len(t.Album) > 0 {
      return true
    }
  }
  return false
}

// combines skipFast & skipArtistOnCollection
func skipFolder(p string) bool {
  if skipArtistOnCollection(p) || skipFast(p) {
    return true
  }
  return false
}

// process album art once per folder of files
func (a *audiocc) processArtwork(file string) error {
  art := &albumart.AlbumArt{ Ffmpeg: a.Ffmpeg, Ffprobe: a.Ffprobe,
    ImgDecode: image.DecodeConfig, WithParentDir: true,
    Fullpath: filepath.Join(a.DirEntry, file) }

  if flags.Write {
    var err error
    a.Image, err = albumart.Process(art)
    if err != nil {
      return err
    }
  }

  return nil
}

func (a *audiocc) process() error {
  if !flags.Write {
    fmt.Printf("\n* To write changes to disk, please provide flag: --write\n")
  }

  // ensure path is is valid directory
  fi, err := os.Stat(a.DirEntry)
  if err != nil || !fi.IsDir() {
    return fmt.Errorf("Invalid directory: %s", a.DirEntry)
  }

  // obtain audio file list
  a.Files = fsutil.FilesAudio(a.DirEntry)

  // group files by parent directory
  err = fsutil.BundleFiles(a.DirEntry, a.Files, func(indexes []int) error {
    file := a.Files[indexes[0]]
    fullDir := filepath.Dir(filepath.Join(a.DirEntry, file))

    // skip if possible
    if skipFolder(file) {
      return nil
    }

    fmt.Printf("\nProcessing: %v ...\n", fullDir)

    // process artwork once per folder
    err = a.processArtwork(file)
    if err != nil {
      return err
    }

    // create new random workdir within current path
    a.Workdir, err = ioutil.TempDir(fullDir, "")
    if err != nil {
      return err
    }
    defer os.RemoveAll(a.Workdir)

    // process folder via threads returning the resulting dir
    dir, err := a.processThreaded(indexes)
    if err != nil {
      return err
    }

    // if not same dir, rename directory to target dir
    if fullDir != dir {
      _, err = fsutil.MergeFolder(fullDir, dir, mergeFolderFunc)
      if err != nil {
        return err
      }
    }

    // remove parent folder if no longer contains audio files
    parentDir := filepath.Dir(fullDir)
    if len(fsutil.FilesAudio(parentDir)) == 0 {
      err = os.RemoveAll(parentDir)
      if err != nil {
        return err
      }
    }

    return nil
  })

  if err != nil {
    return err
  }

  fmt.Printf("\naudiocc finished.\n")
  return nil
}

// passed to fsutil.MergeFolder
func mergeFolderFunc(f string) (int, string) {
  // split filename from path
  _, file := filepath.Split(f)

  // parse disc & track from filename
  i := metadata.NewInfo("", file)

  disc, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(i.Disc))
  track, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(i.Track))

  // combine disc & track into unique integer
  return (disc*1000)+track, i.Title
}

func (a *audiocc) processThreaded(indexes []int) (string, error) {
  var err error
  jobs := make(chan int)
  dir := make(chan string, a.Workers)

  // iterate through files sending them to worker processes
  go func() {
    for x := range indexes {
      if err != nil {
        break
      }
      jobs <- indexes[x]
    }
    close(jobs)
  }()

  // start worker processes
  for i := 0; i < a.Workers; i++ {
    go func() {
      var d string

      for job := range jobs {
        d, err = a.processFile(job)
        if err != nil {
          break
        }
      }

      dir <- d
    }()
  }

  // wait for all workers to finish
  var resultDir string
  for i := 0; i < a.Workers; i++ {
    resultDir = <-dir
  }

  return resultDir, err
}

func (a *audiocc) processFile(index int) (string, error) {
  m := &metadata.Metadata{
    Basepath: a.DirEntry,
    Fullpath: filepath.Join(a.DirEntry, a.Files[index]),
    Ffprobe: a.Ffprobe,
  }

  // if --artist mode, remove innermost dir from basepath so it ends up in infopath
  if flags.Artist != "" {
    m.Artist = flags.Artist
    m.Basepath = filepath.Dir(m.Basepath)
  }

  // if --colleciton mode, artist comes from parent folder name
  if flags.Collection {
    m.Artist = strings.Split(a.Files[index], fsutil.PathSep)[0]
  }

  m, i, err := metadata.New(m)
  if err != nil {
    return "", err
  }

  // skip if sources match (unless --force)
  if m.Match && !flags.Force {
    return m.Fulldir, nil
  }

  // build resulting path
  var path string
  if flags.Collection {
    // build from DirEntry; include artist then year
    path = filepath.Join(a.DirEntry, i.Artist, i.Year)
  } else {
    // remove current dir from fullpath
    path = strings.TrimSuffix(m.Fulldir, m.Infodir)
  }

  // append directory generated from info
  path = filepath.Join(path, i.ToAlbum())

  // print changes to be made
  p := fmt.Sprintf("%v\n", m.Fullpath)
  if !m.Match {
    p += fmt.Sprintf("  * update tags: %#v\n", i)
  }

  // convert audio (if necessary) & update tags
  ext := strings.ToLower(filepath.Ext(m.Fullpath))
  if ext != ".flac" || regexp.MustCompile(` - FLAC$`).FindString(m.Infodir) == "" {
    // convert to mp3 except flac files with " - FLAC" in folder name
    f, err := a.processMp3(m.Fullpath, i)
    if err != nil {
      return "", err
    }

    // add filename to returning path
    _, file := filepath.Split(f)
    newPath := filepath.Join(path, file)

    if m.Fullpath != newPath {
      p += fmt.Sprintf("  * rename to: %v\n", newPath)
    }
  } else {
    // TODO: use metaflac to edit flac metadata & embedd artwork
    p += fmt.Sprintf("\n*** Flac processing with 'metaflac' not yet implemented.\n")
  }

  // print to console all at once
  fmt.Printf(p)

  // path is a directory
  return path, nil
}

func (a *audiocc) processMp3(f string, i *metadata.Info) (string, error) {
  // if already mp3, copy stream; do not convert
  quality := flags.Bitrate
  if strings.ToLower(filepath.Ext(f)) == ".mp3" {
    quality = "copy"
  }

  // TODO: specify lower bitrate if source file is of low bitrate

  // build metadata from tag info
  ffmeta := ffmpeg.Metadata{ Artist: i.Artist, Album: i.ToAlbum(),
    Disc: i.Disc, Track: i.Track, Title: i.Title, Artwork: a.Image }

  // save new file to Workdir subdir within current path
  newFile := filepath.Join(a.Workdir, i.ToFile() + ".mp3")

  // process or convert to mp3
  c := &ffmpeg.Mp3Config{ f, quality, newFile, ffmeta, flags.Fix }
  _, err := a.Ffmpeg.ToMp3(c)
  if err != nil {
    return newFile, err
  }

  // ensure output file was written
  fi, err := os.Stat(newFile)
  if err != nil {
    return newFile, err
  }

  // if flagsWrite & resulting file has size
  // TODO: ensure resulting file is good by reading & comparing metadata
  if fi.Size() > 0 {
    file := filepath.Join(filepath.Dir(f), i.ToFile() + ".mp3")

    if flags.Write {
      // delete original
      err = os.Remove(f)
      if err != nil {
        return file, err
      }

      // move new to original directory
      err = os.Rename(newFile, file)
      if err != nil {
        return file, err
      }
    }

    return file, nil
  }

  return newFile, fmt.Errorf("File didn't have size")
}
