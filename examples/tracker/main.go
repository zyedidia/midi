package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
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
	if len(os.Args) < 4 {
		fmt.Println("Please provide a soundfont and a midi file and a channel number: ./graphics soundfont.sf2 ../midis/take5.mid 0")
		return
	}

	channelNum, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println(err)
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
	window.SetFramerateLimit(60)
	var rects []*sf.RectangleShape

	colors := []sf.Color{sf.ColorBlue, sf.ColorRed, sf.ColorGreen, sf.ColorCyan, sf.ColorMagenta, sf.ColorYellow, sf.ColorWhite}

	time.Sleep(500 * time.Millisecond)

	p.Start()

	fmt.Println("TRACKS:", len(p.Tracks))

	lock := sync.RWMutex{}

	for i, t := range p.Tracks {
		go func(i int, t *midi.Track) {
			for {
				note := <-t.Notes
				go func() {
					// time.Sleep(2000 * time.Millisecond)
					channelID := note.Channel.ID
					if channelNum == -1 || channelID == byte(channelNum) {
						rect := sf.NewRectangleShape(sf.Vector2f{6, 1000})
						rect.SetOrigin(sf.Vector2f{3, 1000})
						// rect.SetOutlineThickness(5)
						// rect.SetOutlineColor(colors[int(channelID)%len(colors)])
						if note.On {
							rect.SetFillColor(colors[int(channelID)%len(colors)])
						} else {
							rect.SetFillColor(sf.ColorBlack)
						}
						rect.SetPosition(sf.Vector2f{float32(note.Pitch) / float32(127) * screenWidth, 0})

						lock.Lock()
						rects = append(rects, rect)
						lock.Unlock()

						go func() {
							start := time.Now()
							played := false
							for {
								pos := rect.GetPosition()
								since := time.Since(start)
								pos.Y = float32(since) / float32(5*time.Second) * screenHeight
								rect.SetPosition(pos)
								// rect.Move(sf.Vector2f{0, 0.2})

								if time.Since(start) >= 5*time.Second && !played {
									if note.On {
										rect.SetFillColor(sf.Color{255, 140, 0, 255})
									}
									// note.Velocity = 127
									p.PlayNote(note)
									played = true
									// break
								}
								time.Sleep(time.Millisecond)
							}
						}()
					} else {
						time.Sleep(time.Second * 5)
						p.PlayNote(note)
					}
				}()
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
		var tempRects []*sf.RectangleShape
		for _, r := range rects {
			window.Draw(r)

			pos := r.GetPosition()
			size := r.GetGlobalBounds()
			if !(pos.Y-size.Height > screenHeight) {
				tempRects = append(tempRects, r)
			}
		}
		rects = tempRects
		lock.RUnlock()

		window.Display()
	}
}
