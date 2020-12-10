package wsmanager

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	sessionClosedChan = make(chan string, 10)
	newSessionChan    = make(chan *Session, 10)

	outWriter = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
)

type ErrorSendHandle func(session *Session, mt int, msg []byte, err error)

func init() {
	log.Logger = log.With().Caller().Timestamp().Logger().Output(outWriter)
}

func EnableLog(enable bool) {
	if enable {
		log.Logger = log.With().Caller().Timestamp().Logger().Output(outWriter)
	} else {
		log.Logger = log.Output(ioutil.Discard)
	}
}

type SessionManager struct {
	mu              sync.RWMutex
	sessions        map[string]*Session
	ErrorSendHandle ErrorSendHandle
}

func NewSessionManager() *SessionManager {
	manager := &SessionManager{
		sessions: make(map[string]*Session),
	}

	go manager.watchSessions()
	return manager
}

func (s *SessionManager) Count() int {
	return len(s.sessions)
}

func (s *SessionManager) AddSession(session *Session) {
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()
}

func (s *SessionManager) GetSession(sessionID interface{}) (session *Session, exist bool) {
	s.mu.RLock()
	session, exist = s.sessions[fmt.Sprintf("%s", sessionID)]
	s.mu.RUnlock()
	return
}

func (s *SessionManager) ExistSession(sessionID interface{}) (exist bool) {
	_, exist = s.GetSession(sessionID)
	return
}

func (s *SessionManager) RemoveSession(sessionID interface{}) {
	s.mu.Lock()
	delete(s.sessions, fmt.Sprintf("%s", sessionID))
	s.mu.Unlock()
}

func (s *SessionManager) Broadcast(mt int, msg []byte) {
	var copySessions []*Session
	s.mu.RLock()
	for _, session := range s.sessions {
		copySessions = append(copySessions, session)
	}
	s.mu.RUnlock()

	for _, session := range copySessions {
		if err := session.Conn.WriteMessage(mt, msg); err != nil && s.ErrorSendHandle != nil {
			s.ErrorSendHandle(session, mt, msg, err)
		}
	}
}

func (s *SessionManager) watchSessions() {
	for {
		select {
		case sessionID := <-sessionClosedChan:
			s.RemoveSession(sessionID)
			log.Info().Str("remove sessionID", sessionID).Int("current sessions count", len(s.sessions)).Send()
		case session := <-newSessionChan:
			s.AddSession(session)
			log.Info().Str("new sessionID", session.ID).Int("current sessions count", len(s.sessions)).Send()
		}
	}
}
