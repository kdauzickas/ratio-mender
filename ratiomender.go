package main

import (
	"container/ring"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var Up = flag.Float64("u", 1, "By how much should the upload be multiplied")
var Down = flag.Float64("d", 1, "By how much should the download be multiplied")
var Switch = flag.Bool("s", false, "Switch download with upload. Multipliers are applied before the switch")
var Port = flag.Int("p", 57998, "Port for proxy to listen")
var LogToOutput = flag.Bool("l", false, "Print log entries to output")
var Help = flag.Bool("h", false, "Print this help and exit")
var Public = flag.Bool("i", false, "Should RM accept connections originating outside this machine")

var Log = ring.New(100)
var Version = "0.2.0"

var StartTime string
var ListeningOn string
var logger chan string

var lockLog = false

const timeFormat = "2006-01-02 15:03"

func main() {
	StartTime = time.Now().Format(timeFormat)

	flag.Parse()

	if *Help {
		fmt.Println("Ratio mender " + Version + " usage:\n  ratiomender [options]\n\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nAny bugs or feature requests can be filled at https://github.com/kdauzickas/ratio-mender/issues\n\n" +
					"This software is provided \"as is\" and \"with all faults.\" Creator of this software makes no representations " + 
					"or warranties of any kind concerning the safety, suitability, lack of viruses, inaccuracies, typographical errors, " +
					"or other harmful components of this software. There are inherent dangers in the use of any software, and you are " +
					"solely responsible for determining whether this software is compatible with your equipment and other software " +
					"installed on your equipment. You are also solely responsible for the protection of your equipment and backup of your " +
					"data, and creator will not be liable for any damages you may suffer in connection with using, modifying, or distributing " +
					"this software.")
		return
	}

	logger = make(chan string)
	go rotateLog()

	http.HandleFunc("/favicon.png", faviconGenerator)
	http.HandleFunc("/favicon.ico", faviconGenerator)
	http.HandleFunc("/log", showLog)
	http.HandleFunc("/", Tamper)

	var domain string
	if *Public {
		domain = ":"
	} else {
		domain = "127.0.0.1:"
	}
	ListeningOn = domain + strconv.Itoa(*Port)

	err := http.ListenAndServe(ListeningOn, nil)
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
		logger <- fmt.Sprintf("[%s] [Info] Not an upload/download update. Passing thru unchanged: %s",
			time.Now().Format(timeFormat), r.URL.String())

		_, err := send(w, r)
		if err != nil {
			w.WriteHeader(500)
			logger <- fmt.Sprintf("[%s] <span title=\"%s\">[Error]</span> Could not send update.",
				time.Now().Format(timeFormat), err.Error())
		}
		return
	}

	encLength := base64.StdEncoding.EncodedLen(len(r.Form.Get("info_hash")))
	hash := make([]byte, encLength)
	base64.StdEncoding.Encode(hash, []byte(r.Form.Get("info_hash")))
	formatedHash := fmt.Sprintf("<span title=\"%s\">[%s]</span>", hash[:27], hash[:8])

	down, err := strconv.ParseInt(r.Form.Get("downloaded"), 10, 64)
	if err != nil {
		w.WriteHeader(500)
		logger <- fmt.Sprintf("[%s] %s <span title=\"%s\">[Error]</span> Could not parse download.",
			time.Now().Format(timeFormat), formatedHash, err.Error())
		return
	}

	upload, err := strconv.ParseInt(r.Form.Get("uploaded"), 10, 64)
	if err != nil {
		w.WriteHeader(500)
		logger <- fmt.Sprintf("[%s] %s <span title=\"%s\">[Error]</span> Could not parse upload.",
			time.Now().Format(timeFormat), formatedHash, err.Error())
		return
	}

	urlA := strings.Split(r.URL.String(), "?")
	if len(urlA) < 1 {
		w.WriteHeader(500)
		logger <- fmt.Sprintf("[%s] %s [Error] Malformed GET string.", time.Now().Format(timeFormat), formatedHash)
		return
	}

	downNew := int64(*Down * float64(down))
	uploadNew := int64(*Up * float64(upload))

	if *Switch {
		t := downNew
		downNew = uploadNew
		uploadNew = t
	}

	r.Form.Set("downloaded", strconv.FormatInt(downNew, 10))
	r.Form.Set("uploaded", strconv.FormatInt(uploadNew, 10))

	r.URL, err = url.Parse(urlA[0] + "?" + r.Form.Encode())
	if err != nil {
		w.WriteHeader(500)
		logger <- fmt.Sprintf("[%s] %s <span title=\"%s\">[Error]</span> Could not build new request.",
			time.Now().Format(timeFormat), formatedHash, err.Error())
		return
	}

	resp, err := send(w, r)
	if err != nil {
		w.WriteHeader(500)
		logger <- fmt.Sprintf("[%s] %s <span title=\"%s\">[Error]</span> Could not send request.",
			time.Now().Format(timeFormat), formatedHash, err.Error())
		return
	}

	logger <- fmt.Sprintf("[%s] %s [Info] Upload: %s / %s, Download: %s / %s, Tracker: %s",
		time.Now().Format(timeFormat), formatedHash, sizeAbbreviation(upload),
		sizeAbbreviation(uploadNew), sizeAbbreviation(down), sizeAbbreviation(downNew),
		resp.Status)
}

