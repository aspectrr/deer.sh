# fluid.AuthApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**auth_github_callback_get**](AuthApi.md#auth_github_callback_get) | **GET** /auth/github/callback | GitHub OAuth callback
[**auth_github_get**](AuthApi.md#auth_github_get) | **GET** /auth/github | GitHub OAuth login
[**auth_google_callback_get**](AuthApi.md#auth_google_callback_get) | **GET** /auth/google/callback | Google OAuth callback
[**auth_google_get**](AuthApi.md#auth_google_get) | **GET** /auth/google | Google OAuth login
[**auth_login_post**](AuthApi.md#auth_login_post) | **POST** /auth/login | Log in
[**auth_logout_post**](AuthApi.md#auth_logout_post) | **POST** /auth/logout | Log out
[**auth_me_get**](AuthApi.md#auth_me_get) | **GET** /auth/me | Get current user
[**auth_register_post**](AuthApi.md#auth_register_post) | **POST** /auth/register | Register a new user


# **auth_github_callback_get**
> auth_github_callback_get(code)

GitHub OAuth callback

Handle GitHub OAuth callback, create or link user, set session cookie, and redirect to dashboard

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
    api_instance = fluid.AuthApi(api_client)
    code = 'code_example' # str | OAuth authorization code

    try:
        # GitHub OAuth callback
        api_instance.auth_github_callback_get(code)
    except Exception as e:
        print("Exception when calling AuthApi->auth_github_callback_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **code** | **str**| OAuth authorization code | 

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
**302** | Redirect to dashboard |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_github_get**
> auth_github_get()

GitHub OAuth login

Redirect to GitHub OAuth authorization page

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
    api_instance = fluid.AuthApi(api_client)

    try:
        # GitHub OAuth login
        api_instance.auth_github_get()
    except Exception as e:
        print("Exception when calling AuthApi->auth_github_get: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

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
**302** | Redirect to GitHub |  -  |
**501** | Not Implemented |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_google_callback_get**
> auth_google_callback_get(code)

Google OAuth callback

Handle Google OAuth callback, create or link user, set session cookie, and redirect to dashboard

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
    api_instance = fluid.AuthApi(api_client)
    code = 'code_example' # str | OAuth authorization code

    try:
        # Google OAuth callback
        api_instance.auth_google_callback_get(code)
    except Exception as e:
        print("Exception when calling AuthApi->auth_google_callback_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **code** | **str**| OAuth authorization code | 

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
**302** | Redirect to dashboard |  -  |
**400** | Bad Request |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_google_get**
> auth_google_get()

Google OAuth login

Redirect to Google OAuth authorization page

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
    api_instance = fluid.AuthApi(api_client)

    try:
        # Google OAuth login
        api_instance.auth_google_get()
    except Exception as e:
        print("Exception when calling AuthApi->auth_google_get: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

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
**302** | Redirect to Google |  -  |
**501** | Not Implemented |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_login_post**
> RestAuthResponse auth_login_post(request)

Log in

Authenticate with email and password, returns a session cookie

### Example


```python
import fluid
from fluid.models.rest_auth_response import RestAuthResponse
from fluid.models.rest_login_request import RestLoginRequest
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
    api_instance = fluid.AuthApi(api_client)
    request = fluid.RestLoginRequest() # RestLoginRequest | Login credentials

    try:
        # Log in
        api_response = api_instance.auth_login_post(request)
        print("The response of AuthApi->auth_login_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AuthApi->auth_login_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**RestLoginRequest**](RestLoginRequest.md)| Login credentials | 

### Return type

[**RestAuthResponse**](RestAuthResponse.md)

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
**401** | Unauthorized |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_logout_post**
> Dict[str, str] auth_logout_post()

Log out

Invalidate the current session and clear the session cookie

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
    api_instance = fluid.AuthApi(api_client)

    try:
        # Log out
        api_response = api_instance.auth_logout_post()
        print("The response of AuthApi->auth_logout_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AuthApi->auth_logout_post: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_me_get**
> RestAuthResponse auth_me_get()

Get current user

Return the currently authenticated user

### Example


```python
import fluid
from fluid.models.rest_auth_response import RestAuthResponse
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
    api_instance = fluid.AuthApi(api_client)

    try:
        # Get current user
        api_response = api_instance.auth_me_get()
        print("The response of AuthApi->auth_me_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AuthApi->auth_me_get: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**RestAuthResponse**](RestAuthResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**401** | Unauthorized |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **auth_register_post**
> RestAuthResponse auth_register_post(request)

Register a new user

Create a new user account and return a session cookie

### Example


```python
import fluid
from fluid.models.rest_auth_response import RestAuthResponse
from fluid.models.rest_register_request import RestRegisterRequest
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
    api_instance = fluid.AuthApi(api_client)
    request = fluid.RestRegisterRequest() # RestRegisterRequest | Registration details

    try:
        # Register a new user
        api_response = api_instance.auth_register_post(request)
        print("The response of AuthApi->auth_register_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AuthApi->auth_register_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**RestRegisterRequest**](RestRegisterRequest.md)| Registration details | 

### Return type

[**RestAuthResponse**](RestAuthResponse.md)

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

