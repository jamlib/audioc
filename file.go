package main

import (
  "io"
  "os"
  "fmt"
  "sort"
  "regexp"
  "strings"
  "path/filepath"
)

var imageExts = []string{ "jpeg", "jpg", "png" }
var audioExts = []string{ "flac", "m4a", "mp3", "mp4", "shn", "wav" }

type pathInfo struct {
  Fullpath, Fulldir string
  Dir, File, Ext string
}

func getPathInfo(base, path string) *pathInfo {
  pi := &pathInfo{Fullpath: filepath.Join(base, path) }

  pi.Fulldir, pi.File = filepath.Split(pi.Fullpath)
  pi.Fulldir = filepath.Clean(pi.Fulldir)

  pi.Dir = strings.TrimPrefix(pi.Fulldir, base)
  if pi.Dir == "" {
    pi.Dir = filepath.Base(pi.Fulldir)
  }
  pi.Dir = strings.TrimPrefix(pi.Dir, string(os.PathSeparator))

  pi.Ext = filepath.Ext(pi.File)
  pi.File = strings.TrimSuffix(pi.File, pi.Ext)

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

func filesByExtension(dir string, exts []string) []string {
  files := []string{}

  // closure to pass to filepath.Walk
  walkFunc := func(p string, f os.FileInfo, err error) error {
    ext := filepath.Ext(p)
    if len(ext) == 0 {
      return nil
    }
    ext = strings.ToLower(ext[1:])

    x := sort.SearchStrings(exts, ext)
    if x < len(exts) && exts[x] == ext {
      p = p[len(dir):]
      if p[0] == os.PathSeparator {
        p = p[1:]
      }
      files = append(files, p)
    }

    return err
  }

  err := filepath.Walk(dir, walkFunc)
  if err != nil {
    return []string{}
  }
  // must sort: nested directories' files list first
  // char / sorts before A-Za-z0-9
  sort.Strings(files)

  return files
}

// strip out characters from filename
func safeFilename(f string) string {
  // replace / or \ with _
  return regexp.MustCompile(`[\/\\]+`).ReplaceAllString(f, "_")
}

// index of smallest/largest file in slice of files
func nthFileSize(files []string, smallest bool) (int, error) {
  sizes := []int64{}

  found := -1
  for i := range files {
    in, err := os.Open(files[i])
    if err != nil {
      return -1, err
    }
    defer in.Close()

    info, err := in.Stat()
    if err != nil {
      return -1, err
    }

    sizes = append(sizes, info.Size())
    if found == -1 || (smallest && info.Size() < sizes[found]) ||
      (!smallest && info.Size() > sizes[found]) {
      found = i
    }
  }

  return found, nil
}

// true if destination does not exist or src has larger file size
func isLarger(file, newFile string) bool {
  f, err := os.Open(file)
  defer f.Close()
  if err != nil {
    return false
  }

  f2, err := os.Open(newFile)
  defer f2.Close()

  if err == nil {
    i, _ := nthFileSize([]string{ file, newFile }, false)
    if i == 1 {
      return false
    }
  }

  return true
}

func copyFile(srcPath, destPath string) (err error) {
  srcFile, err := os.Open(srcPath)
  if err != nil {
    return
  }
  defer srcFile.Close()

  destFile, err := os.Create(destPath)
  if err != nil {
    return
  }
  defer destFile.Close()

  _, err = io.Copy(destFile, srcFile)
  if err != nil {
    return
  }

  err = destFile.Sync()
  return
}
