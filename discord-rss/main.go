package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	//"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

// Used to keep track of the latest RSS Update
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

func sendMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	feedParser := gofeed.NewParser()
	fmt.Println("Parsing RSS Feed...")
	feed, err := feedParser.ParseURL("http://lorem-rss.herokuapp.com/feed?length=100&unit=second&interval=30")
	if err != nil {
		fmt.Println("There was an error parsing the URL:", err)
	}

	newestTitleArray[0] = feed.Items[0].Title

	// Should update the messageArray for the latest 100 entries in the RSS Feed
	for i := 0; i <= 99; i++ {
		message = fmt.Sprintf("%s\n%s", feed.Items[i].Title, feed.Items[i].Link)
		if len(messageArray) > 0 {
			messageArray = append(messageArray, "Checking for Space") // gets overwritten, only exists to test if the array has space
			copy(messageArray[1:], messageArray[0:]) // shifts existing messageArray[0] value to position 1 to make room for the new message
			messageArray[0] = message
		} else {
			messageArray = append(messageArray, message)
		}
	}

	fmt.Println("There's currently this many items in the messageArray:", len(messageArray))

	if m.Author.ID == s.State.User.ID {
		fmt.Println("AWS RSS Bot Message ID:", m.Message.ID)
		fmt.Println("AWS RSS Bot Message Contents:", m.Message.Content)
		botMessage := m.Message.Content
		botMessages = append(botMessages, "Testing for Space") // gets overwritten, only exists to test if the array has space
		copy(botMessages[1:], botMessages[0:]) // shifts existing botMessage[0] value to position 1 to make room for the new message
		botMessages[0] = botMessage
		fmt.Println(botMessages) // Prints out all the messages the bot has sent
		return
	}

	if len(botMessages) > 0 {
		if botMessages[0] == messageArray[0] {
			fmt.Println("I've already posted this:", messageArray[0])
			messageArray = nil // Clears messageArray for next parse
			return
		} else {
			for i:= range messageArray {
				fmt.Println("This is the for loop working on line 61:", messageArray[i]) // Only outputs first value?????? WHYYYYYYYY
				//s.ChannelMessageSend("830896361112076349", messageArray[i])
				//fmt.Println("Sent message from line 61")
				//fmt.Println("Sent message, parsing again in 5 minutes...")
				return
			}
			messageArray = nil // Clears messageArray for next parse
		}
	} else {
		s.ChannelMessageSend("830896361112076349", messageArray[0])
		fmt.Println("Send message from line 68")
		//fmt.Println("Sent message, parsing again in 5 minutes...")
		return
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
	//fmt.Println("Starting AWS RSS Parsing in 10 seconds...")
	//doEvery(10*time.Second, parseAWS)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}
