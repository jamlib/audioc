# audiocc

Audio Collection Cleanup

## Usage

```
Usage: audiocc [OPTIONS] PATH

Positional Args:
  PATH           directory path

Mode (specify only one):
  --artist "NAME"
    treat as specific artist

  --collection
    treat as collection of artists

Options:
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

This program is designed to process a music collection where the albums have a release year
or performance date. This date is then used in both the album tag and folder path.

An example of the resulting folder structure:

```
Grateful Dead/
    1977/
        1977.05.15 St. Louis Arena, St. Louis, MO/
        1977 Terrapin Station/
```

In the above example, `1977.05.15 St. Louis Arena, St. Louis, MO` represents a live performance, while
`1977 Terrapin Station` represents a studio album. Both belong to the artist `Grateful Dead`.

The program also converts audio formats to FLAC or MP3 and optimizes and embeds album artwork
within each audio file.

This results in a app-friendly, organized and visual audio collection sorted by date within both app
library and folder views.

Artists who release SBD (soundboard) quality audio of their live performances are the best fit for
this program.

Converting large file size lossless audio to V0 quality MP3 results in small file size yet
high quality sound, which is ideal for mobile devices using portable bluetooth speakers.

## Dependencies

This tool depends on `ffmpeg` and `ffprobe` binaries installed or included within same folder, 
which are used to process the audio and artwork.

The `metaflac` binary needs to be installed or included to support album artwork embedding, but only
within FLAC files. MP3 artwork embedding is included with `ffmpeg`.

If `metaflac` not found, FLAC embedding will be skipped, but the program will continue without error.

## Mode

### Artist (--artist "Artist Name")

Child directories of specified PATH, as well as files within PATH itself, are considered to be albums
or live performances belonging to the specified artist.

### Collection (--collection)

Immediate child directories of specified PATH are considered to be artists. Child directories of each
artist are considered to be albums or live performances belonging to that artist.

In the event that the artist tag is not found, the artist folder name is used.

To skip processing a child directory, include ` - ` in its name. Such as: `Grateful Dead - UNORGANIZED`

## Options

### Bitrate (--bitrate V0 OR --bitrate 320)

Convert other audio formats to MP3 using `libmp3lame` encoding and either V0 (variable 256kbps) or 320
(constant 320kbps) bitrate.

To skip converting FLAC audio, include ` - FLAC` at the end of the album folder name.

### Fix (--fix)

Fixes incorrect track length (ie, 1035:36:51) by removing all metadata, then adding metadata
back in separate process.

### Force (--force)

Processes each audio file regardless of whether or not the path info matches tag info.

### Write (--write)

By not including `--write`, the process will run in simulation, printing all changes to
the console for review.

Including `--write` will apply changes to disk.

## Developing

[Install Go and Dep](docs/INSTALL_GO_DEP.md).

### Building

Get latest source, run:

    go get github.com/JamTools/audiocc

Navigate to source path, run:

    cd $GOPATH/src/github.com/JamTools/audiocc

Ensure dependencies are installed and up-to-date with `dep`, run:

    dep ensure

From within source path, to build the binary, run:

    go install

To test by displaying usage, run:

    audiocc --help

### Testing

From within source path, run:

    go test -cover -v ./...

### Contributing

[Submit a Pull Request](docs/SUBMIT_PR.md).

## License

This code is available open source under the terms of the [MIT License](http://opensource.org/licenses/MIT).
