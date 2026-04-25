package context

import (
	"errors"
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

func (ctx *bboltDbContext) LoadAll(bucket []byte) (map[string][]byte, error) {
	return dbLoadAll(ctx.dbPath, bucket)
}

func (ctx *bboltDbContext) Update(bucket, key, value []byte) error {
	return dbUpdate(ctx.dbPath, bucket, key, value)
}

func (ctx *bboltDbContext) Remove(bucket, key []byte) error {
	return dbRemove(ctx.dbPath, bucket, key)
}

func closeDBWithErr(db *bolt.DB, errPtr *error) {
	if closeErr := db.Close(); closeErr != nil {
		wrappedCloseErr := fmt.Errorf("failed to close db: %w", closeErr)
		if *errPtr != nil {
			*errPtr = errors.Join(*errPtr, wrappedCloseErr)
			return
		}
		*errPtr = wrappedCloseErr
	}
}

func dbSave(dbPath string, bucket, key, value []byte) (err error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer closeDBWithErr(db, &err)

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})

	return err
}

func dbLoad(dbPath string, bucket, key []byte) (value []byte, err error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	defer closeDBWithErr(db, &err)

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

func dbLoadAll(dbPath string, bucket []byte) (result map[string][]byte, err error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	defer closeDBWithErr(db, &err)

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

func dbUpdate(dbPath string, bucket, key, value []byte) (err error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer closeDBWithErr(db, &err)

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Put(key, value)
	})

	return err
}

func dbRemove(dbPath string, bucket, key []byte) (err error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	defer closeDBWithErr(db, &err)

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete(key)
	})

	return err
}
