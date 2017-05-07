package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/zyedidia/midi"
	sf "github.com/zyedidia/sfml/v2.3/sfml"
)

const (
	screenWidth  = 800
	screenHeight = 1000
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
	whiteSquare := sf.NewEmptyImageFromColor(40, 40, sf.ColorWhite)
	texture := sf.NewTextureFromImage(whiteSquare)

	lock := sync.RWMutex{}

	for i, t := range p.Tracks {
		go func(i int, t *midi.Track) {
			for {
				note := <-t.Notes
				p.PlayNote(note)
				channelID := note.Channel.ID

				if _, ok := rects[channelID]; !ok {
					rect := sf.NewRectangleShape(sf.Vector2f{40, 40})
					rect.SetOrigin(sf.Vector2f{20, 20})
					// rect.SetOutlineThickness(5)
					// rect.SetOutlineColor(colors[int(channelID)%len(colors)])
					rect.SetFillColor(colors[int(channelID)%len(colors)])
					rect.SetTexture(texture, true)

					text := sf.NewText(p.GetInstrument(note.Channel), arial, 15)
					text.SetColor(colors[int(channelID)%len(colors)])
					text.SetPosition(sf.Vector2f{10, float32(100 + int(channelID)*50)})

					lock.Lock()
					rects[channelID] = rect
					texts[channelID] = text
					lock.Unlock()
				}

				rect := rects[channelID]
				color := rect.GetFillColor()
				if note.On {
					color.A = 255
					rect.SetFillColor(color)
				} else {
					color.A = 100
					rect.SetFillColor(color)
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

		lock.RLock()
		for _, r := range rects {
			window.Draw(r)
		}
		for _, t := range texts {
			window.Draw(t)
		}
		lock.RUnlock()

		window.Display()
	}
}
