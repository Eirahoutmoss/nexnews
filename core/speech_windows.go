//go:build windows

package core

import (
	"encoding/base64"
	"os/exec"
	"strings"
	"sync"
)

// windowsSpeechProvider uses the Windows PowerShell/System.Speech stack as a
// zero-dependency bridge. It is intentionally behind SpeechProvider so Phase 6
// can replace it with a native/local TTS engine without changing the reader.
type windowsSpeechProvider struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func (p *windowsSpeechProvider) Available() bool {
	_, err := exec.LookPath("powershell.exe")
	return err == nil
}

func (p *windowsSpeechProvider) Speak(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	_ = p.Stop()
	// Base64 avoids quoting problems with Turkish punctuation and article text.
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	script := "$b=[Convert]::FromBase64String('" + b64 + "');$t=[Text.Encoding]::UTF8.GetString($b);Add-Type -AssemblyName System.Speech;$s=New-Object System.Speech.Synthesis.SpeechSynthesizer;$s.Rate=0;$s.Speak($t);$s.Dispose()"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()
	err := cmd.Start()
	if err != nil {
		p.mu.Lock()
		p.cmd = nil
		p.mu.Unlock()
		return err
	}
	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		if p.cmd == cmd {
			p.cmd = nil
		}
		p.mu.Unlock()
	}()
	return nil
}

func (p *windowsSpeechProvider) Stop() error {
	p.mu.Lock()
	cmd := p.cmd
	p.cmd = nil
	p.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

func newSpeechProvider() SpeechProvider { return &windowsSpeechProvider{} }
