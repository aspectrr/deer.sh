# fluid.HostTokensApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**orgs_slug_hosts_tokens_get**](HostTokensApi.md#orgs_slug_hosts_tokens_get) | **GET** /orgs/{slug}/hosts/tokens | List host tokens
[**orgs_slug_hosts_tokens_post**](HostTokensApi.md#orgs_slug_hosts_tokens_post) | **POST** /orgs/{slug}/hosts/tokens | Create host token
[**orgs_slug_hosts_tokens_token_id_delete**](HostTokensApi.md#orgs_slug_hosts_tokens_token_id_delete) | **DELETE** /orgs/{slug}/hosts/tokens/{tokenID} | Delete host token


# **orgs_slug_hosts_tokens_get**
> Dict[str, object] orgs_slug_hosts_tokens_get(slug)

List host tokens

List all host tokens for the organization

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
    api_instance = fluid.HostTokensApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # List host tokens
        api_response = api_instance.orgs_slug_hosts_tokens_get(slug)
        print("The response of HostTokensApi->orgs_slug_hosts_tokens_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling HostTokensApi->orgs_slug_hosts_tokens_get: %s\n" % e)
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

# **orgs_slug_hosts_tokens_post**
> RestHostTokenResponse orgs_slug_hosts_tokens_post(slug, request)

Create host token

Generate a new host authentication token (owner or admin only)

### Example


```python
import fluid
from fluid.models.rest_create_host_token_request import RestCreateHostTokenRequest
from fluid.models.rest_host_token_response import RestHostTokenResponse
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
    api_instance = fluid.HostTokensApi(api_client)
    slug = 'slug_example' # str | Organization slug
    request = fluid.RestCreateHostTokenRequest() # RestCreateHostTokenRequest | Token details

    try:
        # Create host token
        api_response = api_instance.orgs_slug_hosts_tokens_post(slug, request)
        print("The response of HostTokensApi->orgs_slug_hosts_tokens_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling HostTokensApi->orgs_slug_hosts_tokens_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **request** | [**RestCreateHostTokenRequest**](RestCreateHostTokenRequest.md)| Token details | 

### Return type

[**RestHostTokenResponse**](RestHostTokenResponse.md)

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

# **orgs_slug_hosts_tokens_token_id_delete**
> Dict[str, str] orgs_slug_hosts_tokens_token_id_delete(slug, token_id)

Delete host token

Delete a host token (owner or admin only)

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
    api_instance = fluid.HostTokensApi(api_client)
    slug = 'slug_example' # str | Organization slug
    token_id = 'token_id_example' # str | Token ID

    try:
        # Delete host token
        api_response = api_instance.orgs_slug_hosts_tokens_token_id_delete(slug, token_id)
        print("The response of HostTokensApi->orgs_slug_hosts_tokens_token_id_delete:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling HostTokensApi->orgs_slug_hosts_tokens_token_id_delete: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **token_id** | **str**| Token ID | 

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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

