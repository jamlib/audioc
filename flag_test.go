package main

import (
  "os"
  "testing"
)

func TestProcessFlagsVersion(t *testing.T) {
  os.Args = []string{"audioc", "-version"}
  defer func() { flags.Version = false }()

  if _, cont := processFlags(); cont == true {
    t.Errorf("Expected %v, got %v", false, cont)
  }
}

func TestProcessFlagsUsage(t *testing.T) {
  os.Args = []string{"audioc"}

  if _, cont := processFlags(); cont == true {
    t.Errorf("Expected %v, got %v", false, cont)
  }
}

func TestProcessFlagsNoArtistOrCollection(t *testing.T) {
  os.Args = []string{"audioc", "."}

  if _, cont := processFlags(); cont == true {
    t.Errorf("Expected %v, got %v", false, cont)
  }
}

func TestProcessFlagsArtist(t *testing.T) {
  os.Args = []string{"audioc", "--artist", "Grateful Dead", "."}
  defer func() { flags.Artist = "" }()

  _, cont := processFlags()
  if cont == false {
    t.Errorf("Expected %v, got %v", true, cont)
  }
  if flags.Artist != os.Args[2] {
    t.Errorf("Expected %v, got %v", os.Args[2], flags.Artist)
  }
}

func TestProcessFlagsCollection(t *testing.T) {
  os.Args = []string{"audioc", "--collection", "."}
  defer func() { flags.Collection = false }()

  _, cont := processFlags()
  if cont == false {
    t.Errorf("Expected %v, got %v", true, cont)
  }
}
