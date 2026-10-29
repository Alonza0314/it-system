# TaskDetail


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **number** |  | [default to undefined]
**username** | **string** |  | [default to undefined]
**createTime** | **number** |  | [default to undefined]
**status** | **string** |  | [default to undefined]
**tests** | [**Array&lt;TestDetail&gt;**](TestDetail.md) |  | [optional] [default to undefined]
**nfPrList** | [**Array&lt;NfPr&gt;**](NfPr.md) |  | [optional] [default to undefined]

## Example

```typescript
import { TaskDetail } from './api';

const instance: TaskDetail = {
    id,
    username,
    createTime,
    status,
    tests,
    nfPrList,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
