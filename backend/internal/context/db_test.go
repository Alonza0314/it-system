package context_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

const (
	DB_DIR  = ".db"
	DB_PATH = ".db/test.db"
	BUCKET  = "test"
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

func TestDbOperation(t *testing.T) {
	os.Mkdir(DB_DIR, 0755)
	defer os.RemoveAll(DB_DIR)

	for _, testCase := range testDbCases {
		t.Run(testCase.name, func(t *testing.T) {
			switch testCase.method {
			case SAVE:
				err := ctx.SaveToDb([]byte(BUCKET), []byte(testCase.key), []byte(testCase.value))
				if err != nil {
					t.Errorf("Error saving value: %v", err)
				}
			case LOAD:
				value, err := ctx.LoadFromDb([]byte(BUCKET), []byte(testCase.key))
				if err != nil {
					t.Errorf("Error loading value: %v", err)
				}
				if string(value) != testCase.expectedValue {
					t.Errorf("Expected value: %v, got: %v", testCase.expectedValue, string(value))
				}
			case UPDATE:
				err := ctx.UpdateDb([]byte(BUCKET), []byte(testCase.key), []byte(testCase.value))
				if err != nil {
					t.Errorf("Error updating value: %v", err)
				}
			case REMOVE:
				err := ctx.RemoveFromDb([]byte(BUCKET), []byte(testCase.key))
				if err != nil {
					t.Errorf("Error removing value: %v", err)
				}
			}
		})
	}
}

func TestDbOperationWithMap(t *testing.T) {
	os.Mkdir(DB_DIR, 0755)
	defer os.RemoveAll(DB_DIR)

	for _, testCase := range testDbCasesWithMap {
		t.Run(testCase.name, func(t *testing.T) {
			switch testCase.method {
			case SAVE:
				jsonValue, err := json.Marshal(testCase.value)
				if err != nil {
					t.Errorf("Error marshalling value: %v", err)
				}
				err = ctx.SaveToDb([]byte(BUCKET), []byte(testCase.key), jsonValue)
				if err != nil {
					t.Errorf("Error saving value: %v", err)
				}
			case LOAD:
				value, err := ctx.LoadFromDb([]byte(BUCKET), []byte(testCase.key))
				if err != nil {
					t.Errorf("Error loading value: %v", err)
				}
				var loadedValue struct {
					Map map[string]string `json:"map"`
				}
				err = json.Unmarshal(value, &loadedValue)
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
