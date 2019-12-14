package audioc

import (
  "os"
  "testing"
  "encoding/json"
  "path/filepath"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/libaudio/fsutil"
)

func TestSkipFolderOnCollection(t *testing.T) {
  tests := []struct {
    path string
    col, skip bool
  }{
    // false: missing album folder
    { path: "Jerry Garcia Band/1.mp3", col: true, skip: false },
    // true: artist folder includes ' - '
    { path: "Grateful Dead - Unorganized/Album1/1.mp3", col: true, skip: true },
    // true: artist, year, album folder all exist and match
    { path: "Phish/2003/2003.07.09 Shoreline Amphitheatre, Mountain View, CA/1.mp3", col: true, skip: true },
  }

  for i := range tests {
    a := &audioc{ Config: &Config{ Collection: tests[i].col } }

    r := a.skipFolder(tests[i].path)
    if r != tests[i].skip {
      t.Errorf("%v: Expected %v, got %v", tests[i].path, tests[i].skip, r)
    }
  }
}

func TestSkipFolderOnArtist(t *testing.T) {
  tests := []struct {
    path, artist string
    skip bool
  }{
    // true: album folder equals derived
    { path: "Grateful Dead - Unorganized/1.mp3", artist: "Grateful Dead", skip: true },
    { path: "Random Dir/1.mp3", artist: "Anyone", skip: true },
  }

  for i := range tests {
    a := &audioc{ Config: &Config{ Artist: tests[i].artist } }

    r := a.skipFolder(tests[i].path)
    if r != tests[i].skip {
      t.Errorf("%v: Expected %v, got %v", tests[i].path, tests[i].skip, r)
    }
  }
}

func TestSkipFolderOnAlbum(t *testing.T) {
  tests := []struct {
    path, album string
    skip bool
  }{
    // true: album folder equals specified
    { path: "1980 Go To Heaven/1.mp3", album: "1980 Go To Heaven", skip: true },
    // false: album folder does not equal specified
    { path: "1980 Go To Heaven/1.mp3", album: "Go To Heaven", skip: false },
  }

  for i := range tests {
    a := &audioc{ Config: &Config{ Artist: "Whoever", Album: tests[i].album } }

    r := a.skipFolder(tests[i].path)
    if r != tests[i].skip {
      t.Errorf("%v: Expected %v, got %v", tests[i].path, tests[i].skip, r)
    }
  }
}

func TestProcessDirDNE(t *testing.T) {
  a := &audioc{ Config: &Config{ Dir: "audioc-dir-def-dne" } }
  err := a.Process()
  if err == nil {
    t.Errorf("Expected error, got none.")
  }
}

type TestProcessFiles struct {
  path string
  data *ffprobe.Tags
}

func createTestProcessFiles(t *testing.T, entryDir string,
  files []*TestProcessFiles) (*audioc, []int) {

  // ensure entryDir was passed as argument
  if len(entryDir) == 0 {
    t.Errorf("Expected entryDir")
  }

  a := &audioc{ Config: &Config{}, Ffmpeg: &ffmpeg.MockFfmpeg{},
    Ffprobe: &ffprobe.MockFfprobe{}, Files: []string{}, Workers: 1 }

  indexes := []int{}
  createFiles := []*fsutil.TestFile{}
  for x := range files {
    // convert probe data tags to JSON
    b, _ := json.Marshal(files[x].data)

    indexes = append(indexes, x)
    a.Files = append(a.Files, files[x].path)

    // create file with entry folder
    nestedPath := filepath.Join(entryDir, files[x].path)
    createFiles = append(createFiles, &fsutil.TestFile{nestedPath, string(b)})
  }

  a.Config.Dir, _ = fsutil.CreateTestFiles(t, createFiles)
  a.Config.Dir += fsutil.PathSep + entryDir

  return a, indexes
}

// also tests processArtwork(), processThreaded(), processFile(), processMp3()
func TestProcessMain(t *testing.T) {
  a, _ := createTestProcessFiles(t, "Phish", []*TestProcessFiles{
    { "2003/2003.07.17 Bonner Springs, KS/1-01 Chalk Dust Torture.flac",
      &ffprobe.Tags{},
    },{
      "cd1/file2.mp3",
      &ffprobe.Tags{
        Album: "2003.07.18 Alpine Valley, East Troy, WI",
        Track: "01", Title: "Axilla I",
      },
    },
  })
  defer os.RemoveAll(filepath.Dir(a.Config.Dir))

  a.Config.Artist = "Phish"
  a.Config.Write = true
  a.Config.Force = true

  err := a.Process()

  // ensure no errors in process
  if err != nil {
    t.Errorf("Expected no error, got: %v", err.Error())
  }

  // compare relative folder path & file name with expected result
  fileResults := []string{
    "2003.07.18 Alpine Valley, East Troy, WI/01 Axilla I.mp3",
    "Phish/2003/2003.07.17 Bonner Springs, KS/01-01 Chalk Dust Torture.mp3",
  }

  files := fsutil.FilesAudio(a.Config.Dir)
  if len(files) == 0 {
    t.Errorf("No resulting files found.")
  }

  for x := range files {
    if files[x] != fileResults[x] {
      t.Errorf("Expected %v, got %v", fileResults[x], files[x])
    }
  }

  // TODO: actually read & compare json data encoded within file
}
