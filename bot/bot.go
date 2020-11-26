package bot

import (
	"context"
	"encoding/base64"
	"github.com/google/go-github/v32/github"
	"github.com/hairyhenderson/go-codeowners"
	"github.com/rs/zerolog"
	"strings"
)

type Bot struct {
	ctx      context.Context
	logger   *zerolog.Logger
	ghClient *github.Client
	cache    *Cache

	repoOwner, repoName string
}

func New(ctx context.Context, ghClient *github.Client, repoOwner, repoName string) *Bot {
	bot := &Bot{
		ctx:       ctx,
		ghClient:  ghClient,
		logger:    zerolog.Ctx(ctx),
		repoOwner: repoOwner,
		repoName:  repoName,

		cache: &Cache{},
	}

	bot.cache.Connect()

	return bot
}

func (b *Bot) getRepoURI() string {
	return b.repoOwner + "/" + b.repoName
}

func (b *Bot) GetOwners(path string) []string {
	// check if there is a cached ownersfile
	var ownersFile string
	if cachedOwnersFile, err := b.cache.GetOwnersFileForRepo(b.getRepoURI()); err != nil {
		b.logger.Error().Err(err).Msg("Error while getting cached ownersfile")
	} else {
		ownersFile = cachedOwnersFile
	}

	// if not get if from Github
	if ownersFile == "" {
		if file, _, _, err := b.ghClient.Repositories.GetContents(b.ctx, b.repoOwner, b.repoName, "./CODEOWNERS", nil); err != nil {
			b.logger.Error().Err(err).Msg("Failed to comment on pull request")
		} else {
			b.logger.Debug().Msgf("Successfully got codeowners from Github for %s", b.getRepoURI())
			ownersFile = *file.Content
			b.cache.SetOwnersFileForRepo(b.getRepoURI(), ownersFile)
		}
	}

	// check if there is now something
	if ownersFile != "" {
		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(ownersFile))
		co, _ := codeowners.FromReader(dec, "/")
		return co.Owners(path)
	} else {
		return []string{}
	}
}

func (b *Bot) GetChangedFiles(prNumber int) []string {
	if changedFiles, _, err := b.ghClient.PullRequests.ListFiles(b.ctx, b.repoOwner, b.repoName, prNumber, nil); err != nil {
		b.logger.Error().Err(err).Msgf("Failed to get pull request #%i on repo %s", prNumber, b.getRepoURI())
	} else {
		var filenames []string
		for _, file := range changedFiles {
			filenames = append(filenames, *file.Filename)
		}
		return filenames
	}
	return []string{}
}

func (b *Bot) MentionOwners(owners []string, prNum int) {
	str := "Based on `CODEOWNERS` maybe "
	for _, username := range owners {
		str += username + " "
	}
	if len(owners) > 1 {
		str += " are interested"
	} else {
		str += " is interested"
	}

	prComment := github.IssueComment{
		Body: &str,
	}

	if _, _, err := b.ghClient.Issues.CreateComment(b.ctx, b.repoOwner, b.repoName, prNum, &prComment); err != nil {
		b.logger.Error().Err(err).Msgf("Got error while creating comment for PR #%i on repo %s", prNum, b.getRepoURI())
	}
}

// RemoveAuthor removes the given author from the owners
func (b *Bot) RemoveAuthor(author string, owners []string) []string {
	var result []string
	for _, owner := range owners {
		if owner != author {
			result = append(result, owner)
		}
	}
	return result
}

func (b *Bot) Finish() {
	b.cache.client.Close()
}
