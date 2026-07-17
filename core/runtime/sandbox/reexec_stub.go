//go:build !linux

package sandbox

// MaybeReexec is a no-op outside Linux (landlock child entrypoint).
func MaybeReexec() bool { return false }
