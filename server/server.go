package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/h3poteto/slack-rage/rage"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

var notifyHistory = map[string]time.Time{}

type Server struct {
	threshold int
	period    int
	channel   string
	token     string
	logger    *logrus.Logger
}

func NewServer(threshold, period int, channel string, verbose bool) *Server {
	token := os.Getenv("SLACK_TOKEN")
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	return &Server{
		threshold,
		period,
		channel,
		token,
		logger,
	}
}

func (s *Server) Serve() error {
	http.HandleFunc("/", s.ServeHTTP)
	s.logger.Info("Listening on :9090")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		return fmt.Errorf("Failed to start server: %s", err)
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	event, err := DecodeJSON(r.Body)
	if err != nil {
		s.logger.Errorf("Request body is not Event payload: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	api := slack.New(s.token)
	detector := rage.New(s.threshold, s.period, s.channel, s.logger, api)

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
		api := slack.New(s.token)
		isBot, err := s.userIsBot(api, userID)
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

		err = detector.Detect(message.String("channel"), message.String("ts"))
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

func (s *Server) Post(channelID string) error {
	api := slack.New(s.token)
	params := &slack.GetConversationsParameters{
		Limit: 999,
	}
	channelList, _, err := api.GetConversations(params)
	if err != nil {
		return err
	}
	var notifyChannel *slack.Channel
	for _, c := range channelList {
		if c.Name == s.channel {
			notifyChannel = &c
			break
		}
	}
	if notifyChannel == nil {
		return fmt.Errorf("notify channel %s does not exist", s.channel)
	}
	msgOptText := slack.MsgOptionText("<#"+channelID+"> が盛り上がってるっぽいよ！", false)
	_, _, err = api.PostMessage(notifyChannel.ID, msgOptText)

	return err
}

func keys(m map[string]bool) []string {
	ks := []string{}
	for k, _ := range m {
		ks = append(ks, k)
	}
	return ks
}

func (s *Server) userIsBot(api *slack.Client, userID string) (bool, error) {
	user, err := api.GetUserInfo(userID)
	if err != nil {
		return false, err
	}
	s.logger.Debugf("Author is: %+v", user)
	return user.IsBot, nil
}
