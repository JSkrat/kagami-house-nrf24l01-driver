// Redis translates components to redis database
// we're using database 0, key is stringified UID+FNo, value is plain value for now, no json yet
package Redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"os"

	//"github.com/flynn/json5"
	"../OutsideInterface"
)

var log = logrus.New()

type Interface struct {
	db  *redis.Client
	ctx context.Context
}

func Init(self *Interface, address string) {
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.DebugLevel
	log.Out = os.Stdout
	self.db = redis.NewClient(&redis.Options{Addr: address})
	self.ctx = context.Background()
}

func (i *Interface) UpdateComponent(key string, value string) {
	i.db.Set(i.ctx, key, value, 0)
}

func (i *Interface) RegisterWritableComponent(key string) <-chan OutsideInterface.SubMessage {
	pubsub := i.db.Subscribe(i.ctx, "__keyspace@0__:"+key)
	// Wait for confirmation that subscription is created before publishing anything.
	_, err := pubsub.Receive(i.ctx)
	if err != nil {
		panic(err)
	}
	// Go channel which receives messages.
	ch := pubsub.Channel()
	ret := make(chan OutsideInterface.SubMessage)
	go func() {
		for message := range ch {
			log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s) goroutine: chan <%s>, payload <%s>, payload slice <%v>", key, message.Channel, message.Payload, message.PayloadSlice))
			if "set" == message.Payload {
				value, err := i.db.Get(i.ctx, key).Result()
				if nil != err {
					panic(err)
				}
				log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s) goroutine: value is <%s>", key, value))
				ret <- OutsideInterface.SubMessage{
					Value: value,
					Key:   key,
				}
			}
		}
	}()
	i.db.Set(i.ctx, key, "", 0)
	return ret
}