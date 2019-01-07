package audioc

import (
  "os"
  "testing"
  "encoding/json"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/libaudio/fsutil"
)

func TestSkipArtistOnCollection(t *testing.T) {
  tests := []struct {
    base, path string
    col, skip bool
  }{
    { base: "/", path: "Jerry Garcia Band/1.mp3", col: true, skip: false },
    { base: "/", path: "Grateful Dead - Unorganized/Album1/1.mp3", col: true, skip: true },
    { base: "/Grateful Dead - Unorganized", path: "1.mp3", col: false, skip: true },
  }

  for i := range tests {
    a := &audioc{ Flags: flags{ Collection: tests[i].col } }

    r := a.skipFolder(tests[i].base, tests[i].path)
    if r != tests[i].skip {
      t.Errorf("%v: Expected %v, got %v", tests[i].base+tests[i].path, tests[i].skip, r)
    }
  }
}

func TestSkipFolder(t *testing.T) {
  tests := []struct {
    base, path string
    col, skip bool
  }{
    { base: "/", path: "Random Dir/1.mp3", col: false, skip: true },
    { base: "/", path: "Phish/2003/2003.07.09 Shoreline Amphitheatre, Mountain View, CA/1.mp3", col: true, skip: true },
  }

  for i := range tests {
    a := &audioc{ Flags: flags{ Collection: tests[i].col } }

    r := a.skipFolder(tests[i].base, tests[i].path)
    if r != tests[i].skip {
      t.Errorf("%v: Expected %v, got %v", tests[i].base+tests[i].path, tests[i].skip, r)
    }
  }
}

func TestProcessDirDNE(t *testing.T) {
  a := &audioc{ DirEntry: "audioc-dir-def-dne" }
  err := a.Process()
  if err == nil {
    t.Errorf("Expected error, got none.")
  }
}

type TestProcessFiles struct {
  path string
  data *ffprobe.Tags
}

func createTestProcessFiles(t *testing.T, files []*TestProcessFiles) (*audioc, []int) {
  a := &audioc{ Ffmpeg: &ffmpeg.MockFfmpeg{}, Ffprobe: &ffprobe.MockFfprobe{},
    Files: []string{}, Workers: 1 }

  indexes := []int{}
  createFiles := []*fsutil.TestFile{}
  for x := range files {
    // convert probe data tags to JSON
    b, _ := json.Marshal(files[x].data)

    indexes = append(indexes, x)
    a.Files = append(a.Files, files[x].path)
    createFiles = append(createFiles, &fsutil.TestFile{files[x].path, string(b)})
  }

  a.DirEntry, _ = fsutil.CreateTestFiles(t, createFiles)
  return a, indexes
}

// also tests processArtwork(), processThreaded(), processFile(), processMp3()
func TestProcessMain(t *testing.T) {
  a, _ := createTestProcessFiles(t, []*TestProcessFiles{
    { "Phish/2003/2003.07.17 Bonner Springs, KS/1-01 Chalk Dust Torture.flac",
      &ffprobe.Tags{},
    },{
      "dir2/file2.mp3",
      &ffprobe.Tags{
        Album: "2003.07.18 Alpine Valley, East Troy, WI",
        Track: "1", Title: "Axilla I",
      },
    },
  })
  defer os.RemoveAll(a.DirEntry)

  a.Flags.Write = true
  a.Flags.Force = true

  err := a.Process()

  // ensure no errors in process
  if err != nil {
    t.Errorf("Expected no error, got: %v", err.Error())
  }

  // compare relative folder path & file name with expected result
  fileResults := []string{
    "2003.07.17 Bonner Springs, KS/1-1 Chalk Dust Torture.mp3",
    "2003.07.18 Alpine Valley, East Troy, WI/1 Axilla I.mp3",
  }

  files := fsutil.FilesAudio(a.DirEntry)
  for x := range files {
    if files[x] != fileResults[x] {
      t.Errorf("Expected %v, got %v", fileResults[x], files[x])
    }
  }

  // TODO: actually read & compare json data encoded within file
}
