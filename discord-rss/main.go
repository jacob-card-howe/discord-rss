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

// Declare a global wait group to ensure all goroutines are finished before exiting

// A struct to define what an RSS Feed looks like
type RSSFeed struct {
	Url          string // The URL of the RSS feed (i.e. https://example.com/rss)
	ChannelId    string // The Discord Channel ID to send RSS feed updates to
	Timer        int    // The interval (in seconds) in which the RSS parser will check a feed for updates
	UserName     string // A basic auth username for the provided RSS feed
	Password     string // A basic auth password for the provided RSS feed
	ActiveStatus bool   // A boolean to track if the RSS feed should be actively parsed or not
}

// A struct to define a slice of RSS feeds to prevent concurrent access the slice
type RSSFeeds struct {
	mutex    sync.Mutex // A mutex to ensure only one thread can access the slice at a time
	RSSFeeds []RSSFeed  // The slice of RSS feeds
}

// A struct to define a slice of Discord messages to prevent concurrent access to the slice
type MessageQueue struct {
	mutex        sync.Mutex          // A mutex to ensure only one thread can access the slice at a time
	MessageQueue map[string][]string // The slice of Discord messages
}

// Initializing variables for RSS feeds and Message Queue
var rssFeeds RSSFeeds
var messageQueue MessageQueue

// Variables used for command line parameters
var (
	Token             string // Discord Bot Authentication Token
	FilePath          string // Path to CSV file containing RSS feed information
	Url               string // URL of a SINGLE RSS feed
	ChannelId         string // A SINGLE Discord Channel ID to send RSS feed updates to
	TickerTimer       int    // A SINGLE interval to parse a SINGLE RSS feed (in seconds)
	BasicAuthUsername string // A SINGLE basic auth username for the RSS feed provided by Url
	BasicAuthPassword string // A SINGLE basic auth password for the RSS feed provided by Url
)

// Initialize flags for command line parameters
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

// Updates the RSSFeeds slice with a new RSSFeed struct
func updateRSSFeeds(feed *RSSFeed) {
	// Search for the feed by URL in the RSSFeeds slice
	for i, f := range rssFeeds.RSSFeeds {
		if f.Url == feed.Url {
			// Update the ActiveStatus of the found feed
			rssFeeds.mutex.Lock()
			rssFeeds.RSSFeeds[i].ActiveStatus = feed.ActiveStatus
			rssFeeds.mutex.Unlock()
		}
	}

	// Feed not found, append it to the RSSFeeds slice
	rssFeeds.RSSFeeds = append(rssFeeds.RSSFeeds, *feed)
}

// Updates the MessageQueue slice with the latest RSS feed item
func updateMessageQueue(url string, message string) {
	messageQueue.mutex.Lock()
	defer messageQueue.mutex.Unlock()

	if len(messageQueue.MessageQueue[url]) > 0 {
		messageQueue.MessageQueue[url] = nil
	}

	messageQueue.MessageQueue[url] = append(messageQueue.MessageQueue[url], message)
}

// Compare messages to be sent to Discord with messages that have already been sent to avoid duplicates
func compareMessages(s *discordgo.Session, url string, channelId string) {

	// Fetch the last 50 messages from the Discord channel
	sentMessages, err := s.ChannelMessages(channelId, 50, "", "", "")
	if err != nil {
		log.Println("Error fetching messages from Discord channel:", err)
		return
	}

	// Create a map of previous messages for faster lookup
	previousMessagesMap := make(map[string]bool)
	for i := 0; i < len(sentMessages); i++ {
		if sentMessages[i].Author.ID == s.State.User.ID {
			message := sentMessages[i].Content
			previousMessagesMap[message] = true
		}
	}

	// Check if the latest message has already been sent to Discord
	if previousMessagesMap[messageQueue.MessageQueue[url][0]] {
		log.Printf("Message already sent to Discord channel, skipping %s", url)
	} else {
		// Send the latest message to Discord
		log.Printf("Sending message for %s to Discord channel %s", url, channelId)
		_, err := s.ChannelMessageSend(channelId, messageQueue.MessageQueue[url][0])
		if err != nil {
			log.Println("Error sending message to Discord channel:", err)
			return
		}
	}
}

// Parses a given RSS feed and sends the latest item for comparison
func parseRSSFeed(s *discordgo.Session, feed *RSSFeed) {
	ticker := time.Tick(time.Duration(feed.Timer) * time.Second)
	backoff := 1
	for {
		select {
		case <-ticker:
			// Create new feed parser
			feedParser := gofeed.NewParser()

			// Configure parser with credentials if needed
			if feed.UserName != "" && feed.Password != "" {
				feedParser.AuthConfig = &gofeed.Auth{
					Username: feed.UserName,
					Password: feed.Password,
				}
			}

			// Configures a backoff should the RSS feed be unavailable
			feedItems, err := feedParser.ParseURL(feed.Url)
			if err != nil {
				log.Printf("Error parsing %s: %s", feed.Url, err)
				log.Printf("Retrying in %d seconds...", backoff+feed.Timer)
				time.Sleep(time.Duration(backoff) * time.Second)
				backoff *= 2
				continue
			}

			// Reset backoff if no error
			backoff = 1

			// Format most recent item in RSS Feed
			message := fmt.Sprintf("**%s**\n%s", feedItems.Title, feedItems.Items[0].Link)

			// Update the MessageQueue slice with the latest item
			updateMessageQueue(feed.Url, message)

			// Compare messages to ensure no duplicates
			compareMessages(s, feed.Url, feed.ChannelId)
		}
	}
}

