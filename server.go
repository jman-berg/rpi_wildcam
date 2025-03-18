package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/pion/webrtc/v4"
)

func parseNALUnits(data []byte) [][]byte {
    var nals [][]byte
    i := 0
    for i < len(data) {
        if i+3 < len(data) && data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 {
            start := i + 3
            i += 3
            for i < len(data) && !(data[i] == 0 && data[i+1] == 0 && data[i+2] == 1) {
                i++
            }
            end := i
            if i == len(data) {
                end = len(data)
            }
            nal := data[start:end]
            nals = append(nals, nal)
        } else if i+4 < len(data) && data[i] == 0 && data[i+1] == 0 && data[i+2] == 0 && data[i+3] == 1 {
            start := i + 4
            i += 4
            for i < len(data) && !(data[i] == 0 && data[i+1] == 0 && data[i+2] == 0 && data[i+3] == 1) {
                i++
            }
            end := i
            if i == len(data) {
                end = len(data)
            }
            nal := data[start:end]
            nals = append(nals, nal)
        } else {
            i++
        }
    }
    return nals
}

func main(){
	cmd := exec.Command("gst-launch-1.0", "v4l2src", "device=/dev/video0", "!", "video/x-raw,format=YUY2,width=1920,height=1080", "!", "x264enc", "key-int-max=30", "!", "h264parse", "!", "fdsink")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	defer cmd.Process.Kill()
}

config := webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}
//Peer connectie opzetten met de config van hierboven. We maken gebruik van een stunserver die de peer to peer verbinding regelt.
pc, err := webrtc.NewPeerConnection(config)
if err  != nil {
	panic(err)
}
//Een video track maken om toe te voegen aan de peer connectie. 
track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: "video/h264"}, "video", "stream")
if err != nil {
	panic(err)
}
//Track toevoegen aan de peer connectie.
_, err = pc.AddTrack(track)
if err != nil {
	panic(err)
}
//Hier maken we een packetizer die de videostream als packets kan opdelen in stukjes van 1400 bytes. Dit is nodig vanwege de bandbreedte van een gemiddeld netwerk.
codec := webrtc.RTPCodecParameters{
	RTPCodecCapability: webrtc.RTPCodecCapability{
		MimeType: "video/h264", 
		ClockRate: 90000,
	},
}
packetizer := webrtc.NewRTPH264Packetizer(track.SSRC(), track.SequenceNumberAtomic(), track.TimestampAtomic(), 1400, codec)
//Hier lezen we de gstreamer stream en schrijven we de geparseerde NAL units naar de eerder gedefinieerde track. Die dan weer toegevoegd is aan de eerder gedefinieerde peer connectie. 
reader := bufio.NewReader(stdout)
buf := make([]byte, 1024)
go func() {
	for {
		n, err := reader.Read(buf)
		if err != nil {
		fmt.Println("Error reading from Gstreamer:", err)
		return
		}
		nals := parseNALUnits(buf[:n])
		for _, nal := range nals {
			packets := packetizer.Packetize(nal, 90000)
			for _, pkt := range packets {
				if err := track.WriteRTP(pkt); err != nil {
					fmt.Println("Error writing RTP packet:", err)
				}
			}
		}

	}
}()

//HTTP Server voor de webpagina

mux := http.NewServeMux()
mux.Handlefunc("/", func(w http.ResponseWriter, r *http.Request) {

})


