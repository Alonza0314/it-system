package context_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

const (
	DB_DIR  = ".db_test"
	DB_PATH = ".db_test/test.db"
	BUCKET  = "test"
	BUCKET2 = "test2"
	SAVE    = "save"
	LOAD    = "load"
	UPDATE  = "update"
	REMOVE  = "remove"
)

var testDbCases = []struct {
	name          string
	method        string
	key           string
	value         string
	expectedValue string
}{
	{
		name:   "testDbSave",
		method: SAVE,
		key:    "testKey",
		value:  "testValue",
	},
	{
		name:          "testDbLoadAfterSave",
		method:        LOAD,
		key:           "testKey",
		expectedValue: "testValue",
	},
	{
		name:   "testDbUpdate",
		method: UPDATE,
		key:    "testKey",
		value:  "updatedValue",
	},
	{
		name:          "testDbLoadAfterUpdate",
		method:        LOAD,
		key:           "testKey",
		expectedValue: "updatedValue",
	},
	{
		name:   "testDbRemove",
		method: REMOVE,
		key:    "testKey",
	},
}

var testDbCasesWithMap = []struct {
	name   string
	method string
	key    string
	value  struct {
		Map map[string]string `json:"map"`
	}
	expectedValue struct {
		Map map[string]string `json:"map"`
	}
}{
	{
		name:   "testDbSave",
		method: SAVE,
		key:    "testKey",
		value: struct {
			Map map[string]string `json:"map"`
		}{
			Map: map[string]string{
				"Key1": "Value1",
				"Key2": "Value2",
			},
		},
	},
	{
		name:   "testDbLoadAfterSave",
		method: LOAD,
		key:    "testKey",
		expectedValue: struct {
			Map map[string]string `json:"map"`
		}{
			Map: map[string]string{
				"Key1": "Value1",
				"Key2": "Value2",
			},
		},
	},
}

var testDbLoadAllCases = []struct {
	name          string
	key           string
	value         string
	expectedValue map[string]string
}{
	{
		name:  "testDbLoadAll-1",
		key:   "testKey1",
		value: "testValue1",
		expectedValue: map[string]string{
			"testKey1": "testValue1",
		},
	},
	{
		name:  "testDbLoadAll-2",
		key:   "testKey2",
		value: "testValue2",
		expectedValue: map[string]string{
			"testKey1": "testValue1",
			"testKey2": "testValue2",
		},
	},
}

func TestDB(t *testing.T) {
	defer func() {
		if err := os.RemoveAll(DB_DIR); err != nil {
			t.Fatalf("Failed to remove DB directory: %v", err)
		}
	}()

	t.Run("TestDbOpreation", func(t *testing.T) {
		testDbOperation(t)
	})

	t.Run("TestDbOpreationWithMap", func(t *testing.T) {
		testDbOperationWithMap(t)
	})

	t.Run("TestDbLoadAll", func(t *testing.T) {
		testDbLoadAll(t)
	})
}

func testDbOperation(t *testing.T) {
	for _, testCase := range testDbCases {
		t.Run(testCase.name, func(t *testing.T) {
			switch testCase.method {
			case SAVE:
				err := ctx.SaveToDb(BUCKET, testCase.key, testCase.value)
				if err != nil {
					t.Errorf("Error saving value: %v", err)
				}
			case LOAD:
				value, err := ctx.LoadFromDb(BUCKET, testCase.key)
				if err != nil {
					t.Errorf("Error loading value: %v", err)
				}
				if value != testCase.expectedValue {
					t.Errorf("Expected value: %v, got: %v", testCase.expectedValue, value)
				}
			case UPDATE:
				err := ctx.UpdateDb(BUCKET, testCase.key, testCase.value)
				if err != nil {
					t.Errorf("Error updating value: %v", err)
				}
			case REMOVE:
				err := ctx.RemoveFromDb(BUCKET, testCase.key)
				if err != nil {
					t.Errorf("Error removing value: %v", err)
				}
			}
		})
	}
}

func testDbOperationWithMap(t *testing.T) {
	for _, testCase := range testDbCasesWithMap {
		t.Run(testCase.name, func(t *testing.T) {
			switch testCase.method {
			case SAVE:
				jsonValue, err := json.Marshal(testCase.value)
				if err != nil {
					t.Errorf("Error marshalling value: %v", err)
				}
				err = ctx.SaveToDb(BUCKET, testCase.key, string(jsonValue))
				if err != nil {
					t.Errorf("Error saving value: %v", err)
				}
			case LOAD:
				value, err := ctx.LoadFromDb(BUCKET, testCase.key)
				if err != nil {
					t.Errorf("Error loading value: %v", err)
				}
				var loadedValue struct {
					Map map[string]string `json:"map"`
				}
				err = json.Unmarshal([]byte(value), &loadedValue)
				if err != nil {
					t.Errorf("Error unmarshalling value: %v", err)
				}
				if !reflect.DeepEqual(loadedValue.Map, testCase.expectedValue.Map) {
					t.Errorf("Expected value: %v, got: %v", testCase.expectedValue.Map, loadedValue.Map)
				}
			}
		})
	}
}

func testDbLoadAll(t *testing.T) {
	for _, testCase := range testDbLoadAllCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ctx.SaveToDb(BUCKET2, testCase.key, testCase.value)
			if err != nil {
				t.Errorf("Error saving value: %v", err)
			}

			result, err := ctx.LoadAllFromDb(BUCKET2)
			if err != nil {
				t.Errorf("Error loading all values: %v", err)
			}

			if !reflect.DeepEqual(result, testCase.expectedValue) {
				t.Errorf("Expected value: %v, got: %v", testCase.expectedValue, result)
			}
		})
	}
}
