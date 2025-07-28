package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
)

var stdoutMutex sync.Mutex
var messages = [][]string{
	{"こんにちは！", "元気ですか？", "これは日本語です。", "処理中...", "完了！"},
	{"Hello!", "How are you?", "This is English.", "Processing...", "Done!"},
	{"Hola!", "¿Cómo estás?", "Esto es español.", "Procesando...", "Hecho!"},          // spanish
	{"Bonjour!", "Comment ça va?", "Ceci est français.", "Traitement...", "Terminé!"}, // french
	{"Hallo!", "Wie geht's?", "Das ist Deutsch.", "Verarbeitung...", "Fertig!"},       // Germany
}
var numGoroutines = 5

func hideCursor() {
	fmt.Print("\033[?25l")
}

func showCursor() {
	fmt.Print("\033[0m\033[?25h")
}

func normal() {
	rand.Seed(time.Now().UnixNano())

	hideCursor()
	defer showCursor()

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int, msgs []string) {
			defer wg.Done()

			for j := 0; j < 1000; j++ {
				randomIndex := rand.Intn(len(msgs))
				selectedMsg := msgs[randomIndex]

				stdoutMutex.Lock()

				// move cursor to top
				fmt.Printf("\r\033[K")
				fmt.Printf("Goroutine %d (%s)", id, selectedMsg)

				os.Stdout.Sync()
				stdoutMutex.Unlock()

				// delay
				time.Sleep(time.Duration(10+id*5) * time.Millisecond)
			}
		}(i+1, messages[i]) // Goroutine ID and Message
	}

	wg.Wait()

	stdoutMutex.Lock()
	fmt.Printf("\r\033[KAll goroutines finished multicultural updates.\n")
	os.Stdout.Sync()
	stdoutMutex.Unlock()
}

func spin() {
	//s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	//s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	defer s.Stop()

	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int, msgs []string) {
			defer wg.Done()

			for j := 0; j < 1000; j++ {
				randomIndex := rand.Intn(len(msgs))
				selectedMsg := msgs[randomIndex]

				stdoutMutex.Lock()

				s.Suffix = fmt.Sprintf(" Goroutine %d (%s)", id, selectedMsg)

				os.Stdout.Sync()
				stdoutMutex.Unlock()

				// delay
				time.Sleep(time.Duration(10+id*5) * time.Millisecond)
			}
		}(i+1, messages[i]) // Goroutine ID and Message
	}

	wg.Wait()

	stdoutMutex.Lock()
	fmt.Printf("\r\033[KAll goroutines finished multicultural updates.\n")
	os.Stdout.Sync()
	stdoutMutex.Unlock()
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTSTP)

	go func() {
		_ = <-sigs
		showCursor()
		os.Exit(0)
	}()

	//normal()
	spin()
}
