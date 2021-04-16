package main

import (
    "fmt"
	"time"
	"github.com/mmcdole/gofeed"
)

func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func parseAWS(t time.Time) {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("https://aws.amazon.com/about-aws/whats-new/recent/feed/")

	var titleArray [5]string

	for lastFive := 0; lastFive < 6; lastFive++ {
		if feed.Items[lastFive].Title == titleArray[lastFive] {
			fmt.Println("No updates...")
			doEvery(15*time.Second, parseAWS)
		} else {
			fmt.Printf("%v. %v\n", lastFive, feed.Items[lastFive].Title)
			fmt.Println(feed.Items[lastFive].Link)
			titleArray[lastFive] = feed.Items[lastFive].Title
			fmt.Println("Adding update to titleArray slice!")
		}

	}
}

func main() {
	doEvery(15*time.Second, parseAWS)
}