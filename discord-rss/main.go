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
var titleArray [1]string

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

	if m.Author.ID == s.State.User.ID {
		return
	}

	if oldTitleArray[0] == titleArray[0] {
		fmt.Println("Nothing to see here")
	} else {
		s.ChannelMessageSend("830896361112076349", message)
		fmt.Println("Message sent!")
		doEvery(30*time.Second, parseAWS)
	}
}

// Parses the AWS RSS Feed via URL
// THOUGHT: Could probably use a for loop to parse through additional URLs and send to their respective discord channels as desired
func parseAWS(t time.Time) {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("https://aws.amazon.com/about-aws/whats-new/recent/feed/")

	oldTitleArray[0] = titleArray[0]
	if feed.Items[0].Title == titleArray[0] {
		fmt.Println("No updates, sleeping for 5 minutes...")
		doEvery(300*time.Second, parseAWS)
	} else {
		fmt.Printf("NEW!: %v\n", feed.Items[0].Title)
		fmt.Println(feed.Items[0].Link)
		titleArray[0] = feed.Items[0].Title
		message = fmt.Sprintf("%s\n%s", feed.Items[0].Title, feed.Items[0].Link)
		// TODO: Figure out a way to trigger a send message on an update in this function.
		// Currently, the sendMessage Function only triggers when someone comments something
		// in any of the discord channels in Helping Helpdesk
		fmt.Println("oldArray updated, sleeping for 5 minutes...")
		doEvery(300*time.Second, parseAWS)
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
