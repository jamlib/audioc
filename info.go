// info can be from:
// id3 tag, path, filename

package main

import (
  "fmt"
  "time"
  "regexp"
  "strconv"
  "strings"

  "github.com/JamTools/goff/ffprobe"
)

type info struct {
  Artist, Album, Year, Month, Day string
  Disc, Track, Title string
}

func (i *info) toAlbum() string {
  return fmt.Sprintf("%s.%s.%s %s", i.Year, i.Month, i.Day, i.Album)
}

func (i *info) toFile() string {
  var out string
  if len(i.Disc) > 0 {
    out += i.Disc + "-"
  }

  return out + i.Track + " " + safeFilename(i.Title)
}

// compare info against ffprobe.Tags and combine into best info
func (i *info) matchProbeTags(p *ffprobe.Tags) (*info, bool) {
  tagInfo := &info{
    Artist: p.Artist,
    Disc: p.Disc,
    Track: p.Track,
    Title: matchAlbumOrTitle(p.Title),
  }
  tagInfo.fromAlbum(p.Album)

  compare := tagInfo
  compare.Title = safeFilename(compare.Title)
  if *i != *compare {
    // combine infos
    result := tagInfo
    if len(result.Album) < len(i.Album) {
      result.Album = i.Album
    }
    if len(result.Year) == 0 {
      result.Year = i.Year
    }
    if len(result.Month) == 0 {
      result.Month = i.Month
    }
    if len(result.Day) == 0 {
      result.Day = i.Day
    }
    if len(result.Disc) == 0 {
      result.Disc = i.Disc
    }
    if len(result.Track) == 0 {
      result.Track = i.Track
    }
    if len(result.Title) < len(i.Title) {
      result.Title = i.Title
    }

    return result, false
  }

  return i, true
}

func (i *info) fromAlbum(s string) {
  i.matchDiscOnly(s)
  s = i.matchDate(s)
  s = i.matchYearOnly(s)
  if len(i.Album) == 0 {
    i.Album = matchAlbumOrTitle(s)
  }
}

func (i *info) fromFile(s string) *info {
  s = i.matchDate(s)
  s = i.matchDiscTrack(s)
  i.Title = matchAlbumOrTitle(s)

  return i
}

func (i *info) fromPath(p, sep string) *info {
  // start inner-most folder, work out
  for _, s := range reverse(strings.Split(p, sep)) {
    i.fromAlbum(s)
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
func (i *info) matchYearOnly(s string) string {
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
func (i *info) matchDate(s string) string {
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

func (i *info) matchDiscTrack(s string) string {
  for _, regExpStr := range discTrackRegexps {
    m, r := regexpMatch(s, regExpStr)
    if len(m) == 0 {
      continue
    }

    // remove prefix 0s
    for x := range m {
      m[x] = regexp.MustCompile(`^0+`).ReplaceAllString(m[x], "")
    }

    i.Disc, i.Track = m[1], m[2]
    return r
  }
  return s
}

func (i *info) matchDiscOnly(s string) {
  m, _ := regexpMatch(s, `(?i)(cd|disc|set|disk)\s*(?P<disc>\d{1,2})\s*`)
  if len(m) >= 3 && len(i.Disc) == 0 {
    i.Disc = m[2]
  }
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

  // only these chars allowed: A-Za-z0-9-',.!?&> _()
  // remove all others
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

  return fixWhitespace(s)
}

// replace whitespaces with one space
func fixWhitespace(s string) string {
  return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}
