package godible

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
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

// FIXME:
// - instead of printing row per row, pass a map of directories and files
// - instead of tail.tmpl use a directory/file-tree like https://stackoverflow.com/a/51617657
// - templates can use `range`
// - in order to keep the directory tree's alphabetical order, pass also ordered keys next to the map, c.f.: https://stackoverflow.com/a/18342865
type Row struct {
	Basename        string
	Dirname         string
	CurrentSeconds  int64
	DurationSeconds int64
	Playing         bool
}

type PlayerHandlerPassthrough struct {
	player *Player
}

func trackBasename(track *Track) string {
	base := filepath.Base(track.GetPath())
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func trackToRow(track *Track) Row {
	return Row{
		Basename:        trackBasename(track),
		Dirname:         filepath.Dir(track.GetPath()),
		CurrentSeconds:  track.CurrentSeconds(),
		DurationSeconds: track.duration,
	}
}

func (p *PlayerHandlerPassthrough) trackListToRows() []Row {
	ret := make([]Row, p.player.TrackList.Len())

	element := p.player.TrackList.Front()
	if element == nil {
		slog.Error("failed to transform tracklist into gui rows: no tracks found")
		return nil
	}

	for i := range ret {
		track, ok := element.Value.(*Track)
		if !ok {
			slog.Error("expected value of type Track", "track", track)
			continue
		}
		ret[i] = trackToRow(track)
		element = element.Next()
	}
	return ret
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
	if p.player == nil || p.player.TrackList == nil {
		return
	}
	for e := p.player.TrackList.Front(); e != nil; e = e.Next() {
		track, ok := e.Value.(*Track)
		if !ok {
			slog.Error("expected value of type Track", "track", track)
		}
		row := trackToRow(track)
		renderTemplate(w, "tail", &row)
	}
}

type HttpCommand struct {
	Cmd     string `json:"command"`
	Payload string `json:"payload"`
}

type HttpState struct {
	IsPlaying       bool   `json:"is_playing"`
	Name            string `json:"name"`
	Position        int64  `json:"position"`
	Length          int64  `json:"length"`
	Duration        int64  `json:"duration"`
	DurationCurrent int64  `json:"duration_current"`
}

func (p *PlayerHandlerPassthrough) state() *HttpState {
	current := p.player.getCurrent()
	if current == nil {
		return nil
	}

	name := current.GetPath()
	position := current.GetPosition()
	length := current.GetLength()
	duration := current.GetDuration()
	durationCurrent := duration
	if length > 0 {
		var tmp float64 = float64(position) / float64(length)
		tmp = tmp * float64(durationCurrent)
		durationCurrent = int64(tmp)
	}

	return &HttpState{
		IsPlaying:       p.player.playing,
		Name:            name,
		Position:        position,
		Length:          length,
		Duration:        duration,
		DurationCurrent: durationCurrent,
	}
}

func (p *PlayerHandlerPassthrough) stateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		state := p.state()
		j, _ := json.Marshal(state)
		w.Write(j)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "only GET supported")
	}
}

var upgrader = websocket.Upgrader{
	// XXX: Currently, CheckOrigin in Upgrader allows all connections.
	// TODO: Check r.Host or r.Header[Origin]?
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (p *PlayerHandlerPassthrough) handleCommand(cmd HttpCommand) {
	switch cmd.Cmd {
	case "toggle":
		p.player.Command(TOGGLE)
	case "next":
		p.player.Command(NEXT)
	case "previous":
		p.player.Command(PREVIOUS)
	case "jump":
		duration_current_to_set, err := strconv.Atoi(cmd.Payload)
		if err != nil {
			slog.Error("handleCommand jump can not convert payload to integer", "err", err)
			return
		}

		if p.player.playing {
			p.player.Command(TOGGLE)
			//FIXME: replace sleep synchronizing the cancelfunc (as in: toggle is prohibited if a cancelfunc is running)
			time.Sleep(50 * time.Millisecond)
		}

		track := p.player.getCurrent()
		length := track.GetLength()
		duration := track.GetDuration()

		var position int64
		position = 0
		slog.Debug("XXX jump", "duration_current_to_set", duration_current_to_set, "length", length)
		if duration != 0 {
			div := float64(duration_current_to_set) / float64(duration)
			position = int64(div * float64(length))
			position = position - (position % 4)
		}
		slog.Debug("XXX jump", "position", position)

		track.SetPosition(position)
		//p.player.addQueueElement(p.player.current) // XXX: not necessary as current stays the same
		p.player.Command(TOGGLE)
	default:
		slog.Error("unknown command", "cmd", cmd)
	}
}

func (p *PlayerHandlerPassthrough) wsReader(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			slog.Error("wsReader err", "err", err)
			break
		}

		decoder := json.NewDecoder(bytes.NewReader(message))
		var cmd HttpCommand
		err = decoder.Decode(&cmd)
		if err != nil {
			slog.Error("failed to decode message as HttpCommand", "message", message, "err", err)
			continue
		}
		p.handleCommand(cmd)
	}
}

func (p *PlayerHandlerPassthrough) wsWriter(conn *websocket.Conn) {
	// Time allowed to write the message to the client.
	writeWait := 600 * time.Millisecond
	// Send messages to peer with this period. Must be less than writeWait.
	sendPeriod := 500 * time.Millisecond

	sendTicker := time.NewTicker(sendPeriod)
	defer func() {
		sendTicker.Stop()
		conn.Close()
	}()

	for range sendTicker.C {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		jsonstate, _ := json.Marshal(p.state())
		err := conn.WriteMessage(websocket.TextMessage, jsonstate)
		if err != nil {
			slog.Error("writing state via websocket connection failed", "jsonstate", jsonstate, "err", err)
			return
		}
	}
}

func (p *PlayerHandlerPassthrough) wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "only GET and POST supported")
		return
	}

	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade err", "err", err)
		return
	}
	defer connection.Close()

	go p.wsWriter(connection)
	p.wsReader(connection)
}

func assetsFileServer(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// TODO: further security measures as path sanitizing
		http.ServeFileFS(w, r, assetsFS, "assets"+r.URL.Path)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "only GET supported")
	}
}

func InitHttpHandlers(p *Player) error {
	http.HandleFunc("/css/", assetsFileServer)
	http.HandleFunc("/img/", assetsFileServer)
	http.HandleFunc("/js/", assetsFileServer)
	phPassthrough := &PlayerHandlerPassthrough{player: p}
	http.HandleFunc("/", phPassthrough.rootHandler)
	http.HandleFunc("/state", phPassthrough.stateHandler)
	http.HandleFunc("/ws", phPassthrough.wsHandler)

	go func() {
		address := fmt.Sprintf("0.0.0.0:%d", port)
		slog.Info("listen on ", "address", address)
		err := http.ListenAndServe(address, nil)
		slog.Error("ListenAndServe failed", "address", address, "error", err)
	}()
	return nil
}
