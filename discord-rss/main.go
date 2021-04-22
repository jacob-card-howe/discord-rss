package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

// Used to keep track of the latest RSS Update
var messageArray []string

// Used to keep track of recent Bot Messages
var botMessageArray []string

// Used to accept CLI Parameters
var (
	Token     string
	Url       string
	ChannelId string
)

var message string

// Initializes the Discord Part of the App for DiscordGo module
func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Url, "u", "", "RSS Feed URL")
	flag.StringVar(&ChannelId, "c", "", "Channel ID you want messages to post in")
	flag.Parse()
}

// Sends messageArray anytime a new message is sent to your Discord Server
func sendMessage(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		log.Println("This bot posted the last message. Not parsing again.")
		return
	} else {
		// Parses anytime a new message is detected in your Discord Server
		feedParser := gofeed.NewParser()
		log.Println("Parsing RSS Feed...")
		//feed, err := feedParser.ParseURL("http://lorem-rss.herokuapp.com/feed?length=10&unit=second&interval=60") // Great RSS feed for testing :)
		feed, err := feedParser.ParseURL(Url)
		if err != nil {
			fmt.Println("There was an error parsing the URL:", err)
			return
		}

		// Grabs last 5 RSS Items and appends them to messageMap (Discord Character Limits)
		log.Println("Generating messageArray...")
		for i := 0; i <= 4; i++ {
			message = fmt.Sprintf("%s!\n%s\n", feed.Items[i].Title, feed.Items[i].Link)
			messageArray = append(messageArray, message)
		}

		// Formats messageArray into one big message instead of sending 5 individual messages
		// It's noisy if we don't do it this way, and also exceeds Discord's character limit / message rate limit
		log.Println("Generating bigMessage...")
		convertToStrings := fmt.Sprintf(strings.Join(messageArray, "\n"))
		bigMessage := fmt.Sprintf("Here are the 5 latest RSS Feed Items:\n\n%v", convertToStrings)

		// Checks to see if there's a difference between messageArray & botMessageArray
		if botMessageArray != nil {
			for _, value := range botMessageArray {
				if value == bigMessage {
					log.Println("I've posted this message recently, skipping new post.")

					// Clears the message array
					log.Println("Clearing messageArray...")
					messageArray = nil
					return
				} else {
					log.Println("Sending message to Discord...")
					s.ChannelMessageSend(ChannelId, bigMessage)
					botMessageArray = append(botMessageArray, bigMessage)

					// Clears the message array
					log.Println("Clearing messageArray...")
					messageArray = nil

					// Clears the botMessageArray if len(botMessageArray) > 1000
					if len(botMessageArray) > 1000 {
						botMessageArray = nil
						return
					}
				}
			}
		} else {
			log.Println("Sending message to Discord...")
			s.ChannelMessageSend(ChannelId, bigMessage)
			botMessageArray = append(botMessageArray, bigMessage)

			// Clears the message array
			log.Println("Clearing messageArray...")
			messageArray = nil

			// Clears the botMessageArray if len(botMessageArray) > 1000
			if len(botMessageArray) > 1000 {
				botMessageArray = nil
				return
			}
		}
	}
}

func main() {
	// Creating Discord Session Using Provided Bot Token
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord Session:", err)
		return
	}

	// Registers the sendMessage function as a callback for MessageCreate Events
	dg.AddHandler(sendMessage)

	// Sets the intentions of the bot, read through the docs
	// This specifically says "I want this bot to deal with messages in channels (Guilds)"
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	fmt.Println("Your RSS feed will be parsed any time there's a new message in your Discord Server.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}
