package main

import (
  "io"
  "os"
  "errors"
  "reflect"
  "strings"
  "strconv"
  "testing"
  "io/ioutil"
  "path/filepath"
)

func tmpFile(t *testing.T, input string, f func(in *os.File)) {
  in, err := ioutil.TempFile("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.Remove(in.Name())
  defer in.Close()

  _, err = io.WriteString(in, input)
  if err != nil {
    t.Fatal(err)
  }

  _, _ = in.Seek(0, os.SEEK_SET)

  f(in)
}

type testFile struct {
  name, contents string
}

func createTestFiles(files []*testFile, t *testing.T) string {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }

  for i := range files {
    if len(files[i].name) == 0 {
      continue
    }

    pa := strings.Split(files[i].name, "/")
    p := filepath.Join(td, filepath.Join(pa[:len(pa)-1]...))

    // create parent dirs
    if len(pa) > 1 {
      err := os.MkdirAll(p, 0777)
      if err != nil {
        t.Fatal(err)
      }
    }

    // create file
    if len(pa[len(pa)-1]) > 0 {
      fullpath := filepath.Join(p, pa[len(pa)-1])
      err := ioutil.WriteFile(fullpath, []byte(files[i].contents), 0644)
      if err != nil {
        t.Fatal(err)
      }
    }
  }

  return td
}

func TestPathInfo(t *testing.T) {
  tests := []struct {
    base, path string
    pi *pathInfo
  }{
    { base: "dir1", path: "dir2/dir3/file1.ext",
      pi: &pathInfo{ Fullpath: "dir1/dir2/dir3/file1.ext",
        Fulldir: "dir1/dir2/dir3", Dir: "dir2/dir3", File: "file1", Ext: ".ext" },
    },{
      base: "dir3/dir4", path: "file2.ext",
      pi: &pathInfo{ Fullpath: "dir3/dir4/file2.ext",
        Fulldir: "dir3/dir4", Dir: "dir4", File: "file2", Ext: ".ext" },
    },{
      base: "/dir3/dir4/", path: "file2.ext",
      pi: &pathInfo{ Fullpath: "/dir3/dir4/file2.ext",
        Fulldir: "/dir3/dir4", Dir: "dir4", File: "file2", Ext: ".ext" },
    },
  }

  for x := range tests {
    pi := getPathInfo(tests[x].base, tests[x].path)
    if !reflect.DeepEqual(pi, tests[x].pi) {
      t.Errorf("Expected %v, got %v", tests[x].pi, pi)
    }
  }
}

func TestCheckDirInvalid(t *testing.T) {
  // not exist
  _, err := checkDir("audiocc-path-def-dne")
  if err == nil {
    t.Errorf("Expected error, got nil")
  }

  // not directory
  tmpFile(t, "", func(in *os.File){
    _, err := checkDir(in.Name())
    if err == nil {
      t.Errorf("Expected error, got nil")
    }
  })
}

func TestCheckDir(t *testing.T) {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.RemoveAll(td)

  _, err = checkDir(td)
  if err != nil {
    t.Errorf("Expected nil, got %v", err)
  }
}

func TestOnlyDir(t *testing.T) {
  path := filepath.Join("one", "two", "three.jpg")
  r := onlyDir(path)
  if r != "one/two" {
    t.Errorf("Expected %v, got %v", "one/two", r)
  }
}

func TestFilesByExtensionImages(t *testing.T) {
  files := []*testFile{
    {"file1", ""},
    {"file2.jpeg", ""},
    {"dir1/file3.JPG", ""},
    {"dir1/dir2/file4.png", ""},
  }

  result := []string{
    "dir1/dir2/file4.png",
    "dir1/file3.JPG",
    "file2.jpeg",
  }

  dir := createTestFiles(files, t)
  defer os.RemoveAll(dir)

  paths := filesByExtension(dir, imageExts)
  if strings.Join(paths, "\n") != strings.Join(result, "\n") {
    t.Errorf("Expected %v, got %v", result, paths)
  }
}

func TestFilesByExtensionAudio(t *testing.T) {
  files := []*testFile{
    {"not audio file", ""},
    {"file1.FLAC", ""},
    {"file2.m4a", ""},
    {"dir1/file3.mp3", ""},
    {"dir1/dir2/file4.mp4", ""},
    {"dir1/dir2/file5.SHN", ""},
    {"dir1/dir2/file6.WAV", ""},
  }

  result := []string{
    "dir1/dir2/file4.mp4",
    "dir1/dir2/file5.SHN",
    "dir1/dir2/file6.WAV",
    "dir1/file3.mp3",
    "file1.FLAC",
    "file2.m4a",
  }

  dir := createTestFiles(files, t)
  defer os.RemoveAll(dir)

  paths := filesByExtension(dir, audioExts)
  if strings.Join(paths, "\n") != strings.Join(result, "\n") {
    t.Errorf("Expected %v, got %v", result, paths)
  }
}

