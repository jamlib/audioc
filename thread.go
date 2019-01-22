package audioc

import (
  "github.com/jamlib/audioc/metadata"
)

// process each audio file within bundle in separate cpu thread
func (a *audioc) processThreaded(indexes []int) ([]*metadata.Metadata, error) {
  var err error
  jobs := make(chan int)
  data := make(chan []metadata.Metadata, a.Workers)

  // iterate through files sending them to worker processes
  go func() {
    for x := range indexes {
      if err != nil {
        break
      }
      jobs <- indexes[x]
    }

    // close jobs channel once out of indexes
    close(jobs)
  }()

  // start worker processes
  for i := 0; i < a.Workers; i++ {
    go func() {
      // build metadata slice
      md := make([]metadata.Metadata, 0, len(indexes))

      for job := range jobs {
        m, e := a.processFile(job)
        md = append(md, *m)

        // if single job errors, break & set shared err 
        if e != nil {
          err = e
          break
        }
      }

      data <- md
    }()
  }

  // wait for all workers to finish
  results := make([]*metadata.Metadata, 0, len(indexes))
  for i := 0; i < a.Workers; i++ {
    workerMetadata := <-data
    for x := range workerMetadata {
      results = append(results, &workerMetadata[x])
    }
  }
  close(data)

  return results, err
}
