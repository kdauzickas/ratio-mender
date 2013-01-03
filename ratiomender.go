package main

import (
	"io/ioutil"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"container/ring"
)

var Up = flag.Float64("u", 1, "By how much should the upload be multiplied")
var Down = flag.Float64("d", 1, "By how much should the download be multiplied")
var Switch = flag.Bool("s", false, "Switch download with upload. Multipliers are applied before the switch")
var Port = flag.Int("p", 57998, "Port to listen")
var LogToOutput = flag.Bool("l", false, "Print log entries to output")
var Help = flag.Bool("h", false, "Print this help and exit")
var Log = ring.New(100)
var Version = "0.1.1"
var StartTime string

func main() {
	flag.Parse()

	if *Help {
		fmt.Println("Ratio mender " + Version + " usage:\n  ratiomender [options]\n\nOptions:")
		flag.PrintDefaults()
		return
	}

	http.HandleFunc("/favicon.png", faviconGenerator)
	http.HandleFunc("/favicon.ico", faviconGenerator)
	http.HandleFunc("/log", showLog)
	http.HandleFunc("/", Tamper)

	StartTime = time.Now().Format(time.ANSIC)

	err := http.ListenAndServe("127.0.0.1:" + strconv.Itoa(*Port), nil)
	if err != nil {
		fmt.Println("[Error] " + err.Error())
	}
}

// Tamper checks the request sent and either passes it thru unchanged if it
// doesn't look like a request to a tracker or modifies upload/download report
// according to the parameters passed
func Tamper(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || r.Form.Get("downloaded") == "" || r.Form.Get("uploaded") == "" {
		addLog(fmt.Sprintf(
			"[%s] [Info] Not an upload/download update. Passing thru unchanged: %s",
			time.Now().Format(time.ANSIC), r.URL.String()))

		_, err := send(w, r)
		if err != nil {
			w.WriteHeader(500)
			addLog(fmt.Sprintf("[%s] [Error] Could not send update. %s",
				time.Now().Format(time.ANSIC), err.Error()))
		}
		return
	}

	down, err := strconv.ParseInt(r.Form.Get("downloaded"), 10, 64)
	if err != nil {
		w.WriteHeader(500)
		addLog(fmt.Sprintf("[%s] [Error] Could not parse download. %s",
			time.Now().Format(time.ANSIC), err.Error()))
		return
	}

	upload, err := strconv.ParseInt(r.Form.Get("uploaded"), 10, 64)
	if err != nil {
		w.WriteHeader(500)
		addLog(fmt.Sprintf("[%s] [Error] Could not parse upload. %s",
			time.Now().Format(time.ANSIC), err.Error()))
		return
	}

	urlA := strings.Split(r.URL.String(), "?")
	if len(urlA) < 1 {
		w.WriteHeader(500)
		addLog(fmt.Sprintf("[%s] [Error] Malformed GET string.", time.Now().Format(time.ANSIC)))
		return
	}

	downNew := strconv.FormatInt(int64(*Down*float64(down)), 10)
	uploadNew := strconv.FormatInt(int64(*Up*float64(upload)), 10)

	if *Switch {
		t := downNew
		downNew = uploadNew
		uploadNew = t
	}

	r.Form.Set("downloaded", downNew)
	r.Form.Set("uploaded", uploadNew)

	r.URL, err = url.Parse(urlA[0] + "?" + r.Form.Encode())
	if err != nil {
		w.WriteHeader(500)
		addLog(fmt.Sprintf("[%s] [Error] Could not build new request. %s",
			time.Now().Format(time.ANSIC), err.Error()))
		return
	}

	resp, err := send(w, r)
	if err != nil {
		w.WriteHeader(500)
		addLog(fmt.Sprintf("[%s] [Error] Could not send request. %s",
			time.Now().Format(time.ANSIC), err.Error()))
		return
	}

	addLog(fmt.Sprintf("[%s] [Info] Upload actual: %d, Upload reported: %s, Download actual: %d, Download reported: %s, Tracker said: %s",
		time.Now().Format(time.ANSIC), int64(upload), uploadNew, int64(down), downNew, resp.Status))
}

// Send http request
func send(w http.ResponseWriter, r *http.Request) (resp *http.Response, err error) {
	err = nil
	r.RequestURI = ""
	client := new (http.Client)
	resp, err = client.Do(r)
	if err != nil {
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	
	err = resp.Body.Close()
	if err != nil {
		return
	}
	
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	return
}

// showLog writes Log to the http answer
func showLog(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<html><head><title>Ratio mender [" + Version + "]</title>" +
		"<link rel=\"icon\" type=\"image/png\" href=\"/favicon.png\">" +
		"</head><body><pre>"))

	w.Write([]byte(fmt.Sprintf(
		"[%s] Ratio Mender %s Listening: 127.0.0.1:%d. Upload: x%f, Download: x%f, Switch: %v\n\n",
		StartTime, Version, *Port, *Up, *Down, *Switch)))

	t := Log.Prev()
	for i := Log.Len(); i > 0; i-- {
		if v, ok := t.Value.([]byte); ok {
			w.Write(v)
		}
		t = t.Prev()
	}
	w.Write([]byte("</pre></body></html>"))
}

func addLog(msg string) {
	Log.Value = []byte(msg + "\n")
	Log = Log.Next()
	if *LogToOutput {
		fmt.Println(msg)
	}
}

// Dumps the favicon
func faviconGenerator(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "image/png")
	w.Write(*Icon)
}
