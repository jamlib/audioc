package audioc

// process each audio file within bundle in separate cpu thread
func (a *audioc) processThreaded(indexes []int) (string, error) {
  var err error
  jobs := make(chan int)
  dir := make(chan string, a.Workers)

  // iterate through files sending them to worker processes
  go func() {
    for x := range indexes {
      if err != nil {
        break
      }
      jobs <- indexes[x]
    }
    close(jobs)
  }()

  // start worker processes
  for i := 0; i < a.Workers; i++ {
    go func() {
      // TODO return slice of metadata instead of single dir string
      // this allows full control over post processing
      var d string

      for job := range jobs {
        var e error
        d, e = a.processFile(job)

        // if single job errors, break & set shared err 
        if e != nil {
          err = e
          break
        }
      }

      dir <- d
    }()
  }

  // wait for all workers to finish
  var resultDir string
  for i := 0; i < a.Workers; i++ {
    resultDir = <-dir
  }

  return resultDir, err
}
