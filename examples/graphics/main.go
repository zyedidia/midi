package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/zyedidia/midi"
	sf "github.com/zyedidia/sfml/v2.3/sfml"
)

const (
	screenWidth  = 800
	screenHeight = 800
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Please provide a soundfont and a midi file: ./graphics soundfont.sf2 ../midis/take5.mid")
		return
	}

	runtime.LockOSThread()
	p, err := midi.NewPlayer(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	window := sf.NewRenderWindow(sf.VideoMode{screenWidth, screenHeight, 32}, p.Name, sf.StyleDefault, nil)
	// window.SetVerticalSyncEnabled(true)
	// window.SetFramerateLimit(60)
	rects := make(map[byte]*sf.RectangleShape)
	texts := make(map[byte]*sf.Text)

	colors := []sf.Color{sf.ColorBlue, sf.ColorRed, sf.ColorGreen, sf.ColorCyan, sf.ColorMagenta, sf.ColorYellow, sf.ColorWhite}

	time.Sleep(500 * time.Millisecond)

	p.Start()

	fmt.Println("TRACKS:", len(p.Tracks))

	arial := sf.NewFont("Arial.ttf")

	for i, t := range p.Tracks {
		go func(i int, t *midi.Track) {
			for {
				note := <-t.Notes
				p.PlayNote(note)
				channelID := note.Channel.ID

				if _, ok := rects[channelID]; !ok {
					rect := sf.NewRectangleShape(sf.Vector2f{40, 40})
					rect.SetOrigin(sf.Vector2f{20, 20})
					rect.SetOutlineThickness(5)
					rect.SetOutlineColor(colors[int(channelID)%len(colors)])
					rects[channelID] = rect

					text := sf.NewText(p.GetInstrument(note.Channel), arial, 15)
					text.SetColor(colors[int(channelID)%len(colors)])
					text.SetPosition(sf.Vector2f{10, float32(100 + int(channelID)*50)})

					texts[channelID] = text

				}

				rect := rects[channelID]
				if note.On {
					rect.SetFillColor(colors[int(channelID)%len(colors)])
				} else {
					rect.SetFillColor(sf.ColorBlack)
				}
				rect.SetPosition(sf.Vector2f{float32(note.Pitch) / float32(127) * screenWidth, float32(100 + int(channelID)*50)})
			}
		}(i, t)
	}

	go func() {
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
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	for window.IsOpen() {
		if event := window.PollEvent(); event != nil {
			switch event.Type {
			case sf.EventClosed:
				window.Close()
				os.Exit(0)
			}
		}

		window.Clear(sf.ColorBlack)

		for _, r := range rects {
			window.Draw(r)
		}
		for _, t := range texts {
			window.Draw(t)
		}

		window.Display()
	}
}
