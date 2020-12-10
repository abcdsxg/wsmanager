package wsmanager

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/websocket"
)

type NewMessageHandler func(session *Session, mt int, msg []byte)

type ErrorMessageHandler func(session *Session, err error)

type OnSessionClosed func(session *Session)

type Session struct {
	ID                  string
	Conn                *websocket.Conn
	newMessageHandler   NewMessageHandler
	errorMessageHandler ErrorMessageHandler
	onSessionClosed     OnSessionClosed
	quit                chan struct{}
}

func NewSession(sessionID interface{}, conn *websocket.Conn) *Session {
	session := &Session{
		ID:   fmt.Sprintf("%v", sessionID),
		Conn: conn,
	}

	newSessionChan <- session
	go session.ReadMessage()
	return session
}

func (s *Session) Disconnect() {
	s.quit <- struct{}{}
}

func (s *Session) SetNewMessageHandler(handler NewMessageHandler) {
	s.newMessageHandler = handler
}

func (s *Session) SetErrorMessageHandler(handler ErrorMessageHandler) {
	s.errorMessageHandler = handler
}

func (s *Session) ReadMessage() {
	defer func() {
		s.Close()
	}()

	for {
		select {
		case <-s.quit:
			return
		default:
			mt, message, err := s.Conn.ReadMessage()
			switch {
			case err != nil:
				log.Error().Err(err).Str("sessionID", s.ID).Msg("readMessage error")
				s.errorMessageHandler(s, err)
				return
			default:
				if s.newMessageHandler == nil {
					log.Info().Err(err).Str("sessionID", s.ID).Msgf("receive msg: %s", string(message))
				} else {
					s.newMessageHandler(s, mt, message)
				}
			}
		}

	}
}

func (s *Session) WriteMessage(mt int, msg []byte) error {
	if s.Conn == nil {
		return errors.New("client already closed")
	}

	log.Info().Str("sessionID", s.ID).Interface("writeMessage", string(msg)).Send()
	return s.Conn.WriteMessage(mt, msg)
}

func (s *Session) Close() {
	if s.Conn != nil {
		s.Conn.Close()
	}
	sessionClosedChan <- s.ID
	log.Info().Str("sessionID", s.ID).Msg("closed")

	if s.onSessionClosed != nil {
		s.onSessionClosed(s)
	}
}
