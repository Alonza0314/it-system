# DefaultApi

All URIs are relative to *http://127.0.0.1:5000*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**addTenants**](#addtenants) | **POST** /api/admin/tenant | Add tenants|
|[**addTestcases**](#addtestcases) | **POST** /api/admin/test/testcase | Add testcases|
|[**cancelTask**](#canceltask) | **DELETE** /api/test/task | Cancel task|
|[**deleteRunner**](#deleterunner) | **DELETE** /api/run/runner/test-output | Delete runner|
|[**deleteTenants**](#deletetenants) | **DELETE** /api/admin/tenant | Delete tenants|
|[**deleteTestcases**](#deletetestcases) | **DELETE** /api/admin/test/testcase | Delete testcases|
|[**getGithubPRs**](#getgithubprs) | **GET** /api/github | Get Github PRs|
|[**getRunners**](#getrunners) | **GET** /api/runner | Get runners|
|[**getTask**](#gettask) | **GET** /api/test/task | Get task|
|[**getTasks**](#gettasks) | **GET** /api/test/tasks | Get tasks|
|[**getTenants**](#gettenants) | **GET** /api/admin/tenant | Get tenants|
|[**getTestcases**](#gettestcases) | **GET** /api/test/testcase | Get testcases|
|[**login**](#login) | **POST** /api/login | Login|
|[**logout**](#logout) | **POST** /api/logout | Logout|
|[**registerRunner**](#registerrunner) | **POST** /api/admin/runner | Register runner|
|[**runnerHeartbeat**](#runnerheartbeat) | **POST** /api/run/runner/heartbeat | Runner heartbeat|
|[**submitTask**](#submittask) | **POST** /api/test/task | Submit task|
|[**testOutput**](#testoutput) | **POST** /api/run/runner/test-output | Test output|

# **addTenants**
> MessageResponse addTenants(addTenantsRequest)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    AddTenantsRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let addTenantsRequest: AddTenantsRequest; //

const { status, data } = await apiInstance.addTenants(
    addTenantsRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **addTenantsRequest** | **AddTenantsRequest**|  | |


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**409** | Conflict |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **addTestcases**
> MessageResponse addTestcases(addTestcasesRequest)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    AddTestcasesRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let addTestcasesRequest: AddTestcasesRequest; //

const { status, data } = await apiInstance.addTestcases(
    addTestcasesRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **addTestcasesRequest** | **AddTestcasesRequest**|  | |


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**409** | Conflict |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **cancelTask**
> MessageResponse cancelTask()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let id: number; //Task ID (default to undefined)

const { status, data } = await apiInstance.cancelTask(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**number**] | Task ID | defaults to undefined|


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteRunner**
> MessageResponse deleteRunner()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let name: string; //Runner name (default to undefined)

const { status, data } = await apiInstance.deleteRunner(
    name
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | Runner name | defaults to undefined|


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**404** | Not Found |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteTenants**
> MessageResponse deleteTenants(deleteTenantsRequest)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    DeleteTenantsRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let deleteTenantsRequest: DeleteTenantsRequest; //

const { status, data } = await apiInstance.deleteTenants(
    deleteTenantsRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **deleteTenantsRequest** | **DeleteTenantsRequest**|  | |


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**404** | Not Found |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deleteTestcases**
> MessageResponse deleteTestcases(deleteTestcasesRequest)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    DeleteTestcasesRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let deleteTestcasesRequest: DeleteTestcasesRequest; //

const { status, data } = await apiInstance.deleteTestcases(
    deleteTestcasesRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **deleteTestcasesRequest** | **DeleteTestcasesRequest**|  | |


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**404** | Not Found |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getGithubPRs**
> GetGithubPRsResponse getGithubPRs()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let nf: 'amf' | 'ausf' | 'bsf' | 'chf' | 'n3iwf' | 'nef' | 'nrf' | 'nssf' | 'pcf' | 'smf' | 'tngf' | 'udm' | 'udr' | 'upf'; //Target network function (default to undefined)

const { status, data } = await apiInstance.getGithubPRs(
    nf
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **nf** | [**&#39;amf&#39; | &#39;ausf&#39; | &#39;bsf&#39; | &#39;chf&#39; | &#39;n3iwf&#39; | &#39;nef&#39; | &#39;nrf&#39; | &#39;nssf&#39; | &#39;pcf&#39; | &#39;smf&#39; | &#39;tngf&#39; | &#39;udm&#39; | &#39;udr&#39; | &#39;upf&#39;**]**Array<&#39;amf&#39; &#124; &#39;ausf&#39; &#124; &#39;bsf&#39; &#124; &#39;chf&#39; &#124; &#39;n3iwf&#39; &#124; &#39;nef&#39; &#124; &#39;nrf&#39; &#124; &#39;nssf&#39; &#124; &#39;pcf&#39; &#124; &#39;smf&#39; &#124; &#39;tngf&#39; &#124; &#39;udm&#39; &#124; &#39;udr&#39; &#124; &#39;upf&#39;>** | Target network function | defaults to undefined|


### Return type

**GetGithubPRsResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getRunners**
> GetRunnersResponse getRunners()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getRunners();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetRunnersResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTask**
> ResponseGetTask getTask()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let id: number; //Task ID (default to undefined)

const { status, data } = await apiInstance.getTask(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**number**] | Task ID | defaults to undefined|


### Return type

**ResponseGetTask**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTasks**
> ResponseGetTasks getTasks()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getTasks();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**ResponseGetTasks**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTenants**
> GetTenantsResponse getTenants()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getTenants();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetTenantsResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getTestcases**
> GetTestcasesResponse getTestcases()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getTestcases();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetTestcasesResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **login**
> LoginResponse login(loginRequest)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    LoginRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let loginRequest: LoginRequest; //

const { status, data } = await apiInstance.login(
    loginRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **loginRequest** | **LoginRequest**|  | |


### Return type

**LoginResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **logout**
> logout()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.logout();
```

### Parameters
This endpoint does not have any parameters.


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | No Content |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **registerRunner**
> ResponseRegisterRunner registerRunner(registerRunnerRequest)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RegisterRunnerRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let registerRunnerRequest: RegisterRunnerRequest; //

const { status, data } = await apiInstance.registerRunner(
    registerRunnerRequest
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **registerRunnerRequest** | **RegisterRunnerRequest**|  | |


### Return type

**ResponseRegisterRunner**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**409** | Conflict |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **runnerHeartbeat**
> ResponseRunnerHeartbeat runnerHeartbeat(requestRunnerHeartbeat)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RequestRunnerHeartbeat
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let requestRunnerHeartbeat: RequestRunnerHeartbeat; //

const { status, data } = await apiInstance.runnerHeartbeat(
    requestRunnerHeartbeat
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **requestRunnerHeartbeat** | **RequestRunnerHeartbeat**|  | |


### Return type

**ResponseRunnerHeartbeat**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**204** | No Content |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **submitTask**
> MessageResponse submitTask(requestSubmitTask)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RequestSubmitTask
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let requestSubmitTask: RequestSubmitTask; //

const { status, data } = await apiInstance.submitTask(
    requestSubmitTask
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **requestSubmitTask** | **RequestSubmitTask**|  | |


### Return type

**MessageResponse**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | OK |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **testOutput**
> testOutput(requestTestOutput)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RequestTestOutput
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let requestTestOutput: RequestTestOutput; //

const { status, data } = await apiInstance.testOutput(
    requestTestOutput
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **requestTestOutput** | **RequestTestOutput**|  | |


### Return type

void (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | No Content |  -  |
|**400** | Bad Request |  -  |
|**401** | Unauthorized |  -  |
|**403** | Forbidden |  -  |
|**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

