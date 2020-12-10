package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/abcdsxg/wsmanager"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
	}

	sessionManager = wsmanager.NewSessionManager()
)

func main() {
	wsmanager.EnableLog(true)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := w.Header().Get("Authorization")
		userID, err := Authorize(token)
		if err != nil {
			w.Write([]byte("Authentication Failure"))
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		session := wsmanager.NewSession(userID, conn)
		session.SetErrorMessageHandler(func(session *wsmanager.Session, err error) {
			log.Println("sessionID:", session.ID, " error:", err)
		})
		session.SetNewMessageHandler(func(session *wsmanager.Session, mt int, msg []byte) {
			log.Println("sessionID:", session.ID, " receive msg:", string(msg))
		})

		go func() {
			//send to all sessions
			time.Sleep(5 * time.Second)
			sessionManager.Broadcast(websocket.TextMessage, []byte("broadcast test"))

			//send to a session
			specialSession, ok := sessionManager.GetSession(userID)
			if ok {
				specialSession.WriteMessage(websocket.TextMessage, []byte("send to special session"))
			}
			return
		}()

	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// Authorize get userId from token
func Authorize(token string) (userID int, err error) {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(99999), nil
}
