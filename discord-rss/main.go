package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

// Used to keep track of the latest RSS Update
// THOUGHT: I wonder if we change this to a map to keep track of Items[0].{Title, Description, Link, etc}???
var linkArray []string //might be able to remove in a second
var titleArray []string // might be able to remove in a second
var messageArray []string
var botMessages []string

var newestTitleArray [1]string

var oldTitleArray [1]string

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

// Runs whatever function we feed this every X seconds, minutes, hours, etc.
func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}



func sendMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// build in logic for if len(messageArray) > len(oldMessageArray) to send backlogged RSS messages!!!


	if m.Author.ID == s.State.User.ID {
		fmt.Println("AWS RSS Bot Message ID:", m.Message.ID)
		fmt.Println("AWS RSS Bot Message Contents:", m.Message.Content)
		botMessage := m.Message.Content
		botMessages = append(botMessages, botMessage)
		return
	}

	if oldTitleArray[0] == newestTitleArray[0] {
		fmt.Println("No updates to RSS feed since last Discord Message")
	} else if len(botMessages) > 0 {
		for i := 0; i <= len(botMessages); i++ {
			if botMessages[i] == message {
				fmt.Println("I've already posted this, starting another parse in 5 minutes...")
				return
			} else {
				s.ChannelMessageSend("830896361112076349", message)
				fmt.Println("Sent message, parsing again in 5 minutes...")
				return
			}
		}
	} else {
		s.ChannelMessageSend("830896361112076349", message)
		fmt.Println("Sent message, parsing again in 5 minutes...")
		return
	}
}

// Parses the AWS RSS Feed via URL
// THOUGHT: Could probably use a for loop to parse through additional URLs and send to their respective discord channels as desired
func parseAWS(t time.Time) {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("https://aws.amazon.com/about-aws/whats-new/recent/feed/")

	oldTitleArray[0] = newestTitleArray[0]
	if feed.Items[0].Title == newestTitleArray[0] {
		fmt.Println("No updates, sleeping for 5 minutes...")
		doEvery(30*time.Second, parseAWS)
	} else {
		newestTitleArray[0] = feed.Items[0].Title
		// TODO: Figure out a way to trigger a send message on an update in this function.
		// Currently, the sendMessage Function only triggers when someone comments something
		// in any of the discord channels in Helping Helpdesk

		// Prints out the latest 100 Items in the AWS RSS Feed
		for i := 0; i < 100; i++ {
			titleArray = append(titleArray, feed.Items[i].Title)
			linkArray = append(linkArray, feed.Items[i].Link)
			message = fmt.Sprintf("%s\n%s", feed.Items[i].Title, feed.Items[i].Link)
			messageArray = append(messageArray, message)
		}
		message = messageArray[0]
		fmt.Println("messageArray updated, ready to send message in Discord. Sleeping for 5 minutes...")
		doEvery(30*time.Second, parseAWS)
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
	fmt.Println("Starting AWS RSS Parsing in 10 seconds...")
	doEvery(10*time.Second, parseAWS)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}
