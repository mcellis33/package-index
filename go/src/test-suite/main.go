package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
)

var debugMode = flag.Bool("debug", false, "Prints some extra information and opens a HTTP server on port 8081")

func main() {
	log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	addr := flag.String("addr", ":8080", "The address:port your server exposes to clients")
	concurrencyLevel := flag.Int("concurrency", 10, "A positive value indicating how many concurrent clients to use")
	randomSeed := flag.Int64("seed", 42, "A positive value used to seed the random number generator")
	unluckiness := flag.Int("unluckiness", 5, "A % showing the probability of something bad happenning, like broken messages being sent or random disconnects")
	flag.Parse()
	rand.Seed(*randomSeed)

	test := MakeTestRun(*addr, *concurrencyLevel, *unluckiness)

	if *debugMode {
		log.Println("Running in DEBUG mode")
		go func() {
			log.Println(http.ListenAndServe("localhost:8081", nil))
		}()
	}

	test.Run()
}
