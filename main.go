package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/ruraomsk/camserver/mjpeg"
	"github.com/ruraomsk/camserver/rhttp"
	"io"
	"net/http"
	"runtime"
	"sync"
)

//const uri = "rtsp://10.8.0.4/video11"
var videos map[string]*rhttp.RStream
var streams map[string]*mjpeg.Stream
var counts map[string]int
var datas map[string][]byte
var mutex sync.Mutex

func readData(uri string) {
	for {
		mutex.Lock()
		v, is := videos[uri]
		if !is {
			delete(videos, uri)
			delete(counts, uri)
			delete(datas, uri)
			mutex.Unlock()
			fmt.Printf("Прибили читалку с %s\n", uri)
			return
		}
		pkt, err := v.ReadPacket()
		mutex.Unlock()
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Closed EOF %s\n", uri)
				mutex.Lock()
				delete(videos, uri)
				delete(counts, uri)
				delete(datas, uri)
				mutex.Unlock()
				return
			}
			fmt.Printf("error read packet %s %s\n", uri, err.Error())
			//time.Sleep(100*time.Millisecond)
			continue
		}
		if pkt.IsVideo() {
			mutex.Lock()
			datas[uri] = pkt.Data()
			mutex.Unlock()
		}
	}
}
func writeData(uri string, stream *mjpeg.Stream) {
	for {
		mutex.Lock()
		data, is := datas[uri]
		mutex.Unlock()
		if !is {
			fmt.Printf("Нет канала для %s\n", uri)
			break
		}
		err := stream.Update(data)
		if err != nil {
			fmt.Printf("Ошибка просмотра для %s %s\n", uri, err.Error())
			break
		}
	}
	mutex.Lock()
	if counts[uri] == 1 {
		fmt.Printf("Closed %s\n", uri)
		delete(videos, uri)
		delete(counts, uri)
		delete(datas, uri)
	} else {
		counts[uri] = counts[uri] - 1
		fmt.Printf("Осталось %d на камере %s", counts[uri], uri)
	}
	mutex.Unlock()

}
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	videos = make(map[string]*rhttp.RStream)
	streams = make(map[string]*mjpeg.Stream)
	counts = make(map[string]int)
	datas = make(map[string][]byte)
	streamHandler := func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.Query().Get("camera")
		stream := mjpeg.NewStream()
		mutex.Lock()
		var err error
		var video *rhttp.RStream
		video, is := videos[uri]
		if !is {
			video, err = rhttp.Open(uri)
			if err != nil {
				mutex.Unlock()
				fmt.Printf("нет камеры %s %s\n", uri, err.Error())
				return
			}
			videos[uri] = video
			counts[uri] = 1
			datas[uri] = make([]byte, 0)
			go readData(uri)
		} else {
			counts[uri] = counts[uri] + 1
		}
		mutex.Unlock()
		streams[uri] = stream
		go writeData(uri, stream)
		stream.ServeHTTP(w, r)
	}

	router := mux.NewRouter()
	router.HandleFunc("/stream", streamHandler)
	//router.HandleFunc("/stop", streamStop)
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
