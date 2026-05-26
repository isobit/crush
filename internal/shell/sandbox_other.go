//go:build !linux

package shell

import "mvdan.cc/sh/v3/interp"

// sandboxHandler is a no-op on non-Linux platforms.
func sandboxHandler(_ string, _ *SandboxConfig) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return next
	}
}
