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

  pi.Ext = strings.ToLower(filepath.Ext(pi.File))
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

// index of smallest/largest file in slice of files
func nthFileSize(files []string, smallest bool) (string, error) {
  sizes := []int64{}

  found := -1
  for i := range files {
    in, err := os.Open(files[i])
    if err != nil {
      return "", err
    }
    defer in.Close()

    info, err := in.Stat()
    if err != nil {
      return "", err
    }

    sizes = append(sizes, info.Size())
    if found == -1 || (smallest && info.Size() < sizes[found]) ||
      (!smallest && info.Size() > sizes[found]) {
      found = i
    }
  }

  if found == -1 {
    return "", fmt.Errorf("File not found")
  }
  return files[found], nil
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
    fn, err := nthFileSize([]string{ file, newFile }, false)
    if err != nil || fn == newFile {
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
