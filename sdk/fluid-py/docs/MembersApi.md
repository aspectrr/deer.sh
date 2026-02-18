# fluid.MembersApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**orgs_slug_members_get**](MembersApi.md#orgs_slug_members_get) | **GET** /orgs/{slug}/members | List members
[**orgs_slug_members_member_id_delete**](MembersApi.md#orgs_slug_members_member_id_delete) | **DELETE** /orgs/{slug}/members/{memberID} | Remove member
[**orgs_slug_members_post**](MembersApi.md#orgs_slug_members_post) | **POST** /orgs/{slug}/members | Add member


# **orgs_slug_members_get**
> Dict[str, object] orgs_slug_members_get(slug)

List members

List all members of an organization

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
    api_instance = fluid.MembersApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # List members
        api_response = api_instance.orgs_slug_members_get(slug)
        print("The response of MembersApi->orgs_slug_members_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MembersApi->orgs_slug_members_get: %s\n" % e)
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

# **orgs_slug_members_member_id_delete**
> Dict[str, str] orgs_slug_members_member_id_delete(slug, member_id)

Remove member

Remove a member from an organization (owner or admin only)

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
    api_instance = fluid.MembersApi(api_client)
    slug = 'slug_example' # str | Organization slug
    member_id = 'member_id_example' # str | Member ID

    try:
        # Remove member
        api_response = api_instance.orgs_slug_members_member_id_delete(slug, member_id)
        print("The response of MembersApi->orgs_slug_members_member_id_delete:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MembersApi->orgs_slug_members_member_id_delete: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **member_id** | **str**| Member ID | 

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

# **orgs_slug_members_post**
> RestMemberResponse orgs_slug_members_post(slug, request)

Add member

Add a user to an organization (owner or admin only)

### Example


```python
import fluid
from fluid.models.rest_add_member_request import RestAddMemberRequest
from fluid.models.rest_member_response import RestMemberResponse
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
    api_instance = fluid.MembersApi(api_client)
    slug = 'slug_example' # str | Organization slug
    request = fluid.RestAddMemberRequest() # RestAddMemberRequest | Member details

    try:
        # Add member
        api_response = api_instance.orgs_slug_members_post(slug, request)
        print("The response of MembersApi->orgs_slug_members_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MembersApi->orgs_slug_members_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 
 **request** | [**RestAddMemberRequest**](RestAddMemberRequest.md)| Member details | 

### Return type

[**RestMemberResponse**](RestMemberResponse.md)

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
**409** | Conflict |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

