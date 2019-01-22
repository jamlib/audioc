package audioc

import (
  "os"
  "fmt"
  "regexp"
  "strings"
  "path/filepath"

  "github.com/jamlib/libaudio/ffmpeg"
  "github.com/jamlib/libaudio/fsutil"
  "github.com/jamlib/audioc/metadata"
)

func (a *audioc) processFile(index int) (*metadata.Metadata, error) {
  m := metadata.New(a.DirEntry, a.Files[index])

  // if --artist mode, artist is set from flag
  if a.Flags.Artist != "" {
    m.Info.Artist = a.Flags.Artist
  }

  // if --collection mode, artist set from parent folder name
  if a.Flags.Collection {
    m.Info.Artist = strings.Split(a.Files[index], fsutil.PathSep)[0]
  }

  // if --album mode, album is set from flag
  if a.Flags.Album != "" {
    m.Info.Album = m.Info.MatchCleanAlbum(a.Flags.Album)
  }

  // call Probe after setting m.Info.Artist
  err := m.Probe(a.Ffprobe)
  if err != nil {
    return m, err
  }

  // skip if sources match (unless --force)
  if m.Match && !a.Flags.Force {
    m.Resultpath = a.Files[index]
    return m, nil
  }

  // build resulting path
  fpa := strings.Split(a.Files[index], fsutil.PathSep)

  // if --collection or artist/year folder in expected place
  if a.Flags.Collection ||
    (len(fpa) > 2 && fpa[0] == m.Info.Artist && fpa[1] == m.Info.Year) {

    m.Resultpath = filepath.Join(m.Info.Artist, m.Info.Year)
  } else {
    // strip out innermost dirs that are irrelevant (ie, cd1)
    foundAlbum := false
    for x := len(m.Stripped)-1; x >= 0; x-- {
      if m.Stripped[x] != "" {
        if foundAlbum {
          m.Resultpath = fpa[x] + fsutil.PathSep + m.Resultpath
        } else {
          foundAlbum = true
        }
      }
    }
  }

  // append album name as directory
  m.Resultpath = filepath.Join(m.Resultpath, m.Info.ToAlbum(), m.Info.ToFile())

  fp := filepath.Join(a.DirEntry, a.Files[index])

  // print changes to be made
  p := fmt.Sprintf("\n%v\n", fp)
  if !m.Match {
    p += fmt.Sprintf("  * update tags: %#v\n", m.Info)
  }

  // convert audio (if necessary) & update tags
  ext := strings.ToLower(filepath.Ext(a.Files[index]))
  if ext != ".flac" || !skipConvert(a.Files[index]) {
    // convert to mp3
    m.Resultpath += ".mp3"
    p += fmt.Sprintf("  * convert to MP3 (%s)\n", a.Flags.Bitrate)

    _, err := a.processMp3(fp, m.Info)
    if err != nil {
      return m, err
    }
  } else {
    // TODO: use metaflac to edit flac metadata & embedd artwork
    m.Resultpath += ".flac"
    p += fmt.Sprintf("\n*** Flac processing with 'metaflac' not yet implemented.\n")
  }

  // compare processed to current path
  if a.Files[index] != m.Resultpath {
    p += fmt.Sprintf("  * rename to: %v\n", filepath.Join(a.DirEntry, m.Resultpath))
  }

  // print to console all at once
  fmt.Printf(p)

  // resultPath is a directory
  return m, nil
}

// skip converting if folder contains ' - FLAC'
func skipConvert(file string) bool {
  if regexp.MustCompile(` - FLAC$`).FindString(filepath.Dir(file)) == "" {
    return false
  }
  return true
}

func (a *audioc) processMp3(f string, i *metadata.Info) (string, error) {
  // skip if not writing
  if !a.Flags.Write {
    return "", nil
  }

  // if already mp3, copy stream; do not convert
  quality := a.Flags.Bitrate
  if strings.ToLower(filepath.Ext(f)) == ".mp3" {
    quality = "copy"
  }

  // TODO: specify lower bitrate if source file is of low bitrate

  // build metadata from tag info
  ffmeta := ffmpeg.Metadata{ Artist: i.Artist, Album: i.ToAlbum(),
    Disc: i.Disc, Track: i.Track, Title: i.Title, Artwork: a.Image }

  // save new file to Workdir subdir within current path
  newFile := filepath.Join(a.Workdir, i.ToFile() + ".mp3")

  // process or convert to mp3
  c := &ffmpeg.Mp3Config{ f, quality, newFile, ffmeta, a.Flags.Fix }
  _, err := a.Ffmpeg.ToMp3(c)
  if err != nil {
    return newFile, err
  }

  // ensure output file was written
  fi, err := os.Stat(newFile)
  if err != nil {
    return newFile, err
  }

  // ensure resulting file has size
  // TODO: ensure resulting file is good by reading & comparing metadata
  if fi.Size() <= 0 {
    return newFile, fmt.Errorf("File didn't have size")
  }

  file := filepath.Join(filepath.Dir(f), i.ToFile() + ".mp3")

  // delete original
  err = os.Remove(f)
  if err != nil {
    return file, err
  }

  // move new to original directory
  err = os.Rename(newFile, file)
  return file, err
}
