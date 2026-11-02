package resolve

import (
	"runtime"
	"sync"

	cflog "github.com/cfmleditor/clif/internal/log"
)

// forEachFile runs do for each file on a worker per processor, and returns
// once all are done. It runs on goroutines the scan's own recovery does not
// cover, so a panic on one file is recorded and that file skipped rather than
// ending the process, as recoveredHandlerParse does for a caller parse. do
// writes its answer at index i of a slice the caller owns, which keeps the
// caller's merge in file order whatever order the workers finish in.
func forEachFile(files []string, do func(i int, file string)) {
	var wg sync.WaitGroup

	work := make(chan int)

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for i := range work {
				doRecovered(i, files[i], do)
			}
		})
	}

	for i := range files {
		work <- i
	}

	close(work)
	wg.Wait()
}

func doRecovered(i int, file string, do func(i int, file string)) {
	defer func() {
		if rec := recover(); rec != nil {
			cflog.Recovered(nil, "workspace file scan panic", rec, cflog.String("file", file))
		}
	}()

	do(i, file)
}
