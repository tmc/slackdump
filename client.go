package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/slack-go/slack"
)

type Options struct {
	Token    string
	DCookie  string
	DSCookie string
}

func newClient(token, dCookie, dsCookie string) (*slackClient, error) {
	if dCookie == "" {
		return nil, fmt.Errorf("d cookie flag is required")
	}
	jar, _ := cookiejar.New(nil)
	url, _ := url.Parse("https://slack.com")
	jar.SetCookies(url, []*http.Cookie{
		{
			Name:   "d",
			Value:  dCookie,
			Path:   "/",
			Domain: "slack.com",
		},
	})

	if dsCookie != "" {
		jar.SetCookies(url, []*http.Cookie{
			{
				Name:   "d-s",
				Value:  dsCookie,
				Path:   "/",
				Domain: "slack.com",
			},
		})
	}
	client := &http.Client{
		Jar: jar,
	}
	sc := slack.New(token,
		slack.OptionHTTPClient(client),
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)))

	rtm := sc.NewRTM()
	go rtm.ManageConnection()

	go func() {
		for msg := range rtm.IncomingEvents {
			fmt.Print("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				fmt.Println("Infos:", ev.Info)
				fmt.Println("Connection counter:", ev.ConnectionCount)
				// Replace C2147483705 with your Channel ID
				rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "C2147483705"))

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev)

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				fmt.Printf("Current latency: %v\n", ev.Value)

			case *slack.DesktopNotificationEvent:
				fmt.Printf("Desktop Notification: %v\n", ev)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				return

			default:
				// Ignore other events..
				fmt.Printf("Unexpected: %v\n", msg.Data)
				fmt.Println("sending message: typing")
				rtm.SendMessage(rtm.NewTypingMessage("D02EF7FSVB6"))
				rtm.SendMessage(rtm.NewTypingMessage("C03K5EJRYLU"))
			}
		}
	}()
	time.Sleep(time.Second * 60)
	return &slackClient{sc}, nil
}

type slackClient struct {
	*slack.Client
}

func (s *slackClient) listConversations(ctx context.Context, types ...string) []string {
	var (
		result   []string
		channels []slack.Channel
		err      error
	)
	if len(types) == 0 {
		types = []string{"public_channel", "private_channel", "mpim", "im"}
	}
	params := &slack.GetConversationsParameters{
		Types: types,
	}
	for err == nil {
		for _, channel := range channels {
			result = append(result, channel.Name)
			j, _ := json.Marshal(channel)
			fmt.Println(string(j))
		}
		channels, params.Cursor, err = s.Client.GetConversations(params)
	}
	return result
}

func (s *slackClient) dumpConversation(ctx context.Context, conversationID string) []string {
	var (
		result   []string
		channels []slack.Channel
	)
	params := &slack.GetConversationHistoryParameters{
		ChannelID:          conversationID,
		IncludeAllMetadata: true,
	}
	hasMore := true
	for hasMore {
		for _, channel := range channels {
			result = append(result, channel.Name)
			j, _ := json.Marshal(channel)
			fmt.Println(string(j))
		}
		hist, err := s.Client.GetConversationHistoryContext(ctx, params)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return result
		}
		params.Cursor = hist.ResponseMetaData.NextCursor
	}
	return result
}

func (s *slackClient) listUsers(ctx context.Context) ([]slack.User, error) {
	return s.Client.GetUsersContext(ctx)
}

func (s *slackClient) sendMessage(ctx context.Context, channelID, message string) error {
	a, b, err := s.Client.PostMessageContext(ctx, channelID, slack.MsgOptionText(message, false))
	fmt.Println(a, b, err)
	return err
}
