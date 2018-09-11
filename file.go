package main

import (
  "os"
  "fmt"
  "strings"
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

// group sorted files by common directory
func bundleFiles(dir string, files []string, f func(bundle []int) error) error {
  var dirCur string
  bundle := []int{}

  // need final dir change
  files = append(files, "")

  for x := range files {
    d := filepath.Dir(filepath.Join(dir, files[x]))

    if dirCur == "" {
      dirCur = d
    }

    // if dir changes or last of all files
    if d != dirCur || x == len(files)-1 {
      err := f(bundle)
      if err != nil {
        return err
      }

      bundle = []int{}
      dirCur = d
    }

    bundle = append(bundle, x)
  }

  return nil
}
