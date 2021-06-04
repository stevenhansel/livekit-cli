package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"

	lksdk "github.com/livekit/livekit-sdk-go"

	livekit_cli "github.com/livekit/livekit-cli"
)

func main() {
	app := &cli.App{
		Name:  "livekit-cli",
		Usage: "LiveKit load tester",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Usage:    "URL of LiveKit server",
				EnvVars:  []string{"LIVEKIT_URL"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "api-key",
				EnvVars:  []string{"LIVEKIT_API_KEY"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "api-secret",
				EnvVars:  []string{"LIVEKIT_API_SECRET"},
				Required: true,
			},
			&cli.StringFlag{
				Name:  "room",
				Usage: "name of the room",
				Value: "testroom",
			},
			&cli.DurationFlag{
				Name:  "duration",
				Usage: "duration to run, 1m, 1h, 0 to keep running",
				Value: 0,
			},
			&cli.BoolFlag{
				Name:  "publish",
				Usage: "publish tracks to the room, default false",
			},
			&cli.IntFlag{
				Name:  "count",
				Usage: "number of participants to spin up",
				Value: 1,
			},
			&cli.StringFlag{
				Name:  "identity-prefix",
				Usage: "identity prefix of tester participants, defaults to a random name",
			},
			&cli.Uint64Flag{
				Name:  "video-bitrate",
				Usage: "bitrate (bps) of video track to publish, 0 to disable",
				Value: 1000000,
			},
			&cli.Uint64Flag{
				Name:  "audio-bitrate",
				Usage: "bitrate (bps) of audio track to publish, 0 to disable",
				Value: 50000,
			},
		},
		Action:  loadTest,
		Version: livekit_cli.Version,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}

func loadTest(c *cli.Context) error {
	params := livekit_cli.LoadTesterParams{
		URL:            c.String("url"),
		APIKey:         c.String("api-key"),
		APISecret:      c.String("api-secret"),
		IdentityPrefix: c.String("identity-prefix"),
		Room:           c.String("room"),
		AudioBitrate:   c.Uint64("audio-bitrate"),
		VideoBitrate:   c.Uint64("video-bitrate"),
	}
	if !c.Bool("publish") {
		params.AudioBitrate = 0
		params.VideoBitrate = 0
	}
	if params.IdentityPrefix == "" {
		params.IdentityPrefix = RandStringRunes(5)
	}

	duration := c.Duration("duration")
	count := c.Int("count")
	testers := make([]*livekit_cli.LoadTester, 0, count)

	for i := 0; i < count; i++ {
		testerParams := params
		testerParams.Sequence = i

		tester := livekit_cli.NewLoadTester(testerParams)
		testers = append(testers, tester)
		if err := tester.Start(); err != nil {
			return err
		}

		if c.Bool("publish") {
			videoBitrate := uint32(c.Uint64("video-bitrate"))
			if videoBitrate > 0 {
				err := tester.PublishTrack("video", lksdk.TrackKindVideo, videoBitrate)
				if err != nil {
					return err
				}
			}

			audioBitrate := uint32(c.Uint64("audio-bitrate"))
			if audioBitrate > 0 {
				err := tester.PublishTrack("audio", lksdk.TrackKindAudio, audioBitrate)
				if err != nil {
					return err
				}
			}
		}
	}

	fmt.Printf("started all %d clients\n", count)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	if duration != 0 {
		go func() {
			<-time.After(duration)
			close(done)
		}()
	}

	<-done

	for _, t := range testers {
		t.Stop()
	}

	return nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}