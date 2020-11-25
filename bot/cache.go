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

func (c *Cache) Connect() {
	url, _ := url2.Parse(os.Getenv("REDISCLOUD_URL"))
	pw, _ := url.User.Password()
	c.client = redis.NewClient(&redis.Options{
		Addr:     url.Hostname() + ":" + url.Port(),
		Password: pw,
	})
}

func (c *Cache) GetOwnersFileForRepo(repoURI string) (string, error) {
	return c.client.Get(repoURI).Result()
}

func (c *Cache) SetOwnersFileForRepo(repoURI, content string) (string, error) {
	return c.client.Set(repoURI, content, 30*time.Minute).Result()
}
