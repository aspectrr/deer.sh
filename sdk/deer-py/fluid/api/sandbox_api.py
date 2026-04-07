# coding: utf-8

"""
    fluid-remote API
    API for managing AI Agent VM sandboxes using libvirt
"""

from typing import Any, Dict, List, Optional, Tuple, Union

from pydantic import Field, StrictBool, StrictInt, StrictStr
from typing_extensions import Annotated

from fluid.api_client import ApiClient, RequestSerialized
from fluid.api_response import ApiResponse
from fluid.exceptions import ApiException
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_destroy_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_diff_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_diff_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_discover_ip_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_get_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_inject_ssh_key_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_sandbox_commands_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_sandboxes_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_publish_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_snapshot_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_snapshot_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_start_sandbox_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_start_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse


class SandboxApi:
    """SandboxApi service"""

    def __init__(self, api_client: Optional[ApiClient] = None) -> None:
        if api_client is None:
            api_client = ApiClient.get_default()
        self.api_client = api_client

    def create_sandbox(
        self,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse:
        """Create a new sandbox

        Creates a new virtual machine sandbox by cloning from an existing VM. When multi-host is configured, automatically routes to the host containing the source VM.

        :param request: Sandbox creation parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest
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

        _param = self._create_sandbox_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def create_sandbox_with_http_info(
        self,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[
        GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse
    ]:
        """Create a new sandbox

        Creates a new virtual machine sandbox by cloning from an existing VM. When multi-host is configured, automatically routes to the host containing the source VM.

        :param request: Sandbox creation parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest
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

        _param = self._create_sandbox_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def create_sandbox_without_preload_content(
        self,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Create a new sandbox

        Creates a new virtual machine sandbox by cloning from an existing VM. When multi-host is configured, automatically routes to the host containing the source VM.

        :param request: Sandbox creation parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest
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

        _param = self._create_sandbox_serialize(
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _create_sandbox_serialize(
        self,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest,
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
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes",
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

    def create_snapshot(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse:
        """Create snapshot

        Creates a snapshot of the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Snapshot parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest
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

        _param = self._create_snapshot_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def create_snapshot_with_http_info(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse]:
        """Create snapshot

        Creates a snapshot of the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Snapshot parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest
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

        _param = self._create_snapshot_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def create_snapshot_without_preload_content(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Create snapshot

        Creates a snapshot of the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Snapshot parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest
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

        _param = self._create_snapshot_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "201": "GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _create_snapshot_serialize(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes/{id}/snapshot",
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

    def destroy_sandbox(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse:
        """Destroy sandbox

        Destroys the sandbox and cleans up resources

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._destroy_sandbox_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def destroy_sandbox_with_http_info(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[
        GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse
    ]:
        """Destroy sandbox

        Destroys the sandbox and cleans up resources

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._destroy_sandbox_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def destroy_sandbox_without_preload_content(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Destroy sandbox

        Destroys the sandbox and cleans up resources

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._destroy_sandbox_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _destroy_sandbox_serialize(
        self,
        id: str,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="DELETE",
            resource_path="/v1/sandboxes/{id}",
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

    def diff_snapshots(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse:
        """Diff snapshots

        Computes differences between two snapshots

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Diff parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest
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

        _param = self._diff_snapshots_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def diff_snapshots_with_http_info(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse]:
        """Diff snapshots

        Computes differences between two snapshots

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Diff parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest
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

        _param = self._diff_snapshots_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def diff_snapshots_without_preload_content(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Diff snapshots

        Computes differences between two snapshots

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Diff parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest
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

        _param = self._diff_snapshots_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _diff_snapshots_serialize(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes/{id}/diff",
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

    def discover_sandbox_ip(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse:
        """Discover sandbox IP

        Discovers and returns the IP address for a running sandbox. Use this for async workflows where wait_for_ip was false during start.

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._discover_sandbox_ip_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def discover_sandbox_ip_with_http_info(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse]:
        """Discover sandbox IP

        Discovers and returns the IP address for a running sandbox. Use this for async workflows where wait_for_ip was false during start.

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._discover_sandbox_ip_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def discover_sandbox_ip_without_preload_content(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Discover sandbox IP

        Discovers and returns the IP address for a running sandbox. Use this for async workflows where wait_for_ip was false during start.

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._discover_sandbox_ip_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _discover_sandbox_ip_serialize(
        self,
        id: str,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/v1/sandboxes/{id}/ip",
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

    def generate_configuration(
        self,
        id: str,
        tool: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """Generate configuration

        Generates Ansible or Puppet configuration from sandbox changes

        :param id: Sandbox ID (required)
        :type id: str
        :param tool: Tool type (ansible or puppet) (required)
        :type tool: str
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

        _param = self._generate_configuration_serialize(
            id=id,
            tool=tool,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "501": "GithubComAspectrrFluidShFluidRemoteInternalRestGenerateResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def generate_configuration_with_http_info(
        self,
        id: str,
        tool: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """Generate configuration

        Generates Ansible or Puppet configuration from sandbox changes

        :param id: Sandbox ID (required)
        :type id: str
        :param tool: Tool type (ansible or puppet) (required)
        :type tool: str
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

        _param = self._generate_configuration_serialize(
            id=id,
            tool=tool,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "501": "GithubComAspectrrFluidShFluidRemoteInternalRestGenerateResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def generate_configuration_without_preload_content(
        self,
        id: str,
        tool: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Generate configuration

        Generates Ansible or Puppet configuration from sandbox changes

        :param id: Sandbox ID (required)
        :type id: str
        :param tool: Tool type (ansible or puppet) (required)
        :type tool: str
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

        _param = self._generate_configuration_serialize(
            id=id,
            tool=tool,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "501": "GithubComAspectrrFluidShFluidRemoteInternalRestGenerateResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _generate_configuration_serialize(
        self,
        id: str,
        tool: str,
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
        if id is not None:
            _path_params["id"] = id
        if tool is not None:
            _path_params["tool"] = tool
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="POST",
            resource_path="/v1/sandboxes/{id}/generate/{tool}",
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

    def get_sandbox(
        self,
        id: str,
        include_commands: Optional[bool] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse:
        """Get sandbox details

        Returns detailed information about a specific sandbox including recent commands

        :param id: Sandbox ID (required)
        :type id: str
        :param include_commands: Include command history (optional)
        :type include_commands: bool, optional
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

        _param = self._get_sandbox_serialize(
            id=id,
            include_commands=include_commands,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def get_sandbox_with_http_info(
        self,
        id: str,
        include_commands: Optional[bool] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse]:
        """Get sandbox details

        Returns detailed information about a specific sandbox including recent commands

        :param id: Sandbox ID (required)
        :type id: str
        :param include_commands: Include command history (optional)
        :type include_commands: bool, optional
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

        _param = self._get_sandbox_serialize(
            id=id,
            include_commands=include_commands,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def get_sandbox_without_preload_content(
        self,
        id: str,
        include_commands: Optional[bool] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Get sandbox details

        Returns detailed information about a specific sandbox including recent commands

        :param id: Sandbox ID (required)
        :type id: str
        :param include_commands: Include command history (optional)
        :type include_commands: bool, optional
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

        _param = self._get_sandbox_serialize(
            id=id,
            include_commands=include_commands,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _get_sandbox_serialize(
        self,
        id: str,
        include_commands: Optional[bool],
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        if include_commands is not None:
            _query_params.append(("include_commands", include_commands))

        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/v1/sandboxes/{id}",
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

    def inject_ssh_key(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """Inject SSH key into sandbox

        Injects a public SSH key for a user in the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: SSH key injection parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest
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

        _param = self._inject_ssh_key_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "204": None,
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def inject_ssh_key_with_http_info(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """Inject SSH key into sandbox

        Injects a public SSH key for a user in the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: SSH key injection parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest
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

        _param = self._inject_ssh_key_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "204": None,
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def inject_ssh_key_without_preload_content(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Inject SSH key into sandbox

        Injects a public SSH key for a user in the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: SSH key injection parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest
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

        _param = self._inject_ssh_key_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "204": None,
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _inject_ssh_key_serialize(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes/{id}/sshkey",
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

    def list_sandbox_commands(
        self,
        id: str,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse:
        """List sandbox commands

        Returns all commands executed in the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param limit: Max results to return (optional)
        :type limit: int, optional
        :param offset: Number of results to skip (optional)
        :type offset: int, optional
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

        _param = self._list_sandbox_commands_serialize(
            id=id,
            limit=limit,
            offset=offset,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def list_sandbox_commands_with_http_info(
        self,
        id: str,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[
        GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse
    ]:
        """List sandbox commands

        Returns all commands executed in the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param limit: Max results to return (optional)
        :type limit: int, optional
        :param offset: Number of results to skip (optional)
        :type offset: int, optional
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

        _param = self._list_sandbox_commands_serialize(
            id=id,
            limit=limit,
            offset=offset,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def list_sandbox_commands_without_preload_content(
        self,
        id: str,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """List sandbox commands

        Returns all commands executed in the sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param limit: Max results to return (optional)
        :type limit: int, optional
        :param offset: Number of results to skip (optional)
        :type offset: int, optional
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

        _param = self._list_sandbox_commands_serialize(
            id=id,
            limit=limit,
            offset=offset,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "404": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _list_sandbox_commands_serialize(
        self,
        id: str,
        limit: Optional[int],
        offset: Optional[int],
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        if limit is not None:
            _query_params.append(("limit", limit))

        if offset is not None:
            _query_params.append(("offset", offset))

        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/v1/sandboxes/{id}/commands",
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

    def list_sandboxes(
        self,
        agent_id: Optional[str] = None,
        job_id: Optional[str] = None,
        base_image: Optional[str] = None,
        state: Optional[str] = None,
        vm_name: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse:
        """List sandboxes

        Lists all sandboxes with optional filtering by agent_id, job_id, base_image, state, or vm_name

        :param agent_id: Filter by agent ID (optional)
        :type agent_id: str, optional
        :param job_id: Filter by job ID (optional)
        :type job_id: str, optional
        :param base_image: Filter by base image (optional)
        :type base_image: str, optional
        :param state: Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) (optional)
        :type state: str, optional
        :param vm_name: Filter by VM name (optional)
        :type vm_name: str, optional
        :param limit: Max results to return (optional)
        :type limit: int, optional
        :param offset: Number of results to skip (optional)
        :type offset: int, optional
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

        _param = self._list_sandboxes_serialize(
            agent_id=agent_id,
            job_id=job_id,
            base_image=base_image,
            state=state,
            vm_name=vm_name,
            limit=limit,
            offset=offset,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def list_sandboxes_with_http_info(
        self,
        agent_id: Optional[str] = None,
        job_id: Optional[str] = None,
        base_image: Optional[str] = None,
        state: Optional[str] = None,
        vm_name: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[
        GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse
    ]:
        """List sandboxes

        Lists all sandboxes with optional filtering by agent_id, job_id, base_image, state, or vm_name

        :param agent_id: Filter by agent ID (optional)
        :type agent_id: str, optional
        :param job_id: Filter by job ID (optional)
        :type job_id: str, optional
        :param base_image: Filter by base image (optional)
        :type base_image: str, optional
        :param state: Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) (optional)
        :type state: str, optional
        :param vm_name: Filter by VM name (optional)
        :type vm_name: str, optional
        :param limit: Max results to return (optional)
        :type limit: int, optional
        :param offset: Number of results to skip (optional)
        :type offset: int, optional
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

        _param = self._list_sandboxes_serialize(
            agent_id=agent_id,
            job_id=job_id,
            base_image=base_image,
            state=state,
            vm_name=vm_name,
            limit=limit,
            offset=offset,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def list_sandboxes_without_preload_content(
        self,
        agent_id: Optional[str] = None,
        job_id: Optional[str] = None,
        base_image: Optional[str] = None,
        state: Optional[str] = None,
        vm_name: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """List sandboxes

        Lists all sandboxes with optional filtering by agent_id, job_id, base_image, state, or vm_name

        :param agent_id: Filter by agent ID (optional)
        :type agent_id: str, optional
        :param job_id: Filter by job ID (optional)
        :type job_id: str, optional
        :param base_image: Filter by base image (optional)
        :type base_image: str, optional
        :param state: Filter by state (CREATED, STARTING, RUNNING, STOPPED, DESTROYED, ERROR) (optional)
        :type state: str, optional
        :param vm_name: Filter by VM name (optional)
        :type vm_name: str, optional
        :param limit: Max results to return (optional)
        :type limit: int, optional
        :param offset: Number of results to skip (optional)
        :type offset: int, optional
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

        _param = self._list_sandboxes_serialize(
            agent_id=agent_id,
            job_id=job_id,
            base_image=base_image,
            state=state,
            vm_name=vm_name,
            limit=limit,
            offset=offset,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _list_sandboxes_serialize(
        self,
        agent_id: Optional[str],
        job_id: Optional[str],
        base_image: Optional[str],
        state: Optional[str],
        vm_name: Optional[str],
        limit: Optional[int],
        offset: Optional[int],
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
        if agent_id is not None:
            _query_params.append(("agent_id", agent_id))

        if job_id is not None:
            _query_params.append(("job_id", job_id))

        if base_image is not None:
            _query_params.append(("base_image", base_image))

        if state is not None:
            _query_params.append(("state", state))

        if vm_name is not None:
            _query_params.append(("vm_name", vm_name))

        if limit is not None:
            _query_params.append(("limit", limit))

        if offset is not None:
            _query_params.append(("offset", offset))

        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(
                ["application/json"]
            )

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/v1/sandboxes",
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

    def publish_changes(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """Publish changes

        Publishes sandbox changes to GitOps repository

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Publish parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest
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

        _param = self._publish_changes_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "501": "GithubComAspectrrFluidShFluidRemoteInternalRestPublishResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def publish_changes_with_http_info(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """Publish changes

        Publishes sandbox changes to GitOps repository

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Publish parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest
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

        _param = self._publish_changes_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "501": "GithubComAspectrrFluidShFluidRemoteInternalRestPublishResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def publish_changes_without_preload_content(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Publish changes

        Publishes sandbox changes to GitOps repository

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Publish parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest
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

        _param = self._publish_changes_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "501": "GithubComAspectrrFluidShFluidRemoteInternalRestPublishResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _publish_changes_serialize(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes/{id}/publish",
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

    def run_sandbox_command(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse:
        """Run command in sandbox

        Executes a command inside the sandbox via SSH. If private_key_path is omitted and SSH CA is configured, managed credentials will be used automatically.

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Command execution parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest
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

        _param = self._run_sandbox_command_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def run_sandbox_command_with_http_info(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse]:
        """Run command in sandbox

        Executes a command inside the sandbox via SSH. If private_key_path is omitted and SSH CA is configured, managed credentials will be used automatically.

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Command execution parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest
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

        _param = self._run_sandbox_command_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def run_sandbox_command_without_preload_content(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Run command in sandbox

        Executes a command inside the sandbox via SSH. If private_key_path is omitted and SSH CA is configured, managed credentials will be used automatically.

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Command execution parameters (required)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest
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

        _param = self._run_sandbox_command_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _run_sandbox_command_serialize(
        self,
        id: str,
        request: GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes/{id}/run",
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

    def start_sandbox(
        self,
        id: str,
        request: Optional[
            GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest
        ] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse:
        """Start sandbox

        Starts the virtual machine sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Start parameters (optional)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest, optional
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

        _param = self._start_sandbox_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def start_sandbox_with_http_info(
        self,
        id: str,
        request: Optional[
            GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest
        ] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[
        GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse
    ]:
        """Start sandbox

        Starts the virtual machine sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Start parameters (optional)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest, optional
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

        _param = self._start_sandbox_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def start_sandbox_without_preload_content(
        self,
        id: str,
        request: Optional[
            GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest
        ] = None,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Start sandbox

        Starts the virtual machine sandbox

        :param id: Sandbox ID (required)
        :type id: str
        :param request: Start parameters (optional)
        :type request: GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest, optional
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

        _param = self._start_sandbox_serialize(
            id=id,
            request=request,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "200": "GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse",
            "400": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
            "500": "GithubComAspectrrFluidShFluidRemoteInternalRestErrorResponse",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _start_sandbox_serialize(
        self,
        id: str,
        request: Optional[
            GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxRequest
        ],
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter
        if request is not None:
            _body_params = request

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
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
            resource_path="/v1/sandboxes/{id}/start",
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

    def stream_sandbox_activity(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> None:
        """Stream sandbox activity

        Connects via WebSocket to stream realtime sandbox activity (commands, file changes)

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._stream_sandbox_activity_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "101": "str",
            "400": "str",
            "404": "str",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        ).data

    def stream_sandbox_activity_with_http_info(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> ApiResponse[None]:
        """Stream sandbox activity

        Connects via WebSocket to stream realtime sandbox activity (commands, file changes)

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._stream_sandbox_activity_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "101": "str",
            "400": "str",
            "404": "str",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        response_data.read()
        return self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map=_response_types_map,
        )

    def stream_sandbox_activity_without_preload_content(
        self,
        id: str,
        _request_timeout: Union[None, float, Tuple[float, float]] = None,
        _request_auth: Optional[Dict[str, Any]] = None,
        _content_type: Optional[str] = None,
        _headers: Optional[Dict[str, Any]] = None,
        _host_index: int = 0,
    ) -> Any:
        """Stream sandbox activity

        Connects via WebSocket to stream realtime sandbox activity (commands, file changes)

        :param id: Sandbox ID (required)
        :type id: str
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

        _param = self._stream_sandbox_activity_serialize(
            id=id,
            _request_auth=_request_auth,
            _content_type=_content_type,
            _headers=_headers,
            _host_index=_host_index,
        )

        _response_types_map: Dict[str, Optional[str]] = {
            "101": "str",
            "400": "str",
            "404": "str",
        }
        response_data = self.api_client.call_api(
            *_param, _request_timeout=_request_timeout
        )
        return response_data.response

    def _stream_sandbox_activity_serialize(
        self,
        id: str,
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
        if id is not None:
            _path_params["id"] = id
        # process the query parameters
        # process the header parameters
        # process the form parameters
        # process the body parameter

        # set the HTTP header `Accept`
        if "Sandbox" not in _header_params:
            _header_params["Accept"] = self.api_client.select_header_accept(["*/*"])

        # set the HTTP header `Content-Type`

        # authentication setting
        _auth_settings: List[str] = []

        return self.api_client.param_serialize(
            method="GET",
            resource_path="/v1/sandboxes/{id}/stream",
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
