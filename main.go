package main

import (
	"github.com/gregjones/httpcache"
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/stathat"
	"github.com/rs/zerolog"
	"goji.io/pat"
	"os"
	"strconv"
	"time"
)

func main() {
	port := os.Getenv("PORT")

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	in, _ := strconv.Atoi(port)

	server, err := baseapp.NewServer(
		baseapp.HTTPConfig{Address: "0.0.0.0", Port: in},
		baseapp.DefaultParams(logger, "exampleapp.")...,
	)
	if err != nil {
		panic(err)
	}

	if key := os.Getenv("STATHAT_KEY"); key != "" {
		go stathat.Stathat(metrics.DefaultRegistry, 5*time.Second, key)
	}

	// setup github config from env
	conf := githubapp.Config{}
	conf.SetValuesFromEnv("")

	cc, err := githubapp.NewDefaultCachingClientCreator(
		conf,
		githubapp.WithClientUserAgent("codeowners-mention/1.0.0"),
		githubapp.WithClientTimeout(3*time.Second),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
		githubapp.WithClientMiddleware(
			githubapp.ClientLogging(zerolog.DebugLevel),
			githubapp.ClientMetrics(metrics.DefaultRegistry),
		),
	)
	if err != nil {
		panic(err)
	}

	prCommentHandler := &PRCommentHandler{
		ClientCreator: cc,
	}

	dispatcher := githubapp.NewEventDispatcher([]githubapp.EventHandler{prCommentHandler}, conf.App.WebhookSecret, githubapp.WithScheduler(
		githubapp.AsyncScheduler(
			githubapp.WithSchedulingMetrics(metrics.DefaultRegistry),
		),
	))
	server.Mux().Handle(pat.Post(githubapp.DefaultWebhookRoute), dispatcher)

	// Start is blocking
	err = server.Start()
	if err != nil {
		panic(err)
	}
}
