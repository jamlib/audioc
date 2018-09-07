package main

import (
  "os"
  "testing"
  "encoding/json"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
  "github.com/JamTools/goff/fsutil"
)

func TestSkipArtistOnCollection(t *testing.T) {
  tests := []struct {
    dir string
    col, skip bool
  }{
    { dir: "Jerry Garcia Band", col: true, skip: false },
    { dir: "Grateful Dead - Unorganized/Anything", col: true, skip: true },
    { dir: "Grateful Dead - Unorganized", col: false, skip: false },
  }

  for i := range tests {
    flags.Collection = tests[i].col
    defer func() { flags.Collection = false }()

    r := skipArtistOnCollection(tests[i].dir)
    if r != tests[i].skip {
      t.Errorf("Expected %v, got %v", tests[i].skip, r)
    }
  }
}

func TestSkipFast(t *testing.T) {
  tests := []struct {
    dir string
    skip bool
  }{
    { dir: "Random Dir", skip: false },
    { dir: "Phish/2003/2003.07.09 Shoreline Amphitheatre, Mountain View, CA", skip: true },
  }

  flags.Fast = true
  defer func() { flags.Fast = false }()

  for i := range tests {
    r := skipFast(tests[i].dir)
    if r != tests[i].skip {
      t.Errorf("Expected %v, got %v", tests[i].skip, r)
    }
  }
}

func TestProcessDirDNE(t *testing.T) {
  a := &audiocc{ DirEntry: "audiocc-dir-def-dne" }
  err := a.process()
  if err == nil {
    t.Errorf("Expected error, got none.")
  }
}

type TestProcessFiles struct {
  path string
  data *ffprobe.Tags
}

func createTestProcessFiles(t *testing.T, files []*TestProcessFiles) (*audiocc, []int) {
  a := &audiocc{ Ffmpeg: &ffmpeg.MockFfmpeg{}, Ffprobe: &ffprobe.MockFfprobe{},
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

  a.DirEntry = fsutil.CreateTestFiles(t, createFiles)
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

  flags.Write = true
  defer func() { flags.Write = false }()

  err := a.process()

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
