# fluid.HostsApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**orgs_slug_hosts_get**](HostsApi.md#orgs_slug_hosts_get) | **GET** /orgs/{slug}/hosts | List hosts
[**orgs_slug_hosts_host_id_get**](HostsApi.md#orgs_slug_hosts_host_id_get) | **GET** /orgs/{slug}/hosts/{hostID} | Get host


# **orgs_slug_hosts_get**
> Dict[str, object] orgs_slug_hosts_get(slug)

List hosts

List all connected sandbox hosts

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
    api_instance = fluid.HostsApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # List hosts
        api_response = api_instance.orgs_slug_hosts_get(slug)
        print("The response of HostsApi->orgs_slug_hosts_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling HostsApi->orgs_slug_hosts_get: %s\n" % e)
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

# **orgs_slug_hosts_host_id_get**
> OrchestratorHostInfo orgs_slug_hosts_host_id_get(slug, host_id)

Get host

Get details of a specific connected host

### Example


```python
import fluid
from fluid.models.orchestrator_host_info import OrchestratorHostInfo
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
    api_instance = fluid.HostsApi(api_client)
    slug = 'slug_example' # str | Organization slug
    host_id = 'host_id_example' # str | Host ID

    try:
        # Get host
        api_response = api_instance.orgs_slug_hosts_host_id_get(slug, host_id)
        print("The response of HostsApi->orgs_slug_hosts_host_id_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling HostsApi->orgs_slug_hosts_host_id_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **host_id** | **str**| Host ID | 

### Return type

[**OrchestratorHostInfo**](OrchestratorHostInfo.md)

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

