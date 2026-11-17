package impl

import (
	"regexp"
	"strings"
	"testing"

	"github.com/oraksil/orakki/internal/domain/models"
	"github.com/oraksil/orakki/pkg/utils"
	"github.com/pion/webrtc/v2"
)

// A browser offering recvonly audio/video must get a sendonly answer.
func TestProcessNewOfferAnswersRecvonlyWithSendonly(t *testing.T) {
	me := webrtc.MediaEngine{}
	me.RegisterCodec(webrtc.NewRTPH264Codec(webrtc.DefaultPayloadTypeH264, 90000))
	me.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	browser, err := webrtc.NewAPI(webrtc.WithMediaEngine(me)).NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer browser.Close()

	recvonly := webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}
	for _, kind := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeAudio, webrtc.RTPCodecTypeVideo} {
		if _, err := browser.AddTransceiverFromKind(kind, recvonly); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := browser.CreateDataChannel("message", nil); err != nil {
		t.Fatal(err)
	}

	offer, err := browser.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(offer.SDP, "a=recvonly") {
		t.Fatalf("test offer is not recvonly:\n%s", offer.SDP)
	}

	b64Offer, _ := utils.EncodeToB64EncodedJsonStr(&offer)
	session := NewWebRTCSession("", "", 0)
	answerInfo, err := session.ProcessNewOffer(models.SdpInfo{
		Peer:             models.PeerInfo{PlayerId: 1},
		SdpBase64Encoded: b64Offer,
	})
	if err != nil {
		t.Fatal(err)
	}

	var answer webrtc.SessionDescription
	if err := utils.DecodeFromB64EncodedJsonStr(answerInfo.SdpBase64Encoded, &answer); err != nil {
		t.Fatal(err)
	}

	if n := len(regexp.MustCompile(`a=sendonly`).FindAllString(answer.SDP, -1)); n != 2 {
		t.Fatalf("want a=sendonly for audio and video, got %d:\n%s", n, answer.SDP)
	}
	// the data channel section legitimately carries a=sendrecv; media sections must not
	for _, sec := range strings.Split(answer.SDP, "m=")[1:] {
		if (strings.HasPrefix(sec, "audio") || strings.HasPrefix(sec, "video")) && strings.Contains(sec, "a=sendrecv") {
			t.Fatalf("media section answered sendrecv:\n%s", sec)
		}
	}
}
