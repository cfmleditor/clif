// Package main is the entry point for the clif server.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/confine"
	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/formatter"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/language"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/server"
	"github.com/cfmleditor/clif/internal/vfs"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/uri"
)

var version = "dev"

func main() {
	stopProfiling := startProfiling()
	defer stopProfiling()

	reportUntrustedConfigs()

	// Subcommand routing
	if len(os.Args) > 1 {
		name, args := os.Args[1], os.Args[2:]

		if cmd, ok := subcommands()[name]; ok {
			if wantsHelp(args) {
				fmt.Print(cmd.usage)

				return
			}

			cmd.run(args)

			return
		}

		switch name {
		// --version and -version (CFLint's spelling) as well as the subcommand:
		// a tool checking that a binary is installed runs `<binary> --version`,
		// and without these that started the language server, which waits on
		// stdin for a client that never comes. Editors start the server with no
		// arguments, so no client sends one of these.
		case "version", "--version", "-version", "-v":
			fmt.Printf("clif %s\n", version)

			return
		case "help", "--help", "-h":
			// "help <command>" is that command's --help.
			if len(args) > 0 {
				if cmd, ok := subcommands()[args[0]]; ok {
					fmt.Print(cmd.usage)

					return
				}
			}

			printHelp()

			return
		}
	}

	// Default: run LSP server
	runServer()
}

func printHelp() {
	fmt.Printf(`clif %s

Commands:
  (default)    Run the LSP server over stdio
  parse        Parse CFML files and report timing
  scan         Scan CFML files and report parse errors
  format       Format CFML files (stdout or in-place with -w)
  unresolved   Scan for unresolved component/method calls
  cflint       Run CFLint over files, a project, or what is staged or changed
  suppressions Count CFLint suppressions by rule, and fail if they rise
  hook         Install and run a git pre-commit hook that runs CFLint
  refs         Find references to a component or function
  deps         Print component dependency info
  graph        Build a whole-project map of functions and their connections
  mcp          Serve that map over the Model Context Protocol on stdio
  routes       Report framework routes and what they resolve to
  explain      Explain how a call site's component was resolved
  version      Print version (also --version)
  help         Show this help

Run "clif <command> --help" (or "clif help <command>")
for a command's usage and options.
`, version)
}

