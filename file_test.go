package main

import (
  "os"
  "strings"
  "testing"
  "io/ioutil"
  "path/filepath"
)

func createTestFiles(paths map[string]string, t *testing.T) string {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }

  for p, c := range paths {
    pa := strings.Split(p, "/")
    if len(pa) == 0 {
      continue
    }

    path := filepath.Join(td, filepath.Join(pa[:len(pa)-1]...))

    // create parent dirs
    if len(pa) > 1 {
      err := os.MkdirAll(path, 0777)
      if err != nil {
        t.Fatal(err)
      }
    }

    // create file
    if len(pa[len(pa)-1]) > 0 {
      fullpath := filepath.Join(path, pa[len(pa)-1])
      err := ioutil.WriteFile(fullpath, []byte(c), 0644)
      if err != nil {
        t.Fatal(err)
      }
    }
  }

  return td
}

func TestPathInfo(t *testing.T) {
  tests := [][][]string{
    {
      { "dir1", "dir2/file1.ext" },
      { "dir1/dir2/file1.ext", "dir1/dir2/", "file1" },
    },
  }

  for x := range tests {
    p, dir, file := pathInfo(tests[x][0][0], tests[x][0][1])
    compare := []string{ p, dir, file }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

func TestFilesByExtensionImages(t *testing.T) {
  testFiles := map[string]string{
    "file1": "",
    "file2.jpeg": "",
    "dir1/file3.JPG": "",
    "dir1/dir2/file4.png": "",
  }

  result := []string{
    "dir1/dir2/file4.png",
    "dir1/file3.JPG",
    "file2.jpeg",
  }

  dir := createTestFiles(testFiles, t)
  defer os.RemoveAll(dir)

  paths := filesByExtension(dir, imageExts)
  if strings.Join(paths, "\n") != strings.Join(result, "\n") {
    t.Errorf("Expected %v, got %v", result, paths)
  }
}

func TestFilesByExtensionAudio(t *testing.T) {
  testFiles := map[string]string{
    "not audio file": "",
    "file1.FLAC": "",
    "file2.m4a": "",
    "dir1/file3.mp3": "",
    "dir1/dir2/file4.mp4": "",
    "dir1/dir2/file5.SHN": "",
    "dir1/dir2/file6.WAV": "",
  }

  result := []string{
    "dir1/dir2/file4.mp4",
    "dir1/dir2/file5.SHN",
    "dir1/dir2/file6.WAV",
    "dir1/file3.mp3",
    "file1.FLAC",
    "file2.m4a",
  }

  dir := createTestFiles(testFiles, t)
  defer os.RemoveAll(dir)

  paths := filesByExtension(dir, audioExts)
  if strings.Join(paths, "\n") != strings.Join(result, "\n") {
    t.Errorf("Expected %v, got %v", result, paths)
  }
}
