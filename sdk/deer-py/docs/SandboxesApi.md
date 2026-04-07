# fluid.SandboxesApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**orgs_slug_sandboxes_get**](SandboxesApi.md#orgs_slug_sandboxes_get) | **GET** /orgs/{slug}/sandboxes | List sandboxes
[**orgs_slug_sandboxes_post**](SandboxesApi.md#orgs_slug_sandboxes_post) | **POST** /orgs/{slug}/sandboxes | Create sandbox
[**orgs_slug_sandboxes_sandbox_id_commands_get**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_commands_get) | **GET** /orgs/{slug}/sandboxes/{sandboxID}/commands | List commands
[**orgs_slug_sandboxes_sandbox_id_delete**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_delete) | **DELETE** /orgs/{slug}/sandboxes/{sandboxID} | Destroy sandbox
[**orgs_slug_sandboxes_sandbox_id_get**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_get) | **GET** /orgs/{slug}/sandboxes/{sandboxID} | Get sandbox
[**orgs_slug_sandboxes_sandbox_id_run_post**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_run_post) | **POST** /orgs/{slug}/sandboxes/{sandboxID}/run | Run command
[**orgs_slug_sandboxes_sandbox_id_snapshot_post**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_snapshot_post) | **POST** /orgs/{slug}/sandboxes/{sandboxID}/snapshot | Create snapshot
[**orgs_slug_sandboxes_sandbox_id_start_post**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_start_post) | **POST** /orgs/{slug}/sandboxes/{sandboxID}/start | Start sandbox
[**orgs_slug_sandboxes_sandbox_id_stop_post**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_id_stop_post) | **POST** /orgs/{slug}/sandboxes/{sandboxID}/stop | Stop sandbox
[**orgs_slug_sandboxes_sandbox_idip_get**](SandboxesApi.md#orgs_slug_sandboxes_sandbox_idip_get) | **GET** /orgs/{slug}/sandboxes/{sandboxID}/ip | Get sandbox IP


# **orgs_slug_sandboxes_get**
> Dict[str, object] orgs_slug_sandboxes_get(slug)

List sandboxes

List all sandboxes in the organization

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # List sandboxes
        api_response = api_instance.orgs_slug_sandboxes_get(slug)
        print("The response of SandboxesApi->orgs_slug_sandboxes_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_post**
> StoreSandbox orgs_slug_sandboxes_post(slug, request)

Create sandbox

Create a new sandbox in the organization from a source VM or base image

### Example


```python
import fluid
from fluid.models.orchestrator_create_sandbox_request import OrchestratorCreateSandboxRequest
from fluid.models.store_sandbox import StoreSandbox
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    request = fluid.OrchestratorCreateSandboxRequest() # OrchestratorCreateSandboxRequest | Sandbox configuration

    try:
        # Create sandbox
        api_response = api_instance.orgs_slug_sandboxes_post(slug, request)
        print("The response of SandboxesApi->orgs_slug_sandboxes_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **request** | [**OrchestratorCreateSandboxRequest**](OrchestratorCreateSandboxRequest.md)| Sandbox configuration | 

### Return type

[**StoreSandbox**](StoreSandbox.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Created |  -  |
**400** | Bad Request |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_commands_get**
> Dict[str, object] orgs_slug_sandboxes_sandbox_id_commands_get(slug, sandbox_id)

List commands

List all commands executed in a sandbox

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID

    try:
        # List commands
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_commands_get(slug, sandbox_id)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_commands_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_commands_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_delete**
> Dict[str, object] orgs_slug_sandboxes_sandbox_id_delete(slug, sandbox_id)

Destroy sandbox

Destroy a sandbox and release its resources

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID

    try:
        # Destroy sandbox
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_delete(slug, sandbox_id)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_delete:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_delete: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_get**
> StoreSandbox orgs_slug_sandboxes_sandbox_id_get(slug, sandbox_id)

Get sandbox

Get sandbox details by ID

### Example


```python
import fluid
from fluid.models.store_sandbox import StoreSandbox
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID

    try:
        # Get sandbox
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_get(slug, sandbox_id)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 

### Return type

[**StoreSandbox**](StoreSandbox.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_run_post**
> StoreCommand orgs_slug_sandboxes_sandbox_id_run_post(slug, sandbox_id, request)

Run command

Execute a command in a sandbox

### Example


```python
import fluid
from fluid.models.orchestrator_run_command_request import OrchestratorRunCommandRequest
from fluid.models.store_command import StoreCommand
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID
    request = fluid.OrchestratorRunCommandRequest() # OrchestratorRunCommandRequest | Command to run

    try:
        # Run command
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_run_post(slug, sandbox_id, request)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_run_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_run_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 
 **request** | [**OrchestratorRunCommandRequest**](OrchestratorRunCommandRequest.md)| Command to run | 

### Return type

[**StoreCommand**](StoreCommand.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_snapshot_post**
> OrchestratorSnapshotResponse orgs_slug_sandboxes_sandbox_id_snapshot_post(slug, sandbox_id, request)

Create snapshot

Create a snapshot of a sandbox

### Example


```python
import fluid
from fluid.models.orchestrator_snapshot_request import OrchestratorSnapshotRequest
from fluid.models.orchestrator_snapshot_response import OrchestratorSnapshotResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID
    request = fluid.OrchestratorSnapshotRequest() # OrchestratorSnapshotRequest | Snapshot details

    try:
        # Create snapshot
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_snapshot_post(slug, sandbox_id, request)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_snapshot_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_snapshot_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 
 **request** | [**OrchestratorSnapshotRequest**](OrchestratorSnapshotRequest.md)| Snapshot details | 

### Return type

[**OrchestratorSnapshotResponse**](OrchestratorSnapshotResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Created |  -  |
**400** | Bad Request |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_start_post**
> Dict[str, object] orgs_slug_sandboxes_sandbox_id_start_post(slug, sandbox_id)

Start sandbox

Start a stopped sandbox

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID

    try:
        # Start sandbox
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_start_post(slug, sandbox_id)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_start_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_start_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_id_stop_post**
> Dict[str, object] orgs_slug_sandboxes_sandbox_id_stop_post(slug, sandbox_id)

Stop sandbox

Stop a running sandbox

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID

    try:
        # Stop sandbox
        api_response = api_instance.orgs_slug_sandboxes_sandbox_id_stop_post(slug, sandbox_id)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_id_stop_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_id_stop_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_sandboxes_sandbox_idip_get**
> Dict[str, object] orgs_slug_sandboxes_sandbox_idip_get(slug, sandbox_id)

Get sandbox IP

Get the IP address of a sandbox

### Example


```python
import fluid
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8081/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost:8081/v1"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.SandboxesApi(api_client)
    slug = 'slug_example' # str | Organization slug
    sandbox_id = 'sandbox_id_example' # str | Sandbox ID

    try:
        # Get sandbox IP
        api_response = api_instance.orgs_slug_sandboxes_sandbox_idip_get(slug, sandbox_id)
        print("The response of SandboxesApi->orgs_slug_sandboxes_sandbox_idip_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxesApi->orgs_slug_sandboxes_sandbox_idip_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **sandbox_id** | **str**| Sandbox ID | 

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**403** | Forbidden |  -  |
**404** | Not Found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