func runServer() {
	cwd, _ := os.Getwd()
	cfg, _ := daemon.FindConfig(cwd)

	debug := cfg != nil && cfg.Debug()
	log := cflog.NewLogger(debug)

	// Before anything that can crash: every crash record goes to a file from
	// here on, configured or not. See EnableCrashReports.
	crashPath := cflog.EnableCrashReports(version)

	// Any panic that gets past the per-request handlers is recorded before the
	// process goes. Those handlers recover where recovery is right; one reaching
	// here is one nothing expected, and without this the only trace is a stack on
	// a stderr the client may already have stopped reading.
	defer cflog.CapturePanic("runServer")()

	// Through the logger rather than straight to stderr, which is where this
	// used to go. An LSP client has no way to know that a line on a server's
	// stderr is routine: vscode-languageclient logs all of it at error level, so
	// a healthy startup announced itself as "[error] cfmleditor-lsp dev" and the
	// one banner meant to reassure anyone reading the Output panel was the only
	// thing in it that looked broken.
	//
	// It stays first, before anything that can fail, because the version is what
	// a crash report needs most and a later line might never be reached.
	log.Info("clif starting",
		cflog.String("version", version),
		cflog.Int("pid", os.Getpid()),
		cflog.String("cwd", cwd))

	if path := cflog.FilePath(); path != "" {
		log.Info("logging to file", cflog.String("path", path))
	}

	if crashPath != "" {
		log.Info("crash reports go to", cflog.String("path", crashPath))
	}

	if debug {
		log.Info("debug mode enabled")
	}

	if cfg != nil {
		sock := cfg.SocketPath()

		// Try to connect to an existing daemon
		proxyErr := daemon.Proxy(sock)
		if proxyErr == nil {
			return
		}

		// No daemon running, or a socket that is not this user's: become the
		// daemon and serve this client over stdio
		log.Info("starting daemon mode", cflog.String("socket", sock), cflog.Err(proxyErr))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sharedIndex := index.New()
		ct := daemon.NewConnTracker()
		// One settings value configures every session, whether it arrives over
		// stdio here or over the socket later. Keeping the two in step by hand
		// is what dropped expressionMappings and servicePropertyResolvers from
		// every editor after the first; building it in daemon.SettingsFrom
		// rather than here is what lets a test check that nothing is missing.
		settings := daemon.SettingsFrom(cfg)

		go func() {
			// A daemon that cannot serve its socket still serves this editor
			// over stdio; later editors start sessions of their own.
			if err := daemon.Serve(ctx, sock, log, sharedIndex, ct, &settings); err != nil {
				log.Warn("daemon socket not served", cflog.String("socket", sock), cflog.Err(err))
			}
		}()

		// Serve this editor session over stdio with the shared index
		ct.Add()

		stream := jsonrpc2.NewStream(vfs.Stdio())
		conn := jsonrpc2.NewConn(stream)
		srv := server.NewServer(conn, log, sharedIndex)
		srv.Version = version
		settings.Apply(srv)
		conn.Go(ctx, srv.Handler())

		go func() {
			select {
			case <-conn.Done():
			case <-ctx.Done():
				_ = os.Stdin.Close()

				<-conn.Done()
			}

			ct.Remove()
		}()

		// Shut down when all clients have disconnected — but not on a zero
		// that a new client is already arriving into. Closing one editor
		// window while another opens leaves exactly that, and taking it as
		// final killed the arriving client: it was accepted and counted, then
		// cut off by a shutdown it could not cancel. See ConnTracker.
		daemon.WaitForLastClient(ctx, ct, daemon.DrainGrace)
		cancel()

		return
	}

	// No .clif.json anywhere above the working directory, so there is no
	// project to run a daemon for: a standalone session with its own index.
	// The editor's workspace roots are searched for a config again at
	// initialize, so a project config the walk above could not reach — the
	// editor was started from elsewhere — still configures this session; it
	// just does not put it in a daemon named after whatever directory the
	// process happened to start in.
	log.Info("no .clif.json found; starting standalone")

	stream := jsonrpc2.NewStream(vfs.Stdio())
	conn := jsonrpc2.NewConn(stream)
	srv := server.NewServer(conn, log)
	srv.Version = version
	conn.Go(context.Background(), srv.Handler())
	<-conn.Done()
}

func cmdParse(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, parseUsage)
		os.Exit(1)
	}

	var files []string

	for _, arg := range args {
		info, err := os.Stat(positional(arg, parseUsage))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			os.Exit(1)
		}

		if info.IsDir() {
			files = append(files, cfmlFilesUnder(arg)...)
		} else {
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no CFML files found\n")
		os.Exit(1)
	}

	var totalDur time.Duration

	var totalFuncs, totalRefs, totalFiles int

	for _, f := range files {
		content, err := vfs.ReadRegular(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", f, err)

			continue
		}

		if cfpath.IsBinary(content) {
			continue
		}

		absPath, _ := filepath.Abs(f)
		fileURI := uri.URI("file://" + absPath)
		start := time.Now()
		pr := parser.Parse(fileURI, string(content))
		dur := time.Since(start)

		totalDur += dur
		totalFuncs += len(pr.Funcs)
		totalRefs += len(pr.ComponentRefs)
		totalFiles++

		fmt.Printf("  %s  funcs=%d refs=%d scopes=%d  %v\n", f, len(pr.Funcs), len(pr.ComponentRefs), len(pr.Scopes), dur)
	}

	avg := time.Duration(0)
	if totalFiles > 0 {
		avg = totalDur / time.Duration(totalFiles)
	}

	fmt.Printf("\n  total: %d files, %d funcs, %d refs in %v (avg %v/file)\n",
		totalFiles, totalFuncs, totalRefs, totalDur, avg)
}

