package resolve

import (
	"runtime"
	"sync"

	cflog "github.com/cfmleditor/clif/internal/log"
)

// forEachFile runs do for each file on a worker per processor, and returns
// once all are done. It runs on goroutines the scan's own recovery does not
// cover, so a panic on one file is recorded rather than ending the process,
// as recoveredHandlerParse does for a caller parse. do writes its answer at
// index i of a slice the caller owns, which keeps the caller's merge in file
// order whatever order the workers finish in.
//
// failed is the files whose do panicked, in file order. A caller must not
// read one's empty slot as a file with nothing in it: that is an answer the
// file never gave, and caller inference, which needs every caller to agree,
// would type an argument from the callers that were left.
func forEachFile(files []string, do func(i int, file string)) (failed []string) {
	var (
		wg     sync.WaitGroup
		broken = make([]bool, len(files))
	)

	work := make(chan int)

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for i := range work {
				broken[i] = !doRecovered(i, files[i], do)
			}
		})
	}

	for i := range files {
		work <- i
	}

	close(work)
	wg.Wait()

	for i, b := range broken {
		if b {
			failed = append(failed, files[i])
		}
	}

	return failed
}

// doRecovered runs do, and reports false when it panicked.
func doRecovered(i int, file string, do func(i int, file string)) (ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			cflog.Recovered(nil, "workspace file scan panic", rec, cflog.String("file", file))

			ok = false
		}
	}()

	do(i, file)

	return true
}
