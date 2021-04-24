# discord-rss
A repo for me to work on a Golang powered RSS Bot that can be tossed into a Docker container and run anywhere.

## Usage
Once the bot is up and running, send a message anywhere in your discord server to pull down the 5 latest RSS Items from your RSS Feed of choice

## Running the project
There are two ways you can run this project.

### Locally
First of all, you'll need to be using [Go version 1.16](https://golang.org/doc/go1.16).

Kick things off by downloading any dependencies from `go.mod` by entering `go download` into your terminal.

Next, run `go build`

Once you've built the Go Binary (titled `discord-rss`), navigate to the location of the Binary and run it. You'll get an error because you're missing the necessary parameters for the Bot to function:

The correct syntax looks something like this:
`./discord-rss -t YOUR_BOT_TOKEN -u YOUR_RSS_FEED -c YOUR_DISCORD_CHANNEL_ID`

Where you'll replace the capitalized strings with your own values. 


### Docker
Start by either building the image (`docker build . -t discord-rss:latest`), or by pulling it down from DockerHub (`docker pull howemando/discord-rss`).

Next, run the image (`docker run -e BOT_TOKEN=YOUR_BOT_TOKEN -e RSS_URL=YOUR_RSS_FEED -e CHANNEL_ID=YOUR_DISCORD_CHANNEL_ID discord-rss`)

## But what about Discord?
To generate a Bot Token, you'll need to go to the [Discord Developer Portal](https://discord.com/developers/applications/). [This article](https://www.freecodecamp.org/news/create-a-discord-bot-with-python/) by [freecodecamp.org](https://www.freecodecamp.org) does a great job of going through the steps / permissions you'll need for a simple Discord Bot.

To get your `CHANNEL_ID`, you'll need to enable developer mode on your Discord Client. [This Support Documentation by Discord](https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) does a really good walkthrough of how to set that up. 

## Additional Documentation / References
* [mmcdole/gofeed](https://github.com/mmcdole/gofeed)
* [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
* [Discord Support](https://support.discord.com/hc/en-us)
* [Discord API Documentation](https://discord.com/developers/docs/intro)