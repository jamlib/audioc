package albumart

import (
  "io"
  "os"
  "image"
  "testing"
  "io/ioutil"
  "path/filepath"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
  "github.com/JamTools/goff/fsutil"
)

// passes a shared TempDir and labels: folder.jpg, folder-orig.jpg to
// provided function
func testArtwork(t *testing.T, testFunc func(td, f, fo string)) {
  td, err := ioutil.TempDir("", "")
  if err != nil {
    t.Fatal(err)
  }
  defer os.RemoveAll(td)

  testFunc(td, "folder.jpg", "folder-orig.jpg")
}

// creates test image files into temp directory which is passed to
// provided function
func testArtworkFiles(t *testing.T,
  testFiles map[string]string, testFunc func(dir string)) {

  files := []*fsutil.TestFile{}
  for k, v := range testFiles {
    files = append(files, &fsutil.TestFile{k, v})
  }

  dir, _ := fsutil.CreateTestFiles(t, files)
  defer os.RemoveAll(dir)

  testFunc(dir)
}

func TestArtworkProcess(t *testing.T) {
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

      a := &AlbumArt{ TempDir: td,
        Ffmpeg: &ffmpeg.MockFfmpeg{ Embedded: tests[i].embedded }, ImgDecode: imageDecode,
        Ffprobe: &ffprobe.MockFfprobe{ Width: tests[i].width, Embedded: tests[i].embedded } }

      testArtworkFiles(t, tests[i].files, func(dir string) {
        a.Fulldir = dir

        _, err := Process(a)
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

func TestArtworkEmbedded(t *testing.T) {
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
      a := &AlbumArt{ Ffmpeg: &ffmpeg.MockFfmpeg{ Embedded: tests[i].embedded }, TempDir: td }

      testArtworkFiles(t, tests[i].files, func(dir string) {
        a.Fulldir = dir

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

func TestArtworkFromPath(t *testing.T) {
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

      a := &AlbumArt{ Ffmpeg: &ffmpeg.MockFfmpeg{}, TempDir: td, ImgDecode: imageDecode }

      testArtworkFiles(t, tests[i].files, func(dir string) {
        a.Fulldir = dir

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

// also covers: copyAsFolderOrigJpg
func TestArtworkCopyAsFolderJpg(t *testing.T) {
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
          a := &AlbumArt{ Fulldir: dir2 }

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
