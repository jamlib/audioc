package metadata

import (
  "strings"
  "testing"

  "github.com/JamTools/goff/ffprobe"
)

func TestToAlbum(t *testing.T) {
  tests := []struct {
    i *Info
    result string
  }{
    { i: &Info{ Year: "2004", Month: "06", Day: "15", Album: "Somewhere, USA" },
      result: "2004.06.15 Somewhere, USA",
    },{
      i: &Info{ Year: "2004", Album: "Great Album" },
      result: "2004 Great Album",
    },
  }

  for x := range tests {
    r := tests[x].i.ToAlbum()
    if r != tests[x].result {
      t.Errorf("Expected %v, got %v", tests[x].result, r)
    }
  }
}

func TestToFile(t *testing.T) {
  tests := []struct {
    i *Info
    result string
  }{
    { i: &Info{ Disc: "", Track: "6", Title: "After Midnight" },
      result: "6 After Midnight",
    },{
      i: &Info{ Disc: "2", Track: "03", Title: "Russian Lullaby" },
      result: "2-03 Russian Lullaby",
    },
  }

  for x := range tests {
    r := tests[x].i.ToFile()
    if r != tests[x].result {
      t.Errorf("Expected %v, got %v", tests[x].result, r)
    }
  }
}

func TestMatchProbeTags(t *testing.T) {
  tests := []struct {
    info, comb *Info
    tags *ffprobe.Tags
    match bool
  }{
    { info: &Info{ Album: "Kean College After Midnight", Title: "After Midnight" },
      tags: &ffprobe.Tags{ Album: "Something Else" },
      comb: &Info{ Album: "Kean College After Midnight", Title: "After Midnight" },
      match: false,
    },{
      info: &Info{ Album: "Kean College After Midnight", Year: "1980" },
      tags: &ffprobe.Tags{ Album: "1980 Kean College After Midnight" },
      comb: &Info{ Album: "Kean College After Midnight", Year: "1980" },
      match: true,
    },{
      info: &Info{ Album: "Kean College After Midnight", Year: "1980" },
      tags: &ffprobe.Tags{ Album: "1980.02.28 Kean College After Midnight" },
      comb: &Info{ Album: "Kean College After Midnight", Year: "1980", Month: "02", Day: "28" },
      match: false,
    },{
      info: &Info{ Disc: "1" },
      tags: &ffprobe.Tags{ Disc: "1/2" },
      comb: &Info{ Disc: "1" },
      match: true,
    },
  }

  for x := range tests {
    rInfo, match := tests[x].info.MatchProbeTags(tests[x].tags)

    if *rInfo != *tests[x].comb {
      t.Errorf("Expected %v, got %v", rInfo, tests[x].comb)
    }

    if match != tests[x].match {
      t.Errorf("Expected %v, got %v", match, tests[x].match)
    }
  }
}

