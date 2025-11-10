package godible

import (
	"fmt"
	"log/slog"
	"net/http"
	"text/template"
)

var templates = template.Must(template.ParseFiles("/tmp/index.html", "/tmp/index_header.html"))
var port = 1234

type Row struct {
	FilePath string
}

type PlayerHandlerPassthrough struct {
	player *Player
}

func (p *PlayerHandlerPassthrough) rootHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index_header", nil)
	// TODO: render this in a nice table with the possibility to highlight
	// the currently played track and even show and change the position
	// e.g. via 20 horizontally aligned css buttons (each button 5%) and
	// disabling the left-over buttons
	// - https://www.w3schools.com/css/css3_buttons.asp
	// - https://css-tricks.com/snippets/css/a-guide-to-flexbox/
	for e := p.player.TrackList.Front(); e != nil; e = e.Next() {
		track, ok := e.Value.(*Track)
		if !ok {
			slog.Error("expected value of type Track", "track", track)
		}
		renderTemplate(w, "index", &Row{FilePath: track.GetPath()})
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, r *Row) {
	err := templates.ExecuteTemplate(w, tmpl+".html", r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func InitHttpHandlers(p *Player) error {
	phPassthrough := &PlayerHandlerPassthrough{player: p}
	http.HandleFunc("/", phPassthrough.rootHandler)
	go func() {
		address := fmt.Sprintf(":%d", port)
		slog.Info("listen on ", "address", address)
		err := http.ListenAndServe(address, nil)
		slog.Error("ListenAndServe failed", "address", address, "error", err)
	}()
	return nil
}
