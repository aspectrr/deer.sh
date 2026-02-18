# coding: utf-8

"""
    Fluid API
    API for managing sandboxes, organizations, billing, and hosts
"""

from typing import Any, Dict, List, Optional, Tuple, Union

from pydantic import Field, StrictStr
from typing_extensions import Annotated

from fluid.api_client import ApiClient, RequestSerialized
from fluid.api_response import ApiResponse
from fluid.exceptions import ApiException
from fluid.models.rest_create_org_request import RestCreateOrgRequest
from fluid.models.rest_org_response import RestOrgResponse
from fluid.models.rest_update_org_request import RestUpdateOrgRequest


class OrganizationsApi:
    """OrganizationsApi service"""

    def __init__(self, api_client: Optional[ApiClient] = None) -> None:
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def orgs_get(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Dict[str, object]:
        """List organizations

        List all organizations the current user belongs to

        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object.
        """

        _param = self._orgs_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, object]",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def orgs_get_with_http_info(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[Dict[str, object]]:
        """List organizations

        List all organizations the current user belongs to

        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object with HTTP info.
        """

        _param = self._orgs_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, object]",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def orgs_get_without_preload_content(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """List organizations

        List all organizations the current user belongs to

        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object without preloading content.
        """

        _param = self._orgs_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, object]",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_get_serialize(
        self,
        _request_auth: Optional[Dict[str, Any]],
        _content_type: Optional[str],
        _headers: Optional[Dict[str, Any]],
        _host_index: int,
    ) -> RequestSerialized:
        _host = None

        _collection_formats: Dict[str, str] = {}

        _path_params: Dict[str, str] = {}
        _query_params: List[Tuple[str, str]] = []
        _header_params: Dict[str, Optional[str]] = _headers or {}
        _form_params: List[Tuple[str, str]] = []
        _files: Dict[
            str, Union[str, bytes, List[str], List[bytes], List[Tuple[str, bytes]]]
        ] = {}
        _body_params: Any = None

        # process the path parameters
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Organizations" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/orgs",
            path_params=_path_params,
            query_params=_query_params,
            header_params=_header_params,
            body=_body_params,
            post_params=_form_params,
            files=_files,
            auth_settings=_auth_settings,
            collection_formats=_collection_formats,
            _host=_host,
            _request_auth=_request_auth,
        )

    def orgs_post(
        self,
        request: RestCreateOrgRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestOrgResponse:
        """Create organization

        Create a new organization and add the current user as owner

        :param request: Organization details (required)
        :type request: RestCreateOrgRequest
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object.
        """

        _param = self._orgs_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestOrgResponse",
            "400": "RestSwaggerError",
            "409": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def orgs_post_with_http_info(
        self,
        request: RestCreateOrgRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestOrgResponse]:
        """Create organization

        Create a new organization and add the current user as owner

        :param request: Organization details (required)
        :type request: RestCreateOrgRequest
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object with HTTP info.
        """

        _param = self._orgs_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestOrgResponse",
            "400": "RestSwaggerError",
            "409": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def orgs_post_without_preload_content(
        self,
        request: RestCreateOrgRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Create organization

        Create a new organization and add the current user as owner

        :param request: Organization details (required)
        :type request: RestCreateOrgRequest
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object without preloading content.
        """

        _param = self._orgs_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestOrgResponse",
            "400": "RestSwaggerError",
            "409": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_post_serialize(
        self,
        request: RestCreateOrgRequest,
        _request_auth: Optional[Dict[str, Any]],
        _content_type: Optional[str],
        _headers: Optional[Dict[str, Any]],
        _host_index: int,
    ) -> RequestSerialized:
        _host = None

        _collection_formats: Dict[str, str] = {}

        _path_params: Dict[str, str] = {}
        _query_params: List[Tuple[str, str]] = []
        _header_params: Dict[str, Optional[str]] = _headers or {}
        _form_params: List[Tuple[str, str]] = []
        _files: Dict[
            str, Union[str, bytes, List[str], List[bytes], List[Tuple[str, bytes]]]
        ] = {}
        _body_params: Any = None

        # process the path parameters
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Organizations" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`
        if _content_type:
            _header_params["Content-Type"] = _content_type
        else:
            _default_content_type = self.api_client.select_header_content_type(
                ["application/json"]
            )
            if _default_content_type is not None:
                _header_params["Content-Type"] = _default_content_type

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="POST",
            resource_path="/orgs",
            path_params=_path_params,
            query_params=_query_params,
            header_params=_header_params,
            body=_body_params,
            post_params=_form_params,
            files=_files,
            auth_settings=_auth_settings,
            collection_formats=_collection_formats,
            _host=_host,
            _request_auth=_request_auth,
        )

    def orgs_slug_delete(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Dict[str, str]:
        """Delete organization

        Delete an organization (owner only)

        :param slug: Organization slug (required)
        :type slug: str
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object.
        """

        _param = self._orgs_slug_delete_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, str]",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def orgs_slug_delete_with_http_info(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[Dict[str, str]]:
        """Delete organization

        Delete an organization (owner only)

        :param slug: Organization slug (required)
        :type slug: str
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object with HTTP info.
        """

        _param = self._orgs_slug_delete_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, str]",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def orgs_slug_delete_without_preload_content(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Delete organization

        Delete an organization (owner only)

        :param slug: Organization slug (required)
        :type slug: str
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object without preloading content.
        """

        _param = self._orgs_slug_delete_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, str]",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_slug_delete_serialize(
        self,
        slug: str,
        _request_auth: Optional[Dict[str, Any]],
        _content_type: Optional[str],
        _headers: Optional[Dict[str, Any]],
        _host_index: int,
    ) -> RequestSerialized:
        _host = None

        _collection_formats: Dict[str, str] = {}

        _path_params: Dict[str, str] = {}
        _query_params: List[Tuple[str, str]] = []
        _header_params: Dict[str, Optional[str]] = _headers or {}
        _form_params: List[Tuple[str, str]] = []
        _files: Dict[
            str, Union[str, bytes, List[str], List[bytes], List[Tuple[str, bytes]]]
        ] = {}
        _body_params: Any = None

        # process the path parameters
        if slug is not None:
            _path_params["slug"] = slug
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Organizations" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="DELETE",
            resource_path="/orgs/{slug}",
            path_params=_path_params,
            query_params=_query_params,
            header_params=_header_params,
            body=_body_params,
            post_params=_form_params,
            files=_files,
            auth_settings=_auth_settings,
            collection_formats=_collection_formats,
            _host=_host,
            _request_auth=_request_auth,
        )

    def orgs_slug_get(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestOrgResponse:
        """Get organization

        Get organization details by slug

        :param slug: Organization slug (required)
        :type slug: str
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object.
        """

        _param = self._orgs_slug_get_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestOrgResponse",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def orgs_slug_get_with_http_info(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestOrgResponse]:
        """Get organization

        Get organization details by slug

        :param slug: Organization slug (required)
        :type slug: str
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object with HTTP info.
        """

        _param = self._orgs_slug_get_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestOrgResponse",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def orgs_slug_get_without_preload_content(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Get organization

        Get organization details by slug

        :param slug: Organization slug (required)
        :type slug: str
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object without preloading content.
        """

        _param = self._orgs_slug_get_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestOrgResponse",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_slug_get_serialize(
        self,
        slug: str,
        _request_auth: Optional[Dict[str, Any]],
        _content_type: Optional[str],
        _headers: Optional[Dict[str, Any]],
        _host_index: int,
    ) -> RequestSerialized:
        _host = None

        _collection_formats: Dict[str, str] = {}

        _path_params: Dict[str, str] = {}
        _query_params: List[Tuple[str, str]] = []
        _header_params: Dict[str, Optional[str]] = _headers or {}
        _form_params: List[Tuple[str, str]] = []
        _files: Dict[
            str, Union[str, bytes, List[str], List[bytes], List[Tuple[str, bytes]]]
        ] = {}
        _body_params: Any = None

        # process the path parameters
        if slug is not None:
            _path_params["slug"] = slug
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Organizations" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/orgs/{slug}",
            path_params=_path_params,
            query_params=_query_params,
            header_params=_header_params,
            body=_body_params,
            post_params=_form_params,
            files=_files,
            auth_settings=_auth_settings,
            collection_formats=_collection_formats,
            _host=_host,
            _request_auth=_request_auth,
        )

    def orgs_slug_patch(
        self,
        slug: str,
        request: RestUpdateOrgRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestOrgResponse:
        """Update organization

        Update organization details (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param request: Fields to update (required)
        :type request: RestUpdateOrgRequest
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object.
        """

        _param = self._orgs_slug_patch_serialize(
            slug=slug,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestOrgResponse",
            "400": "RestSwaggerError",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def orgs_slug_patch_with_http_info(
        self,
        slug: str,
        request: RestUpdateOrgRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestOrgResponse]:
        """Update organization

        Update organization details (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param request: Fields to update (required)
        :type request: RestUpdateOrgRequest
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object with HTTP info.
        """

        _param = self._orgs_slug_patch_serialize(
            slug=slug,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestOrgResponse",
            "400": "RestSwaggerError",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def orgs_slug_patch_without_preload_content(
        self,
        slug: str,
        request: RestUpdateOrgRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Update organization

        Update organization details (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param request: Fields to update (required)
        :type request: RestUpdateOrgRequest
        :param _request_timeout: Timeout setting for this request. If one
                                 number is provided, it will be the total request
                                 timeout. It can also be a pair (tuple) of
                                 (connection, read) timeouts.
        :type _request_timeout: int, tuple(int, int), optional
        :param _request_auth: Override the auth_settings for a single request.
        :type _request_auth: dict, optional
        :param _content_type: Force content-type for the request.
        :type _content_type: str, optional
        :param _headers: Override headers for a single request.
        :type _headers: dict, optional
        :param _host_index: Override host index for a single request.
        :type _host_index: int, optional
        :return: Returns the result object without preloading content.
        """

        _param = self._orgs_slug_patch_serialize(
            slug=slug,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestOrgResponse",
            "400": "RestSwaggerError",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_slug_patch_serialize(
        self,
        slug: str,
        request: RestUpdateOrgRequest,
        _request_auth: Optional[Dict[str, Any]],
        _content_type: Optional[str],
        _headers: Optional[Dict[str, Any]],
        _host_index: int,
    ) -> RequestSerialized:
        _host = None

        _collection_formats: Dict[str, str] = {}

        _path_params: Dict[str, str] = {}
        _query_params: List[Tuple[str, str]] = []
        _header_params: Dict[str, Optional[str]] = _headers or {}
        _form_params: List[Tuple[str, str]] = []
        _files: Dict[
            str, Union[str, bytes, List[str], List[bytes], List[Tuple[str, bytes]]]
        ] = {}
        _body_params: Any = None

        # process the path parameters
        if slug is not None:
            _path_params["slug"] = slug
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Organizations" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`
        if _content_type:
            _header_params["Content-Type"] = _content_type
        else:
            _default_content_type = self.api_client.select_header_content_type(
                ["application/json"]
            )
            if _default_content_type is not None:
                _header_params["Content-Type"] = _default_content_type

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="PATCH",
            resource_path="/orgs/{slug}",
            path_params=_path_params,
            query_params=_query_params,
            header_params=_header_params,
            body=_body_params,
            post_params=_form_params,
            files=_files,
            auth_settings=_auth_settings,
            collection_formats=_collection_formats,
            _host=_host,
            _request_auth=_request_auth,
        )
