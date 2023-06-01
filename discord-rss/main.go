package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
)

// Variables used to track sent messages to avoid repeats
// Each element in the outer slice is a slice of strings representing the message content sent to a given channel
var (
	messageQueue     [][]string // Messages to be sent in the future
	previousMessages [][]string // Messages that have already been sent
)

// Creates a map to store tickers for each given channel
var tickerMap sync.Map

// Creates a struct to define what a Feed looks like
type RSSFeed struct {
	Url       string // The URL of the RSS feed (i.e. https://example.com/rss)
	ChannelId string // The Discord Channel ID to send RSS feed updates to
	Timer     int    // The interval in which the RSS parser will check for updates (in seconds)
	UserName  string // A basic auth username for the provided RSS feed
	Password  string // A basic auth password for the provided RSS feed
}

// Creates a slice of RSSFeed structs to store all RSS feeds
var rssFeeds []RSSFeed

// Creates a global ticker to be used across multiple functions
var ticker *time.Ticker // Used to keep track of the ticker across multiple functions

// Variables used for command line parameters
// URL, ChannelId, and TickerTimer are all used when a user only wants to parse a SINGLE RSS feed
// and is inherently incompatible when leveraging FilePath
var (
	Token             string // Discord Bot Authentication Token
	FilePath          string // Path to CSV file containing RSS feed information
	Url               string // URL of a SINGLE RSS feed
	ChannelId         string // A SINGLE Discord Channel ID to send RSS feed updates to
	TickerTimer       int    // A SINGLE interval to parse a SINGLE RSS feed (in seconds)
	BasicAuthUsername string // A SINGLE basic auth username for the RSS feed provided by Url
	BasicAuthPassword string // A SINGLE basic auth password for the RSS feed provided by Url
)

func init() {
	flag.StringVar(&Token, "t", "", "Discord authentication token")
	flag.StringVar(&FilePath, "f", "", "Path to CSV file containing information for multiple RSS feeds")
	flag.StringVar(&Url, "u", "", "URL of a SINGLE RSS feed (incompatible with -f)")
	flag.StringVar(&ChannelId, "c", "", "A SINGLE Discord Channel ID to send RSS feed updates to (incompatible with -f)")
	flag.IntVar(&TickerTimer, "timer", 60, "A SINGLE interval to parse a SINGLE RSS feed in seconds (incompatible with -f, defaults to 60 seconds)")
	flag.StringVar(&BasicAuthUsername, "user", "", "A SINGLE basic auth username for the RSS feed provided by Url (incompatible with -f)")
	flag.StringVar(&BasicAuthPassword, "pass", "", "A SINGLE basic auth password for the RSS feed provided by Url (incompatible with -f)")

	flag.Parse()
}

// Retrieves information about a given feed from the rssFeeds slice via the ChannelId
func getFeedInfo(channelId string) RSSFeed {
	// Find the index of the URL and ChannelId in the rssFeeds slice
	var index int = -1
	for i := range rssFeeds {
		if rssFeeds[i].ChannelId == channelId {
			index = i
			break
		}
	}

	if index == -1 {
		log.Printf("Feed not found for channel %s", channelId)
		return RSSFeed{}
	}

	return rssFeeds[index]
}

// !help command
func helpCommand() string {
	return "**Commands:**\n`!help` - Display this message\n`!status` - Check if the bot is running, and if your RSS feed is actively being parsed\n`!pause` - Pause RSS feed updates in a channel \n`!resume` - Resume RSS feed updates in a channel\n`!update` - Manually trigger RSS feed updates\n`!add` - Add a new RSS feeds. See documentation for syntax\n`!remove` - Remove an RSS feed from a channel\n`!list` - List all RSS feeds being parsed by the bot"
}

