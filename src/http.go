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
	),
)
var playerWebGuiPort = 1234

var upgrader = websocket.Upgrader{
	// XXX: Currently, CheckOrigin in Upgrader allows all connections.
	// TODO: Check r.Host or r.Header[Origin]?
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Row struct {
	Fullpath        string `json:"fullpath"`
	Basename        string `json:"basename"`
	Dirname         string `json:"dirname"`
	CurrentSeconds  int64  `json:"current_seconds"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type PlayerHandlerPassthrough struct {
	player *Player
}

func trackBasename(track *Track) string {
	base := filepath.Base(track.GetPath())
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func trackDirname(track *Track) string {
	dir := filepath.Dir(track.GetPath())
	dir_without_datadir := strings.TrimPrefix(dir, strings.TrimSuffix(DATADIR, "/"))
	if strings.HasPrefix(dir_without_datadir, "/") {
		return dir_without_datadir
	}
	return "/" + dir_without_datadir
}

func trackToRow(track *Track) Row {
	return Row{
		Fullpath:        track.GetPath(),
		Basename:        trackBasename(track),
		Dirname:         trackDirname(track),
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

type Tbody struct {
	Header string
	Rows   []Row
}

type Data struct {
	Tbodies []Tbody
}

func (d *Data) addRow(header string, row Row) {
	for i := range d.Tbodies {
		if d.Tbodies[i].Header == header {
			d.Tbodies[i].Rows = append(d.Tbodies[i].Rows, row)
			return
		}
	}
	d.Tbodies = append(d.Tbodies, Tbody{Header: header, Rows: []Row{row}})
}

func renderTemplate(w http.ResponseWriter, filename string, data *Data) {
	tmpl, err := template.ParseFS(assetsFS, "assets/tmpl/"+filename+".tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (p *PlayerHandlerPassthrough) rootHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "header", nil)
	renderTemplate(w, "body", nil)
}

type WebsocketApiRequest struct {
	Type    string `json:"type"`
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

func (p *PlayerHandlerPassthrough) handleCommand(req WebsocketApiRequest) {
	switch req.Type {
	case "toggle":
		p.player.Command(TOGGLE)
	case "next":
		p.player.Command(NEXT)
	case "previous":
		p.player.Command(PREVIOUS)
	case "slide":
		duration_current_to_set, err := strconv.Atoi(req.Payload)
		if err != nil {
			slog.Error("handleCommand 'slide' can not convert payload to integer", "err", err)
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
		if duration != 0 {
			div := float64(duration_current_to_set) / float64(duration)
			position = int64(div * float64(length))
			position = position - (position % 4)
		}

		track.SetPosition(position)
		//p.player.addQueueElement(p.player.current) // XXX: not necessary as current stays the same
		p.player.Command(TOGGLE)
	default:
		slog.Error("unknown WebsocketApiRequest type", "type", req.Type)
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
		var req WebsocketApiRequest
		err = decoder.Decode(&req)
		if err != nil {
			slog.Error("failed to decode message as WebsocketApiRequest", "message", message, "err", err)
			continue
		}
		p.handleCommand(req)
	}
}

func (p *PlayerHandlerPassthrough) wsWriteState(conn *websocket.Conn) {
	jsonstate, _ := json.Marshal(p.state())
	req, _ := json.Marshal(WebsocketApiRequest{
		Type:    "state",
		Payload: string(jsonstate),
	})
	err := conn.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		slog.Error("writing state via websocket connection failed", "req", req, "err", err)
		return
	}
}

func (p *PlayerHandlerPassthrough) wsWriteRows(conn *websocket.Conn) {
	jsonrows, _ := json.Marshal(p.trackListToRows())
	req, _ := json.Marshal(WebsocketApiRequest{
		Type:    "rows",
		Payload: string(jsonrows),
	})
	err := conn.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		slog.Error("writing state via websocket connection failed", "req", req, "err", err)
		return
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
		p.wsWriteState(conn)
		p.wsWriteRows(conn)
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
	http.HandleFunc("/fonts/", assetsFileServer)
	phPassthrough := &PlayerHandlerPassthrough{player: p}
	http.HandleFunc("/", phPassthrough.rootHandler)
	http.HandleFunc("/ws", phPassthrough.wsHandler)

	go func() {
		address := fmt.Sprintf("0.0.0.0:%d", playerWebGuiPort)
		slog.Info("listen on ", "address", address)
		err := http.ListenAndServe(address, nil)
		slog.Error("ListenAndServe failed", "address", address, "error", err)
	}()
	return nil
}
