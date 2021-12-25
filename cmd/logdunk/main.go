// Copyright 2021 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		go func(i int) {
			filename := "log-" + strconv.Itoa(i) + ".txt"
			file, err := os.Create(filename)
			if err != nil {
				log.Fatal("Got error when creating file "+filename+":", err)
			}
			logger := log.New(file, "", log.Ldate|log.Lmicroseconds|log.Llongfile)
			for {
				randRow := gofakeit.RandString(logRows)
				logger.Println(gofakeit.Generate(randRow))
				if sleepTime.Nanoseconds() != 0 {
					time.Sleep(*sleepTime)
				}
			}
		}(i)
	}

	select {}
}