// Send http request
func send(w http.ResponseWriter, r *http.Request) (resp *http.Response, err error) {
	err = nil
	r.RequestURI = ""
	client := new(http.Client)
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

// Writes out the log to http answer
func showLog(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<html><head><link rel=\"icon\" type=\"image/png\" href=\"/favicon.png\" /><title>Ratio mender " + Version + "</title>" +
		"<link rel=\"icon\" type=\"image/png\" href=\"/favicon.png\">" +
		"</head><body><pre>"))

	w.Write([]byte(fmt.Sprintf(
		"[%s] Ratio Mender Listening %s. Upload: x%f, Download: x%f, Switch: %v\n\n"+
			"Log format:\n"+
			"[Requested] [Torrent identifier] [Log level] Upload: actual / reported, Download: actual / reported, Tracker: tracker response\n\n",
		StartTime, ListeningOn, *Up, *Down, *Switch)))

	t := Log.Prev()
	for i := Log.Len(); i > 0; i-- {
		if v, ok := t.Value.([]byte); ok {
			w.Write(v)
		}
		t = t.Prev()
	}
	w.Write([]byte("</pre></body></html>"))
}

// Add log entries as they come in via the channel.
func rotateLog() {
	var msg string
	for {
		msg = <-logger
		Log.Value = []byte(msg + "\n")
		Log = Log.Next()
		if *LogToOutput {
			fmt.Println(msg)
		}
	}
}

// Dumps the favicon.
// Image from http://game-icons.net/
func faviconGenerator(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "favicon.png")
}

// Any number bigger the 4-5 digits gets skipped by our internet corrupted brain.
// We want meaningful data, thus this function makes it readable
func sizeAbbreviation(size int64) string {
	floatingSize := float64(size)
	if floatingSize/1099511627776 > 1 {
		return strconv.FormatFloat(floatingSize/1099511627776, 'f', 3, 64) + " TB"
	} else if floatingSize/1073741824 > 1 {
		return strconv.FormatFloat(floatingSize/1073741824, 'f', 3, 64) + " GB"
	} else if floatingSize/1048576 > 1 {
		return strconv.FormatFloat(floatingSize/1048576, 'f', 3, 64) + " MB"
	} else if floatingSize/1024 > 1 {
		return strconv.FormatFloat(floatingSize/1024, 'f', 3, 64) + " KB"
	}
	return strconv.FormatInt(size, 10) + " B"
}
