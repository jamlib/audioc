package main

import (
  "io"
  "os"
  "io/ioutil"
  "encoding/json"

  "github.com/JamTools/goff/ffmpeg"
)

type mockFfmpeg struct {
  Embedded string
}

func (m *mockFfmpeg) OptimizeAlbumArt(s, d string) (string, error) {
  // temp file for optimizing
  tmp, err := ioutil.TempFile("", "")
  if err != nil {
    return "", err
  }
  defer os.Remove(tmp.Name())
  defer tmp.Close()

  b, err := ioutil.ReadFile(s)
  if err != nil {
    return "", err
  }

  // can make smaller
  contents := string(b)
  if len(contents) > 0 {
    _, err = io.WriteString(tmp, contents[:len(contents)-1])
    if err != nil {
      return "", err
    }
    // use instead of original source
    s = tmp.Name()
  }

  err = copyFile(s, d)
  if err != nil {
    return "", err
  }
  return "", nil
}

func (m *mockFfmpeg) Exec(args ...string) (string, error) {
  // hook on extract audio
  if len(args) == 4 {
    err := ioutil.WriteFile(args[3], []byte(m.Embedded), 0644)
    if err != nil {
      return "", err
    }
  }
  return "", nil
}

func (m *mockFfmpeg) ToMp3(i, q string, meta ffmpeg.Metadata, o string) (string, error) {
  a := struct {
    Input, Quality, Output string
    Meta *ffmpeg.Metadata
  }{ i, q, o, &meta }

  b, err := json.Marshal(a)
  if err != nil {
    return "", err
  }

  err = ioutil.WriteFile(o, b, 0644)
  if err != nil {
    return "", err
  }

  return o, nil
}

type mockFfprobe struct {
  Width int
  Embedded string
}

func (m *mockFfprobe) EmbeddedImage() (int, int, bool) {
  if len(m.Embedded) > 0 {
    return m.Width, 0, true
  }
  return 0, 0, false
}