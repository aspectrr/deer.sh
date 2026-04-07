# coding: utf-8

"""Configuration for Fluid"""

import copy
import http.client as httplib
import logging
import sys
from typing import Any, Dict, List, Optional, TypeVar, Union

T = TypeVar("T")


class Configuration:
    """API client configuration.

    This class contains the configuration for the API client.

    :param host: Base URL for the API.
    :param api_key: Dict to store API keys.
    :param api_key_prefix: Dict to store API key prefixes.
    :param username: Username for HTTP basic authentication.
    :param password: Password for HTTP basic authentication.
    :param access_token: Access token for OAuth/Bearer authentication.
    :param server_index: Index to servers configuration.
    :param server_variables: Mapping with string values to replace variables in
        templated server configuration.
    :param server_operation_index: Mapping from operation ID to index to server
        configuration.
    :param server_operation_variables: Mapping from operation ID to mapping with
        string values to replace variables in templated server configuration.
    :param ssl_ca_cert: Path to a file of concatenated CA certificates in PEM
        format.
    :param retries: Number of retries for API requests.

    Example:
        >>> config = Configuration(
        ...     host="https://api.example.com",
        ...     api_key={"Authorization": "your-api-key"},
        ...     api_key_prefix={"Authorization": "Bearer"}
        ... )
    """

    _default: Optional["Configuration"] = None

    def __init__(
        self,
        host: Optional[str] = None,
        api_key: Optional[Dict[str, str]] = None,
        api_key_prefix: Optional[Dict[str, str]] = None,
        username: Optional[str] = None,
        password: Optional[str] = None,
        access_token: Optional[str] = None,
        server_index: Optional[int] = None,
        server_variables: Optional[Dict[str, str]] = None,
        server_operation_index: Optional[Dict[str, int]] = None,
        server_operation_variables: Optional[Dict[str, Dict[str, str]]] = None,
        ssl_ca_cert: Optional[str] = None,
        retries: Optional[int] = None,
        ca_cert_data: Optional[Union[str, bytes]] = None,
    ) -> None:
        """Initialize configuration."""
        self._base_path = "http://localhost:8080/v1" if host is None else host
        """Default Base url.
        """

        self.server_index = 0 if server_index is None else server_index
        self.server_operation_index = server_operation_index or {}
        """Default server index.
        """

        self.server_variables = server_variables or {}
        self.server_operation_variables = server_operation_variables or {}
        """Default server variables.
        """

        self.temp_folder_path: Optional[str] = None
        """Temp file folder for downloading files.
        """

        # Authentication Settings
        self.api_key = api_key if api_key else {}
        """Dict to store API key(s).
        """

        self.api_key_prefix = api_key_prefix if api_key_prefix else {}
        """Dict to store API prefix (e.g. Bearer).
        """

        self.refresh_api_key_hook = None
        """Function hook to refresh API key if expired.
        """

        self.username = username
        """Username for HTTP basic authentication.
        """

        self.password = password
        """Password for HTTP basic authentication.
        """

        self.access_token = access_token
        """Access token for OAuth/Bearer.
        """

        self.logger: Dict[str, logging.Logger] = {}
        """Logging settings.
        """

        self.logger_format = "%(asctime)s %(levelname)s %(message)s"
        """Log format.
        """

        self.logger_stream_handler: Optional[logging.StreamHandler] = None  # type: ignore[type-arg]
        """Log stream handler.
        """

        self.logger_file_handler: Optional[logging.FileHandler] = None
        """Log file handler.
        """

        self.logger_file: Optional[str] = None
        """Debug file location.
        """

        self.debug = False
        """Debug status.
        """

        self.verify_ssl = True
        """SSL/TLS verification.
        """

        self.ssl_ca_cert = ssl_ca_cert
        """SSL Certificate Authority cert file.
        """

        self.ca_cert_data = ca_cert_data
        """SSL Certificate Authority cert data.
        """

        self.cert_file: Optional[str] = None
        """Client certificate file.
        """

        self.key_file: Optional[str] = None
        """Client key file.
        """

        self.assert_hostname: Optional[bool] = None
        """Verify hostname.
        """

        self.tls_server_name: Optional[str] = None
        """TLS/SSL Server Name Indication (SNI).
        """

        self.connection_pool_maxsize = 100
        """urllib3 connection pool maxsize.
        """

        self.proxy: Optional[str] = None
        """Proxy URL.
        """

        self.proxy_headers: Optional[Dict[str, str]] = None
        """Proxy headers.
        """

        self.retries = retries
        """Retry configuration.
        """

        self.socket_options: Optional[List[tuple]] = None  # type: ignore[type-arg]
        """Socket options.
        """

        self.datetime_format = "%Y-%m-%dT%H:%M:%S.%f%z"
        """Datetime format.
        """

        self.date_format = "%Y-%m-%d"
        """Date format.
        """

        self.client_side_validation = True
        """Client side validation.
        """

    def __deepcopy__(self, memo: Dict[int, Any]) -> "Configuration":
        """Deep copy configuration."""
        cls = self.__class__
        result = cls.__new__(cls)
        memo[id(self)] = result
        for k, v in self.__dict__.items():
            if k not in ("logger", "logger_file_handler"):
                setattr(result, k, copy.deepcopy(v, memo))
        # Shallow copy for logger
        result.logger = self.logger
        result.logger_file_handler = self.logger_file_handler
        return result

    @classmethod
    def set_default(cls, default: Optional["Configuration"]) -> None:
        """Set default configuration.

        :param default: Configuration object to use as default.
        """
        cls._default = default

    @classmethod
    def get_default(cls) -> "Configuration":
        """Get default configuration.

        If no default has been set, a new Configuration instance is created
        and returned.

        :return: Configuration object.
        """
        if cls._default is None:
            cls._default = Configuration()
        return cls._default

    @property
    def host(self) -> str:
        """Return generated host."""
        return self._base_path

    @host.setter
    def host(self, value: str) -> None:
        """Set host.

        :param value: New host value.
        """
        self._base_path = value

    @property
    def logger_file(self) -> Optional[str]:
        """Logger file.

        If set, logging output will be written to the specified file.

        :return: Path to logger file or None.
        """
        return self._logger_file

    @logger_file.setter
    def logger_file(self, value: Optional[str]) -> None:
        """Logger file setter.

        :param value: Path to logger file.
        """
        self._logger_file = value
        if value is not None:
            self.logger_file_handler = logging.FileHandler(value)
            self.logger_file_handler.setFormatter(logging.Formatter(self.logger_format))

    @property
    def debug(self) -> bool:
        """Debug status.

        :return: True if debug is enabled, False otherwise.
        """
        return self._debug

    @debug.setter
    def debug(self, value: bool) -> None:
        """Debug status setter.

        :param value: True to enable debug, False to disable.
        """
        self._debug = value

        if value:
            httplib.HTTPConnection.debuglevel = 1
        else:
            httplib.HTTPConnection.debuglevel = 0

        self._configure_logger()

    def _configure_logger(self) -> None:
        """Configure logger."""
        self.logger["urllib3"] = logging.getLogger("urllib3")

        for key in self.logger:
            if self._debug:
                self.logger[key].setLevel(logging.DEBUG)
                if self.logger_stream_handler is None:
                    self.logger_stream_handler = logging.StreamHandler()
                    self.logger_stream_handler.setFormatter(
                        logging.Formatter(self.logger_format)
                    )
                self.logger[key].addHandler(self.logger_stream_handler)
                if self.logger_file_handler is not None:
                    self.logger[key].addHandler(self.logger_file_handler)
            else:
                self.logger[key].setLevel(logging.WARNING)
                if self.logger_stream_handler is not None:
                    self.logger[key].removeHandler(self.logger_stream_handler)
                if self.logger_file_handler is not None:
                    self.logger[key].removeHandler(self.logger_file_handler)

    def get_api_key_with_prefix(
        self, identifier: str, alias: Optional[str] = None
    ) -> Optional[str]:
        """Get API key with prefix.

        :param identifier: API key identifier.
        :param alias: API key alias.
        :return: API key with prefix if available, None otherwise.
        """
        if self.refresh_api_key_hook is not None:
            self.refresh_api_key_hook(self)

        key = self.api_key.get(identifier, self.api_key.get(alias) if alias else None)
        if key is not None:
            prefix = self.api_key_prefix.get(identifier)
            if prefix is not None:
                return f"{prefix} {key}"
            return key
        return None

    def get_basic_auth_token(self) -> Optional[str]:
        """Get HTTP basic auth header.

        :return: Basic auth token if credentials are set, None otherwise.
        """
        if self.username is None or self.password is None:
            return None
        import base64

        credentials = f"{self.username}:{self.password}"
        encoded = base64.b64encode(credentials.encode()).decode()
        return f"Basic {encoded}"

    def auth_settings(self) -> Dict[str, Dict[str, Any]]:
        """Get auth settings dict for API client.

        :return: Dict with auth settings.
        """
        auth: Dict[str, Dict[str, Any]] = {}
        return auth

    def to_debug_report(self) -> str:
        """Get debug report.

        :return: String with debug information.
        """
        return (
            f"Python SDK Debug Report:\n"
            f"OS: {sys.platform}\n"
            f"Python Version: {sys.version}\n"
            f"Version of the API: 1.0\n"
            f"SDK Package Version: 0.1.0"
        )

    def get_host_settings(self) -> List[Dict[str, Any]]:
        """Get host settings.

        :return: List of server configurations.
        """
        return [
            {
                "url": "//localhost:8080/v1",
                "description": "No description provided",
            }
        ]

    def get_host_from_settings(
        self,
        index: Optional[int] = None,
        variables: Optional[Dict[str, str]] = None,
        servers: Optional[List[Dict[str, Any]]] = None,
    ) -> str:
        """Get host from settings.

        :param index: Index of the server configuration.
        :param variables: Variables to replace in server URL template.
        :param servers: Server configurations list.
        :return: Host URL.
        """
        if index is None:
            index = self.server_index

        if servers is None:
            servers = self.get_host_settings()

        if index < 0 or index >= len(servers):
            raise ValueError(
                f"Invalid server index {index}. Must be between 0 and {len(servers) - 1}."
            )

        server = servers[index]
        url: str = str(server["url"])

        # Use server variables or provided variables
        if variables is None:
            variables = self.server_variables

        # Replace variables in URL
        if "variables" in server:
            for var_name, var_config in server["variables"].items():
                var_value = variables.get(var_name, var_config.get("default_value", ""))

                # Validate enum values
                if (
                    "enum_values" in var_config
                    and var_value not in var_config["enum_values"]
                ):
                    raise ValueError(
                        f"Variable '{var_name}' has invalid value '{var_value}'. "
                        f"Must be one of {var_config['enum_values']}."
                    )

                url = url.replace(f"", var_value)

        return url
