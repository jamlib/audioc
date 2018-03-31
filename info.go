// info can be from:
// id3 tag, path, filename

package main

import (
  "os"
  "fmt"
  "sort"
  "strings"
  "path/filepath"
)

func infoFromPath(p string) {
  dir, file := filepath.Split(p)

  fmt.Printf("File: %v, Ext: %v\n\n", file, filepath.Ext(file))

  fmt.Printf("Images:\n")
  fmt.Printf("%v\n\n", filesByExtension(dir, imageExts))

  fmt.Printf("Path[]:\n")
  pathArray := strings.Split(dir, string(os.PathSeparator))
  for i := range reverse(pathArray) {
    if len(pathArray[i]) > 0 {
      fmt.Printf("%v\n", pathArray[i])
    }
  }
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
      files = append(files, p)
    }

    return err
  }

  err := filepath.Walk(dir, walkFunc)
  if err != nil {
    return []string{}
  }

  return files
}
