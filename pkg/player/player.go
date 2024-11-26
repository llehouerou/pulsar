package player

import (
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

type Player struct {
	streamer beep.StreamSeekCloser
	ctrl     *beep.Ctrl
}

func New() *Player {
	return &Player{}
}

func (p *Player) Play(filepath string) error {

	f, err := os.Open(filepath)
	if err != nil {
		return err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return err
	}

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	p.streamer = streamer
	p.ctrl = &beep.Ctrl{Streamer: streamer}

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
