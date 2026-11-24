//go:build wasip1

package main

import "errors"

type mcpDaemonClient struct {
	sockPath string
}

func connectMCPDaemon(_, _ string) (*mcpDaemonClient, error) {
	return nil, errors.New("daemon mode is not supported in the wasip1 build")
}

func (*mcpDaemonClient) Close() error { return nil }

func (*mcpDaemonClient) ExplainCall(string, int, string) (string, error) {
	return "", errors.New("daemon mode is not supported in the wasip1 build")
}

func (*mcpDaemonClient) ResolveRoute(string) (any, error) {
	return nil, errors.New("daemon mode is not supported in the wasip1 build")
}

func (*mcpDaemonClient) LiveExplore(string, bool) (any, error) {
	return nil, errors.New("daemon mode is not supported in the wasip1 build")
}
