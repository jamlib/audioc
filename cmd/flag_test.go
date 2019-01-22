package main

import (
  "os"
  "testing"
)

func TestProcessFlagsVersion(t *testing.T) {
  os.Args = []string{"audioc", "-version"}

  if _, cont := configFromFlags(); cont == true {
    t.Errorf("Expected %v, got %v", false, cont)
  }
}

func TestProcessFlagsUsage(t *testing.T) {
  os.Args = []string{"audioc"}

  if _, cont := configFromFlags(); cont == true {
    t.Errorf("Expected %v, got %v", false, cont)
  }
}

func TestProcessFlagsNoArtistOrCollection(t *testing.T) {
  os.Args = []string{"audioc", "."}

  if _, cont := configFromFlags(); cont == true {
    t.Errorf("Expected %v, got %v", false, cont)
  }
}

func TestProcessFlagsArtist(t *testing.T) {
  os.Args = []string{"audioc", "--artist", "Grateful Dead", "."}

  c, cont := configFromFlags()
  if cont == false {
    t.Errorf("Expected %v, got %v", true, cont)
  }
  if c.Flags.Artist != os.Args[2] {
    t.Errorf("Expected %v, got %v", os.Args[2], c.Flags.Artist)
  }
}

func TestProcessFlagsCollection(t *testing.T) {
  os.Args = []string{"audioc", "--collection", "."}

  _, cont := configFromFlags()
  if cont == false {
    t.Errorf("Expected %v, got %v", true, cont)
  }
}
