# audioc

Clean up audio collection setting meta tags & embedding artwork

## Usage

```
Usage: audioc [MODE] [OPTIONS] PATH

Positional Args:
  PATH           directory path

MODE (specify only one):
  --artist "ARTIST" --album "ALBUM"
    treat as specific album belonging to specific artist

  --artist "ARTIST"
    treat as specific artist

  --collection
    treat as collection of artists

OPTIONS:
  --bitrate "BITRATE"
    V0 (default)
      convert to variable 256kbps mp3
    320
      convert to constant 320kbps mp3

  --fix
    fixes incorrect track length, ie 1035:36:51

  --force
    processes all files, even if path info matches tag info

  --write
    write changes to disk

Debug:
  --version
    print program version, then exit

```

## Purpose

This program is designed to process a music collection, keeping specified FLAC
audio files while converting all other audio formats to MP3.

Source albums have a release year or performance date. This date is then used
in both the album tag and folder path.

An example of the resulting folder structure:

```
Grateful Dead/
    1977/
        1977.05.15 St. Louis Arena, St. Louis, MO/
        1977 Terrapin Station/
```

In the above example, `1977.05.15 St. Louis Arena, St. Louis, MO` represents a
live performance, while `1977 Terrapin Station` represents a studio album. Both
belong to the artist `Grateful Dead`, nested within an additional folder
representing the year `1977`.

## Dependencies

This tool depends on `ffmpeg` and `ffprobe` binaries installed or included
within same folder, which are used to process the audio files and artwork.

Information on how to download `ffmpeg`:
[https://ffmpeg.org/download.html](https://ffmpeg.org/download.html)

The `metaflac` binary needs to be installed or included to support album
artwork embedding within FLAC files. If `metaflac` is not found, FLAC artwork
embedding will be skipped, but the program will continue without error.

The `metaflac` binary is part of the `flac` package.

Information on how to download `flac`:
[https://xiph.org/flac/download.html](https://xiph.org/flac/download.html)

## Mode

### Album (--artist "Artist Name" --album "Album Name")

Files nested within specified PATH are considered to be part of a specified
album or live performance belonging to a specified artist.

### Artist (--artist "Artist Name")

Child directories of specified PATH are considered to be albums or live
performances belonging to the specified artist.

### Collection (--collection)

Child directories of specified PATH are considered to be artists. Child
directories of each artist are considered to be albums or live performances
belonging to that artist.

The artist folder name overrides the audio file embedded artist metadata.

To skip processing a child directory, include ` - ` in its name. Such as:
`Grateful Dead - UNORGANIZED`

## Options

### Bitrate (--bitrate V0 OR --bitrate 320)

Convert other audio formats to MP3 using `libmp3lame` encoding and either V0
(variable 256kbps) or 320 (constant 320kbps) bitrate.

To skip converting FLAC audio to MP3, include ` - FLAC` at the end of the album
folder name.

### Fix (--fix)

Fixes incorrect track length (ie, 1035:36:51) affecting certain variable MP3
encodes by removing all metadata, then adding minimal metadata back in a
separate process.

### Force (--force)

Processes each audio file regardless of whether or not the path and file info
matches its tag info.

### Write (--write)

By not including `--write`, the process will run in simulation, printing all
changes to the console for review.

Including `--write` will apply changes by writing to disk. This process cannot
be undone.

## Developing

Instructions on how to: [Install Go and Dep](docs/INSTALL_GO_DEP.md)

### Building

Get latest source, run:

    go get github.com/jamlib/audioc

Navigate to source path, run:

    cd $GOPATH/src/github.com/jamlib/audioc

Ensure dependencies are installed and up-to-date with `dep`, run:

    dep ensure

From within source path, to build the binary, run:

    go install

To test by displaying usage, run:

    audioc --help

### Testing

From within source path, run:

    go test -cover -v ./...

### Contributing

Instructions on how to: [Submit a Pull Request](docs/SUBMIT_PR.md)

## License

This code is available open source under the terms of the
[MIT License](http://opensource.org/licenses/MIT).
