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

type Server struct {
	threshold int
	channel   string
	token     string
}

func NewServer(threshold int, channel string) *Server {
	token := os.Getenv("SLACK_TOKEN")
	return &Server{
		threshold,
		channel,
		token,
	}
}

func (s *Server) Serve() error {
	http.HandleFunc("/", s.ServeHTTP)
	logrus.Info("Listening on :9090")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		return fmt.Errorf("Failed to start server: %s", err)
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	event, err := DecodeJSON(r.Body)
	if err != nil {
		logrus.Errorf("Request body is not Event payload: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch event.Type() {
	case "url_verification":
		logrus.Info("Receive URL Verifciation event")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(event.String("challenge")))
		return
	case "event_callback":
		mes, ok := event["event"].(map[string]interface{})
		if !ok {
			logrus.Errorf("Failed to cast event: %+v", event)
			http.Error(w, "failed to cast", http.StatusBadRequest)
			return
		}
		logrus.Infof("Received event: %+v", mes)

		message := Event(mes)
		if message.Type() != "message" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Through posts from bots.
		username := message.String("user")
		api := slack.New(s.token)
		user, err := api.GetUserInfo(username)
		if err != nil {
			logrus.Errorf("Can not get user info: %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logrus.Infof("Author is: %+v", user)
		if user.IsBot {
			logrus.Info("User is bot")
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
			logrus.Errorf("Can not get history: %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		oldest := history.Messages[len(history.Messages)-1]
		logrus.Infof("Oldest: %+v", oldest)
		startUnix, err := strconv.ParseFloat(oldest.Timestamp, 64)
		if err != nil {
			logrus.Errorf("Failed to parse timestamp: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		endUnix, err := strconv.ParseFloat(message.String("ts"), 64)
		if err != nil {
			logrus.Errorf("Failed to parse timestamp: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		diff := time.Unix(int64(endUnix), 0).Sub(time.Unix(int64(startUnix), 0))
		logrus.Infof("Diff: %v", diff)

		if (10 * time.Minute) < diff {
			w.WriteHeader(http.StatusOK)
			return
		}

		logrus.Info("Post")

		err = s.Post(channel)
		if err != nil {
			logrus.Errorf("Failed to post: %s", err)
		}

		w.WriteHeader(http.StatusOK)
		return
	default:
		logrus.Warnf("Receive unknown event type: %s", event.Type())
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
	var notifyChannel slack.Channel
	for _, c := range channelList {
		if c.Name == s.channel {
			notifyChannel = c
		}
	}
	msgOptText := slack.MsgOptionText("<#"+channelID+"> が盛り上がってるっぽいよ！", false)
	_, _, err = api.PostMessage(notifyChannel.ID, msgOptText)

	return err
}
