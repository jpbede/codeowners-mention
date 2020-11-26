package main

import (
	"context"
	"encoding/json"
	"github.com/google/go-github/v32/github"
	"github.com/jpbede/codeowners-mention/bot"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
)

type PRCommentHandler struct {
	githubapp.ClientCreator
}

func (h *PRCommentHandler) Handles() []string {
	return []string{"pull_request"}
}

func (h *PRCommentHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment event payload")
	}

	repo := event.GetRepo()
	installationID := githubapp.GetInstallationIDFromEvent(&event)

	ctx, _ = githubapp.PreparePRContext(ctx, installationID, repo, event.GetNumber())

	if event.GetAction() != "opened" && event.GetAction() != "reopened" {
		return nil
	}

	// get authenticated client
	client, err := h.NewInstallationClient(installationID)
	if err != nil {
		return err
	}

	// create a new bot
	b := bot.New(ctx, client, repo.GetOwner().GetLogin(), repo.GetName())

	// get all changed files and owners therefor
	files := b.GetChangedFiles(event.GetNumber())
	var owners []string
	for _, file := range files {
		owners = append(owners, b.GetOwners(file)...)
	}

	// now remove the author from the slice
	owners = b.RemoveAuthor(*event.GetPullRequest().GetUser().Login, unique(owners))

	// mention owners
	if len(owners) > 0 {
		b.MentionOwners(owners, event.GetPullRequest().GetNumber())
	}

	// do finish stuff
	b.Finish()

	return nil
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
