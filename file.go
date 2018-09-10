package main

import (
  "os"
  "fmt"
  "regexp"
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
