// Copyright 2021 The Logsuck Authors
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
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit"
)

type fakeFile struct {
	name string
	size int
}

func main() {
	numUsers := flag.Int("numUsers", 1, "The number of users to simulate. Each user will have its own randomly generated IP address in the output log.")
	sleepTime := flag.Duration("sleepTime", 100*time.Millisecond, "The duration to sleep between logging")
	flag.Parse()

	printChan := make(chan string)
	outFile, err := os.Create("access-" + time.Now().Format("2006_01_02_15_04_05") + ".txt")
	if err != nil {
		fmt.Println("Got error when opening output file:", err)
		os.Exit(1)
		return
	}

	fakeUrls := make([]fakeFile, 50)
	for i := 0; i < 50; i++ {
		url := gofakeit.Generate("/{lorem.word}/{lorem.word}.{file.extension}")
		size := rand.Intn(i + 1*20000)
		fakeUrls[i] = fakeFile{url, size}
	}

	for i := 0; i < *numUsers; i++ {
		go func(idx int) {
			ipAddr := strconv.Itoa(rand.Intn(256)) + "." + strconv.Itoa(rand.Intn(256)) + "." + strconv.Itoa(rand.Intn(256)) + "." + strconv.Itoa(rand.Intn(256))
			ticker := time.NewTicker(*sleepTime)
			sb := strings.Builder{}
			for {
				<-ticker.C
				sb.WriteString(ipAddr)
				sb.WriteString(" - - [")
				sb.WriteString(time.Now().Format("02/Jan/2006:15:04:05 -0700"))
				method := randomMethod()
				fakeUrlIdx := rand.Intn(50)
				fakeUrl := &fakeUrls[fakeUrlIdx]
				status := randomStatus()
				sb.WriteString(gofakeit.Generate(fmt.Sprintf("] \"%v %v HTTP/1.1\" %v %v \"-\" {internet.browser}", method, fakeUrl.name, status, fakeUrl.size)))
				printChan <- sb.String()
				sb.Reset()
			}
		}(i)
	}

	for {
		outString := <-printChan
		_, err = outFile.WriteString(outString)
		if err != nil {
			fmt.Println("Got error when writing to output file:", err)
		}
		_, err = outFile.WriteString("\n")
		if err != nil {
			fmt.Println("Got error when writing to output file:", err)
		}
	}
}

func randomStatus() int {
	r := rand.Intn(100)
	if r < 80 {
		return 200
	}
	if r < 90 {
		return 204
	}
	if r < 95 {
		return 301
	}
	if r < 97 {
		return 404
	}
	if r < 99 {
		return 400
	}
	if r < 100 {
		return 500
	}
	return 200
}

func randomMethod() string {
	r := rand.Intn(100)
	if r < 80 {
		return "GET"
	}
	if r < 90 {
		return "POST"
	}
	if r < 95 {
		return "DELETE"
	}
	if r < 100 {
		return "PUT"
	}
	return "GET"
}
