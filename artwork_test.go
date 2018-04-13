package main

import (
  "io"
  "os"
  "image"
  "testing"
  "io/ioutil"
  "path/filepath"
)

type mockFfmpeg struct {
  Embedded string
}

func (m *mockFfmpeg) OptimizeAlbumArt(s, d string) (string, error) {
  // temp file for optimizing
  tmp, err := ioutil.TempFile("", "")
  if err != nil {
    return "", err
  }
  defer os.Remove(tmp.Name())
  defer tmp.Close()

  b, err := ioutil.ReadFile(s)
  if err != nil {
    return "", err
  }

  // can make smaller
  contents := string(b)
  if len(contents) > 0 {
    _, err = io.WriteString(tmp, contents[:len(contents)-1])
    if err != nil {
      return "", err
    }
    // use instead of original source
    s = tmp.Name()
  }

  err = copyFile(s, d)
  if err != nil {
    return "", err
  }
  return "", nil
}

func (m *mockFfmpeg) Exec(args ...string) (string, error) {
  // hook on extract audio
  if len(args) == 4 {
    err := ioutil.WriteFile(args[3], []byte(m.Embedded), 0644)
    if err != nil {
      return "", err
    }
  }
  return "", nil
}

type mockFfprobe struct {
  Width int
  Embedded string
}

func (m *mockFfprobe) EmbeddedImage() (int, int, bool) {
  if len(m.Embedded) > 0 {
    return m.Width, 0, true
  }
  return 0, 0, false
}

func testArtwork(t *testing.T, testFunc func(td, f, fo string)) {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.RemoveAll(td)

  testFunc(td, "folder.jpg", "folder-orig.jpg")
}

func testArtworkFiles(t *testing.T,
  testFiles map[string]string, testFunc func(dir string)) {

  files := []string{}
  contents := []string{}
  for k, v := range testFiles {
    files = append(files, k)
    contents = append(contents, v)
  }

  dir := createTestFiles(files, contents, t)
  defer os.RemoveAll(dir)

  testFunc(dir)
}

func TestProcess(t *testing.T) {
  testArtwork(t, func(td, f, fo string) {
    tests := []struct {
      width int
      embedded string
      files, results map[string]string
    }{
      { width: 1000, embedded: "123",
        files: map[string]string{ f: "abcdefgh", fo: "a" },
        results: map[string]string{ f: "12", fo: "abcdefgh" },
      },{
        width: 0,
        files: map[string]string{ f: "abcdefgh", fo: "a" },
        results: map[string]string{ f: "abcdefgh", fo: "a" },
      },
    }

    for i := range tests {
      imageDecode := func (r io.Reader) (image.Config, string, error) {
        c := image.Config{ Width: tests[i].width }
        return c, "", nil
      }

      a := &artwork{ TempDir: td,
        Ffmpeg: &mockFfmpeg{ Embedded: tests[i].embedded }, ImgDecode: imageDecode,
        Ffprobe: &mockFfprobe{ Width: tests[i].width, Embedded: tests[i].embedded } }

      testArtworkFiles(t, tests[i].files, func(dir string) {
        a.PathInfo = &pathInfo{ Fulldir: dir }

        _, err := a.process()
        if err != nil {
          t.Fatal(err)
        }

        for k, v := range tests[i].results {
          b, _ := ioutil.ReadFile(filepath.Join(dir, k))
          if string(b) != v {
            t.Errorf("Expected %v, got %v", v, string(b))
          }
        }
      })
    }
  })
}

func TestEmbedded(t *testing.T) {
  testArtwork(t, func(td, f, fo string) {
    tests := []struct {
      width int
      embedded string
      files, results map[string]string
    }{
      { width: 1000, embedded: "123",
        files: map[string]string{ f: "abcdefgh", fo: "a" },
        results: map[string]string{ f: "12", fo: "abcdefgh" },
      },{
        width: 500, embedded: "123",
        files: map[string]string{ f: "abcdefgh", fo: "a" },
        results: map[string]string{ f: "123", fo: "abcdefgh" },
      },
    }

    for i := range tests {
      a := &artwork{ Ffmpeg: &mockFfmpeg{ Embedded: tests[i].embedded }, TempDir: td }

      testArtworkFiles(t, tests[i].files, func(dir string) {
        a.PathInfo = &pathInfo{ Fulldir: dir }

        err := a.embedded(tests[i].width, 1)
        if err != nil {
          t.Fatal(err)
        }

        for k, v := range tests[i].results {
          b, _ := ioutil.ReadFile(filepath.Join(dir, k))
          if string(b) != v {
            t.Errorf("Expected %v, got %v", v, string(b))
          }
        }
      })
    }
  })
}

func TestFromPath(t *testing.T) {
  testArtwork(t, func(td, f, fo string) {
    tests := []struct {
      width int
      files, results map[string]string
    }{
      { files: map[string]string{},
        results: map[string]string{},
      },{
        files: map[string]string{ f: "abc" },
        results: map[string]string{ f: "abc" },
      },{
        width: 1000,
        files: map[string]string{ "cover.jpg": "abcdefgh", fo: "a" },
        results: map[string]string{ f: "abcdefg", fo: "a" },
      },
    }

    for i := range tests {
      imageDecode := func (r io.Reader) (image.Config, string, error) {
        c := image.Config{ Width: tests[i].width }
        return c, "", nil
      }

      a := &artwork{ Ffmpeg: &mockFfmpeg{}, TempDir: td, ImgDecode: imageDecode }

      testArtworkFiles(t, tests[i].files, func(dir string) {
        a.PathInfo = &pathInfo{ Fulldir: dir }

        err := a.fromPath()
        if err != nil {
          t.Fatal(err)
        }

        for k, v := range tests[i].results {
          b, _ := ioutil.ReadFile(filepath.Join(dir, k))
          if string(b) != v {
            t.Errorf("Expected %v, got %v", v, string(b))
          }
        }
      })
    }
  })
}

func TestCopyAsFolderJpg(t *testing.T) {
  testArtwork(t, func(td, f, fo string) {
    testFiles := map[string]string{
      "file1.jpg": "abcde",
      "file2.jpeg": "a",
      "dir1/file3.JPG": "acddfefsefd",
      "dir1/dir2/file4.png": "dfadfd",
    }

    testArtworkFiles(t, testFiles, func(dir string) {
      tests := []struct {
        key string
        files, results map[string]string
      }{
        { key: "dir1/dir2/file4.png",
          files: map[string]string{ f: "abc", fo: "ab" },
          results: map[string]string{ f: "dfadfd", fo: "abc" },
        },{
          key: "dir1/file3.JPG",
          files: map[string]string{ f: "abc", fo: "sdgyhrdw" },
          results: map[string]string{ f: "acddfefsefd", fo: "sdgyhrdw" },
        },
      }

      for i := range tests {
        testArtworkFiles(t, tests[i].files, func(dir2 string) {
          a := &artwork{ PathInfo: &pathInfo{ Fulldir: dir2 } }

          err := a.copyAsFolderJpg(filepath.Join(dir, tests[i].key))
          if err != nil {
            t.Fatal(err)
          }

          for k, v := range tests[i].results {
            b, _ := ioutil.ReadFile(filepath.Join(dir2, k))
            if string(b) != v {
              t.Errorf("Expected %v, got %v", v, string(b))
            }
          }
        })
      }
    })
  })
}
