package rest

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"encoding/base64"

	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v2"
	rtsp "github.com/deepch/sample_rtsp"
	"github.com/wenwu-bianjie/rtsp2webrtc/service"
	//ice "github.com/pions/webrtc/internal/ice"
)
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

	vp8Track, err := peerConnection.NewTrack(webrtc.DefaultPayloadTypeH264, rand.Uint32(), "video", rtspUrl)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = peerConnection.AddTrack(vp8Track)
	if err != nil {
		log.Println(err)
		return
	}

	peerConnection.OnICEConnectionStateChange(func(rtspUrl string, vp8Track *webrtc.Track) func(webrtc.ICEConnectionState) {
		return func(connectionState webrtc.ICEConnectionState) {
			if connectionState.String() == "disconnected" {
				removeTrackToVideoTracks(rtspUrl, vp8Track)
				err = peerConnection.Close()
				if err != nil {
					log.Println(err)
				}
			}
			fmt.Printf("Connection State has changed %s \n", connectionState.String())
		}
	}(rtspUrl, vp8Track))

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

	err = addTrackToVideoTracks(rtspUrl, vp8Track)

	if err != nil {
		peerConnection.Close()
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%s", err)))
		return
	}

	w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(answer.SDP))))
}

func addTrackToVideoTracks (rtspUrl string, newTrack *webrtc.Track) error {
	service.VideoWebrtcTracks.Lock.Lock();
	defer service.VideoWebrtcTracks.Lock.Unlock();
	if track, ok := service.VideoWebrtcTracks.RtspTracks[rtspUrl]; ok {
		newTracks := append(track.Tracks, newTrack);
		service.VideoWebrtcTracks.RtspTracks[rtspUrl].Tracks = newTracks;
	} else {
		service.VideoWebrtcTracks.RtspTracks[rtspUrl] = &service.RtspTrack{
			Tracks: []*webrtc.Track{newTrack},
		}
	}
	// 创建 rtspClient
	if service.VideoWebrtcTracks.RtspTracks[rtspUrl].RtspClient == nil {
		client := rtsp.RtspClientNew()
		service.VideoWebrtcTracks.RtspTracks[rtspUrl].RtspClient = client
		err := service.NewRtspClient(client, rtspUrl)
		if err != nil {
			service.VideoWebrtcTracks.RtspTracks[rtspUrl].RtspClient = nil
			return err
		}
	}
	return nil
}

func removeTrackToVideoTracks (rtspUrl string, delTrack *webrtc.Track) {
	service.VideoWebrtcTracks.Lock.Lock();
	defer service.VideoWebrtcTracks.Lock.Unlock();

	if track, ok := service.VideoWebrtcTracks.RtspTracks[rtspUrl]; ok {
		tracks := track.Tracks
    	for i := 0; i < len(tracks); i++ {
    		if tracks[i] == delTrack {
        		newTracks := append(tracks[:i], tracks[i+1:]...)
				service.VideoWebrtcTracks.RtspTracks[rtspUrl].Tracks = newTracks
            	break
            }
        }
        if len(service.VideoWebrtcTracks.RtspTracks[rtspUrl].Tracks) == 0 {
        	client := service.VideoWebrtcTracks.RtspTracks[rtspUrl].RtspClient
        	if client != nil {
				client.Close()
				service.VideoWebrtcTracks.RtspTracks[rtspUrl].RtspClient = nil
        	}
        }
	}
}

