package main

import (
  "testing"
)

// TODO: fully test process()

func TestProcessDirDNE(t *testing.T) {
  a := &audiocc{ DirEntry: "audiocc-dir-def-dne" }
  err := a.process()
  if err == nil {
    t.Errorf("Expected error, got none.")
  }
}

func TestSkipArtistOnCollection(t *testing.T) {
  tests := []map[string][]bool{
    { "Jerry Garcia Band": { true, false } },
    { "Grateful Dead - Unorganized": { true, true } },
    { "Grateful Dead - Unorganized": { false, false } },
  }

  for i := range tests {
    for k, v := range tests[i] {
      flags.Collection = v[0]
      defer func() {
        flags.Collection = false
      }()

      r := skipArtistOnCollection(k)
      if r != v[1] {
        t.Errorf("Expected %v, got %v", v[1], r)
      }
    }
  }
}
