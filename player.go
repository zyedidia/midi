package midi

/*
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lfluidsynth
#include <fluidsynth.h>
char* go_sfont_get_preset_name(fluid_preset_t* preset) {
	return preset->get_name(preset);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jstesta/gomidi"
	"github.com/jstesta/gomidi/cfg"
	"github.com/jstesta/gomidi/midi"
)

const oneMinuteInMicroseconds = 60000000

// A Player plays midi files
type Player struct {
	Name string
	Done chan bool

	Tracks []*Track

	tempo         int32
	midi          *midi.Midi
	fluidsynth    *C.struct__fluid_synth_t
	fluidsettings *C.struct__fluid_hashtable_t
}

// NewPlayer returns a new audio player using a soundfont and a midi file
func NewPlayer(soundfont, midipath string) (*Player, error) {
	p := new(Player)

	// Set up fluid synth with some calls to C functions
	p.fluidsettings = C.new_fluid_settings()
	p.fluidsynth = C.new_fluid_synth(p.fluidsettings)
	C.new_fluid_audio_driver(p.fluidsettings, p.fluidsynth)
	C.fluid_synth_sfload(p.fluidsynth, C.CString(soundfont), 1)

	// Read the midi file
	var err error
	p.midi, err = gomidi.ReadMidiFromFile(midipath, cfg.GomidiConfig{nil, nil})
	if err != nil {
		return nil, err
	}

	format := p.midi.Header().Format()
	if format != 0 {
		return nil, errors.New("Midi format not supported: " + strconv.Itoa(format))
	}

	midiTracks := p.midi.Tracks()
	p.Tracks = make([]*Track, len(midiTracks))

	for i, t := range midiTracks {
		track := &Track{make(chan Note), make([]*Channel, 0), make(chan bool)}
		p.Tracks[i] = track
		events := t.Events()

		for _, e := range events {
			switch ev := e.(type) {
			case *midi.SysexEvent:
			case *midi.MetaEvent:
				data := ev.Data()

				metatype := fmt.Sprintf("%x", ev.MetaType())

				if metatype == "3" {
					p.Name = string(data)
				}
			case *midi.MidiEvent:
				str := fmt.Sprintf("%x", ev.Status())
				channelID, _ := strconv.ParseInt(string(str[1]), 16, 32)

				var channel *Channel
				for _, c := range track.Channels {
					if c.ID == byte(channelID) {
						channel = c
						break
					}
				}

				if channel == nil {
					channel = &Channel{byte(channelID), 0, 0}
					track.Channels = append(track.Channels, channel)
				}
			}
		}
	}

	return p, nil
}

// Start begins playing the midi file.
// This function spawns a goroutine and returns immediately.
func (p *Player) Start() {
	midiTracks := p.midi.Tracks()

	// Default 120 bpm tempo
	// This can be overridden by a meta event
	tempo := int32(oneMinuteInMicroseconds / 120)
	ppqn := p.midi.Header().Division().(*midi.MetricalDivision).Resolution()

	for i, t := range midiTracks {
		events := t.Events()
		track := p.Tracks[i]

		go func() {
			for _, e := range events {
				switch ev := e.(type) {
				case *midi.SysexEvent:
				case *midi.MetaEvent:
					data := ev.Data()

					metatype := fmt.Sprintf("%x", ev.MetaType())

					if metatype == "51" {
						tempo = convertByteToInt([4]byte{0, data[0], data[1], data[2]})
					}

					if metatype == "2f" {
						track.Done <- true
					}
				case *midi.MidiEvent:
					data := ev.Data()
					dt := ev.DeltaTime()
					time.Sleep(time.Duration(float64(dt)*float64(tempo)/float64(ppqn)*1000) * time.Nanosecond)

					str := fmt.Sprintf("%x", ev.Status())
					channelID, _ := strconv.ParseInt(string(str[1]), 16, 32)

					var channel *Channel
					for _, c := range track.Channels {
						if c.ID == byte(channelID) {
							channel = c
							break
						}
					}
					if channel == nil {
						continue
					}

					if str[0] == '8' {
						track.Notes <- Note{channel, data[0], data[1], false}
					} else if str[0] == '9' {
						noteon := true
						if data[1] == 0 {
							// velocity = 0 also means note off even if sent in a note on event
							noteon = false
						}

						track.Notes <- Note{channel, data[0], data[1], noteon}
					} else if str[0] == 'a' {
					} else if str[0] == 'b' {
						// Controller change
						channel.controller = data[0]
						C.fluid_synth_cc(p.fluidsynth, C.int(channelID), C.int(data[0]), C.int(data[1]))
					} else if str[0] == 'c' {
						// Program change
						channel.program = data[0]
						C.fluid_synth_program_change(p.fluidsynth, C.int(channelID), C.int(data[0]))
					} else if str[0] == 'd' {
						C.fluid_synth_channel_pressure(p.fluidsynth, C.int(channelID), C.int(data[0]))
					} else if str[0] == 'e' {
						pitchBend, _ := strconv.ParseInt(string(data[0])+string(data[1]), 16, 32)
						C.fluid_synth_pitch_bend(p.fluidsynth, C.int(channelID), C.int(pitchBend))
					}
				}
			}
		}()
	}
}

// PlayNote plays the specificed Note.
// If note.On is true it will play the note
// If note.On is false it will release the note
func (p *Player) PlayNote(n Note) {
	if n.On {
		C.fluid_synth_noteon(p.fluidsynth, C.int(n.Channel.ID), C.int(n.Pitch), C.int(n.Velocity))
	} else {
		C.fluid_synth_noteoff(p.fluidsynth, C.int(n.Channel.ID), C.int(n.Pitch))
	}
}

// A Track is a piece of music. It contains the notes to be played
// As well as all of the channels
type Track struct {
	// Notes will be placed in this channel at the right time in the song.
	// Make sure to read from the channel immediately after the note is placed
	// so that there is no delay
	Notes    chan Note
	Channels []*Channel
	// The value 'true' will be placed in this channel when the track is done playing
	Done chan bool
}

// A Channel corresponds to a certain instrument
type Channel struct {
	// This Channel's ID number
	ID         byte
	program    byte
	controller byte
}

// GetInstrument returns the instrument that this channel is using to play notes
func (p *Player) GetInstrument(channel *Channel) string {
	preset := C.fluid_synth_get_channel_preset(p.fluidsynth, C.int(channel.ID))
	name := C.go_sfont_get_preset_name(preset)
	return C.GoString(name)
}

// A Note holds the information to play a pitch in the song
type Note struct {
	// Channel: The channel that this note is being played on
	Channel *Channel

	// Pitch: what pitch to play
	Pitch byte

	// Velocity: how 'hard' to play the note
	Velocity byte

	// On: If the note is being played or released
	On bool
}
