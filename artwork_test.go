package main

import (
  "io"
  "os"
  "image"
  "testing"
  "io/ioutil"
  "path/filepath"
)

type mockFfmpeg struct {}

// TODO make file smaller (if possible)
func (m *mockFfmpeg) OptimizeAlbumArt(s, d string) (string, error) {
  err := copyFile(s, d)
  if err != nil {
    return "", err
  }
  return "", nil
}

func (m *mockFfmpeg) Exec(args ...string) (string, error) {
  return "", nil
}

func testArtworkFullPath(t *testing.T,
  testFiles [][]string, f func(dir string, files []string)) {

  filenames := []string{}
  contents := []string{}
  for i := range testFiles {
    if len(testFiles[i]) == 2 {
      filenames = append(filenames, testFiles[i][0])
      contents = append(contents, testFiles[i][1])
    }
  }

  dir := createTestFiles(filenames, contents, t)
  defer os.RemoveAll(dir)

  files := []string{}
  for x := range filenames {
    files = append(files, filepath.Join(dir, filenames[x]))
  }

  f(dir, files)
}

func testFileSize(t *testing.T, src string, size int64) {
  fi, err := os.Stat(src)
  if err != nil {
    t.Fatal(err)
  }
  if fi.Size() != size {
    t.Errorf("Expected %v, got %v", size, fi.Size())
  }
}

func TestFromPath(t *testing.T) {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.RemoveAll(td)

  tests := []struct {
    width int
    size, sizeOrig int64
    files [][]string
  }{
    { files: [][]string{{}} },
    { size: 3, files: [][]string{{ "folder.jpg", "abc" }} },
    { width: 1000, size: 8, sizeOrig: 1,
      files: [][]string{{ "cover.jpg", "abcdefgh" }, { "folder-orig.jpg", "a" }},
    },
  }

  for i := range tests {
    imageDecode := func (r io.Reader) (image.Config, string, error) {
      c := image.Config{ Width: tests[i].width }
      return c, "", nil
    }

    a := &artwork{ Ffmpeg: &mockFfmpeg{}, TempDir: td, ImgDecode: imageDecode }

    testArtworkFullPath(t, tests[i].files, func(dir string, files []string) {
      a.PathInfo = &pathInfo{ Fulldir: dir }
      err = a.fromPath()
      if err != nil {
        t.Fatal(err)
      }

      if len(files) > 0 {
        testFileSize(t, files[0], tests[i].size)
      }
      if len(files) > 1 {
        testFileSize(t, files[1], tests[i].sizeOrig)
      }
    })
  }
}

func TestCopyAsFolderJpg(t *testing.T) {
  testFiles := [][]string{
    { "file1.jpg", "abcde" },
    { "file2.jpeg", "a" },
    { "dir1/file3.JPG", "acddfefsefd" },
    { "dir1/dir2/file4.png", "dfadfd" },
  }

  testArtworkFullPath(t, testFiles, func(dir string, files []string) {
    tests := []struct {
      index int
      size, sizeOrig int64
      files [][]string
    }{
      { index: 3, size: 6, sizeOrig: 3,
        files: [][]string{{ "folder.jpg", "abc" }, { "folder-orig.jpg", "ab" }},
      }, {
        index: 2, size: 11, sizeOrig: 8,
        files: [][]string{{ "folder.jpg", "abc" }, { "folder-orig.jpg", "sdgyhrdw" }},
      },
    }

    for i := range tests {
      testArtworkFullPath(t, tests[i].files, func(dir2 string, files2 []string) {
        a := &artwork{ PathInfo: &pathInfo{ Fulldir: dir2 } }
        err := a.copyAsFolderJpg(files[tests[i].index])
        if err != nil {
          t.Fatal(err)
        }

        testFileSize(t, files2[0], tests[i].size)
        testFileSize(t, files2[1], tests[i].sizeOrig)
      })
    }
  })

}