func cmdFormat(args []string) {
	var (
		write              bool
		allowNonWhitespace bool
		followSymlinks     bool
		configRoot         string
		files              []string
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-w":
			write = true
		case "--allow-non-whitespace":
			allowNonWhitespace = true
		case "--follow-symlinks":
			followSymlinks = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: --root needs a directory\n")
				os.Exit(1)
			}

			configRoot = args[i+1]
			i++
		default:
			files = append(files, positional(args[i], formatUsage))
		}
	}

	if len(files) == 0 {
		fmt.Fprint(os.Stderr, formatUsage)
		os.Exit(1)
	}

	optionsFor := formatOptionsFor(configRoot, allowNonWhitespace)
	links := symlinkRule{follow: followSymlinks, root: configRoot}

	failed := false

	for _, f := range files {
		if err := formatOneFile(f, new(optionsFor(f)), write, links); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", f, err)

			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

// formatOptionsFor returns a lookup that maps a file to the formatter options
// its governing .clif.json asks for, memoised per config directory.
//
// The subcommand used to format from formatter.DefaultOptions() alone, so it
// ignored every key under "formatting" and produced different bytes than the
// editor did for the same file. Config is discovered from each file's own
// directory upwards, matching how the LSP picks a config, unless --root names
// one explicitly (same semantics as `explain --root`).
func formatOptionsFor(configRoot string, allowNonWhitespace bool) func(path string) formatter.Options {
	cache := make(map[string]formatter.Options)

	return func(path string) formatter.Options {
		dir := configRoot

		if dir == "" {
			abs, err := filepath.Abs(path)
			if err != nil {
				abs = path
			}

			dir = filepath.Dir(abs)
		}

		if opts, ok := cache[dir]; ok {
			return opts
		}

		fmtCfg := config.DefaultResolvedFormatting()
		if cfg, err := daemon.FindConfig(dir); err == nil && cfg != nil {
			fmtCfg = cfg.ResolvedFormatting()
		}

		opts := fmtCfg.FormatterOptions()

		// The flag can only loosen the guard, never tighten it away: a config
		// that turns whitespaceOnly off has already accepted the risk.
		if allowNonWhitespace {
			opts.WhitespaceOnly = false
		}

		opts.ParseScript = func(src []byte) *sitter.Tree {
			return language.Parse(language.CFScript, src, nil)
		}
		opts.ParseQuery = func(src []byte) *sitter.Tree {
			return language.Parse(language.CFQuery, src, nil)
		}
		opts.ParseCFML = func(src []byte) *sitter.Tree {
			return language.Parse(language.CFML, src, nil)
		}

		cache[dir] = opts

		return opts
	}
}

// formatOneFile formats a single file, writing it back in place when write is
// set. A file is only ever rewritten after Format returned successfully, so a
// refused format leaves the original untouched.
func formatOneFile(path string, opts *formatter.Options, write bool, links symlinkRule) error {
	content, err := vfs.ReadRegular(path)
	if err != nil {
		return err
	}

	tree := language.Parse(language.CFML, content, nil)
	defer tree.Close()

	// Refuse a file the grammar could not parse, exactly as the LSP handler
	// does. Formatting an incomplete CST is how a grammar gap turns into
	// deleted source, and the whitespaceOnly guard does not catch every shape
	// of that (see .ai/docs/FORMATTER-ISSUES.md).
	if err := formatter.ParseError(tree, content); err != nil {
		return err
	}

	out, err := formatter.Format(content, tree, opts)
	if err != nil {
		return err
	}

	if !write {
		_, _ = os.Stdout.Write(out)

		return nil
	}

	// Rewriting a file the formatter left alone costs nothing but a bumped
	// mtime, which is enough to wake every file watcher and rebuild watching
	// the tree.
	if bytes.Equal(content, out) {
		return nil
	}

	if err := writeFileInPlace(path, content, out, links); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "formatted %s\n", path)

	return nil
}

