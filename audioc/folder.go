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

  // process folder via threads returning the resulting dir
  // calls a.processFile() for each index
  dir, err := a.processThreaded(indexes)
  if err != nil {
    return err
  }

  // return here unless writing
  if !a.Flags.Write {
    return nil
  }

  // explicitly remove workdir (before folder is renamed)
  os.RemoveAll(a.Workdir)

  // if not same dir, rename directory to target dir
  if fullDir != dir {
    _, err = fsutil.MergeFolder(fullDir, dir, mergeFolderFunc)
    if err != nil {
      return err
    }
  }

  // remove parent folder if no longer contains audio files
  // TODO add check to ensure not removing any of DirEntry
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

  var err error
  a.Image, err = albumart.Process(art)
  return err
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