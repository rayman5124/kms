package cache

import (
	"crypto/ecdsa"
	"sync"
)

type PubKeyCache struct {
	pubKeys map[string]*ecdsa.PublicKey
	mutex   sync.RWMutex
}

func NewPubKeyCache() *PubKeyCache {
	return &PubKeyCache{
		pubKeys: make(map[string]*ecdsa.PublicKey),
	}
}

func (c *PubKeyCache) Add(keyID string, pubKey *ecdsa.PublicKey) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.pubKeys[keyID] = pubKey
}

func (c *PubKeyCache) Get(keyID string) *ecdsa.PublicKey {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.pubKeys[keyID]
}
