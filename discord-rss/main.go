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

// Used to keep track of recent posts about the RSS feed
var last100DiscordMessages []string

// Used to accept CLI Parameters
var (
	Token             string
	Url               string
	ChannelId         string
	TickerTimer       int
	BasicAuthUsername string
	BasicAuthPassword string
)

var message string

// Initializes the Discord Part of the App for DiscordGo module
func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Url, "u", "", "RSS Feed URL")
	flag.StringVar(&ChannelId, "c", "", "Channel ID you want messages to post in")
	flag.IntVar(&TickerTimer, "timer", 0, "Sets how long the auto parser will run for in hours")
	flag.StringVar(&BasicAuthUsername, "user", "", "Allows you to pass in the 'Username' part of your BasicAuthentication credentials")
	flag.StringVar(&BasicAuthPassword, "pass", "", "Allows you to pass in the 'Password' part of your BasicAuthentication credentials")
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

// Listens for new messages, starts a timer / loop to send messages to discord anytime there's a new RSS message
func messageCreated(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Content == "!status" {
		_, err := s.ChannelMessageSend(ChannelId, "I'm running!\n\nIf you're not getting updates from your RSS feed, it's likely there hasn't been an update recently.")
		if err != nil {
			log.Println("Error sending message:", err)
		}
	} else if m.Content == "!hours" {
		_, err := s.ChannelMessageSend(ChannelId, "This bot runs from 9AM ET to 5PM ET. @Howe should write in logic to output my uptime on this command too!")
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

		if diff.Hours() < float64(TickerTimer) {
			log.Println("The loop is still running! Skipping starting another loop.")
		} else {
			log.Println("Starting parser loop...")
			ticker := time.NewTicker(time.Duration(TickerTimer) * time.Second)
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

			time.Sleep(time.Duration(TickerTimer) * time.Hour)
			ticker.Stop()
			done <- true
			log.Println("Stopped ticker.")
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

			// Prevents any doubling of this loop
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

			time.Sleep(time.Duration(TickerTimer) * time.Hour)
			ticker.Stop()
			done <- true
			log.Println("Stopped ticker.")
		}
	}
}

