package main

// reverse order of string slice
func reverse(s []string) []string {
  for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
    s[i], s[j] = s[j], s[i]
  }
  return s
}
