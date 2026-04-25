package context

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

type bboltDbContext struct {
	db *bolt.DB
}

func newBboltDbContext(dbPath string) *bboltDbContext {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		panic(fmt.Sprintf("Failed to create DB directory: %v", err))
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(fmt.Sprintf("Failed to open DB: %v", err))
	}

	return &bboltDbContext{
		db: db,
	}
}

func releaseBboltDbContext(ctx *bboltDbContext) error {
	if ctx.db != nil {
		return ctx.db.Close()
	}

	return nil
}

func (ctx *bboltDbContext) Save(bucket, key, value []byte) error {
	return dbSave(ctx.db, bucket, key, value)
}

func (ctx *bboltDbContext) Load(bucket, key []byte) ([]byte, error) {
	return dbLoad(ctx.db, bucket, key)
}

func (ctx *bboltDbContext) LoadAll(bucket []byte) (map[string][]byte, error) {
	return dbLoadAll(ctx.db, bucket)
}

func (ctx *bboltDbContext) Update(bucket, key, value []byte) error {
	return dbUpdate(ctx.db, bucket, key, value)
}

func (ctx *bboltDbContext) Remove(bucket, key []byte) error {
	return dbRemove(ctx.db, bucket, key)
}

func (ctx *bboltDbContext) Exists(bucket, key []byte) (bool, error) {
	return dbExists(ctx.db, bucket, key)
}

func dbSave(db *bolt.DB, bucket, key, value []byte) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})

	return err
}

func dbLoad(db *bolt.DB, bucket, key []byte) (value []byte, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get(key)
		value = make([]byte, len(v))
		copy(value, v)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return value, nil
}

func dbLoadAll(db *bolt.DB, bucket []byte) (result map[string][]byte, err error) {
	result = make(map[string][]byte)

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			value := make([]byte, len(v))
			copy(value, v)
			result[string(k)] = value
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func dbUpdate(db *bolt.DB, bucket, key, value []byte) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Put(key, value)
	})

	return err
}

func dbRemove(db *bolt.DB, bucket, key []byte) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete(key)
	})

	return err
}

func dbExists(db *bolt.DB, bucket, key []byte) (exists bool, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			exists = false
			return nil
		}
		v := b.Get(key)
		exists = v != nil
		return nil
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}
