package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hudsn/rss_go/internal/config"
	"github.com/hudsn/rss_go/internal/database"
	"github.com/lib/pq"
)

type state struct {
	queries *database.Queries
	config  *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlerMap map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	if fn, ok := c.handlerMap[cmd.name]; ok {
		return fn(s, cmd)
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlerMap[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("login command requires a username")
	}
	username := cmd.args[0]
	if err := s.config.SetUser(username); err != nil {
		return err
	}

	_, err := s.queries.GetUser(context.Background(), username)
	if err != nil {
		return err
	}

	fmt.Printf("user %q has been set\n", username)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("register command requires a username")
	}
	username := cmd.args[0]
	if err := s.config.SetUser(username); err != nil {
		return err
	}

	u, err := s.queries.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})
	if err != nil {
		return err
	}

	if err := s.config.SetUser(username); err != nil {
		return err
	}
	fmt.Printf("user %q was created\nuser details: %v", username, u)
	return nil
}

func handleReset(s *state, cmd command) error {
	return s.queries.ResetUsers(context.Background())
}

func handleListUsers(s *state, cmd command) error {
	userList, err := s.queries.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, entry := range userList {
		if entry == s.config.UserName {
			entry += " (current)"
		}
		fmt.Println(entry)
	}
	return nil
}

func handleAgg(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("agg command requires one argument, the time between requests")
	}
	dur, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}
	ticker := time.NewTicker(dur)

	for range ticker.C {
		scrapeNext(s)
	}

	return nil
}

func handleAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("addfeed command requires 2 arguments: a name and a url")
	}
	feedName := cmd.args[0]
	url := cmd.args[1]

	createdFeed, err := s.queries.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      feedName,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return err
	}

	followArgs := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    createdFeed.ID,
	}
	s.queries.CreateFeedFollow(context.Background(), followArgs)

	fmt.Println(createdFeed)
	return nil
}

func handleListFeeds(s *state, cmd command) error {
	entries, err := s.queries.ListFeedsAndUsers(context.Background())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		fmt.Printf("Name: %s\nURL: %s\nCreatedBy: %s\n\n", entry.Name, entry.Url, entry.CreatedByUser)
	}
	return nil
}

func handleFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("follow command requires a url to follow")
	}
	followURL := cmd.args[0]

	f, err := s.queries.GetFeedByURL(context.Background(), followURL)
	if err != nil {
		return err
	}

	followArgs := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    f.ID,
	}

	followResult, err := s.queries.CreateFeedFollow(context.Background(), followArgs)
	if err != nil {
		return err
	}
	fmt.Printf("User %s is now following the feed %q:\n\n", followResult.UserName, followResult.FeedName)
	return nil
}

func handleFollowing(s *state, cmd command, user database.User) error {

	follows, err := s.queries.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}
	fmt.Printf("%s is currently following %d feed(s):\n", s.config.UserName, len(follows))
	for _, entry := range follows {
		fmt.Println(entry.FeedName)
	}

	return nil
}

func handleUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("unfollow command requires a url")
	}
	unfollowTarget := cmd.args[0]

	unfollowParams := database.UnfollowFeedByURLParams{
		UserID: user.ID,
		Url:    unfollowTarget,
	}
	return s.queries.UnfollowFeedByURL(context.Background(), unfollowParams)
}

func handleBrowse(s *state, cmd command, user database.User) error {
	limit := 2
	if len(cmd.args) > 0 {
		conv, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("argument to browse must be a valid integer")
		}
		limit = conv
	}
	postsForUserParams := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	}
	posts, err := s.queries.GetPostsForUser(context.Background(), postsForUserParams)
	if err != nil {
		return err
	}
	for _, entry := range posts {
		fmt.Printf("Title: %s\nDescription: %s\nURL: %s\nPublished At: %s\n\n", entry.Title, entry.Description, entry.Url, entry.PublishedAt.String())
	}

	return nil
}

// helpers

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, c command) error {
		u, err := s.queries.GetUser(context.Background(), s.config.UserName)
		if err != nil {
			return err
		}
		return handler(s, c, u)
	}
}

func scrapeNext(s *state) {

	next, err := s.queries.GetNextFeedToFetch(context.Background())
	if err != nil {
		slog.Error("issue fetching next feed info", "error", err.Error())
		return
	}

	markFetchedParams := database.MarkFeedFetchedParams{
		ID:            next.ID,
		LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}
	err = s.queries.MarkFeedFetched(context.Background(), markFetchedParams)
	if err != nil {
		slog.Error("issue marking feed as fetched", "error", err.Error(), "feed_id", next.ID, "feed_url", next.Url)
		return
	}

	rss, err := fetchFeed(context.Background(), next.Url)
	if err != nil {
		slog.Error("issue fetching feed from source", "error", err.Error(), "feed_url", next.Url)
		return
	}
	fmt.Printf("Ingesting posts for feed %q:\n", rss.Channel.Title)
	newPostCount := 0
	for _, entry := range rss.Channel.Item {
		// hackernews  Mon, 16 Jun 2025 12:32:02 +0000
		// techcrunchj Mon, 16 Jun 2025 15:15:49 +0000
		// Tue, 10 Jun 2025 00:00:00 +0000
		// magic date: "Mon, 02 Jan 2006 15:04:05 -0700"
		pubdate, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", entry.PubDate)
		if err != nil {
			slog.Error("issue parsing time for published entry", "error", err.Error(), "source_rss_title", rss.Channel.Link, "entry_title", entry.Title)
			continue
		}

		createPostParams := database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       entry.Title,
			Url:         entry.Link,
			Description: entry.Description,
			PublishedAt: pubdate,
			FeedID:      next.ID,
		}
		_, err = s.queries.CreatePost(context.Background(), createPostParams)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok {
				if pqErr.Code == "23505" {
					// silent continue if we have a duplicate we're trying to insert; it's expected in a crawler.
					continue
				}
			}
			slog.Error("issue creating post entry", "error", err, "source_rss_title", rss.Channel.Link, "entry_title", entry.Title)
			continue
		}
		newPostCount++
	}
	fmt.Printf("Added %d new posts for feed %q\n\n", newPostCount, rss.Channel.Title)
}
