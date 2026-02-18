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
from fluid.models.rest_auth_response import RestAuthResponse
from fluid.models.rest_login_request import RestLoginRequest
from fluid.models.rest_register_request import RestRegisterRequest


class AuthApi:
    """AuthApi service"""

    def __init__(self, api_client: Optional[ApiClient] = None) -> None:
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def auth_github_callback_get(
        self,
        code: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """GitHub OAuth callback

        Handle GitHub OAuth callback, create or link user, set session cookie, and redirect to dashboard

        :param code: OAuth authorization code (required)
        :type code: str
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

        _param = self._auth_github_callback_get_serialize(
            code=code,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "400": "RestSwaggerError",
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

    def auth_github_callback_get_with_http_info(
        self,
        code: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """GitHub OAuth callback

        Handle GitHub OAuth callback, create or link user, set session cookie, and redirect to dashboard

        :param code: OAuth authorization code (required)
        :type code: str
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

        _param = self._auth_github_callback_get_serialize(
            code=code,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "400": "RestSwaggerError",
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

    def auth_github_callback_get_without_preload_content(
        self,
        code: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """GitHub OAuth callback

        Handle GitHub OAuth callback, create or link user, set session cookie, and redirect to dashboard

        :param code: OAuth authorization code (required)
        :type code: str
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

        _param = self._auth_github_callback_get_serialize(
            code=code,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "400": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_github_callback_get_serialize(
        self,
        code: str,
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
        if code is not None:
            _query_params.append(("code", code))

        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Auth" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(["*/*"])

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/auth/github/callback",
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

    def auth_github_get(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """GitHub OAuth login

        Redirect to GitHub OAuth authorization page

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

        _param = self._auth_github_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "501": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def auth_github_get_with_http_info(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """GitHub OAuth login

        Redirect to GitHub OAuth authorization page

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

        _param = self._auth_github_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "501": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def auth_github_get_without_preload_content(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """GitHub OAuth login

        Redirect to GitHub OAuth authorization page

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

        _param = self._auth_github_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "501": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_github_get_serialize(
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
        if "Auth" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(["*/*"])

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/auth/github",
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

    def auth_google_callback_get(
        self,
        code: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """Google OAuth callback

        Handle Google OAuth callback, create or link user, set session cookie, and redirect to dashboard

        :param code: OAuth authorization code (required)
        :type code: str
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

        _param = self._auth_google_callback_get_serialize(
            code=code,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "400": "RestSwaggerError",
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

    def auth_google_callback_get_with_http_info(
        self,
        code: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """Google OAuth callback

        Handle Google OAuth callback, create or link user, set session cookie, and redirect to dashboard

        :param code: OAuth authorization code (required)
        :type code: str
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

        _param = self._auth_google_callback_get_serialize(
            code=code,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "400": "RestSwaggerError",
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

    def auth_google_callback_get_without_preload_content(
        self,
        code: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Google OAuth callback

        Handle Google OAuth callback, create or link user, set session cookie, and redirect to dashboard

        :param code: OAuth authorization code (required)
        :type code: str
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

        _param = self._auth_google_callback_get_serialize(
            code=code,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "400": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_google_callback_get_serialize(
        self,
        code: str,
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
        if code is not None:
            _query_params.append(("code", code))

        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Auth" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(["*/*"])

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/auth/google/callback",
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

    def auth_google_get(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """Google OAuth login

        Redirect to Google OAuth authorization page

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

        _param = self._auth_google_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "501": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def auth_google_get_with_http_info(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """Google OAuth login

        Redirect to Google OAuth authorization page

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

        _param = self._auth_google_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "501": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def auth_google_get_without_preload_content(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Google OAuth login

        Redirect to Google OAuth authorization page

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

        _param = self._auth_google_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "302": None,
            "501": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_google_get_serialize(
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
        if "Auth" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(["*/*"])

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/auth/google",
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

    def auth_login_post(
        self,
        request: RestLoginRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestAuthResponse:
        """Log in

        Authenticate with email and password, returns a session cookie

        :param request: Login credentials (required)
        :type request: RestLoginRequest
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

        _param = self._auth_login_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestAuthResponse",
            "400": "RestSwaggerError",
            "401": "RestSwaggerError",
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

    def auth_login_post_with_http_info(
        self,
        request: RestLoginRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestAuthResponse]:
        """Log in

        Authenticate with email and password, returns a session cookie

        :param request: Login credentials (required)
        :type request: RestLoginRequest
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

        _param = self._auth_login_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestAuthResponse",
            "400": "RestSwaggerError",
            "401": "RestSwaggerError",
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

    def auth_login_post_without_preload_content(
        self,
        request: RestLoginRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Log in

        Authenticate with email and password, returns a session cookie

        :param request: Login credentials (required)
        :type request: RestLoginRequest
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

        _param = self._auth_login_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestAuthResponse",
            "400": "RestSwaggerError",
            "401": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_login_post_serialize(
        self,
        request: RestLoginRequest,
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
        if "Auth" not in _header_params:
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
            resource_path="/auth/login",
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

    def auth_logout_post(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Dict[str, str]:
        """Log out

        Invalidate the current session and clear the session cookie

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

        _param = self._auth_logout_post_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, str]",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def auth_logout_post_with_http_info(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[Dict[str, str]]:
        """Log out

        Invalidate the current session and clear the session cookie

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

        _param = self._auth_logout_post_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, str]",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def auth_logout_post_without_preload_content(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Log out

        Invalidate the current session and clear the session cookie

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

        _param = self._auth_logout_post_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "Dict[str, str]",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_logout_post_serialize(
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
        if "Auth" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="POST",
            resource_path="/auth/logout",
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

    def auth_me_get(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestAuthResponse:
        """Get current user

        Return the currently authenticated user

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

        _param = self._auth_me_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestAuthResponse",
            "401": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def auth_me_get_with_http_info(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestAuthResponse]:
        """Get current user

        Return the currently authenticated user

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

        _param = self._auth_me_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestAuthResponse",
            "401": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def auth_me_get_without_preload_content(
        self,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Get current user

        Return the currently authenticated user

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

        _param = self._auth_me_get_serialize(
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "RestAuthResponse",
            "401": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_me_get_serialize(
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
        if "Auth" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/auth/me",
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

    def auth_register_post(
        self,
        request: RestRegisterRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> RestAuthResponse:
        """Register a new user

        Create a new user account and return a session cookie

        :param request: Registration details (required)
        :type request: RestRegisterRequest
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

        _param = self._auth_register_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestAuthResponse",
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

    def auth_register_post_with_http_info(
        self,
        request: RestRegisterRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[RestAuthResponse]:
        """Register a new user

        Create a new user account and return a session cookie

        :param request: Registration details (required)
        :type request: RestRegisterRequest
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

        _param = self._auth_register_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestAuthResponse",
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

    def auth_register_post_without_preload_content(
        self,
        request: RestRegisterRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Register a new user

        Create a new user account and return a session cookie

        :param request: Registration details (required)
        :type request: RestRegisterRequest
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

        _param = self._auth_register_post_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "RestAuthResponse",
            "400": "RestSwaggerError",
            "409": "RestSwaggerError",
            "500": "RestSwaggerError",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _auth_register_post_serialize(
        self,
        request: RestRegisterRequest,
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
        if "Auth" not in _header_params:
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
            resource_path="/auth/register",
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
