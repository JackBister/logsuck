package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit"
)

var logRows = []string{
	"Reticulated numSplines=### for userId=#### in timeInMs=###",
	"Setting password=??#?#?#? for userId=####, userName={person.first}",
	"{hacker.verb} {hacker.noun}, {hacker.noun}=###",
	"{company.buzzwords}=### {company.buzzwords}=??? {company.buzzwords}=###",
}

func main() {
	numFiles := flag.Int("numFiles", 1, "The number of files that will be written to. The files will be named log-*.txt where * is an increasing number.")
	sleepTime := flag.Duration("sleepTime", 100*time.Millisecond, "The duration to sleep between logging")

	flag.Parse()

	for i := 0; i < *numFiles; i++ {
		go func() {
			filename := "log-" + strconv.Itoa(i) + ".txt"
			file, err := os.Create(filename)
			if err != nil {
				log.Fatal("Got error when creating file "+filename+":", err)
			}
			logger := log.New(file, "", log.Ldate|log.Lmicroseconds|log.Llongfile)
			for {
				randRow := gofakeit.RandString(logRows)
				logger.Println(gofakeit.Generate(randRow))
				time.Sleep(*sleepTime)
			}
		}()
	}

	select {}
}
