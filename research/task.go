package research

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func FixRedisEntries(ctx context.Context, rdb *redis.Client, log *logrus.Logger) {
	data := []struct {
		fileId string
		pageId string
	}{
		{"id:JTYKZ7Uf9NAAAAAAAAAEow", "4242f12ad03e4924b20afa3ea8d5c35f"},
		{"id:JTYKZ7Uf9NAAAAAAAAAFiQ", "7bf0686cfd0f406e98aa48aff387c0d5"},
		{"id:JTYKZ7Uf9NAAAAAAAAAEkg", "053b29d96af648768c1b18b546bfe029"},
		{"id:JTYKZ7Uf9NAAAAAAAAAEsw", "b23f239f2b9140bcad47a3bb9d9ed400"},
		{"id:JTYKZ7Uf9NAAAAAAAAAFYA", "94b5e47a95464cef933ab5d348fc8945"},
		{"id:JTYKZ7Uf9NAAAAAAAAAEpA", "0bfcaa88e8a4464bafc6fafb8afe0cb5"},
		{"id:JTYKZ7Uf9NAAAAAAAAAEpQ", "07fc0b40b89343ae8ce7afc131b2aa65"},
	}

	for _, entry := range data {
		c := CloudFile{
			FileID:   entry.fileId,
			Provider: "dropbox",
		}
		key := c.GetKey()
		err := rdb.Set(ctx, key, entry.pageId, 0).Err()
		if err != nil {
			log.WithField("entry", entry).Error(err)
		}
	}
}
