package twealtime

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"net/http"
)

type TrackMention struct {
	Tweet            string
	TwitterUser      string
	TwitterFollowers int
	TrackUri         string
}

type Server struct {
	sockets []*websocket.Conn
}

func NewServer() *Server {
	return new(Server)
}

func (server *Server) Send(data interface{}) (err error) {
	var bytes []byte

	switch data.(type) {
	case []byte:
		bytes = data.([]byte)
	case string:
		bytes = []byte(data.(string))
	default:
		bytes, err = json.Marshal(data)
	}

	if err != nil {
		return
	}

	// Write the data to all the sockets.
	for _, socket := range server.sockets {
		socket.Write(bytes)
	}

	return
}

func (server *Server) Serve(addr string) error {
	http.Handle("/stream", websocket.Handler(func(ws *websocket.Conn) {
		server.sockets = append(server.sockets, ws)

		// Start reading from the socket.
		for {
			buffer := make([]byte, 4096)
			if _, err := ws.Read(buffer); err != nil {
				break
			}
			// TODO: Do something with received data.
		}

		// Remove socket.
		for i, s := range server.sockets {
			if s == ws {
				server.sockets = append(server.sockets[:i], server.sockets[i+1:]...)
				break
			}
		}
	}))

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		return err
	}

	return nil
}
