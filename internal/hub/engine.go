package hub

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Engine struct {
	db *leveldb.DB
}

func NewEngine(path string) (*Engine, error) {
	db, err := leveldb.OpenFile(path, &opt.Options{
		WriteBuffer: 4 * opt.MiB,
	})
	if err != nil {
		return nil, fmt.Errorf("open leveldb: %w", err)
	}
	return &Engine{db: db}, nil
}

func (e *Engine) Close() error {
	return e.db.Close()
}

func (e *Engine) Put(namespace, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s:%s: %w", namespace, key, err)
	}
	return e.db.Put([]byte(namespace+":"+key), data, nil)
}

func (e *Engine) BatchPut(namespace string, entries map[string]any) error {
	batch := new(leveldb.Batch)
	for key, value := range entries {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal %s:%s: %w", namespace, key, err)
		}
		batch.Put([]byte(namespace+":"+key), data)
	}
	return e.db.Write(batch, nil)
}

func (e *Engine) BatchDelete(namespace string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	batch := new(leveldb.Batch)
	for _, key := range keys {
		batch.Delete([]byte(namespace + ":" + key))
	}
	return e.db.Write(batch, nil)
}

var ErrNotFound = fmt.Errorf("not found")

func (e *Engine) Get(namespace, key string, dest any) error {
	data, err := e.db.Get([]byte(namespace+":"+key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("get %s:%s: %w", namespace, key, err)
	}
	return json.Unmarshal(data, dest)
}

func (e *Engine) Delete(namespace, key string) error {
	return e.db.Delete([]byte(namespace+":"+key), nil)
}

func (e *Engine) Has(namespace, key string) bool {
	exists, err := e.db.Has([]byte(namespace+":"+key), nil)
	return err == nil && exists
}

func (e *Engine) Iterate(namespace string, fn func(key string, value []byte) error) error {
	iter := e.db.NewIterator(util.BytesPrefix([]byte(namespace+":")), nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())[len(namespace)+1:]
		if err := fn(key, iter.Value()); err != nil {
			return err
		}
	}
	return iter.Error()
}

func (e *Engine) IterateJSON(namespace string, dest any, fn func(key string, value any) error) error {
	return e.Iterate(namespace, func(key string, value []byte) error {
		v := dest
		if err := json.Unmarshal(value, v); err != nil {
			return fmt.Errorf("unmarshal %s:%s: %w", namespace, key, err)
		}
		return fn(key, v)
	})
}

func (e *Engine) Count(namespace string) (int, error) {
	count := 0
	err := e.Iterate(namespace, func(key string, value []byte) error {
		count++
		return nil
	})
	return count, err
}

func (e *Engine) PutWithTTL(namespace, key string, value any, ttl time.Duration) error {
	wrapped := map[string]any{
		"data":       value,
		"expires_at": time.Now().Add(ttl).Unix(),
		"created_at": time.Now().Unix(),
	}
	return e.Put(namespace, key, wrapped)
}

func (e *Engine) GetWithTTL(namespace, key string, dest any) error {
	var wrapped map[string]any
	if err := e.Get(namespace, key, &wrapped); err != nil {
		return err
	}

	expiresAt, ok := wrapped["expires_at"].(float64)
	if ok && int64(expiresAt) < time.Now().Unix() {
		e.Delete(namespace, key)
		return fmt.Errorf("%s:%s expired", namespace, key)
	}

	data, err := json.Marshal(wrapped["data"])
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (e *Engine) PruneExpired(namespace string) (int, error) {
	pruned := 0
	var toDelete []string

	err := e.Iterate(namespace, func(key string, value []byte) error {
		var wrapped map[string]any
		if err := json.Unmarshal(value, &wrapped); err != nil {
			return nil
		}

		expiresAt, ok := wrapped["expires_at"].(float64)
		if ok && int64(expiresAt) < time.Now().Unix() {
			toDelete = append(toDelete, key)
		}
		return nil
	})

	if len(toDelete) > 0 {
		if batchErr := e.BatchDelete(namespace, toDelete); batchErr != nil {
			return pruned, batchErr
		}
		pruned = len(toDelete)
	}

	return pruned, err
}
