package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	cflog "github.com/cfmleditor/clif/internal/log"
)

// startProfiling writes a CPU profile to CLIF_CPUPROFILE, a heap profile to
// CLIF_MEMPROFILE and an execution trace to CLIF_TRACE when they are set, so a
// subcommand can be profiled against a real workspace without a separate build.
// A trace is what shows time a CPU profile cannot: workers waiting on a lock or
// a system call (`go tool trace -pprof=sync|syscall|sched`). The returned
// function stops the CPU profile and the trace and writes the heap profile.
func startProfiling() func() {
	cpuPath := cflog.Getenv("CLIF_CPUPROFILE", "CFMLEDITOR_CPUPROFILE")
	memPath := cflog.Getenv("CLIF_MEMPROFILE", "CFMLEDITOR_MEMPROFILE")

	var cpu, tr *os.File

	if path := os.Getenv("CLIF_TRACE"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "trace: %v\n", err)
		} else if err := trace.Start(f); err != nil {
			fmt.Fprintf(os.Stderr, "trace: %v\n", err)

			_ = f.Close()
		} else {
			tr = f
		}
	}

	if cpuPath != "" {
		f, err := os.Create(cpuPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cpu profile: %v\n", err)
		} else if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "cpu profile: %v\n", err)

			_ = f.Close()
		} else {
			cpu = f
		}
	}

	return func() {
		if tr != nil {
			trace.Stop()

			_ = tr.Close()
		}

		if cpu != nil {
			pprof.StopCPUProfile()

			_ = cpu.Close()
		}

		if memPath == "" {
			return
		}

		f, err := os.Create(memPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "heap profile: %v\n", err)

			return
		}

		runtime.GC()

		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "heap profile: %v\n", err)
		}

		_ = f.Close()
	}
}
