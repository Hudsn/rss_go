# RSS-Go
A small RSS metadata browser called "gator" where you can follow your favorite RSS feeds and automatically ingest metadata about new posts

## Requirements
- Postgres >= v14
- Go > v1.23

## Installation
with Go installed: 
```
go get github.com/hudsn/rss_go
```

## Usage 

### Config 

This RSS browser uses a configuration file that you'll store in your home directory (`~/.gatorconfig`). It takes one initial configuration line `db_url` which is the data source url for a new postgres database that gator can use.

An example initial config file: 
```json
{
	"db_url": "postgres://my_postgres_user:my_postgres_password@localhost:5432/gator"
}
```

### Commands
- You can start by registering a new user with `rss_go register my_cool_username`

- You check a list of existing users with `rss_go users`

- You can switch to any existing user with `rrs_go login my_existing_username`

- You can add an rss feed to the program with `rss_go addfeed my_feed_name my_feed_url` 

- You can list all the existing feeds with `rss_go feeds`

- You can start gathering the most recent posts for all added feeds with `rss_go agg checkin_duration` where "checkin_duration" is a time expression for how often to fetch new posts. Ex: `rss_go agg 5s` would fetch new post every 5 seconds, and `rss_go agg 10m` would fetch new posts every 10 minutes.

- You can follow an existing feed with `rss_go follow cool_feed_url`

- You can list all the feeds you're following with `rss_go following`


- You can list all the most recent posts of all feeds you're following with `rss_go browse limit` where "limit" is the number of recent posts to get. Ex: `rss_go browse 10`

- You can reset the data in your database at any time with `rss_go reset`