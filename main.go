package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type counters struct {
	sync.Mutex
	view    int
	click   int
	content string
}

var (
	content     = []string{"sports", "entertainment", "business", "education"}
	allCounters = make([]counters, len(content))
	limit       = 10 // maximum 10 requests can be handled at a time
	sem         = make(chan int, limit)
)

func initialize() {
	for i := 0; i < len(content); i++ {
		allCounters[i] = counters{
			Mutex:   sync.Mutex{},
			view:    0,
			click:   0,
			content: content[i],
		}
	}
}

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	index := rand.Intn(len(content))

	allCounters[index].Lock()
	allCounters[index].view++
	allCounters[index].Unlock()

	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(index)
	}
}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(index int) error {
	allCounters[index].Lock()
	allCounters[index].click++
	allCounters[index].Unlock()

	return nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAllowed() {
		w.WriteHeader(429)
		return
	}
}

func isAllowed() bool {
	select {
	case sem <- 1:
		return true
	default:
		return false
	}
}

func uploadCounters() error {
	out := make(chan counters, len(content))

	for i := 0; i < len(content); i++ {
		go func(i int) {
			allCounters[i].Lock()
			out <- allCounters[i]
			allCounters[i].click = 0
			allCounters[i].view = 0
			allCounters[i].Unlock()
		}(i)
	}

	go func() {
		f, _ := os.OpenFile("store.txt", os.O_APPEND|os.O_WRONLY, 0644)
		if f == nil {
			f, _ = os.Create("store.txt")
		}
		defer f.Close()
		for c := range out {
			d := c.content + ":" + time.Now().Format("2006-01-02 15:04:05") + " {views: " + strconv.Itoa(c.view) + ", clicks: " + strconv.Itoa(c.click) + "}"
			fmt.Fprintln(f, d)
		}
	}()

	return nil
}

func main() {
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/stats/", statsHandler)
	initialize()

	go func() {
		startUpload := time.Tick(5 * time.Second)
		for {
			select {
			case <-startUpload:
				err := uploadCounters()
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	go func() {
		release := time.Tick(1 * time.Second)
		for {
			select {
			case <-release:
				<-sem
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
