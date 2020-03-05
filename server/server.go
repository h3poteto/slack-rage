package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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
	fmt.Println(verbose)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if isBot {
			s.logger.Info("User is bot")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Get recently messages.
		channel := message.String("channel")
		params := &slack.GetConversationHistoryParameters{
			ChannelID: channel,
			Limit:     s.threshold,
		}
		history, err := api.GetConversationHistory(params)
		if err != nil {
			s.logger.Errorf("Can not get history: %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Pick up oldest message in the coversation.
		oldest := history.Messages[len(history.Messages)-1]
		s.logger.Debugf("Oldest: %+v", oldest)

		// Get the oldest timestamp.
		startUnix, err := strconv.ParseFloat(oldest.Timestamp, 64)
		if err != nil {
			s.logger.Errorf("Failed to parse timestamp: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Get most recently timestamp.
		endUnix, err := strconv.ParseFloat(message.String("ts"), 64)
		if err != nil {
			s.logger.Errorf("Failed to parse timestamp: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Compaire two timestamps.
		diff := time.Unix(int64(endUnix), 0).Sub(time.Unix(int64(startUnix), 0))
		s.logger.Infof("Diff: %v", diff)

		if (time.Duration(s.period) * time.Second) < diff {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Get speakers.
		speakers := map[string]bool{}
		for _, mes := range history.Messages {
			speakers[mes.User] = true
		}

		s.logger.Debugf("speackers: %+v", speakers)
		// Remove bots in speakers.
		for _, userID := range keys(speakers) {
			isBot, err := s.userIsBot(api, userID)
			if err != nil {
				s.logger.Errorf("Can not get user info: %+v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if isBot {
				delete(speakers, userID)
			}
		}

		s.logger.Infof("%d speakers in the conversation", len(speakers))
		if len(speakers) < 2 {
			w.WriteHeader(http.StatusOK)
			return
		}

		now := time.Now()
		if timestamp, ok := notifyHistory[channel]; ok && now.Sub(timestamp) < 10*time.Minute {
			s.logger.Info("Skip notification because of cool time")
			w.WriteHeader(http.StatusOK)
			return
		}

		s.logger.Info("Notify")
		err = s.Post(channel)
		if err != nil {
			s.logger.Errorf("Failed to post: %s", err)
		}
		notifyHistory[channel] = now

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
