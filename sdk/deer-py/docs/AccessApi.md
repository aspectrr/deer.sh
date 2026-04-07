# fluid.AccessApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_ca_public_key**](AccessApi.md#get_ca_public_key) | **GET** /v1/access/ca-pubkey | Get the SSH CA public key
[**get_certificate**](AccessApi.md#get_certificate) | **GET** /v1/access/certificate/{certID} | Get certificate details
[**list_certificates**](AccessApi.md#list_certificates) | **GET** /v1/access/certificates | List certificates
[**list_sessions**](AccessApi.md#list_sessions) | **GET** /v1/access/sessions | List sessions
[**record_session_end**](AccessApi.md#record_session_end) | **POST** /v1/access/session/end | Record session end
[**record_session_start**](AccessApi.md#record_session_start) | **POST** /v1/access/session/start | Record session start
[**request_access**](AccessApi.md#request_access) | **POST** /v1/access/request | Request SSH access to a sandbox
[**revoke_certificate**](AccessApi.md#revoke_certificate) | **DELETE** /v1/access/certificate/{certID} | Revoke a certificate


# **get_ca_public_key**
> InternalRestCaPublicKeyResponse get_ca_public_key()

Get the SSH CA public key

Returns the CA public key that should be trusted by VMs

### Example


```python
import fluid
from fluid.models.internal_rest_ca_public_key_response import InternalRestCaPublicKeyResponse
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
    api_instance = fluid.AccessApi(api_client)

    try:
        # Get the SSH CA public key
        api_response = api_instance.get_ca_public_key()
        print("The response of AccessApi->get_ca_public_key:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->get_ca_public_key: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**InternalRestCaPublicKeyResponse**](InternalRestCaPublicKeyResponse.md)

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

# **get_certificate**
> InternalRestCertificateResponse get_certificate(cert_id)

Get certificate details

Returns details about an issued certificate

### Example


```python
import fluid
from fluid.models.internal_rest_certificate_response import InternalRestCertificateResponse
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
    api_instance = fluid.AccessApi(api_client)
    cert_id = 'cert_id_example' # str | Certificate ID

    try:
        # Get certificate details
        api_response = api_instance.get_certificate(cert_id)
        print("The response of AccessApi->get_certificate:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->get_certificate: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cert_id** | **str**| Certificate ID | 

### Return type

[**InternalRestCertificateResponse**](InternalRestCertificateResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | OK |  -  |
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_certificates**
> InternalRestListCertificatesResponse list_certificates(sandbox_id=sandbox_id, user_id=user_id, status=status, active_only=active_only, limit=limit, offset=offset)

List certificates

Lists issued certificates with optional filtering

### Example


```python
import fluid
from fluid.models.internal_rest_list_certificates_response import InternalRestListCertificatesResponse
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
    api_instance = fluid.AccessApi(api_client)
    sandbox_id = 'sandbox_id_example' # str | Filter by sandbox ID (optional)
    user_id = 'user_id_example' # str | Filter by user ID (optional)
    status = 'status_example' # str | Filter by status (ACTIVE, EXPIRED, REVOKED) (optional)
    active_only = True # bool | Only show active, non-expired certificates (optional)
    limit = 56 # int | Maximum results to return (optional)
    offset = 56 # int | Offset for pagination (optional)

    try:
        # List certificates
        api_response = api_instance.list_certificates(sandbox_id=sandbox_id, user_id=user_id, status=status, active_only=active_only, limit=limit, offset=offset)
        print("The response of AccessApi->list_certificates:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->list_certificates: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **sandbox_id** | **str**| Filter by sandbox ID | [optional] 
 **user_id** | **str**| Filter by user ID | [optional] 
 **status** | **str**| Filter by status (ACTIVE, EXPIRED, REVOKED) | [optional] 
 **active_only** | **bool**| Only show active, non-expired certificates | [optional] 
 **limit** | **int**| Maximum results to return | [optional] 
 **offset** | **int**| Offset for pagination | [optional] 

### Return type

[**InternalRestListCertificatesResponse**](InternalRestListCertificatesResponse.md)

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

# **list_sessions**
> InternalRestListSessionsResponse list_sessions(sandbox_id=sandbox_id, certificate_id=certificate_id, user_id=user_id, active_only=active_only, limit=limit, offset=offset)

List sessions

Lists access sessions with optional filtering

### Example


```python
import fluid
from fluid.models.internal_rest_list_sessions_response import InternalRestListSessionsResponse
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
    api_instance = fluid.AccessApi(api_client)
    sandbox_id = 'sandbox_id_example' # str | Filter by sandbox ID (optional)
    certificate_id = 'certificate_id_example' # str | Filter by certificate ID (optional)
    user_id = 'user_id_example' # str | Filter by user ID (optional)
    active_only = True # bool | Only show active sessions (optional)
    limit = 56 # int | Maximum results to return (optional)
    offset = 56 # int | Offset for pagination (optional)

    try:
        # List sessions
        api_response = api_instance.list_sessions(sandbox_id=sandbox_id, certificate_id=certificate_id, user_id=user_id, active_only=active_only, limit=limit, offset=offset)
        print("The response of AccessApi->list_sessions:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->list_sessions: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **sandbox_id** | **str**| Filter by sandbox ID | [optional] 
 **certificate_id** | **str**| Filter by certificate ID | [optional] 
 **user_id** | **str**| Filter by user ID | [optional] 
 **active_only** | **bool**| Only show active sessions | [optional] 
 **limit** | **int**| Maximum results to return | [optional] 
 **offset** | **int**| Offset for pagination | [optional] 

### Return type

[**InternalRestListSessionsResponse**](InternalRestListSessionsResponse.md)

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

# **record_session_end**
> InternalRestSessionEndResponse record_session_end(request)

Record session end

Records the end of an SSH session

### Example


```python
import fluid
from fluid.models.internal_rest_session_end_request import InternalRestSessionEndRequest
from fluid.models.internal_rest_session_end_response import InternalRestSessionEndResponse
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
    api_instance = fluid.AccessApi(api_client)
    request = fluid.InternalRestSessionEndRequest() # InternalRestSessionEndRequest | Session end request

    try:
        # Record session end
        api_response = api_instance.record_session_end(request)
        print("The response of AccessApi->record_session_end:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->record_session_end: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**InternalRestSessionEndRequest**](InternalRestSessionEndRequest.md)| Session end request | 

### Return type

[**InternalRestSessionEndResponse**](InternalRestSessionEndResponse.md)

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

# **record_session_start**
> InternalRestSessionStartResponse record_session_start(request)

Record session start

Records the start of an SSH session (called by VM or auth service)

### Example


```python
import fluid
from fluid.models.internal_rest_session_start_request import InternalRestSessionStartRequest
from fluid.models.internal_rest_session_start_response import InternalRestSessionStartResponse
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
    api_instance = fluid.AccessApi(api_client)
    request = fluid.InternalRestSessionStartRequest() # InternalRestSessionStartRequest | Session start request

    try:
        # Record session start
        api_response = api_instance.record_session_start(request)
        print("The response of AccessApi->record_session_start:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->record_session_start: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**InternalRestSessionStartRequest**](InternalRestSessionStartRequest.md)| Session start request | 

### Return type

[**InternalRestSessionStartResponse**](InternalRestSessionStartResponse.md)

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

# **request_access**
> InternalRestRequestAccessResponse request_access(request)

Request SSH access to a sandbox

Issues a short-lived SSH certificate for accessing a sandbox via tmux

### Example


```python
import fluid
from fluid.models.internal_rest_request_access_request import InternalRestRequestAccessRequest
from fluid.models.internal_rest_request_access_response import InternalRestRequestAccessResponse
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
    api_instance = fluid.AccessApi(api_client)
    request = fluid.InternalRestRequestAccessRequest() # InternalRestRequestAccessRequest | Access request

    try:
        # Request SSH access to a sandbox
        api_response = api_instance.request_access(request)
        print("The response of AccessApi->request_access:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->request_access: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**InternalRestRequestAccessRequest**](InternalRestRequestAccessRequest.md)| Access request | 

### Return type

[**InternalRestRequestAccessResponse**](InternalRestRequestAccessResponse.md)

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
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **revoke_certificate**
> InternalRestRevokeCertificateResponse revoke_certificate(cert_id, request=request)

Revoke a certificate

Immediately revokes a certificate, terminating any active sessions

### Example


```python
import fluid
from fluid.models.internal_rest_revoke_certificate_request import InternalRestRevokeCertificateRequest
from fluid.models.internal_rest_revoke_certificate_response import InternalRestRevokeCertificateResponse
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
    api_instance = fluid.AccessApi(api_client)
    cert_id = 'cert_id_example' # str | Certificate ID
    request = fluid.InternalRestRevokeCertificateRequest() # InternalRestRevokeCertificateRequest | Revocation reason (optional)

    try:
        # Revoke a certificate
        api_response = api_instance.revoke_certificate(cert_id, request=request)
        print("The response of AccessApi->revoke_certificate:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AccessApi->revoke_certificate: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cert_id** | **str**| Certificate ID | 
 **request** | [**InternalRestRevokeCertificateRequest**](InternalRestRevokeCertificateRequest.md)| Revocation reason | [optional] 

### Return type

[**InternalRestRevokeCertificateResponse**](InternalRestRevokeCertificateResponse.md)

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
**404** | Not Found |  -  |
**500** | Internal Server Error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

