package rage

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func keys(m map[string]bool) []string {
	ks := []string{}
	for k, _ := range m {
		ks = append(ks, k)
	}
	return ks
}

type Rage struct {
	threshold     int
	period        int
	channel       string
	logger        *logrus.Logger
	slackClient   *slack.Client
	notifyHistory map[string]time.Time
}

func New(threshold, period int, channel string, logger *logrus.Logger, token string) *Rage {
	notifyHistory := map[string]time.Time{}
	slackClient := slack.New(token)
	return &Rage{
		threshold,
		period,
		channel,
		logger,
		slackClient,
		notifyHistory,
	}
}

func (r *Rage) Detect(messageChannelID string, messageTimestamp string) error {
	// Get recently messages.
	params := &slack.GetConversationHistoryParameters{
		ChannelID: messageChannelID,
		Limit:     r.threshold,
	}
	history, err := r.slackClient.GetConversationHistory(params)
	if err != nil {
		r.logger.Errorf("Can not get history: %+v", err)
		return err
	}

	// Pick up oldest message in the coversation.
	oldest := history.Messages[len(history.Messages)-1]
	r.logger.Debugf("Oldest: %+v", oldest)

	// Get the oldest timestamp.
	startUnix, err := strconv.ParseFloat(oldest.Timestamp, 64)
	if err != nil {
		r.logger.Errorf("Failed to parse timestamp: %s", err)
		return err
	}

	// Get most recently timestamp.
	endUnix, err := strconv.ParseFloat(messageTimestamp, 64)
	if err != nil {
		r.logger.Errorf("Failed to parse timestamp: %s", err)
		return err
	}

	// Compaire two timestamps.
	diff := time.Unix(int64(endUnix), 0).Sub(time.Unix(int64(startUnix), 0))
	r.logger.Infof("Diff: %v", diff)

	if (time.Duration(r.period) * time.Second) < diff {
		return nil
	}

	// Get speakers.
	speakers := map[string]bool{}
	for _, mes := range history.Messages {
		speakers[mes.User] = true
	}

	r.logger.Debugf("speackers: %+v", speakers)
	// Remove bots in speakers.
	for _, userID := range keys(speakers) {
		isBot, err := r.UserIsBot(userID)
		if err != nil {
			r.logger.Warnf("Can not get user info: %+v", err)
			delete(speakers, userID)
			continue
		}
		if isBot {
			delete(speakers, userID)
		}
	}

	r.logger.Infof("%d speakers in the conversation", len(speakers))
	if len(speakers) < 2 {
		return nil
	}

	now := time.Now()
	if timestamp, ok := r.notifyHistory[messageChannelID]; ok && now.Sub(timestamp) < 10*time.Minute {
		r.logger.Info("Skip notification because of cool time")
		return nil
	}

	r.logger.Info("Notify")
	err = r.Post(messageChannelID)
	if err != nil {
		r.logger.Errorf("Failed to post: %s", err)
		return err
	}
	r.notifyHistory[messageChannelID] = now

	return nil
}

func (r *Rage) UserIsBot(userID string) (bool, error) {
	user, err := r.slackClient.GetUserInfo(userID)
	if err != nil {
		return false, err
	}
	r.logger.Debugf("Author is: %+v", user)
	return user.IsBot, nil
}

func (r *Rage) Post(messageChannelID string) error {
	params := &slack.GetConversationsParameters{
		Limit: 999,
	}
	channelList, _, err := r.slackClient.GetConversations(params)
	if err != nil {
		return err
	}
	var notifyChannel *slack.Channel
	for _, c := range channelList {
		if c.Name == r.channel {
			notifyChannel = &c
			break
		}
	}
	if notifyChannel == nil {
		return fmt.Errorf("notify channel %s does not exist", r.channel)
	}
	msgOptText := slack.MsgOptionText("<#"+messageChannelID+"> が盛り上がってるっぽいよ！", false)
	_, _, err = r.slackClient.PostMessage(notifyChannel.ID, msgOptText)

	return err
}
