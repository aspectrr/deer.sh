"""
Fluid

API for managing sandboxes, organizations, billing, and hosts

Installation:
    pip install fluid

Quick Start:
    >>> from fluid import Configuration, ServiceA, ServiceB
    >>> config = Configuration(api_key="your-key")
    >>> service_a = ServiceA(config)
    >>> service_a.users.list()
"""

__version__ = "0.1.0"

# Import all API classes
from fluid.api.auth_api import AuthApi
from fluid.api.billing_api import BillingApi
from fluid.api.health_api import HealthApi
from fluid.api.host_tokens_api import HostTokensApi
from fluid.api.hosts_api import HostsApi
from fluid.api.members_api import MembersApi
from fluid.api.organizations_api import OrganizationsApi
from fluid.api.sandboxes_api import SandboxesApi
from fluid.api.source_vms_api import SourceVMsApi
from fluid.api_client import ApiClient
from fluid.configuration import Configuration
from fluid.exceptions import ApiException
# Import all models
from fluid.models.orchestrator_create_sandbox_request import \
    OrchestratorCreateSandboxRequest
from fluid.models.orchestrator_host_info import OrchestratorHostInfo
from fluid.models.orchestrator_prepare_request import \
    OrchestratorPrepareRequest
from fluid.models.orchestrator_read_source_request import \
    OrchestratorReadSourceRequest
from fluid.models.orchestrator_run_command_request import \
    OrchestratorRunCommandRequest
from fluid.models.orchestrator_run_source_request import \
    OrchestratorRunSourceRequest
from fluid.models.orchestrator_snapshot_request import \
    OrchestratorSnapshotRequest
from fluid.models.orchestrator_snapshot_response import \
    OrchestratorSnapshotResponse
from fluid.models.orchestrator_source_command_result import \
    OrchestratorSourceCommandResult
from fluid.models.orchestrator_source_file_result import \
    OrchestratorSourceFileResult
from fluid.models.rest_add_member_request import RestAddMemberRequest
from fluid.models.rest_auth_response import RestAuthResponse
from fluid.models.rest_billing_response import RestBillingResponse
from fluid.models.rest_calculator_request import RestCalculatorRequest
from fluid.models.rest_calculator_response import RestCalculatorResponse
from fluid.models.rest_create_host_token_request import \
    RestCreateHostTokenRequest
from fluid.models.rest_create_org_request import RestCreateOrgRequest
from fluid.models.rest_free_tier_info import RestFreeTierInfo
from fluid.models.rest_host_token_response import RestHostTokenResponse
from fluid.models.rest_login_request import RestLoginRequest
from fluid.models.rest_member_response import RestMemberResponse
from fluid.models.rest_org_response import RestOrgResponse
from fluid.models.rest_register_request import RestRegisterRequest
from fluid.models.rest_swagger_error import RestSwaggerError
from fluid.models.rest_update_org_request import RestUpdateOrgRequest
from fluid.models.rest_usage_summary import RestUsageSummary
from fluid.models.rest_user_response import RestUserResponse
from fluid.models.store_command import StoreCommand
from fluid.models.store_sandbox import StoreSandbox
from fluid.models.store_sandbox_state import StoreSandboxState

__all__ = [
    "Configuration",
    "ApiClient",
    "ApiException",
    "AuthApi",
    "BillingApi",
    "HealthApi",
    "HostTokensApi",
    "HostsApi",
    "MembersApi",
    "OrganizationsApi",
    "SandboxesApi",
    "SourceVMsApi",
]
