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

func TestLastOfCurrentDir(t *testing.T) {
  paths := []string{
    "/anything/nest/file3.ext",
    "/anything/nest/file4.ext",
    "/anything/file1.ext",
    "/anything/file2.ext",
  }

  tests := []map[string]bool{
    { "anything/nest": false },
    { "anything/nest": true },
    { "anything": false },
    { "anything": true },
  }

  a := &audiocc{ DirEntry: "/anything", Files: paths }

  for i := range tests {
    for k, v := range tests[i] {
      a.DirCur = k
      r := a.lastOfCurrentDir(i)
      if r != v {
        t.Errorf("Expected %v, got %v", v, r)
      }

    }
  }
}

func TestLastOfCurrentDirBounds(t *testing.T) {
  a := &audiocc{ Files: []string{} }
  r := a.lastOfCurrentDir(1)
  if r != false {
    t.Errorf("Expected %v, got %v", false, r)
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
      flagCollection = v[0]
      defer func() {
        flagCollection = false
      }()

      r := skipArtistOnCollection(k)
      if r != v[1] {
        t.Errorf("Expected %v, got %v", v[1], r)
      }
    }
  }
}
