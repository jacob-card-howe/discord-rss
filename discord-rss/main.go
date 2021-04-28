package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

// Used to keep track of the latest RSS Update
var messageArray []string
var previousMessage []string

// Used to keep track of recent Bot Messages
var botMessageArray []string

// Used to accept CLI Parameters
var (
	Token     string
	Url       string
	ChannelId string
	// Can add this in later
	// TickerTimer int
)

var message string

// Initializes the Discord Part of the App for DiscordGo module
func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Url, "u", "", "RSS Feed URL")
	flag.StringVar(&ChannelId, "c", "", "Channel ID you want messages to post in")
	// Can add this in later
	// flag.IntVar(&TickerTimer, "timer", "", "Sets how long the auto parser will run for")
	flag.Parse()
}

func GetCreationDate(ID string) (t time.Time, timeInRFC3339 string, err error) {
	i, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return
	}
	timestamp := (i >> 22) + 1420070400000 // converts Snowflake ID to a unix timestamp
	t = time.Unix(0, timestamp*1000000)
	timeInRFC3339 = t.Format(time.RFC3339)
	return
}

func messageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Rip out this logic since I already have built in "if there's a new message, don't start another loop" logic?

	/*if m.Author.ID == s.State.User.ID {
		log.Println("This bot posted the last message.")
		return
	}*/

	if m.Content == "!status" {
		_, err := s.ChannelMessageSend(ChannelId, "I'm running!\n\nIf you're not getting updates from your RSS feed, it's likely there hasn't been an update recently.")
		if err != nil {
			log.Println("Error sending message:", err)
		}
	} else if m.Content == "!hours" {
		_, err := s.ChannelMessageSend(ChannelId, "This bot runs from 9AM ET to 6PM ET. @Howe should write in logic to output my uptime on this command too!")
		if err != nil {
			log.Println("Error sending message:", err)
		}
	} else if len(previousMessage) > 0 {
		// Compare timestamps of previous message and newest message
		// ignore if latest message is too recent

		_, timestamp, err := GetCreationDate(m.ID) // timestamp of message sent in Discord
		if err != nil {
			log.Println("Not a valid Snowflake ID:", err)
		}

		parseTimestamp, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			log.Println("Error Parsing new message's timestamp:", err)
		}

		parseOldMessage, err := time.Parse(time.RFC3339, previousMessage[0])
		if err != nil {
			log.Println("Error Parsing old message's timestamp:", err)
		}

		diff := parseTimestamp.Sub(parseOldMessage)
		log.Printf("These messages were posted %s apart", diff)

		if diff.Hours() < 9 { // Still should build in logic to allow for a dynamic integer here
			log.Println("The loop is still running! Skipping starting another loop.")
		} else {
			log.Println("Starting parser loop...")
			ticker := time.NewTicker(30 * time.Second)
			done := make(chan bool)

			// Prevents the loop below from running overtop itself
			previousMessage = append(previousMessage, timestamp)
			copy(previousMessage[1:], previousMessage)
			previousMessage[0] = timestamp

			go func() {
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						sendUpdate(s)
					}
				}
			}()

			time.Sleep(9 * time.Hour)
			ticker.Stop()
			done <- true
			fmt.Println("Stopped ticker.")
		}
	} else {
		_, timestamp, err := GetCreationDate(m.ID)
		if err != nil {
			log.Println("Error getting Message Timestamp:", err)
		} else {
			// Send update to Channel
			log.Println("Starting parser loop...")
			ticker := time.NewTicker(30 * time.Second)
			done := make(chan bool)

			// An attempt to prevent doubling up on the loop below
			previousMessage = append(previousMessage, timestamp)
			copy(previousMessage[1:], previousMessage)
			previousMessage[0] = timestamp

			go func() {
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						sendUpdate(s)
					}
				}
			}()

			time.Sleep(9 * time.Hour)
			ticker.Stop()
			done <- true
			fmt.Println("Stopped ticker.")
		}
	}
}

// Sends messageArray anytime a new message is sent to your Discord Server
func sendUpdate(s *discordgo.Session) {

	// Clears the message array on every new message
	log.Println("Clearing the messageArray...")
	messageArray = nil

	// Parses anytime a new message is detected in your Discord Server
	feedParser := gofeed.NewParser()
	log.Println("Parsing RSS Feed...")
	//feed, err := feedParser.ParseURL("http://lorem-rss.herokuapp.com/feed?unit=second&interval=15") // Great RSS feed for testing :)
	feed, err := feedParser.ParseURL(Url)
	if err != nil {
		log.Println("There was an error parsing the URL:", err)
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
	if botMessageArray != nil && messageArray != nil {
		if bigMessage != botMessageArray[0] {
			log.Println("Sending message to Discord...")
			_, err := s.ChannelMessageSend(ChannelId, bigMessage)
			if err != nil {
				log.Println("There was an error sending your message:", err)
			} else {
				log.Println("Your message:\n", bigMessage)
			}

			// Appends message to the front of botMessageArray
			botMessageArray = append(botMessageArray, bigMessage)
			copy(botMessageArray[1:], botMessageArray)
			botMessageArray[0] = bigMessage
		} else {
			log.Println("I've posted this message recently, skipping new post.")
		}
	}

	if botMessageArray == nil {
		log.Println("Sending message to Discord...")
		_, err := s.ChannelMessageSend(ChannelId, bigMessage)
		if err != nil {
			log.Println("There was an error sending your message:", err)
		} else {
			log.Println("Your message:\n", bigMessage)
		}

		// Appends message to the front of botMessageArray
		botMessageArray = append(botMessageArray, bigMessage)
		copy(botMessageArray[1:], botMessageArray)
		botMessageArray[0] = bigMessage
	}

	// Clears the botMessageArray if len(botMessageArray) > 1000
	if len(botMessageArray) > 1000 {
		botMessageArray = nil
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

	// Registers the messageCreated function as a callback for a MessageCreated Event
	dg.AddHandler(messageCreated)

	// Sets the intentions of the bot, read through the docs
	// This specifically says "I want this bot to deal with messages in channels (Guilds)"
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	} else {
		dg.ChannelMessageSend(ChannelId, "I'm running!")
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
