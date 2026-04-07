# fluid.BillingApi

All URIs are relative to *http://localhost:8081/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**billing_calculator_post**](BillingApi.md#billing_calculator_post) | **POST** /billing/calculator | Pricing calculator
[**orgs_slug_billing_get**](BillingApi.md#orgs_slug_billing_get) | **GET** /orgs/{slug}/billing | Get billing info
[**orgs_slug_billing_portal_post**](BillingApi.md#orgs_slug_billing_portal_post) | **POST** /orgs/{slug}/billing/portal | Billing portal
[**orgs_slug_billing_subscribe_post**](BillingApi.md#orgs_slug_billing_subscribe_post) | **POST** /orgs/{slug}/billing/subscribe | Subscribe
[**orgs_slug_billing_usage_get**](BillingApi.md#orgs_slug_billing_usage_get) | **GET** /orgs/{slug}/billing/usage | Get usage
[**webhooks_stripe_post**](BillingApi.md#webhooks_stripe_post) | **POST** /webhooks/stripe | Stripe webhook


# **billing_calculator_post**
> RestCalculatorResponse billing_calculator_post(request)

Pricing calculator

Calculate estimated monthly costs based on resource usage

### Example


```python
import fluid
from fluid.models.rest_calculator_request import RestCalculatorRequest
from fluid.models.rest_calculator_response import RestCalculatorResponse
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
    api_instance = fluid.BillingApi(api_client)
    request = fluid.RestCalculatorRequest() # RestCalculatorRequest | Resource quantities

    try:
        # Pricing calculator
        api_response = api_instance.billing_calculator_post(request)
        print("The response of BillingApi->billing_calculator_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling BillingApi->billing_calculator_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**RestCalculatorRequest**](RestCalculatorRequest.md)| Resource quantities | 

### Return type

[**RestCalculatorResponse**](RestCalculatorResponse.md)

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

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **orgs_slug_billing_get**
> RestBillingResponse orgs_slug_billing_get(slug)

Get billing info

Get the current billing plan, status, and usage summary for an organization

### Example


```python
import fluid
from fluid.models.rest_billing_response import RestBillingResponse
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
    api_instance = fluid.BillingApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # Get billing info
        api_response = api_instance.orgs_slug_billing_get(slug)
        print("The response of BillingApi->orgs_slug_billing_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling BillingApi->orgs_slug_billing_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **slug** | **str**| Organization slug | 

### Return type

[**RestBillingResponse**](RestBillingResponse.md)

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

# **orgs_slug_billing_portal_post**
> Dict[str, str] orgs_slug_billing_portal_post(slug)

Billing portal

Create a Stripe billing portal session (owner only)

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
    api_instance = fluid.BillingApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # Billing portal
        api_response = api_instance.orgs_slug_billing_portal_post(slug)
        print("The response of BillingApi->orgs_slug_billing_portal_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling BillingApi->orgs_slug_billing_portal_post: %s\n" % e)
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

# **orgs_slug_billing_subscribe_post**
> Dict[str, str] orgs_slug_billing_subscribe_post(slug)

Subscribe

Create a Stripe checkout session for the organization (owner only)

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
    api_instance = fluid.BillingApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # Subscribe
        api_response = api_instance.orgs_slug_billing_subscribe_post(slug)
        print("The response of BillingApi->orgs_slug_billing_subscribe_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling BillingApi->orgs_slug_billing_subscribe_post: %s\n" % e)
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

# **orgs_slug_billing_usage_get**
> Dict[str, object] orgs_slug_billing_usage_get(slug)

Get usage

Get current month usage records for the organization

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
    api_instance = fluid.BillingApi(api_client)
    slug = 'slug_example' # str | Organization slug

    try:
        # Get usage
        api_response = api_instance.orgs_slug_billing_usage_get(slug)
        print("The response of BillingApi->orgs_slug_billing_usage_get:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling BillingApi->orgs_slug_billing_usage_get: %s\n" % e)
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

# **webhooks_stripe_post**
> Dict[str, str] webhooks_stripe_post()

Stripe webhook

Handle incoming Stripe webhook events

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
    api_instance = fluid.BillingApi(api_client)

    try:
        # Stripe webhook
        api_response = api_instance.webhooks_stripe_post()
        print("The response of BillingApi->webhooks_stripe_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling BillingApi->webhooks_stripe_post: %s\n" % e)
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

