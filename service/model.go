package service

import (
	"sync"
	"github.com/pion/webrtc/v2"
	rtsp "github.com/deepch/sample_rtsp"
)

type RtspTrack struct {
	Tracks []*webrtc.Track
	RtspClient *rtsp.RtspClient
}

type RtspTrackList map[string]*RtspTrack

type WebrtcTracks struct {
	RtspTracks RtspTrackList
	Lock 	sync.RWMutex
}

var VideoWebrtcTracks WebrtcTracks = WebrtcTracks{
	RtspTracks: RtspTrackList{},
}

func GetVideoWebrtcTracks () WebrtcTracks{
	return VideoWebrtcTracks
}
