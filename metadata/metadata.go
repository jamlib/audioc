package metadata

import (
  "fmt"
  "time"
  "regexp"
  "strconv"
  "strings"
  "path/filepath"

  "github.com/jamlib/libaudio/ffprobe"
  "github.com/jamlib/libaudio/fsutil"
)

type Metadata struct {
  Filepath, Resultpath string
  Match bool
  Info *Info
}

type Info struct {
  Artist, Album, Year, Month, Day string
  Disc, Track, Title string
}

// filePath used to derive info
func New(filePath string) *Metadata {
  var file string
  m := &Metadata{ Filepath: filePath, Info: &Info{} }

  // derive file info if has file extension
  if regexp.MustCompile(`\.[A-Za-z0-9]{1,5}$`).FindString(filePath) != "" {
    file = filepath.Base(filePath)
    filePath = filepath.Dir(filePath)
  }

  // derive path info
  m.fromPath(filePath)

  if len(file) > 0 {
    m.fromFile(fixWhitespace(strings.TrimSuffix(file, filepath.Ext(file))))
  }

  return m
}

// build info from ffprobe.Tags
func ProbeTagsToInfo(p *ffprobe.Tags) *Info {
  return &Info{ Artist: p.Artist, Album: p.Album, Disc: p.Disc, Track: p.Track,
    Title: p.Title }
}

// compare file & path info against ffprobe.Tags info and combine into best
// return includes boolean if info sources match (no update necessary)
func (m *Metadata) MatchBestInfo(c, p *Info) (*Info, bool) {
  // pull date info from ffprobe.Tags album and force merge into itself
  p.mergeAlbumInfo(infoFromAlbum(p.Album), true)

  // compare using safeFilename since info is derived from filename
  // and it is acceptable for tags to have special characters
  compare := p
  compare.Album = safeFilename(compare.Album)
  compare.Title = safeFilename(compare.Title)

  match := true
  if *m.Info != *compare {
    match = false

    // take longer album
    m.Info.mergeAlbumInfo(p, false)

    if len(m.Info.Artist) == 0 {
      m.Info.Artist = p.Artist
    }
    if len(m.Info.Disc) == 0 {
      m.Info.Disc = regexp.MustCompile(`^\d+`).FindString(p.Disc)
    }
    if len(m.Info.Track) == 0 {
      m.Info.Track = regexp.MustCompile(`^\d+`).FindString(p.Track)
    }
    // take longer title
    if len(m.Info.Title) < len(p.Title) {
      m.Info.Title = p.Title
    }
  }

  // set custom artist
  if len(c.Artist) > 0 {
    if c.Artist != m.Info.Artist {
      match = false
    }
    m.Info.Artist = c.Artist
  }

  // set custom album
  if len(c.Album) > 0 {
    if c.Album != m.Info.ToAlbum() {
      match = false
    }
    m.Info.mergeAlbumInfo(infoFromAlbum(c.Album), true)
  }

  return m.Info, match
}

// derive info album info from nested folder path
// strip out innermost dirs that are irreleveant (ie cd1)
func (m *Metadata) fromPath(p string) {
  sa := strings.Split(p, fsutil.PathSep)

  // start inner-most folder, work out
  foundAlbum := false
  for x := len(sa)-1; x >= 0; x-- {
    i := infoFromAlbum(sa[x])

    if len(i.Album) > 0 {
      if len(m.Info.Album) == 0 {
        m.Info.mergeAlbumInfo(i, true)
      }

      if foundAlbum {
        m.Resultpath = sa[x] + fsutil.PathSep + m.Resultpath
      } else {
        foundAlbum = true
      }
    }
  }
}

// determine Disc, Year, Month, Day, Track, Title from file string
func (m *Metadata) fromFile(s string) {
  // match and remove date from anywhere within string
  s = m.Info.matchDate(s)

  // attempt to remove artist or album prefixes
  strs := []string{m.Info.Artist, m.Info.Album}
  vars := []string{" - ", " ", "-"}

  for x := range strs {
    for y := range vars {
      s = strings.TrimPrefix(s, strs[x] + vars[y])
    }
  }

  // match disc, track number, title
  s = m.Info.matchDiscTrack(s)
  m.Info.Title = matchAlbumOrTitle(s)
}

func infoFromAlbum(s string) *Info {
  i := &Info{}
  s = i.matchDiscOnly(s)
  s = i.matchDate(s)
  s = i.matchYearOnly(s)
  i.Album = matchAlbumOrTitle(s)
  return i
}

