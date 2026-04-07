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
from fluid.models.rest_add_member_request import RestAddMemberRequest
from fluid.models.rest_member_response import RestMemberResponse


class MembersApi:
    """MembersApi service"""

    def __init__(self, api_client: Optional[ApiClient] = None) -> None:
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def orgs_slug_members_get(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Dict[str, object]:
        """List members

        List all members of an organization

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

        _param = self._orgs_slug_members_get_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, object]",
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

    def orgs_slug_members_get_with_http_info(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[Dict[str, object]]:
        """List members

        List all members of an organization

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

        _param = self._orgs_slug_members_get_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, object]",
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

    def orgs_slug_members_get_without_preload_content(
        self,
        slug: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """List members

        List all members of an organization

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

        _param = self._orgs_slug_members_get_serialize(
            slug=slug,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, object]",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_slug_members_get_serialize(
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
        if "Members" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/orgs/{slug}/members",
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

    def orgs_slug_members_member_id_delete(
        self,
        slug: str,
        member_id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Dict[str, str]:
        """Remove member

        Remove a member from an organization (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param member_id: Member ID (required)
        :type member_id: str
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

        _param = self._orgs_slug_members_member_id_delete_serialize(
            slug=slug,
            member_id=member_id,
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

    def orgs_slug_members_member_id_delete_with_http_info(
        self,
        slug: str,
        member_id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[Dict[str, str]]:
        """Remove member

        Remove a member from an organization (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param member_id: Member ID (required)
        :type member_id: str
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

        _param = self._orgs_slug_members_member_id_delete_serialize(
            slug=slug,
            member_id=member_id,
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

    def orgs_slug_members_member_id_delete_without_preload_content(
        self,
        slug: str,
        member_id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Remove member

        Remove a member from an organization (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param member_id: Member ID (required)
        :type member_id: str
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

        _param = self._orgs_slug_members_member_id_delete_serialize(
            slug=slug,
            member_id=member_id,
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

    def _orgs_slug_members_member_id_delete_serialize(
        self,
        slug: str,
        member_id: str,
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
        if member_id is not None:
            _path_params["memberID"] = member_id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Members" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="DELETE",
            resource_path="/orgs/{slug}/members/{memberID}",
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

    def orgs_slug_members_post(
        self,
        slug: str,
        request: RestAddMemberRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestMemberResponse:
        """Add member

        Add a user to an organization (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param request: Member details (required)
        :type request: RestAddMemberRequest
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

        _param = self._orgs_slug_members_post_serialize(
            slug=slug,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestMemberResponse",
            "400": "RestSwaggerError",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
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

    def orgs_slug_members_post_with_http_info(
        self,
        slug: str,
        request: RestAddMemberRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestMemberResponse]:
        """Add member

        Add a user to an organization (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param request: Member details (required)
        :type request: RestAddMemberRequest
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

        _param = self._orgs_slug_members_post_serialize(
            slug=slug,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestMemberResponse",
            "400": "RestSwaggerError",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
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

    def orgs_slug_members_post_without_preload_content(
        self,
        slug: str,
        request: RestAddMemberRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Add member

        Add a user to an organization (owner or admin only)

        :param slug: Organization slug (required)
        :type slug: str
        :param request: Member details (required)
        :type request: RestAddMemberRequest
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

        _param = self._orgs_slug_members_post_serialize(
            slug=slug,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestMemberResponse",
            "400": "RestSwaggerError",
            "403": "RestSwaggerError",
            "404": "RestSwaggerError",
            "409": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _orgs_slug_members_post_serialize(
        self,
        slug: str,
        request: RestAddMemberRequest,
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
        if "Members" not in _header_params:
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
            resource_path="/orgs/{slug}/members",
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
