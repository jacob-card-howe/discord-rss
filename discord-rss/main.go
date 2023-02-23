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

func discordMessageSentInChannel(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Useful for debugging :)
	//log.Printf("Message sent in channel: %s", m.ChannelID)

	// Listen for discord-rss bot commands
	// i.e. !status
	// @TODO: Implement !help, !pause, !resume, !update, !add, !remove, and !list commands
	if m.Content == "!help" {
		message := fmt.Sprintf("**Commands:**\n`!status` - Check if the bot is running\n`!help` - Display this message\n`!pause` - Pause RSS feed updates (_not implemented_)\n`!resume` - Resume RSS feed updates (_not implemented_)\n`!update` - Manually trigger RSS feed updates (_not implemented_)\n`!add` - Add a new RSS feeds (_not implemented_)\n`!remove` - Remove an RSS feeds (_not implemented_)\n`!list` - List RSS feeds (_not implemented_)")
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if m.Content == "!status" { // @TODO: Add functionality to check state of RSS feed updates (paused or active)
		message := fmt.Sprintf("**Discord RSS bot is currently running! :white_check_mark:**")
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if m.Content == "!pause" { // @TODO: Add functionality to pause RSS feed updates (pause timer?)
		message := fmt.Sprintf("_This is where I would tell you how to pause RSS feed updates, but I haven't implemented this yet_ :sweat_smile:")
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if m.Content == "!resume" { // @TODO: Add functionality to resume RSS feed updates (resume timer?)
		message := fmt.Sprintf("_This is where I would tell you how to resume RSS feed updates, but I haven't implemented this yet_ :sweat_smile:")
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if m.Content == "!update" { // @TODO: Add functionality to manually trigger RSS feed updates
		message := fmt.Sprintf("_This is where I would tell you how to manually trigger RSS feed updates, but I haven't implemented this yet_ :sweat_smile:")
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if strings.HasPrefix(m.Content, "!add") {
		// Parse user message to extract the URL and ChannelID
		parts := strings.Split(m.Content, " ")
		if len(parts) != 3 {
			message := fmt.Sprintf("Invalid syntax. Please use the following syntax: `!add <url> <channel_id>`")
			_, err := s.ChannelMessageSend(m.ChannelID, message)
			if err != nil {
				log.Printf("Error sending message: %s", err)
			}
			return
		}
		url := parts[1]
		channelID := parts[2]

		// Append URL and ChannelID to variables
		Urls = append(Urls, url)
		ChannelIds = append(ChannelIds, channelID)

		// Send confirmation message
		message := fmt.Sprintf("Added URL `%s` to channel `%s`.\n**WARNING**: _This URL will not persist after a bot shutdown, please update your RSS Bot's startup configuration to permanently add this URL._", url, channelID)
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if m.Content == "!remove" { // @TODO: Add functionality to remove RSS feeds from Urls
		message := fmt.Sprintf("_This is where I would tell you how to remove an RSS feed, but I haven't implemented this yet_ :sweat_smile:")
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
	} else if m.Content == "!list" {
		message := "RSS feeds:\n"
		for _, url := range Urls {
			message += fmt.Sprintf("- %s\n", url)
		}
		_, err := s.ChannelMessageSend(m.ChannelID, message)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
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

		ticker := time.NewTicker(time.Duration(TickerTimer) * time.Second)

		done := make(chan bool)

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					getMessageHistory(s, currentUrl, currentChannel, feedParser)
				}
			}
		}()
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
		dg.ChannelMessageSend(ChannelIds[i], "**_Discord RSS bot is now shutting down..._**")
	}
	dg.Close()
}