func (i *Info) mergeAlbumInfo(a *Info, force bool) {
  if len(a.Album) > 0 && (force || !force && len(a.Album) > len(i.Album)) {
    i.Album = a.Album
  }
  if len(a.Year) > 0 && (force || !force && len(i.Year) == 0) {
    i.Year = a.Year
  }
  if len(a.Month) > 0 && (force || !force && len(i.Month) == 0) {
    i.Month = a.Month
  }
  if len(a.Day) > 0 && (force || !force && len(i.Day) == 0) {
    i.Day = a.Day
  }
}

// returns album prefixed with fulldate, year, or nothing (if no year)
func (i *Info) ToAlbum() string {
  if len(i.Year) > 0 {
    if len(i.Month) > 0 && len(i.Day) > 0 {
      return fmt.Sprintf("%s.%s.%s %s", i.Year, i.Month, i.Day, i.Album)
    }
    return fmt.Sprintf("%s %s", i.Year, i.Album)
  }
  return i.Album
}

// returns filename string from Disc, Track, Title (ex: "01-01 Title")
// without Disc (ex: "01 Title")
func (i *Info) ToFile() string {
  // closure to pad Disc & Track
  pad := func (s string) string {
    d, _ := strconv.Atoi(s)
    if d == 0 {
      return ""
    }
    return fmt.Sprintf("%02d", d)
  }

  var out string
  if len(i.Disc) > 0 {
    out += pad(i.Disc) + "-"
  }
  if len(i.Track) > 0 {
    out += pad(i.Track) + " "
  }
  return out + safeFilename(i.Title)
}

// converts roman numeral to int; only needs to support up to 5
var romanNumeralMap = map[string]string{
  "I": "1", "II": "2", "III": "3", "IV": "4", "V": "5",
}

// date expressed in multiple ways
var dateRegexps = []string{
  // pattern: '2000-1-01' '2000/01/01' '2000.1.1'
  // also multiple days: '2000.01.01-03' '2000.01.31,01'
  `(?P<year>\d{4})[/\.-]{1}(?P<month>\d{1,2})[/\.-]{1}(?P<day>\d{1,2}[-,]*\d*)`,
  // pattern: nugs.net: sci160318d1_01_Shine, ph990710d1_01_Wilson
  `[a-z0-9]{2,10}(?P<year>\d{2})(?P<month>\d{2})(?P<day>\d{2})`,
  // pattern: '01.01.2000' '1/1/2000' '1-01-2000'
  `(?P<month>\d{1,2})[/\.-]{1}(?P<day>\d{1,2})[/\.-]{1}(?P<year>\d{4})`,
  // pattern: '03-30-69' '06.09.73'
  `(?P<month>\d{1,2})[/\.-]{1}(?P<day>\d{1,2})[/\.-]{1}(?P<year>\d{2})`,
  // pattern: '98-08-23'
  `(?P<year>\d{2})[/\.-]{1}(?P<month>\d{1,2})[/\.-]{1}(?P<day>\d{1,2})`,
}

// ensure date inputs are valid
func validDate(year, mon, day string) bool {
  var err error
  _, err = time.Parse("2006-01-02", fmt.Sprintf("%s-%s-%s", year, mon, day))
  if err != nil {
    return false
  }
  return true
}

// if full date not found, try year only
func (i *Info) matchYearOnly(s string) string {
  m, remain := regexpMatch(s, `^(?P<year>\d{4})\s{1}-*\s*`)
  if len(m) < 2 {
    return s
  }
  if len(i.Year) == 0 {
    i.Year = m[1]
  }
  return remain
}

// irerate through dateRegexps returning first valid date found
func (i *Info) matchDate(s string) string {
  for index, regExpStr := range dateRegexps {
    m, remain := regexpMatch(s, regExpStr)
    if len(m) == 0 {
      continue
    }

    var year, mon, day string

    // order of matches depends on position within dateRegexps slice
    if index > 1 && index != 4 {
      // month day year
      year, mon, day = m[3], m[1], m[2]
    } else {
      // year month day
      year, mon, day = m[1], m[2], m[3]
    }

    // formatting
    mon = fmt.Sprintf("%02s", mon)
    day = fmt.Sprintf("%02s", day)
    year = yearEnsureCentury(year)

    v := validDate(year, mon, regexp.MustCompile(`\d{1,2}`).FindString(day))
    if !v {
      continue
    }

    if len(i.Year) == 0 || len(i.Month) == 0 || len(i.Day) == 0 {
      i.Year, i.Month, i.Day = year, mon, day
    }
    return strings.TrimSpace(remain)
  }
  return s
}

