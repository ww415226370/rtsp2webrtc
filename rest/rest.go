package rest

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"strings"

	"encoding/base64"

	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v2"
	//ice "github.com/pions/webrtc/internal/ice"
)

type Track []*webrtc.Track

type WebrtcTracks struct {
	Tracks map[string]Track
	lock   sync.RWMutex
}

var VideoTracks = WebrtcTracks{
	Tracks: map[string]Track{},
}
// var DataChanelTest chan<- webrtc.RTCSample

func StartHTTPServer() {
	r := mux.NewRouter()
	r.HandleFunc("/recive", HTTPHome)
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static/"))))
	fmt.Println("server listen in 8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Println(err)
		return
	}
}
func HTTPHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	data := r.FormValue("data")
	rtspUrl := r.FormValue("rtspUrl")

	sd, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Println(err)
		return
	}
	createVideoTrack(sd, rtspUrl, w)
}


func createVideoTrack(sd []byte, rtspUrl string, w http.ResponseWriter) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	rtspUrlSlice := strings.Split(rtspUrl, ",")
	for _, v := range rtspUrlSlice {
		vp8Track, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "video", v)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = peerConnection.AddTrack(vp8Track)
		if err != nil {
			log.Println(err)
			return
		}
		addTrackToVideoTracks(v, vp8Track)
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(sd),
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		log.Println(err)
		return
	}
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Println(err)
		return
	}
	w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(answer.SDP))))
	
}


func addTrackToVideoTracks (rtspUrl string, vp8Track *webrtc.Track) {
	VideoTracks.lock.Lock();
	defer VideoTracks.lock.Unlock();

	if tracks, ok := VideoTracks.Tracks[rtspUrl]; ok {
		newTracks := append(tracks, vp8Track);
		VideoTracks.Tracks[rtspUrl] = newTracks;
	} else {
		VideoTracks.Tracks[rtspUrl] = []*webrtc.Track{vp8Track}
	}
}