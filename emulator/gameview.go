// credit to https://github.com/fogleman/nes
package emulator

import (
	"image"
	"log"

	"github.com/giongto35/cloud-game/emulator/nes"
)

// List key pressed
const (
	a1 = iota
	b1
	select1
	start1
	up1
	down1
	left1
	right1

	a2
	b2
	select2
	start2
	up2
	down2
	left2
	right2
)
const NumKeys = 8

// Audio consts
const (
	SampleRate = 16000
	Channels   = 1
	TimeFrame  = 60
	AppAudio   = 1
)

type GameView struct {
	console *nes.Console
	title   string
	hash    string

	// equivalent to the list key pressed const above
	keyPressed [NumKeys * 2]bool

	savingJob  *job
	loadingJob *job

	imageChannel chan *image.RGBA
	audioChannel chan float32
	inputChannel chan int
}

type job struct {
	path      string
	extraFunc func() error
}

func NewGameView(console *nes.Console, title, hash string, imageChannel chan *image.RGBA, audioChannel chan float32, inputChannel chan int) *GameView {
	gameview := &GameView{
		console:      console,
		title:        title,
		hash:         hash,
		keyPressed:   [NumKeys * 2]bool{false},
		imageChannel: imageChannel,
		audioChannel: audioChannel,
		inputChannel: inputChannel,
	}

	go gameview.ListenToInputChannel()
	return gameview
}

// ListenToInputChannel listen from input channel streamm, which is exposed to WebRTC session
//func (view *GameView) UpdateInput(keysInBinary int) {
//for i := 0; i < NumKeys*2; i++ {
//b := ((keysInBinary & 1) == 1)
//view.keyPressed[i] = (view.keyPressed[i] && b) || b
//keysInBinary = keysInBinary >> 1
//}
//}

// ListenToInputChannel listen from input channel streamm, which is exposed to WebRTC session
func (view *GameView) ListenToInputChannel() {
	for {
		keysInBinary := <-view.inputChannel
		for i := 0; i < NumKeys*2; i++ {
			b := ((keysInBinary & 1) == 1)
			view.keyPressed[i] = (view.keyPressed[i] && b) || b
			keysInBinary = keysInBinary >> 1
		}
	}
}

// Enter enter the game view.
func (view *GameView) Enter() {
	view.console.SetAudioSampleRate(SampleRate)
	view.console.SetAudioChannel(view.audioChannel)

	// load state if the hash file existed (Join the old room)
	if err := view.console.LoadState(savePath(view.hash)); err == nil {
		return
	} else {
		view.console.Reset()
	}
	//view.console.Reset()

	// load sram
	cartridge := view.console.Cartridge
	if cartridge.Battery != 0 {
		if sram, err := readSRAM(sramPath(view.hash)); err == nil {
			cartridge.SRAM = sram
		}
	}
}

// Exit ...
func (view *GameView) Exit() {
	view.console.SetAudioChannel(nil)
	view.console.SetAudioSampleRate(0)
	// save sram
	cartridge := view.console.Cartridge
	if cartridge.Battery != 0 {
		writeSRAM(sramPath(view.hash), cartridge.SRAM)
	}
}

// Update is called for every period of time, dt is the elapsed time from the last frame
func (view *GameView) Update(t, dt float64) {
	if dt > 1 {
		dt = 0
	}
	console := view.console
	view.updateControllers()
	view.UpdateEvents()
	console.StepSeconds(dt)

	// fps to set frame
	view.imageChannel <- console.Buffer()
}

func (view *GameView) Save(hash string, extraSaveFunc func() error) {
	// put saving event to queue, process in updateEvent
	view.savingJob = &job{
		path:      savePath(view.hash),
		extraFunc: extraSaveFunc,
	}
}

func (view *GameView) Load(path string, extraLoadFunc func() error) {
	// put saving event to queue, process in updateEvent
	view.loadingJob = &job{
		path:      savePath(view.hash),
		extraFunc: extraLoadFunc,
	}
}

func (view *GameView) UpdateEvents() {
	// If there is saving event, save and discard the save event
	log.Println(view.savingJob)
	if view.savingJob != nil {
		view.console.SaveState(view.savingJob.path)
		// Run extra function (online saving for example)
		go view.savingJob.extraFunc()
		view.savingJob = nil
	}
	// If there is loading event, save and discard the load event
	if view.loadingJob != nil {
		view.console.LoadState(view.loadingJob.path)
		// Run extra function (online saving for example)
		go view.loadingJob.extraFunc()
		view.loadingJob = nil
	}
}

func (view *GameView) updateControllers() {
	// Divide keyPressed to player 1 and player 2
	// First 8 keys are player 1
	var player1Keys [8]bool
	copy(player1Keys[:], view.keyPressed[:8])

	var player2Keys [8]bool
	copy(player2Keys[:], view.keyPressed[8:])

	view.console.Controller1.SetButtons(player1Keys)
	view.console.Controller2.SetButtons(player2Keys)
}
