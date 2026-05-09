//go:build !linux

package shell

import "mvdan.cc/sh/v3/interp"

// sandboxHandler is a no-op on non-Linux platforms.
func (s *Shell) sandboxHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return next
	}
}