// !status command
func statusCommand(m *discordgo.MessageCreate) string {
	var response string
	_, ok := tickerMap.Load(m.ChannelID)

	if !ok {
		log.Printf("Ticker not found for channel %s", m.ChannelID)
		response = "**:white_check_mark: RSS Bot is currently running!**\n**:x: RSS Feed Parser is currently not running.**\n\nTo start the RSS Feed Parser, use the `!update` or `!resume` command."
	} else {
		log.Printf("Ticker for channel %s found!", m.ChannelID)
		response = fmt.Sprintf("**:white_check_mark: RSS Bot is currently running!**\n**:alarm_clock: RSS Feed Parser is currently running every %d seconds**", getFeedInfo(m.ChannelID).Timer)
	}
	return response
}

// !add command
func addCommand(m *discordgo.MessageCreate) string {
	// Check if the user has provided all fields for an RSSFeed object
	var (
		url       string
		channelId string
		timer     int
		username  string
		password  string
	)

	_, err := fmt.Sscanf(m.Content, "!add %s %s %d %s %s", &url, &channelId, &timer, &username, &password)
	if err != nil {
		log.Printf("Error parsing !add command: %s", err)
		return "Error: Invalid command syntax. Please use the following syntax: `!add <url> <channelId> <timer> <username> <password>`"
	}

	newFeed := RSSFeed{
		Url:       url,
		ChannelId: channelId,
		Timer:     timer,
		UserName:  username,
		Password:  password,
	}

	// Append the RSSFeed object to the rssFeeds slice
	rssFeeds = append(rssFeeds, newFeed)

	return fmt.Sprintf("Successfully added RSS feed %s to channel %s", url, channelId)
}

// !remove command
func removeCommand(m *discordgo.MessageCreate) string {
	var index int = -1
	for i := range rssFeeds {
		if rssFeeds[i].ChannelId == m.ChannelID {
			index = i
			break
		}
	}

	if index == -1 {
		log.Printf("Feed not found for channel %s", m.ChannelID)
		return "Error: No RSS feed found for this channel."
	}

	// Remove the RSSFeed object from the rssFeeds slice
	rssFeeds = append(rssFeeds[:index], rssFeeds[index+1:]...)
	return "**_RSS feed removed successfully!_**"
}

// Fetches most recent messages sent to a given channel to ensure we're not sending any repeats
func getMessageHistory(s *discordgo.Session, url string, channelId string, feedParser *gofeed.Parser) {
	log.Printf("Getting message history in %s...", channelId)
	// Checks the last 50 messages sent to the channel
	previousMessagesStructs, err := s.ChannelMessages(channelId, 50, "", "", "")
	if err != nil {
		log.Printf("Error getting message history in %s: %s", channelId, err)
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

	// Parse the RSS feed and create a message to check against the previousMessagesMap
	log.Printf("Checking for new messages in %s", url)
	feed, err := feedParser.ParseURL(url)
	if err != nil {
		log.Printf("Error parsing RSS feed: %s", err)
		return
	}

	message := fmt.Sprintf("**%s**\n%s", feed.Items[0].Title, feed.Items[0].Link)

	// Check if the message has been sent previously
	if previousMessagesMap[message] {
		log.Printf("Message has already been sent, not sending again!")
		return
	}

	log.Printf("No previous messages found, sending message to %s", channelId)
	s.ChannelMessageSend(channelId, message)

	// Add the new message to the previousMessagesMap map
	previousMessagesMap[message] = true

	// Drop previous messages if there are too many
	if len(previousMessagesMap) > 50 {
		log.Printf("Dropping previous messages to avoid memory leak")
		previousMessagesMap = make(map[string]bool)
	}
}

// A simple ticker to run a function on an interval
func startTicker(s *discordgo.Session, url string, channelId string, timer int, feedParser *gofeed.Parser) *time.Ticker {
	ticker = time.NewTicker(time.Duration(timer) * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				// Get the most recent sent message to confirm that it is still the most recent
				// If it is not, then there is a new message to send
				getMessageHistory(s, url, channelId, feedParser)
			}
		}
	}()

	return ticker
}

