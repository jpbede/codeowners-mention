package bot

import (
	"github.com/go-redis/redis"
	url2 "net/url"
	"os"
	"time"
)

type Cache struct {
	client *redis.Client
}

func (c *Cache) Connect() error {
	url, parseErr := url2.Parse(os.Getenv("REDISCLOUD_URL"))
	if parseErr != nil {
		return parseErr
	}

	opts := &redis.Options{
		Addr: url.Hostname() + ":" + url.Port(),
	}
	if pw, set := url.User.Password(); set {
		opts.Password = pw
	}

	c.client = redis.NewClient(opts)

	return nil
}

func (c *Cache) GetOwnersFileForRepo(repoURI string) (string, error) {
	return c.client.Get(repoURI).Result()
}

func (c *Cache) SetOwnersFileForRepo(repoURI, content string) (string, error) {
	return c.client.Set(repoURI, content, 30*time.Minute).Result()
}
