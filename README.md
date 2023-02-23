# discord-rss
A simple Discord Bot that periodically parses provided RSS feeds and posts the latest updates to your favorite Discord Server.

## Usage
This project is designed to be run either as a Docker container or as a Go Binary. Once the bot is running, you can use the following commands to interact with it from within your Discord Server:

* `!help` - Displays a list of available commands
* `!status` - Displays the current status of the Bot and RSS parser
* `!pause` - Pauses the RSS parser
* `!resume` - Resumes the RSS parser
* `!add` - Adds a new RSS feed to the parser
* `!remove` - Removes an RSS feed from the parser

### Running the project locally
First, you'll need to be using [Go version 1.16](https://golang.org/doc/go1.16) or later.

Kick things off by downloading any dependencies from `go.mod` by entering `go download` into your terminal.

Next, run `go build`

Once you've built the Go Binary (titled `discord-rss`), navigate to the location of the Binary and run it. You'll get an error because you're missing the necessary parameters for the Bot to function:

The correct syntax looks something like this:
`./discord-rss -t YOUR_BOT_TOKEN -u YOUR_RSS_FEED -c YOUR_DISCORD_CHANNEL_ID -timer INTEGER_VALUE -user YOUR_USERNAME -pass YOUR_PASSWORD`

You can pass in multiple RSS feeds, and multiple channels by separating them with a comma (`,`). 

> Multiple URLs example: `-u "https://www.reddit.com/r/golang/.rss,https://www.reddit.com/r/golang/.rss"`

> Multiple Channels example: `-c "123456789,987654321"`

If you pass in an uneven number of URLs and Channels, the Bot will default to the first Channel for all provided URLs.


### Running the project via Docker
Start by either building the image (`docker build . -t discord-rss:latest`), or by pulling it down from DockerHub (`docker pull howemando/discord-rss`).

Next, run the image (`docker run -e BOT_TOKEN=YOUR_BOT_TOKEN -e RSS_URL=YOUR_RSS_FEED -e CHANNEL_ID=YOUR_DISCORD_CHANNEL_ID -e TIMER_INT=YOUR_TIMER_INT discord-rss`)

## But what about Discord?
To generate a Bot Token, you'll need to go to the [Discord Developer Portal](https://discord.com/developers/applications/). [This article](https://www.freecodecamp.org/news/create-a-discord-bot-with-python/) by [freecodecamp.org](https://www.freecodecamp.org) does a great job of going through the steps / permissions you'll need for a simple Discord Bot.

To get your `CHANNEL_ID`, you'll need to enable developer mode on your Discord Client. [This Support Documentation by Discord](https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) does a really good walkthrough of how to set that up. 

## Additional Documentation / References
* [mmcdole/gofeed](https://github.com/mmcdole/gofeed)
* [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
* [Discord Support](https://support.discord.com/hc/en-us)
* [Discord API Documentation](https://discord.com/developers/docs/intro)