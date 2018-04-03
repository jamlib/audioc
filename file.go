package main

import (
  "os"
  "sort"
  "strings"
  "path/filepath"
)

var imageExts = []string{ "jpeg", "jpg", "png" }
var audioExts = []string{ "flac", "m4a", "mp3", "mp4", "shn", "wav" }

func pathInfo(base, path string) (string, string, string) {
  p := filepath.Join(base, path)
  dir, file := filepath.Split(p)
  file = strings.TrimRight(file, filepath.Ext(file))
  return p, dir, file
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
  sort.Strings(files)

  return files
}
