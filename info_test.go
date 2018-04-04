package main

import (
  "strings"
  "testing"
)

func TestInfoFromFile(t *testing.T) {
    tests := [][][]string{
    { { "sci160318d1_01_Shine" }, { "2016", "03", "18", "1", "01", "Shine" } },
    { { "jgb1980-02-28d1t1 Sugaree" }, { "1980", "02", "28", "1", "1", "Sugaree" } },
  }

  for x := range tests {
    i, _ := infoFromFile(tests[x][0][0])
    compare := []string{ i.Year, i.Month, i.Day, i.Disc, i.Track, i.Title }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

func TestInfoFromPath(t *testing.T) {
    tests := [][][]string{
    {
      { "Jerry Garcia Band/1980/1980.02.28 Kean College After Midnight - FLAC" },
      { "1980", "02", "28", "Kean College After Midnight - FLAC" },
    },{
      { "Grateful Dead/1975/1975 Blues For Allah" }, { "1975", "", "", "Blues For Allah" },
    },
  }

  for x := range tests {
    i := infoFromPath(tests[x][0][0], "/")
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
    { "Test 2000 More", "", "Test 2000 More" },
    { "2000 Album Name", "2000", "Album Name" },
    { "2001 - Venue, City", "2001", "Venue, City" },
  }

  for x := range tests {
    i := &info{}
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
    i := &info{}
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
    { { "1-01 " }, { "1", "01", "" } },
    { { "01-02 Album" }, { "01", "02", "Album" } },
    { { "1-3 - Venue" }, { "1", "3", "Venue" } },
    { { "1-Label" }, { "", "1", "Label" } },
    { { "01 - City" }, { "", "01", "City" } },
    { { "s01t01" }, { "01", "01", "" } },
    { { "d01t02" }, { "01", "02", "" } },
    { { "s2 01" }, { "2", "01", "" } },
    { { "d301" }, { "3", "01", "" } },
    { { "d2_05" }, { "2", "05", "" } },
  }

  for x := range tests {
    i := &info{}
    remain := i.matchDiscTrack(tests[x][0][0])
    compare := []string{ i.Disc, i.Track, remain }
    if strings.Join(compare, "\n") != strings.Join(tests[x][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[x][1], compare)
    }
  }
}

func TestMatchDiscOnly(t *testing.T) {
  tests := [][]string{
    { "SET 1", "1", "" },
    { "disc 02 ", "02", "" },
    { "yes cd 3 no", "3", "no" },
  }

  for x := range tests {
    i := &info{}
    remain := i.matchDiscOnly(tests[x][0])
    if i.Disc != tests[x][1] {
      t.Errorf("Expected %v, got %v", tests[x][1], i.Disc)
    }
    if remain != tests[x][2] {
      t.Errorf("Expected %v, got %v", tests[x][1], remain)
    }
  }
}

func TestMatchTitle(t *testing.T) {
  tests := [][]string{
    { "( ) Silly () (5) []", "Silly" },
  }

  for i := range tests {
    r := matchTitle(tests[i][0])
    if r != tests[i][1] {
      t.Errorf("Expected %v, got %v", tests[i][1], r)
    }
  }
}
