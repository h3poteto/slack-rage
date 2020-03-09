package rtm

import (
	"os"

	"github.com/h3poteto/slack-rage/rage"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

type RTM struct {
	threshold int
	period    int
	channel   string
	token     string
	logger    *logrus.Logger
	detector  *rage.Rage
}

func New(threshold, period int, channel string, verbose bool) *RTM {
	token := os.Getenv("SLACK_TOKEN")
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}
	// We have to create classic slack app using RTM.
	// Classic slack app require OAuth token to call REST API separately from bot token.
	detector := rage.New(threshold, period, channel, logger, os.Getenv("OAUTH_TOKEN"))
	return &RTM{
		threshold,
		period,
		channel,
		token,
		logger,
		detector,
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
			isBot, err := r.detector.UserIsBot(userID)
			if err != nil {
				r.logger.Errorf("Can not get user info: %s", err)
				continue
			}
			if isBot {
				r.logger.Info("User is bot")
				continue
			}
			// Detect rage
			r.detector.Detect(ev.Msg.Channel, ev.Msg.Timestamp)

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
