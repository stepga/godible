package godible

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

var upgrader = websocket.Upgrader{
	// XXX: Currently, CheckOrigin in Upgrader allows all connections.
	// TODO: Check r.Host or r.Header[Origin]?
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (p *PlayerHandlerPassthrough) wsReader(conn *websocket.Conn) {
	for {
		slog.Debug("XXX new wsRead loop wait")
		//messageType, message, err := conn.ReadMessage()
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
		slog.Debug("XXX", "decoded cmd", cmd)

		// TODO: handle commands

		//err = conn.WriteMessage(messageType, message)
		//if err != nil {
		//	slog.Error("wsReader reply err", "err", err)
		//	break
		//}
	}
}

func (p *PlayerHandlerPassthrough) wsWriter(conn *websocket.Conn) {
	// Time allowed to write the message to the client.
	writeWait := 500 * time.Millisecond
	sendPeriod := 500 * time.Millisecond
	pingPeriod := 250 * time.Millisecond

	sendTicker := time.NewTicker(sendPeriod)
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		sendTicker.Stop()
		pingTicker.Stop()
		conn.Close()
	}()

	for {
		select {
		case <-sendTicker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			jsonstate, _ := json.Marshal(p.state())
			if err := conn.WriteMessage(websocket.TextMessage, jsonstate); err != nil {
				return
			}
		// TODO: test the pingTicker ... is this useful at all?
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				slog.Error("XXX ping timeout?", "err", err)
				return
			}
		}
	}
}

// TODO: implement ping/pong feature (c.f. https://github.com/gorilla/websocket/blob/main/examples/filewatch/main.go#L73)
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
		fmt.Fprintf(w, "only GET and POST supported")
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
