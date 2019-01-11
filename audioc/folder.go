package audioc

import (
  "os"
  "fmt"
  "image"
  "regexp"
  "strings"
  "strconv"
  "io/ioutil"
  "path/filepath"

  "github.com/jamlib/libaudio/fsutil"
  "github.com/jamlib/audioc/metadata"
  "github.com/jamlib/audioc/albumart"
)

// process each bundle or folder of audio files
func (a *audioc) processBundle(indexes []int) error {
  fullDir := filepath.Dir(filepath.Join(a.DirEntry, a.Files[indexes[0]]))

  // skip if possible (unless --force)
  if !a.Flags.Force && a.skipFolder(a.DirEntry, a.Files[indexes[0]]) {
    return nil
  }

  fmt.Printf("\nProcessing: %v ...\n", fullDir)

  // reset image on each folder
  a.Image = ""

  if a.Flags.Write {
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
  }

  // process folder via threads returning the resulting metadata slice
  // calls a.processFile() for each index
  d, err := a.processThreaded(indexes)
  if err != nil {
    return err
  }

  // return here unless writing
  if !a.Flags.Write {
    return nil
  }

  // explicitly remove workdir (before folder is renamed)
  os.RemoveAll(a.Workdir)

  fullResultD := filepath.Dir(filepath.Join(a.DirEntry, d[0].Resultpath))

  // if not same dir, rename directory to target dir
  if fullDir != fullResultD {
    _, err = fsutil.MergeFolder(fullDir, fullResultD, mergeFolderFunc)
    if err != nil {
      return err
    }
  }

  // remove parent folder if no longer contains audio files
  // TODO fsutil.FilesAudio(parentDir) breaks when parentDir is a symlink
  parentDir := filepath.Dir(fullDir)
  if len(fsutil.FilesAudio(parentDir)) == 0 {
    err = os.RemoveAll(parentDir)
    if err != nil {
      return err
    }
  }

  return nil
}

// helper to determine if bundle should be skipped by analyzing the
// first audio files album folder
func (a *audioc) skipFolder(base, path string) bool {
  pa := strings.Split(path, fsutil.PathSep)

  // determine which folder in path is the album name
  var alb string
  if a.Flags.Collection {
    // true if --collection & artist path contains " - "
    if strings.Index(pa[0], " - ") != -1 {
      return true
    }
    if len(pa) > 3 {
      // Artist / Year / Album / File
      alb = pa[2]
    }
  } else {
    // if --artist, set to innermost dir
    if len(pa) > 1 {
      alb = pa[len(pa)-2]
    }
  }

  // true if album folder matches metadata.ToAlbum
  if len(alb) > 0 {
    m := metadata.New("", alb + fsutil.PathSep + "1.mp3")

    if m.Info.ToAlbum() == alb {
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

  var err error
  a.Image, err = albumart.Process(art)
  return err
}

// passed to fsutil.MergeFolder
func mergeFolderFunc(f string) (int, string) {
  // parse disc & track from filename
  m := metadata.New("", f)

  disc, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(m.Info.Disc))
  track, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(m.Info.Track))

  // combine disc & track into unique integer
  return (disc*1000)+track, m.Info.Title
}
