# fluid.OrganizationsApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**orgs_get**](OrganizationsApi.md#orgs_get) | **GET** /orgs | List organizations
[**orgs_post**](OrganizationsApi.md#orgs_post) | **POST** /orgs | Create organization
[**orgs_slug_delete**](OrganizationsApi.md#orgs_slug_delete) | **DELETE** /orgs/{slug} | Delete organization
[**orgs_slug_get**](OrganizationsApi.md#orgs_slug_get) | **GET** /orgs/{slug} | Get organization
[**orgs_slug_patch**](OrganizationsApi.md#orgs_slug_patch) | **PATCH** /orgs/{slug} | Update organization


# **orgs_get**
> Dict[str, object] orgs_get()

List organizations

List all organizations the current user belongs to

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
    api_instance = fluid.OrganizationsApi(api_client)

    try:
        # List organizations
        api_response = api_instance.orgs_get()
        print("The response of OrganizationsApi->orgs_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OrganizationsApi->orgs_get: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

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
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_post**
> RestOrgResponse orgs_post(request)

Create organization

Create a new organization and add the current user as owner

### Example


```python
import fluid
from fluid.models.rest_create_org_request import RestCreateOrgRequest
from fluid.models.rest_org_response import RestOrgResponse
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
    api_instance = fluid.OrganizationsApi(api_client)
    request = fluid.RestCreateOrgRequest() # RestCreateOrgRequest | Organization details

    try:
        # Create organization
        api_response = api_instance.orgs_post(request)
        print("The response of OrganizationsApi->orgs_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OrganizationsApi->orgs_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**RestCreateOrgRequest**](RestCreateOrgRequest.md)| Organization details | 

### Return type

[**RestOrgResponse**](RestOrgResponse.md)

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
**409** | Conflict |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_delete**
> Dict[str, str] orgs_slug_delete(slug)

Delete organization

Delete an organization (owner only)

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
    api_instance = fluid.OrganizationsApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # Delete organization
        api_response = api_instance.orgs_slug_delete(slug)
        print("The response of OrganizationsApi->orgs_slug_delete:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OrganizationsApi->orgs_slug_delete: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 

### Return type

**Dict[str, str]**

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

# **orgs_slug_get**
> RestOrgResponse orgs_slug_get(slug)

Get organization

Get organization details by slug

### Example


```python
import fluid
from fluid.models.rest_org_response import RestOrgResponse
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
    api_instance = fluid.OrganizationsApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # Get organization
        api_response = api_instance.orgs_slug_get(slug)
        print("The response of OrganizationsApi->orgs_slug_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OrganizationsApi->orgs_slug_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 

### Return type

[**RestOrgResponse**](RestOrgResponse.md)

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

# **orgs_slug_patch**
> RestOrgResponse orgs_slug_patch(slug, request)

Update organization

Update organization details (owner or admin only)

### Example


```python
import fluid
from fluid.models.rest_org_response import RestOrgResponse
from fluid.models.rest_update_org_request import RestUpdateOrgRequest
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
    api_instance = fluid.OrganizationsApi(api_client)
    slug = 'slug_example' # str | Organization slug
    request = fluid.RestUpdateOrgRequest() # RestUpdateOrgRequest | Fields to update

    try:
        # Update organization
        api_response = api_instance.orgs_slug_patch(slug, request)
        print("The response of OrganizationsApi->orgs_slug_patch:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OrganizationsApi->orgs_slug_patch: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **request** | [**RestUpdateOrgRequest**](RestUpdateOrgRequest.md)| Fields to update | 

### Return type

[**RestOrgResponse**](RestOrgResponse.md)

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

