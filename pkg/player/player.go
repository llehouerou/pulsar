package player

import (
	"os"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

type Metadata struct {
	Artist string
	Title  string
}

type Player struct {
	streamer   beep.StreamSeekCloser
	ctrl       *beep.Ctrl
	length     int
	sampleRate beep.SampleRate
	metadata   Metadata
}

func New() *Player {
	return &Player{}
}

func (p *Player) Play(filepath string) error {
	// Read metadata first
	tag, err := id3v2.Open(filepath, id3v2.Options{Parse: true})
	if err == nil {
		p.metadata = Metadata{
			Artist: tag.Artist(),
			Title:  tag.Title(),
		}
		tag.Close()
	}

	f, err := os.Open(filepath)
	if err != nil {
		return err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return err
	}

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	p.streamer = streamer
	p.ctrl = &beep.Ctrl{Streamer: streamer}
	p.length = streamer.Len()
	p.sampleRate = format.SampleRate

	speaker.Play(p.ctrl)
	return nil
}

func (p *Player) Toggle() {
	if p.ctrl != nil {
		p.ctrl.Paused = !p.ctrl.Paused
	}
}

func (p *Player) Stop() {
	if p.streamer != nil {
		p.streamer.Seek(0)
	}
}

func (p *Player) Close() {
	if p.streamer != nil {
		p.streamer.Close()
	}
	p.ctrl = nil
}

func (p *Player) Position() float64 {
	if p.streamer == nil || p.length == 0 {
		return 0
	}
	return float64(p.streamer.Position()) / float64(p.length)
}

func (p *Player) GetMetadata() Metadata {
	return p.metadata
}

func (p *Player) Duration() time.Duration {
	if p.streamer == nil || p.sampleRate == 0 {
		return 0
	}
	return p.sampleRate.D(p.length)
}

func (p *Player) CurrentPosition() time.Duration {
	if p.streamer == nil || p.sampleRate == 0 {
		return 0
	}
	return p.sampleRate.D(p.streamer.Position())
}
