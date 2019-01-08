package audioc

import (
  "os"
  "fmt"
  "image"
  "regexp"
  "strings"
  "strconv"
  "runtime"
  "io/ioutil"
  "path/filepath"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/libaudio/fsutil"
  "github.com/jamlib/audioc/albumart"
  "github.com/jamlib/audioc/metadata"
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
  Artist, Bitrate string
  Collection, Fix, Force, Version, Write bool
}

func New(c *Config, ffm ffmpeg.Ffmpeger, ffp ffprobe.Ffprober) *audioc {
  return &audioc{ DirEntry: c.DirEntry, Flags: c.Flags,
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

  // group files by parent directory
  err = fsutil.BundleFiles(a.DirEntry, a.Files, a.processFolder)
  if err != nil {
    return err
  }

  fmt.Printf("\naudioc finished.\n")
  return nil
}

func (a *audioc) skipFolder(base, path string) bool {
  var alb string
  m := metadata.New(base, path)
  pa := strings.Split(m.Infodir, fsutil.PathSep)

  if a.Flags.Collection {
    // true if --collection & artist path contains " - "
    if strings.Index(pa[0], " - ") != -1 {
      return true
    }
    if len(pa) > 2 {
      // Artist / Year / Album
      alb = pa[2]
    }
  } else {
    // if --artist, album should be innermost dir
    alb = pa[len(pa)-1]
  }

  // true if album folder matches metadata.ToAlbum
  if alb != "" {
    i := &metadata.Info{}
    i.FromPath(alb)

    if i.ToAlbum() == alb {
      return true
    }
  }

  return false
}

// process album art once per folder of files
func (a *audioc) processArtwork(file string) error {
  art := &albumart.AlbumArt{ Ffmpeg: a.Ffmpeg, Ffprobe: a.Ffprobe,
    ImgDecode: image.DecodeConfig, WithParentDir: true,
    Fullpath: filepath.Join(a.DirEntry, file) }

  if a.Flags.Write {
    var err error
    a.Image, err = albumart.Process(art)
    if err != nil {
      return err
    }
  }

  return nil
}

func (a *audioc) processFolder(indexes []int) error {
  fullDir := filepath.Dir(filepath.Join(a.DirEntry, a.Files[indexes[0]]))

  // skip if possible (unless --force)
  if !a.Flags.Force && a.skipFolder(a.DirEntry, a.Files[indexes[0]]) {
    return nil
  }

  fmt.Printf("\nProcessing: %v ...\n", fullDir)

  // process artwork once per folder
  err := a.processArtwork(a.Files[indexes[0]])
  if err != nil {
    return err
  }

  // create new random workdir within current path
  a.Workdir, err = ioutil.TempDir(fullDir, "")
  if err != nil {
    return err
  }

  // process folder via threads returning the resulting dir
  dir, err := a.processThreaded(indexes)
  if err != nil {
    return err
  }

  // remove workdir here else forder is renamed and workdir becomes invalid
  os.RemoveAll(a.Workdir)

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
}

// passed to fsutil.MergeFolder
func mergeFolderFunc(f string) (int, string) {
  // split filename from path
  _, file := filepath.Split(f)

  // parse disc & track from filename
  i := &metadata.Info{}
  i.FromFile(file)

  disc, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(i.Disc))
  track, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(i.Track))

  // combine disc & track into unique integer
  return (disc*1000)+track, i.Title
}

func (a *audioc) processThreaded(indexes []int) (string, error) {
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
        var e error
        d, e = a.processFile(job)
        if e != nil {
          err = e
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

func (a *audioc) processFile(index int) (string, error) {
  m := metadata.New(a.DirEntry, a.Files[index])

  // if --artist mode, remove innermost dir from basepath so it ends up in infodir
  if a.Flags.Artist != "" {
    m.Artist = a.Flags.Artist
    m.Basepath = filepath.Dir(m.Basepath)
  }

  // if --colleciton mode, artist comes from parent folder name
  if a.Flags.Collection {
    m.Artist = strings.Split(a.Files[index], fsutil.PathSep)[0]
  }

  m, i, err := m.NewInfo(a.Ffprobe)
  if err != nil {
    return "", err
  }

  // skip if sources match (unless --force)
  if m.Match && !a.Flags.Force {
    return m.Fulldir, nil
  }

  // build resulting path
  var path string
  if a.Flags.Collection {
    // build from DirEntry; include artist then year
    path = filepath.Join(a.DirEntry, i.Artist, i.Year)
  } else {
    // remove current dir from fullpath
    path = strings.TrimSuffix(m.Fulldir, m.Infodir)
  }

  // append directory generated from info
  path = filepath.Join(path, i.ToAlbum())

  // print changes to be made
  p := fmt.Sprintf("\n%v\n", m.Fullpath)
  if !m.Match {
    p += fmt.Sprintf("  * update tags: %#v\n", i)
  }

  // convert audio (if necessary) & update tags
  ext := strings.ToLower(filepath.Ext(m.Fullpath))
  if ext != ".flac" || regexp.MustCompile(` - FLAC$`).FindString(m.Infodir) == "" {
    // skip converting if folder contains ' - FLAC'

    _, err := a.processMp3(m.Fullpath, i)
    if err != nil {
      return "", err
    }

    // compare processed to current path
    newPath := filepath.Join(path, i.ToFile() + ".mp3")
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

func (a *audioc) processMp3(f string, i *metadata.Info) (string, error) {
  // skip if not writing
  if !a.Flags.Write {
    return "", nil
  }

  // if already mp3, copy stream; do not convert
  quality := a.Flags.Bitrate
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
  c := &ffmpeg.Mp3Config{ f, quality, newFile, ffmeta, a.Flags.Fix }
  _, err := a.Ffmpeg.ToMp3(c)
  if err != nil {
    return newFile, err
  }

  // ensure output file was written
  fi, err := os.Stat(newFile)
  if err != nil {
    return newFile, err
  }

  // ensure resulting file has size
  // TODO: ensure resulting file is good by reading & comparing metadata
  if fi.Size() <= 0 {
    return newFile, fmt.Errorf("File didn't have size")
  }

  file := filepath.Join(filepath.Dir(f), i.ToFile() + ".mp3")

  // delete original
  err = os.Remove(f)
  if err != nil {
    return file, err
  }

  // move new to original directory
  err = os.Rename(newFile, file)
  return file, err
}
