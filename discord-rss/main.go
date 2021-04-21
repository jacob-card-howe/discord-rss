package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"reflect"
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
	Token string
)

var message string

// Initializes the Discord Part of the App for DiscordGo module
func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

// Sends messageArray anytime a new message is sent to your Discord Server
func sendMessage(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		fmt.Println("This bot posted the last message. Not parsing again.")
		return
	} else {
		// Parses anytime a new message is detected in your Discord Server
		feedParser := gofeed.NewParser()
		fmt.Println("Parsing RSS Feed...")
		//feed, err := feedParser.ParseURL("http://lorem-rss.herokuapp.com/feed?length=10&unit=second&interval=30")
		feed, err := feedParser.ParseURL("https://aws.amazon.com/about-aws/whats-new/recent/feed/")
		if err != nil {
			fmt.Println("There was an error parsing the URL:", err)
		}

		// Grabs last 10 RSS Items and appends them to messageArray
		for i := 0; i <= 9; i++ {
			message = fmt.Sprintf("%s\n%s", feed.Items[i].Title, feed.Items[i].Link)
			messageArray = append(messageArray, message)
		}

		// Checks if messageArray and botMessageArray are both equal
		if reflect.DeepEqual(messageArray, botMessageArray) {
			fmt.Println("I've already posted these messages in this messageArray.")
		} else {
			// Sends entire messageArray
			for i := 0; i < len(messageArray); i++ {
				s.ChannelMessageSend("830896361112076349", messageArray[i])
				botMessageArray = append(botMessageArray, messageArray[i])
			}
		}
		// Clears the message array
		fmt.Println("Clearing messageArray...")
		messageArray = nil
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
