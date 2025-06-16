package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	"github.com/hudsn/rss_go/internal/config"
	"github.com/hudsn/rss_go/internal/database"

	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.ReadDefaultConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	queries := database.New(db)

	stateInstance := &state{
		queries: queries,
		config:  cfg,
	}

	commandsInstance := &commands{
		handlerMap: make(map[string]func(*state, command) error),
	}

	commandsInstance.register("login", handlerLogin)
	commandsInstance.register("register", handlerRegister)
	commandsInstance.register("reset", handleReset)
	commandsInstance.register("users", handleListUsers)
	commandsInstance.register("agg", handleAgg)
	commandsInstance.register("addfeed", middlewareLoggedIn(handleAddFeed))
	commandsInstance.register("feeds", handleListFeeds)
	commandsInstance.register("follow", middlewareLoggedIn(handleFollow))
	commandsInstance.register("following", middlewareLoggedIn(handleFollowing))
	commandsInstance.register("unfollow", middlewareLoggedIn(handleUnfollow))
	commandsInstance.register("browse", middlewareLoggedIn(handleBrowse))

	args := os.Args
	if len(args) < 2 {
		log.Fatal("program needs to include at least one command")
	}
	commandKeyword := strings.ToLower(args[1])
	remaining := []string{}
	if len(args) > 2 {
		remaining = args[2:]
	}
	cmdInstance := command{
		name: commandKeyword,
		args: remaining,
	}

	err = commandsInstance.run(stateInstance, cmdInstance)
	if err != nil {
		log.Fatal(err)
	}
}
