# fluid.SourceVMsApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**orgs_slug_sources_vm_prepare_post**](SourceVMsApi.md#orgs_slug_sources_vm_prepare_post) | **POST** /orgs/{slug}/sources/{vm}/prepare | Prepare source VM
[**orgs_slug_sources_vm_read_post**](SourceVMsApi.md#orgs_slug_sources_vm_read_post) | **POST** /orgs/{slug}/sources/{vm}/read | Read source file
[**orgs_slug_sources_vm_run_post**](SourceVMsApi.md#orgs_slug_sources_vm_run_post) | **POST** /orgs/{slug}/sources/{vm}/run | Run source command
[**orgs_slug_vms_get**](SourceVMsApi.md#orgs_slug_vms_get) | **GET** /orgs/{slug}/vms | List source VMs


# **orgs_slug_sources_vm_prepare_post**
> Dict[str, object] orgs_slug_sources_vm_prepare_post(slug, vm, request)

Prepare source VM

Prepare a source VM for sandbox cloning

### Example


```python
import fluid
from fluid.models.orchestrator_prepare_request import OrchestratorPrepareRequest
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
    api_instance = fluid.SourceVMsApi(api_client)
    slug = 'slug_example' # str | Organization slug
    vm = 'vm_example' # str | Source VM name
    request = fluid.OrchestratorPrepareRequest() # OrchestratorPrepareRequest | SSH credentials

    try:
        # Prepare source VM
        api_response = api_instance.orgs_slug_sources_vm_prepare_post(slug, vm, request)
        print("The response of SourceVMsApi->orgs_slug_sources_vm_prepare_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SourceVMsApi->orgs_slug_sources_vm_prepare_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **vm** | **str**| Source VM name | 
 **request** | [**OrchestratorPrepareRequest**](OrchestratorPrepareRequest.md)| SSH credentials | 

### Return type

**Dict[str, object]**

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

# **orgs_slug_sources_vm_read_post**
> OrchestratorSourceFileResult orgs_slug_sources_vm_read_post(slug, vm, request)

Read source file

Read a file from a source VM

### Example


```python
import fluid
from fluid.models.orchestrator_read_source_request import OrchestratorReadSourceRequest
from fluid.models.orchestrator_source_file_result import OrchestratorSourceFileResult
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
    api_instance = fluid.SourceVMsApi(api_client)
    slug = 'slug_example' # str | Organization slug
    vm = 'vm_example' # str | Source VM name
    request = fluid.OrchestratorReadSourceRequest() # OrchestratorReadSourceRequest | File path

    try:
        # Read source file
        api_response = api_instance.orgs_slug_sources_vm_read_post(slug, vm, request)
        print("The response of SourceVMsApi->orgs_slug_sources_vm_read_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SourceVMsApi->orgs_slug_sources_vm_read_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **vm** | **str**| Source VM name | 
 **request** | [**OrchestratorReadSourceRequest**](OrchestratorReadSourceRequest.md)| File path | 

### Return type

[**OrchestratorSourceFileResult**](OrchestratorSourceFileResult.md)

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

# **orgs_slug_sources_vm_run_post**
> OrchestratorSourceCommandResult orgs_slug_sources_vm_run_post(slug, vm, request)

Run source command

Execute a read-only command on a source VM

### Example


```python
import fluid
from fluid.models.orchestrator_run_source_request import OrchestratorRunSourceRequest
from fluid.models.orchestrator_source_command_result import OrchestratorSourceCommandResult
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
    api_instance = fluid.SourceVMsApi(api_client)
    slug = 'slug_example' # str | Organization slug
    vm = 'vm_example' # str | Source VM name
    request = fluid.OrchestratorRunSourceRequest() # OrchestratorRunSourceRequest | Command to run

    try:
        # Run source command
        api_response = api_instance.orgs_slug_sources_vm_run_post(slug, vm, request)
        print("The response of SourceVMsApi->orgs_slug_sources_vm_run_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SourceVMsApi->orgs_slug_sources_vm_run_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **vm** | **str**| Source VM name | 
 **request** | [**OrchestratorRunSourceRequest**](OrchestratorRunSourceRequest.md)| Command to run | 

### Return type

[**OrchestratorSourceCommandResult**](OrchestratorSourceCommandResult.md)

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

# **orgs_slug_vms_get**
> Dict[str, object] orgs_slug_vms_get(slug)

List source VMs

List all source VMs across connected hosts

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
    api_instance = fluid.SourceVMsApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # List source VMs
        api_response = api_instance.orgs_slug_vms_get(slug)
        print("The response of SourceVMsApi->orgs_slug_vms_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SourceVMsApi->orgs_slug_vms_get: %s\n" % e)
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

