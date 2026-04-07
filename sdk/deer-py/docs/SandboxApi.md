# fluid.SandboxApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_sandbox**](SandboxApi.md#create_sandbox) | **POST** /v1/sandboxes | Create a new sandbox
[**create_snapshot**](SandboxApi.md#create_snapshot) | **POST** /v1/sandboxes/{id}/snapshot | Create snapshot
[**destroy_sandbox**](SandboxApi.md#destroy_sandbox) | **DELETE** /v1/sandboxes/{id} | Destroy sandbox
[**diff_snapshots**](SandboxApi.md#diff_snapshots) | **POST** /v1/sandboxes/{id}/diff | Diff snapshots
[**discover_sandbox_ip**](SandboxApi.md#discover_sandbox_ip) | **GET** /v1/sandboxes/{id}/ip | Discover sandbox IP
[**generate_configuration**](SandboxApi.md#generate_configuration) | **POST** /v1/sandboxes/{id}/generate/{tool} | Generate configuration
[**get_sandbox**](SandboxApi.md#get_sandbox) | **GET** /v1/sandboxes/{id} | Get sandbox details
[**inject_ssh_key**](SandboxApi.md#inject_ssh_key) | **POST** /v1/sandboxes/{id}/sshkey | Inject SSH key into sandbox
[**list_sandbox_commands**](SandboxApi.md#list_sandbox_commands) | **GET** /v1/sandboxes/{id}/commands | List sandbox commands
[**list_sandboxes**](SandboxApi.md#list_sandboxes) | **GET** /v1/sandboxes | List sandboxes
[**publish_changes**](SandboxApi.md#publish_changes) | **POST** /v1/sandboxes/{id}/publish | Publish changes
[**run_sandbox_command**](SandboxApi.md#run_sandbox_command) | **POST** /v1/sandboxes/{id}/run | Run command in sandbox
[**start_sandbox**](SandboxApi.md#start_sandbox) | **POST** /v1/sandboxes/{id}/start | Start sandbox
[**stream_sandbox_activity**](SandboxApi.md#stream_sandbox_activity) | **GET** /v1/sandboxes/{id}/stream | Stream sandbox activity


# **create_sandbox**
> GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse create_sandbox(request)

Create a new sandbox

Creates a new virtual machine sandbox by cloning from an existing VM. When multi-host is configured, automatically routes to the host containing the source VM.

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_request import GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response import GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse
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
    api_instance = fluid.SandboxApi(api_client)
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest | Sandbox creation parameters

    try:
        # Create a new sandbox
        api_response = api_instance.create_sandbox(request)
        print("The response of SandboxApi->create_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->create_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest.md)| Sandbox creation parameters | 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse.md)

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
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_snapshot**
> GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse create_snapshot(id, request)

Create snapshot

Creates a snapshot of the sandbox

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_snapshot_request import GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_snapshot_response import GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest | Snapshot parameters

    try:
        # Create snapshot
        api_response = api_instance.create_snapshot(id, request)
        print("The response of SandboxApi->create_snapshot:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->create_snapshot: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest.md)| Snapshot parameters | 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse.md)

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
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **destroy_sandbox**
> GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse destroy_sandbox(id)

Destroy sandbox

Destroys the sandbox and cleans up resources

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_destroy_sandbox_response import GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID

    try:
        # Destroy sandbox
        api_response = api_instance.destroy_sandbox(id)
        print("The response of SandboxApi->destroy_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->destroy_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **diff_snapshots**
> GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse diff_snapshots(id, request)

Diff snapshots

Computes differences between two snapshots

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_diff_request import GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_diff_response import GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest | Diff parameters

    try:
        # Diff snapshots
        api_response = api_instance.diff_snapshots(id, request)
        print("The response of SandboxApi->diff_snapshots:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->diff_snapshots: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest.md)| Diff parameters | 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse.md)

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
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **discover_sandbox_ip**
> GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse discover_sandbox_ip(id)

Discover sandbox IP

Discovers and returns the IP address for a running sandbox. Use this for async workflows where wait_for_ip was false during start.

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_discover_ip_response import GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID

    try:
        # Discover sandbox IP
        api_response = api_instance.discover_sandbox_ip(id)
        print("The response of SandboxApi->discover_sandbox_ip:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->discover_sandbox_ip: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **generate_configuration**
> generate_configuration(id, tool)

Generate configuration

Generates Ansible or Puppet configuration from sandbox changes

### Example


```python
import fluid
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    tool = 'tool_example' # str | Tool type (ansible or puppet)

    try:
        # Generate configuration
        api_instance.generate_configuration(id, tool)
    except Exception as e:
        print("Exception when calling SandboxApi->generate_configuration: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **tool** | **str**| Tool type (ansible or puppet) | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**400** | Bad Request |  -  |
**501** | Not Implemented |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_sandbox**
> GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse get_sandbox(id, include_commands=include_commands)

Get sandbox details

Returns detailed information about a specific sandbox including recent commands

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_get_sandbox_response import GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    include_commands = True # bool | Include command history (optional)

    try:
        # Get sandbox details
        api_response = api_instance.get_sandbox(id, include_commands=include_commands)
        print("The response of SandboxApi->get_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->get_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **include_commands** | **bool**| Include command history | [optional] 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **inject_ssh_key**
> inject_ssh_key(id, request)

Inject SSH key into sandbox

Injects a public SSH key for a user in the sandbox

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_inject_ssh_key_request import GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest | SSH key injection parameters

    try:
        # Inject SSH key into sandbox
        api_instance.inject_ssh_key(id, request)
    except Exception as e:
        print("Exception when calling SandboxApi->inject_ssh_key: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest.md)| SSH key injection parameters | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**204** | No Content |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_sandbox_commands**
> GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse list_sandbox_commands(id, limit=limit, offset=offset)

List sandbox commands

Returns all commands executed in the sandbox

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_sandbox_commands_response import GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    limit = 56 # int | Max results to return (optional)
    offset = 56 # int | Number of results to skip (optional)

    try:
        # List sandbox commands
        api_response = api_instance.list_sandbox_commands(id, limit=limit, offset=offset)
        print("The response of SandboxApi->list_sandbox_commands:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->list_sandbox_commands: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **limit** | **int**| Max results to return | [optional] 
 **offset** | **int**| Number of results to skip | [optional] 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**400** | Bad Request |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_sandboxes**
> GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse list_sandboxes(agent_id=agent_id, job_id=job_id, base_image=base_image, state=state, vm_name=vm_name, limit=limit, offset=offset)

List sandboxes

Lists all sandboxes with optional filtering by agent_id, job_id, base_image, state, or vm_name

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_sandboxes_response import GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse
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
    api_instance = fluid.SandboxApi(api_client)
    agent_id = 'agent_id_example' # str | Filter by agent ID (optional)
    job_id = 'job_id_example' # str | Filter by job ID (optional)
    base_image = 'base_image_example' # str | Filter by base image (optional)
    state = 'state_example' # str | Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) (optional)
    vm_name = 'vm_name_example' # str | Filter by VM name (optional)
    limit = 56 # int | Max results to return (optional)
    offset = 56 # int | Number of results to skip (optional)

    try:
        # List sandboxes
        api_response = api_instance.list_sandboxes(agent_id=agent_id, job_id=job_id, base_image=base_image, state=state, vm_name=vm_name, limit=limit, offset=offset)
        print("The response of SandboxApi->list_sandboxes:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->list_sandboxes: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **agent_id** | **str**| Filter by agent ID | [optional] 
 **job_id** | **str**| Filter by job ID | [optional] 
 **base_image** | **str**| Filter by base image | [optional] 
 **state** | **str**| Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) | [optional] 
 **vm_name** | **str**| Filter by VM name | [optional] 
 **limit** | **int**| Max results to return | [optional] 
 **offset** | **int**| Number of results to skip | [optional] 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse.md)

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

# **publish_changes**
> publish_changes(id, request)

Publish changes

Publishes sandbox changes to GitOps repository

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_publish_request import GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest | Publish parameters

    try:
        # Publish changes
        api_instance.publish_changes(id, request)
    except Exception as e:
        print("Exception when calling SandboxApi->publish_changes: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest.md)| Publish parameters | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**400** | Bad Request |  -  |
**501** | Not Implemented |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **run_sandbox_command**
> GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse run_sandbox_command(id, request)

Run command in sandbox

Executes a command inside the sandbox via SSH. If private_key_path is omitted and SSH CA is configured, managed credentials will be used automatically.

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request import GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_response import GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest | Command execution parameters

    try:
        # Run command in sandbox
        api_response = api_instance.run_sandbox_command(id, request)
        print("The response of SandboxApi->run_sandbox_command:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->run_sandbox_command: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest.md)| Command execution parameters | 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse.md)

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
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **start_sandbox**
> GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse start_sandbox(id, request=request)

Start sandbox

Starts the virtual machine sandbox

### Example


```python
import fluid
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_start_sandbox_request import GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_start_sandbox_response import GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID
    request = fluid.GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest() # GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest | Start parameters (optional)

    try:
        # Start sandbox
        api_response = api_instance.start_sandbox(id, request=request)
        print("The response of SandboxApi->start_sandbox:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SandboxApi->start_sandbox: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 
 **request** | [**GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest**](GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest.md)| Start parameters | [optional] 

### Return type

[**GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse**](GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse.md)

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
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **stream_sandbox_activity**
> stream_sandbox_activity(id)

Stream sandbox activity

Connects via WebSocket to stream realtime sandbox activity (commands, file changes)

### Example


```python
import fluid
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
    api_instance = fluid.SandboxApi(api_client)
    id = 'id_example' # str | Sandbox ID

    try:
        # Stream sandbox activity
        api_instance.stream_sandbox_activity(id)
    except Exception as e:
        print("Exception when calling SandboxApi->stream_sandbox_activity: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **id** | **str**| Sandbox ID | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: */*

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**101** | Switching Protocols - WebSocket connection established |  -  |
**400** | Invalid sandbox ID |  -  |
**404** | Sandbox not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

