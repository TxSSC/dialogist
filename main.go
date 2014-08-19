package main

import (
	"encoding/json"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	Port     string
	ClipPath string
}

type Clip struct {
	Title    string `json:"title"`
	Location string `json:"location"`
}

var config Config

func init() {
	file, err := ioutil.ReadFile("config.json")

	if err != nil {
		log.Fatalf("There was an error while trying to read the config.")
	}

	json.Unmarshal(file, &config)
}

func main() {
	broker := &Broker{
		make(map[chan []byte]bool),
		make(chan []byte),
	}

	go broker.Process()

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal("Unable to initialize fsnotify:", err)
	}

	defer watcher.Close()

	watcher.Add(config.ClipPath)

	// Start watching
	go watchClips(watcher, broker)
	log.Println("Watching clips...")

	// Server routes
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.Handle("/events/", broker)
	http.HandleFunc("/clips/", serveClips)

	var port string

	if config.Port != "" {
		port = ":" + config.Port
	} else {
		port = ":3000"
	}

	log.Printf("Listening on port %v...", port[1:])
	log.Fatal(http.ListenAndServe(port, nil))
}

// List all clips in the clip directory
func serveClips(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile("/clips/(.+.mp3)")
	path := r.URL.Path

	// Handle serve request
	if re.MatchString(path) {
		match := re.FindStringSubmatch(path)
		file := config.ClipPath + "/" + match[1]
		http.ServeFile(w, r, file)
		return
	}

	var clips []Clip
	files, err := ioutil.ReadDir(config.ClipPath)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	for _, file := range files {
		clip := createClip(&file)
		clips = append(clips, clip)
	}

	msg, err := json.Marshal(clips)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(msg)
}

func createClip(file *os.FileInfo) Clip {
	title := (*file).Name()
	location := "/clips/" + (*file).Name()

	title = strings.Replace(title, ".mp3", "", -1)
	title = strings.Replace(title, "-", " ", -1)

	return Clip{
		Title:    title,
		Location: location,
	}
}

func watchClips(w *fsnotify.Watcher, b *Broker) {
	for {
		select {
		case event := <-w.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				file, err := os.Stat(event.Name)

				if err != nil {
					log.Println("Error stat'in created file:", err)
				}

				clip := createClip(&file)
				msg, err := json.Marshal(clip)

				if err != nil {
					log.Printf("Error serializing clip %v\n", clip)
				}

				b.Send(msg)
			}

		case err := <-w.Errors:
			log.Println("fsnotify error:", err)
		}
	}
}
