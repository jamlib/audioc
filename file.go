package main

import (
  "os"
  "fmt"
  "regexp"
  "strings"
  "strconv"
  "path/filepath"

  "github.com/JamTools/goff/fsutil"
)

type pathInfo struct {
  Fullpath, Fulldir string
  Dir, File, Ext string
}

// split full path into pieces used to determine info
func getPathInfo(basePath, filePath string) *pathInfo {
  basePath = filepath.Clean(basePath)
  pi := &pathInfo{Fullpath: filepath.Join(basePath, filePath) }

  pi.Fulldir, pi.File = filepath.Split(pi.Fullpath)
  pi.Fulldir = filepath.Clean(pi.Fulldir)
  pi.Ext = filepath.Ext(pi.File)

  pi.File = strings.TrimSuffix(pi.File, pi.Ext)
  pi.Ext = strings.ToLower(pi.Ext)

  // if --artist mode, remove inner-most dir from basePath
  // so it can be used as a source of info
  if flags.Artist != "" {
    basePath = filepath.Dir(basePath)
  }

  pi.Dir = strings.TrimPrefix(pi.Fulldir, basePath)
  pi.Dir = strings.TrimPrefix(pi.Dir, fsutil.PathSep)
  if pi.Dir == "" {
    // use inner-most dir of full path
    pi.Dir = filepath.Base(pi.Fulldir)
  }

  return pi
}

func checkDir(dir string) (string, error) {
  dir = filepath.Clean(dir)
  fi, err := os.Stat(dir)
  if err != nil {
    return dir, err
  }
  if !fi.IsDir() {
    return dir, fmt.Errorf("Not a directory")
  }
  return dir, nil
}

// separate dir from fullpath
func onlyDir(path string) string {
  path, _ = filepath.Split(path)
  path = strings.TrimSuffix(path, fsutil.PathSep)
  return path
}

// group sorted files by common directory
func bundleFiles(dir string, files []string, f func(bundle []int) error) error {
  dirCur := ""
  bundle := []int{}
  files = append(files, "")

  for x := range files {
    pi := getPathInfo(dir, files[x])

    if dirCur == "" {
      dirCur = string(pi.Dir)
    }

    // if dir changes or last of all files
    if pi.Dir != dirCur || x == len(files)-1 {
      err := f(bundle)
      if err != nil {
        return err
      }

      bundle = []int{}
      dirCur = string(pi.Dir)
    }

    bundle = append(bundle, x)
  }

  return nil
}

// strip out characters from filename
func safeFilename(f string) string {
  return regexp.MustCompile(`[^A-Za-z0-9-'!?& _()]+`).ReplaceAllString(f, "")
}

// if dir already exists, prepend (x) to folder name
// increment (x) until dir not found
func renameFolder(src, dest string) (string, error) {
  _, err := os.Stat(dest)
  if err == nil {
    x := 1
    found := true
    for found {
      newDir := fmt.Sprintf("%v (%v)", dest, x)

      _, err := os.Stat(newDir)
      if err != nil {
        dest = newDir
        found = false
      }

      x += 1
    }
  }

  // trim off last path element and create full path
  err = os.MkdirAll(filepath.Dir(dest), 0777)
  if err != nil {
    return dest, err
  }

  err = os.Rename(src, dest)
  return dest, err
}

// if dest folder already exists, merge audio file if Disc/Track not already present
// within. merge all images currently not present
// TODO: tests
func mergeFolder(src, dest string) (string, error) {
  // return disc*1000+track as int & title for each audio file
  infoFromAudio := func(file string) (int, string) {
    // split filename from path
    _, f := filepath.Split(file)
    i := &info{}
    i.fromFile(f)

    disc, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(i.Disc))
    track, _ := strconv.Atoi(regexp.MustCompile(`^\d+`).FindString(i.Track))

    return (disc*1000)+track, i.Title
  }

  // if folder already exists
  _, err := os.Stat(dest)
  if err == nil {
    // build dest audio file info maps
    destAudios := fsutil.FilesAudio(dest)
    lookup := make(map[int]string, len(destAudios))
    for _, destFile := range destAudios {
      index, title := infoFromAudio(destFile)
      lookup[index] = title
    }

    // copy only src audio files that don't already exist
    copied := false
    for _, srcFile := range fsutil.FilesAudio(src) {
      index, title := infoFromAudio(srcFile)
      if _, found := lookup[index]; !found {
        srcPath := filepath.Join(src, srcFile)

        // if not found, copy audio file
        _, f := filepath.Split(srcFile)
        err = fsutil.CopyFile(srcPath, filepath.Join(dest, f))
        if err != nil {
          return dest, err
        }

        // add to lookup, ensure copied is true
        lookup[index] = title
        copied = true

        // remove source audio file
        err = os.Remove(srcPath)
        if err != nil {
          return dest, err
        }
      }
    }

    // copy all image files (if copied at least one audio file)
    if copied {
      for _, imgFile := range fsutil.FilesImage(src) {
        _, img := filepath.Split(imgFile)
        _ = fsutil.CopyFile(imgFile, filepath.Join(dest, img))
      }
    }

    // if remaining audio files, rename to folder (x)
    if len(fsutil.FilesAudio(src)) > 0 {
      return renameFolder(src, dest)
    }

    // else delete folder
    err = os.RemoveAll(src)
    return dest, err
  }

  // folder doesn't exist
  return renameFolder(src, dest)
}
