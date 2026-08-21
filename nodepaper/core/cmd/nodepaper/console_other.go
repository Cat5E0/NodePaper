//go:build !windows

package main

// ownsConsoleWindow is Windows-specific: only there can a graphical shell open
// a console window that belongs to this process alone. Other platforms never
// hold the output open.
func ownsConsoleWindow() bool { return false }
