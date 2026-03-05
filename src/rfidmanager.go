package godible

import (
	"fmt"
	"log/slog"
	"time"
)

const (
	TrackTrainingSeconds = 10
)

// TODO: replace Track with struct that contains "Track" or "Directory and Track"
type TrackTrainer struct {
	Track     *Track
	TimeStamp int64
	Done      bool
	TimeLeft  int64
}

func newTrackTrainer(track *Track) *TrackTrainer {
	return &TrackTrainer{
		Track:     track,
		TimeStamp: time.Now().UnixNano(),
		TimeLeft:  TrackTrainingSeconds,
	}
}

func (t *TrackTrainer) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf(
		"TrackTrainer{Track: %s, TimeStamp: %d, Done: %t}",
		t.Track.Path,
		t.TimeStamp,
		t.Done,
	)
}

// In order to support a RFID UID <-> Directory mapping, extend the Track type
// with a directory path. During playing the directory, the track pointer moves
// also on, pointing to the last track being played.
type TrackMapping struct {
	*Track
	Directory string
}

type RfidTrackManager struct {
	UidTrackMap  map[string]*TrackMapping
	TrackTrainer *TrackTrainer
}

func newRfidTrackManager() *RfidTrackManager {
	return &RfidTrackManager{
		UidTrackMap: make(map[string]*TrackMapping),
	}
}

func (rtm *RfidTrackManager) GetTrack(rfidUid string) *Track {
	el, ok := rtm.UidTrackMap[rfidUid]
	if ok {
		return el.Track
	}
	return nil
}

// TODO also implement directory case
func (rtm *RfidTrackManager) GetUid(track *Track) string {
	for key, value := range rtm.UidTrackMap {
		if track == value.Track {
			return key
		}
	}
	return ""
}

// TODO also implement directory case
// Set a new RFID UID Track mapping. An already existing mapping with the
// given RFID UID will be deleted.
func (rtm *RfidTrackManager) SetMapping(rfidUid string, track *Track) {
	existingTrack := rtm.GetTrack(rfidUid)
	if existingTrack != nil {
		delete(rtm.UidTrackMap, rfidUid)
	}
	rtm.UidTrackMap[rfidUid] = &TrackMapping{Track: track, Directory: ""}
}

func (rtm *RfidTrackManager) runTrackTrainerCountdown(oldTrackTrainer *TrackTrainer) {
	slog.Debug("runTrackTrainerCountdown: begin", "oldTrackTrainer", oldTrackTrainer.String())
	for range TrackTrainingSeconds {
		time.Sleep(1 * time.Second)
		if rtm.TrackTrainer == nil || rtm.TrackTrainer.TimeStamp != oldTrackTrainer.TimeStamp {
			slog.Debug("runTrackTrainerCountdown: nothing to reset, training already completed")
			return
		}
		rtm.TrackTrainer.TimeLeft = rtm.TrackTrainer.TimeLeft - 1
	}
	rtm.TrackTrainer = nil
	slog.Debug("runTrackTrainerCountdown: TrackTrainer reset", "oldTrackTrainer", oldTrackTrainer.String())
}

func (rtm *RfidTrackManager) SetTrackTrainer(track *Track) bool {
	if rtm.TrackTrainer != nil {
		return false
	}
	rtm.TrackTrainer = newTrackTrainer(track)
	go rtm.runTrackTrainerCountdown(rtm.TrackTrainer)
	return true
}
