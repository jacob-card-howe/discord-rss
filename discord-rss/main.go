package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

// Used to keep track of already sent messages
// Each element in the outer array is an array of strings representing messages sent to a given channel

var messageQueue [][]string     // Messages to be sent
var previousMessages [][]string // Messages that have been sent

// Set variables for flags passed to discord-rss
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, strings.Split(value, ",")...)
	return nil
}

var (
	Token             string
	Urls              stringSlice
	ChannelIds        stringSlice
	TickerTimer       int
	BasicAuthUsername string
	BasicAuthPassword string
)

var ticker *time.Ticker // Used to keep track of the ticker across multiple functions

// Initializes the provided flags
func init() {
	flag.StringVar(&Token, "t", "", "Discord authentication token")
	flag.Var(&Urls, "u", "Comma-separated list of RSS feed URLs")
	flag.Var(&ChannelIds, "c", "Comma-separated list of Discord channel IDs")
	flag.IntVar(&TickerTimer, "timer", 60, "Time between feed checks in seconds")
	flag.StringVar(&BasicAuthUsername, "user", "", "Basic auth username")
	flag.StringVar(&BasicAuthPassword, "pass", "", "Basic auth password")

	flag.Parse()
}

func stopTicker() {
	if ticker != nil {
		ticker.Stop()
	}
}

func startTicker(s *discordgo.Session, currentUrl string, currentChannel string, feedParser *gofeed.Parser) {
	ticker = time.NewTicker(time.Duration(TickerTimer) * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				getMessageHistory(s, currentUrl, currentChannel, feedParser)
			}
		}
	}()
}

func addFeed(message string) string {
	var url string
	var channelID string
	_, err := fmt.Sscanf(message, "!add %s %s", &url, &channelID)
	if err != nil {
		return "Error: Invalid input"
	}

	Urls = append(Urls, url)
	ChannelIds = append(ChannelIds, channelID)

	return "**_RSS feed added_**"
}

func removeFeed(message string) string {
	var url string
	var channelID string
	_, err := fmt.Sscanf(message, "!remove %s %s", &url, &channelID)
	if err != nil {
		return "Error: Invalid input"
	}

	// Find the index of the URL and Channel ID to remove
	var index int = -1
	for i, u := range Urls {
		if u == url && ChannelIds[i] == channelID {
			index = i
			break
		}
	}

	if index == -1 {
		return "Error: Feed not found"
	}

	// Remove the URL and Channel ID from the slice
	Urls = append(Urls[:index], Urls[index+1:]...)
	ChannelIds = append(ChannelIds[:index], ChannelIds[index+1:]...)

	return "**_RSS feed removed_**"
}

func discordMessageSentInChannel(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		var response string

		switch {
		case strings.HasPrefix(m.Content, "!help"):
			response = "**Commands:**\n`!status` - Check if the bot is running\n`!help` - Display this message\n`!pause` - Pause RSS feed updates (_not implemented_)\n`!resume` - Resume RSS feed updates (_not implemented_)\n`!update` - Manually trigger RSS feed updates (_not implemented_)\n`!add` - Add a new RSS feeds (_not implemented_)\n`!remove` - Remove an RSS feeds (_not implemented_)\n`!list` - List RSS feeds (_not implemented_)"
		case strings.HasPrefix(m.Content, "!status"):
			if ticker != nil {
				response = fmt.Sprintf("**:white_check_mark: RSS Bot is currently running!**\n**:alarm_clock: RSS Feed Parser is currently running every %d seconds**", TickerTimer)
			} else {
				response = "**:white_check_mark: RSS Bot is currently running!**\n**:x: RSS Feed Parser is currently not running.**\n\nTo start the RSS Feed Parser, use the `!update` or `!resume` command."
			}
		case strings.HasPrefix(m.Content, "!add"):
			response = addFeed(m.Content)
			if response == "**_RSS feed added_**" {
				stopTicker()
				parseRSSFeeds(s)
			}
		case strings.HasPrefix(m.Content, "!remove"):
			response = removeFeed(m.Content)
			if response == "**_RSS feed removed_**" {
				stopTicker()
				parseRSSFeeds(s)
			}
		case strings.HasPrefix(m.Content, "!pause"):
			stopTicker()
			response = "**:pause_button: RSS Feed Parser has been paused.**\n\nTo resume the RSS Feed Parser, use the `!update` or `!resume` command."
		case strings.HasPrefix(m.Content, "!resume"):
			response = "**:arrow_forward: RSS Feed Parser has been resumed.**"
			parseRSSFeeds(s)
		case strings.HasPrefix(m.Content, "!update"):
			response = "**:muscle: RSS Feed Parser has been manually triggered.**"
			stopTicker()
			parseRSSFeeds(s)
		case strings.HasPrefix(m.Content, "!list"):
			response = "**RSS Feeds:**\n"
			for i, url := range Urls {
				response += fmt.Sprintf("%v. `%s` - %s\n", i+1, ChannelIds[i], url)
			}
		}

		s.ChannelMessageSend(m.ChannelID, response)
	}
}

