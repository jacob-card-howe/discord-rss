# discord-rss
A simple Discord Bot that periodically parses provided RSS feeds and posts the latest updates to your favorite Discord Server.

### Publication Status:
[![Build & Publish Docker Image](https://github.com/jacob-card-howe/discord-rss/actions/workflows/publish-docker.yaml/badge.svg)](https://github.com/jacob-card-howe/discord-rss/actions/workflows/publish-docker.yaml)

## Usage
This project is designed to be run either as a Docker container or as a Go Binary. Once the bot is running, you can use the following commands to interact with it from within your Discord Server:

* `!help` - Displays a list of available commands
* `!status <RSS feed URL>` - Displays the current status of the Bot and RSS parser for a given URL
* `!list` - Displays a list of all RSS feeds currently being parsed in the channel this command is run in

### `feeds.csv`
The `feeds.csv` file is used to store RSS feed information for `discord-rss` to parse. This file can have any name, but must follow the following format:
```csv
DEFAULT_URL,DEFAULT_CHANNEL_ID,DEFAULT_TIMER_INT,DEFAULT_USERNAME,DEFAULT_PASSWORD
https://www.example.com/rss,123456789,60,username,password
https://www.example2.com/rss,,,,
https://www.example3.com/rss,987654321,10,,
https://www.example4.com/rss,,,,another_password
```
The first line of the CSV exists for default values. If a value is not provided for a given RSS feed, the default value will be used instead. If a default value is not provided, and you do not provide a value for _at least_ the `url`, `channel id` and `timer` for a given RSS feed, `discord-rss` will not parse that feed.

Here's a quick breakdown of what each line in the provided CSV is doing:
1. This line is setting default values for the rest of the CSV. **Note**: Be sure to use proper values for each default or `discord-rss` will not parse your RSS feeds.
1. This line is parsing the RSS feed at `https://www.example.com/rss` and posting updates to the Discord Channel with ID `123456789` every `60` seconds. This line also provides a username and password for the RSS feed, which is used for authentication.
1. This line is parsing the RSS feed at `https://www.example2.com/rss`. This line does not provide values for `channel id`, `timer`, `username` or `password`, so the default values will be used instead.
1. This line is parsing the RSS feed at `https://www.example3.com/rss`. This line specifies a `987654321` as its channel ID, and a 10 second interval timer. This line will use the default values for `username` and `password`.
1. This line is parsing the RSS feed at `https://www.example4.com/rss`. This line uses default values for all fields except `url` and `password`.

### Running `discord-rss` from its binary
If you're running `discord-rss` from its binary, you'll need to either compile it yourself using `go build`, or download the latest version from the [`discord-rss` Releases page](https://github.com/jacob-card-howe/discord-rss/releases).

The ***recommended*** syntax for running the project is:
```sh
./discord-rss -t YOUR_BOT_TOKEN -f "/path/to/your/feeds/file.csv"
```

If you are only parsing a single RSS feed, you can use the following syntax:
```sh
./discord-rss -t YOUR_BOT_TOKEN -u YOUR_RSS_FEED_URL -c YOUR_DISCORD_CHANNEL_ID -timer INTEGER_VALUE -user YOUR_USERNAME -pass YOUR_PASSWORD
```
> **⚠️ _Note:_** You cannot use both the `-f` and `-u` flags at the same time. If you do, `discord-rss` will default to using the `-f` flag.

### Running `discord-rss` from a Docker container
If you are building the Docker image yourself, you can put your `feeds.csv` file in the `discord-rss/` subdirectory of this project. This file will be copied into your local Docker image upon build. You can then run the following command to build and run the Docker container:
```sh
docker build . -t discord-rss:local && \
docker run -d -e BOT_TOKEN=YOUR_BOT_TOKEN discord-rss:local
```

Otherwise, pull down the latest version of `discord-rss` and run the image with the following command:
```sh
docker pull howemando/discord-rss && \
docker run -d -e BOT_TOKEN=YOUR_BOT_TOKEN -v "/path/to/your/feeds/file.csv:/app/feeds.csv" howemando/discord-rss:latest
```

> **⚠️ _Note:_** If you are pulling the image down, Docker will require permission to access the path leading to `feeds.csv`. If you do not provide this permission, `discord-rss` will not be able to parse your RSS feeds. For more information, see Docker's documentation on [Voumes](https://docs.docker.com/storage/volumes/).

## But what about Discord?
To generate a Bot Token, you'll need to go to the [Discord Developer Portal](https://discord.com/developers/applications/). [This article](https://www.freecodecamp.org/news/create-a-discord-bot-with-python/) by [freecodecamp.org](https://www.freecodecamp.org) does a great job of going through the steps / permissions you'll need for a simple Discord Bot.

To get your `CHANNEL_ID`, you'll need to enable developer mode on your Discord Client. [This Support Documentation by Discord](https://support.discord.com/hc/en-us/articles/206346498-Where-can-I-find-my-User-Server-Message-ID-) does a really good walkthrough of how to set that up.

## Additional Documentation / References
* [mmcdole/gofeed](https://github.com/mmcdole/gofeed)
* [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo)
* [Discord Support](https://support.discord.com/hc/en-us)
* [Discord API Documentation](https://discord.com/developers/docs/intro)
* [Docker Volumes](https://docs.docker.com/storage/volumes/)