// writeFileInPlace replaces path's contents with out, preserving the file's
// permission bits and leaving the original intact if the write fails partway.
//
// os.WriteFile would do neither: it hardcodes the new mode, so a 0600 or 0755
// file silently became 0644, and it truncates in place, so an error partway
// through leaves a half-written source file with no copy of the original
// anywhere. Writing a sibling temp file and renaming it over the target makes
// the replacement atomic for any concurrent reader.
func writeFileInPlace(path string, original, out []byte, links symlinkRule) error {
	// A symlinked file should have its target rewritten; renaming onto the link
	// itself would replace the link with a regular file.
	target := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		target = resolved
	}

	if err := links.check(path, target); err != nil {
		return err
	}

	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".fmt-*")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()

	// Harmless once the rename succeeded, and the only cleanup on every path
	// that does not.
	defer func() { _ = os.Remove(tmpName) }()

	if err := writeAndSync(tmp, out); err != nil {
		return err
	}

	// CreateTemp always makes the file 0600, so the mode has to be restored
	// from the file being replaced rather than inherited.
	if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		return err
	}

	// Re-read rather than trusting the content from before Format ran: a write
	// landing on top of an edit made in the meantime would discard it silently.
	if current, err := vfs.ReadRegular(target); err == nil && !bytes.Equal(current, original) {
		return errors.New("file changed on disk while formatting, not overwriting")
	}

	return os.Rename(tmpName, target)
}

// symlinkRule says where format -w may rewrite a file it reached through a
// symlink.
//
// Following every link meant a cloned repository could commit x.cfm as a link
// to ~/.zshrc or a file in another project, and formatting the tree rewrote
// that file. A link is therefore followed only when its target lies in the
// link's own directory or beneath it, or beneath --root when one is given:
// that keeps a project's own links (a shared template linked into a sibling
// folder under --root, a renamed file kept under its old name) and refuses
// one that leaves. --follow-symlinks follows every link, as before.
type symlinkRule struct {
	follow bool   // --follow-symlinks
	root   string // --root, "" when not given
}

// check refuses to rewrite target, which path resolved to, when path is a
// symlink the rule does not follow.
func (r symlinkRule) check(path, target string) error {
	if r.follow {
		return nil
	}

	// writeFileInPlace stats the target next and reports a missing file, so
	// a failed Lstat is left to it.
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
		return nil //nolint:nilerr // see above
	}

	roots := confine.Roots(filepath.Dir(path), r.root)
	if confine.Within(target, roots) {
		return nil
	}

	return fmt.Errorf("is a symlink to %s, outside %s; pass --follow-symlinks to rewrite it", target, strings.Join(roots, " and "))
}

// writeAndSync writes out to f and flushes it to disk before closing.
//
// Without the fsync, a crash shortly after the rename can leave the target
// visible but empty, since the rename is durable before the data is.
func writeAndSync(f *os.File, out []byte) error {
	if _, err := f.Write(out); err != nil {
		_ = f.Close()

		return err
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()

		return err
	}

	return f.Close()
}

// reportUntrustedConfigs says on stderr, once per file, that a config file was
// skipped because another user could have written it (config.TrustedCandidates).
// Without it a planted file is ignored silently, and so is a project's own
// config with the wrong owner or mode, which reads as clif ignoring it.
func reportUntrustedConfigs() {
	var seen sync.Map

	config.ReportUntrustedTo(func(path, reason string) {
		if _, dup := seen.LoadOrStore(path, true); !dup {
			fmt.Fprintf(os.Stderr, "clif: ignoring %s: %s\n", path, reason)
		}
	})
}
