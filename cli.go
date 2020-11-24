package main

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"

	"github.com/knadh/pfxsigner/internal/processor"
	"github.com/urfave/cli"
)

// initCLI initializes CLI mode.
func initCLI(c *cli.Context) error {
	// Start workers.
	var (
		num  = c.Int("workers")
		jobQ = make(chan processor.Job, 1000)
	)
	logger.Printf("starting %d workers", num)
	proc.Wg.Add(num)
	for n := 0; n < num; n++ {
		go proc.Listen(jobQ)
	}

	// Start a separate goroutine to read stdin.
	go func() {
		log.Println("waiting for jobs from stdin")
		if err := readJobsFromStdin(jobQ); err != nil {
			log.Fatalf("error reading jobs from stdin: %v", err)
		}
		close(jobQ)
	}()

	// Wait until all jobs are done.
	proc.Wg.Wait()
	var (
		s       = proc.GetStats()
		elapsed = time.Now().Sub(s.StartTime)
	)
	log.Printf("%d succeeded. %d failed.", s.JobsDone, s.JobsFailed)
	log.Printf("%0.2f seconds. %0.2f / sec",
		elapsed.Seconds(),
		float64(s.JobsDone+s.JobsFailed)/elapsed.Seconds())

	return nil
}

// readJobsFromStdin reads PDF jobs from stdin in the following format
// and sends them to workers for processing.
// inFile|outFile
// inFile|outFile
// inFile|outFile
// ...
func readJobsFromStdin(ch chan processor.Job) error {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if len(line) == 0 {
			continue
		}

		chunks := strings.Split(line, "|")
		if len(line) == 0 {
			continue
		}
		if len(chunks) != 3 || len(chunks[0]) == 0 || len(chunks[1]) == 0 || len(chunks[2]) == 0 {
			logger.Printf("skipping invalid item: %s", line)
			continue
		}

		ch <- processor.Job{
			CertName: chunks[0],
			InFile:   chunks[1],
			OutFile:  chunks[2]}
	}
	return s.Err()
}