func TestInfoFromFile(t *testing.T) {
  tests := [][][]string{
    { { "sci160318d1_01_Shine" }, { "2016", "03", "18", "1", "1", "Shine" } },
    { { "jgb1980-02-28d1t1 Sugaree" }, { "1980", "02", "28", "1", "1", "Sugaree" } },
    { { "03 - 02 Cold Rain and Snow"}, { "", "", "", "3", "2", "Cold Rain and Snow" } },
  }

  for x := range tests {
    i := &Info{}
    i.fromFile(tests[x][0][0])
    compare := []string{ i.Year, i.Month, i.Day, i.Disc, i.Track, i.Title }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

// also tests: fromAlbum
func TestInfoFromPath(t *testing.T) {
  tests := [][][]string{
    {
      { "Jerry Garcia Band/1980/1980.02.28 Kean College After Midnight - FLAC" },
      { "1980", "02", "28", "Kean College After Midnight" },
    },{
      { "Grateful Dead/1975/1975 Blues For Allah" }, { "1975", "", "", "Blues For Allah" },
    },
  }

  for x := range tests {
    i := &Info{}
    i.fromPath(tests[x][0][0])
    compare := []string{ i.Year, i.Month, i.Day, i.Album }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

func TestValidDate(t *testing.T) {
  tests := []map[string]bool{
    { "2000-01-01": true },
    { "2000-13-01": false },
    { "2000-01-32": false },
    { "00-01-01": false },
  }

  for i := range tests {
    for k, v := range tests[i] {
      dates := strings.Split(k, "-")
      r := validDate(dates[0], dates[1], dates[2])
      if r != v {
        t.Errorf("Expected %v, got %v", v, r)
      }
    }
  }
}

func TestMatchYearOnly(t *testing.T) {
  tests := [][]string{
    { "No Year Here", "", "No Year Here" },
    { "2000 Album Name", "2000", "Album Name" },
    { "2001 - Venue, City", "2001", "Venue, City" },
  }

  for x := range tests {
    i := &Info{}
    remain := i.matchYearOnly(tests[x][0])
    if i.Year != tests[x][1] {
      t.Errorf("Expected %v, got %v", tests[x][1], i.Year)
    }
    if remain != tests[x][2] {
      t.Errorf("Expected %v, got %v", tests[x][1], remain)
    }
  }
}

func TestMatchDate(t *testing.T) {
  tests := [][][]string{
    { { "not a date" }, { "", "", "", "not a date" } },
    { { "2000.01.01 Venue, City" }, { "2000", "01", "01", " Venue, City" } },
    { { "2000/1/01INFO" }, { "2000", "01", "01", "INFO" } },
    { { "2000-1-1" }, { "2000", "01", "01", "" } },
    { { "2000.01.31,01 Title" }, { "2000", "01", "31,01", " Title" } },
    { { "2000.01.01-03 Title" }, { "2000", "01", "01-03", " Title" } },
    { { "98-08-23 Title" }, { "1998", "08", "23", " Title" } },
    { { "5-6-22" }, { "1922", "05", "06", "" } },
    { { "sci160318d1_01_Shine" }, { "2016", "03", "18", "d1_01_Shine" } },
    { { "jgb1980-02-28d1t1 Sugaree" }, { "1980", "02", "28", "d1t1 Sugaree" } },
    { { "01.01.2001" }, { "2001", "01", "01", "" } },
    { { "1/1/2002" }, { "2002", "01", "01", "" } },
    { { "1-01-2003" }, { "2003", "01", "01", "" } },
    { { "03-30-69" }, { "1969", "03", "30", "" } },
    { { "06.15.10" }, { "2010", "06", "15", "" } },
    { { "04.05.06" }, { "2006", "04", "05", "" } },
  }

  for x := range tests {
    i := &Info{}
    remain := i.matchDate(tests[x][0][0])
    compare := []string{ i.Year, i.Month, i.Day, remain }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

func TestYearEnsureCentury(t *testing.T) {
  tests := [][]string{
    { "01", "2001" },
    { "01342", "" },
    { "ab", "" },
  }

  for i := range tests {
    r := yearEnsureCentury(tests[i][0])
    if r != tests[i][1] {
      t.Errorf("Expected %v, got %v", tests[i][1], r)
    }
  }
}

func TestMatchDiscTrack(t *testing.T) {
  tests := [][][]string{
    { { "not a track" }, { "", "", "not a track" } },
    { { "1-01 " }, { "1", "1", "" } },
    { { "01-02 Album" }, { "1", "2", "Album" } },
    { { "1-3 - Venue" }, { "1", "3", "Venue" } },
    { { "1-Label" }, { "", "1", "Label" } },
    { { "01 - City" }, { "", "1", "City" } },
    { { "s01t01" }, { "1", "1", "" } },
    { { "d01t02" }, { "1", "2", "" } },
    { { "s2 01" }, { "2", "1", "" } },
    { { "d301" }, { "3", "1", "" } },
    { { "d2_05" }, { "2", "5", "" } },
  }

  for x := range tests {
    i := &Info{}
    remain := i.matchDiscTrack(tests[x][0][0])
    compare := []string{ i.Disc, i.Track, remain }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

func TestMatchDiscOnly(t *testing.T) {
  tests := [][]string{
    { "SET 1", "1" },
    { "disc 02 ", "02" },
    { "yes cd 3 no", "3" },
  }

  for x := range tests {
    i := &Info{}
    i.matchDiscOnly(tests[x][0])
    if i.Disc != tests[x][1] {
      t.Errorf("Expected %v, got %v", tests[x][1], i.Disc)
    }
  }
}

func TestMatchAlbumOrTitle(t *testing.T) {
  tests := [][]string{
    { "/( ) Silly\\ () (5) []", "Silly" },
    { "Album - FLAC", "Album" },
    { ", SBD Album SBD", "SBD Album" },
    { "!^?Bitrate Album!? -320", "Bitrate Album!?" },
    { "intro/crowd", "intro-crowd" },
    { " Whereever [SBD 320-MP3]", "Whereever" },
  }

  for i := range tests {
    r := matchAlbumOrTitle(tests[i][0])
    if r != tests[i][1] {
      t.Errorf("Expected %v, got %v", tests[i][1], r)
    }
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
