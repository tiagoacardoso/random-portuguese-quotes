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
	"reflect"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/spf13/viper"
)

var (
	api           *slack.Client
	signingSecret string
)

func sendMessage(channelID string, text string) {
	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(text, false),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Message successfully sent to channel " + channelID)
}

func snakeCaseToSentence(txt string) string {
	txt = strings.ToLower(txt)

	split := strings.Split(txt, "_")
	var slice []string
	for _, word := range split {
		c := strings.ToUpper(string(word[0]))
		slice = append(slice, c+word[1:])
	}

	return strings.Join(slice, " ")
}

func main() {
	slackToken := os.Getenv("SLACK_TOKEN")
	slackSigningKey := os.Getenv("SLACK_SIGNING_KEY")
	api = slack.New(slackToken)

	viper.SetConfigFile("messages.json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	log.Println("Using messages config:", viper.ConfigFileUsed())

	flag.StringVar(&signingSecret, "secret", slackSigningKey, "Your Slack app's signing secret")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("It works!"))
	})

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
			messages := viper.GetStringSlice("random")

			rand.Seed(time.Now().UnixNano())
			randIndex := rand.Intn(len(messages))

			sendMessage(s.ChannelID, messages[randIndex])
			break
		case "/randomauthorquote":
			author := strings.Split(s.Text, " ")[0]

			messages := viper.GetStringSlice("author." + author)

			if len(messages) == 0 {
				authors := viper.GetStringMap("author")
				keys := reflect.ValueOf(authors).MapKeys()
				sendMessage(s.UserID, fmt.Sprintf("Author %s not found, available authors: %v", author, keys))
				return
			}

			rand.Seed(time.Now().UnixNano())
			randIndex := rand.Intn(len(messages))

			sendMessage(s.ChannelID, fmt.Sprintf("\"%s\" - %s", messages[randIndex], snakeCaseToSentence(author)))
			break
		default:
			log.Println("Command not found...")
			return
		}
	})

	fmt.Println("[INFO] Server listening")

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	http.ListenAndServe(":"+port, nil)
}
