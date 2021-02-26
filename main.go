package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

type counters struct {
	view    int64
	click   int64
	content string
}

var (
	content     = []string{"sports", "entertainment", "business", "education"}
	allCounters = make([]counters, len(content))

	limit = 10 // maximum 10 requests can be handled at a time
	sem   = make(chan int, limit)
)

func initialize() {
	for i := 0; i < len(content); i++ {
		allCounters[i] = counters{
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

	atomic.AddInt64(&allCounters[index].view, 1)

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
	atomic.AddInt64(&allCounters[index].click, 1)

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
	out := make(chan counters)

	for i := 0; i < len(content); i++ {
		go func(i int) {
			out <- allCounters[i]
			atomic.StoreInt64(&allCounters[i].view, 0)
			atomic.StoreInt64(&allCounters[i].click, 0)
		}(i)
	}

	go func() {
		f, _ := os.OpenFile("store.txt", os.O_APPEND|os.O_WRONLY, 0644)
		if f == nil {
			f, _ = os.Create("store.txt")
		}
		defer f.Close()
		// use fan-in channels to persist data and add timeout to avoid deadlock
		for {
			select {
			case o := <-out:
				d := o.content + ":" + time.Now().Format("2006-01-02 15:04:05") + " {views: " + strconv.FormatInt(o.view, 10) + ", clicks: " + strconv.FormatInt(o.click, 10) + "}"
				fmt.Fprintln(f, d)
			case <-time.After(2 * time.Second):
				return
			}
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