func getMessageHistory(s *discordgo.Session, url string, channel string, feedParser *gofeed.Parser) {
	log.Printf("Getting message history in %s...", channel)
	previousMessagesStructs, err := s.ChannelMessages(channel, 100, "", "", "")
	if err != nil {
		log.Printf("Error getting message history: %s", err)
		return
	}

	// Create a map of previous messages for faster lookup
	previousMessagesMap := make(map[string]bool)
	for i := 0; i < len(previousMessagesStructs); i++ {
		if previousMessagesStructs[i].Author.ID == s.State.User.ID {
			message := previousMessagesStructs[i].Content
			previousMessagesMap[message] = true
		}
	}

	log.Printf("Checking for RSS feed updates for %s", url)
	feed, err := feedParser.ParseURL(url)
	if err != nil {
		log.Printf("Error parsing RSS feed: %s", err)
		return
	}

	message := fmt.Sprintf("**%s**\n%s", feed.Items[0].Title, feed.Items[0].Link)

	// Check if the message has already been sent
	if previousMessagesMap[message] {
		log.Println("Message already sent, not sending again!")
		//log.Println("This was the message: ", message) // Commenting out to prevent flood of logs
		return
	}

	log.Printf("No previous messages found, sending message in %s...", channel)
	s.ChannelMessageSend(channel, message)

	// Add the message to the previous messages map
	previousMessagesMap[message] = true

	// Clear previous messages if there are too many
	if len(previousMessagesMap) > 100 {
		previousMessagesMap = make(map[string]bool)
	}
}

func parseRSSFeeds(s *discordgo.Session) {
	// Create a new RSS feed parser
	feedParser := gofeed.NewParser()

	// Loop through each RSS feed URL
	for i := 0; i < len(Urls); i++ {

		currentUrl := Urls[i]
		currentChannel := ChannelIds[i]

		if BasicAuthUsername != "" && BasicAuthPassword != "" {
			feedParser.AuthConfig = &gofeed.Auth{
				Username: BasicAuthUsername,
				Password: BasicAuthPassword,
			}
		}

		// Parse the RSS feed
		log.Printf("Parsing RSS feed: %s", currentUrl)
		feed, err := feedParser.ParseURL(currentUrl)
		if err != nil {
			log.Printf("Error parsing RSS feed: %s", err)
			return
		}

		// Grab the most recent RSS message
		log.Printf("Grabbing most recent RSS message from feed: %s", currentUrl)
		message := fmt.Sprintf("**%s**\n%s", feed.Items[0].Title, feed.Items[0].Link)

		// Append message to messageQueue
		messageQueue = append(messageQueue, []string{message})

		// Generate a big message for initial RSS feed update
		log.Println("Generating initial RSS feed update message...")
		convertToStrings := fmt.Sprintf(strings.Join(messageQueue[i], "\n"))
		bigMessage := fmt.Sprintf("Here's the most recent RSS feed item:\n\n%v", convertToStrings)

		s.ChannelMessageSend(currentChannel, bigMessage)

		// Start a ticker to check for new RSS feed items
		log.Printf("Starting ticker to check for new RSS feed items every %d seconds...", TickerTimer)

		startTicker(s, currentUrl, currentChannel, feedParser)
	}
}

func main() {
	// Check that all required flags are set
	if Token == "" || len(Urls) == 0 || len(ChannelIds) == 0 {
		flag.Usage()
		return
	}

	// Check that the number of URLs and channel IDs match
	if len(ChannelIds) < len(Urls) {
		log.Printf("Warning: More URLs than channel IDs provided. The Discord RSS bot will only post RSS feeds to the first channel provided: %s", ChannelIds[0])
		firstChannelId := ChannelIds[0]
		for len(ChannelIds) < len(Urls) {
			ChannelIds = append([]string{firstChannelId}, ChannelIds...)
		}
	}

	// Create Discord Session using
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal("Error creating Discord Session: ", err)
	}

	// Registers the discordMessageSentInChannel function as a callback for a MessageCreated event
	dg.AddHandler(discordMessageSentInChannel)

	// Sets the intentions of the bot
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection: ", err)
	} else {
		parseRSSFeeds(dg)
	}

	// Wait here until CTRL-C or other term signal is received
	log.Println("Discord RSS bot is now running. Press CTRL-C to exit.")
	log.Printf("Your provided RSS feed URLs will be parsed every %d seconds.", TickerTimer)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session
	for i := 0; i < len(ChannelIds); i++ {
		dg.ChannelMessageSend(ChannelIds[i], "**_Discord RSS bot is now shutting down..._** :zzz:\n\n**WARNING**: Any RSS feeds added via the `!add` command will not persist on the next bot start up. Here's a list of all the RSS feeds that were being parsed:\n\n"+strings.Join(Urls, "\n"))
	}
	dg.Close()
}
