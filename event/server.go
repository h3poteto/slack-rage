package event

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/h3poteto/slack-rage/rage"
	"github.com/sirupsen/logrus"
)

var notifyHistory = map[string]time.Time{}

type Server struct {
	threshold int
	period    int
	channel   string
	token     string
	logger    *logrus.Logger
	detector  *rage.Rage
}

func NewServer(threshold, period int, channel string, verbose bool) *Server {
	token := os.Getenv("SLACK_TOKEN")
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	detector := rage.New(threshold, period, channel, logger, token)
	return &Server{
		threshold,
		period,
		channel,
		token,
		logger,
		detector,
	}
}

func (s *Server) Serve() error {
	http.HandleFunc("/health_check", s.HealthCheck)
	http.HandleFunc("/", s.HandleEvent)
	s.logger.Info("Listening on :9090")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		return fmt.Errorf("Failed to start server: %s", err)
	}

	return nil
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

func (s *Server) HandleEvent(w http.ResponseWriter, r *http.Request) {
	event, err := DecodeJSON(r.Body)
	if err != nil {
		s.logger.Errorf("Request body is not Event payload: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch event.Type() {
	case "url_verification":
		s.logger.Info("Receive URL Verifciation event")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(event.String("challenge")))
		return
	case "event_callback":
		mes, ok := event["event"].(map[string]interface{})
		if !ok {
			s.logger.Errorf("Failed to cast event: %+v", event)
			http.Error(w, "failed to cast", http.StatusBadRequest)
			return
		}
		s.logger.Infof("Received event: %+v", mes)

		message := Event(mes)
		if message.Type() != "message" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Through posts from bots.
		userID := message.String("user")
		isBot, err := s.detector.UserIsBot(userID)
		if err != nil {
			s.logger.Errorf("Can not get user info: %+v", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		if isBot {
			s.logger.Info("User is bot")
			w.WriteHeader(http.StatusOK)
			return
		}

		err = s.detector.Detect(message.String("channel"), message.String("ts"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	default:
		s.logger.Warnf("Receive unknown event type: %s", event.Type())
		w.WriteHeader(http.StatusOK)
		return
	}
}
