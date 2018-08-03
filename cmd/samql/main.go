package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	arg "github.com/alexflint/go-arg"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/mnsmar/samql"
)

// VERSION defines the program version.
const VERSION = "0.1"

// Opts is the struct with the options that the program accepts.
// Opts encapsulates common command line options.
type Opts struct {
	Input string `arg:"positional,required" help:"file (- for STDIN)"`
	Where string `arg:"" help:"SQL clause to match records"`
	Count bool   `arg:"-c" help:"print only the count of matching records"`
}

// Version returns the program name and version.
func (Opts) Version() string { return "samql " + VERSION }

// Description returns an extended description of the program.
func (Opts) Description() string { return "Filters a SAM/BAM file using the SQL clause provided" }

func main() {
	var opts Opts
	arg.MustParse(&opts)

	// Open file for reading.
	fh, err := os.Open(opts.Input)
	if err != nil {
		log.Fatalf("cannot open file: %v", err)
	}

	// Open SAM/BAM reader.
	br, err := bam.NewReader(fh, 2)
	if err != nil {
		fh.Close()
		log.Fatalf("cannot create sam reader: %v", err)
	}

	// Close all readers at the end.
	defer func() {
		if err = fh.Close(); err != nil {
			log.Fatalf("cannot close sam file: %v", err)
		}
		if err = br.Close(); err != nil {
			log.Fatalf("cannot close sam reader: %v", err)
		}
	}()

	// Create new filtering reader that reads from br.
	r := samql.NewReader(br)
	if opts.Where != "" {
		filter, err := samql.Where(opts.Where)
		if err != nil {
			log.Fatalf("filter creation from where clause failed: %v", err)
		}
		r.Filters = append(r.Filters, filter)
	}

	var w *sam.Writer
	if !opts.Count {
		// Create a writer that writes to STDOUT.
		stdout := bufio.NewWriter(os.Stdout)
		defer stdout.Flush()
		w, err = sam.NewWriter(stdout, br.Header(), sam.FlagDecimal)
		if err != nil {
			log.Fatalf("write of header failed: %v", err)
		}
	}

	// Loop on the matching records.
	cnt := 0
	for {
		rec, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("filtering failed: %v", err)
		}
		cnt++

		if opts.Count {
			continue
		}

		if err := w.Write(rec); err != nil {
			log.Fatalf("write failed: %v for %s", err, rec.Name)
		}
	}
	if opts.Count {
		fmt.Println(cnt)
	}
}