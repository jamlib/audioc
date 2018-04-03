// info can be from:
// id3 tag, path, filename

package main

import (
  "fmt"
  "time"
  "regexp"
  "strconv"
  "strings"
)

type info struct {
  Year, Month, Day string
  Disc, Track string
}

func infoFromFile(file string) (*info, string) {
  i := &info{}
  file = i.matchDate(file)
  file = i.matchDiscTrack(file)

  return i, file
}

func infoFromPath(p, sep string) {
  pathArray := strings.Split(p, sep)
  for _, s := range reverse(pathArray) {
    if len(s) == 0 {
      continue
    }
    fmt.Printf("%v\n", s)
  }
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

// strip off multiple days or day range
func matchDay(d string) string {
  return regexp.MustCompile(`\d{1,2}`).FindString(d)
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
  i.Year = m[1]
  return remain
}

// irerate through dateRegexps returning first valid date found
func (i *info) matchDate(s string) string {
  for index, regExpStr := range dateRegexps {
    m, remain := regexpMatch(s, regExpStr)
    if len(m) == 0 {
      continue
    }

    // order of matches depends on position within dateRegexps slice
    if index > 1 && index != 4 {
      // month day year
      i.Year, i.Month, i.Day = m[3], m[1], m[2]
    } else {
      // year month day
      i.Year, i.Month, i.Day = m[1], m[2], m[3]
    }
    i.Month = fmt.Sprintf("%02s", i.Month)
    i.Day = fmt.Sprintf("%02s", i.Day)

    // expand year to include century
    if len(i.Year) == 2 {
      y, err := strconv.Atoi(i.Year)
      if err != nil {
        continue
      }

      // compare with current year to determine prefix
      nowYear := strconv.Itoa(time.Now().Year())
      l, r := nowYear[:2], nowYear[2:]
      ri, _ := strconv.Atoi(r)

      if y > ri {
        li, _ := strconv.Atoi(l)
        i.Year = strconv.Itoa(li-1) + i.Year
      } else {
        i.Year = l + i.Year
      }
    }

    v := validDate(i.Year, i.Month, matchDay(i.Day))
    if !v {
      continue
    }
    return remain
  }
  return s
}

var discTrackRegexps = []string{
  // pattern:^ '1-01 ', '01-02 ', '1-3 - '
  `^(?P<disc>\d{1,2})-(?P<track>\d{1,2})\s{1}[-]*\s*`,
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

    i.Disc, i.Track = m[1], m[2]
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
