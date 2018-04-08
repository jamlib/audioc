package main

import (
  "io"
  "os"
  "fmt"
  "regexp"
  "image"
  "io/ioutil"
  "path/filepath"
  _ "image/jpeg"
  _ "image/png"
)

type artwork struct {
  TempDir string
  PathInfo *pathInfo
  Source string
  Ffmpeg interface {
    OptimizeAlbumArt(s, d string) (string, error)
    Exec(args ...string) (string, error)
  }
  Ffprobe interface {
    EmbeddedImage() (int, int, bool)
  }
  ImgDecode func (r io.Reader) (image.Config, string, error)
}

// uses optimized embedded artwork OR optimized artwork within file path
// TODO: tests
func (a *artwork) process() (string, error) {
  // create temporary directory to work out of
  td, err := ioutil.TempDir("", "")
  if err != nil {
    return "", err
  }
  defer os.RemoveAll(td)
  a.TempDir = td

  // if file has embedded artwork, extract & optimize
  w, h, has := a.Ffprobe.EmbeddedImage() 
  if has {
    err = a.embedded(w, h)
    if err != nil {
      fmt.Printf("Error: %v\n\n", err.Error())
    }
  }

  // find & optimize path images (if embedded not found)
  if len(a.Source) == 0 {
    err = a.fromPath()
    if err != nil {
      fmt.Printf("Error: %v\n\n", err.Error())
    }
  }

  return "", nil
}

// extract & optimize embedded artwork
// TODO: tests
func (a *artwork) embedded(width, height int) error {
  // extract image with ffmpeg
  src := filepath.Join(a.TempDir, "embedded-orig.jpg")
  _, err := a.Ffmpeg.Exec([]string{ "-y", "-i", a.PathInfo.Fullpath, src }...)
  if err != nil {
    return err
  }

  orig := filepath.Join(a.PathInfo.Fulldir, "folder-orig.jpg")

  if width > 501 {
    // optimize image
    opt := filepath.Join(a.TempDir, "embedded.jpg")
    _, err = a.Ffmpeg.OptimizeAlbumArt(src, opt)
    if err != nil {
      return err
    }

    // use the smallest size
    sources := []string{ src, opt }
    i, _ := nthFileSize(sources, true)

    // copy original to folder-orig.jpg if larger
    if i == 1 && isLarger(sources[0], orig) {
      err = copyFile(sources[0], orig)
      if err != nil {
        return err
      }
    }

    src = sources[i]
  }

  err = a.copyAsFolderJpg(src)
  if err != nil {
    return err
  }

  return nil
}

// iterate to find best image, then optimize
func (a *artwork) fromPath() error {
  // specific names that take precedence over file size
  matches := []string{
    `^(?i)folder\.jpg$`,
  }

  found := ""
  imgs := []string{}
  images := filesByExtension(a.PathInfo.Fulldir, imageExts)

  // break if find specific match
  for i := range images {
    imgs = append(imgs, filepath.Join(a.PathInfo.Fulldir, images[i]))
    for x := range matches {
      if regexp.MustCompile(matches[x]).MatchString(images[i]) {
        found = imgs[i]
        break
      }
    }
  }

  // if didn't find specific, try largest file size
  if len(imgs) > 0 && len(found) == 0 {
    i, _ := nthFileSize(imgs, false)
    found = imgs[i]
  }
  if len(found) == 0 {
    return nil
  }

  // only optimize if width > 501
  file, err := os.Open(found)
  if err != nil {
    return err
  }
  img, _, err := a.ImgDecode(file)
  if err != nil {
    return err
  }

  if img.Width > 501 {
    // optimize image
    opt := filepath.Join(a.TempDir, "path.jpg")
    _, err := a.Ffmpeg.OptimizeAlbumArt(found, opt)
    if err != nil {
      return err
    }

    // use smallest size
    sources := []string{ found, opt }
    i, _ := nthFileSize(sources, true)
    found = sources[i]
  }

  err = a.copyAsFolderJpg(found)
  if err != nil {
    return err
  }

  return nil
}

// update folder.jpg & folder-orig.jpg
func (a *artwork) copyAsFolderJpg(src string) error {
  orig := filepath.Join(a.PathInfo.Fulldir, "folder-orig.jpg")
  folder := filepath.Join(a.PathInfo.Fulldir, "folder.jpg")

  if src == folder {
    return nil
  }

  // copy folder.jpg to folder-orig.jpg if larger
  if isLarger(folder, orig) {
    err := copyFile(folder, orig)
    if err != nil {
      return err
    }
  }

  // copy to folder.jpg
  err := copyFile(src, folder)
  if err != nil {
    return err
  }

  return nil
}
