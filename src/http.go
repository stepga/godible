package godible

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"text/template"
)

//go:embed assets/*
var assetsFS embed.FS

var templates = template.Must(
	template.ParseFS(
		assetsFS,
		"assets/tmpl/header.tmpl",
		"assets/tmpl/body.tmpl",
		"assets/tmpl/tail.tmpl",
	),
)
var port = 1234

type Row struct {
	FilePath string
}

type PlayerHandlerPassthrough struct {
	player *Player
}

func renderTemplate(w http.ResponseWriter, filename string, r *Row) {
	tmpl, err := template.ParseFS(assetsFS, "assets/tmpl/"+filename+".tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (p *PlayerHandlerPassthrough) rootHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "header", nil)
	renderTemplate(w, "body", nil)
	// TODO: render this in a nice table with the possibility to highlight
	// the currently played track and even show and change the position
	// e.g. via 20 horizontally aligned css buttons (each button 5%) and
	// disabling the left-over buttons
	// - https://www.w3schools.com/css/css3_buttons.asp
	// - https://css-tricks.com/snippets/css/a-guide-to-flexbox/
	if p.player == nil || p.player.TrackList == nil {
		return
	}
	for e := p.player.TrackList.Front(); e != nil; e = e.Next() {
		track, ok := e.Value.(*Track)
		if !ok {
			slog.Error("expected value of type Track", "track", track)
		}
		renderTemplate(w, "tail", &Row{FilePath: track.GetPath()})
	}
}

type State struct {
	IsPlaying bool   `json:"is_playing"`
	Name      string `json:"name"`
	Position  int64  `json:"position"`
	Length    int64  `json:"length"`
}

func (p *PlayerHandlerPassthrough) stateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		state := &State{
			IsPlaying: p.player.playing,
			Name:      p.player.getCurrent().GetPath(),
			Position:  p.player.getCurrent().GetPosition(),
			Length:    p.player.getCurrent().GetLength(),
		}
		j, _ := json.Marshal(state)
		w.Write(j)
	case "POST":
		// TODO: implement me
		//// Decode the JSON in the body and update the player/track state
		//d := json.NewDecoder(r.Body)
		//s := &State{}
		//err := d.Decode(s)
		//if err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//}
		//...
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "only GET and POST supported")
	}
}
func assetsFileServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// TODO: further security measures as path sanitizing
		http.ServeFileFS(w, r, assetsFS, "assets"+r.URL.Path)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "only GET and POST supported")
	}
}

func InitHttpHandlers(p *Player) error {
	http.HandleFunc("/css/", assetsFileServer)
	http.HandleFunc("/js/", assetsFileServer)
	phPassthrough := &PlayerHandlerPassthrough{player: p}
	http.HandleFunc("/", phPassthrough.rootHandler)
	http.HandleFunc("/state", phPassthrough.stateHandler)

	go func() {
		address := fmt.Sprintf(":%d", port)
		slog.Info("listen on ", "address", address)
		err := http.ListenAndServe(address, nil)
		slog.Error("ListenAndServe failed", "address", address, "error", err)
	}()
	return nil
}

// XXX: debugging embed.FS (source https://gist.github.com/clarkmcc/1fdab4472283bb68464d066d6b4169bc?permalink_comment_id=4405804#gistcomment-4405804)
//func getAllFilenames(efs *embed.FS) (files []string, err error) {
//	if err := fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
//		if d.IsDir() {
//			return nil
//		}
//		files = append(files, path)
//		return nil
//	}); err != nil {
//		return nil, err
//	}
//	return files, nil
//}
