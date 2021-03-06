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
	"io/ioutil"
	"encoding/json"
)

var ddMap map[int64]int64
var distanceArray []int64
var minDelay int64 = 100
var maxDelay int64 = 1000
var distanceStep int64 = 0
var lastDistance int64 = 0
var mapFileName = "mapData"
var arrayFileName = "arrayData"

func handleJump(w http.ResponseWriter, r *http.Request){
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	q := r.URL.Query();
	x, e := strconv.ParseInt(q.Get("x"), 10, 64)
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	y, e := strconv.ParseInt(q.Get("y"), 10, 64)
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	d, e := strconv.ParseInt(q.Get("d"), 10, 64)
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 记录 用于adjust
	lastDistance = d
	
	//最小距离为屏幕的0.5%（x为屏幕的50%）
	if distanceStep == 0 {
		distanceStep = x/100;
	}
	
	// 初始化 step map array
	if ddMap == nil || len(ddMap) == 0 {
		ddMap = make(map[int64]int64)	
		ddMap[120] = minDelay
		ddMap[1200] = maxDelay
		distanceArray = append(distanceArray, 120)
		distanceArray = append(distanceArray, 1200)
		fmt.Println("distance:", 120, "delay:", minDelay, "length:", len(distanceArray))
		fmt.Println("distance:", 1200, "delay:", maxDelay, "length:", len(distanceArray))
	}
	
	// 取为step的整数倍
	d = d*int64(distanceStep)
	d = d/int64(distanceStep)
	
	delay := distance2Delay(lastDistance)
	click(x, y, delay)
	w.WriteHeader(http.StatusOK)

	fmt.Println("distance:", lastDistance, "delay:", delay, "length:", len(distanceArray))
}

func handleAdjust(w http.ResponseWriter, r *http.Request){
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	d, e := strconv.ParseInt(q.Get("d"), 10, 64)
	if e != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 取为step的整数倍
	d = d*int64(distanceStep)
	d = d/int64(distanceStep)

	ddMap[lastDistance] = ddMap[lastDistance]*(lastDistance+d)/lastDistance

	w.WriteHeader(http.StatusOK)

	fmt.Println("distance:", lastDistance, "delay:", ddMap[lastDistance], "length:", len(distanceArray))
}

func click(x, y, delay int64){
	cmd := exec.Command("adb", "shell", "input", "touchscreen", "swipe",
		strconv.FormatInt(int64(x), 10),
		strconv.FormatInt(int64(y), 10),
		strconv.FormatInt(int64(x+5), 10),
		strconv.FormatInt(int64(y+5), 10),
		strconv.FormatInt(int64(delay), 10))
	var out bytes.Buffer
	cmd.Stdout = &out
	e := cmd.Run()
	s := out.String()
	if e != nil {
		fmt.Println("cmd error:", e.Error())
		return
	} else if len(s) > 0 {
		fmt.Println("out:", s)
	}
}

func distance2Delay(d int64)int64{
	// 如果有则返回
	if delay := ddMap[d]; delay != 0 {
		return delay
	}

	// 如果是第一次点击则直接估算
	if len(ddMap) < 3 {
		delay := (ddMap[distanceArray[1]]-ddMap[distanceArray[0]])*(d-distanceArray[0])/(distanceArray[1]-distanceArray[0])+ddMap[distanceArray[0]]
		ddMap[d] = delay
		distanceArray = append(distanceArray,0)
		distanceArray[2] = distanceArray[1]
		distanceArray[1] = d
		return delay
	}

	// 从array取出与d最近的两个值，根据相似三角形计算delay
	small, big, pos := queryNearTwo(d)
	delay := (ddMap[big]-ddMap[small])*(d-small)/(big-small)+ddMap[small]

	// 保存d和delay
	ddMap[d] = delay
	insertArray(pos, d)
	return delay
}

// 在指定position插入value
func insertArray(p, v int64){
	distanceArray = append(distanceArray, 0)
	for i := int64(len(distanceArray))-1; i>p; i--{
		distanceArray[i] = distanceArray[i-1]
	}
	distanceArray[p] = v
}

// 二分查找最近的两个
func queryNearTwo(v int64) (int64, int64, int64){
	i := int64(0)
	j := int64(len(distanceArray)-1)
	
	for {
		if i == j-1 {
			return distanceArray[i], distanceArray[j], j
		}
		k := (i+j)/2
		if distanceArray[k] < v {
			i = k;
		}else{
			j = k;
		}
	}

	fmt.Println("query two error")
	return 0, 0, 0
}

func readMapFile(){
	f, e := os.OpenFile(mapFileName, os.O_RDONLY|os.O_CREATE, 0644)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer f.Close()

	b, e := ioutil.ReadAll(f)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	if len(b) == 0{
		return
	}

	if e := json.Unmarshal(b, &ddMap); e != nil {
		fmt.Println(e.Error())
		return
	}
	fmt.Println(ddMap)
}

func readArrayFile(){
	f, e := os.OpenFile(arrayFileName, os.O_RDONLY|os.O_CREATE, 0644)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer f.Close()

	b, e := ioutil.ReadAll(f)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	if len(b) == 0{
		return
	}

	if e := json.Unmarshal(b, &distanceArray); e!=nil{
		fmt.Println(e.Error())
		return
	}
	fmt.Println(distanceArray)
}

func writeMapFile(){
	if len(ddMap) == 0 {
		return
	}

	b, e := json.Marshal(ddMap)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	ioutil.WriteFile(mapFileName, b, 0644)
}

func writeArrayFile(){
	if len(distanceArray) == 0 {
		return
	}

	b, e := json.Marshal(distanceArray)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	ioutil.WriteFile(arrayFileName, b, 0644)
}

func main() {
	fmt.Println("hello")

	readMapFile();
	readArrayFile();

	port := "80"
	srv := &http.Server{ Addr: ":"+port }

	http.HandleFunc("/jump", handleJump)
	http.HandleFunc("/adjust", handleAdjust)

	go srv.ListenAndServe();

    // process kill and interrupt
    interrupt := make(chan os.Signal)
    signal.Notify(interrupt, os.Interrupt)
    signal.Notify(interrupt, os.Kill)
	<-interrupt
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    srv.Shutdown(ctx)
	cancel()

	writeMapFile();
	readArrayFile();

	fmt.Println("bye")
}