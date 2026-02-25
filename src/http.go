package godible

import (
	"bytes"
	"crypto/sha1"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed assets/*
var assetsFS embed.FS

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
	FullpathHashSum string `json:"fullpath_hash_sum"`
	Basename        string `json:"basename"`
	DirnameShow     string `json:"dirname_show"`
	DirnameHashSum  string `json:"dirname_hash_sum"`
	DirnameFull     string `json:"dirname_full"`
	CurrentSeconds  int64  `json:"current_seconds"`
	DurationSeconds int64  `json:"duration_seconds"`
	RfidUid         string `json:"rfid_uid"`
	HashSum         string `json:"hash_sum"`
}

func (row *Row) setHashSum() error {
	oldHashSum := row.HashSum
	row.HashSum = ""
	jsonEnc, err := json.Marshal(row)
	if err != nil {
		row.HashSum = oldHashSum
		return err
	}
	row.HashSum = fmt.Sprintf("%x", sha1.Sum(jsonEnc))
	return nil
}

type PlayerHandlerPassthrough struct {
	*Player
}

func (p *PlayerHandlerPassthrough) trackToRow(track *Track) Row {
	row := Row{
		Fullpath:        track.Path,
		FullpathHashSum: fmt.Sprintf("%x", sha1.Sum([]byte(track.Path))),
		Basename:        track.Basename(),
		DirnameShow:     track.DirnameShow(),
		DirnameHashSum:  fmt.Sprintf("%x", sha1.Sum([]byte(track.DirnameFull()))),
		DirnameFull:     track.DirnameFull(),
		CurrentSeconds:  track.CurrentSeconds(),
		DurationSeconds: track.duration,
		RfidUid:         p.GetRfidUidForTrack(track),
	}
	err := row.setHashSum()
	if err != nil {
		slog.Error("failed to calculate hash sum for row", "row", row, "err", err)
	}
	return row
}

func (p *PlayerHandlerPassthrough) trackListToRows() []Row {
	ret := make([]Row, p.TrackList.Len())

	element := p.TrackList.Front()
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
		ret[i] = p.trackToRow(track)
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

func renderTemplate(w http.ResponseWriter, filename string, data *Data) {
	tmpl, err := template.ParseFS(assetsFS, "assets/tmpl/"+filename+".tmpl.html")
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
	RfidTrackLearn  string `json:"rfid_track_learn"`
}

func (p *PlayerHandlerPassthrough) state() *HttpState {
	current := p.getCurrent()
	if current == nil {
		return nil
	}

	name := current.Basename()
	position := current.position
	length := current.length
	duration := current.duration
	durationCurrent := int64(0)
	if length > 0 {
		var tmp float64 = float64(position) / float64(length)
		tmp = tmp * float64(duration)
		durationCurrent = int64(tmp)
	}
	rfidTrackLearn := ""
	if p.rfidTrackLearn != nil {
		rfidTrackLearn = p.rfidTrackLearn.TrackPath
	}

	return &HttpState{
		IsPlaying:       p.playing,
		Name:            name,
		Position:        position,
		Length:          length,
		Duration:        duration,
		DurationCurrent: durationCurrent,
		RfidTrackLearn:  rfidTrackLearn,
	}
}

func (p *PlayerHandlerPassthrough) handleCommand(req WebsocketApiRequest) {
	switch req.Type {
	case "toggle":
		p.Command(TOGGLE)
	case "next":
		p.Command(NEXT)
	case "previous":
		p.Command(PREVIOUS)
	case "slide":
		duration_current_to_set, err := strconv.Atoi(req.Payload)
		if err != nil {
			slog.Error("handleCommand 'slide' can not convert payload to integer", "err", err)
			return
		}

		if p.playing {
			p.Command(TOGGLE)
			//FIXME: replace sleep synchronizing the cancelfunc (as in: toggle is prohibited if a cancelfunc is running)
			time.Sleep(50 * time.Millisecond)
		}

		track := p.getCurrent()
		length := track.length
		duration := track.duration

		var position int64
		position = 0
		if duration > 0 {
			div := float64(duration_current_to_set) / float64(duration)
			position = int64(div * float64(length))
			position = position - (position % 4)
		}

		track.SetPosition(position)
		p.Command(TOGGLE)
	case "rfidtracklearn":
		// TODO: re-implement me
		// - if currently not already learning: set the track to learn
		// - if payload is directory: save directory -> extend the data structs
		path := req.Payload
		slog.Debug("XXX: rfidtracklearn", "path", path)
		rfidTrackLearn := p.NewRfidTrackLearn(path)
		if rfidTrackLearn == nil {
			slog.Error("rfidtracklearn: could not find respective track for given payload", "payload", req.Payload)
			return
		}
		p.rfidTrackLearn = rfidTrackLearn
		go func() {
			// TODO: just introduce a cancelfunc here?
			slog.Info("rfidTrackLearn: try to reset the player's rfid learn mode")
			time.Sleep(10 * time.Second)

			if p.rfidTrackLearn == nil {
				return
			}
			if p.rfidTrackLearn.TimeStamp != rfidTrackLearn.TimeStamp {
				return
			}
			p.rfidTrackLearn = nil
		}()
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

func (p *PlayerHandlerPassthrough) wsWriteState(conn *websocket.Conn) bool {
	jsonstate, _ := json.Marshal(p.state())
	req, _ := json.Marshal(WebsocketApiRequest{
		Type:    "state",
		Payload: string(jsonstate),
	})
	err := conn.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		slog.Error("writing state via websocket connection failed", "req", req, "err", err)
		return false
	}
	return true
}

func (p *PlayerHandlerPassthrough) wsWriteHideRfidAlertBox(conn *websocket.Conn) bool {
	if p.rfidTrackLearn != nil {
		// nothing to un-show/hide
		return true
	}
	req, _ := json.Marshal(WebsocketApiRequest{
		Type: "hiderfidalertbox",
	})
	err := conn.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		slog.Error("writing hiderfidalertbox via websocket connection failed", "req", req, "err", err)
		return false
	}
	return true
}

func (p *PlayerHandlerPassthrough) wsWriteRows(conn *websocket.Conn) bool {
	jsonrows, _ := json.Marshal(p.trackListToRows())
	req, _ := json.Marshal(WebsocketApiRequest{
		Type:    "rows",
		Payload: string(jsonrows),
	})
	err := conn.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		slog.Error("writing rows via websocket connection failed", "req", req, "err", err)
		return false
	}
	return true
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
		if !p.wsWriteState(conn) || !p.wsWriteRows(conn) || !p.wsWriteHideRfidAlertBox(conn) {
			slog.Error("abort (broken?) wsWriter routine due to erros")
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
	http.HandleFunc("/fonts/", assetsFileServer)
	phPassthrough := &PlayerHandlerPassthrough{p}
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
