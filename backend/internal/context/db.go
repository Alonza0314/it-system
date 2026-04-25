package context

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

type bboltDbContext struct {
	dbPath string
}

func newBboltDbContext(dbPath string) *bboltDbContext {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		panic(fmt.Sprintf("Failed to create DB directory: %v", err))
	}
	return &bboltDbContext{
		dbPath: dbPath,
	}
}

func (ctx *bboltDbContext) Save(bucket, key, value []byte) error {
	return dbSave(ctx.dbPath, bucket, key, value)
}

func (ctx *bboltDbContext) Load(bucket, key []byte) ([]byte, error) {
	return dbLoad(ctx.dbPath, bucket, key)
}

func (ctx *bboltDbContext) Update(bucket, key, value []byte) error {
	return dbUpdate(ctx.dbPath, bucket, key, value)
}

func (ctx *bboltDbContext) Remove(bucket, key []byte) error {
	return dbRemove(ctx.dbPath, bucket, key)
}

func dbSave(dbPath string, bucket, key, value []byte) error {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}

func dbLoad(dbPath string, bucket, key []byte) ([]byte, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var value []byte
	if err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get(key)
		value = make([]byte, len(v))
		copy(value, v)
		return nil
	}); err != nil {
		return nil, err
	}

	return value, nil
}

func dbUpdate(dbPath string, bucket, key, value []byte) error {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Put(key, value)
	})
}

func dbRemove(dbPath string, bucket, key []byte) error {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete(key)
	})
}