// Configures RSS Feed parsers and starts them concurrently
func configureRSSFeeds(s *discordgo.Session) {
	// Initialize the MessageQueue map
	messageQueue.MessageQueue = make(map[string][]string)

	rssFeeds.mutex.Lock()
	defer rssFeeds.mutex.Unlock()
	for _, feed := range rssFeeds.RSSFeeds {
		if !feed.ActiveStatus {
			continue
		}

		// Creates a copy of the feed in memory for manipulation
		localFeed := feed

		go func(feed *RSSFeed) {
			parseRSSFeed(s, feed)
		}(&localFeed)
	}
}

// Listens for commands prefixed by `!` coming from Discord
func discordCommandRecieved(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages sent by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Look for messages that are prefixed with `!`
	if strings.HasPrefix(m.Content, "!") {
		// Initialize the response variable
		var response string

		switch {
		case strings.HasPrefix(m.Content, "!help"):
			response = "**Commands:**\n`!help` - Display this message\n`!status` - Check if the bot is running, and if your RSS feed is actively being parsed\n`!list` - List all RSS feeds being parsed by the bot"
		case strings.HasPrefix(m.Content, "!status"):
			var url string
			_, err := fmt.Sscanf(m.Content, "!status %s", &url)
			if err != nil {
				log.Println("Error grabbing URL from status command:", err)
				response = ":x: Error fetching status, please follow the syntax provided in the documentation and check the logs for more information."
			}

			for _, feed := range rssFeeds.RSSFeeds {
				if feed.Url == url {
					if feed.ActiveStatus {
						response = fmt.Sprintf(":white_check_mark: %s is currently being parsed in <#%s>", feed.Url, feed.ChannelId)
						break
					} else {
						response = fmt.Sprintf(":x: %s is not currently being parsed in <#%s>", feed.Url, feed.ChannelId)
						break
					}
				}
			}
		case strings.HasPrefix(m.Content, "!list"):
			var channelList string
			loopCount := 0
			rssFeeds.mutex.Lock()
			for _, feed := range rssFeeds.RSSFeeds {
				if feed.ActiveStatus && feed.ChannelId == m.ChannelID {
					channelList += fmt.Sprintf("%d. %s\n", loopCount+1, feed.Url)
				}
			}

			if channelList == "" {
				response = ":x: No RSS feeds are currently being parsed in this channel."
				rssFeeds.mutex.Unlock()
			} else {
				response = fmt.Sprintf("**RSS Feeds being parsed in <#%s>**\n%s", m.ChannelID, channelList)
				rssFeeds.mutex.Unlock()
			}
		}

		s.ChannelMessageSend(m.ChannelID, response)
	}
}

// Reads CSVs and builds a slice of RSSFeed structs
func readCSV(path string) error {
	// Attempt to open the CSV file
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read the header line to extract default values for blank RSS Feed fields
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	// Set default values from header line
	defaultUrl := header[0]
	defaultChannelId := header[1]
	defaultTimer, _ := strconv.Atoi(header[2])
	defaultUserName := header[3]
	defaultPassword := header[4]

	// Read remaining lines of CSV file
	for {
		line, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		url := line[0]
		channelId := line[1]
		timer, _ := strconv.Atoi(line[2])
		username := line[3]
		password := line[4]

		// Populate an RSSFeed object with the values from the CSV file
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
			Url:          url,
			ChannelId:    channelId,
			Timer:        timer,
			UserName:     username,
			Password:     password,
			ActiveStatus: true,
		}

		// Append the RSSFeed struct to the FeedsSlice
		updateRSSFeeds(&feed)
	}
	// log.Println(rssFeeds.RSSFeeds)
	return nil
}

// Main function
func main() {
	// Check that the user has provided a Discord authentication token & RSS feed information
	if Token == "" || (FilePath == "" && (Url == "" || ChannelId == "")) {
		log.Println("You must provide a Discord authentication token and RSS feed information.")
		flag.Usage()
		return
	}

	// If the user has provided a CSV file, parse it and add each RSS feed to the FeedsSlice
	if FilePath != "" {
		log.Printf("Reading CSV file at %s", FilePath)
		err := readCSV(FilePath)
		if err != nil {
			log.Fatal("Error opening CSV file: ", err)
		}
	} else {
		// If no CSV provided, create a new RSSFeed struct and add it to the FeedsSlice
		feed := RSSFeed{
			Url:          Url,
			ChannelId:    ChannelId,
			Timer:        TickerTimer,
			UserName:     BasicAuthUsername,
			Password:     BasicAuthPassword,
			ActiveStatus: true,
		}

		// Append the RSSFeed struct to the FeedsSlice
		updateRSSFeeds(&feed)
	}

	// Create a new Discord session using the provided authentication token
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	// Identify the intents that the bot will use
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Listen for commands prefixed by `!` coming from Discord
	dg.AddHandler(discordCommandRecieved)

	// Open a websocket connection to Discord and begin parsing RSS feeds
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening Discord websocket: ", err)
	} else {
		log.Println("Discord websocket connection opened successfully")
		go configureRSSFeeds(dg)
	}

	// Wait here until CTRL-C or other term signal is received
	log.Println("RSS feed parser is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Close the Discord websocket connection
	log.Println("Closing Discord websocket connection...")
	dg.Close()
}