func TestBundleFiles(t *testing.T) {
  testFiles := []string{
    "artist1/file1",
    "artist1/file2",
    "artist1/file3",
    "artist2/file1",
    "artist2/file2",
    "artist3/file1",
    "artist4/file1",
  }

  bundles := []string{
    "012",
    "34",
    "5",
    "6",
  }

  results := make([]string, 0)
  _ = bundleFiles("/test", testFiles, func(bundle []int) error {
    var r string
    for i := range bundle {
      r += strconv.Itoa(bundle[i])
    }

    results = append(results, r)

    // TODO: test returning error
    return nil
  })

  err := false
  for x := range bundles {
    if x > len(results)-1 || bundles[x] != results[x] {
      err = true
      break
    }
  }

  if err {
    t.Errorf("Expected %v, got %v", bundles, results)
  }
}

// TODO update this
func TestSafeFilename(t *testing.T) {
  tests := [][]string{
    { "", "" },
  }

  for i := range tests {
    r := safeFilename(tests[i][0])
    if r != tests[i][1] {
      t.Errorf("Expected %v, got %v", tests[i][1], r)
    }
  }
}

func testFilesFullPath(t *testing.T, f func(dir string, files []string)) {
  newFiles := []*testFile{
    {"file1", "abcde"},
    {"file2.jpeg", "a"},
    {"dir1/file3.JPG", "acddfefsefd"},
    {"dir1/dir2/file4.png", "dfadfd"},
  }

  dir := createTestFiles(newFiles, t)
  defer os.RemoveAll(dir)

  files := []string{}
  for i := range newFiles {
    files = append(files, filepath.Join(dir, newFiles[i].name))
  }

  f(dir, files)
}

func TestNthFileSize(t *testing.T) {
  tests := []struct {
    smallest bool
    result string
    other string
  }{
    { smallest: true, result: "file2.jpeg" },
    { smallest: false, result: "dir1/file3.JPG" },
    { other: "audiocc-file-def-dne", result: "" },
  }

  testFilesFullPath(t, func(dir string, files []string) {
    for i := range tests {
      // test file does not exist
      if len(tests[i].other) > 0 {
        files = []string{ tests[i].other }
      }
      r, err := nthFileSize(files, tests[i].smallest)

      // test errors by setting empty result
      if err != nil && tests[i].result == "" {
        continue
      }

      res := filepath.Join(dir, tests[i].result)
      if r != res {
        t.Errorf("Expected %v, got %v", res, r)
      }
    }
  })
}

func TestIsLarger(t *testing.T) {
  testFilesFullPath(t, func(dir string, files []string) {
    tests := []struct {
      src, dest string
      result bool
    }{
      { src: files[0], dest: files[1], result: true },
      { src: files[1], dest: files[0], result: false },
      { src: files[2], dest: files[3], result: true },
      { src: "audiocc-file-def-dne", dest: files[3], result: false },
    }

    for i := range tests {
      r := isLarger(tests[i].src, tests[i].dest)
      if r != tests[i].result {
        t.Errorf("Expected %v, got %v", tests[i].result, r)
      }
    }
  })
}

func TestCopyFile(t *testing.T) {
  testFilesFullPath(t, func(dir string, files []string) {
    // destination dir
    td, err := ioutil.TempDir("", "")
    if err != nil {
      t.Fatal(err)
    }
    defer os.RemoveAll(td)

    tests := []struct {
      src, dest string
      error error
    }{
      { src: files[2], dest: filepath.Join(td, "file1"), error: nil },
      { src: files[3], dest: filepath.Join(td, "file3"), error: nil },
      { src: "audiocc-file-def-dne", dest: files[1],
        error: errors.New("audiocc-file-def-dne") },
    }

    for i := range tests {
      e := copyFile(tests[i].src, tests[i].dest)
      if e == nil && tests[i].error == nil {
        break
      }
      if e.Error() != tests[i].error.Error() {
        t.Errorf("Expected %#v, got %#v", tests[i].error.Error(), e.Error())
      }
    }
  })
}

func TestRenameFolder(t *testing.T) {
  testFiles := []*testFile{
    {"dir1/file1", "abcde"},
    {"dir2/file2", "a"},
    {"dir3/file3", ""},
    {"dir4/file4", ""},
    {"dir6/file6", ""},
  }

  dir := createTestFiles(testFiles, t)
  defer os.RemoveAll(dir)

  tests := [][]string{
    {"dir2", "dir1", "dir1 (1)"},
    {"dir3", "dir1", "dir1 (2)"},
    {"dir4", "dir5", "dir5"},
    {"dir6", "path2/dir5", "path2/dir5"},
  }

  for i := 0; i < len(tests); i++ {
    r, err := renameFolder(filepath.Join(dir, tests[i][0]), filepath.Join(dir, tests[i][1]))
    if err != nil {
      t.Errorf("Unexpected error %v", err.Error())
    }

    exp := filepath.Join(dir, tests[i][2])
    if r != exp {
      t.Errorf("Expected %v, got %v", exp, r)
    }
  }
}
