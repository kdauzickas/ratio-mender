package main

import (
	"net/http"
	"net/url"
	"fmt"
	"strconv"
	"strings"
	"flag"
	"bytes"
	"time"
)

var xUP = flag.Float64("u", 1, "By how much should the upload be multiplied")
var xDOWN = flag.Float64("d", 1, "By how much should the download be multiplied")
var PORT = flag.Int("p", 57998, "Port to listen to")
var HELP = flag.Bool("h", false, "Print this help and exit") 

func main() {
	flag.Parse()
	if *HELP == true {
		fmt.Println(`
  Ratio Mender is a program that allows you to tamper your torrent upload and
  download data.
		
  Usage:
    ratiomender [arguments]` + "\n")
		flag.PrintDefaults()
		return
	}

	http.HandleFunc("/", proxy)
	fmt.Printf("Ratio Mender Listening: 127.0.0.1:%d. Upload x%f, Download x%f\n", *PORT, *xUP, *xDOWN)

	err := http.ListenAndServe("127.0.0.1:" + strconv.Itoa(*PORT), nil)
	if err != nil {
		fmt.Println("[Error] " + err.Error())
	}	
}

func proxy(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Format(time.ANSIC)

	err := r.ParseForm()
	if err != nil {
		fmt.Println("[" + now + "] [Error] Could not parse request. " + err.Error())
		return
	}
	
	down, err := strconv.ParseFloat(r.Form.Get("downloaded"), 64)
	if err != nil {
		fmt.Println("[" + now + "] [Error] Could not get download. " + err.Error())
		return
	}
	
	upload, err := strconv.ParseFloat(r.Form.Get("uploaded"), 64)
	if err != nil {
		fmt.Println("[" + now + "] [Error] Could not get upload. " + err.Error())
		return
	}

	urlA := strings.Split(r.URL.String(), "?")
	if len(urlA) < 1 {
		fmt.Println("[" + now + "] [Error] Malformed GET string.")
		return
	}
	
	downNew   := strconv.Itoa(int(*xDOWN * down))
	uploadNew := strconv.Itoa(int(*xUP * upload))

	r.Form.Set("downloaded", downNew)
	r.Form.Set("uploaded", uploadNew)
	
	r.URL, err = url.Parse(urlA[0] + "?" + r.Form.Encode())
	if err != nil {
		fmt.Println("[" + now + "] [Error] Could not build new request. " + err.Error())
		return
	}
	
	r.RequestURI = ""
	
	client  := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		fmt.Println("[" + now + "] [Error] Could not send update. " + err.Error())
		return
	}
	
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	
	w.WriteHeader(resp.StatusCode)
	w.Write(buf.Bytes())
	
	fmt.Printf("[%s] Upload actual: %d, Upload reported: %s, Download actual: %d, Download reported: %s, Tracker said: %s\n",
				now, int(upload), uploadNew, int(down), downNew, resp.Status)
}


