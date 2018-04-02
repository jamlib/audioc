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

// irerate through dateRegexps returning first valid date found
func matchDate(s string) (year, mon, day, result string) {
  for i, regExpStr := range dateRegexps {
    m, remain := regexpMatch(s, regExpStr)
    if len(m) == 0 {
      continue
    }

    // order of matches depends on position within dateRegexps slice
    if i > 1 && i != 4 {
      // month day year
      year, mon, day = m[3], m[1], m[2]
    } else {
      // year month day
      year, mon, day = m[1], m[2], m[3]
    }
    mon = fmt.Sprintf("%02s", mon)
    day = fmt.Sprintf("%02s", day)

    // expand year to include century
    if len(year) == 2 {
      y, err := strconv.Atoi(year)
      if err != nil {
        continue
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

    v := validDate(year, mon, matchDay(day))
    if !v {
      continue
    }
    result = remain
    break
  }
  return year, mon, day, result
}

var discTrackRegexps = []string{
  // pattern: 's01t01', 'd01t01', 's1 01', 'd301', 'd1_01'
  `[sd](?P<disc>\d{2})[-. _t]*(?P<track>\d{2})`,
  `[sd](?P<disc>\d{1})[-. _t]*(?P<track>\d{2})`,
  `[sd](?P<disc>\d{1})[-. _t]*(?P<track>\d{1})`,
}

func matchDiscTrack(s string) (disc, track, result string) {
  for _, regExpStr := range discTrackRegexps {
    m, r := regexpMatch(s, regExpStr)
    if len(m) == 0 {
      continue
    }

    disc, track = m[1], m[2]
    result = r
    break
  }
  return disc, track, result
}

func regexpMatch(s, regExpStr string) ([]string, string) {
  m := regexp.MustCompile(regExpStr).FindStringSubmatch(s)

  if len(m) > 0 {
    i := strings.Index(s, m[0])
    s = s[(i+len(m[0])):]
  }

  return m, s
}
