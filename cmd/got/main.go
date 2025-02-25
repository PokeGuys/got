package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pokeguys/got"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

var version string

var HeaderSlice []got.GotHeader

func main() {
	// New context.
	ctx, cancel := context.WithCancel(context.Background())

	interruptChan := make(chan os.Signal, 1)

	signal.Notify(interruptChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	go func() {
		<-interruptChan
		cancel()
		signal.Stop(interruptChan)
		log.Fatal(got.ErrDownloadAborted)
	}()

	// CLI app.
	app := &cli.App{
		Name:  "Got",
		Usage: "The fastest http downloader.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Usage:   "Download `path`, if dir passed the path witll be `dir + output`.",
				Aliases: []string{"o"},
			},
			&cli.StringFlag{
				Name:    "dir",
				Usage:   "Save downloaded file to a `directory`.",
				Aliases: []string{"d"},
			},
			&cli.StringFlag{
				Name:    "file",
				Usage:   "Batch download from list of urls in a `file`.",
				Aliases: []string{"bf", "f"},
			},
			&cli.Uint64Flag{
				Name:    "size",
				Usage:   "Chunk size in `bytes` to split the file.",
				Aliases: []string{"chunk"},
			},
			&cli.UintFlag{
				Name:    "concurrency",
				Usage:   "Chunks that will be downloaded concurrently.",
				Aliases: []string{"c"},
			},
			&cli.StringSliceFlag{
				Name:    "header",
				Usage:   `Set these HTTP-Headers on the requests. The format has to be: -H "Key: Value"`,
				Aliases: []string{"H"},
			},
			&cli.StringFlag{
				Name:    "agent",
				Usage:   `Set user agent for got HTTP requests.`,
				Aliases: []string{"u"},
			},
		},
		Version: version,
		Authors: []*cli.Author{
			{
				Name:  "Mohamed Elbahja and Contributors",
				Email: "bm9qdW5r@gmail.com",
			},
		},
		Action: func(c *cli.Context) error {
			return run(ctx, c)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, c *cli.Context) error {
	var (
		g *got.Got                 = got.NewWithContext(ctx)
		p *progressbar.ProgressBar = progressbar.New(0)
	)

	// Progress func.
	g.ProgressFunc = func(d *got.Download) {
		p.ChangeMax(int(d.TotalSize()))
		p.Add(int(d.Size()))
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	// Create dir if not exists.
	if c.String("dir") != "" {
		if _, err := os.Stat(c.String("dir")); os.IsNotExist(err) {
			os.MkdirAll(c.String("dir"), os.ModePerm)
		}
	}

	// Set default user agent.
	if c.String("agent") != "" {
		got.UserAgent = c.String("agent")
	}

	// Piped stdin
	if info.Mode()&os.ModeNamedPipe > 0 || info.Size() > 0 {
		if err := multiDownload(ctx, c, g, bufio.NewScanner(os.Stdin)); err != nil {
			return err
		}
	}

	// Batch file.
	if c.String("file") != "" {

		file, err := os.Open(c.String("file"))
		if err != nil {
			return err
		}

		if err := multiDownload(ctx, c, g, bufio.NewScanner(file)); err != nil {
			return err
		}
	}

	if c.StringSlice("header") != nil {
		header := c.StringSlice("header")

		for _, h := range header {
			split := strings.SplitN(h, ":", 2)
			if len(split) == 1 {
				return errors.New("malformatted header " + h)
			}
			HeaderSlice = append(HeaderSlice, got.GotHeader{Key: split[0], Value: strings.TrimSpace(split[1])})
		}
	}

	// Download from args.
	for _, url := range c.Args().Slice() {

		if err = download(ctx, c, g, url); err != nil {
			return err
		}

		fmt.Print("\x1b[2K")
		fmt.Printf("✔ %s\n", url)
	}

	return nil
}

func multiDownload(ctx context.Context, c *cli.Context, g *got.Got, scanner *bufio.Scanner) error {
	for scanner.Scan() {

		url := strings.TrimSpace(scanner.Text())

		if url == "" {
			continue
		}

		if err := download(ctx, c, g, url); err != nil {
			return err
		}

		fmt.Print("\x1b[2K")
		fmt.Printf("✔ %s\n", url)
	}

	return nil
}

func download(ctx context.Context, c *cli.Context, g *got.Got, url string) (err error) {
	if url, err = getURL(url); err != nil {
		return err
	}

	return g.Do(&got.Download{
		URL:         url,
		Dir:         c.String("dir"),
		Dest:        c.String("output"),
		Header:      HeaderSlice,
		Interval:    150,
		ChunkSize:   c.Uint64("size"),
		Concurrency: c.Uint("concurrency"),
	})
}

func getURL(URL string) (string, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return "", err
	}

	// Fallback to https by default.
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	return u.String(), nil
}
