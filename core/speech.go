package core

// SpeechProvider is the stable boundary for future article narration.
// The reader owns text extraction and presentation; speech providers only
// receive plain text, so a future cloud/local TTS engine can be swapped in
// without changing the reader UI.
type SpeechProvider interface {
	Available() bool
	Speak(text string) error
	Stop() error
}
