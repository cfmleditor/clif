package resolve

// forCaller binds mappings to the application's execution context. A library's
// nearest Application.cfc governs direct requests to that directory, not method
// calls from another application. Keep physical directories for relative lookup
// and injection policy, but retain the caller's mappings across method/return
// traversal. With no caller evidence, ordinary physical lookup still applies.
func (r *Resolver) forCaller(baseDir string) *Resolver {
	if r.indexer != nil {
		return r
	}

	root := r.FindApplicationRoot(baseDir)

	r.mu.RLock()
	view := r.contextViews[root]
	r.mu.RUnlock()

	if view != nil {
		return view
	}

	view = &Resolver{
		NativeEngine:         r.NativeEngine,
		Discovery:            r.Discovery,
		SpeculativeReceivers: r.SpeculativeReceivers,
		FS:                   r.FS, WorkspaceFolders: r.WorkspaceFolders, Mappings: r.Mappings,
		BeanPaths: r.BeanPaths, StartupFiles: r.StartupFiles,
		ExpressionMappings: r.ExpressionMappings, Index: r.Index, IndexTracksFiles: r.IndexTracksFiles,
		Resolvers: r.Resolvers, ImplicitExtends: r.ImplicitExtends,
		HelperScope: r.HelperScope, Stubs: r.Stubs, InferArgsFiles: r.InferArgsFiles, CallerInferenceLimits: r.CallerInferenceLimits, stubFS: r.fs(),
		callerApplication: root, callerMappings: r.effectiveMappings(baseDir), indexer: r,
	}
	// Source indexing is deliberately delegated to the original resolver. Shared
	// metadata must not acquire whichever caller happened to index a CFC first.
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.contextViews == nil {
		r.contextViews = make(map[string]*Resolver)
	}

	if existing := r.contextViews[root]; existing != nil {
		return existing
	}

	r.contextViews[root] = view

	return view
}

// executionApplication retains an explicit caller even when it has no
// Application.cfc; crossing a library must not manufacture a request context.
func (r *Resolver) executionApplication(baseDir string) string {
	if r.indexer != nil {
		return r.callerApplication
	}

	return r.FindApplicationRoot(baseDir)
}