// Sends Update to Discord Channel on any new RSS Item
func sendUpdate(s *discordgo.Session) {

	log.Println("Getting last 100 message structs...")
	last100MessageStructs, err := s.ChannelMessages(ChannelId, 100, "", "", "")
	if err != nil {
		log.Println("Error getting last 100 messages:", err)
	}

	if len(last100MessageStructs) > 0 {
		for i := 0; i < len(last100MessageStructs); i++ {
			if last100MessageStructs[i].Author.ID == s.State.User.ID && (last100MessageStructs[i].Content == "Shutting down..." || last100MessageStructs[i].Content == "I'm running!" || last100MessageStructs[i].Content == "Placeholder Text :)" || last100MessageStructs[i].Content == "Here's the messages you missed while I was offline:" || last100MessageStructs[i].Content == "Looks like you're up to date on your RSS feed!") {
				log.Println("This is a message we don't need.")
			} else if last100MessageStructs[i].Author.ID == s.State.User.ID {
				last100DiscordMessages = append(last100DiscordMessages, last100MessageStructs[i].Content)
			}
		}
	}

	// Clears the message array on every new message
	log.Println("Clearing the messageArray...")
	messageArray = nil

	// Parses the RSS URL on every loop
	feedParser := gofeed.NewParser()
	log.Println("Parsing RSS Feed...")
	//feed, err := feedParser.ParseURL("http://lorem-rss.herokuapp.com/feed?unit=second&interval=15") // Great RSS feed for testing :)
	feed, err := feedParser.ParseURL(Url)
	if err != nil {
		log.Println("There was an error parsing the URL:", err)
		return
	}

	// Grabs latest message at RSS Item position 0
	log.Println("Generating messageArray...")
	message = fmt.Sprintf("%s\n%s", feed.Items[0].Title, feed.Items[0].Link)
	messageArray = append(messageArray, message)

	if len(last100DiscordMessages) > 0 {
	out:
		for i := 0; i < len(messageArray); i++ {
			for j := 0; j < len(last100DiscordMessages); j++ {
				if messageArray[i] == last100DiscordMessages[j] || strings.Contains(last100DiscordMessages[j], messageArray[i]) {
					log.Println("We have a matching message, shouldn't send an update!")
					// Appends message to the front of botMessageArray
					botMessageArray = append(botMessageArray, messageArray[0])
					break out
				}
			}
		}
	} else {
		message = fmt.Sprintf("%s\n%s", feed.Items[0].Title, feed.Items[0].Link)
		s.ChannelMessageSend(ChannelId, "Here's my latest RSS Message:")
		s.ChannelMessageSend(ChannelId, message)

		botMessageArray = append(botMessageArray, message)
	}

	// Checks to see if there's a difference between messageArray & botMessageArray
	if botMessageArray != nil && messageArray != nil {
		if strings.Contains(botMessageArray[0], message) {
			log.Println("I've posted this message recently, skipping new post.")
		} else {
			log.Println(message)
			log.Println(botMessageArray[0])
			log.Println("Sending message to Discord...")
			_, err := s.ChannelMessageSend(ChannelId, message)
			if err != nil {
				log.Println("There was an error sending your message:", err)
			} else {
				log.Println("Your message:\n", message)
			}

			// Appends message to the front of botMessageArray
			botMessageArray = append(botMessageArray, message)
			copy(botMessageArray[1:], botMessageArray)
			botMessageArray[0] = message
		}
	}

	if botMessageArray == nil {
		log.Println("Sending message to Discord...")
		_, err := s.ChannelMessageSend(ChannelId, message)
		if err != nil {
			log.Println("There was an error sending your message:", err)
		} else {
			log.Println("Your message:\n", message)
		}

		// Appends message to the front of botMessageArray
		botMessageArray = append(botMessageArray, message)
		copy(botMessageArray[1:], botMessageArray)
		botMessageArray[0] = message
	}

	// Clears the botMessageArray if len(botMessageArray) > 1000
	if len(botMessageArray) > 1000 {
		botMessageArray = nil
		return
	}
}

// Sends 5 most recent messages on Bot Start Up
func mostRecentUpdate(s *discordgo.Session) {

	// Initial Parse of RSS feed
	feedParser := gofeed.NewParser()

	// Checks for BasicAuth
	if BasicAuthUsername != "" && BasicAuthPassword != "" {
		feedParser.AuthConfig = &gofeed.Auth{
			Username: BasicAuthUsername,
			Password: BasicAuthPassword,
		}
	}

	log.Println("Parsing RSS Feed...")
	//feed, err := feedParser.ParseURL("http://lorem-rss.herokuapp.com/feed?unit=second&interval=30") // Great RSS feed for testing :)
	feed, err := feedParser.ParseURL(Url)
	if err != nil {
		log.Println("There was an error parsing the URL:", err)
		return
	}

	// Grabs the most recent RSS message
	log.Println("Generating messageArray...")
	for i := 0; i < 1; i++ {
		message = fmt.Sprintf("%s\n%s", feed.Items[i].Title, feed.Items[i].Link)
		messageArray = append(messageArray, message)
	}

	// It's noisy if we don't do it this way, and also exceeds Discord's character limit / message rate limit
	log.Println("Generating bigMessage...")
	convertToStrings := fmt.Sprintf(strings.Join(messageArray, "\n"))
	bigMessage := fmt.Sprintf("Here's the most recent RSS feed item:\n\n%v", convertToStrings)

	s.ChannelMessageSend(ChannelId, bigMessage)
}

func main() {
	// Creating Discord Session Using Provided Bot Token
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Println("Error creating Discord Session:", err)
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
		log.Println("error opening connection,", err)
		return
	} else {
		mostRecentUpdate(dg)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")
	log.Println("Your RSS feed will be parsed any time there's a new message in your Discord Server.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.ChannelMessageSend(ChannelId, "Shutting down...")
	dg.Close()
}