func stopTicker(channelId string) {
	// Get the ticker for a given channel from the tickerMap
	ticker, ok := tickerMap.LoadAndDelete(channelId)

	if !ok {
		log.Printf("Ticker for channel %s not found!", channelId)
		return
	}

	// Stop the ticker
	ticker.(*time.Ticker).Stop()
}

// Parses RSS feeds and sends updates to Discord
func parseRSSFeeds(s *discordgo.Session) {
	// Create a new RSS feed parser
	feedParser := gofeed.NewParser()

	// Loop through each RSS feed in the rssFeeds slice
	for i := 0; i < len(rssFeeds); i++ {
		currentUrl := rssFeeds[i].Url
		currentChannelId := rssFeeds[i].ChannelId

		if rssFeeds[i].UserName != "" && rssFeeds[i].Password != "" {
			// If the RSS feed requires basic auth, set the appropriate headers
			feedParser.AuthConfig = &gofeed.Auth{
				Username: rssFeeds[i].UserName,
				Password: rssFeeds[i].Password,
			}
		}

		if rssFeeds[i].Timer < 15 {
			// If the user has set a timer less than 15 seconds, set it to 15 seconds so that they are not hammering the RSS feed endpoint
			log.Println("Warning: Timer set to less than 15 seconds, setting to 15 seconds. Be kind to your RSS feed providers!")
			rssFeeds[i].Timer = 15
		}

		// Parse the RSS feed and store the result in the feed variable
		log.Printf("Parsing RSS feed: %s", currentUrl)
		feed, err := feedParser.ParseURL(currentUrl)
		if err != nil {
			log.Printf("Error parsing RSS feed: %s", err)
			continue
		}

		// Grab the most recent RSS message from the feed
		log.Printf("Grabbing most recent RSS message from feed: %s", currentUrl)
		message := fmt.Sprintf("**%s**\n%s", feed.Items[0].Title, feed.Items[0].Link)

		// Append the message to the messageQueue slice
		messageQueue = append(messageQueue, []string{message})

		convertToStrings := fmt.Sprintf(strings.Join(messageQueue[i], "\n"))

		s.ChannelMessageSend(currentChannelId, convertToStrings)

		// Start a ticker for this RSS feed & discord channel to check for updates on an interval
		log.Printf("Starting ticker to check for RSS feed items every %d seconds", rssFeeds[i].Timer)
		ticker := startTicker(s, currentUrl, currentChannelId, rssFeeds[i].Timer, feedParser)

		// Store the ticker in the tickerMap to be used to identify tickers for individual
		// pauses and resumes
		tickerMap.Store(currentChannelId, ticker)
	}
}

// Serves as a callback for a MessageCreated Event
func messageReceived(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		// Ignore messages sent by the bot itself
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		var response string

		switch {
		case strings.HasPrefix(m.Content, "!help"):
			response = helpCommand()
		case strings.HasPrefix(m.Content, "!status"):
			response = statusCommand(m)
		case strings.HasPrefix(m.Content, "!add"):
			response = addCommand(m)
			if strings.Contains(response, "Successfully") {
				// If the user has successfully added a new RSS feed, parse it
				stopTicker(m.ChannelID)
				parseRSSFeeds(s)
			}
		case strings.HasPrefix(m.Content, "!remove"):
			response = removeCommand(m)
		case strings.HasPrefix(m.Content, "!pause"):
			stopTicker(m.ChannelID)
			response = "**:pause_button: RSS Feed Parser has been paused.**\n\nTo resume the RSS Feed Parser, use the `!update` or `!resume` command."
		case strings.HasPrefix(m.Content, "!resume"):
			response = "**:arrow_forward: RSS Feed Parser has been resumed.**"
			stopTicker(m.ChannelID)
			startTicker(s, getFeedInfo(m.ChannelID).Url, m.ChannelID, getFeedInfo(m.ChannelID).Timer, gofeed.NewParser())
		case strings.HasPrefix(m.Content, "!update"):
			response = "**:muscle: RSS Feed Parser has been manually triggered.**"
			stopTicker(m.ChannelID)
			startTicker(s, getFeedInfo(m.ChannelID).Url, m.ChannelID, getFeedInfo(m.ChannelID).Timer, gofeed.NewParser())
		case strings.HasPrefix(m.Content, "!list"):
			response = "**RSS Feeds:**\n"
			for i, feed := range rssFeeds {
				response += fmt.Sprintf("%d. %s\n", i+1, feed.Url)
			}
		}

		s.ChannelMessageSend(m.ChannelID, response)

	}
}

