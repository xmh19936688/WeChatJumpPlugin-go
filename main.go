package main

import(
	"fmt"
	"os/exec"
	"bytes"
	"net/http"
	"os"
	"context"
	"time"
	"os/signal"
	"strconv"
)

func main() {
	fmt.Println("hello")

	port := "80"
	srv := &http.Server{ Addr: ":"+port }

	http.HandleFunc("/jump", handleJump)

	go srv.ListenAndServe();

    // process kill and interrupt
    interrupt := make(chan os.Signal)
    signal.Notify(interrupt, os.Interrupt)
    signal.Notify(interrupt, os.Kill)
	<-interrupt
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    srv.Shutdown(ctx)
	cancel()
}

func handleJump(w http.ResponseWriter,r *http.Request){
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	q := r.URL.Query();
	x, e := strconv.ParseUint(q.Get("x"), 10, 64)
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	y, e := strconv.ParseUint(q.Get("y"), 10, 64)
	if e!=nil{
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	d, e := strconv.ParseUint(q.Get("d"), 10, 64)
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	click(x, y, d)
	w.WriteHeader(http.StatusOK)
}

func click(x, y, delay uint64){
	cmd := exec.Command("adb", "shell", "input", "touchscreen", "swipe",
		strconv.FormatUint(x, 10),
		strconv.FormatUint(y, 10),
		strconv.FormatUint(x+5, 10),
		strconv.FormatUint(y+5, 10),
		strconv.FormatUint(delay, 10))
	var out bytes.Buffer
	cmd.Stdout = &out
	e := cmd.Run()
	s := out.String()
	if e != nil {
		fmt.Println(e.Error())
		return
	} else if len(s) > 0 {
		fmt.Println(s)
	}
	fmt.Println(delay)
}