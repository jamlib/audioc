package main

import (
  "os"
  "strings"
  "testing"
  "path/filepath"
  "encoding/json"

  "github.com/JamTools/goff/ffmpeg"
  "github.com/JamTools/goff/ffprobe"
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

func TestValidWorkdir(t *testing.T) {
  tests := []struct {
    a *audiocc
    v bool
  }{
    { a: &audiocc{ Workdir: "" }, v: false },
    { a: &audiocc{ Workdir: "test" }, v: true },
  }

  for i := range tests {
    r := tests[i].a.validWorkdir() == nil
    if r != tests[i].v {
      t.Errorf("Expected %v, got %v", tests[i].v, r)
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
  createFiles := []*testFile{}
  for x := range files {
    // convert probe data tags to JSON
    b, _ := json.Marshal(files[x].data)

    indexes = append(indexes, x)
    a.Files = append(a.Files, files[x].path)
    createFiles = append(createFiles, &testFile{files[x].path, string(b)})
  }

  a.DirEntry = createTestFiles(createFiles, t)
  return a, indexes
}

// also tests processArtwork()
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

  a.Workdir = "audiocc"

  flags.Write = true
  defer func() { flags.Write = false }()

  err := a.process()
  if err != nil {
    t.Errorf("Expected no error, got: %v", err.Error())
  }
}

func TestProcessThreaded(t *testing.T) {
  // assume files within same directory
  a, indexes := createTestProcessFiles(t, []*TestProcessFiles{
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

  r := a.processThreaded(indexes)

  // trim dir entry, remove file from path
  r = onlyDir(strings.TrimPrefix(r, a.DirEntry + string(os.PathSeparator)))

  expected := "2003.07.18 Alpine Valley, East Troy, WI"
  if r != expected {
    t.Errorf("Expected %v, got %v", expected, r)
  }
}

func TestProcessIndex(t *testing.T) {
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

  tests := []struct {
    result string
  }{
    { result: "2003.07.17 Bonner Springs, KS/1-1 Chalk Dust Torture.mp3" },
    { result: "2003.07.18 Alpine Valley, East Troy, WI/1 Axilla I.mp3" },
  }

  for x := range tests {
    r, err := a.processIndex(x)
    if err != nil {
      t.Errorf("Expected no error, got: %v", err.Error())
    }

    r = strings.TrimPrefix(r, a.DirEntry + string(os.PathSeparator))
    if r != tests[x].result {
      t.Errorf("Expected %v, got %v", tests[x].result, r)
    }
  }
}

func TestProcessMp3(t *testing.T) {
  testFiles := []*testFile{
    {"dir1/file1.flac", ""},
    {"dir2/file2.mp3", ""},
  }

  dir := createTestFiles(testFiles, t)
  defer os.RemoveAll(dir)

  a := &audiocc{ DirEntry: dir, Ffmpeg: &ffmpeg.MockFfmpeg{} }

  tests := []struct {
    pi *pathInfo
    i *info
    result string
  }{
    { pi: getPathInfo(dir, testFiles[0].name),
      i: &info{ Disc: "", Track: "01", Title: "You Enjoy Myself" },
      result: "dir1/01 You Enjoy Myself.mp3",
    },{
      pi: getPathInfo(dir, testFiles[1].name),
      i: &info{ Disc: "2", Track: "04", Title: "Twist" },
      result: "dir2/2-04 Twist.mp3",
    },
  }

  for x := range tests {
    _, err := a.processMp3(tests[x].pi, tests[x].i)
    if err != nil {
      t.Errorf("Expected no error, got: %v", err.Error())
    }

    // TODO: actually read & compare json data encoded within file
    _, err = os.Stat(filepath.Join(dir, tests[x].result))
    if err != nil {
      t.Errorf("Expected file '%v' not found", tests[x].result)
    }
  }
}
