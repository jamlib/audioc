package main

import (
  "io"
  "os"
  "fmt"
  "sort"
  "regexp"
  "strings"
  "strconv"
  "path/filepath"
)

const sep = string(os.PathSeparator)
var imageExts = []string{ "jpeg", "jpg", "png" }
var audioExts = []string{ "flac", "m4a", "mp3", "mp4", "shn", "wav" }

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
  pi.Dir = strings.TrimPrefix(pi.Dir, sep)
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
  path = strings.TrimSuffix(path, sep)
  return path
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
      // remove base directory
      p = p[len(dir):]
      // remove prefixed path separator
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
    destAudios := filesByExtension(dest, audioExts)
    lookup := make(map[int]string, len(destAudios))
    for _, destFile := range destAudios {
      index, title := infoFromAudio(destFile)
      lookup[index] = title
    }

    // copy only src audio files that don't already exist
    copied := false
    for _, srcFile := range filesByExtension(src, audioExts) {
      index, title := infoFromAudio(srcFile)
      if _, found := lookup[index]; !found {
        srcPath := filepath.Join(src, srcFile)

        // if not found, copy audio file
        _, f := filepath.Split(srcFile)
        err = copyFile(srcPath, filepath.Join(dest, f))
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
      for _, imgFile := range filesByExtension(src, imageExts) {
        _, img := filepath.Split(imgFile)
        _ = copyFile(imgFile, filepath.Join(dest, img))
      }
    }

    // if remaining audio files, rename to folder (x)
    if len(filesByExtension(src, audioExts)) > 0 {
      return renameFolder(src, dest)
    }

    // else delete folder
    err = os.RemoveAll(src)
    return dest, err
  }

  // folder doesn't exist
  return renameFolder(src, dest)
}
