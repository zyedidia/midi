package main

import (
	"fmt"
	"os"

	"github.com/zyedidia/midi"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Please provide a soundfont and a midi file: ./simple soundfont.sf2 ../midis/take5.mid")
		return
	}

	p, err := midi.NewPlayer(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	p.Start()

	fmt.Println("TRACKS:", len(p.Tracks))

	for i, t := range p.Tracks {
		go func(i int, t *midi.Track) {
			for {
				note := <-t.Notes
				p.PlayNote(note)
				fmt.Println("Track", i, "-", note.Channel.ID, note.Channel.GetInstrument(), "- Pitch", note.Pitch, "-", note.On)
			}
		}(i, t)
	}

	finish := make(chan bool)
	var doneTracks int
	for _, t := range p.Tracks {
		go func(c chan bool) {
			if <-t.Done {
				doneTracks++
				if doneTracks == len(p.Tracks) {
					finish <- true
				}
			}
		}(t.Done)
	}

	<-finish
}
