# fluid.VMsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_virtual_machines**](VMsApi.md#list_virtual_machines) | **GET** /v1/vms | List all host VMs


# **list_virtual_machines**
> GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse list_virtual_machines()

List all host VMs

Returns a list of host virtual machines from libvirt (excludes sandboxes). When multi-host is configured, aggregates VMs from all hosts.

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response import GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse
from fluid.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost
# See configuration.py for a list of all supported configuration parameters.
configuration = fluid.Configuration(
    host = "http://localhost"
)


# Enter a context with an instance of the API client
with fluid.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = fluid.VMsApi(api_client)

    try:
        # List all host VMs
        api_response = api_instance.list_virtual_machines()
        print("The response of VMsApi->list_virtual_machines:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling VMsApi->list_virtual_machines: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

