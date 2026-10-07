//go:build !windows

package core

// noopSpeechProvider keeps the cross-platform core compilable.
type noopSpeechProvider struct{}

func (noopSpeechProvider) Available() bool    { return false }
func (noopSpeechProvider) Speak(string) error { return nil }
func (noopSpeechProvider) Stop() error        { return nil }
func newSpeechProvider() SpeechProvider       { return noopSpeechProvider{} }
