package albumart

import (
  "io"
  "os"
  "fmt"
  "image"
  "regexp"
  "strings"
  "io/ioutil"
  "path/filepath"
  _ "image/jpeg"
  _ "image/png"

  "github.com/JamTools/goff/ffprobe"
  "github.com/JamTools/goff/fsutil"
)

type AlbumArt struct {
  TempDir string
  Fullpath string
  WithParentDir bool
  Source string
  Ffmpeg interface {
    OptimizeAlbumArt(s, d string) (string, error)
    Exec(args ...string) (string, error)
  }
  Ffprobe interface {
    EmbeddedImage() (int, int, bool)
    GetData(filePath string) (*ffprobe.Data, error)
  }
  ImgDecode func (r io.Reader) (image.Config, string, error)
}

// uses optimized embedded artwork OR optimized artwork within file path
func Process(a *AlbumArt) (string, error) {
  // create temporary directory to work out of
  td, err := ioutil.TempDir("", "")
  if err != nil {
    return "", err
  }
  defer os.RemoveAll(td)
  a.TempDir = td

  // TODO: if folder.jpg, use (do not compress) as is (skip embedded)

  // probe to determine if has embedded artwork
  _, err = os.Stat(a.Fullpath)
  if err == nil {
    _, err = a.Ffprobe.GetData(a.Fullpath)
    if err != nil {
      return "", err
    }
  }

  // if file has embedded artwork, extract & optimize
  w, h, has := a.Ffprobe.EmbeddedImage()
  if has {
    err = a.embedded(w, h)
    if err != nil {
      fmt.Printf("\nNo embedded artwork found.\n")
    }
  }

  // find & optimize path images (if embedded not found)
  if len(a.Source) == 0 {
    err = a.fromPath()
    if err != nil {
      fmt.Printf("\nNo artwork image files found.\n")
    }
  }

  if a.WithParentDir && len(a.Source) == 0 {
    a.Source, err = a.processParentFolderArtwork()
    if err != nil {
      return a.Source, err
    }
  }

  return a.Source, nil
}

// extract & optimize embedded artwork
func (a *AlbumArt) embedded(width, height int) error {
  // extract image with ffmpeg
  src := filepath.Join(a.TempDir, "embedded-orig.jpg")
  _, err := a.Ffmpeg.Exec([]string{ "-y", "-i", a.Fullpath, src }...)
  if err != nil {
    return err
  }

  if width > 501 {
    // optimize image
    opt := filepath.Join(a.TempDir, "embedded.jpg")
    _, err = a.Ffmpeg.OptimizeAlbumArt(src, opt)
    if err != nil {
      return err
    }

    // use the smallest size
    r, _ := fsutil.NthFileSize([]string{src, opt}, true)

    // if optimized is smaller, copy original to folder-orig.jpg if larger
    if r == opt {
      err = a.copyAsFolderOrigJpg(src)
      if err != nil {
        return err
      }
    }

    src = r
  }

  err = a.copyAsFolderJpg(src)
  if err != nil {
    return err
  }

  return nil
}

// iterate to find best image, then optimize
func (a *AlbumArt) fromPath() error {
  // specific names that take precedence over file size
  matches := []string{
    `^(?i)folder\.jpg$`,
  }

  found := ""
  imgs := []string{}
  fd := filepath.Dir(a.Fullpath)
  images := fsutil.FilesImage(fd)

  // break if find specific match
  for i := range images {
    imgs = append(imgs, filepath.Join(fd, images[i]))
    for x := range matches {
      if regexp.MustCompile(matches[x]).MatchString(images[i]) {
        found = imgs[i]
        break
      }
    }
  }

  // if didn't find specific, try largest file size
  if len(imgs) > 0 && len(found) == 0 {
    found, _ = fsutil.NthFileSize(imgs, false)
  }
  if len(found) == 0 {
    return nil
  }

  // open image file and determine width/height
  file, err := os.Open(found)
  if err != nil {
    return err
  }
  img, _, err := a.ImgDecode(file)
  if err != nil {
    return err
  }

  // only optimize if width > 501
  if img.Width > 501 {
    // optimize through ffmpeg
    opt := filepath.Join(a.TempDir, "path.jpg")
    _, err := a.Ffmpeg.OptimizeAlbumArt(found, opt)
    if err != nil {
      return err
    }

    // use smallest size
    found, _ = fsutil.NthFileSize([]string{found, opt}, true)
  }

  err = a.copyAsFolderJpg(found)
  if err != nil {
    return err
  }

  return nil
}

// if parent folder does not contain audio files, copy any images files
// TODO: tests
func (a *AlbumArt) processParentFolderArtwork() (string, error) {
  // if parent folder exists
  pf := filepath.Dir(filepath.Dir(a.Fullpath))
  if len(filepath.Base(pf)) > 0 {

    // ensure parent folder contains no audio files
    for _, x := range fsutil.FilesAudio(pf) {
      ia := strings.Split(x, fsutil.PathSep)
      if len(ia) == 1 {
        return "", nil
      }
    }

    // if parent folder has images
    images := fsutil.FilesImage(pf)
    if len(images) > 0 {
      // copy images found with parent folder
      hasImage := false
      for _, y := range images {
        ia := strings.Split(y, fsutil.PathSep)
        if len(ia) == 1 {
          hasImage = true
          // copy ignoring any errors
          _ = fsutil.CopyFile(filepath.Join(pf, y),
            filepath.Join(filepath.Dir(a.Fullpath), filepath.Base(y)))
        }
      }

      // if image not set, try again with image files copied from parent dir
      if hasImage {
        return Process(a)
      }
    }
  }
  return "", nil
}

// copy src to folder-orig.jpg if larger
func (a *AlbumArt) copyAsFolderOrigJpg(src string) error {
  orig := filepath.Join(filepath.Dir(a.Fullpath), "folder-orig.jpg")
  if fsutil.IsLarger(src, orig) {
    err := fsutil.CopyFile(src, orig)
    if err != nil {
      return err
    }
  }
  return nil
}

// update folder.jpg & folder-orig.jpg
func (a *AlbumArt) copyAsFolderJpg(src string) error {
  folder := filepath.Join(filepath.Dir(a.Fullpath), "folder.jpg")

  // skip if src already is folder.jpg
  if src == folder {
    return nil
  }

  // copy folder.jpg to folder-orig.jpg if larger
  err := a.copyAsFolderOrigJpg(folder)
  if err != nil {
    return err
  }

  // copy to folder.jpg
  err = fsutil.CopyFile(src, folder)
  if err != nil {
    return err
  }

  a.Source = folder
  return nil
}
