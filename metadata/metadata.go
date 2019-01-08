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
  Basepath, Filepath, Infodir string
  Match bool
  Info *Info
}

type Info struct {
  Artist, Album, Year, Month, Day string
  Disc, Track, Title string
}

// Ffprober within context of metadata only needs to implement GetData
type Ffprober interface {
  GetData(filePath string) (*ffprobe.Data, error)
}

// create metadata, set Infodir
func New(basePath, filePath string) *Metadata {
  m := &Metadata{ Basepath: basePath, Filepath: filePath,
    Infodir: filepath.Dir(filePath), Info: &Info{},
  }

  // if no Infodir, use innermost dir of basePath 
  if m.Infodir == "" || m.Infodir == "." {
    m.Infodir = filepath.Base(basePath)
  }

  return m
}

func (m *Metadata) Probe(ffp Ffprober) error {
  fp := filepath.Join(m.Basepath, m.Filepath)
  _, file := filepath.Split(m.Filepath)

  // info from path & file name
  m.Info.FromPath(m.Infodir)
  m.Info.FromFile(strings.TrimSuffix(file, filepath.Ext(file)))

  // info from embedded tags within audio file
  d, err := ffp.GetData(fp)
  if err != nil {
    return err
  }

  // if artist not yet specified, use ffprobe artist tag
  if m.Info.Artist == "" {
    m.Info.Artist = d.Format.Tags.Artist
  }

  // combine info w/ embedded tags
  m.Info, m.Match = m.Info.MatchProbeTags(d.Format.Tags)

  return nil
}

// returns album prefixed with fulldate, year, or nothing (if no year)
func (i *Info) ToAlbum() string {
  if i.Year != "" {
    if i.Month != "" && i.Day != ""{
      return fmt.Sprintf("%s.%s.%s %s", i.Year, i.Month, i.Day, i.Album)
    }
    return fmt.Sprintf("%s %s", i.Year, i.Album)
  }
  return i.Album
}

// returns filename string from Disc, Track, Title (ex: "01-01 Title.mp3")
// without Disc (ex: "01 Title.mp3")
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

// compare file & path info against ffprobe.Tags and combine into best info
// return includes boolean if info sources match (no update necessary)
func (i *Info) MatchProbeTags(p *ffprobe.Tags) (*Info, bool) {

  // build info from ffprobe.Tags
  tagInfo := &Info{
    Artist: p.Artist,
    Disc: p.Disc,
    Track: p.Track,
    Title: safeFilename(p.Title),
  }

  // separate date from album
  a := p.Album
  a = tagInfo.matchDate(a)
  a = tagInfo.matchYearOnly(a)
  tagInfo.Album = safeFilename(a)

  // compare using safeFilename since info is derived from filename
  compare := tagInfo
  compare.Title = safeFilename(compare.Title)

  if *i != *compare {
    // combine infos
    r := tagInfo

    // info overrides probe for Artist
    if r.Artist != i.Artist {
      r.Artist = i.Artist
    }

    // update Year, Month, Day, Disc, Track if not set
    if len(r.Year) == 0 {
      r.Year = i.Year
    }
    if len(r.Month) == 0 {
      r.Month = i.Month
    }
    if len(r.Day) == 0 {
      r.Day = i.Day
    }
    if len(r.Disc) == 0 || regexp.MustCompile(`^\d+`).FindString(r.Disc) != r.Disc {
      r.Disc = i.Disc
    }
    if len(r.Track) == 0 {
      r.Track = i.Track
    }

    // update Album, Title if longer
    r.Album = strings.TrimSpace(r.Album)
    if len(r.Album) < len(i.Album) {
      r.Album = i.Album
    }

    if len(r.Title) < len(i.Title) {
      r.Title = i.Title
    }

    return r, false
  }

  return i, true
}

// determine Disc, Year, Month, Day, Track, Title from file string
func (i *Info) FromFile(s string) *Info {
  strs := []string{i.Artist, i.Album}

  // trim slice of prefixes w/ multiple separator variations
  for x := range strs {
    s = strings.TrimLeft(s, strs[x] + " - ")
    s = strings.TrimLeft(s, strs[x] + "-")
  }

  s = i.matchDate(s)
  s = i.matchDiscTrack(s)
  i.Title = matchAlbumOrTitle(s)

  return i
}

// derive info album info from nested folder path
func (i *Info) FromPath(p string) *Info {
  // start inner-most folder, work out
  sa := strings.Split(p, fsutil.PathSep)
  for x := len(sa)-1; x >= 0; x-- {
    // determine Disc, Year, Month, Day, Album
    sa[x] = i.matchDiscOnly(sa[x])
    sa[x] = i.matchDate(sa[x])
    sa[x] = i.matchYearOnly(sa[x])

    if len(i.Album) == 0 {
      i.Album = matchAlbumOrTitle(sa[x])
    }
  }
  return i
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
    return remain
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

func matchAlbumOrTitle(s string) string {
  // replace / or \ with -
  s = regexp.MustCompile(`[\/\\]+`).ReplaceAllString(s, "-")

  // remove [*] from end where * is wildcard
  s = regexp.MustCompile(`\s*\[[^\[\]]*\]\s*$`).ReplaceAllString(s, "")

  // only these chars allowed: A-Za-z0-9-',.!?&> _()
  s = regexp.MustCompile(`[^A-Za-z0-9-',.!?&> _()]+`).ReplaceAllString(s, "")

  // remove () (1)
  s = regexp.MustCompile(`\s*\({1}[\d\s]*\){1}\s*`).ReplaceAllString(s, "")
  s = fixWhitespace(s)

  // remove file extension from end
  s = regexp.MustCompile(`\s*-*\s*(?i)(flac|m4a|mp3|mp4|shn|wav)$`).ReplaceAllString(s, "")

  // remove bitrate/sbd from end
  s = regexp.MustCompile(`\s*-*\s*(?i)(128|192|256|320|sbd)$`).ReplaceAllString(s, "")

  // from beginning: remove anything except A-Za-z0-9(
  s = regexp.MustCompile(`^[-',.&>_)!?]+`).ReplaceAllString(s, "")

  // from end: remove anything except A-Za-z0-9)?!
  s = regexp.MustCompile(`[-',.&>_(]+$`).ReplaceAllString(s, "")

  // replace _ with space
  s = regexp.MustCompile(`[_]+`).ReplaceAllString(s, " ")

  return fixWhitespace(safeFilename(s))
}

// replace whitespaces with one space
func fixWhitespace(s string) string {
  return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}

// strip out characters from filename
func safeFilename(f string) string {
  return fixWhitespace(regexp.MustCompile(`[^A-Za-z0-9-,'!?& _()]+`).ReplaceAllString(f, ""))
}
