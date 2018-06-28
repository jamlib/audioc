package main

import (
  "io"
  "os"
  "errors"
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

func createTestFiles(paths, contents []string, t *testing.T) string {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }

  for i := range paths {
    if len(paths[i]) == 0 {
      continue
    }

    pa := strings.Split(paths[i], "/")
    if len(pa) == 0 {
      continue
    }

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
      err := ioutil.WriteFile(fullpath, []byte(contents[i]), 0644)
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
      { "dir1", "dir2/dir3/file1.ext" },
      { "dir1/dir2/dir3/file1.ext", "dir1/dir2/dir3", "dir2/dir3", "file1", ".ext" },
    }, {
      { "dir3/dir4", "file2.ext" },
      { "dir3/dir4/file2.ext", "dir3/dir4", "dir4", "file2", ".ext" },
    },
  }

  for x := range tests {
    pi := getPathInfo(tests[x][0][0], tests[x][0][1])
    compare := []string{ pi.Fullpath, pi.Fulldir, pi.Dir, pi.File, pi.Ext }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
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

func TestFilesByExtensionImages(t *testing.T) {
  testFiles := []string{
    "file1",
    "file2.jpeg",
    "dir1/file3.JPG",
    "dir1/dir2/file4.png",
  }
  var contents = make([]string, len(testFiles))

  result := []string{
    "dir1/dir2/file4.png",
    "dir1/file3.JPG",
    "file2.jpeg",
  }

  dir := createTestFiles(testFiles, contents, t)
  defer os.RemoveAll(dir)

  paths := filesByExtension(dir, imageExts)
  if strings.Join(paths, "\n") != strings.Join(result, "\n") {
    t.Errorf("Expected %v, got %v", result, paths)
  }
}

func TestFilesByExtensionAudio(t *testing.T) {
  testFiles := []string{
    "not audio file",
    "file1.FLAC",
    "file2.m4a",
    "dir1/file3.mp3",
    "dir1/dir2/file4.mp4",
    "dir1/dir2/file5.SHN",
    "dir1/dir2/file6.WAV",
  }
  var contents = make([]string, len(testFiles))

  result := []string{
    "dir1/dir2/file4.mp4",
    "dir1/dir2/file5.SHN",
    "dir1/dir2/file6.WAV",
    "dir1/file3.mp3",
    "file1.FLAC",
    "file2.m4a",
  }

  dir := createTestFiles(testFiles, contents, t)
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
  testFiles := []string{
    "file1", "file2.jpeg", "dir1/file3.JPG", "dir1/dir2/file4.png",
  }
  contents := []string{
    "abcde", "a", "acddfefsefd", "dfadfd",
  }

  dir := createTestFiles(testFiles, contents, t)
  defer os.RemoveAll(dir)

  files := []string{}
  for i := range testFiles {
    files = append(files, filepath.Join(dir, testFiles[i]))
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