// expand year to include century
func yearEnsureCentury(year string) string {
  if len(year) == 2 {
    y, err := strconv.Atoi(year)
    if err != nil {
      return ""
    }

    // compare with current year to determine prefix
    nowYear := strconv.Itoa(time.Now().Year())
    l, r := nowYear[:2], nowYear[2:]
    ri, _ := strconv.Atoi(r)

    if y > ri {
      li, _ := strconv.Atoi(l)
      year = strconv.Itoa(li-1) + year
    } else {
      year = l + year
    }
  }
  if len(year) != 4 {
    return ""
  }
  return year
}

var discTrackRegexps = []string{
  // pattern:^ '1-01 ', '01-02 ', '1-3 - ', '03 - 02 '
  `^(?P<disc>\d{1,2})\s*-\s*(?P<track>\d{1,2})\s{1}[-]*\s*`,
  // pattern:^ '01 - ', '1 ', '1-' (only track)
  `^(?P<disc>)(?P<track>\d{1,2})\s*[-]*\s*`,
  // pattern: 's01t01', 'd01t01', 's1 01', 'd301', 'd1_01'
  `[sd](?P<disc>\d{2})[-. _t]*(?P<track>\d{2})`,
  `[sd](?P<disc>\d{1})[-. _t]*(?P<track>\d{2})`,
  `[sd](?P<disc>\d{1})[-. _t]*(?P<track>\d{1})`,
}

func (i *Info) matchDiscTrack(s string) string {
  for _, regExpStr := range discTrackRegexps {
    m, r := regexpMatch(s, regExpStr)
    if len(m) == 0 {
      continue
    }

    // remove prefix 0s
    for x := range m {
      m[x] = regexp.MustCompile(`^0+`).ReplaceAllString(m[x], "")
    }

    i.Track = m[2]
    if len(i.Disc) == 0 {
      i.Disc = m[1]
    }

    return r
  }
  return s
}

func (i *Info) matchDiscOnly(s string) string {
  m, r := regexpMatch(s, `(?i)(cd|disc|set|disk)\s*(?P<disc>\d{1,2})\s*`)
  if len(m) >= 3 && len(i.Disc) == 0 {
    i.Disc = m[2]
    return r
  }
  return s
}

func regexpMatch(s, regExpStr string) ([]string, string) {
  m := regexp.MustCompile(regExpStr).FindStringSubmatch(s)

  if len(m) > 0 {
    i := strings.Index(s, m[0])
    s = s[(i+len(m[0])):]
  }

  return m, s
}

var albumTitleRemoveRegexps = []string{
  // from end: remove [*] from end where * is wildcard
  `\s*\[[^\[\]]*\]\s*$`,
  // anywhere: only these chars allowed: A-Za-z0-9-',.!?&> _()
  `[^A-Za-z0-9-',.!?&> _()]+`,
  // anywhere: remove () (1) ( )
  `\s*\({1}[\d\s]*\){1}\s*`,
  // from end: remove file extension
  `\s*-*\s*(?i)(flac|m4a|mp3|mp4|shn|wav)$`,
  // from end: remove bitrate/sbd
  `\s*-*\s*(?i)(128|192|256|320|sbd)$`,
  // from beginning: remove anything except A-Za-z0-9(
  `^[-',.&>_)!?]+`,
  // from end: remove anything except A-Za-z0-9)?!
  `[-',.&>_(]+$`,
}

func matchAlbumOrTitle(s string) string {
  // replace / or \ with -
  s = regexp.MustCompile(`[\/\\]+`).ReplaceAllString(s, "-")

  // replace various matches with blank string
  for _, regExpStr := range albumTitleRemoveRegexps {
    s = regexp.MustCompile(regExpStr).ReplaceAllString(s, "")
  }

  // replace _ with space
  s = regexp.MustCompile(`[_]+`).ReplaceAllString(s, " ")

  return fixWhitespace(s)
}

// replace whitespaces with one space
func fixWhitespace(s string) string {
  return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}

// strip out characters from filename
func safeFilename(f string) string {
  return fixWhitespace(regexp.MustCompile(`[^A-Za-z0-9-,'!?& _()]+`).ReplaceAllString(f, ""))
}
