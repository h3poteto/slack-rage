package rtm

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type RTM struct {
	token  string
	logger *logrus.Logger
}

func New() *RTM {
	token := os.Getenv("SLACK_TOKEN")
	logger := logrus.New()
	return &RTM{
		token,
		logger,
	}
}

func (r *RTM) Start() {
	api := slack.New(r.token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
		case *slack.ConnectedEvent:
			r.logger.Infof("Info: %+v", ev.Info)
			r.logger.Infof("Connection counter: %d", ev.ConnectionCount)
		case *slack.PresenceChangeEvent:
			r.logger.Debugf("Presence Change: %v", ev)
		case *slack.LatencyReport:
			r.logger.Debugf("Current latency: %v", ev.Value)
		case *slack.DesktopNotificationEvent:
			r.logger.Debugf("Desktop Notification: %v", ev)
		case *slack.MessageEvent:
			r.logger.Infof("Receive message: %+v", ev)

			// Through posts from bots.
			userID := ev.Msg.User
			user, err := api.GetUserInfo(userID)
			if err != nil {
				r.logger.Errorf("Can not get user info: %s", err)
				continue
			}
			r.logger.Debugf("Author is: %+v", user)
			if user.IsBot {
				r.logger.Info("User is bot")
				continue
			}
			// Detect rage
		case *slack.RTMError:
			r.logger.Errorf("Error: %s", ev.Error())
		case *slack.InvalidAuthEvent:
			r.logger.Error("Invalid credentials")
			return
		default:
			r.logger.Warnf("Unexpected: %+v", msg.Data)
		}
	}

}
