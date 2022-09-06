package rhttp

import (
	"bytes"
	"fmt"
	"github.com/ruraomsk/camserver/rtsp"
	"io"
	"net/http"
	"strings"
	"time"
)

var frame = []byte("--frame")
var endFrame = []byte{13, 10, 13, 10}

type RStream struct {
	Rstp  *rtsp.Stream
	Rhttp *Stream
	Type  int
}
type RPacket struct {
	Packet  *rtsp.Packet
	RPacket []byte
	Type    int
}

func (p *RPacket) IsVideo() bool {
	switch p.Type {
	case 0:
		return p.Packet.IsVideo()
	case 1:
		if len(p.RPacket) == 0 {
			return false
		}
		return true
	}
	return false
}
func (p *RPacket) Data() []byte {
	switch p.Type {
	case 0:
		return p.Packet.Data()
	case 1:
		return p.RPacket
	}
	return make([]byte, 0)
}

type Stream struct {
	Client http.Client
	Url    string
	Resp   *http.Response
}

func OpenHttp(uri string) (s *Stream, err error) {
	s = new(Stream)
	s.Client = http.Client{Timeout: 20 * time.Second}
	s.Url = uri
	s.Resp, err = s.Client.Get(uri)
	return s, nil
}
func Open(uri string) (s *RStream, err error) {
	s = new(RStream)
	if strings.HasPrefix(uri, "rtsp:") {
		s.Rstp, err = rtsp.Open(uri)
		s.Type = 0
		return
	}
	if strings.HasPrefix(uri, "http:") {
		s.Rhttp, err = OpenHttp(uri)
		s.Type = 1
		return
	}
	err = fmt.Errorf("неверный тип подключения к камере %s", uri)
	return
}
func (r *RStream) ReadPacket() (packet *RPacket, err error) {
	packet = new(RPacket)
	packet.Type = r.Type
	switch r.Type {
	case 0:
		packet.Packet, err = r.Rstp.ReadPacket()
		return
	case 1:
		r.Rhttp.Resp, err = r.Rhttp.Client.Get(r.Rhttp.Url)
		packet.RPacket = make([]byte, 186000)
		_, err = io.ReadAtLeast(r.Rhttp.Resp.Body, packet.RPacket, len(packet.RPacket))
		if bytes.HasPrefix(packet.RPacket, frame) {
			pos := bytes.Index(packet.RPacket, endFrame)
			if pos > 0 {
				packet.RPacket = packet.RPacket[pos+len(endFrame):]
			}
			pos = bytes.Index(packet.RPacket, frame)
			if pos > 0 {
				packet.RPacket = packet.RPacket[:pos]
			}
		}
		return
	}
	err = fmt.Errorf("неверный тип подключения к камере %d", r.Type)
	return
}
