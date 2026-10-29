# LogBriefEntry


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**category** | **string** |  | [default to undefined]
**message** | **string** |  | [default to undefined]
**line** | **number** | 1-based line of the first occurrence in the log | [default to undefined]
**count** | **number** |  | [default to undefined]
**source** | **string** | Container that printed the line, for lines from the compose log | [optional] [default to undefined]
**time** | **string** | Timestamp of the first occurrence, for lines printed by an NF | [optional] [default to undefined]
**items** | **Array&lt;string&gt;** | Values that differ between merged lines | [optional] [default to undefined]

## Example

```typescript
import { LogBriefEntry } from './api';

const instance: LogBriefEntry = {
    category,
    message,
    line,
    count,
    source,
    time,
    items,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
