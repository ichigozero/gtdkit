package inmem

import (
	"errors"

	consul "github.com/hashicorp/consul/api"
)

type Client interface {
	Get(key string) error
	Put(key string, value []byte) error
	Delete(key string) error
}

type client struct {
	consul *consul.Client
}

func NewClient(c *consul.Client) Client {
	return &client{c}
}

func (c *client) Get(key string) error {
	kv, _, err := c.consul.KV().Get(key, nil)
	if err != nil {
		return err
	}

	if kv == nil {
		return ErrKeyNotFound
	}

	return nil
}

func (c *client) Put(key string, value []byte) error {
	p := &consul.KVPair{Key: key, Value: value}
	_, err := c.consul.KV().Put(p, nil)

	return err
}

func (c *client) Delete(key string) error {
	_, err := c.consul.KV().Delete(key, nil)

	return err
}

var ErrKeyNotFound = errors.New("key not found")
