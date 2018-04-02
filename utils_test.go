package main

import (
  "strings"
  "testing"
)

func TestReverse(t *testing.T) {
  tests := [][][]string{
    {
      { "1", "2", "3", "4", "5", "6", "7", "8", "9", "10" },
      { "10", "9", "8", "7", "6", "5", "4", "3", "2", "1" },
    }, {
      { "second", "hello", "goodbye", "first" },
      { "first", "goodbye", "hello", "second" },
    },
  }

  for i := range tests {
    r := reverse(tests[i][0])
    if strings.Join(r, "\n") != strings.Join(tests[i][1], "\n") {
      t.Errorf("Expected %v, got %v", tests[i][1], r)
    }
  }
}
