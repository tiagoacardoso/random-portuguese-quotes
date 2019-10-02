package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

func main() {
	slackToken := os.Getenv("SLACK_TOKEN")
	slackSigningKey := os.Getenv("SLACK_SIGNING_KEY")
	api := slack.New(slackToken)

	viper.SetConfigFile("messages.json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	log.Println("Using messages config:", viper.ConfigFileUsed())

	var (
		signingSecret string
	)

	flag.StringVar(&signingSecret, "secret", slackSigningKey, "Your Slack app's signing secret")
	flag.Parse()

	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {

		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			log.Println(err)
			return
		}

		if err = verifier.Ensure(); err != nil {
			log.Println(err)
			return
		}

		switch s.Command {
		case "/randomportuguesequote":

			messages := viper.GetStringSlice("messages")

			rand.Seed(time.Now().UnixNano())
			randIndex := rand.Intn(len(messages))

			_, _, err := api.PostMessage(
				s.ChannelID,
				slack.MsgOptionText(messages[randIndex], false),
				slack.MsgOptionAttachments(slack.Attachment{}),
			)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println("Message successfully sent to channel " + s.ChannelID)
		default:
			log.Println("Command not found...")
			return
		}
	})

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":3000", nil)
}
