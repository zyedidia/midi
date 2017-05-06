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
	colors := []sf.Color{sf.ColorBlue, sf.ColorRed, sf.ColorGreen, sf.ColorCyan, sf.ColorMagenta, sf.ColorYellow, sf.ColorWhite}

	time.Sleep(500 * time.Millisecond)

	p.Start()

	fmt.Println("TRACKS:", len(p.Tracks))

	// lock := sync.RWMutex{}

	for i, t := range p.Tracks {
		go func(i int, t *midi.Track) {
			for {
				note := <-t.Notes
				p.PlayNote(note)
				fmt.Println("Track", i, "-", note.Channel.GetInstrument(), "- Pitch", note.Pitch, "-", note.On)
				channelID := note.Channel.ID

				if _, ok := rects[channelID]; !ok {
					rect := sf.NewRectangleShape(sf.Vector2f{40, 40})
					rect.SetOrigin(sf.Vector2f{20, 20})
					rect.SetOutlineThickness(5)
					rect.SetOutlineColor(colors[int(channelID)%len(colors)])
					// lock.Lock()
					rects[channelID] = rect
					// lock.Unlock()
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

		// lock.RLock()
		for _, r := range rects {
			window.Draw(r)
		}
		// lock.RUnlock()

		window.Display()
	}
}