// Reads a CSV file and returns a slice of RSSFeed structs
func readCSV(path string) ([]RSSFeed, error) {
	// Attempt to open the CSV file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read the header line to extract default values for each RSS feed
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return rssFeeds, nil
		}
		return nil, err
	}

	// Set default values from the header line
	defaultUrl := header[0]
	defaultChannelId := header[1]
	defaultTimer, err := strconv.Atoi(header[2])
	defaultUserName := header[3]
	defaultPassword := header[4]

	// Read the remaining lines of the CSV file
	for {
		line, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		url := line[0]
		channelId := line[1]
		timer, err := strconv.Atoi(line[2])
		username := line[3]
		password := line[4]

		// Populate any blank values with defaults
		if url == "" {
			url = defaultUrl
		}
		if channelId == "" {
			channelId = defaultChannelId
		}
		if timer == 0 {
			timer = defaultTimer
		}
		if username == "" {
			username = defaultUserName
		}
		if password == "" {
			password = defaultPassword
		}

		feed := RSSFeed{
			Url:       url,
			ChannelId: channelId,
			Timer:     timer,
			UserName:  username,
			Password:  password,
		}

		// Append the RSSFeed object to the rssFeeds slice
		rssFeeds = append(rssFeeds, feed)
	}
	return rssFeeds, nil
}

// Main function
func main() {
	// Check that the user has provided a Discord authentication token, and one of
	// either a CSV file path or a URL and Channel ID
	if Token == "" || (FilePath == "" && (Url == "" || ChannelId == "")) {
		flag.Usage()
		return
	}

	if FilePath != "" {
		// If the user has passed in the path to a CSV, attempt to read it
		log.Printf("Reading CSV file at %s", FilePath)
		_, err := readCSV(FilePath)
		if err != nil {
			log.Fatal("Error opening CSV file: ", err)
		}
	} else {
		// Create a new RSSFeed object using the provided command line parameters
		feed := RSSFeed{
			Url:       Url,
			ChannelId: ChannelId,
			Timer:     TickerTimer,
			UserName:  BasicAuthUsername,
			Password:  BasicAuthPassword,
		}

		// Append the RSSFeed object to the rssFeeds slice
		rssFeeds = append(rssFeeds, feed)
	}

	// Create Discord Session using the provided authentication token (-t)
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal("Error creating Discord Session: ", err)
	}

	// Register the messageReceived function as a callback for a MessageCreated Event
	dg.AddHandler(messageReceived)

	// Set the intentions of the bot to only listen for new messages
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening for new messages
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection: ", err)
	} else {
		log.Println("Connection to Discord established, beginning feed parser")
		parseRSSFeeds(dg)
	}

	// Wait here until CTRL-C or other term signal is received
	log.Println("RSS feed parser is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the active Discord session
	shutdownMessage := "**_Discord RSS bot is now shutting down..._** :zzz:\n\n**WARNING**: Any RSS feeds added via the `!add` command will not persist on the next bot start up. Here's a list of all the RSS feeds that were being parsed:\n\n"

	for i, feed := range rssFeeds {
		shutdownMessage += fmt.Sprintf("%d. %s\n", i+1, feed.Url)
	}

	// Send shutdownMessage to all channels
	for i := 0; i < len(rssFeeds); i++ {
		dg.ChannelMessageSend(rssFeeds[i].ChannelId, shutdownMessage)
	}
	dg.Close()
}
