package main

import (
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/generators"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

var speakerInitialized = false
var speakerInitOnce sync.Once
var speakerSampleRate beep.SampleRate = 44100
var soundAvailable = true

func initSpeaker() {
	speakerInitOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				// Audio backend not available (e.g., missing ALSA); disable sound gracefully
				soundAvailable = false
				speakerInitialized = false
			}
		}()
		speaker.Init(speakerSampleRate, speakerSampleRate.N(time.Second/10))
		speakerInitialized = true
		soundAvailable = true
	})
}

func playMp3(name string) {
	f, err := os.Open(name)
	if err != nil {
		// No file; skip sound
		return
	}

	streamer, _, err := mp3.Decode(f)
	if err != nil {
		// Decode failed; skip sound
		return
	}
	defer streamer.Close()
	defer f.Close()

	// Initialize speaker if not already done
	initSpeaker()
	if !soundAvailable {
		return
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}

func playMid(name string) {
	// not using for now - soundfont is ridiculously big
	return
	/*
		var sampleRate beep.SampleRate = 44100

		err := speaker.Init(sampleRate, sampleRate.N(time.Second/30))
		if err != nil {
			log.Fatal(err)
		}

		// Load a soundfont.
		soundFontFile, err := os.Open("Florestan-Basic-GM-GS-by-Nando-Florestan(Public-Domain).sf2")
		if err != nil {
			log.Fatal(err)
		}
		soundFont, err := midi.NewSoundFont(soundFontFile)
		if err != nil {
			log.Fatal(err)
		}

		// Load a midi track
		midiFile, err := os.Open(name)
		if err != nil {
			log.Fatal(err)
		}
		s, format, err := midi.Decode(midiFile, soundFont, sampleRate)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Song duration: %v\n", format.SampleRate.D(s.Len()))
		speaker.PlayAndWait(s)
	*/
}

func playWav(name string) {
	f, err := os.Open(name)
	if err != nil {
		return
	}

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return
	}
	defer streamer.Close()

	// Initialize speaker and check availability
	speakerInitOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				soundAvailable = false
				speakerInitialized = false
			}
		}()
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		speakerInitialized = true
		soundAvailable = true
	})
	if !soundAvailable {
		return
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	<-done
}

func playBeep(style string) {
	// accept updown, up, down, ding
	sr := beep.SampleRate(48000)
	// Initialize speaker if needed
	speakerInitOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				soundAvailable = false
				speakerInitialized = false
			}
		}()
		speaker.Init(sr, 4800)
		speakerInitialized = true
		soundAvailable = true
	})
	if !soundAvailable {
		return
	}

	ch := make(chan struct{})
	buzzer1, _ := generators.SawtoothTone(sr, float64(750))
	buzzer2, _ := generators.SawtoothTone(sr, float64(850))
	buzzer3, _ := generators.SawtoothTone(sr, float64(950))
	buzzer4, _ := generators.SawtoothTone(sr, float64(1050))
	buzzer5, _ := generators.SawtoothTone(sr, float64(1150))
	// Play 1/n second of each tone
	t := sr.N(time.Second / 10)
	f := sr.N(time.Second / 5)
	switch style {
	case "updown":
		buzz := []beep.Streamer{
			beep.Take(t, buzzer1),
			beep.Take(t, buzzer2),
			beep.Take(t, buzzer3),
			beep.Take(t, buzzer4),
			beep.Take(t, buzzer5),
			beep.Take(t, buzzer4),
			beep.Take(t, buzzer3),
			beep.Take(t, buzzer2),
			beep.Take(f, buzzer1),
			beep.Callback(func() {
				ch <- struct{}{}
			}),
		}
		speaker.Play(beep.Seq(buzz...))
		<-ch
	case "up":
		buzz := []beep.Streamer{
			beep.Take(t, buzzer1),
			beep.Take(t, buzzer2),
			beep.Take(t, buzzer3),
			beep.Take(t, buzzer4),
			beep.Take(t, buzzer5),
			beep.Callback(func() {
				ch <- struct{}{}
			}),
		}
		speaker.Play(beep.Seq(buzz...))
		<-ch
	case "down":
		buzz := []beep.Streamer{
			beep.Take(t, buzzer5),
			beep.Take(t, buzzer4),
			beep.Take(t, buzzer3),
			beep.Take(t, buzzer2),
			beep.Take(f, buzzer1),
			beep.Callback(func() {
				ch <- struct{}{}
			}),
		}
		speaker.Play(beep.Seq(buzz...))
		<-ch
	case "ding":
		t = sr.N(time.Second / 4)
		buzzer1, _ := generators.SawtoothTone(sr, float64(350))
		buzz := []beep.Streamer{
			beep.Take(t, buzzer1),
			beep.Callback(func() {
				ch <- struct{}{}
			}),
		}
		speaker.Play(beep.Seq(buzz...))
		<-ch
	}
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
